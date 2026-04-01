// env-monitor — VIMI Environmental Monitoring Service
// Phase 2: Mission Processing
// Monitors environmental factors affecting sensor performance:
// cloud cover, atmospheric conditions, solar events, EM interference
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
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
	TopicEnvEvents = "vimi.env.events"
	TopicAlerts    = "vimi.alerts"
)

// Environmental phenomenon types
type EnvPhenomenon int

const (
	ENV_CLEAR EnvPhenomenon = iota
	ENV_CLOUD
	ENV_RAIN
	ENV_STORM
	ENV_FOG
	ENV_DUST
	ENV_SOLAR_FLARE
	ENV_EM_INTERFERENCE
	ENV_NORTHERN_LIGHTS
)

func (e EnvPhenomenon) String() string {
	names := []string{"Clear", "Cloud", "Rain", "Storm", "Fog", "Dust", "Solar Flare", "EM Interference", "Northern Lights"}
	if e < 0 || int(e) >= len(names) {
		return "Unknown"
	}
	return names[e]
}

// SensorImpact describes how an environmental phenomenon affects sensor types
type SensorImpact struct {
	SBIRSImpact  float64 // 0=none, 1=total degradation
	AWACSImpact  float64
	PATRIOTImpact float64
	THAADImpact  float64
	RadiusKm     float64 // Geographic radius of effect
	AltitudeKm   float64 // Altitude (for space-based effects)
}

// EnvEvent from DoD meteorological/oceanographic (METOC) standards
type EnvEvent struct {
	EventID       uint32           `json:"event_id"`
	Type          EnvPhenomenon    `json:"type"`
	Severity      int              `json:"severity"` // 1-5
	CenterLat     float64           `json:"center_lat"`
	CenterLon     float64           `json:"center_lon"`
	RadiusKm      float64           `json:"radius_km"`
	AltitudeMinKm float64           `json:"altitude_min_km"`
	AltitudeMaxKm float64           `json:"altitude_max_km"`
	Probability   float64           `json:"probability"` // confidence 0-1
	Duration      time.Duration     `json:"duration"`
	StartTime     time.Time         `json:"start_time"`
	EndTime       time.Time         `json:"end_time"`
	SensorImpact  SensorImpact      `json:"sensor_impact"`
	DataSource    string            `json:"data_source"`
}

// SensorCoverage grid cell
type GridCell struct {
	Lat, Lon      float64
	CloudCover    float64 // 0-1
	Precipitation float64 // mm/h
	VisibilityKm  float64
	WindSpeedMs   float64
	SolarIndex    float64 // 0-1 (flare activity)
	EMNoiseDb     float64 // dB above baseline
}

// envMonitor state
type envMonitor struct {
	activeEvents []*EnvEvent
	cells        [][]GridCell // 1-degree grid
	gridRes      int          // degrees per cell
	nextEvent    uint32
}

func newEnvMonitor() *envMonitor {
	// 180x360 grid at 1-degree resolution
	cells := make([][]GridCell, 180)
	for i := range cells {
		cells[i] = make([]GridCell, 360)
		for j := range cells[i] {
			lat := float64(i) - 90
			lon := float64(j) - 180
			cells[i][j] = GridCell{
				Lat:          lat,
				Lon:          lon,
				CloudCover:   rand.Float64() * 0.3, // baseline 0-30%
				Precipitation: 0,
				VisibilityKm:  50,
				WindSpeedMs:  rand.Float64() * 15,
				SolarIndex:   0.1,
				EMNoiseDb:    -100, // baseline
			}
		}
	}
	return &envMonitor{
		activeEvents: []*EnvEvent{},
		cells:        cells,
		gridRes:      1,
		nextEvent:    1,
	}
}

