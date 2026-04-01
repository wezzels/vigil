// dis-hla-gateway — VIMI DIS↔HLA Bridge
// Phase 3: Advanced Integration
// Translates between DIS PDUs (IEEE 1278.1) and HLA objects/interactions
// Supports DIS 7 and HLA 1516-2010 (RTI-NG)
package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	//"strconv"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
)

/*
HLA Object and Interaction class IDs (OMT format)
These would normally come from the FOM — here we define a minimal subset
for DIS↔HLA translation
*/
const (
	// Object classes
	HLAObjectRoot          = 0x00000001
	HLAObjectPlatform      = 0x00000010
	HLAObjectMissile       = 0x00000020
	HLAObjectSensor        = 0x00000030
	HLAObjectTrack         = 0x00000040

	// Interaction classes
	HLAInteractionRoot     = 0x00000100
	HLAInteractionFire     = 0x00000110
	HLAInteractionDetonate  = 0x00000120
	HLAInteractionAlert    = 0x00000130
)

// ForceID matches DIS enumeration
type ForceID uint8

const (
	ForceOther    ForceID = 0
	ForceFriendly ForceID = 1
	ForceOpposing ForceID = 2
	ForceNeutral  ForceID = 3
)

// DIS PDU types
type PDUType uint8

const (
	PDUEntityState    PDUType = 1
	PDUFire           PDUType = 2
	PDUDetonation    PDUType = 3
	PDUCollision      PDUType = 4
	PDUServiceRequest PDUType = 5
	PDUStartResume    PDUType = 6
	PDUAcknowledge    PDUType = 7
	PDUActionRequest  PDUType = 8
	PDUData           PDUType = 9
)

// EntityStatePDU — IEEE 1278.1-2012 Entity State PDU (simplified)
type EntityStatePDU struct {
	ProtocolVersion uint16
	ExerciseID     uint8
	PDUType        PDUType
	Timestamp      uint32
	Length         uint16
	SiteID         uint16
	ApplicationID  uint16
	EntityID       uint32
	ForceID        uint8
	EntityTypeKind uint8
	EntityTypeDomain uint8
	EntityTypeCountry uint16
	EntityTypeCategory uint8
	EntityTypeSubcategory uint8
	EntityTypeSpecific uint8
	OrientationYaw   float32
	OrientationPitch float32
	OrientationRoll  float32
	VelocityX float32
	VelocityY float32
	VelocityZ float32
	LocationX float32
	LocationY float32
	LocationZ float32
}

// Fire PDU —DIS
type FirePDU struct {
	PDUHeader
	ExercID     uint8
	EventID     uint32
	WeaponID    EntityIdentifier
	LocationX   float32
	LocationY   float32
	LocationZ   float32
	VelocityX   float32
	VelocityY   float32
	VelocityZ   float32
	Range       float32
}

// Detonation PDU — DIS
type DetonationPDU struct {
	PDUHeader
	EventID           uint32
	WeaponID          EntityIdentifier
	LocationX         float32
	LocationY         float32
	LocationZ         float32
	VelocityX         float32
	VelocityY         float32
	VelocityZ         float32
	DetonationResult  uint8
}

type PDUHeader struct {
	ProtocolVersion uint16
	PDUType        PDUType
	Timestamp      uint32
	Length         uint16
	SiteID         uint16
	ApplicationID  uint16
}

type EntityIdentifier struct {
	Site uint16; App uint16; Entity uint32
}

// HLA Attribute Update (Object)
type HLAObjectUpdate struct {
	ClassID    uint32       `json:"class_id"`
	InstanceID uint32       `json:"instance_id"`
	Attributes map[string]interface{} `json:"attributes"`
	Timestamp  time.Time   `json:"timestamp"`
}

// HLA Interaction
type HLAInteraction struct {
	ClassID    uint32       `json:"class_id"`
	Parameters map[string]interface{} `json:"parameters"`
	Timestamp  time.Time   `json:"timestamp"`
}

// Bridge state
type bridgeState struct {
	// DIS→HLA translation state
	disEntities map[uint32]*EntityStatePDU

	// HLA→DIS translation state
	hlaObjects map[uint32]*HLAObjectUpdate

	// Sequence counters
	disSeq uint32
	hlaSeq uint32

	// Federation
	federateName string
	federationName string
	rtiConnected bool
}

