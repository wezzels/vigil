// alert-dissemination — VIMI Alert Dissemination Service
// Phase 2: Mission Processing
// Subscribes to alerts from missile-warning-engine,
// applies CONOPREP/IMMINENT doctrine, and disseminates to C2 systems
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	TopicAlerts    = "vimi.alerts"
	TopicC2Out     = "vimi.c2.alerts"
	TopicAlertLog  = "vimi.alert-log"
)

// AlertLevel DoD doctrine
type AlertLevel int

const (
	ALERT_UNKNOWN  AlertLevel = 0
	ALERT_CONOPREP AlertLevel = 1
	ALERT_IMMINENT AlertLevel = 2
	ALERT_INCOMING AlertLevel = 3
	ALERT_HOSTILE  AlertLevel = 4
)

func (a AlertLevel) String() string {
	names := []string{"UNKNOWN", "CONOPREP", "IMMINENT", "INCOMING", "HOSTILE"}
	if a < 0 || int(a) >= len(names) {
		return "UNKNOWN"
	}
	return names[a]
}

// ThreatType
type ThreatType int

const (
	THREAT_UNKNOWN ThreatType = 0
	THREAT_SRBM   ThreatType = 1
	THREAT_MRBM   ThreatType = 2
	THREAT_IRBM   ThreatType = 3
	THREAT_ICBM   ThreatType = 4
)

func (t ThreatType) String() string {
	names := []string{"Unknown", "SRBM", "MRBM", "IRBM", "ICBM"}
	if t < 0 || int(t) >= len(names) {
		return "Unknown"
	}
	return names[t]
}

// Incoming alert from missile-warning-engine
type Alert struct {
	AlertID      uint32    `json:"alert_id"`
	AlertLevel   AlertLevel `json:"alert_level"`
	ThreatType   ThreatType `json:"threat_type"`
	TrackNumber  uint32    `json:"track_number"`
	LaunchLat    float64   `json:"launch_lat"`
	LaunchLon    float64   `json:"launch_lon"`
	ImpactLat    float64   `json:"impact_lat"`
	ImpactLon    float64   `json:"impact_lon"`
	TimeToImpact float64   `json:"time_to_impact"` // seconds
	ImpactTime   time.Time `json:"impact_time"`
	NCARequired  bool      `json:"nca_required"`
	SourceSensor string    `json:"source_sensor"`
	Confidence   float64   `json:"confidence"`
	IssuedAt     time.Time `json:"issued_at"`
}

// C2Message is the disseminated alert to C2 systems
type C2Message struct {
	MessageID     string       `json:"message_id"`
	 precedence   string       `json:"precedence"` // ROUTINE/PRIORITY/IMMEDIATE/OFFICER
	AlertLevel    AlertLevel   `json:"alert_level"`
	ThreatType    ThreatType   `json:"threat_type"`
	JTIDSNet      uint8        `json:"jtids_net"` // JTIDS network (0=broadcast)
	IRIGBMFlag    bool         `json:"iribm_flag"` // true if IRBM or ICBM
	NCAApproval   bool         `json:"nca_approval_required"`
	TrackNumber   uint32       `json:"track_number"`
	LaunchPoint   Coordinate   `json:"launch_point"`
	ImpactPoint   Coordinate   `json:"impact_point"`
	TimeToImpact  float64      `json:"time_to_impact"` // seconds
	ThreatWeapon  string       `json:"threat_weapon_designation"`
	SensorSource  string       `json:"sensor_source"`
	PDL           string       `json:"pdl"` // Processing Delay Line (latency ms)
	IssuedAt      time.Time    `json:"issued_at"`
}

type Coordinate struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// AlertRule applies DoD CONOPREP doctrine
type AlertRule struct {
	Level        AlertLevel
	TTIThreshold float64 // Time-to-impact threshold (seconds)
	NCARequired  bool
	JTIDSNet     uint8
	IRIGBM       bool
	Precedence   string
	Weapon       string
}

var doctrine = []AlertRule{
	{ALERT_CONOPREP, 900, false, 0, false, "PRIORITY", "UNKNOWN"},
	{ALERT_IMMINENT, 600, false, 1, false, "IMMEDIATE", "SSM"},
	{ALERT_INCOMING, 300, true, 2, false, "IMMEDIATE", "SSM"},
	{ALERT_HOSTILE, 120, true, 3, true, "OFFICER", "SSM"},
}

func applyDoctrine(a *Alert) *C2Message {
	// Find matching doctrine rule
	var rule AlertRule
	for _, r := range doctrine {
		if a.AlertLevel == r.Level {
			rule = r
			break
		}
		if r.TTIThreshold > 0 && a.TimeToImpact < r.TTIThreshold {
			rule = r
		}
	}

	// Determine weapon designation from threat type
	weapon := map[ThreatType]string{
		THREAT_SRBM: "SS-21",
		THREAT_MRBM: "SS-26",
		THREAT_IRBM: "CSS-20",
		THREAT_ICBM: "CSS-4",
	}
	if rule.Weapon == "SSM" {
		if w, ok := weapon[a.ThreatType]; ok {
			rule.Weapon = w
		}
	}

	return &C2Message{
		MessageID:    fmt.Sprintf("VIMI-%d-%d", a.TrackNumber, time.Now().Unix()),
		precedence:   rule.Precedence,
		AlertLevel:   a.AlertLevel,
		ThreatType:   a.ThreatType,
		JTIDSNet:     rule.JTIDSNet,
		IRIGBMFlag:   rule.IRIGBM,
		NCAApproval:  rule.NCARequired || a.NCARequired,
		TrackNumber:  a.TrackNumber,
		LaunchPoint:  Coordinate{Lat: a.LaunchLat, Lon: a.LaunchLon},
		ImpactPoint:  Coordinate{Lat: a.ImpactLat, Lon: a.ImpactLon},
		TimeToImpact: a.TimeToImpact,
		ThreatWeapon: rule.Weapon,
		SensorSource: a.SourceSensor,
		PDL:         fmt.Sprintf("%d", int(time.Since(a.IssuedAt).Milliseconds())),
		IssuedAt:    time.Now(),
	}
}

