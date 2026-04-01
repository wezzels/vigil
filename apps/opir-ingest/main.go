// opir-ingest — VIMI OPIR Satellite Ingest Service
// Phase 1: Core Infrastructure
// Simulates SBIRS/NG-OPIR infrared satellite sensor data and publishes to Kafka
package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"strconv"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
)

const (
	TopicOPIR = "vimi.opir.sensor-data"
	SiteID    = 1 // VIMI FORGE Site
	AppID     = 1 // OPIR Ingest Application
)

var (
	kafkaBroker = getEnv("KAFKA_BROKERS", "kafka:9092")
	redisAddr   = getEnv("REDIS_ADDR", "redis:6379")
	port        = getEnv("PORT", "8080")
	disSite     = uint16(mustAtoi(getEnv("DIS_SITE_ID", "1")))
	disApp      = uint16(mustAtoi(getEnv("DIS_APP_ID", "1")))
	
	writer *kafka.Writer
	mode   = flag.String("mode", "simulate", "simulate|replay")
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

// OPIRSighting represents a single IR detection from a satellite sensor
type OPIRSighting struct {
	SightingID       uint64    `json:"sighting_id"`
	SatelliteID      uint32    `json:"satellite_id"`   // e.g., 1=SBIRS-High, 2=NG-OPIR
	SensorID         uint16    `json:"sensor_id"`
	ScanMode         uint8     `json:"scan_mode"`      // 0=Survey, 1=Sector, 2=Spot
	Timestamp        time.Time `json:"timestamp"`
	DetectionLat     float64   `json:"detection_lat"`  // degrees
	DetectionLon     float64   `json:"detection_lon"`  // degrees
	DetectionAlt     float64   `json:"detection_alt"`  // km
	Intensity        float64   `json:"intensity"`      // W/sr (watts per steradian)
	BackgroundTemp   float64   `json:"background_temp"` // K
	SNR              float64   `json:"snr"`             // signal-to-noise ratio
	SatelliteLat     float64   `json:"satellite_lat"`   // satellite position
	SatelliteLon     float64   `json:"satellite_lon"`
	SatelliteAlt     float64   `json:"satellite_alt"`   // km
	FOVCenterLat     float64   `json:"fov_center_lat"`
	FOVCenterLon     float64   `json:"fov_center_lon"`
	FOVHalfAngle     float64   `json:"fov_half_angle"`  // degrees
	ProcessingFlags  uint16    `json:"processing_flags"`
}

// DIS Entity State PDU for OPIR satellite (simplified)
type EntityStatePDU struct {
	ProtocolVersion uint16
	ExerciseID      uint8
	PDUType         uint8
	Timestamp       uint32
	Length          uint16
	SiteID          uint16
	ApplicationID   uint16
	EntityID        uint32
	ForceID         uint8
	EntityTypeKind  uint8
	EntityTypeDomain uint8
	EntityTypeCountry uint16
	EntityTypeCategory uint8
	EntityTypeSubcategory uint8
	EntityTypeSpecific uint8
	LocationX       float32 // meters (ECEF)
	LocationY       float32
	LocationZ       float32
	OrientationYaw  float32 // radians
	OrientationPitch float32
	OrientationRoll float32
	VelocityX       float32 // m/s
	VelocityY       float32
	VelocityZ       float32
}

// Encode entity state PDU per IEEE 1278.1 (big-endian)
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
	// Padding bytes 26-35
	binary.BigEndian.PutUint32(buf[36:40], math.Float32bits(pdu.LocationX))
	binary.BigEndian.PutUint32(buf[40:44], math.Float32bits(pdu.LocationY))
	binary.BigEndian.PutUint32(buf[44:48], math.Float32bits(pdu.LocationZ))
	binary.BigEndian.PutUint32(buf[48:52], math.Float32bits(pdu.OrientationYaw))
	binary.BigEndian.PutUint32(buf[52:56], math.Float32bits(pdu.OrientationPitch))
	binary.BigEndian.PutUint32(buf[56:60], math.Float32bits(pdu.OrientationRoll))
	binary.BigEndian.PutUint32(buf[60:64], math.Float32bits(pdu.VelocityX))
	binary.BigEndian.PutUint32(buf[64:68], math.Float32bits(pdu.VelocityY))
	binary.BigEndian.PutUint32(buf[68:72], math.Float32bits(pdu.VelocityZ))
	return buf
}