func newBridge() *bridgeState {
	return &bridgeState{
		disEntities: make(map[uint32]*EntityStatePDU),
		hlaObjects:  make(map[uint32]*HLAObjectUpdate),
	}
}

// ECEF ↔ Geodetic (WGS84)
func ecefToGeodetic(x, y, z float64) (lat, lon, alt float64) {
	a := 6378137.0
	f := 1 / 298.257223563
	e2 := 2*f - f*f

	lon = math.Atan2(y, x)
	p := math.Sqrt(x*x + y*y)
	lat = math.Atan2(z, p*(1-e2))

	for i := 0; i < 5; i++ {
		N := a / math.Sqrt(1-e2*math.Sin(lat)*math.Sin(lat))
		lat = math.Atan2(z+e2*N*math.Sin(lat), p)
	}
	N := a / math.Sqrt(1-e2*math.Sin(lat)*math.Sin(lat))
	alt = p/math.Cos(lat) - N

	return lat * 180 / math.Pi, lon * 180 / math.Pi, alt / 1000
}

func geodeticToECEF(lat, lon, alt float64) (x, y, z float64) {
	a := 6378137.0
	f := 1 / 298.257223563
	e2 := 2*f - f*f

	latRad := lat * math.Pi / 180
	lonRad := lon * math.Pi / 180

	N := a / math.Sqrt(1-e2*math.Sin(latRad)*math.Sin(latRad))
	x = (N + alt*1000) * math.Cos(latRad) * math.Cos(lonRad)
	y = (N + alt*1000) * math.Cos(latRad) * math.Sin(lonRad)
	z = (N*(1-e2) + alt*1000) * math.Sin(latRad)
	return
}

// DIS PDU parsing
func parseDISPDU(data []byte) (pduType PDUType, pdu interface{}) {
	if len(data) < 8 {
		return 0, nil
	}
	pduType = PDUType(data[3])

	switch pduType {
	case PDUEntityState:
		pdu = parseEntityStatePDU(data)
	case PDUFire:
		pdu = parseFirePDU(data)
	case PDUDetonation:
		pdu = parseDetonationPDU(data)
	}
	return
}

func parseEntityStatePDU(data []byte) *EntityStatePDU {
	if len(data) < 68 {
		return nil
	}
	return &EntityStatePDU{
		ProtocolVersion: binary.BigEndian.Uint16(data[0:2]),
		ExerciseID:     data[2],
		PDUType:        PDUType(data[3]),
		Timestamp:      binary.BigEndian.Uint32(data[4:8]),
		Length:         binary.BigEndian.Uint16(data[8:10]),
		SiteID:         binary.BigEndian.Uint16(data[10:12]),
		ApplicationID:  binary.BigEndian.Uint16(data[12:14]),
		EntityID:       binary.BigEndian.Uint32(data[14:18]),
		ForceID:        data[18],
		EntityTypeKind: data[19],
		EntityTypeDomain: data[20],
		EntityTypeCountry: binary.BigEndian.Uint16(data[21:23]),
		EntityTypeCategory: data[23],
		EntityTypeSubcategory: data[24],
		EntityTypeSpecific: data[25],
		OrientationYaw:   math.Float32frombits(binary.BigEndian.Uint32(data[28:32])),
		OrientationPitch: math.Float32frombits(binary.BigEndian.Uint32(data[32:36])),
		OrientationRoll:  math.Float32frombits(binary.BigEndian.Uint32(data[36:40])),
		VelocityX: math.Float32frombits(binary.BigEndian.Uint32(data[40:44])),
		VelocityY: math.Float32frombits(binary.BigEndian.Uint32(data[44:48])),
		VelocityZ: math.Float32frombits(binary.BigEndian.Uint32(data[48:52])),
		LocationX: math.Float32frombits(binary.BigEndian.Uint32(data[52:56])),
		LocationY: math.Float32frombits(binary.BigEndian.Uint32(data[56:60])),
		LocationZ: math.Float32frombits(binary.BigEndian.Uint32(data[60:64])),
	}
}

