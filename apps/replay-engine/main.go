// replay-engine — VIMI Replay Engine Service
// Phase 2: Mission Processing
// Records DIS entity state and events, provides playback for exercises,
// post-mission analysis, and LVC federation training
package main

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	//"io"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	//"strconv"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
)

const (
	TopicDISIn   = "vimi.dis.entity-state"
	TopicAlerts  = "vimi.alerts"
	TopicReplay  = "vimi.replay.events"
	RecordDir    = "/var/vimi/recordings"
)

// PDUType DIS enumeration
type PDUType uint8

const (
	PDUEntityState PDUType = 1
	PDUFire        PDUType = 2
	PDUDetonation  PDUType = 3
	PDUAction      PDUType = 4
	PDUSignal      PDUType = 5
)

// RecordedPDU with timestamp
type RecordedPDU struct {
	Sequence   uint64    `json:"sequence"`
	ReceiveTS  time.Time `json:"receive_ts"`  // when we received it
	OriginTS   time.Time `json:"origin_ts"`  // timestamp from PDU
	SiteID     uint16    `json:"site_id"`
	AppID      uint16    `json:"app_id"`
	EntityID   uint32    `json:"entity_id"`
	ForceID    uint8     `json:"force_id"`
	PDUType    PDUType   `json:"pdu_type"`
	LVCType    string    `json:"lvc_type"`
	Raw        []byte    `json:"raw,omitempty"`
	
	// Parsed entity state (if applicable)
	Lat        float64   `json:"lat,omitempty"`
	Lon        float64   `json:"lon,omitempty"`
	Alt        float64   `json:"alt,omitempty"`
	Yaw        float32   `json:"yaw,omitempty"`
	Pitch      float32   `json:"pitch,omitempty"`
	Roll       float32   `json:"roll,omitempty"`
	VX         float32   `json:"vx,omitempty"`
	VY         float32   `json:"vy,omitempty"`
	VZ         float32   `json:"vz,omitempty"`
	Marking    string    `json:"marking,omitempty"`
}