func ecefFromGeodetic(lat, lon, alt float64) (x, y, z float64) {
	// WGS84 ellipsoid
	a := 6378137.0 // semi-major axis (m)
	f := 1 / 298.257223563
	e2 := 2*f - f*f

	latRad := lat * math.Pi / 180
	lonRad := lon * math.Pi / 180

	N := a / math.Sqrt(1-e2*math.Sin(latRad)*math.Sin(latRad))
	x = (N + alt) * math.Cos(latRad) * math.Cos(lonRad)
	y = (N + alt) * math.Cos(latRad) * math.Sin(lonRad)
	z = (N*(1-e2) + alt) * math.Sin(latRad)
	return
}

func timestampDIS(t time.Time) uint32 {
	// DIS timestamp: 1/10 ms since hour
	ms := uint64(t.UnixMilli() % 3600000)
	return uint32(ms / 10)
}

// Simulate a satellite orbit (geostationary approximation)
func satellitePosition(t time.Time, baseLon float64) (lat, lon, alt float64) {
	// Geostationary: fixed longitude, slight inclination
	lon = baseLon
	lat = 0.1 * math.Sin(float64(t.Unix())/3600.0*0.05) // tiny inclination wobble
	alt = 35786.0 // km
	return
}

// Generate a random IR detection
func generateSighting(id uint64, t time.Time) *OPIRSighting {
	// Satellite types: 0=SBIRS-High, 1=NG-OPIR
	satType := rand.Intn(2)
	satellites := []struct {
		id       uint32
		baseLon  float64
		fovAngle float64
	}{
		{1, -80.0, 8.0},   // SBIRS-High over CONUS
		{2, 0.0, 10.0},    // NG-OPIR demo
	}
	sat := satellites[satType]

	satLat, satLon, satAlt := satellitePosition(t, sat.baseLon)
	satX, satY, satZ := ecefFromGeodetic(satLat, satLon, satAlt)
	_ = satX; _ = satY; _ = satZ

	// Detection on Earth's surface (random point in FOV)
	detLat := satLat + (rand.Float64()-0.5)*sat.fovAngle
	detLon := satLon + (rand.Float64()-0.5)*sat.fovAngle*1.5
	detAlt := 0.0 // sea level

	// IR intensity: missile launch = hot, background = cold
	// Plume signature: 1500-3000 K effective temp vs 200-300 K background
	isPlume := rand.Float64() < 0.05 // 5% chance of detection (realistic revisit rate)
	var intensity float64
	var snr float64
	var background float64
	
	if isPlume {
		intensity = 2000.0 + rand.ExpFloat64()*1000.0 // W/sr, missile-like
		snr = 5.0 + rand.ExpFloat64()*5.0
		background = 220.0 + rand.Float64()*60.0
	} else {
		// Background clutter (clouds, terrain)
		intensity = 150.0 + rand.ExpFloat64()*50.0
		snr = 1.0 + rand.Float64()*2.0
		background = 250.0 + rand.Float64()*30.0
	}

	return &OPIRSighting{
		SightingID:     id,
		SatelliteID:    sat.id,
		SensorID:       1,
		ScanMode:       0, // Survey
		Timestamp:      t,
		DetectionLat:   detLat,
		DetectionLon:   detLon,
		DetectionAlt:   detAlt,
		Intensity:      intensity,
		BackgroundTemp: background,
		SNR:            snr,
		SatelliteLat:   satLat,
		SatelliteLon:   satLon,
		SatelliteAlt:   satAlt,
		FOVCenterLat:   satLat,
		FOVCenterLon:   satLon,
		FOVHalfAngle:   sat.fovAngle,
		ProcessingFlags: 0,
	}
}

// Publish sighting to Kafka
func publishSighting(s *OPIRSighting) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	err = writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(fmt.Sprintf("%d-%d", s.SatelliteID, s.SightingID)),
		Value: data,
		Time:  s.Timestamp,
		Headers: []kafka.Header{
			{Key: "satellite_id", Value: []byte(fmt.Sprintf("%d", s.SatelliteID))},
			{Key: "sensor_id", Value: []byte(fmt.Sprintf("%d", s.SensorID))},
		},
	})
	if err != nil {
		return fmt.Errorf("kafka write: %w", err)
	}

	log.Printf("opir-ingest: sighting %d satellite=%d lat=%.2f lon=%.2f intensity=%.1f snr=%.1f",
		s.SightingID, s.SatelliteID, s.DetectionLat, s.DetectionLon, s.Intensity, s.SNR)
	return nil
}

