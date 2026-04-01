// sensor-fusion — VIMI Sensor Fusion Service
// Phase 1: Core Infrastructure
// Subscribes to tracks from missile-warning-engine + external sources,
// performs JPDA (Joint Probabilistic Data Association) track fusion,
// and publishes composite tracks
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
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
)

const (
	TopicTracks      = "vimi.tracks"
	TopicFusionTracks = "vimi.fusion.tracks"
	TopicEvents      = "vimi.events"
	FusionWindow     = 60 * time.Second // Time window for JPDA association
)

// SensorSource from DoD sensor taxonomy
type SensorSource int

const (
	SENSOR_UNKNOWN    SensorSource = 0
	SENSOR_SBIRS_HIGH SensorSource = 1
	SENSOR_SBIRS_LOW  SensorSource = 2
	SENSOR_NGOPIR     SensorSource = 3
	SENSOR_AWACS      SensorSource = 4
	SENSOR_PATRIOT    SensorSource = 5
	SENSOR_THAAD      SensorSource = 6
	SENSOR_AEGIS      SensorSource = 7
	SENSOR_GMD        SensorSource = 8
)

func (s SensorSource) String() string {
	names := []string{"Unknown", "SBIRS-High", "SBIRS-Low", "NG-OPIR", "AWACS", "Patriot", "THAAD", "Aegis", "GMD"}
	if s < 0 || int(s) >= len(names) {
		return "Unknown"
	}
	return names[s]
}

// ExternalTrack from various sensor sources
type ExternalTrack struct {
	TrackNumber   uint32      `json:"track_number"`
	Source       SensorSource `json:"source"`
	Lat          float64     `json:"lat"`
	Lon          float64     `json:"lon"`
	Alt          float64     `json:"alt"`          // km
	Velocity     float64     `json:"velocity"`     // m/s
	Heading      float64     `json:"heading"`      // degrees
	ThreatLevel  int         `json:"threat_level"` // 0-5
	Confidence   float64     `json:"confidence"`   // 0-1
	Timestamp    time.Time   `json:"timestamp"`
}

// CompositeTrack is the fused result from multiple sensors
type CompositeTrack struct {
	TrackNumber    uint32          `json:"track_number"`
	FusedLat      float64         `json:"fused_lat"`
	FusedLon      float64         `json:"fused_lon"`
	FusedAlt      float64         `json:"fused_alt"`
	FusedVelocity float64         `json:"fused_velocity"`
	FusedHeading  float64         `json:"fused_heading"`
	ThreatLevel   int             `json:"threat_level"`
	Confidence    float64         `json:"confidence"`
	UpdateCount   int             `json:"update_count"`
	LastUpdate    time.Time       `json:"last_update"`
	Sources       []SensorSource  `json:"sources"`
	
	// Kalman state
	kalmanLat, kalmanLon, kalmanAlt float64
	kalmanVLat, kalmanVLon          float64
	kalmanVar                        float64
}

// JPDA association matrix entry
type Association struct {
	ExternalTrack *ExternalTrack
	FusedTrack   *CompositeTrack
	Probability  float64 // JPDA association probability
	Gate         bool    // passed gating test
}

type fusedTrackManager struct {
	fusedTracks map[uint32]*CompositeTrack
	nextTrack   uint32
	sources     map[uint32]SensorSource
}

func newFusedTrackManager() *fusedTrackManager {
	return &fusedTrackManager{
		fusedTracks: make(map[uint32]*CompositeTrack),
		nextTrack:   1,
		sources:     make(map[uint32]SensorSource),
	}
}

// Kalman filter update for fused position
func (ft *CompositeTrack) kalmanUpdate(lat, lon, alt float64, varmeas float64, weight float64) {
	// 1D Kalman filter per coordinate
	Q := 0.001  // Process noise (meters²/s)
	R := varmeas // Measurement noise (from sensor SNR)
	
	ft.kalmanVar += Q
	K := ft.kalmanVar / (ft.kalmanVar + R) // Kalman gain
	
	// Weighted update (weight from JPDA probability)
	ft.kalmanLat += K * weight * (lat - ft.kalmanLat)
	ft.kalmanLon += K * weight * (lon - ft.kalmanLon)
	ft.kalmanAlt += K * weight * (alt - ft.kalmanAlt)
	ft.kalmanVLat += K * weight * 0.001 // Approximate velocity update
	ft.kalmanVLon += K * weight * 0.001
	
	ft.kalmanVar *= (1 - K) // Variance update
	
	ft.FusedLat = ft.kalmanLat
	ft.FusedLon = ft.kalmanLon
	ft.FusedAlt = ft.kalmanAlt
}

