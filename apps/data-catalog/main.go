// data-catalog — VIMI Data Catalog (JFCDS / OGC CSW)
// Phase 2: Mission Processing
// Provides sensor metadata registry, observation discovery,
// and OGC CSW-style query interface for mission data
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
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	TopicSensorMeta  = "vimi.sensor.metadata"
	TopicObservation = "vimi.observations"
	TopicCatalog     = "vimi.catalog"
)

// OGC CSW queryables
type Queryables struct {
	AnyText, Title, Abstract, Keywords, Creator, Publisher string
	Contributor, Type, Format, Identifier, Source string
	Spatial, Temporal, LatLonBox string
}

// SensorType taxonomy
type SensorType int

const (
	ST_EO SensorType = iota
	ST_IR
	ST_SAR
	ST_RADAR
	ST_ADS_B
	ST_AIS
	ST_HF
)

func (s SensorType) String() string {
	names := []string{"EO/IR", "IR", "SAR", "Radar", "ADS-B", "AIS", "HF"}
	if s < 0 || int(s) >= len(names) {
		return "Unknown"
	}
	return names[s]
}

// Sensor metadata
type Sensor struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Type           SensorType `json:"type"`
	Platform       string   `json:"platform"` // "SBIRS-High", "NRO", "AWACS", "Global Hawk"
	Orbit          string   `json:"orbit"`    // "GEO", "LEO", "MEO", "AIR", "GROUND"
	CoverageRegion string   `json:"coverage_region"`
	CoverageType   string   `json:"coverage_type"` // "GLOBAL", "REGIONAL", "THEATER"
	FOV            float64  `json:"fov_degrees"`  // Field of view
	SpectralMin    float64  `json:"spectral_min_um"`  // microns
	SpectralMax    float64  `json:"spectral_max_um"`
	Resolution     float64  `json:"resolution_m"`    // meters
	RevisitTime    float64  `json:"revisit_s"`       // seconds
	Status         string   `json:"status"`    // "OPERATIONAL", "DEGRADED", "STANDBY"
	Operator       string   `json:"operator"`  // "USSF", "USN", "USAF"
	
	// OGC CSW metadata
	Title       string   `json:"title"`
	Keywords    []string `json:"keywords"`
	Description string   `json:"abstract"`
	
	// Bounding box
	MinLat, MaxLat float64 `json:"min_lat,max_lat"`
	MinLon, MaxLon float64 `json:"min_lon,max_lon"`
	
	LastUpdate time.Time `json:"last_update"`
}

// Observation record
type Observation struct {
	ID           string    `json:"id"`
	SensorID     string    `json:"sensor_id"`
	ObsTime      time.Time `json:"observation_time"`
	IngestTime   time.Time `json:"ingest_time"`
	ProductType  string    `json:"product_type"`  // "L0", "L1A", "L1B", "L2", "L3"
	FeatureType  string    `json:"feature_type"` // "IR_DETECTION", "TRACK", "ALERT", "IMAGE"
	Region       string    `json:"region"`
	Lat, Lon     float64   `json:"lat,lon"`
	MinLat, MaxLat float64 `json:"min_lat,max_lat"`
	MinLon, MaxLon float64 `json:"min_lon,max_lon"`
	SizeBytes    int64     `json:"size_bytes"`
	Format       string    `json:"format"`  // "DIS_PDU", "JSON", "MSG", "STANAG"
	URI          string    `json:"uri"`    // Storage URI
	Checksum     string    `json:"checksum"`
	Validity     string    `json:"validity"` // "VALID", "DEGRADED", "CORRUPT"
	
	// JRMB data quality
	QualityFlags uint32 `json:"quality_flags"`
}

// BoundingBox for OGC CSW WGS84 bounding box
type BoundingBox struct {
	MinLat float64 `json:"min_lat"`
	MinLon float64 `json:"min_lon"`
	MaxLat float64 `json:"max_lat"`
	MaxLon float64 `json:"max_lon"`
}

// CatalogRecord is the searchable record
type CatalogRecord struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"` // "sensor" / "observation"
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Keywords    []string `json:"keywords"`
	
	// OGC CSW bounding box
	WGS84BoundingBox BoundingBox `json:"wgs84_bounding_box"`
	
	// Temporal extent
	TempExtentBegin time.Time `json:"temp_extent_begin"`
	TempExtentEnd   time.Time `json:"temp_extent_end"`
	
	// Resource links
	URLs []URL `json:"urls"`
	
	Updated time.Time `json:"updated"`
}

type URL struct {
	Protocol string `json:"protocol"`
	URL     string `json:"url"`
	Name    string `json:"name"`
}

type Catalog struct {
	sensors      map[string]*Sensor
	observations map[string]*Observation
	records      []*CatalogRecord
}

func newCatalog() *Catalog {
	return &Catalog{
		sensors:      make(map[string]*Sensor),
		observations: make(map[string]*Observation),
		records:      make([]*CatalogRecord, 0),
	}
}

