// lvc-coordinator — VIMI LVC Coordinator Service
// Phase 2: Mission Processing
// Manages DIS entity state, dead reckoning algorithms, and coordinates
// Live/Virtual/Constructive entities in the federation
package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
)

const (
	TopicLE     = "vimi.fusion.tracks"    // Live Entity tracks from sensor fusion
	TopicDISOut = "vimi.dis.entity-state" // DIS Entity State PDUs out
	TopicDISIn  = "vimi.dis.fire"         // DIS Fire/Detonation PDUs in
	TopicLVC    = "vimi.lvc.events"       // LVC coordination events
)

// ForceID from DIS
type ForceID uint8

const (
	ForceOther    ForceID = 0
	ForceFriendly ForceID = 1
	ForceOpposing ForceID = 2
	ForceNeutral  ForceID = 3
)

// Dead Reckoning Algorithm (DRM)
type DRM int

const (
	DRMStatic DRM = iota
	DRMFPM        // Fixed Position
	DRMRPM        // Rotating Position
	DRMRVM        // Rotating Velocity
	DRMFVM        // Fixed Velocity
	DRMVVW        // Velocity Vector W
)

// DRM name
func (d DRM) String() string {
	names := []string{"Static", "DRM_FPM", "DRM_RPM", "DRM_RVM", "DRM_FVM", "DRM_VVW", "DRM_VVW"}
	if d < 0 || int(d) >= len(names) {
		return "Unknown"
	}
	return names[d]
}

// Entity is a managed DIS entity
type Entity struct {
	EntityID     uint32  `json:"entity_id"`
	SiteID       uint16  `json:"site_id"`
	AppID        uint16  `json:"app_id"`
	EntityNumber uint32  `json:"entity_number"`
	ForceID      ForceID `json:"force_id"`
	TypeKind     uint8   `json:"type_kind"`
	TypeDomain   uint8   `json:"type_domain"`
	TypeCountry  uint16  `json:"type_country"`
	TypeCategory uint8   `json:"type_category"`
	TypeSubcat   uint8   `json:"type_subcategory"`
	TypeSpecific uint8   `json:"type_specific"`

	// Position (geodetic)
	Lat, Lon float64 `json:"lat,lon"`
	Alt      float64 `json:"alt"` // km

	// ECEF position
	X, Y, Z float64 `json:"x,y,z"` // meters

	// Orientation (Euler angles, radians)
	Yaw, Pitch, Roll float64 `json:"yaw,pitch,roll"`

	// Velocity (m/s)
	VX, VY, VZ float64 `json:"vx,vy,vz"`

	// Dead reckoning
	DRMAlgorithm     DRM     `json:"drm"`
	DRPX, DRPY, DRPZ float64 `json:"drp_x,drp_y,drp_z"` // DR position
	DRVX, DRVY, DRVZ float64 `json:"drv_x,drv_y,drv_z"` // DR velocity

	// State
	State   int     `json:"state"`   // 0=active, 1=deactivated, 2=destroyed, 3=damage
	Health  float32 `json:"health"`  // 0-1
	Marking string  `json:"marking"` // Unit callsign

	// LVC classification
	LVCType string `json:"lvc_type"` // "Live", "Virtual", "Constructive"

	// Timing
	LastUpdate   time.Time     `json:"last_update"`
	HeartbeatInt time.Duration `json:"heartbeat_interval"`
	IsLocal      bool          `json:"is_local"` // Generated locally vs received from network
}

// EntityStatePDU per IEEE 1278.1-2012
type EntityStatePDU struct {
	ProtocolVersion       uint16
	ExerciseID            uint8
	PDUType               uint8
	Timestamp             uint32
	Length                uint16
	SiteID                uint16
	ApplicationID         uint16
	EntityID              uint32
	ForceID               uint8
	EntityTypeKind        uint8
	EntityTypeDomain      uint8
	EntityTypeCountry     uint16
	EntityTypeCategory    uint8
	EntityTypeSubcategory uint8
	EntityTypeSpecific    uint8
	// 8 bits articulation parameters
	OrientationYaw   float32
	OrientationPitch float32
	OrientationRoll  float32
	VelocityX        float32
	VelocityY        float32
	VelocityZ        float32
	LocationX        float32
	LocationY        float32
	LocationZ        float32
}