// Compute Mahalanobis distance for JPDA gating
func mahalanobisDistance(lat1, lon1, lat2, lon2, var1, var2 float64) float64 {
	dlat := (lat1 - lat2) * 111000   // degrees to meters (rough)
	dlon := (lon1 - lon2) * 111000 * math.Cos(lat1*math.Pi/180)
	
	variance := var1 + var2 + 0.01 // Added noise floor
	return math.Sqrt(dlat*dlat/variance + dlon*dlon/variance)
}

// JPDA: compute association probabilities
func (fm *fusedTrackManager) jpdaAssociate(et *ExternalTrack) []*Association {
	var associations []*Association
	
	// Measurement noise from sensor type (degrees)
	varSensor := 0.01
	switch et.Source {
	case SENSOR_SBIRS_HIGH:
		varSensor = 0.001
	case SENSOR_NGOPIR:
		varSensor = 0.002
	case SENSOR_AWACS:
		varSensor = 0.005
	case SENSOR_PATRIOT:
		varSensor = 0.003
	}
	
	// Check association with all existing fused tracks
	associationThreshold := 3.0 // Mahalanobis sigma equivalent
	
	for _, ft := range fm.fusedTracks {
		dist := mahalanobisDistance(et.Lat, et.Lon, ft.FusedLat, ft.FusedLon, varSensor, ft.kalmanVar)
		passedGate := dist < associationThreshold
		
		// Compute association probability (simplified JPDA)
		var prob float64
		if passedGate {
			// Probability decreases with distance
			prob = math.Exp(-dist*dist/2) / (1 + math.Exp(-dist*dist/2))
		} else {
			prob = 0
		}
		
		associations = append(associations, &Association{
			ExternalTrack: et,
			FusedTrack:    ft,
			Probability:   prob,
			Gate:         passedGate,
		})
	}
	
	return associations
}

// Fuse a new measurement into the track
func (fm *fusedTrackManager) fuse(et *ExternalTrack) {
	// JPDA association
	associations := fm.jpdaAssociate(et)
	
	// Find best association
	var bestAssoc *Association
	var bestProb float64 = -1
	
	for _, assoc := range associations {
		if assoc.Probability > bestProb {
			bestProb = assoc.Probability
			bestAssoc = assoc
		}
	}
	
	if bestAssoc != nil && bestAssoc.Probability > 0.3 && bestAssoc.Gate {
		// Update existing fused track with JPDA-weighted measurement
		ft := bestAssoc.FusedTrack
		
		// Weighted Kalman update
		measurementVariance := 0.01 / (et.Confidence + 0.01) // Better confidence = lower variance
		ft.kalmanUpdate(et.Lat, et.Lon, et.Alt, measurementVariance, bestAssoc.Probability)
		
		// Update other fields
		ft.FusedVelocity = (ft.FusedVelocity*float64(ft.UpdateCount) + et.Velocity) / float64(ft.UpdateCount+1)
		ft.FusedHeading = et.Heading // Use latest heading
		if et.ThreatLevel > ft.ThreatLevel {
			ft.ThreatLevel = et.ThreatLevel
		}
		if et.Confidence > ft.Confidence {
			ft.Confidence = (ft.Confidence + et.Confidence) / 2
		} else {
			ft.Confidence = ft.Confidence * 0.95 + et.Confidence * 0.05
		}
		ft.UpdateCount++
		ft.LastUpdate = time.Now()
		
		// Add source if new
		hasSource := false
		for _, s := range ft.Sources {
			if s == et.Source {
				hasSource = true
				break
			}
		}
		if !hasSource {
			ft.Sources = append(ft.Sources, et.Source)
		}
		
		log.Printf("FUSION: track %d updated by %s (prob=%.2f, sources=%d, conf=%.1f%%)",
			ft.TrackNumber, et.Source.String(), bestProb, len(ft.Sources), ft.Confidence*100)
	} else {
		// Create new fused track
		ft := &CompositeTrack{
			TrackNumber:   fm.nextTrack,
			FusedLat:     et.Lat,
			FusedLon:     et.Lon,
			FusedAlt:     et.Alt,
			FusedVelocity: et.Velocity,
			FusedHeading: et.Heading,
			ThreatLevel:  et.ThreatLevel,
			Confidence:   et.Confidence,
			UpdateCount:  1,
			LastUpdate:   time.Now(),
			Sources:      []SensorSource{et.Source},
			kalmanLat:   et.Lat,
			kalmanLon:   et.Lon,
			kalmanAlt:   et.Alt,
			kalmanVar:   0.01,
		}
		fm.nextTrack++
		fm.fusedTracks[ft.TrackNumber] = ft
		fm.sources[et.TrackNumber] = et.Source
		
		log.Printf("FUSION: new track %d from %s (lat=%.4f lon=%.4f, conf=%.0f%%)",
			ft.TrackNumber, et.Source.String(), et.Lat, et.Lon, et.Confidence*100)
	}
}

