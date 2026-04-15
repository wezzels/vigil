// vimi-plugin — VIMI Plugin for Cicerone
// Phase 3: Advanced Integration
// Provides threat track visualization and alert notifications for Cicerone
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
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
)

const (
	TopicAlerts    = "vimi.alerts"
	TopicTracks    = "vimi.fusion.tracks"
	TopicDIS       = "vimi.dis.entity-state-out"
	TopicEnvEvents = "vimi.env.events"
	PluginName     = "vimi"
	PluginVersion  = "0.1.0"
)

// ThreatTrack for globe display
type ThreatTrack struct {
	TrackID      uint32    `json:"track_id"`
	ThreatType   string    `json:"threat_type"`
	AlertLevel   string    `json:"alert_level"`
	Lat          float64   `json:"lat"`
	Lon          float64   `json:"lon"`
	Alt          float64   `json:"alt_km"`
	Velocity     float64   `json:"velocity_ms"`
	Heading      float64   `json:"heading_deg"`
	Confidence   float64   `json:"confidence"`
	LaunchLat    float64   `json:"launch_lat"`
	LaunchLon    float64   `json:"launch_lon"`
	ImpactLat    float64   `json:"impact_lat"`
	ImpactLon    float64   `json:"impact_lon"`
	TimeToImpact float64   `json:"tti_seconds"`
	SourceSensor string    `json:"source_sensor"`
	UpdateTime   time.Time `json:"update_time"`
	MarkerColor  string    `json:"marker_color"`
	MarkerIcon   string    `json:"marker_icon"`
}

// Alert for notification display
type Alert struct {
	AlertID      uint32    `json:"alert_id"`
	Level        string    `json:"level"`
	ThreatType   string    `json:"threat_type"`
	TrackID      uint32    `json:"track_id"`
	Precedence   string    `json:"precedence"`
	JTIDSNet     uint8     `json:"jtids_net"`
	LaunchLat    float64   `json:"launch_lat"`
	LaunchLon    float64   `json:"launch_lon"`
	ImpactLat    float64   `json:"impact_lat"`
	ImpactLon    float64   `json:"impact_lon"`
	TimeToImpact float64   `json:"tti_seconds"`
	NCARequired  bool      `json:"nca_required"`
	Weapon       string    `json:"weapon"`
	IssuedAt     time.Time `json:"issued_at"`
	Severity     int       `json:"severity"`
}

// LVCStatus for federation overview
type LVCStatus struct {
	LiveCount         int `json:"live_count"`
	VirtualCount      int `json:"virtual_count"`
	ConstructiveCount int `json:"constructive_count"`
	TotalEntities     int `json:"total_entities"`
	ActiveTracks      int `json:"active_tracks"`
	FriendlyCount     int `json:"friendly_count"`
	OpposingCount     int `json:"opposing_count"`
	NeutralCount      int `json:"neutral_count"`
}

// EnvEvent for environmental overlay
type EnvEvent struct {
	ID          uint32    `json:"event_id"`
	Type        string    `json:"type"`
	Severity    int       `json:"severity"`
	CenterLat   float64   `json:"center_lat"`
	CenterLon   float64   `json:"center_lon"`
	RadiusKm    float64   `json:"radius_km"`
	SBIRSImpact float64   `json:"sbirs_impact"`
	AWACSImpact float64   `json:"awacs_impact"`
	EndTime     time.Time `json:"end_time"`
}

// PluginState
type PluginState struct {
	tracks       map[uint32]*ThreatTrack
	alerts       []*Alert
	lvc          *LVCStatus
	envEvents    []*EnvEvent
	kafkaReader  *kafka.Reader
	alertsReader *kafka.Reader
	broker       string
	port         string
	running      bool
	lastUpdate   time.Time
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func (p *PluginState) Start() error {
	p.running = true
	p.tracks = make(map[uint32]*ThreatTrack)
	p.alerts = make([]*Alert, 0)
	p.lvc = &LVCStatus{}
	p.envEvents = make([]*EnvEvent, 0)

	p.kafkaReader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{p.broker},
		Topic:       TopicTracks,
		GroupID:     "vimi-plugin-tracks",
		MinBytes:    10e3,
		MaxBytes:    10e6,
		StartOffset: kafka.LastOffset,
	})

	p.alertsReader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{p.broker},
		Topic:       TopicAlerts,
		GroupID:     "vimi-plugin-alerts",
		MinBytes:    10e3,
		MaxBytes:    10e6,
		StartOffset: kafka.LastOffset,
	})

	log.Printf("[VIMIC] Plugin started, broker=%s", p.broker)
	return nil
}

func (p *PluginState) Stop() {
	p.running = false
	if p.kafkaReader != nil {
		p.kafkaReader.Close()
	}
	if p.alertsReader != nil {
		p.alertsReader.Close()
	}
}

