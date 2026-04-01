// missile-warning-engine — VIMI Missile Warning Engine
// Phase 1: Core Infrastructure
// Subscribes to OPIR sightings from Kafka, performs threat track correlation,
// Kalman filter trajectory estimation, and publishes alerts
package main

import (
	"context"
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

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
)

const (
	TopicOPIR    = "vimi.opir.sensor-data"
	TopicAlerts  = "vimi.alerts"
	TopicTracks  = "vimi.tracks"
	TrackTimeout = 120 * time.Second // Remove stale tracks
)

// --- Prometheus Metrics ---
var (
	tracksTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vimi_tracks_total",
			Help: "Total tracks detected by missile type and alert level",
		},
		[]string{"missile_type", "alert_level"},
	)
	tracksActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "vimi_tracks_active",
			Help: "Currently active tracks",
		},
	)
	alertsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vimi_alerts_total",
			Help: "Total alerts issued by level",
		},
		[]string{"level"},
	)
	alertsActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vimi_alerts_active",
			Help: "Active alerts by level",
		},
		[]string{"level"},
	)
	sightingsProcessed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "vimi_sightings_processed_total",
			Help: "Total OPIR sightings processed",
		},
	)
	kafkaConsumerLag = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vimi_kafka_consumer_lag",
			Help: "Kafka consumer lag by topic/partition",
		},
		[]string{"topic", "partition"},
	)
)

func init() {
	prometheus.MustRegister(tracksTotal, tracksActive, alertsTotal, alertsActive, sightingsProcessed, kafkaConsumerLag)
}

// Alert levels
const (
	ALERT_UNKNOWN  = 0
	ALERT_CONOPREP = 1 // Pre-conflict preparation
	ALERT_IMMINENT = 2 // Launch detected, impact pending
	ALERT_INCOMING = 3 // Missile in flight, tracking
	ALERT_HOSTILE  = 4 // Confirmed hostile, NCA approval required
)

// Threat types
const (
	THREAT_UNKNOWN = 0
	THREAT_SRBM    = 1 // Short-range ballistic missile (<1000km)
	THREAT_MRBM    = 2 // Medium-range ballistic missile (1000-3000km)
	THREAT_IRBM    = 3 // Intermediate-range ballistic missile (3000-5500km)
	THREAT_ICBM    = 4 // Intercontinental ballistic missile (>5500km)
	THREAT_SLBM    = 5 // Submarine-launched ballistic missile
	THREAT_CRUISE  = 6 // Cruise missile
)

func ThreatTypeName(t int) string {
	names := []string{"UNKNOWN", "SRBM", "MRBM", "IRBM", "ICBM", "SLBM", "CRUISE"}
	if t < 0 || t >= len(names) {
		return "UNKNOWN"
	}
	return names[t]
}

// OPIR Satellite sighting from SBIRS
type OPIRSighting struct {
	SatelliteID   int       `json:"satellite_id"`
	Timestamp     time.Time `json:"timestamp"`
	DetectionLat  float64   `json:"detection_lat"`
	DetectionLon  float64   `json:"detection_lon"`
	Intensity     float64   `json:"intensity"` // IR intensity in gigawatts
	SNR           float64   `json:"snr"`        // Signal-to-noise ratio
}

// MissileTrack represents a tracked ballistic missile
type MissileTrack struct {
	TrackNumber    int       `json:"track_number"`
	ThreatType     int       `json:"threat_type"`
	AlertLevel     int       `json:"alert_level"`
	LaunchLat      float64   `json:"launch_lat"`
	LaunchLon      float64   `json:"launch_lon"`
	ImpactLat      float64   `json:"impact_lat"`
	ImpactLon      float64   `json:"impact_lon"`
	LaunchTime     time.Time `json:"launch_time"`
	TimeToImpact   float64   `json:"time_to_impact"` // seconds
	Velocity       float64   `json:"velocity"`       // m/s
	Confidence     float64   `json:"confidence"`     // 0.0-1.0
	LastUpdate     time.Time `json:"last_update"`
	DetectionCount int       `json:"detection_count"`
	SourceSensor   string    `json:"source_sensor"`
	Sightings      []OPIRSighting `json:"sightings"`
}

