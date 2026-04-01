// data-catalog — VIMI Data Catalog Service
// Phase 2: Mission Processing
// Implements JFCDS data discovery and OGC CSW (Catalog Service for the Web)
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
)

const (
	TopicTracks    = "vimi.fusion.tracks"
	TopicAlerts    = "vimi.alerts"
	TopicEnvEvents = "vimi.env.events"
	RecordingsDir  = "/var/vimi/recordings"
)

// OGC CSW response containers
type CSWGetRecordsResponse struct {
	XMLName xml.Name `xml:"GetRecordsResponse"`
	SearchStatus struct {
		Timestamp string `xml:"timestamp,attr"`
	} `xml:"SearchStatus"`
	SearchResults struct {
		NumberOfRecordsMatched string          `xml:"numberOfRecordsMatched,attr"`
		NumberOfRecordsReturned string         `xml:"numberOfRecordsReturned,attr"`
		NextRecord            string           `xml:"nextRecord,attr"`
		Record                []CatalogRecord  `xml:"Record"`
	} `xml:"SearchResults"`
}

type CatalogRecord struct {
	XMLName   xml.Name `xml:"Record"`
	Title     string   `xml:"Title"`
	Abstract  string   `xml:"Abstract"`
	Keywords  []string `xml:"Keywords>Keyword"`
	Type      string   `xml:"Type"`
	Format    string   `xml:"Format"`
	Identifier string  `xml:"Identifier"`
	BoundingBox struct {
		Minx, Miny, Maxx, Maxy string `xml:"minx,attr,ymin,attr,maxx,attr,ymax,attr"`
	} `xml:"BoundingBox"`
	CRS       string `xml:"CRS"`
	Created   string `xml:"Created"`
	Modified  string `xml:"Modified"`
}

// Asset types
type AssetType string

const (
	AssetTrack      AssetType = "track"
	AssetAlert      AssetType = "alert"
	AssetSensorData AssetType = "sensor-data"
	AssetRecording  AssetType = "recording"
	AssetEnvData    AssetType = "environmental"
	AssetMapTile    AssetType = "map-tile"
)

