package radar

import (
	"context"
	"encoding/binary"
	"testing"
	"time"
)

// TestRadarTrack tests track structure
func TestRadarTrack(t *testing.T) {
	track := RadarTrack{
		ID:          "TPY2-1001-1234567890",
		TrackNumber: 1001,
		SensorID:    "TPY2-1",
		Timestamp:   time.Now(),
		Latitude:    38.8977,
		Longitude:   -77.0365,
		Altitude:    10000.0,
		VelocityN:   300.0,
		VelocityE:   0.0,
		VelocityU:   0.0,
		TrackQuality: 5,
		TrackStatus: TrackStatusTrack,
	}
	
	if track.ID != "TPY2-1001-1234567890" {
		t.Errorf("Expected ID TPY2-1001-1234567890, got %s", track.ID)
	}
	if track.TrackNumber != 1001 {
		t.Errorf("Expected TrackNumber 1001, got %d", track.TrackNumber)
	}
}

// TestRadarConfig tests configuration
func TestRadarConfig(t *testing.T) {
	config := DefaultConfig()
	
	if config.Port != 5001 {
		t.Errorf("Expected default port 5001, got %d", config.Port)
	}
	if config.MinSNR != 10.0 {
		t.Errorf("Expected min SNR 10.0, got %f", config.MinSNR)
	}
	if config.ConnectTimeout != 30*time.Second {
		t.Errorf("Expected connect timeout 30s, got %v", config.ConnectTimeout)
	}
}

// TestConfigValidation tests configuration validation
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *RadarConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  &RadarConfig{Endpoints: []string{"localhost"}, ConnectTimeout: time.Second, ReadTimeout: time.Second},
			wantErr: false,
		},
		{
			name:    "no endpoints",
			config:  &RadarConfig{ConnectTimeout: time.Second, ReadTimeout: time.Second},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestTPY2FeedCreation tests TPY2 feed creation
func TestTPY2FeedCreation(t *testing.T) {
	config := DefaultConfig()
	config.Endpoints = []string{"localhost"}
	
	feed, err := NewTPY2Feed(config)
	if err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}
	
	if feed == nil {
		t.Fatal("Feed should not be nil")
	}
	
	if feed.IsConnected() {
		t.Error("Feed should not be connected initially")
	}
}

// TestTrackStatusParse tests status parsing
func TestTrackStatusParse(t *testing.T) {
	feed := &TPY2Feed{}
	
	tests := []struct {
		status   uint32
		expected string
	}{
		{0, TrackStatusInit},
		{1, TrackStatusTrack},
		{2, TrackStatusCoast},
		{3, TrackStatusDrop},
		{99, TrackStatusUnknown},
	}
	
	for _, tt := range tests {
		result := feed.parseStatus(tt.status)
		if result != tt.expected {
			t.Errorf("parseStatus(%d) = %s, expected %s", tt.status, result, tt.expected)
		}
	}
}

// TestHeadingCalculation tests heading calculation
func TestHeadingCalculation(t *testing.T) {
	feed := &TPY2Feed{}
	
	tests := []struct {
		velE     float64
		velN     float64
		minBound float64
		maxBound float64
	}{
		{300.0, 0.0, 80, 100},    // East (~90°)
		{0.0, 300.0, -10, 10},   // North (0°)
		{-300.0, 0.0, 260, 280}, // West (~270°)
		{0.0, -300.0, 170, 190}, // South (180°)
	}
	
	for _, tt := range tests {
		heading := feed.calculateHeading(tt.velE, tt.velN)
		// Normalize heading for comparison
		for heading < 0 {
			heading += 360
		}
		for heading >= 360 {
			heading -= 360
		}
		// For North case (0°), accept small range
		if tt.velN > 0 && tt.velE == 0 {
			if heading < tt.minBound || heading > tt.maxBound {
				t.Errorf("calculateHeading(%.1f, %.1f) = %.1f, expected near 0°",
					tt.velE, tt.velN, heading)
			}
		} else if heading < tt.minBound || heading > tt.maxBound {
			t.Errorf("calculateHeading(%.1f, %.1f) = %.1f, expected between %.1f and %.1f",
				tt.velE, tt.velN, heading, tt.minBound, tt.maxBound)
		}
	}
}