// alertState tracks alert history for deduplication
type alertState struct {
	alerts    map[uint32]*Alert
	sent      map[uint32]time.Time
	nextAlert uint32
}

func newAlertState() *alertState {
	return &alertState{
		alerts: make(map[uint32]*Alert),
		sent:   make(map[uint32]time.Time),
		nextAlert: 1,
	}
}

func (s *alertState) process(a *Alert) *C2Message {
	// Check if we already sent a similar alert (debounce 30s)
	for _, prev := range s.sent {
		if time.Since(prev) < 30*time.Second && prev.Unix() == a.IssuedAt.Unix() {
			return nil // duplicate, skip
		}
	}

	a.AlertID = s.nextAlert
	s.nextAlert++
	s.alerts[a.AlertID] = a

	// Apply doctrine and create C2 message
	c2 := applyDoctrine(a)

	// Mark as sent
	s.sent[a.TrackNumber] = time.Now()

	return c2
}

var (
	as            *alertState
	kafkaWriter   *kafka.Writer
	kafkaReader   *kafka.Reader
	kafkaBroker   = getEnv("KAFKA_BROKERS", "kafka:9092")
	port          = getEnv("PORT", "8084")
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
		Topic:     TopicAlerts,
		GroupID:   "alert-dissemination",
		MinBytes:  10e3,
		MaxBytes:  10e6,
		StartOffset: kafka.LastOffset,
	})
	defer reader.Close()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}

			var a Alert
			if err := json.Unmarshal(msg.Value, &a); err != nil {
				continue
			}

			// Override types from headers
			for _, h := range msg.Headers {
				if h.Key == "alert_level" {
					switch string(h.Value) {
					case "CONOPREP":
						a.AlertLevel = ALERT_CONOPREP
					case "IMMINENT":
						a.AlertLevel = ALERT_IMMINENT
					case "INCOMING":
						a.AlertLevel = ALERT_INCOMING
					case "HOSTILE":
						a.AlertLevel = ALERT_HOSTILE
					}
				}
				if h.Key == "threat_type" {
					switch string(h.Value) {
					case "SRBM":
						a.ThreatType = THREAT_SRBM
					case "MRBM":
						a.ThreatType = THREAT_MRBM
					case "IRBM":
						a.ThreatType = THREAT_IRBM
					case "ICBM":
						a.ThreatType = THREAT_ICBM
					}
				}
			}

			c2 := as.process(&a)
			if c2 == nil {
				continue // duplicate
			}

			// Publish C2 message
			c2Data, _ := json.Marshal(c2)
			ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
			kafkaWriter.WriteMessages(ctx2, kafka.Message{
				Key:   []byte(fmt.Sprintf("c2-%d", c2.TrackNumber)),
				Value: c2Data,
				Headers: []kafka.Header{
					{Key: "alert_level", Value: []byte(c2.AlertLevel.String())},
					{Key: "precedence", Value: []byte(c2.precedence)},
					{Key: "nca_required", Value: []byte(fmt.Sprintf("%t", c2.NCAApproval))},
				},
			})
			cancel()

			// Also log to alert-log topic
			logData, _ := json.Marshal(map[string]interface{}{
				"alert":      a,
				"c2_message": c2,
				"log_time":   time.Now(),
			})
			kafkaWriter.WriteMessages(ctx, kafka.Message{
				Key:   []byte(fmt.Sprintf("log-%d", a.AlertID)),
				Value: logData,
				Topic: TopicAlertLog,
			})

			log.Printf("ALERT: [%-10s] %s track=%d tti=%.0fs weapon=%s nca=%t",
				c2.precedence, c2.AlertLevel.String(), c2.TrackNumber,
				c2.TimeToImpact, c2.ThreatWeapon, c2.NCAApproval)
		}
	}
}

type HealthResponse struct {
	Service   string    `json:"service"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"`
	Sent      int       `json:"sent_alerts"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Service:   "alert-dissemination",
		Version:   "0.1.0",
		Timestamp: time.Now().UTC(),
		Status:    "healthy",
		Sent:      len(as.sent),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	as = newAlertState()

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

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/alerts", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		alerts := make([]*Alert, 0, len(as.alerts))
		for _, a := range as.alerts {
			alerts = append(alerts, a)
		}
		sort.Slice(alerts, func(i, j int) bool {
			return alerts[i].IssuedAt.After(alerts[j].IssuedAt)
		})
		json.NewEncoder(w).Encode(alerts)
	})

	log.Printf("alert-dissemination starting")
	log.Printf("Kafka broker: %s", kafkaBroker)
	log.Printf("Input: %s, Output: %s", TopicAlerts, TopicC2Out)
	go run(ctx)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("HTTP server on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
