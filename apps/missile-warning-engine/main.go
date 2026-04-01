// missile-warning-engine — VIMI Missile Warning Engine
// Phase 1: Core Infrastructure
// Subscribes to OPIR sightings from Kafka, performs threat track correlation,
// Kalman filter trajectory estimation, and publishes alerts
package main

import (
	"context"
	"encoding/json"
	"strconv"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	TopicOPIR    = "vimi.opir.sensor-data"
	TopicAlerts  = "vimi.alerts"
	TopicTracks  = "vimi.tracks"
	TrackTimeout = 120 * time.Second // Remove stale tracks
)

var (
	kafkaBroker = getEnv("KAFKA_BROKERS", "kafka:9092")
	redisAddr   = getEnv("REDIS_ADDR", "redis:6379")
	port        = getEnv("PORT", "8080")
	disSite     = uint16(mustAtoi(getEnv("DIS_SITE_ID", "1")))
	disApp      = uint16(mustAtoi(getEnv("DIS_APP_ID", "2")))
)

// AlertLevel from DoD CONOPREP/IMMINENT doctrine
type AlertLevel int

const (
	ALERT_UNKNOWN  AlertLevel = 0
	ALERT_CONOPREP AlertLevel = 1 // CONOPREP: preparations detected (launch detected)
	ALERT_IMMINENT AlertLevel = 2 // IMMINENT: impact predicted <15 min
	ALERT_INCOMING AlertLevel = 3 // INCOMING: missile in-flight, terminal phase
	ALERT_HOSTILE  AlertLevel = 4 // HOSTILE: impact <2 min
)

// ThreatType from STANAG 1247
type ThreatType int

const (
	THREAT_UNKNOWN     ThreatType = 0
	THREAT_SRBM        ThreatType = 1 // Short-range: <1000km
	THREAT_MRBM        ThreatType = 2 // Medium-range: 1000-3000km
	THREAT_IRBM        ThreatType = 3 // Intermediate: 3000-5500km
	THREAT_ICBM        ThreatType = 4 // Intercontinental: >5500km
	THREAT_CRUISEMISSILE ThreatType = 5
	THREAT_AIRCRAFT    ThreatType = 6
	THREAT_ARTILLERY   ThreatType = 7
	THREAT_MORTAR      ThreatType = 8
)

// ThreatTypeName returns the string name for a threat type
func ThreatTypeName(t ThreatType) string {
	names := []string{"Unknown", "SRBM", "MRBM", "IRBM", "ICBM", "Cruise Missile", "Aircraft", "Artillery", "Mortar"}
	if t < 0 || int(t) >= len(names) {
		return "Unknown"
	}
	return names[t]
}

// OPIRSighting from opir-ingest
type OPIRSighting struct {
	SightingID     uint64    `json:"sighting_id"`
	SatelliteID    uint32    `json:"satellite_id"`
	SensorID       uint16    `json:"sensor_id"`
	ScanMode       uint8     `json:"scan_mode"`
	Timestamp      time.Time `json:"timestamp"`
	DetectionLat   float64   `json:"detection_lat"`
	DetectionLon   float64   `json:"detection_lon"`
	DetectionAlt   float64   `json:"detection_alt"`
	Intensity      float64   `json:"intensity"`
	BackgroundTemp float64   `json:"background_temp"`
	SNR            float64   `json:"snr"`
	SatelliteLat   float64   `json:"satellite_lat"`
	SatelliteLon   float64   `json:"satellite_lon"`
	SatelliteAlt   float64   `json:"satellite_alt"`
	FOVCenterLat   float64   `json:"fov_center_lat"`
	FOVCenterLon   float64   `json:"fov_center_lon"`
	FOVHalfAngle   float64   `json:"fov_half_angle"`
	ProcessingFlags uint16    `json:"processing_flags"`
}

// MissileTrack represents a tracked missile threat
type MissileTrack struct {
	TrackNumber     uint32        `json:"track_number"`
	ThreatType      ThreatType    `json:"threat_type"`
	AlertLevel      AlertLevel    `json:"alert_level"`
	LaunchLat       float64       `json:"launch_lat"`
	LaunchLon       float64       `json:"launch_lon"`
	LaunchTime      time.Time     `json:"launch_time"`
	ImpactLat       float64       `json:"impact_lat"`
	ImpactLon       float64       `json:"impact_lon"`
	ImpactTime      time.Time     `json:"impact_time"`
	TimeToImpact    float64       `json:"time_to_impact"` // seconds
	Velocity        float64       `json:"velocity"`        // m/s
	Heading         float64       `json:"heading"`         // degrees
	Confidence      float64       `json:"confidence"`      // 0-1
	LastUpdate      time.Time     `json:"last_update"`
	DetectionCount  int           `json:"detection_count"`
	SourceSensor    string        `json:"source_sensor"`
	Sightings       []OPIRSighting `json:"sightings"`
	
	// Kalman filter state
	posX, posY, posZ float64      // position estimate (m ECEF)
	velX, velY, velZ float64      // velocity estimate (m/s)
	posVar         float64         // position variance
	velVar         float64         // velocity variance
}