// TestTrackValidation tests track validation
func TestTrackValidation(t *testing.T) {
	config := DefaultConfig()
	config.EnableFiltering = true
	feed := &TPY2Feed{config: config}
	
	tests := []struct {
		name    string
		track   *RadarTrack
		wantErr bool
	}{
		{
			name: "valid track",
			track: &RadarTrack{
				Latitude:   38.8977,
				Longitude:  -77.0365,
				Altitude:   10000.0,
				SNR:       20.0,
				RangeRate: 300.0,
			},
			wantErr: false,
		},
		{
			name: "invalid latitude",
			track: &RadarTrack{
				Latitude:   100.0,
				Longitude:  -77.0365,
				Altitude:   10000.0,
				SNR:       20.0,
			},
			wantErr: true,
		},
		{
			name: "invalid longitude",
			track: &RadarTrack{
				Latitude:   38.8977,
				Longitude:  200.0,
				Altitude:   10000.0,
				SNR:       20.0,
			},
			wantErr: true,
		},
		{
			name: "low SNR",
			track: &RadarTrack{
				Latitude:   38.8977,
				Longitude:  -77.0365,
				Altitude:   10000.0,
				SNR:       5.0,
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := feed.validateTrack(tt.track)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTrack() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestTrackParsing tests track parsing
func TestTrackParsing(t *testing.T) {
	feed := &TPY2Feed{config: DefaultConfig(), trackCache: make(map[uint32]*RadarTrack)}
	
	// Create test track data
	data := make([]byte, 116)
	binary.BigEndian.PutUint32(data[0:4], 1001)                    // Track number
	binary.BigEndian.PutUint64(data[4:12], uint64(time.Now().UnixNano())) // Timestamp
	binary.BigEndian.PutUint64(data[12:20], 0x404b2c6e5d1a0000)  // Latitude (~38.9)
	binary.BigEndian.PutUint64(data[20:28], 0xc053b2c6e5d1a000)  // Longitude (~-77.0)
	binary.BigEndian.PutUint64(data[28:36], 0x40c3880000000000)  // Altitude (10000)
	binary.BigEndian.PutUint64(data[36:44], 0x4072c00000000000)  // Velocity North (300)
	binary.BigEndian.PutUint64(data[44:52], 0)                    // Velocity East
	binary.BigEndian.PutUint64(data[52:60], 0)                    // Velocity Up
	binary.BigEndian.PutUint32(data[60:64], 5)                    // Track quality
	binary.BigEndian.PutUint32(data[64:68], 1)                    // Track status
	binary.BigEndian.PutUint64(data[68:76], 0x40e35fa000000000)  // Range (50000)
	binary.BigEndian.PutUint64(data[76:84], 0x4072c00000000000)  // Range rate (300)
	binary.BigEndian.PutUint64(data[84:92], 0x4026800000000000)  // Azimuth (45)
	binary.BigEndian.PutUint64(data[92:100], 0x3ff0000000000000)  // Elevation (10)
	binary.BigEndian.PutUint64(data[100:108], 0xc031200000000000) // RCS (-20)
	binary.BigEndian.PutUint64(data[108:116], 0x4034000000000000) // SNR (20)
	
	track, err := feed.parseTrack(1, data)
	if err != nil {
		t.Fatalf("Failed to parse track: %v", err)
	}
	
	if track.TrackNumber != 1001 {
		t.Errorf("Expected TrackNumber 1001, got %d", track.TrackNumber)
	}
	if track.TrackQuality != 5 {
		t.Errorf("Expected TrackQuality 5, got %d", track.TrackQuality)
	}
	if track.TrackStatus != TrackStatusTrack {
		t.Errorf("Expected TrackStatus %s, got %s", TrackStatusTrack, track.TrackStatus)
	}
}

// TestTrackCache tests track caching
func TestTrackCache(t *testing.T) {
	feed := &TPY2Feed{
		config:     DefaultConfig(),
		trackCache: make(map[uint32]*RadarTrack),
	}
	
	track1 := &RadarTrack{
		TrackNumber: 1001,
		SensorID:   "TPY2-1",
		Timestamp: time.Now(),
	}
	
	track2 := &RadarTrack{
		TrackNumber: 1002,
		SensorID:   "TPY2-1",
		Timestamp: time.Now(),
	}
	
	feed.trackCache[1001] = track1
	feed.trackCache[1002] = track2
	
	// Test GetTrack
	retrieved := feed.GetTrack(1001)
	if retrieved == nil || retrieved.TrackNumber != 1001 {
		t.Error("Failed to get track from cache")
	}
	
	// Test GetActiveTracks
	active := feed.GetActiveTracks()
	if len(active) != 2 {
		t.Errorf("Expected 2 active tracks, got %d", len(active))
	}
}

// TestMockFeed tests mock feed for testing
type MockRadarFeed struct {
	tracks    chan RadarTrack
	errors    chan error
	connected bool
	stats     FeedStats
}

func NewMockRadarFeed() *MockRadarFeed {
	return &MockRadarFeed{
		tracks: make(chan RadarTrack, 100),
		errors: make(chan error, 10),
	}
}

func (m *MockRadarFeed) Connect(ctx context.Context) error {
	m.connected = true
	m.stats.Connected = true
	return nil
}

func (m *MockRadarFeed) Disconnect() error {
	m.connected = false
	m.stats.Connected = false
	return nil
}

func (m *MockRadarFeed) Receive() <-chan RadarTrack {
	return m.tracks
}

func (m *MockRadarFeed) Errors() <-chan error {
	return m.errors
}

func (m *MockRadarFeed) IsConnected() bool {
	return m.connected
}

func (m *MockRadarFeed) Stats() FeedStats {
	return m.stats
}

func (m *MockRadarFeed) Send(track RadarTrack) {
	m.tracks <- track
	m.stats.TotalReceived++
}

// TestMockFeedUsage tests using mock feed
func TestMockFeedUsage(t *testing.T) {
	feed := NewMockRadarFeed()
	
	ctx := context.Background()
	if err := feed.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	
	if !feed.IsConnected() {
		t.Error("Feed should be connected")
	}
	
	// Send a track
	track := RadarTrack{
		ID:          "TEST-001",
		TrackNumber: 1001,
		SensorID:    "MOCK-1",
		Timestamp:   time.Now(),
		Latitude:    38.8977,
		Longitude:   -77.0365,
		Altitude:    10000.0,
	}
	
	feed.Send(track)
	
	// Receive track
	select {
	case r := <-feed.Receive():
		if r.ID != "TEST-001" {
			t.Errorf("Expected ID TEST-001, got %s", r.ID)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for track")
	}
	
	if err := feed.Disconnect(); err != nil {
		t.Fatalf("Failed to disconnect: %v", err)
	}
}

// TestRadarError tests error types
func TestRadarError(t *testing.T) {
	err := NewConnectionError("test error", true)
	
	if err.Code != ErrCodeConnection {
		t.Errorf("Expected code %s, got %s", ErrCodeConnection, err.Code)
	}
	if !err.Retryable {
		t.Error("Connection error should be retryable")
	}
	
	trackErr := NewTrackLostError("TRACK-001")
	if trackErr.Code != ErrCodeTrackLost {
		t.Errorf("Expected code %s, got %s", ErrCodeTrackLost, trackErr.Code)
	}
}

// TestFeedStats tests feed statistics
func TestFeedStats(t *testing.T) {
	stats := FeedStats{
		Connected:      true,
		TotalReceived:  1000,
		TotalErrors:    5,
		ReceiveRate:    100.5,
		TracksActive:   50,
		TracksDropped:  10,
	}
	
	if !stats.Connected {
		t.Error("Stats should show connected")
	}
	if stats.TotalReceived != 1000 {
		t.Errorf("Expected 1000 received, got %d", stats.TotalReceived)
	}
	if stats.TracksActive != 50 {
		t.Errorf("Expected 50 active tracks, got %d", stats.TracksActive)
	}
}

// BenchmarkTrackParsing benchmarks track parsing
func BenchmarkTrackParsing(b *testing.B) {
	feed := &TPY2Feed{
		config:     DefaultConfig(),
		trackCache: make(map[uint32]*RadarTrack),
	}
	
	data := make([]byte, 116)
	binary.BigEndian.PutUint32(data[0:4], 1001)
	binary.BigEndian.PutUint64(data[4:12], uint64(time.Now().UnixNano()))
	binary.BigEndian.PutUint64(data[12:20], 0x404b2c6e5d1a0000)
	binary.BigEndian.PutUint64(data[20:28], 0xc053b2c6e5d1a000)
	binary.BigEndian.PutUint64(data[28:36], 0x40c3880000000000)
	binary.BigEndian.PutUint32(data[60:64], 5)
	binary.BigEndian.PutUint32(data[64:68], 1)
	binary.BigEndian.PutUint64(data[68:76], 0x40e35fa000000000)
	binary.BigEndian.PutUint64(data[76:84], 0x4072c00000000000)
	binary.BigEndian.PutUint64(data[84:92], 0x4026800000000000)
	binary.BigEndian.PutUint64(data[92:100], 0x3ff0000000000000)
	binary.BigEndian.PutUint64(data[100:108], 0xc031200000000000)
	binary.BigEndian.PutUint64(data[108:116], 0x4034000000000000)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		feed.parseTrack(1, data)
	}
}

// BenchmarkValidation benchmarks track validation
func BenchmarkValidation(b *testing.B) {
	config := DefaultConfig()
	config.EnableFiltering = true
	feed := &TPY2Feed{config: config}
	
	track := &RadarTrack{
		Latitude:   38.8977,
		Longitude:  -77.0365,
		Altitude:   10000.0,
		SNR:       20.0,
		RangeRate:  300.0,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		feed.validateTrack(track)
	}
}