// Generate random environmental event
func (em *envMonitor) generateEvent() *EnvEvent {
	eventType := EnvPhenomenon(rand.Intn(9))
	severity := rand.Intn(5) + 1

	lat := float64(rand.Intn(160) - 80)
	lon := float64(rand.Intn(360) - 180)

	var impact SensorImpact
	var radiusKm, altMin, altMax float64
	var duration time.Duration

	switch eventType {
	case ENV_CLOUD:
		radiusKm = float64(rand.Intn(500) + 100)
		altMin, altMax = 0, 10
		impact = SensorImpact{SBIRSImpact: 0.3, AWACSImpact: 0.4, PATRIOTImpact: 0.2, THAADImpact: 0.1, RadiusKm: radiusKm}
		duration = time.Duration(rand.Intn(12)+2) * time.Hour

	case ENV_RAIN:
		radiusKm = float64(rand.Intn(300) + 50)
		altMin, altMax = 0, 8
		impact = SensorImpact{SBIRSImpact: 0.4, AWACSImpact: 0.5, PATRIOTImpact: 0.6, THAADImpact: 0.3, RadiusKm: radiusKm}
		duration = time.Duration(rand.Intn(6)+1) * time.Hour

	case ENV_STORM:
		radiusKm = float64(rand.Intn(200) + 50)
		altMin, altMax = 0, 15
		impact = SensorImpact{SBIRSImpact: 0.6, AWACSImpact: 0.8, PATRIOTImpact: 0.7, THAADImpact: 0.5, RadiusKm: radiusKm}
		duration = time.Duration(rand.Intn(8)+2) * time.Hour

	case ENV_FOG:
		radiusKm = float64(rand.Intn(400) + 100)
		altMin, altMax = 0, 0.5
		impact = SensorImpact{SBIRSImpact: 0.1, AWACSImpact: 0.6, PATRIOTImpact: 0.8, THAADImpact: 0.2, RadiusKm: radiusKm}
		duration = time.Duration(rand.Intn(10)+2) * time.Hour

	case ENV_DUST:
		radiusKm = float64(rand.Intn(600) + 200)
		altMin, altMax = 0, 5
		impact = SensorImpact{SBIRSImpact: 0.3, AWACSImpact: 0.5, PATRIOTImpact: 0.4, THAADImpact: 0.2, RadiusKm: radiusKm}
		duration = time.Duration(rand.Intn(48)+6) * time.Hour

	case ENV_SOLAR_FLARE:
		radiusKm = 0 // global effect
		altMin, altMax = 0, 0 // affects space-based sensors
		impact = SensorImpact{SBIRSImpact: 0.9, AWACSImpact: 0.7, PATRIOTImpact: 0.3, THAADImpact: 0.8, RadiusKm: 0, AltitudeKm: 36000}
		duration = time.Duration(rand.Intn(72)+6) * time.Hour

	case ENV_EM_INTERFERENCE:
		radiusKm = float64(rand.Intn(150) + 50)
		altMin, altMax = 0, 10
		impact = SensorImpact{SBIRSImpact: 0.2, AWACSImpact: 0.8, PATRIOTImpact: 0.9, THAADImpact: 0.7, RadiusKm: radiusKm}
		duration = time.Duration(rand.Intn(2)+1) * time.Hour

	case ENV_NORTHERN_LIGHTS:
		radiusKm = float64(rand.Intn(1000) + 500)
		lat = 65 + rand.Float64()*20 // aurora belt
		altMin, altMax = 100, 300
		impact = SensorImpact{SBIRSImpact: 0.5, AWACSImpact: 0.3, PATRIOTImpact: 0.1, THAADImpact: 0.4, RadiusKm: radiusKm}
		duration = time.Duration(rand.Intn(12)+2) * time.Hour

	default:
		radiusKm = 0
		altMin, altMax = 0, 0
		duration = 0
	}

	e := &EnvEvent{
		EventID:       em.nextEvent,
		Type:          eventType,
		Severity:      severity,
		CenterLat:     lat,
		CenterLon:     lon,
		RadiusKm:      radiusKm,
		AltitudeMinKm: altMin,
		AltitudeMaxKm: altMax,
		Probability:   0.7 + rand.Float64()*0.3,
		Duration:      duration,
		StartTime:     time.Now(),
		EndTime:       time.Now().Add(duration),
		SensorImpact:  impact,
		DataSource:    "SIM",
	}
	em.nextEvent++

	return e
}