func timestampDIS(t time.Time) uint32 {
	ms := uint64(t.UnixMilli() % 3600000)
	return uint32(ms / 10)
}

func ecefFromGeodetic(lat, lon, alt float64) (x, y, z float64) {
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

func (pdu *EntityStatePDU) Encode() []byte {
	buf := make([]byte, 144)
	binary.BigEndian.PutUint16(buf[0:2], pdu.ProtocolVersion)
	buf[2] = pdu.ExerciseID
	buf[3] = pdu.PDUType
	binary.BigEndian.PutUint32(buf[4:8], pdu.Timestamp)
	binary.BigEndian.PutUint16(buf[8:10], pdu.Length)
	binary.BigEndian.PutUint16(buf[10:12], pdu.SiteID)
	binary.BigEndian.PutUint16(buf[12:14], pdu.ApplicationID)
	binary.BigEndian.PutUint32(buf[14:18], pdu.EntityID)
	buf[18] = pdu.ForceID
	buf[19] = pdu.EntityTypeKind
	buf[20] = pdu.EntityTypeDomain
	binary.BigEndian.PutUint16(buf[21:23], pdu.EntityTypeCountry)
	buf[23] = pdu.EntityTypeCategory
	buf[24] = pdu.EntityTypeSubcategory
	buf[25] = pdu.EntityTypeSpecific
	// Orientation (bytes 26-37)
	binary.BigEndian.PutUint32(buf[28:32], math.Float32bits(pdu.OrientationYaw))
	binary.BigEndian.PutUint32(buf[32:36], math.Float32bits(pdu.OrientationPitch))
	binary.BigEndian.PutUint32(buf[36:40], math.Float32bits(pdu.OrientationRoll))
	// Velocity (bytes 40-52)
	binary.BigEndian.PutUint32(buf[40:44], math.Float32bits(pdu.VelocityX))
	binary.BigEndian.PutUint32(buf[44:48], math.Float32bits(pdu.VelocityY))
	binary.BigEndian.PutUint32(buf[48:52], math.Float32bits(pdu.VelocityZ))
	// Location (bytes 52-64)
	binary.BigEndian.PutUint32(buf[52:56], math.Float32bits(pdu.LocationX))
	binary.BigEndian.PutUint32(buf[56:60], math.Float32bits(pdu.LocationY))
	binary.BigEndian.PutUint32(buf[60:64], math.Float32bits(pdu.LocationZ))
	return buf
}

func entityToPDU(e *Entity, siteID, appID uint16) *EntityStatePDU {
	x, y, z := ecefFromGeodetic(e.Lat, e.Lon, e.Alt)
	_ = x
	_ = y
	_ = z

	// Apply dead reckoning if needed
	drX, drY, drZ := e.applyDR()

	return &EntityStatePDU{
		ProtocolVersion:       7,
		ExerciseID:            1,
		PDUType:               1, // Entity State
		Timestamp:             timestampDIS(time.Now()),
		Length:                144,
		SiteID:                siteID,
		ApplicationID:         appID,
		EntityID:              e.EntityID,
		ForceID:               uint8(e.ForceID),
		EntityTypeKind:        e.TypeKind,
		EntityTypeDomain:      e.TypeDomain,
		EntityTypeCountry:     e.TypeCountry,
		EntityTypeCategory:    e.TypeCategory,
		EntityTypeSubcategory: e.TypeSubcat,
		EntityTypeSpecific:    e.TypeSpecific,
		OrientationYaw:        float32(e.Yaw),
		OrientationPitch:      float32(e.Pitch),
		OrientationRoll:       float32(e.Roll),
		VelocityX:             float32(e.VX),
		VelocityY:             float32(e.VY),
		VelocityZ:             float32(e.VZ),
		LocationX:             float32(drX),
		LocationY:             float32(drY),
		LocationZ:             float32(drZ),
	}
}

// Apply dead reckoning algorithm
func (e *Entity) applyDR() (x, y, z float64) {
	elapsed := time.Since(e.LastUpdate).Seconds()

	switch e.DRMAlgorithm {
	case DRMStatic:
		x, y, z = e.DRPX, e.DRPY, e.DRPZ
	case DRMFPM:
		x, y, z = e.DRPX, e.DRPY, e.DRPZ
	case DRMFVM, DRMRVM:
		// Linear extrapolation: P(t) = P0 + V*t
		x = e.DRPX + e.DRVX*elapsed
		y = e.DRPY + e.DRVY*elapsed
		z = e.DRPZ + e.DRVZ*elapsed
	default:
		x, y, z = e.X, e.Y, e.Z
	}
	return
}

// Initialize dead reckoning from current state
func (e *Entity) initDR() {
	e.DRPX, e.DRPY, e.DRPZ = e.X, e.Y, e.Z
	e.DRVX, e.DRVY, e.DRVZ = e.VX, e.VY, e.VZ
}

type entityManager struct {
	entities   map[uint32]*Entity
	siteID     uint16
	appID      uint16
	nextEntity uint32
}

func newEntityManager(siteID, appID uint16) *entityManager {
	return &entityManager{
		entities:   make(map[uint32]*Entity),
		siteID:     siteID,
		appID:      appID,
		nextEntity: (uint32(siteID) << 24) | (uint32(appID) << 16) | 1,
	}
}

func (em *entityManager) addEntity(e *Entity) uint32 {
	if e.EntityID == 0 {
		e.EntityID = em.nextEntity
		em.nextEntity++
	}
	e.LastUpdate = time.Now()
	e.initDR()
	em.entities[e.EntityID] = e
	return e.EntityID
}

func (em *entityManager) removeEntity(id uint32) {
	delete(em.entities, id)
}

func (em *entityManager) updateEntity(id uint32, lat, lon, alt float64, vx, vy, vz float64) {
	if e, ok := em.entities[id]; ok {
		e.Lat, e.Lon, e.Alt = lat, lon, alt
		e.VX, e.VY, e.VZ = vx, vy, vz
		e.X, e.Y, e.Z = ecefFromGeodetic(lat, lon, alt)
		e.LastUpdate = time.Now()

		// Switch to velocity-based DR when moving
		speed := math.Sqrt(vx*vx + vy*vy + vz*vz)
		if speed > 1.0 {
			e.DRMAlgorithm = DRMFVM
		}
	}
}

func (em *entityManager) cleanup() time.Duration {
	now := time.Now()
	var minInterval time.Duration = time.Hour

	for _, e := range em.entities {
		interval := e.HeartbeatInt
		if interval == 0 {
			interval = 5 * time.Second // Default heartbeat
		}

		if now.Sub(e.LastUpdate) > interval*3 {
			log.Printf("LVC: entity %d (%s) timed out, removing", e.EntityID, e.Marking)
			delete(em.entities, e.EntityID)
		} else if interval < minInterval {
			minInterval = interval
		}
	}

	return minInterval
}

var (
	em          *entityManager
	kafkaWriter *kafka.Writer
	kafkaReader *kafka.Reader
	kafkaBroker = getEnv("KAFKA_BROKERS", "kafka:9092")
	port        = getEnv("PORT", "8083")
	disSite     = uint16(mustAtoi(getEnv("DIS_SITE_ID", "1")))
	disApp      = uint16(mustAtoi(getEnv("DIS_APP_ID", "3")))
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustAtoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func run(ctx context.Context) {
	// Subscribe to fused tracks from sensor fusion
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{kafkaBroker},
		Topic:       TopicLE,
		GroupID:     "lvc-coordinator",
		MinBytes:    10e3,
		MaxBytes:    10e6,
		StartOffset: kafka.LastOffset,
	})
	defer reader.Close()

	// Publish entity state periodically
	heartbeatTick := time.NewTicker(1 * time.Second)
	defer heartbeatTick.Stop()

	cleanupTick := time.NewTicker(30 * time.Second)
	defer cleanupTick.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-cleanupTick.C:
			interval := em.cleanup()
			if interval > 0 {
				// Adjust heartbeat tick
			}
		case <-heartbeatTick.C:
			// Publish all local entity states
			em.publishAll()
		default:
			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}

			// Parse fused track
			var track struct {
				TrackNumber   uint32  `json:"track_number"`
				FusedLat      float64 `json:"fused_lat"`
				FusedLon      float64 `json:"fused_lon"`
				FusedAlt      float64 `json:"fused_alt"`
				FusedVelocity float64 `json:"fused_velocity"`
				FusedHeading  float64 `json:"fused_heading"`
				ThreatLevel   int     `json:"threat_level"`
				Confidence    float64 `json:"confidence"`
				UpdateCount   int     `json:"update_count"`
			}
			if err := json.Unmarshal(msg.Value, &track); err != nil {
				continue
			}

			// Convert to entity
			// ThreatLevel 4-5 = hostile, map to Opposing force
			forceID := ForceFriendly
			// _entityTypeKind := uint8(1) // unused // Platform
			entityTypeCategory := uint8(1) // Ground vehicle (default)

			if track.ThreatLevel >= 4 {
				forceID = ForceOpposing
				entityTypeCategory = 1 // Ballistic missile
			}

			// Calculate velocity from heading and speed
			headingRad := track.FusedHeading * math.Pi / 180
			vx := track.FusedVelocity * math.Cos(headingRad)
			vy := track.FusedVelocity * math.Sin(headingRad)
			vz := 0.0 // Simplified - no vertical velocity in track data

			entityID := em.findOrCreate(track.TrackNumber, forceID)
			em.updateEntity(entityID, track.FusedLat, track.FusedLon, track.FusedAlt, vx, vy, vz)

			// Update entity type based on threat
			if e, ok := em.entities[entityID]; ok {
				e.TypeCategory = entityTypeCategory
			}
		}
	}
}