// OGC CQL filter (simplified subset)
func matchesFilter(record *CatalogRecord, filter string) bool {
	if filter == "" {
		return true
	}

	// Simple keyword search
	filter = strings.ToLower(filter)
	searchIn := strings.ToLower(record.Title + " " + record.Description + " " + strings.Join(record.Keywords, " "))
	
	// Support quoted phrases
	quoted := regexp.MustCompile(`"([^"]+)"`)
	for _, match := range quoted.FindAllStringSubmatch(filter, -1) {
		if !strings.Contains(searchIn, strings.ToLower(match[1])) {
			return false
		}
	}

	// Support AND/OR
	if strings.Contains(filter, " and ") {
		parts := strings.Split(filter, " and ")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			part = quoted.ReplaceAllString(part, "")
			if part != "" && !strings.Contains(searchIn, part) {
				return false
			}
		}
		return true
	}

	return strings.Contains(searchIn, filter)
}

func (c *Catalog) search(filter string, bbox *struct{ MinLat, MinLon, MaxLat, MaxLon float64 }, start, end *time.Time, limit int) []*CatalogRecord {
	var results []*CatalogRecord

	for _, r := range c.records {
		if !matchesFilter(r, filter) {
			continue
		}

		// BBOX filter
		if bbox != nil {
			if r.WGS84BoundingBox.MaxLat < bbox.MinLat || r.WGS84BoundingBox.MinLat > bbox.MaxLat ||
				r.WGS84BoundingBox.MaxLon < bbox.MinLon || r.WGS84BoundingBox.MinLon > bbox.MaxLon {
				continue
			}
		}

		// Temporal filter
		if start != nil && r.TempExtentEnd.Before(*start) {
			continue
		}
		if end != nil && r.TempExtentBegin.After(*end) {
			continue
		}

		results = append(results, r)
	}

	// Sort by updated time
	sort.Slice(results, func(i, j int) bool {
		return results[i].Updated.After(results[j].Updated)
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results
}

func (c *Catalog) addSensor(s *Sensor) {
	c.sensors[s.ID] = s

	record := &CatalogRecord{
		ID:          "sensor:" + s.ID,
		Type:        "sensor",
		Title:       s.Title,
		Description: s.Description,
		Keywords:    s.Keywords,
	}
	record.WGS84BoundingBox = BoundingBox{
		MinLat: s.MinLat, MinLon: s.MinLon,
		MaxLat: s.MaxLat, MaxLon: s.MaxLon,
	}
	record.TempExtentBegin = s.LastUpdate
	record.TempExtentEnd = s.LastUpdate.Add(24 * time.Hour) // Assume daily refresh
	record.Updated = s.LastUpdate

	c.records = append(c.records, record)
}

func (c *Catalog) addObservation(o *Observation) {
	c.observations[o.ID] = o

	sensor, ok := c.sensors[o.SensorID]
	title := fmt.Sprintf("%s %s %s", o.ProductType, o.FeatureType, o.ObsTime.Format("20060102150405"))
	if ok {
		title = fmt.Sprintf("%s %s %s", sensor.Name, o.FeatureType, o.ObsTime.Format("20060102150405"))
	}

	record := &CatalogRecord{
		ID:          "obs:" + o.ID,
		Type:        "observation",
		Title:       title,
		Description: fmt.Sprintf("%s from %s, region %s", o.FeatureType, o.SensorID, o.Region),
		Keywords:    []string{o.ProductType, o.FeatureType, o.Region, o.Format},
	}
	record.WGS84BoundingBox = BoundingBox{
		MinLat: o.MinLat, MinLon: o.MinLon,
		MaxLat: o.MaxLat, MaxLon: o.MaxLon,
	}
	record.TempExtentBegin = o.ObsTime
	record.TempExtentEnd = o.ObsTime
	record.Updated = o.IngestTime

	if o.URI != "" {
		record.URLs = append(record.URLs, URL{
			Protocol: "HTTPS",
			URL:      o.URI,
			Name:     o.Format,
		})
	}

	c.records = append(c.records, record)
}

func (c *Catalog) getCapabilities() map[string]interface{} {
	return map[string]interface{}{
		"service":          "VIMI-DataCatalog",
		"version":          "0.1.0",
		"serviceType":      "OGC:CSW",
		"serviceTypeVersion": []string{"2.0.2"},
		"title":            "VIMI Joint Federated Common Data Services Catalog",
		"abstract":         "Sensor registry and observation discovery for VIMI mission processing",
		"keywords":         []string{"VIMI", "JFCDS", "sensor", "OPIR", "missile warning", "DIS"},
		"operatesOn":      []string{"Sensor", "Observation"},
		"queryables":       []string{"AnyText", "Title", "Abstract", "Keywords", "Creator", "Publisher", "Contributor", "Type", "Format", "Identifier", "Source", "Spatial", "Temporal", "LatLonBox"},
	}
}

var (
	cat           *Catalog
	kafkaWriter   *kafka.Writer
	kafkaReaders  []*kafka.Reader
	kafkaBroker   = getEnv("KAFKA_BROKERS", "kafka:9092")
	port          = getEnv("PORT", "8087")
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func run(ctx context.Context) {
	topics := []string{TopicSensorMeta, TopicObservation}
	for _, topic := range topics {
		reader := kafka.NewReader(kafka.ReaderConfig{
			Brokers:   []string{kafkaBroker},
			Topic:     topic,
			GroupID:   "data-catalog",
			MinBytes:  10e3,
			MaxBytes:  10e6,
			StartOffset: kafka.LastOffset,
		})
		kafkaReaders = append(kafkaReaders, reader)
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			for _, reader := range kafkaReaders {
				msg, err := reader.ReadMessage(ctx)
				if err != nil {
					continue
				}

				switch msg.Topic {
				case TopicSensorMeta:
					var s Sensor
					if json.Unmarshal(msg.Value, &s) == nil {
						cat.addSensor(&s)
					}
				case TopicObservation:
					var o Observation
					if json.Unmarshal(msg.Value, &o) == nil {
						cat.addObservation(&o)
					}
				}
			}
		}
	}
}

// CSW-style HTTP handlers

func capabilitiesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cat.getCapabilities())
}