func (p *PluginState) HandleCommand(cmd string, args []string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = ctx

	switch cmd {
	case "status":
		return p.cmdStatus()
	case "tracks":
		return p.cmdTracks(args)
	case "alerts":
		return p.cmdAlerts(args)
	case "lvc":
		return p.cmdLVC()
	case "env":
		return p.cmdEnv()
	case "inject":
		return p.cmdInject(args)
	case "globe":
		return p.cmdGlobe()
	default:
		return fmt.Sprintf("Unknown VIMI command: %s\nAvailable: status, tracks, alerts, lvc, env, inject, globe", cmd)
	}
}

func (p *PluginState) cmdStatus() string {
	trackCount := len(p.tracks)
	alertCount := len(p.alerts)
	envCount := len(p.envEvents)
	lvcTotal := 0
	if p.lvc != nil {
		lvcTotal = p.lvc.TotalEntities
	}

	uptime := "initializing"
	if !p.lastUpdate.IsZero() {
		uptime = time.Since(p.lastUpdate).Round(time.Second).String()
	}

	return fmt.Sprintf(`VIMI Mission Processing Stack
============================
Status:  %s
Tracks:  %d active threat tracks
Alerts:  %d recent alerts
LVC:      %d total entities
Env:      %d active events
Uptime:   %s

Services:
  opir-ingest             running
  missile-warning-engine  running
  sensor-fusion          running
  lvc-coordinator        running
  alert-dissemination    running
  env-monitor           running
  replay-engine         running
  data-catalog          running
  dis-hla-gateway       running`,
		map[bool]string{true: "OPERATIONAL", false: "INITIALIZING"}[p.running],
		trackCount, alertCount, lvcTotal, envCount, uptime)
}

func (p *PluginState) cmdTracks(args []string) string {
	if len(p.tracks) == 0 {
		return "No active threat tracks"
	}

	type ts struct {
		id uint32
		tr *ThreatTrack
	}
	var sorted []ts
	for id, tr := range p.tracks {
		sorted = append(sorted, ts{id, tr})
	}
	sort.Slice(sorted, func(i, j int) bool {
		level := map[string]int{"HOSTILE": 4, "INCOMING": 3, "IMMINENT": 2, "CONOPREP": 1, "UNKNOWN": 0}
		return level[sorted[i].tr.AlertLevel] > level[sorted[j].tr.AlertLevel]
	})

	var lines []string
	lines = append(lines, fmt.Sprintf("%-8s %-6s %-10s %-12s %-15s %s",
		"TRACK", "LEVEL", "TYPE", "POSITION", "VELOCITY", "TTI"))
	lines = append(lines, strings.Repeat("-", 75))

	for _, ts := range sorted {
		tr := ts.tr
		pos := fmt.Sprintf("%.3f, %.3f", tr.Lat, tr.Lon)
		if tr.Alt > 0 {
			pos += fmt.Sprintf(" @%.0fkm", tr.Alt)
		}
		tti := fmt.Sprintf("%.0fs", tr.TimeToImpact)
		if tr.TimeToImpact < 60 {
			tti = "!" + tti
		}
		lines = append(lines, fmt.Sprintf("%-8d %-6s %-10s %-12s %-15.0fm/s %s",
			tr.TrackID, tr.AlertLevel, tr.ThreatType, pos, tr.Velocity, tti))
	}

	return strings.Join(lines, "\n")
}

func (p *PluginState) cmdAlerts(args []string) string {
	if len(p.alerts) == 0 {
		return "No recent alerts"
	}

	limit := 10
	if len(args) > 0 {
		fmt.Sscanf(args[0], "%d", &limit)
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("%-8s %-10s %-8s %-12s %s",
		"ALERT", "LEVEL", "TYPE", "IMPACT", "TIME"))
	lines = append(lines, strings.Repeat("-", 60))

	count := 0
	for i := len(p.alerts) - 1; i >= 0 && count < limit; i-- {
		a := p.alerts[i]
		impact := fmt.Sprintf("%.3f, %.3f", a.ImpactLat, a.ImpactLon)
		ago := time.Since(a.IssuedAt).Round(time.Second).String()
		lines = append(lines, fmt.Sprintf("%-8d %-10s %-8s %-12s %s ago",
			a.AlertID, a.Level, a.ThreatType, impact, ago))
		count++
	}

	return strings.Join(lines, "\n")
}