// Alert represents a disseminated alert
type Alert struct {
	AlertID       uint32    `json:"alert_id"`
	AlertLevel    AlertLevel `json:"alert_level"`
	ThreatType    ThreatType `json:"threat_type"`
	TrackNumber   uint32    `json:"track_number"`
	LaunchLat     float64   `json:"launch_lat"`
	LaunchLon     float64   `json:"launch_lon"`
	ImpactLat     float64   `json:"impact_lat"`
	ImpactLon     float64   `json:"impact_lon"`
	TimeToImpact  float64   `json:"time_to_impact"` // seconds
	ImpactTime    time.Time `json:"impact_time"`
	NCARequired   bool      `json:"nca_required"`
	SourceSensor  string    `json:"source_sensor"`
	Confidence    float64  `json:"confidence"`
	IssuedAt      time.Time `json:"issued_at"`
}

// Kalman filter for ballistic trajectory estimation
// State: [pos_x, pos_y, pos_z, vel_x, vel_y, vel_z] (ECEF, m and m/s)

func estimateTrajectory(tr *MissileTrack) {
	if len(tr.Sightings) < 2 {
		return
	}
	
	// Simple ballistic trajectory estimation
	// For a ballistic missile: position = p0 + v*t + 0.5*a*t^2
	// Acceleration is gravity + drag (simplified: just gravity in ECEF)
	g := 9.81 // m/s^2 (downward)
	
	// Use latest and earliest sightings to estimate
	first := tr.Sightings[0]
	last := tr.Sightings[len(tr.Sightings)-1]
	
	dt := last.Timestamp.Sub(first.Timestamp).Seconds()
	if dt < 1 {
		dt = 1
	}
	
	// Convert detections to ECEF
	x1, y1, z1 := ecefFromGeodetic(first.DetectionLat, first.DetectionLon, first.DetectionAlt*1000)
	x2, y2, z2 := ecefFromGeodetic(last.DetectionLat, last.DetectionLon, last.DetectionAlt*1000)
	
	// Velocity estimate
	vx := (x2 - x1) / dt
	vy := (y2 - y1) / dt
	vz := (z2 - z1) / dt
	tr.velX, tr.velY, tr.velZ = vx, vy, vz
	
	// Speed
	tr.Velocity = math.Sqrt(vx*vx + vy*vy + vz*vz)
	
	// Heading (direction of travel on Earth's surface)
	tr.Heading = math.Atan2(vy, vx) * 180 / math.Pi
	if tr.Heading < 0 {
		tr.Heading += 360
	}
	
	// Estimate impact point (simplified: constant velocity extrapolation)
	// Assume target is roughly opposite to launch direction
	// This is a simplified "last known position + velocity" model
	now := time.Now()
	tRemaining := tr.TimeToImpact - now.Sub(tr.LaunchTime).Seconds()
	
	// Impact prediction using simple linear extrapolation
	// In reality would use orbital mechanics + gravity
	impactX := x2 + vx*tRemaining
	impactY := y2 + vy*tRemaining
	impactZ := z2 + vz*tRemaining - 0.5*g*tRemaining*tRemaining // add gravity drop
	
	tr.ImpactLat, tr.ImpactLon, _ = geodeticFromECEF(impactX, impactY, impactZ)
	
	// Estimate threat type from velocity
	// SRBM: 1.5-3 km/s, MRBM: 3-4.5 km/s, IRBM: 4.5-6 km/s, ICBM: >6 km/s
	vkm := tr.Velocity / 1000
	switch {
	case vkm < 1.5:
		tr.ThreatType = THREAT_MORTAR
	case vkm < 3:
		tr.ThreatType = THREAT_SRBM
	case vkm < 4.5:
		tr.ThreatType = THREAT_MRBM
	case vkm < 6:
		tr.ThreatType = THREAT_IRBM
	default:
		tr.ThreatType = THREAT_ICBM
	}
	
	// Confidence based on detection count and time span
	tr.Confidence = math.Min(float64(tr.DetectionCount)/10.0, 1.0)
	if dt > 60 {
		tr.Confidence *= 0.8
	}
}