// Publish entity state for satellite to Kafka DIS topic
func publishSatelliteESP(satID, entityID uint32, lat, lon, alt float64, t time.Time) error {
	x, y, z := ecefFromGeodetic(lat, lon, alt)
	
	pdu := EntityStatePDU{
		ProtocolVersion:  7,
		ExerciseID:       1,
		PDUType:          1, // Entity State
		Timestamp:        timestampDIS(t),
		Length:           144,
		SiteID:           uint16(disSite),
		ApplicationID:    uint16(disApp),
		EntityID:         entityID,
		ForceID:          0,
		EntityTypeKind:   1,  // Platform
		EntityTypeDomain: 5,  // Space
		EntityTypeCountry: 225, // USA
		EntityTypeCategory: 1, // Spacecraft
		EntityTypeSubcategory: 0,
		EntityTypeSpecific: 1, // Surveillance satellite
		LocationX:        float32(x),
		LocationY:        float32(y),
		LocationZ:        float32(z),
		OrientationYaw:   0,
		OrientationPitch: 0,
		OrientationRoll:  0,
		VelocityX:        0,
		VelocityY:        0,
		VelocityZ:        0,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(fmt.Sprintf("esp-%d", satID)),
		Value: pdu.Encode(),
		Headers: []kafka.Header{
			{Key: "pdu_type", Value: []byte("espdu")},
			{Key: "dis_version", Value: []byte("7")},
		},
	})
	return err
}

func ensureTopic(topic string) error {
	conn, err := kafka.Dial("tcp", kafkaBroker)
	if err != nil {
		return fmt.Errorf("kafka dial: %w", err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("kafka controller: %w", err)
	}

	controllerConn, err := kafka.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	if err != nil {
		return fmt.Errorf("kafka controller dial: %w", err)
	}
	defer controllerConn.Close()

	err = controllerConn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     3,
		ReplicationFactor: 1,
	})
	if err != nil {
		// Topic might already exist
		log.Printf("topic %s: %v", topic, err)
	}
	return nil
}

func runSimulation(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var sightingCount uint64 = 0
	entityID := uint32(disSite<<16 | disApp<<8 | 1)

	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			// Publish satellite entity state
			satLat, satLon, satAlt := satellitePosition(t, -80.0)
			publishSatelliteESP(1, entityID, satLat, satLon, satAlt, t)

			// Generate and publish OPIR sightings
			sightingCount++
			s := generateSighting(sightingCount, t)
			if err := publishSighting(s); err != nil {
				log.Printf("ERROR publishing sighting: %v", err)
			}
		}
	}
}

type HealthResponse struct {
	Service    string    `json:"service"`
	Version    string    `json:"version"`
	Timestamp  time.Time `json:"timestamp"`
	Status     string    `json:"status"`
	Kafka      string    `json:"kafka"`
	Redis      string    `json:"redis"`
	Simulation string    `json:"simulation"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Service:    "opir-ingest",
		Version:    "0.1.0",
		Timestamp:  time.Now().UTC(),
		Status:     "healthy",
		Kafka:      "connected",
		Redis:      "not_used_yet",
		Simulation: *mode,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Ensure Kafka topic exists
	if err := ensureTopic(TopicOPIR); err != nil {
		log.Printf("Warning: could not create topic: %v", err)
	}

	// Create Kafka writer
	writer = &kafka.Writer{
		Addr:         kafka.TCP(kafkaBroker),
		Topic:        TopicOPIR,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async:        false,
	}
	defer writer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Shutting down...")
		cancel()
	}()

	// HTTP endpoints
		http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/ready", healthHandler)

	// Start simulation
	log.Printf("opir-ingest starting in %s mode", *mode)
	log.Printf("Kafka broker: %s, topic: %s", kafkaBroker, TopicOPIR)
	log.Printf("DIS site=%d app=%d", disSite, disApp)
	go runSimulation(ctx)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("HTTP server on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