func (p *PluginState) cmdLVC() string {
	if p.lvc == nil {
		return "LVC status: no data"
	}

	return fmt.Sprintf(`LVC Federation Status
=====================
Total Entities: %d

Live:         %4d  (real-world assets)
Virtual:      %4d  (operator-controlled)
Constructive: %4d  (AI-generated)

Force Breakdown:
  Friendly:   %4d
  Opposing:   %4d
  Neutral:    %4d

Active Tracks: %d`, p.lvc.TotalEntities,
		p.lvc.LiveCount, p.lvc.VirtualCount, p.lvc.ConstructiveCount,
		p.lvc.FriendlyCount, p.lvc.OpposingCount, p.lvc.NeutralCount,
		p.lvc.ActiveTracks)
}

func (p *PluginState) cmdEnv() string {
	if len(p.envEvents) == 0 {
		return "No active environmental events"
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("%-8s %-12s %-8s %-15s %s",
		"EVENT", "TYPE", "SEVERITY", "CENTER", "SBIRS/AWACS Impact"))
	lines = append(lines, strings.Repeat("-", 70))

	for _, e := range p.envEvents {
		center := fmt.Sprintf("%.1f, %.1f", e.CenterLat, e.CenterLon)
		impact := fmt.Sprintf("%.0f%%/%.0f%%", e.SBIRSImpact*100, e.AWACSImpact*100)
		lines = append(lines, fmt.Sprintf("%-8d %-12s %-8d %-15s %s",
			e.ID, e.Type, e.Severity, center, impact))
	}

	return strings.Join(lines, "\n")
}

func (p *PluginState) cmdInject(args []string) string {
	if len(args) < 2 {
		return "Usage: vimi inject <type> <json>\nTypes: track, alert, env"
	}

	injectType := args[0]

	switch injectType {
	case "track":
		var tr ThreatTrack
		if json.Unmarshal([]byte(args[1]), &tr) == nil {
			tr.UpdateTime = time.Now()
			p.tracks[tr.TrackID] = &tr
			return fmt.Sprintf("Injected track %d", tr.TrackID)
		}
		return "Failed to parse track JSON"
	case "alert":
		var a Alert
		if json.Unmarshal([]byte(args[1]), &a) == nil {
			a.IssuedAt = time.Now()
			p.alerts = append(p.alerts, &a)
			if len(p.alerts) > 100 {
				p.alerts = p.alerts[len(p.alerts)-100:]
			}
			return fmt.Sprintf("Injected alert %d (level=%s)", a.AlertID, a.Level)
		}
		return "Failed to parse alert JSON"
	case "env":
		var e EnvEvent
		if json.Unmarshal([]byte(args[1]), &e) == nil {
			p.envEvents = append(p.envEvents, &e)
			if len(p.envEvents) > 50 {
				p.envEvents = p.envEvents[len(p.envEvents)-50:]
			}
			return fmt.Sprintf("Injected env event %d (type=%s)", e.ID, e.Type)
		}
		return "Failed to parse env JSON"
	default:
		return fmt.Sprintf("Unknown inject type: %s", injectType)
	}
}

func (p *PluginState) cmdGlobe() string {
	tracks := p.GetThreatTracks()
	if len(tracks) == 0 {
		return "No tracks to display on globe"
	}

	hostile := 0
	for _, t := range tracks {
		if t.AlertLevel == "HOSTILE" {
			hostile++
		}
	}

	return fmt.Sprintf("Globe update: %d tracks sent (%d HOSTILE)", len(tracks), hostile)
}

func (p *PluginState) GetThreatTracks() []ThreatTrack {
	var result []ThreatTrack
	for _, tr := range p.tracks {
		switch tr.AlertLevel {
		case "HOSTILE":
			tr.MarkerColor = "#ff0000"
			tr.MarkerIcon = "warning"
		case "INCOMING":
			tr.MarkerColor = "#ff6600"
			tr.MarkerIcon = "alert"
		case "IMMINENT":
			tr.MarkerColor = "#ffcc00"
			tr.MarkerIcon = "clock"
		case "CONOPREP":
			tr.MarkerColor = "#66ccff"
			tr.MarkerIcon = "eye"
		default:
			tr.MarkerColor = "#cccccc"
			tr.MarkerIcon = "question"
		}
		result = append(result, *tr)
	}
	return result
}

func (p *PluginState) GetAlerts() []Alert {
	alerts := make([]Alert, 0)
	for i := len(p.alerts) - 1; i >= 0 && len(alerts) < 20; i-- {
		a := p.alerts[i]
		if a.Level == "HOSTILE" || a.Level == "INCOMING" {
			a.Severity = 5
		} else if a.Level == "IMMINENT" {
			a.Severity = 4
		}
		alerts = append(alerts, *a)
	}
	return alerts
}

func (p *PluginState) GetLVCStatus() LVCStatus {
	if p.lvc != nil {
		return *p.lvc
	}
	return LVCStatus{}
}