func (em *envMonitor) applyEvent(e *EnvEvent) {
	// Update grid cells within event radius
	for i := range em.cells {
		for j := range em.cells[i] {
			cell := &em.cells[i][j]

			// Calculate distance from event center
			dlat := cell.Lat - e.CenterLat
			dlon := cell.Lon - e.CenterLon
			distKm := math.Sqrt(dlat*dlat+dlon*dlon) * 111.0 // rough km

			if distKm > e.RadiusKm && e.RadiusKm > 0 {
				continue
			}

			// Apply effects based on event type
			factor := 1.0
			if e.RadiusKm > 0 {
				factor = 1.0 - distKm/e.RadiusKm
			}

			switch e.Type {
			case ENV_CLOUD:
				cell.CloudCover = math.Min(1, cell.CloudCover+0.4*factor*float64(e.Severity)/5)
			case ENV_RAIN:
				cell.Precipitation = math.Min(50, cell.Precipitation+10*factor*float64(e.Severity))
			case ENV_STORM:
				cell.Precipitation = math.Min(100, cell.Precipitation+20*factor*float64(e.Severity))
				cell.VisibilityKm = math.Max(0.1, cell.VisibilityKm-5*factor)
				cell.EMNoiseDb = math.Min(0, cell.EMNoiseDb+20*factor)
			case ENV_FOG:
				cell.VisibilityKm = math.Max(0.05, cell.VisibilityKm-10*factor)
			case ENV_DUST:
				cell.VisibilityKm = math.Max(0.1, cell.VisibilityKm-8*factor)
				cell.EMNoiseDb = math.Min(0, cell.EMNoiseDb+10*factor)
			case ENV_SOLAR_FLARE:
				cell.SolarIndex = math.Min(1, cell.SolarIndex+0.8*factor)
			case ENV_EM_INTERFERENCE:
				cell.EMNoiseDb = math.Min(0, cell.EMNoiseDb+30*factor)
			}
		}
	}
}

func (em *envMonitor) cleanup() {
	now := time.Now()
	var remaining []*EnvEvent
	for _, e := range em.activeEvents {
		if now.Before(e.EndTime) {
			remaining = append(remaining, e)
		}
	}
	em.activeEvents = remaining
}

// Get sensor performance rating for a location
func (em *envMonitor) getSensorRating(lat, lon float64, sensorType string) float64 {
	// Find grid cell
	i := int(lat + 90)
	j := int(lon + 180)
	if i < 0 || i >= 180 || j < 0 || j >= 360 {
		return 0.5 // unknown area
	}
	cell := em.cells[i][j]

	// Calculate base rating (1.0 = perfect)
	rating := 1.0

	// Cloud cover impact
	cloudFactor := 1.0 - cell.CloudCover*0.3
	rating *= cloudFactor

	// Precipitation impact
	if cell.Precipitation > 20 {
		rating *= 0.7
	} else if cell.Precipitation > 5 {
		rating *= 0.85
	}

	// Visibility impact
	if cell.VisibilityKm < 5 {
		rating *= 0.6
	} else if cell.VisibilityKm < 20 {
		rating *= 0.8
	}

	// Solar flare impact (space-based sensors)
	if cell.SolarIndex > 0.5 {
		if sensorType == "SBIRS" || sensorType == "NGOPIR" {
			rating *= 1.0 - cell.SolarIndex*0.7
		}
	}

	// EM noise impact
	if cell.EMNoiseDb > -80 {
		noiseFactor := (cell.EMNoiseDb + 100) / 20 // 0-1 scale
		if sensorType == "AWACS" || sensorType == "PATRIOT" || sensorType == "THAAD" {
			rating *= 1.0 - noiseFactor*0.5
		}
	}

	return math.Max(0, math.Min(1, rating))
}

// EnvReport for a location query
type EnvReport struct {
	Location     Coordinate      `json:"location"`
	Timestamp    time.Time       `json:"timestamp"`
	Conditions   string          `json:"conditions"`
	SBRating     float64         `json:"sbirs_rating"`
	AWACSRating  float64         `json:"awacs_rating"`
	PATRIOTRating float64        `json:"patriot_rating"`
	THAADRating  float64         `json:"thaad_rating"`
	CloudCover   float64         `json:"cloud_cover"`
	VisibilityKm float64         `json:"visibility_km"`
	Precipitation float64        `json:"precipitation_mm_h"`
	ActiveEvents int              `json:"active_events"`
}

type Coordinate struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