func parseFirePDU(data []byte) *FirePDU {
	if len(data) < 64 {
		return nil
	}
	return &FirePDU{
		PDUHeader: PDUHeader{
			ProtocolVersion: binary.BigEndian.Uint16(data[0:2]),
			PDUType:        PDUType(data[3]),
			Timestamp:      binary.BigEndian.Uint32(data[4:8]),
			Length:         binary.BigEndian.Uint16(data[8:10]),
			SiteID:         binary.BigEndian.Uint16(data[10:12]),
			ApplicationID:  binary.BigEndian.Uint16(data[12:14]),
		},
		EventID: binary.BigEndian.Uint32(data[20:24]),
		WeaponID: EntityIdentifier{
			Site: binary.BigEndian.Uint16(data[24:26]),
			App:  binary.BigEndian.Uint16(data[26:28]),
			Entity: binary.BigEndian.Uint32(data[28:32]),
		},
		LocationX: math.Float32frombits(binary.BigEndian.Uint32(data[32:36])),
		LocationY: math.Float32frombits(binary.BigEndian.Uint32(data[36:40])),
		LocationZ: math.Float32frombits(binary.BigEndian.Uint32(data[40:44])),
		VelocityX: math.Float32frombits(binary.BigEndian.Uint32(data[44:48])),
		VelocityY: math.Float32frombits(binary.BigEndian.Uint32(data[48:52])),
		VelocityZ: math.Float32frombits(binary.BigEndian.Uint32(data[52:56])),
		Range:     math.Float32frombits(binary.BigEndian.Uint32(data[56:60])),
	}
}

func parseDetonationPDU(data []byte) *DetonationPDU {
	if len(data) < 64 {
		return nil
	}
	return &DetonationPDU{
		PDUHeader: PDUHeader{
			ProtocolVersion: binary.BigEndian.Uint16(data[0:2]),
			PDUType:        PDUType(data[3]),
			Timestamp:      binary.BigEndian.Uint32(data[4:8]),
			Length:         binary.BigEndian.Uint16(data[8:10]),
			SiteID:         binary.BigEndian.Uint16(data[10:12]),
			ApplicationID:  binary.BigEndian.Uint16(data[12:14]),
		},
		EventID: binary.BigEndian.Uint32(data[20:24]),
		WeaponID: EntityIdentifier{
			Site: binary.BigEndian.Uint16(data[24:26]),
			App:  binary.BigEndian.Uint16(data[26:28]),
			Entity: binary.BigEndian.Uint32(data[28:32]),
		},
		LocationX: math.Float32frombits(binary.BigEndian.Uint32(data[32:36])),
		LocationY: math.Float32frombits(binary.BigEndian.Uint32(data[36:40])),
		LocationZ: math.Float32frombits(binary.BigEndian.Uint32(data[40:44])),
		VelocityX: math.Float32frombits(binary.BigEndian.Uint32(data[44:48])),
		VelocityY: math.Float32frombits(binary.BigEndian.Uint32(data[48:52])),
		VelocityZ: math.Float32frombits(binary.BigEndian.Uint32(data[52:56])),
		DetonationResult: data[60],
	}
}

// DIS → HLA translation
func (b *bridgeState) disToHLA(pduType PDUType, pdu interface{}) *HLAInteraction {
	switch pduType {
	case PDUFire:
		if fire, ok := pdu.(*FirePDU); ok {
			return b.translateFireToHLA(fire)
		}
	case PDUDetonation:
		if det, ok := pdu.(*DetonationPDU); ok {
			return b.translateDetonationToHLA(det)
		}
	}
	return nil
}

func (b *bridgeState) translateFireToHLA(fire *FirePDU) *HLAInteraction {
	x := float64(fire.LocationX)
	y := float64(fire.LocationY)
	z := float64(fire.LocationZ)
	lat, lon, alt := ecefToGeodetic(x, y, z)

	return &HLAInteraction{
		ClassID: HLAInteractionFire,
		Parameters: map[string]interface{}{
			"event_id":        fire.EventID,
			"firing_entity":   fire.WeaponID.Entity,
			"firing_location": map[string]float64{"lat": lat, "lon": lon, "alt": alt},
			"velocity":        map[string]float32{"x": fire.VelocityX, "y": fire.VelocityY, "z": fire.VelocityZ},
			"range":           fire.Range,
		},
		Timestamp: time.Now(),
	}
}