func (em *entityManager) findOrCreate(trackNum uint32, forceID ForceID) uint32 {
	// Find existing entity with matching track number in marking
	marker := fmt.Sprintf("TRK-%d", trackNum)
	for _, e := range em.entities {
		if e.Marking == marker {
			return e.EntityID
		}
	}

	// Create new entity
	e := &Entity{
		EntityID:     0, // auto-assign
		SiteID:       em.siteID,
		AppID:        em.appID,
		ForceID:      forceID,
		TypeKind:     1,   // Platform
		TypeDomain:   1,   // Land (default)
		TypeCountry:  225, // USA
		TypeCategory: 1,
		TypeSubcat:   0,
		TypeSpecific: 0,
		State:        0, // Active
		Health:       1.0,
		Marking:      marker,
		LVCType:      "Constructive", // Derived from sensor fusion
		HeartbeatInt: 5 * time.Second,
		IsLocal:      true,
		DRMAlgorithm: DRMFVM,
	}
	return em.addEntity(e)
}

func (em *entityManager) publishAll() {
	for _, e := range em.entities {
		pdu := entityToPDU(e, em.siteID, em.appID)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := kafkaWriter.WriteMessages(ctx, kafka.Message{
			Key:   []byte(fmt.Sprintf("esp-%d", e.EntityID)),
			Value: pdu.Encode(),
			Headers: []kafka.Header{
				{Key: "pdu_type", Value: []byte("espdu")},
				{Key: "force_id", Value: []byte(fmt.Sprintf("%d", e.ForceID))},
				{Key: "lvc_type", Value: []byte(e.LVCType)},
			},
		})
		cancel()

		if err != nil {
			log.Printf("LVC: failed to publish entity %d: %v", e.EntityID, err)
		}
	}
}

type HealthResponse struct {
	Service   string    `json:"service"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"`
	Entities  int       `json:"entities"`
	DRMAlgo   string    `json:"drm_algorithm"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Service:   "lvc-coordinator",
		Version:   "0.1.0",
		Timestamp: time.Now().UTC(),
		Status:    "healthy",
		Entities:  len(em.entities),
		DRMAlgo:   "DRM_FVM",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	em = newEntityManager(disSite, disApp)

	kafkaWriter = &kafka.Writer{
		Addr:     kafka.TCP(kafkaBroker),
		Topic:    TopicDISOut,
		Balancer: &kafka.LeastBytes{},
	}
	defer kafkaWriter.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("LVC: shutting down...")
		cancel()
	}()

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/entities", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		entities := make([]*Entity, 0, len(em.entities))
		for _, e := range em.entities {
			entities = append(entities, e)
		}
		sort.Slice(entities, func(i, j int) bool {
			return entities[i].EntityID < entities[j].EntityID
		})
		json.NewEncoder(w).Encode(entities)
	})

	log.Printf("lvc-coordinator starting")
	log.Printf("DIS site=%d app=%d", disSite, disApp)
	log.Printf("Kafka broker: %s", kafkaBroker)
	go run(ctx)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("HTTP server on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