// Recording metadata
type Recording struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	PDUCount    uint64    `json:"pdu_count"`
	FileSize    int64     `json:"file_size"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
}

// replayState
type replayState struct {
	recordings    map[string]*Recording
	currentSeq    uint64
	recording     bool
	currentFile   *os.File
	recordingName string
	startTime     time.Time
}

func newReplayState() *replayState {
	// Ensure record directory exists
	os.MkdirAll(RecordDir, 0755)
	return &replayState{
		recordings: make(map[string]*Recording),
	}
}

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

func parseEntityStatePDU(data []byte) *RecordedPDU {
	if len(data) < 68 {
		return nil
	}

	pdu := &RecordedPDU{
		PDUType: PDUEntityState,
		SiteID:  binary.BigEndian.Uint16(data[10:12]),
		AppID:  binary.BigEndian.Uint16(data[12:14]),
		EntityID: binary.BigEndian.Uint32(data[14:18]),
		ForceID: data[18],
	}

	// Location (ECEF) at bytes 52-64
	x := float64(math.Float32frombits(binary.BigEndian.Uint32(data[52:56])))
	y := float64(math.Float32frombits(binary.BigEndian.Uint32(data[56:60])))
	z := float64(math.Float32frombits(binary.BigEndian.Uint32(data[60:64])))
	
	pdu.Lat, pdu.Lon, pdu.Alt = ecefToGeodetic(x, y, z)

	// Orientation at bytes 26-38
	pdu.Yaw = math.Float32frombits(binary.BigEndian.Uint32(data[28:32]))
	pdu.Pitch = math.Float32frombits(binary.BigEndian.Uint32(data[32:36]))
	pdu.Roll = math.Float32frombits(binary.BigEndian.Uint32(data[36:40]))

	// Velocity at bytes 40-52
	pdu.VX = math.Float32frombits(binary.BigEndian.Uint32(data[40:44]))
	pdu.VY = math.Float32frombits(binary.BigEndian.Uint32(data[44:48]))
	pdu.VZ = math.Float32frombits(binary.BigEndian.Uint32(data[48:52]))

	// Marking (bytes 104-120, 16 bytes ASCII)
	marking := make([]byte, 16)
	copy(marking, data[104:120])
	pdu.Marking = string(marking)

	return pdu
}

func parsePDUHeaders(data []byte) (siteID, appID uint16, entityID uint32, pduType PDUType) {
	if len(data) < 20 {
		return
	}
	siteID = binary.BigEndian.Uint16(data[10:12])
	appID = binary.BigEndian.Uint16(data[12:14])
	entityID = binary.BigEndian.Uint32(data[14:18])
	pduType = PDUType(data[3])
	return
}

func (rs *replayState) startRecording(name string) error {
	if rs.recording {
		return fmt.Errorf("already recording: %s", rs.recordingName)
	}

	ts := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("%s_%s.dispcap", name, ts)
	path := filepath.Join(RecordDir, filename)

	f, err := os.Create(path)
	if err != nil {
		return err
	}

	rs.currentFile = f
	rs.recordingName = name
	rs.recording = true
	rs.startTime = time.Now()
	rs.currentSeq = 0

	// Write header
	header := map[string]interface{}{
		"version":    "VIMI-RECORD-1.0",
		"name":       name,
		"start_time": rs.startTime,
	}
	hdata, _ := json.Marshal(header)
	f.Write(hdata)
	f.Write([]byte("\n"))

	log.Printf("REPLAY: started recording %s -> %s", name, filename)
	return nil
}

func (rs *replayState) stopRecording() (*Recording, error) {
	if !rs.recording {
		return nil, fmt.Errorf("not recording")
	}

	rs.recording = false
	rs.currentFile.Close()

	info, _ := os.Stat(rs.currentFile.Name())
	rec := &Recording{
		ID:          filepath.Base(rs.currentFile.Name()),
		Name:        rs.recordingName,
		StartTime:   rs.startTime,
		EndTime:     time.Now(),
		PDUCount:    rs.currentSeq,
		FileSize:    info.Size(),
		Description: "",
		Tags:        []string{},
	}
	rs.recordings[rec.ID] = rec
	rs.currentFile = nil

	log.Printf("REPLAY: stopped recording %s (%d PDUs, %s)", rec.Name, rec.PDUCount, formatBytes(rec.FileSize))
	return rec, nil
}

func (rs *replayState) recordPDU(pdu *RecordedPDU) error {
	if !rs.recording || rs.currentFile == nil {
		return nil
	}

	rs.currentSeq++
	pdu.Sequence = rs.currentSeq

	data, err := json.Marshal(pdu)
	if err != nil {
		return err
	}

	_, err = rs.currentFile.Write(data)
	rs.currentFile.Write([]byte("\n"))
	return err
}

func formatBytes(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	if n < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(n)/1024/1024)
}

func loadRecordings(rs *replayState) {
	ents, err := os.ReadDir(RecordDir)
	if err != nil {
		return
	}

	for _, ent := range ents {
		if filepath.Ext(ent.Name()) != ".dispcap" {
			continue
		}
		path := filepath.Join(RecordDir, ent.Name())
		info, _ := os.Stat(path)
		
		rec := &Recording{
			ID:       ent.Name(),
			Name:     "unlabeled",
			FileSize: info.Size(),
			Tags:     []string{},
		}

		// Try to read header
		f, _ := os.Open(path)
		if f != nil {
			scanner := bufio.NewScanner(f)
			if scanner.Scan() {
				var header map[string]interface{}
				if json.Unmarshal(scanner.Bytes(), &header) == nil {
					if n, ok := header["name"].(string); ok {
						rec.Name = n
					}
					if t, ok := header["start_time"].(string); ok {
						if st, err := time.Parse(time.RFC3339, t); err == nil {
							rec.StartTime = st
						}
					}
				}
			}
			f.Close()
		}

		// Count PDUs
		f, _ = os.Open(path)
		if f != nil {
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				rec.PDUCount++
			}
			f.Close()
			rec.PDUCount-- // subtract header
		}

		rs.recordings[rec.ID] = rec
	}
}

// Playback state
type playbackState struct {
	active      bool
	recordingID string
	position    uint64
	speed       float64 // 1.0 = real-time, 2.0 = 2x, etc.
	paused      bool
}

var (
	rs   *replayState
	ps   *playbackState
	kafkaWriter *kafka.Writer
	kafkaReader *kafka.Reader
	kafkaBroker = getEnv("KAFKA_BROKERS", "kafka:9092")
	port        = getEnv("PORT", "8086")
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
		Topic:     TopicDISIn,
		GroupID:   "replay-engine",
		MinBytes:  10e3,
		MaxBytes:  10e6,
		StartOffset: kafka.LastOffset,
	})
	defer reader.Close()

	// Also subscribe to alerts for context
	alertReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{kafkaBroker},
		Topic:     TopicAlerts,
		GroupID:   "replay-engine-alerts",
		MinBytes:  10e3,
		MaxBytes:  10e6,
		StartOffset: kafka.LastOffset,
	})
	defer alertReader.Close()

	alertCh := make(chan *RecordedPDU, 100)

	// Start alert reader goroutine
	go func() {
		for {
			msg, err := alertReader.ReadMessage(ctx)
			if err != nil {
				continue
			}
			siteID, appID, entityID, _ := parsePDUHeaders(msg.Value)
			pdu := &RecordedPDU{
				ReceiveTS: time.Now(),
				SiteID:    siteID,
				AppID:     appID,
				EntityID:  entityID,
				PDUType:   PDUType(msg.Value[3]),
				LVCType:   "Alert",
				Raw:       msg.Value,
			}
			select {
			case alertCh <- pdu:
			default:
			}
		}
	}()

	seqBuf := 0

	for {
		select {
		case <-ctx.Done():
			return
		case pdu := <-alertCh:
			if rs.recording {
				rs.recordPDU(pdu)
			}
		default:
			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}

			siteID, appID, entityID, pduType := parsePDUHeaders(msg.Value)
			
			// Get LVC type from headers
			lvcType := "Unknown"
			for _, h := range msg.Headers {
				if h.Key == "lvc_type" {
					lvcType = string(h.Value)
				}
			}

			pdu := &RecordedPDU{
				ReceiveTS: time.Now(),
				OriginTS:  time.Now(), // TODO: extract from PDU timestamp
				SiteID:    siteID,
				AppID:     appID,
				EntityID:  entityID,
				PDUType:   pduType,
				ForceID:   msg.Value[18],
				LVCType:   lvcType,
				Raw:       msg.Value,
			}

			// Parse entity state data
			if pduType == PDUEntityState {
				if ep := parseEntityStatePDU(msg.Value); ep != nil {
					pdu.Lat = ep.Lat
					pdu.Lon = ep.Lon
					pdu.Alt = ep.Alt
					pdu.Yaw = ep.Yaw
					pdu.Pitch = ep.Pitch
					pdu.Roll = ep.Roll
					pdu.VX = ep.VX
					pdu.VY = ep.VY
					pdu.VZ = ep.VZ
					pdu.Marking = ep.Marking
				}
			}

			// Record if active
			if rs.recording {
				if err := rs.recordPDU(pdu); err != nil {
					log.Printf("REPLAY: record error: %v", err)
				}
				seqBuf++
				if seqBuf >= 100 {
					rs.currentFile.Sync()
					seqBuf = 0
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
	Recording    bool      `json:"recording"`
	RecordingName string   `json:"recording_name,omitempty"`
	PDUCount     uint64    `json:"pdu_count"`
	Recordings   int       `json:"recordings"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Service:    "replay-engine",
		Version:    "0.1.0",
		Timestamp:  time.Now().UTC(),
		Status:     "healthy",
		Recording:  rs.recording,
		PDUCount:   rs.currentSeq,
		Recordings: len(rs.recordings),
	}
	if rs.recording {
		resp.RecordingName = rs.recordingName
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	rs = newReplayState()
	ps = &playbackState{speed: 1.0}
	loadRecordings(rs)

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
		if rs.recording {
			rs.stopRecording()
		}
		log.Println("Shutting down...")
		cancel()
	}()

	http.Handle("/metrics", promhttp.Handler())
