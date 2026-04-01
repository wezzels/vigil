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
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
)

const (
	TopicAlerts   = "vimi.alerts"
	TopicC2Out    = "vimi.c2.alerts"
	TopicAlertLog = "vimi.alert-log"
)

// --- Prometheus Metrics ---
var (
	alertsReceived = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vimi_alerts_received_total",
			Help: "Total alerts received from missile-warning-engine",
		},
		[]string{"level", "threat_type"},
	)
	alertsIssued = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vimi_alerts_issued_total",
			Help: "Total C2 alerts issued by level",
		},
		[]string{"level", "precedence"},
	)
	alertsActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vimi_alerts_active",
			Help: "Currently active alerts by level",
		},
		[]string{"level"},
	)
	ncaApprovalRequired = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "vimi_nca_approvals_total",
			Help: "Alerts requiring NCA approval",
		},
	)
	jtidsMessages = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vimi_jtids_messages_total",
			Help: "JTIDS messages sent by net",
		},
		[]string{"net"},
	)
)

func init() {
	prometheus.MustRegister(alertsReceived, alertsIssued, alertsActive, ncaApprovalRequired, jtidsMessages)
}

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

func alertLevelName(a AlertLevel) string { return a.String() }

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

// alertState manages active and historical alerts
type alertState struct {
	mu     sync.Mutex
	alerts map[uint32]*Alert
	sent   map[uint32]time.Time
}

func newAlertState() *alertState {
	return &alertState{
		alerts: make(map[uint32]*Alert),
		sent:   make(map[uint32]time.Time),
	}
}

func (as *alertState) add(a *Alert) {
	as.mu.Lock()
	defer as.mu.Unlock()
	// Check if this is an escalation
	existing, ok := as.alerts[a.TrackNumber]
	if ok {
		if a.AlertLevel > existing.AlertLevel {
			// Escalation — log it
			log.Printf("ALERT: [%s  ] track=%d tti=%.0fs weapon=%s nca=%t",
				a.AlertLevel.String(), a.TrackNumber, a.TimeToImpact,
				a.ThreatType.String(), a.NCARequired)
		}
	} else {
		// New alert
		alertsReceived.WithLabelValues(a.AlertLevel.String(), a.ThreatType.String()).Inc()
		alertsActive.WithLabelValues(a.AlertLevel.String()).Inc()
		log.Printf("ALERT: [%s  ] track=%d tti=%.0fs weapon=%s nca=%t",
			a.AlertLevel.String(), a.TrackNumber, a.TimeToImpact,
			a.ThreatType.String(), a.NCARequired)
	}
	as.alerts[a.TrackNumber] = a
}

func (as *alertState) clearStale() int {
	as.mu.Lock()
	defer as.mu.Unlock()
	cleared := 0
	for num, a := range as.alerts {
		if time.Since(a.IssuedAt) > 10*time.Minute {
			alertsActive.WithLabelValues(a.AlertLevel.String()).Dec()
			delete(as.alerts, num)
			cleared++
		}
	}
	return cleared
}

var (
	kafkaBroker string
	port        string
	kafkaWriter *kafka.Writer
	as          *alertState
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	as.mu.Lock()
	defer as.mu.Unlock()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service":   "alert-dissemination",
		"status":    "healthy",
		"active":    len(as.alerts),
		"timestamp": time.Now().UTC(),
	})
}

func main() {
	flag.StringVar(&kafkaBroker, "kafka", "kafka:9092", "Kafka broker")
	flag.StringVar(&port, "port", "8084", "HTTP port")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	as = newAlertState()

	// Prometheus metrics endpoint
	http.Handle("/metrics", promhttp.Handler())

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
		as.mu.Lock()
		defer as.mu.Unlock()
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

func run(ctx context.Context) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{kafkaBroker},
		Topic:    TopicAlerts,
		GroupID:  "alert-dissemination",
		MinBytes: 1,
		MaxBytes: 1e6,
	})
	defer reader.Close()

	cleanup := time.NewTicker(60 * time.Second)
	defer cleanup.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-cleanup.C:
			n := as.clearStale()
			if n > 0 {
				log.Printf("Cleared %d stale alerts", n)
			}
		default:
			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}

			var alert Alert
			if err := json.Unmarshal(msg.Value, &alert); err != nil {
				log.Printf("ERROR parsing alert: %v", err)
				continue
			}

			as.add(&alert)

			// Apply doctrine and issue C2 message
			c2 := applyDoctrine(&alert)
			
			c2JSON, _ := json.Marshal(c2)
			if err := kafkaWriter.WriteMessages(ctx, kafka.Message{
				Key:   []byte(fmt.Sprintf("%d", alert.TrackNumber)),
				Value: c2JSON,
			}); err != nil {
				log.Printf("ERROR writing C2 message: %v", err)
			} else {
				alertsIssued.WithLabelValues(c2.AlertLevel.String(), c2.precedence).Inc()
				jtidsMessages.WithLabelValues(fmt.Sprintf("%d", c2.JTIDSNet)).Inc()
				if c2.NCAApproval {
					ncaApprovalRequired.Inc()
				}
			}

			// Also log to alert-log topic
			logJSON, _ := json.Marshal(map[string]interface{}{
				"type":      "alert",
				"alert":     alert,
				"c2":        c2,
				"timestamp": time.Now().UTC(),
			})
			kafkaWriter.WriteMessages(ctx, kafka.Message{
				Key:   []byte(fmt.Sprintf("%d-log", alert.TrackNumber)),
				Value: logJSON,
			})
		}
	}
}