func (b *bridgeState) translateDetonationToHLA(det *DetonationPDU) *HLAInteraction {
	x := float64(det.LocationX)
	y := float64(det.LocationY)
	z := float64(det.LocationZ)
	lat, lon, alt := ecefToGeodetic(x, y, z)

	return &HLAInteraction{
		ClassID: HLAInteractionDetonate,
		Parameters: map[string]interface{}{
			"event_id":          det.EventID,
			"weapon_id":         det.WeaponID.Entity,
			"detonation_location": map[string]float64{"lat": lat, "lon": lon, "alt": alt},
			"velocity":          map[string]float32{"x": det.VelocityX, "y": det.VelocityY, "z": det.VelocityZ},
			"detonation_result": det.DetonationResult,
		},
		Timestamp: time.Now(),
	}
}

// DIS EntityState → HLA Object Update
func (b *bridgeState) disESPDUToHLA(esp *EntityStatePDU) *HLAObjectUpdate {
	x := float64(esp.LocationX)
	y := float64(esp.LocationY)
	z := float64(esp.LocationZ)
	lat, lon, alt := ecefToGeodetic(x, y, z)

	// Determine HLA class from entity type
	var classID uint32 = HLAObjectRoot
	switch {
	case esp.EntityTypeCategory >= 1 && esp.EntityTypeCategory <= 9: // Land platform
		classID = HLAObjectPlatform
	case esp.EntityTypeCategory >= 10 && esp.EntityTypeCategory <= 19: // Air platform
		classID = HLAObjectPlatform
	case esp.EntityTypeCategory >= 20 && esp.EntityTypeCategory <= 29: // Missile
		classID = HLAObjectMissile
	case esp.EntityTypeCategory >= 40 && esp.EntityTypeCategory <= 49: // Sensor
		classID = HLAObjectSensor
	default:
		classID = HLAObjectTrack
	}

	forceName := map[uint8]string{0: "Other", 1: "Friendly", 2: "Opposing", 3: "Neutral"}
	domainName := map[uint8]string{0: "Other", 1: "Land", 2: "Air", 3: "Sea", 4: "Space"}

	obj := &HLAObjectUpdate{
		ClassID:    classID,
		InstanceID: esp.EntityID,
		Attributes: map[string]interface{}{
			"entity_id":            esp.EntityID,
			"force_id":             forceName[esp.ForceID],
			"entity_type": map[string]interface{}{
				"kind":       esp.EntityTypeKind,
				"domain":     domainName[esp.EntityTypeDomain],
				"country":    esp.EntityTypeCountry,
				"category":   esp.EntityTypeCategory,
				"subcategory": esp.EntityTypeSubcategory,
				"specific":   esp.EntityTypeSpecific,
			},
			"position": map[string]float64{"lat": lat, "lon": lon, "alt": alt},
			"orientation": map[string]float32{
				"yaw":   esp.OrientationYaw,
				"pitch": esp.OrientationPitch,
				"roll":  esp.OrientationRoll,
			},
			"velocity": map[string]float32{
				"x": esp.VelocityX,
				"y": esp.VelocityY,
				"z": esp.VelocityZ,
			},
			"world_coordinates": map[string]float32{
				"x": esp.LocationX,
				"y": esp.LocationY,
				"z": esp.LocationZ,
			},
		},
		Timestamp: time.Now(),
	}

	b.disEntities[esp.EntityID] = esp
	return obj
}