func (em *envMonitor) generateReport(lat, lon float64) *EnvReport {
	r := &EnvReport{
		Location:     Coordinate{Lat: lat, Lon: lon},
		Timestamp:   time.Now(),
		SBRating:    em.getSensorRating(lat, lon, "SBIRS"),
		AWACSRating: em.getSensorRating(lat, lon, "AWACS"),
		PATRIOTRating: em.getSensorRating(lat, lon, "PATRIOT"),
		THAADRating: em.getSensorRating(lat, lon, "THAAD"),
		ActiveEvents: len(em.activeEvents),
	}

	i := int(lat + 90)
	j := int(lon + 180)
	if i >= 0 && i < 180 && j >= 0 && j < 360 {
		cell := em.cells[i][j]
		r.CloudCover = cell.CloudCover
		r.VisibilityKm = cell.VisibilityKm
		r.Precipitation = cell.Precipitation

		if cell.CloudCover < 0.1 && cell.Precipitation < 1 {
			r.Conditions = "Clear"
		} else if cell.Precipitation > 20 {
			r.Conditions = "Storm"
		} else if cell.CloudCover > 0.6 {
			r.Conditions = "Overcast"
		} else if cell.Precipitation > 5 {
			r.Conditions = "Rain"
		} else {
			r.Conditions = "Partly Cloudy"
		}
	}

	return r
}

var (
	em           *envMonitor
	kafkaWriter  *kafka.Writer
	kafkaBroker  = getEnv("KAFKA_BROKERS", "kafka:9092")
	port         = getEnv("PORT", "8085")
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func run(ctx context.Context) {
	eventTick := time.NewTicker(5 * time.Minute)
	defer eventTick.Stop()
	cleanupTick := time.NewTicker(30 * time.Second)
	defer cleanupTick.Stop()
	reportTick := time.NewTicker(30 * time.Second)
	defer reportTick.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-eventTick.C:
			// Generate new event (Poisson arrival ~1 per 5 min)
			if rand.Float64() < 0.7 {
				e := em.generateEvent()
				em.activeEvents = append(em.activeEvents, e)
				em.applyEvent(e)

				data, _ := json.Marshal(e)
				ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
				kafkaWriter.WriteMessages(ctx2, kafka.Message{
					Key:   []byte(fmt.Sprintf("env-%d", e.EventID)),
					Value: data,
					Topic: TopicEnvEvents,
				})
				cancel()

				log.Printf("ENV: %s severity=%d lat=%.1f lon=%.1f radius=%dkm duration=%s",
					e.Type.String(), e.Severity, e.CenterLat, e.CenterLon,
					int(e.RadiusKm), e.Duration.Round(time.Minute))
			}
		case <-cleanupTick.C:
			em.cleanup()
		case <-reportTick.C:
			// Publish periodic global report (sparse grid)
			for lat := -60.0; lat <= 60.0; lat += 20 {
				for lon := -120.0; lon <= 120.0; lon += 20 {
					rep := em.generateReport(lat, lon)
					if rep.SBRating < 0.7 || rep.AWACSRating < 0.7 {
						data, _ := json.Marshal(rep)
						kafkaWriter.WriteMessages(ctx, kafka.Message{
							Key:   []byte(fmt.Sprintf("envreport-%.0f-%.0f", lat, lon)),
							Value: data,
							Topic: TopicEnvEvents,
						})
					}
				}
			}
		}
	}
}

type HealthResponse struct {
	Service      string    `json:"service"`
	Version      string    `json:"version"`
	Timestamp    time.Time `json:"timestamp"`
	Status       string    `json:"status"`
	ActiveEvents int       `json:"active_events"`
	GridRes      string    `json:"grid_resolution"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Service:      "env-monitor",
		Version:      "0.1.0",
		Timestamp:    time.Now().UTC(),
		Status:       "healthy",
		ActiveEvents: len(em.activeEvents),
		GridRes:      "1 degree",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	em = newEnvMonitor()

	kafkaWriter = &kafka.Writer{
		Addr:     kafka.TCP(kafkaBroker),
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
	http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		sort.Slice(em.activeEvents, func(i, j int) bool {
			return em.activeEvents[i].StartTime.After(em.activeEvents[j].StartTime)
		})
		json.NewEncoder(w).Encode(em.activeEvents)
	})
	http.HandleFunc("/report", func(w http.ResponseWriter, r *http.Request) {
		lat := -33.0 // default: mid-Atlantic
		lon := -70.0
		if l := r.URL.Query().Get("lat"); l != "" {
			fmt.Sscanf(l, "%f", &lat)
		}
		if l := r.URL.Query().Get("lon"); l != "" {
			fmt.Sscanf(l, "%f", &lon)
		}
		rep := em.generateReport(lat, lon)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rep)
	})

	log.Printf("env-monitor starting")
	log.Printf("Kafka broker: %s", kafkaBroker)
	go run(ctx)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("HTTP server on %ss", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