http.HandleFunc("/health", healthHandler)

	// Recording control
	http.HandleFunc("/record/start", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			name = "exercise"
		}
		if err := rs.startRecording(name); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		w.WriteHeader(204)
	})

	http.HandleFunc("/record/stop", func(w http.ResponseWriter, r *http.Request) {
		rec, err := rs.stopRecording()
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rec)
	})

	// List recordings
	http.HandleFunc("/recordings", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		list := make([]*Recording, 0, len(rs.recordings))
		for _, rec := range rs.recordings {
			list = append(list, rec)
		}
		sort.Slice(list, func(i, j int) bool {
			return list[i].StartTime.After(list[j].StartTime)
		})
		json.NewEncoder(w).Encode(list)
	})

	// Get recording
	http.HandleFunc("/recordings/", func(w http.ResponseWriter, r *http.Request) {
		id := filepath.Base(r.URL.Path)
		rec, ok := rs.recordings[id]
		if !ok {
			http.Error(w, "not found", 404)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rec)
	})

	// Download recording file
	http.HandleFunc("/download/", func(w http.ResponseWriter, r *http.Request) {
		id := filepath.Base(r.URL.Path)
		path := filepath.Join(RecordDir, id)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			http.Error(w, "not found", 404)
			return
		}
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", id))
		w.Header().Set("Content-Type", "application/octet-stream")
		http.ServeFile(w, r, path)
	})

	log.Printf("replay-engine starting")
	log.Printf("Kafka broker: %s", kafkaBroker)
	log.Printf("Recordings dir: %s", RecordDir)
	go run(ctx)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("HTTP server on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