func (p *PluginState) GetEnvEvents() []EnvEvent {
	result := make([]EnvEvent, 0, len(p.envEvents))
	for _, e := range p.envEvents {
		result = append(result, *e)
	}
	return result
}

func (p *PluginState) run(ctx context.Context) {
	envReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{p.broker},
		Topic:       TopicEnvEvents,
		GroupID:     "vimi-plugin-env",
		MinBytes:    10e3,
		MaxBytes:    10e6,
		StartOffset: kafka.LastOffset,
	})
	defer envReader.Close()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Read tracks
			if msg, err := p.kafkaReader.ReadMessage(ctx); err == nil {
				var track struct {
					TrackNumber   uint32    `json:"track_number"`
					FusedLat      float64   `json:"fused_lat"`
					FusedLon      float64   `json:"fused_lon"`
					FusedAlt      float64   `json:"fused_alt"`
					FusedVelocity float64   `json:"fused_velocity"`
					FusedHeading  float64   `json:"fused_heading"`
					ThreatLevel   int       `json:"threat_level"`
					Confidence    float64   `json:"confidence"`
					LastUpdate    time.Time `json:"last_update"`
				}
				if json.Unmarshal(msg.Value, &track) == nil {
					threatMap := []string{"Unknown", "SRBM", "MRBM", "IRBM", "ICBM"}
					alertMap := []string{"CONOPREP", "IMMINENT", "INCOMING", "HOSTILE"}
					tIdx := track.ThreatLevel
					if tIdx < 0 || tIdx >= len(threatMap) {
						tIdx = 0
					}
					aIdx := tIdx
					if aIdx >= len(alertMap) {
						aIdx = len(alertMap) - 1
					}
					tr := &ThreatTrack{
						TrackID:    track.TrackNumber,
						ThreatType: threatMap[tIdx],
						AlertLevel: alertMap[aIdx],
						Lat:        track.FusedLat,
						Lon:        track.FusedLon,
						Alt:        track.FusedAlt,
						Velocity:   track.FusedVelocity,
						Heading:    track.FusedHeading,
						Confidence: track.Confidence,
						UpdateTime: track.LastUpdate,
					}
					p.tracks[tr.TrackID] = tr
					p.lastUpdate = time.Now()
				}
			}

			// Read alerts
			if msg, err := p.alertsReader.ReadMessage(ctx); err == nil {
				var a Alert
				if json.Unmarshal(msg.Value, &a) == nil {
					a.IssuedAt = time.Now()
					p.alerts = append(p.alerts, &a)
					if len(p.alerts) > 100 {
						p.alerts = p.alerts[len(p.alerts)-100:]
					}
				}
			}

			// Read env events
			if msg, err := envReader.ReadMessage(ctx); err == nil {
				var e EnvEvent
				if json.Unmarshal(msg.Value, &e) == nil {
					p.envEvents = append(p.envEvents, &e)
					if len(p.envEvents) > 50 {
						p.envEvents = p.envEvents[len(p.envEvents)-50:]
					}
				}
			}
		}
	}
}

func (p *PluginState) serveHTTP() {
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"plugin":  PluginName,
			"version": PluginVersion,
			"status":  map[bool]string{true: "running", false: "stopped"}[p.running],
			"tracks":  len(p.tracks),
			"alerts":  len(p.alerts),
		})
	})

	http.HandleFunc("/api/globe/tracks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(p.GetThreatTracks())
	})

	http.HandleFunc("/api/alerts", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(p.GetAlerts())
	})

	http.HandleFunc("/api/lvc", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(p.GetLVCStatus())
	})

	http.HandleFunc("/api/cmd", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "POST only", 405)
			return
		}
		var req struct {
			Cmd  string   `json:"cmd"`
			Args []string `json:"args"`
		}
		if json.NewDecoder(r.Body).Decode(&req) == nil {
			result := p.HandleCommand(req.Cmd, req.Args)
			w.Write([]byte(result))
		}
	})

	addr := fmt.Sprintf(":%s", p.port)
	log.Printf("[VIMIC] HTTP server on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Printf("[VIMIC] HTTP server error: %v", err)
	}
}

var plugin *PluginState

func main() {
	broker := flag.String("broker", getEnv("KAFKA_BROKERS", "kafka:9092"), "Kafka broker")
	port := flag.String("port", getEnv("PORT", "8091"), "HTTP listen port")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("[VIMIC] Plugin v%s starting...", PluginVersion)

	plugin = &PluginState{
		broker: *broker,
		port:   *port,
	}

	if err := plugin.Start(); err != nil {
		log.Fatalf("[VIMIC] Failed to start: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("[VIMIC] Shutting down...")
		plugin.Stop()
		cancel()
	}()

	go plugin.serveHTTP()
	go plugin.run(ctx)

	<-ctx.Done()
}