// HLA → DIS translation
func (b *bridgeState) hlaToDIS(obj *HLAObjectUpdate) *EntityStatePDU {
	lat := 0.0
	lon := 0.0
	alt := 0.0

	if pos, ok := obj.Attributes["position"].(map[string]interface{}); ok {
		if la, ok := pos["lat"].(float64); ok {
			lat = la
		}
		if lo, ok := pos["lon"].(float64); ok {
			lon = lo
		}
		if al, ok := pos["alt"].(float64); ok {
			alt = al
		}
	}

	x, y, z := geodeticToECEF(lat, lon, alt)

	// Map HLA class to DIS entity type
	category := uint8(1)
	if obj.ClassID == HLAObjectMissile {
		category = 24 // Missile
	} else if obj.ClassID == HLAObjectSensor {
		category = 41
	}

	forceID := uint8(1)
	if force, ok := obj.Attributes["force_id"].(string); ok {
		switch force {
		case "Friendly":
			forceID = 1
		case "Opposing":
			forceID = 2
		case "Neutral":
			forceID = 3
		}
	}

	var vx, vy, vz float32 = 0, 0, 0
	if vel, ok := obj.Attributes["velocity"].(map[string]interface{}); ok {
		if x, ok := vel["x"].(float32); ok {
			vx = x
		}
		if y, ok := vel["y"].(float32); ok {
			vy = y
		}
		if z, ok := vel["z"].(float32); ok {
			vz = z
		}
	}

	var yaw, pitch, roll float32 = 0, 0, 0
	if ori, ok := obj.Attributes["orientation"].(map[string]interface{}); ok {
		if y, ok := ori["yaw"].(float32); ok {
			yaw = y
		}
		if p, ok := ori["pitch"].(float32); ok {
			pitch = p
		}
		if r, ok := ori["roll"].(float32); ok {
			roll = r
		}
	}

	return &EntityStatePDU{
		ProtocolVersion: 7,
		ExerciseID:     1,
		PDUType:        PDUEntityState,
		Timestamp:      timestampDIS(time.Now()),
		Length:         144,
		SiteID:         1,
		ApplicationID:  10, // dis-hla-gateway
		EntityID:       obj.InstanceID,
		ForceID:        uint8(forceID),
		EntityTypeKind: 1,
		EntityTypeDomain: 1,
		EntityTypeCountry: 225,
		EntityTypeCategory: category,
		OrientationYaw:   yaw,
		OrientationPitch: pitch,
		OrientationRoll:  roll,
		VelocityX: vx,
		VelocityY: vy,
		VelocityZ: vz,
		LocationX: float32(x),
		LocationY: float32(y),
		LocationZ: float32(z),
	}
}

func timestampDIS(t time.Time) uint32 {
	return uint32(t.UnixMilli() % 3600000 / 10)
}

func espduToBytes(esp *EntityStatePDU) []byte {
	buf := make([]byte, 144)
	binary.BigEndian.PutUint16(buf[0:2], esp.ProtocolVersion)
	buf[2] = esp.ExerciseID
	buf[3] = uint8(esp.PDUType)
	binary.BigEndian.PutUint32(buf[4:8], esp.Timestamp)
	binary.BigEndian.PutUint16(buf[8:10], esp.Length)
	binary.BigEndian.PutUint16(buf[10:12], esp.SiteID)
	binary.BigEndian.PutUint16(buf[12:14], esp.ApplicationID)
	binary.BigEndian.PutUint32(buf[14:18], esp.EntityID)
	buf[18] = esp.ForceID
	buf[19] = esp.EntityTypeKind
	buf[20] = esp.EntityTypeDomain
	binary.BigEndian.PutUint16(buf[21:23], esp.EntityTypeCountry)
	buf[23] = esp.EntityTypeCategory
	buf[24] = esp.EntityTypeSubcategory
	buf[25] = esp.EntityTypeSpecific
	binary.BigEndian.PutUint32(buf[28:32], math.Float32bits(esp.OrientationYaw))
	binary.BigEndian.PutUint32(buf[32:36], math.Float32bits(esp.OrientationPitch))
	binary.BigEndian.PutUint32(buf[36:40], math.Float32bits(esp.OrientationRoll))
	binary.BigEndian.PutUint32(buf[40:44], math.Float32bits(esp.VelocityX))
	binary.BigEndian.PutUint32(buf[44:48], math.Float32bits(esp.VelocityY))
	binary.BigEndian.PutUint32(buf[48:52], math.Float32bits(esp.VelocityZ))
	binary.BigEndian.PutUint32(buf[52:56], math.Float32bits(esp.LocationX))
	binary.BigEndian.PutUint32(buf[56:60], math.Float32bits(esp.LocationY))
	binary.BigEndian.PutUint32(buf[60:64], math.Float32bits(esp.LocationZ))
	return buf
}