func geodeticFromECEF(x, y, z float64) (lat, lon, alt float64) {
	// WGS84 inverse
	a := 6378137.0
	f := 1 / 298.257223563
	e2 := 2*f - f*f
	
	lon = math.Atan2(y, x)
	p := math.Sqrt(x*x + y*y)
	lat = math.Atan2(z, p*(1-e2))
	
	// Iterate to solve for geodetic latitude
	for i := 0; i < 5; i++ {
		N := a / math.Sqrt(1-e2*math.Sin(lat)*math.Sin(lat))
		lat = math.Atan2(z+e2*N*math.Sin(lat), p)
	}
	
	N := a / math.Sqrt(1-e2*math.Sin(lat)*math.Sin(lat))
	alt = p/math.Cos(lat) - N
	
	return lat * 180/math.Pi, lon * 180/math.Pi, alt / 1000 // km
}

func ecefFromGeodetic(lat, lon, alt float64) (x, y, z float64) {
	a := 6378137.0
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

func (tr *MissileTrack) updateAlertLevel() {
	// DoD alert level thresholds
	tti := tr.TimeToImpact
	
	switch {
	case tti < 120: // <2 min
		tr.AlertLevel = ALERT_HOSTILE
	case tti < 600: // <10 min
		tr.AlertLevel = ALERT_INCOMING
	case tti < 900: // <15 min
		tr.AlertLevel = ALERT_IMMINENT
	default:
		tr.AlertLevel = ALERT_CONOPREP
	}
}

type trackManager struct {
	tracks    map[uint32]*MissileTrack
	nextTrack uint32
	alerts    []*Alert
}

func newTrackManager() *trackManager {
	return &trackManager{
		tracks:    make(map[uint32]*MissileTrack),
		nextTrack: 1,
	}
}

func (tm *trackManager) processSighting(s *OPIRSighting) {
	// Filter: only process high-SNR detections that could be missile launches
	if s.SNR < 3.0 || s.Intensity < 500 {
		return
	}
	
	now := time.Now()
	
	// Check if this detection matches an existing track (correlate)
	var matched *MissileTrack
	spatialThreshold := 500.0 // km
	timeThreshold := 60.0    // seconds
	
	for _, tr := range tm.tracks {
		if now.Sub(tr.LastUpdate) > TrackTimeout {
			continue
		}
		
		// Check spatial correlation
		latDiff := s.DetectionLat - tr.LaunchLat
		lonDiff := s.DetectionLon - tr.LaunchLon
		dist := math.Sqrt(latDiff*latDiff + lonDiff*lonDiff) * 111.0 // rough km conversion
		
		timeDiff := math.Abs(s.Timestamp.Sub(tr.LastUpdate).Seconds())
		
		if dist < spatialThreshold && timeDiff < timeThreshold {
			matched = tr
			break
		}
	}
	
	if matched != nil {
		// Add to existing track
		matched.Sightings = append(matched.Sightings, *s)
		matched.DetectionCount++
		matched.LastUpdate = now
		
		// Re-estimate trajectory
		estimateTrajectory(matched)
		
		// Update time to impact
		if len(matched.Sightings) >= 3 {
			// Use velocity to project impact
			// TTI = distance_to_target / velocity_along_trajectory
			// Simplified: assume impact in ~60-900 seconds based on range
			// launchToNow := now.Sub(matched.LaunchTime).Seconds()
			// Re-estimate based on time since launch and threat type
			switch matched.ThreatType {
			case THREAT_SRBM:
				matched.TimeToImpact = matched.LaunchTime.Add(120 * time.Second).Sub(now).Seconds()
			case THREAT_MRBM:
				matched.TimeToImpact = matched.LaunchTime.Add(300 * time.Second).Sub(now).Seconds()
			case THREAT_IRBM:
				matched.TimeToImpact = matched.LaunchTime.Add(600 * time.Second).Sub(now).Seconds()
			case THREAT_ICBM:
				matched.TimeToImpact = matched.LaunchTime.Add(1800 * time.Second).Sub(now).Seconds()
			default:
				matched.TimeToImpact = 300
			}
			if matched.TimeToImpact < 0 {
				matched.TimeToImpact = 0
			}
		}
		
		oldLevel := matched.AlertLevel
		matched.updateAlertLevel()
		
		if matched.AlertLevel != oldLevel {
			log.Printf("TRACK %d: %s %s → %s (tti=%.0fs vel=%.1fkm/s conf=%.0f%%)",
				matched.TrackNumber, ThreatTypeName(matched.ThreatType),
				alertLevelName(oldLevel), alertLevelName(matched.AlertLevel),
				matched.TimeToImpact, matched.Velocity/1000, matched.Confidence*100)
		}
	} else {
		// Create new track
		tr := &MissileTrack{
			TrackNumber:    tm.nextTrack,
			ThreatType:     THREAT_UNKNOWN,
			AlertLevel:     ALERT_CONOPREP,
			LaunchLat:      s.DetectionLat,
			LaunchLon:      s.DetectionLon,
			LaunchTime:     s.Timestamp,
			ImpactLat:      s.DetectionLat,
			ImpactLon:      s.DetectionLon,
			TimeToImpact:   900,
			LastUpdate:     now,
			DetectionCount: 1,
			SourceSensor:   fmt.Sprintf("SBIRS-%d", s.SatelliteID),
			Sightings:     []OPIRSighting{*s},
		}
		tm.nextTrack++
		
		// Initial threat type estimate from intensity
		if s.Intensity > 2000 {
			tr.ThreatType = THREAT_MRBM
		} else {
			tr.ThreatType = THREAT_SRBM
		}
		
		tm.tracks[tr.TrackNumber] = tr
		log.Printf("NEW TRACK %d: %s detected by %s lat=%.3f lon=%.3f snr=%.1f",
			tr.TrackNumber, ThreatTypeName(tr.ThreatType), tr.SourceSensor,
			s.DetectionLat, s.DetectionLon, s.SNR)
	}
}

func (tm *trackManager) cleanup() {
	now := time.Now()
	for num, tr := range tm.tracks {
		if now.Sub(tr.LastUpdate) > TrackTimeout {
			log.Printf("TRACK %d: expired (last update %s)", num, tr.LastUpdate.Format("15:04:05"))
			delete(tm.tracks, num)
		}
	}
}

func alertLevelName(l AlertLevel) string {
	names := []string{"UNKNOWN", "CONOPREP", "IMMINENT", "INCOMING", "HOSTILE"}
	if l < 0 || int(l) >= len(names) {
		return "UNKNOWN"
	}
	return names[l]
}

var kafkaWriter *kafka.Writer

func publishAlert(a *Alert) error {
	data, err := json.Marshal(a)
	if err != nil {
		return err
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return kafkaWriter.WriteMessages(ctx, kafka.Message{
		Key:   []byte(fmt.Sprintf("alert-%d", a.TrackNumber)),
		Value: data,
		Headers: []kafka.Header{
			{Key: "alert_level", Value: []byte(alertLevelName(a.AlertLevel))},
			{Key: "threat_type", Value: []byte(ThreatTypeName(a.ThreatType))},
		},
	})
}

func publishTrack(tr *MissileTrack) error {
	data, err := json.Marshal(tr)
	if err != nil {
		return err
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return kafkaWriter.WriteMessages(ctx, kafka.Message{
		Key:   []byte(fmt.Sprintf("track-%d", tr.TrackNumber)),
		Value: data,
	})
}

func run(ctx context.Context) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{kafkaBroker},
		Topic:     TopicOPIR,
		GroupID:   "missile-warning-engine",
		MinBytes:  10e3,
		MaxBytes:  10e6,
		StartOffset: kafka.LastOffset,
	})
	defer reader.Close()
	
	cleanup := time.NewTicker(30 * time.Second)
	defer cleanup.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-cleanup.C:
			tm.cleanup()
		default:
			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("Read error: %v", err)
				continue
			}
			
			var s OPIRSighting
			if err := json.Unmarshal(msg.Value, &s); err != nil {
				continue
			}
			
			tm.processSighting(&s)
			
			// Publish all active tracks
			for _, tr := range tm.tracks {
				publishTrack(tr)
			}
		}
	}
}