func getRecordsHandler(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("filter")
	bboxStr := r.URL.Query().Get("bbox")
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	maxStr := r.URL.Query().Get("max")

	maxRecords := 100
	if maxStr != "" {
		if m, err := strconv.Atoi(maxStr); err == nil {
			maxRecords = m
		}
	}

	var bbox *struct{ MinLat, MinLon, MaxLat, MaxLon float64 }
	if bboxStr != "" {
		parts := strings.Split(bboxStr, ",")
		if len(parts) == 4 {
			bbox = &struct{ MinLat, MinLon, MaxLat, MaxLon float64 }{}
			fmt.Sscanf(parts[0], "%f", &bbox.MinLon)
			fmt.Sscanf(parts[1], "%f", &bbox.MinLat)
			fmt.Sscanf(parts[2], "%f", &bbox.MaxLon)
			fmt.Sscanf(parts[3], "%f", &bbox.MaxLat)
		}
	}

	var start, end *time.Time
	if startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			start = &t
		}
	}
	if endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			end = &t
		}
	}

	results := cat.search(filter, bbox, start, end, maxRecords)

	resp := map[string]interface{}{
		"recordsMatched": len(results),
		"recordsReturned": len(results),
		"records":         results,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func getRecordByIDHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id required", 400)
		return
	}

	for _, s := range cat.sensors {
		if "sensor:"+s.ID == id {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(s)
			return
		}
	}
	for _, o := range cat.observations {
		if "obs:"+o.ID == id {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(o)
			return
		}
	}
	http.Error(w, "not found", 404)
}

func sensorsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	list := make([]*Sensor, 0, len(cat.sensors))
	for _, s := range cat.sensors {
		list = append(list, s)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].LastUpdate.After(list[j].LastUpdate)
	})
	json.NewEncoder(w).Encode(list)
}

func observationsHandler(w http.ResponseWriter, r *http.Request) {
	region := r.URL.Query().Get("region")
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	list := make([]*Observation, 0)
	for _, o := range cat.observations {
		if region != "" && o.Region != region {
			continue
		}
		list = append(list, o)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].ObsTime.After(list[j].ObsTime)
	})
	if len(list) > limit {
		list = list[:limit]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func registerSensorHandler(w http.ResponseWriter, r *http.Request) {
	var s Sensor
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	cat.addSensor(&s)
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(map[string]string{"status": "registered", "id": s.ID})
}

func ingestObservationHandler(w http.ResponseWriter, r *http.Request) {
	var o Observation
	if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	cat.addObservation(&o)
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(map[string]string{"status": "ingested", "id": o.ID})
}

type HealthResponse struct {
	Service       string    `json:"service"`
	Version       string    `json:"version"`
	Timestamp     time.Time `json:"timestamp"`
	Status        string    `json:"status"`
	Sensors       int       `json:"sensors"`
	Observations  int       `json:"observations"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Service:      "data-catalog",
		Version:      "0.1.0",
		Timestamp:    time.Now().UTC(),
		Status:       "healthy",
		Sensors:      len(cat.sensors),
		Observations: len(cat.observations),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	cat = newCatalog()

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
	http.HandleFunc("/csw/capabilities", capabilitiesHandler)
	http.HandleFunc("/csw/records", getRecordsHandler)
	http.HandleFunc("/csw/record", getRecordByIDHandler)
	http.HandleFunc("/sensors", sensorsHandler)
	http.HandleFunc("/sensors/register", registerSensorHandler)
	http.HandleFunc("/observations", observationsHandler)
	http.HandleFunc("/observations/ingest", ingestObservationHandler)

	log.Printf("data-catalog starting")
	log.Printf("Kafka broker: %s", kafkaBroker)
	go run(ctx)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("HTTP server on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

// Import math for sqrt
var _ = math.Pow