var (
	br               *bridgeState
	kafkaDISIn       *kafka.Reader
	kafkaHLAIn       *kafka.Reader
	kafkaDISOut      *kafka.Writer
	kafkaHLAOut      *kafka.Writer
	kafkaBroker      = getEnv("KAFKA_BROKERS", "kafka:9092")
	port             = getEnv("PORT", "8090")
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

const (
	TopicDISIn       = "vimi.dis.entity-state"
	TopicHLAIn       = "vimi.hla.object-update"
	TopicDISOut      = "vimi.dis.entity-state-out"
	TopicHLAOut      = "vimi.hla.interaction"
)

func run(ctx context.Context) {
	// DIS input → translate to HLA
	disReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{kafkaBroker},
		Topic:     TopicDISIn,
		GroupID:   "dis-hla-gateway-dis",
		MinBytes:  10e3,
		MaxBytes:  10e6,
		StartOffset: kafka.LastOffset,
	})
	defer disReader.Close()

	// HLA input → translate to DIS
	hlaReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{kafkaBroker},
		Topic:     TopicHLAIn,
		GroupID:   "dis-hla-gateway-hla",
		MinBytes:  10e3,
		MaxBytes:  10e6,
		StartOffset: kafka.LastOffset,
	})
	defer hlaReader.Close()

	disCh := make(chan []byte, 100)
	hlaCh := make(chan []byte, 100)

	// Read DIS PDUs
	go func() {
		for {
			msg, err := disReader.ReadMessage(ctx)
			if err != nil {
				continue
			}
			select {
			case disCh <- msg.Value:
			default:
			}
		}
	}()

	// Read HLA updates
	go func() {
		for {
			msg, err := hlaReader.ReadMessage(ctx)
			if err != nil {
				continue
			}
			select {
			case hlaCh <- msg.Value:
			default:
			}
		}
	}()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Periodic publish of all tracked DIS entities as HLA objects
			for entityID, esp := range br.disEntities {
				hlaObj := br.disESPDUToHLA(esp)
				hlaData, _ := json.Marshal(hlaObj)
				ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
				kafkaHLAOut.WriteMessages(ctx2, kafka.Message{
					Key:   []byte(fmt.Sprintf("hla-obj-%d", entityID)),
					Value: hlaData,
					Headers: []kafka.Header{
						{Key: "class_id", Value: []byte(fmt.Sprintf("%d", hlaObj.ClassID))},
						{Key: "instance_id", Value: []byte(fmt.Sprintf("%d", entityID))},
					},
				})
				cancel()
			}
		case data := <-disCh:
			pduType, pdu := parseDISPDU(data)
			if pduType == PDUEntityState {
				if esp, ok := pdu.(*EntityStatePDU); ok {
					hlaObj := br.disESPDUToHLA(esp)
					hlaData, _ := json.Marshal(hlaObj)
					ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
					kafkaHLAOut.WriteMessages(ctx2, kafka.Message{
						Key:   []byte(fmt.Sprintf("hla-obj-%d", esp.EntityID)),
						Value: hlaData,
						Headers: []kafka.Header{
							{Key: "class_id", Value: []byte(fmt.Sprintf("%d", hlaObj.ClassID))},
							{Key: "instance_id", Value: []byte(fmt.Sprintf("%d", esp.EntityID))},
						},
					})
					cancel()
				}
			} else {
				// Fire/Detonation → HLA Interaction
				if inter := br.disToHLA(pduType, pdu); inter != nil {
					interData, _ := json.Marshal(inter)
					ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
					kafkaHLAOut.WriteMessages(ctx2, kafka.Message{
						Key:   []byte(fmt.Sprintf("hla-int-%d", br.hlaSeq)),
						Value: interData,
						Headers: []kafka.Header{
							{Key: "class_id", Value: []byte(fmt.Sprintf("%d", inter.ClassID))},
						},
					})
					br.hlaSeq++
					cancel()
				}
			}
		case data := <-hlaCh:
			var obj HLAObjectUpdate
			if json.Unmarshal(data, &obj) == nil {
				esp := br.hlaToDIS(&obj)
				disData := espduToBytes(esp)
				ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
				kafkaDISOut.WriteMessages(ctx2, kafka.Message{
					Key:   []byte(fmt.Sprintf("dis-esp-%d", obj.InstanceID)),
					Value: disData,
					Headers: []kafka.Header{
						{Key: "pdu_type", Value: []byte("espdu")},
						{Key: "force_id", Value: []byte(fmt.Sprintf("%d", esp.ForceID))},
					},
				})
				cancel()
				br.hlaObjects[obj.InstanceID] = &obj
			}
		}
	}
}