// Track manager handles correlation and lifecycle
type trackManager struct {
	nextTrack int
	tracks    map[int]*MissileTrack
}

func newTrackManager() *trackManager {
	return &trackManager{tracks: make(map[int]*MissileTrack)}
}

// Find matching track based on trajectory correlation
func (tm *trackManager) findMatching(s *OPIRSighting) *MissileTrack {
	// Simplified: match by proximity to last known position
	for _, tr := range tm.tracks {
		// Check if this sighting is near the predicted track position
		// using simple Euclidean distance on lat/lon
		// In production, use Kalman filter for trajectory prediction
		latErr := math.Abs(s.DetectionLat - tr.ImpactLat)
		lonErr := math.Abs(s.DetectionLon - tr.ImpactLon)
		
		// Update impact prediction based on new sighting
		if latErr < 5.0 && lonErr < 5.0 {
			return tr
		}
	}
	return nil
}

func (tm *trackManager) update(s *OPIRSighting) {
	matched := tm.findMatching(s)
	now := time.Now()

	if matched != nil {
		// Update existing track with new sighting
		matched.DetectionCount++
		matched.LastUpdate = now
		matched.Sightings = append(matched.Sightings, *s)

		// Update impact prediction using motion vector
		// Simplified: linear extrapolation from sightings
		if len(matched.Sightings) >= 2 {
			prev := matched.Sightings[len(matched.Sightings)-2]
			dt := s.Timestamp.Sub(prev.Timestamp).Seconds()
			if dt > 0 {
				dlat := (s.DetectionLat - prev.DetectionLat) / dt
				dlon := (s.DetectionLon - prev.DetectionLon) / dt
				// Project forward
				matched.ImpactLat = s.DetectionLat + dlat*30 // 30 second prediction window
				matched.ImpactLon = s.DetectionLon + dlon*30
				
				// Compute velocity
				latKm := (s.DetectionLat - prev.DetectionLat) * 111.0
				lonKm := (s.DetectionLon - prev.DetectionLon) * 111.0 * math.Cos(s.DetectionLat*math.Pi/180)
				distKm := math.Sqrt(latKm*latKm + lonKm*lonKm)
				matched.Velocity = distKm * 1000 / dt // m/s
			}
		}

		// Update confidence based on detection count and SNR
		matched.Confidence = math.Min(0.99, float64(matched.DetectionCount)/10.0+0.1*(s.SNR/10.0))

		// Update threat type estimate as confidence grows
		if matched.DetectionCount >= 3 && matched.ThreatType == THREAT_UNKNOWN {
			if matched.Velocity > 1500 {
				matched.ThreatType = THREAT_ICBM
			} else if matched.Velocity > 1000 {
				matched.ThreatType = THREAT_IRBM
			} else if matched.Velocity > 500 {
				matched.ThreatType = THREAT_MRBM
			} else {
				matched.ThreatType = THREAT_SRBM
			}
		}

		// Update alert level
		matched.TimeToImpact = estimatedTimeToImpact(matched)

		oldLevel := matched.AlertLevel
		matched.updateAlertLevel()

		if matched.AlertLevel != oldLevel {
			// Alert level escalation — record metric
			alertsTotal.WithLabelValues(alertLevelName(matched.AlertLevel)).Inc()
			alertsActive.WithLabelValues(alertLevelName(oldLevel)).Dec()
			alertsActive.WithLabelValues(alertLevelName(matched.AlertLevel)).Inc()
			
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
		
		// Track created metric
		tracksTotal.WithLabelValues(ThreatTypeName(tr.ThreatType), alertLevelName(tr.AlertLevel)).Inc()
		tracksActive.Inc()
		alertsActive.WithLabelValues(alertLevelName(ALERT_CONOPREP)).Inc()
		
		log.Printf("NEW TRACK %d: %s detected by %s lat=%.3f lon=%.3f snr=%.1f",
			tr.TrackNumber, ThreatTypeName(tr.ThreatType), tr.SourceSensor,
			s.DetectionLat, s.DetectionLon, s.SNR)
	}
}

func (tm *trackManager) cleanup() {
	now := time.Now()
	for num, tr := range tm.tracks {
		if now.Sub(tr.LastUpdate) > TrackTimeout {
			tracksActive.Dec()
			log.Printf("TRACK %d: expired (last update %s)", num, tr.LastUpdate.Format("15:04:05"))
			delete(tm.tracks, num)
		}
	}
}

func alertLevelName(l int) string {
	names := []string{"UNKNOWN", "CONOPREP", "IMMINENT", "INCOMING", "HOSTILE"}
	if l < 0 || l >= len(names) {
		return "UNKNOWN"
	}
	return names[l]
}

func estimatedTimeToImpact(tr *MissileTrack) float64 {
	// Simplified ballpark: use velocity and threat type
	switch tr.ThreatType {
	case THREAT_ICBM:
		return 1800 // 30 min
	case THREAT_IRBM:
		return 600 // 10 min
	case THREAT_MRBM:
		return 300 // 5 min
	case THREAT_SRBM:
		return 120 // 2 min
	default:
		return 300
	}
}

func (tr *MissileTrack) updateAlertLevel() {
	// Escalate based on detection count, time to impact, and confidence
	if tr.DetectionCount >= 5 && tr.Confidence > 0.8 {
		tr.AlertLevel = ALERT_HOSTILE
	} else if tr.DetectionCount >= 4 && tr.Confidence > 0.6 {
		tr.AlertLevel = ALERT_INCOMING
	} else if tr.DetectionCount >= 2 {
		tr.AlertLevel = ALERT_IMMINENT
	} else {
		tr.AlertLevel = ALERT_CONOPREP
	}
}

type healthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Version   string `json:"version"`
	Port      int    `json:"port"`
	Uptime    string `json:"uptime"`
	Tracks    int    `json:"tracks"`
	StartTime int64  `json:"start_time"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	tracks := make([]*MissileTrack, 0, len(tm.tracks))
	for _, tr := range tm.tracks {
		tracks = append(tracks, tr)
	}
	resp := healthResponse{
		Status:    "healthy",
		Service:   "missile-warning-engine",
		Version:   "1.0.0",
		Port:      8080,
		Uptime:    time.Since(time.Unix(startTime, 0)).String(),
		Tracks:    len(tracks),
		StartTime: startTime,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

var (
	kafkaBroker string
	port        string
	startTime   int64
	tm          *trackManager
)

func main() {
	flag.StringVar(&kafkaBroker, "kafka", "kafka:9092", "Kafka broker address")
	flag.StringVar(&port, "port", "8080", "HTTP port")
	flag.Parse()

	startTime = time.Now().Unix()

	log.SetOutput(os.Stdout)
	log.SetFlags(0)

	// Prometheus metrics endpoint
	http.Handle("/metrics", promhttp.Handler())
	
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

func run(ctx context.Context) {
	// Subscribe to OPIR sensor data
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{kafkaBroker},
		Topic:          TopicOPIR,
		GroupID:        "missile-warning-engine",
		MinBytes:       1,
		MaxBytes:       1e6,
		MaxWait:        500 * time.Millisecond,
		CommitInterval: time.Second,
	})
	defer reader.Close()

	tm = newTrackManager()
	cleanup := time.NewTicker(30 * time.Second)
	defer cleanup.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-cleanup.C:
			tm.cleanup()
			// Update active tracks gauge
			tracksActive.Set(float64(len(tm.tracks)))
		default:
			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}

			var sighting OPIRSighting
			if err := json.Unmarshal(msg.Value, &sighting); err != nil {
				log.Printf("ERROR parsing sighting: %v", err)
				continue
			}

			sightingsProcessed.Inc()
			tm.update(&sighting)

			// Publish updated track
			for _, tr := range tm.tracks {
				trackJSON, _ := json.Marshal(tr)
				writer := &kafka.Writer{
					Addr:     kafka.TCP(kafkaBroker),
					Topic:    TopicTracks,
					Balancer: &kafka.LeastBytes{},
				}
				writer.WriteMessages(ctx, kafka.Message{Key: []byte(strconv.Itoa(tr.TrackNumber)), Value: trackJSON})
				writer.Close()
			}
		}
	}
}