// DataAsset in the catalog
type DataAsset struct {
	ID           string     `json:"id"`
	Type         AssetType  `json:"type"`
	Name         string     `json:"name"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	Format       string     `json:"format"`
	MimeType     string     `json:"mime_type"`
	SizeBytes    int64      `json:"size_bytes"`
	CoverageStart time.Time `json:"coverage_start"`
	CoverageEnd   time.Time `json:"coverage_end"`
	BBoxMinLat   float64    `json:"bbox_min_lat"`
	BBoxMinLon   float64    `json:"bbox_min_lon"`
	BBoxMaxLat   float64    `json:"bbox_max_lat"`
	BBoxMaxLon   float64    `json:"bbox_max_lon"`
	Classification string   `json:"classification"`
	Caveats      []string   `json:"caveats"`
	DataSource   string     `json:"data_source"`
	CreatedAt    time.Time  `json:"created_at"`
	ModifiedAt   time.Time  `json:"modified_at"`
	SHA256       string     `json:"sha256,omitempty"`
	Keywords     []string   `json:"keywords"`
	RecordingID  string     `json:"recording_id,omitempty"`
	PDUCount     uint64     `json:"pdu_count,omitempty"`
	AccessURL    string     `json:"access_url,omitempty"`
	ThumbnailURL string     `json:"thumbnail_url,omitempty"`
	CRS          string     `json:"crs"`
}

// Query filters
type CatalogQuery struct {
	Type           AssetType
	Keywords       []string
	Classification string
	TimeStart      *time.Time
	TimeEnd        *time.Time
	BBox          *BoundingBox
	Format         string
	Limit          int
	Offset         int
}

type BoundingBox struct {
	MinLat, MinLon, MaxLat, MaxLon float64
}

// catalogState
type catalogState struct {
	assets    map[string]*DataAsset
	nextID    uint64
	indexByKW map[string][]string
}

func newCatalogState() *catalogState {
	return &catalogState{
		assets:    make(map[string]*DataAsset),
		nextID:    1,
		indexByKW: make(map[string][]string),
	}
}

func (cs *catalogState) addAsset(a *DataAsset) {
	a.ID = fmt.Sprintf("urn:vimi:data:%d", cs.nextID)
	cs.nextID++

	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now()
	}
	a.ModifiedAt = time.Now()

	if a.CRS == "" {
		a.CRS = "EPSG:4326"
	}
	if a.Classification == "" {
		a.Classification = "UNCLASSIFIED"
	}

	cs.assets[a.ID] = a

	for _, kw := range a.Keywords {
		kw = strings.ToLower(kw)
		cs.indexByKW[kw] = append(cs.indexByKW[kw], a.ID)
	}
}

func (cs *catalogState) query(q *CatalogQuery) []*DataAsset {
	var results []*DataAsset

	for _, a := range cs.assets {
		if q.Type != "" && a.Type != q.Type {
			continue
		}
		if q.Classification != "" && a.Classification != q.Classification {
			continue
		}
		if q.Format != "" && a.Format != q.Format && a.MimeType != q.Format {
			continue
		}
		if q.TimeStart != nil && a.CoverageEnd.Before(*q.TimeStart) {
			continue
		}
		if q.TimeEnd != nil && a.CoverageStart.After(*q.TimeEnd) {
			continue
		}
		if q.BBox != nil {
			if a.BBoxMaxLat < q.BBox.MinLat || a.BBoxMinLat > q.BBox.MaxLat ||
				a.BBoxMaxLon < q.BBox.MinLon || a.BBoxMinLon > q.BBox.MaxLon {
				continue
			}
		}
		if len(q.Keywords) > 0 {
			match := false
			for _, qkw := range q.Keywords {
				qkw = strings.ToLower(qkw)
				if ids, ok := cs.indexByKW[qkw]; ok {
					for _, id := range ids {
						if id == a.ID {
							match = true
							break
						}
					}
				}
				if match {
					break
				}
			}
			if !match {
				continue
			}
		}
		results = append(results, a)
	}

	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].ModifiedAt.After(results[i].ModifiedAt) {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	if q.Offset > 0 && q.Offset < len(results) {
		results = results[q.Offset:]
	}
	if q.Limit > 0 && q.Limit < len(results) {
		results = results[:q.Limit]
	}

	return results
}

func (cs *catalogState) loadRecordings() {
	ents, err := os.ReadDir(RecordingsDir)
	if err != nil {
		return
	}

	for _, ent := range ents {
		if filepath.Ext(ent.Name()) != ".dispcap" {
			continue
		}
		path := filepath.Join(RecordingsDir, ent.Name())
		info, _ := os.Stat(path)

		a := &DataAsset{
			Type:          AssetRecording,
			Name:          ent.Name(),
			Title:         "DIS Recording: " + ent.Name(),
			Description:   "Recorded DIS PDU capture",
			Format:        "application/json",
			MimeType:      "application/json",
			SizeBytes:     info.Size(),
			Classification: "UNCLASSIFIED",
			DataSource:    "replay-engine",
			Keywords:      []string{"dis", "pdu", "recording", "entity-state", "lvc"},
			RecordingID:   ent.Name(),
			AccessURL:     fmt.Sprintf("/api/v1/recordings/%s/download", ent.Name()),
			CRS:           "EPSG:4326",
			CoverageStart: info.ModTime().Add(-24 * time.Hour),
			CoverageEnd:   info.ModTime(),
		}

		f, _ := os.Open(path)
		if f != nil {
			defer f.Close()
			count := int64(0)
			sc := bufio.NewScanner(f)
			for sc.Scan() {
				count++
			}
			a.PDUCount = uint64(count - 1)
		}

		cs.addAsset(a)
	}
}

func handleGetRecords(w http.ResponseWriter, r *http.Request) {
	var requestXML string
	if r.Method == "POST" {
		body, _ := io.ReadAll(r.Body)
		requestXML = string(body)
	} else {
		requestXML = r.URL.RawQuery
	}

	var queryKeywords []string
	var queryType string
	var queryBBox *BoundingBox

	if strings.Contains(requestXML, "GetRecords") {
		if strings.Contains(requestXML, "TypeName>Track</TypeName") {
			queryType = "track"
		} else if strings.Contains(requestXML, "TypeName>Alert</TypeName") {
			queryType = "alert"
		} else if strings.Contains(requestXML, "TypeName>Recording</TypeName") {
			queryType = "recording"
		}

		for _, kw := range []string{"missile", "ballistic", "sbirs", "awacs", "dis", "entity"} {
			if strings.Contains(strings.ToLower(requestXML), kw) {
				queryKeywords = append(queryKeywords, kw)
			}
		}

		if strings.Contains(requestXML, "BoundingBox") {
			queryBBox = &BoundingBox{-90, -180, 90, 180}
		}
	}

	q := &CatalogQuery{
		Type:      AssetType(queryType),
		Keywords:  queryKeywords,
		BBox:      queryBBox,
		Limit:     100,
	}

	results := cs.query(q)

	resp := CSWGetRecordsResponse{}
	resp.SearchStatus.Timestamp = time.Now().UTC().Format(time.RFC3339)
	resp.SearchResults.NumberOfRecordsMatched = strconv.Itoa(len(results))
	resp.SearchResults.NumberOfRecordsReturned = strconv.Itoa(len(results))
	resp.SearchResults.NextRecord = "0"

	for _, a := range results {
		rec := CatalogRecord{
			Title:      a.Title,
			Abstract:   a.Description,
			Type:       string(a.Type),
			Format:     a.MimeType,
			Identifier: a.ID,
			CRS:        a.CRS,
			Created:    a.CreatedAt.Format(time.RFC3339),
			Modified:   a.ModifiedAt.Format(time.RFC3339),
		}
		rec.Keywords = a.Keywords
		rec.BoundingBox.Minx = strconv.FormatFloat(a.BBoxMinLon, 'f', 6, 64)
		rec.BoundingBox.Miny = strconv.FormatFloat(a.BBoxMinLat, 'f', 6, 64)
		rec.BoundingBox.Maxx = strconv.FormatFloat(a.BBoxMaxLon, 'f', 6, 64)
		rec.BoundingBox.Maxy = strconv.FormatFloat(a.BBoxMaxLat, 'f', 6, 64)

		resp.SearchResults.Record = append(resp.SearchResults.Record, rec)
	}

	w.Header().Set("Content-Type", "application/xml")
	xml.NewEncoder(w).Encode(resp)
}

var (
	cs          *catalogState
	kafkaBroker = getEnv("KAFKA_BROKERS", "kafka:9092")
	port        = getEnv("PORT", "8087")
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func run(ctx context.Context) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{kafkaBroker},
		Topic:        TopicTracks,
		GroupID:     "data-catalog-tracks",
		MinBytes:    10e3,
		MaxBytes:    10e6,
		StartOffset: kafka.LastOffset,
	})
	defer reader.Close()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cs.loadRecordings()
		default:
			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}

			var track struct {
				TrackNumber  uint32    `json:"track_number"`
				FusedLat    float64   `json:"fused_lat"`
				FusedLon    float64   `json:"fused_lon"`
				FusedAlt    float64   `json:"fused_alt"`
				ThreatLevel int       `json:"threat_level"`
				Confidence  float64   `json:"confidence"`
				UpdateCount int       `json:"update_count"`
				LastUpdate  time.Time `json:"last_update"`
				Sources     []string  `json:"sources"`
			}
			if json.Unmarshal(msg.Value, &track) == nil {
				keywords := []string{"track", "fusion", fmt.Sprintf("threat-level-%d", track.ThreatLevel)}
				for _, s := range track.Sources {
					keywords = append(keywords, strings.ToLower(s))
				}
				if track.ThreatLevel >= 4 {
					keywords = append(keywords, "hostile", "ballistic")
				}

				a := &DataAsset{
					Type:          AssetTrack,
					Name:          fmt.Sprintf("track-%d", track.TrackNumber),
					Title:         fmt.Sprintf("Track #%d", track.TrackNumber),
					Description:   fmt.Sprintf("Fused track from %d sources, threat level %d", len(track.Sources), track.ThreatLevel),
					Format:        "application/json",
					MimeType:      "application/json",
					SizeBytes:     int64(len(msg.Value)),
					Classification: "UNCLASSIFIED",
					DataSource:    "sensor-fusion",
					Keywords:      keywords,
					CRS:           "EPSG:4326",
					CoverageStart: track.LastUpdate.Add(-5 * time.Minute),
					CoverageEnd:   track.LastUpdate,
					BBoxMinLat:   track.FusedLat - 1,
					BBoxMaxLat:   track.FusedLat + 1,
					BBoxMinLon:   track.FusedLon - 1,
					BBoxMaxLon:   track.FusedLon + 1,
				}
				cs.addAsset(a)
			}
		}
	}
}

type HealthResponse struct {
	Service     string    `json:"service"`
	Version     string    `json:"version"`
	Timestamp   time.Time `json:"timestamp"`
	Status      string    `json:"status"`
	TotalAssets int       `json:"total_assets"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Service:     "data-catalog",
		Version:     "0.1.0",
		Timestamp:   time.Now().UTC(),
		Status:      "healthy",
		TotalAssets: len(cs.assets),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	cs = newCatalogState()
	cs.loadRecordings()

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
	http.HandleFunc("/api/v1/assets", func(w http.ResponseWriter, r *http.Request) {
		q := &CatalogQuery{Limit: 100}
		if t := r.URL.Query().Get("type"); t != "" {
			q.Type = AssetType(t)
		}
		if k := r.URL.Query().Get("keywords"); k != "" {
			q.Keywords = strings.Split(k, ",")
		}
		if c := r.URL.Query().Get("classification"); c != "" {
			q.Classification = c
		}
		if l := r.URL.Query().Get("limit"); l != "" {
			q.Limit, _ = strconv.Atoi(l)
		}
		results := cs.query(q)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	})

	http.HandleFunc("/api/v1/assets/", func(w http.ResponseWriter, r *http.Request) {
		id := filepath.Base(r.URL.Path)
		a, ok := cs.assets[id]
		if !ok {
			http.Error(w, "not found", 404)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(a)
	})

	http.HandleFunc("/api/v1/recordings/", func(w http.ResponseWriter, r *http.Request) {
		id := filepath.Base(r.URL.Path)
		path := filepath.Join(RecordingsDir, id)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			http.Error(w, "not found", 404)
			return
		}
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", id))
		w.Header().Set("Content-Type", "application/octet-stream")
		http.ServeFile(w, r, path)
	})

	http.HandleFunc("/csw", handleGetRecords)
	http.HandleFunc("/csw/", handleGetRecords)

	http.HandleFunc("/csw-endpoint", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("request") == "GetCapabilities" || r.URL.Query().Get("service") == "CSW" {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<Capabilities version="2.0.2" xmlns="http://www.opengis.net/cat/csw/2.0.2">
<ows:Sections xmlns:ows="http://www.opengis.net/ows">
<ows:Section>ServiceIdentification</ows:Section>
<ows:Section>ServiceProvider</ows:Section>
<ows:Section>OperationsMetadata</ows:Section>
<ows:Section>Filter_Capabilities</ows:Section>
</ows:Sections>
</Capabilities>`))
			return
		}
		handleGetRecords(w, r)
	})

	log.Printf("data-catalog starting")
	log.Printf("Kafka broker: %s", kafkaBroker)
	log.Printf("OGC CSW endpoint: /csw")
	log.Printf("REST API: /api/v1/assets")
	go run(ctx)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("HTTP server on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