type HealthResponse struct {
	Service      string    `json:"service"`
	Version      string    `json:"version"`
	Timestamp    time.Time `json:"timestamp"`
	Status       string    `json:"status"`
	DisEntities  int       `json:"dis_entities"`
	HlaObjects   int       `json:"hla_objects"`
	RtiConnected bool      `json:"rti_connected"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Service:     "dis-hla-gateway",
		Version:     "0.1.0",
		Timestamp:   time.Now().UTC(),
		Status:      "healthy",
		DisEntities: len(br.disEntities),
		HlaObjects:  len(br.hlaObjects),
		RtiConnected: false, // RTI connection not implemented (would use Portico/Mak's RTI)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	br = newBridge()

	kafkaDISOut = &kafka.Writer{
		Addr:     kafka.TCP(kafkaBroker),
		Topic:    TopicDISOut,
		Balancer: &kafka.LeastBytes{},
	}
	kafkaHLAOut = &kafka.Writer{
		Addr:     kafka.TCP(kafkaBroker),
		Topic:    TopicHLAOut,
		Balancer: &kafka.LeastBytes{},
	}
	defer kafkaDISOut.Close()
	defer kafkaHLAOut.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Shutting down...")
		cancel()
	}()

	http.HandleFunc("/health", healthHandler)

	// Manual translation endpoints for testing
	http.HandleFunc("/translate/dis-to-hla", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "POST only", 405)
			return
		}
		data, _ := io.ReadAll(r.Body)
		pduType, pdu := parseDISPDU(data)
		if pduType == PDUEntityState {
			if esp, ok := pdu.(*EntityStatePDU); ok {
				hlaObj := br.disESPDUToHLA(esp)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(hlaObj)
				return
			}
		}
		http.Error(w, "Failed to parse DIS PDU", 400)
	})

	http.HandleFunc("/translate/hla-to-dis", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "POST only", 405)
			return
		}
		data, _ := io.ReadAll(r.Body)
		var obj HLAObjectUpdate
		if json.Unmarshal(data, &obj) == nil {
			esp := br.hlaToDIS(&obj)
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(espduToBytes(esp))
			return
		}
		http.Error(w, "Failed to parse HLA object", 400)
	})

	// Entity state summary
	http.HandleFunc("/entities", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		type entitySummary struct {
			EntityID uint32 `json:"entity_id"`
			SiteID   uint16 `json:"site_id"`
			AppID    uint16 `json:"app_id"`
			ForceID  uint8  `json:"force_id"`
			Lat      float64 `json:"lat"`
			Lon      float64 `json:"lon"`
			Alt      float64 `json:"alt"`
		}
		var list []entitySummary
		for id, esp := range br.disEntities {
			lat, lon, alt := ecefToGeodetic(float64(esp.LocationX), float64(esp.LocationY), float64(esp.LocationZ))
			list = append(list, entitySummary{
				EntityID: id,
				SiteID:   esp.SiteID,
				AppID:    esp.ApplicationID,
				ForceID:  esp.ForceID,
				Lat:      lat,
				Lon:      lon,
				Alt:      alt,
			})
		}
		json.NewEncoder(w).Encode(list)
	})

	log.Printf("dis-hla-gateway starting")
	log.Printf("Kafka broker: %s", kafkaBroker)
	log.Printf("DIS in: %s, DIS out: %s", TopicDISIn, TopicDISOut)
	log.Printf("HLA in: %s, HLA out: %s", TopicHLAIn, TopicHLAOut)
	go run(ctx)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("HTTP server on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