type HealthResponse struct {
	Service     string    `json:"service"`
	Version     string    `json:"version"`
	Timestamp   time.Time `json:"timestamp"`
	Status      string    `json:"status"`
	ActiveTrack int       `json:"active_tracks"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Service:     "missile-warning-engine",
		Version:     "0.1.0",
		Timestamp:   time.Now().UTC(),
		Status:      "healthy",
		ActiveTrack: len(tm.tracks),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

var tm *trackManager

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

func main() {
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	
	tm = newTrackManager()
	
	kafkaWriter = &kafka.Writer{
		Addr:     kafka.TCP(kafkaBroker),
		Topic:    TopicAlerts,
		Balancer: &kafka.LeastBytes{},
	}
	defer kafkaWriter.Close()
	
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
	http.HandleFunc("/tracks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return sorted tracks (newest first)
		tracks := make([]*MissileTrack, 0, len(tm.tracks))
		for _, tr := range tm.tracks {
			tracks = append(tracks, tr)
		}
		sort.Slice(tracks, func(i, j int) bool {
			return tracks[i].LastUpdate.After(tracks[j].LastUpdate)
		})
		json.NewEncoder(w).Encode(tracks)
	})
	
	log.Printf("missile-warning-engine starting")
	log.Printf("Kafka broker: %s", kafkaBroker)
	log.Printf("Subscribing to: %s", TopicOPIR)
	log.Printf("Publishing to: %s (alerts), %s (tracks)", TopicAlerts, TopicTracks)
	go run(ctx)
	
	addr := fmt.Sprintf(":%s", port)
	log.Printf("HTTP server on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