func (fm *fusedTrackManager) cleanup() {
	now := time.Now()
	for num, ft := range fm.fusedTracks {
		if now.Sub(ft.LastUpdate) > FusionWindow*2 {
			log.Printf("FUSION: track %d expired (sources: %v)", num, ft.Sources)
			delete(fm.fusedTracks, num)
		}
	}
}

var (
	fm             *fusedTrackManager
	kafkaWriter   *kafka.Writer
	kafkaBroker   = getEnv("KAFKA_BROKERS", "kafka:9092")
	port          = getEnv("PORT", "8082")
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func run(ctx context.Context) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{kafkaBroker},
		Topic:     TopicTracks,
		GroupID:   "sensor-fusion",
		MinBytes:  10e3,
		MaxBytes:  10e6,
		StartOffset: kafka.LastOffset,
	})
	defer reader.Close()
	
	cleanup := time.NewTicker(30 * time.Second)
	defer cleanup.Stop()
	
	publishTick := time.NewTicker(5 * time.Second)
	defer publishTick.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-cleanup.C:
			fm.cleanup()
		case <-publishTick.C:
			// Publish all fused tracks
			for _, ft := range fm.fusedTracks {
				data, _ := json.Marshal(ft)
				ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
				kafkaWriter.WriteMessages(ctx2, kafka.Message{
					Key:   []byte(fmt.Sprintf("fusion-%d", ft.TrackNumber)),
					Value: data,
				})
				cancel()
			}
		default:
			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}
			
			var et ExternalTrack
			if err := json.Unmarshal(msg.Value, &et); err != nil {
				continue
			}
			
			// Override source from headers if present
			for _, h := range msg.Headers {
				if h.Key == "source" {
					switch string(h.Value) {
					case "SBIRS-High":
						et.Source = SENSOR_SBIRS_HIGH
					case "SBIRS-Low":
						et.Source = SENSOR_SBIRS_LOW
					case "NG-OPIR":
						et.Source = SENSOR_NGOPIR
					case "AWACS":
						et.Source = SENSOR_AWACS
					case "Patriot":
						et.Source = SENSOR_PATRIOT
					case "THAAD":
						et.Source = SENSOR_THAAD
					}
				}
			}
			
			fm.fuse(&et)
		}
	}
}

type HealthResponse struct {
	Service     string    `json:"service"`
	Version     string    `json:"version"`
	Timestamp   time.Time `json:"timestamp"`
	Status      string    `json:"status"`
	FusedTracks int       `json:"fused_tracks"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Service:     "sensor-fusion",
		Version:     "0.1.0",
		Timestamp:   time.Now().UTC(),
		Status:      "healthy",
		FusedTracks: len(fm.fusedTracks),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	
	fm = newFusedTrackManager()
	
	kafkaWriter = &kafka.Writer{
		Addr:     kafka.TCP(kafkaBroker),
		Topic:    TopicFusionTracks,
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
	
	http.Handle("/metrics", promhttp.Handler())
http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/tracks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		tracks := make([]*CompositeTrack, 0, len(fm.fusedTracks))
		for _, tr := range fm.fusedTracks {
			tracks = append(tracks, tr)
		}
		sort.Slice(tracks, func(i, j int) bool {
			return tracks[i].LastUpdate.After(tracks[j].LastUpdate)
		})
		json.NewEncoder(w).Encode(tracks)
	})
	
	log.Printf("sensor-fusion starting")
	log.Printf("Kafka broker: %s", kafkaBroker)
	log.Printf("Input: %s, Output: %s", TopicTracks, TopicFusionTracks)
	go run(ctx)
	
	addr := fmt.Sprintf(":%s", port)
	log.Printf("HTTP server on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
