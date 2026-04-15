package radar

import (
	"context"
	"testing"
	"time"
)

// TestSBXFeedCreation tests SBX feed creation
func TestSBXFeedCreation(t *testing.T) {
	config := DefaultConfig()
	config.Endpoints = []string{"localhost"}

	feed, err := NewSBXFeed(config)
	if err != nil {
		t.Fatalf("Failed to create SBX feed: %v", err)
	}

	if feed == nil {
		t.Fatal("Feed should not be nil")
	}

	if feed.IsConnected() {
		t.Error("Feed should not be connected initially")
	}

	if feed.config.RadarType != "SBX" {
		t.Errorf("Expected radar type SBX, got %s", feed.config.RadarType)
	}
}

// TestUEWRFeedCreation tests UEWR feed creation
func TestUEWRFeedCreation(t *testing.T) {
	config := DefaultConfig()
	config.Endpoints = []string{"localhost"}

	feed, err := NewUEWRFeed(config)
	if err != nil {
		t.Fatalf("Failed to create UEWR feed: %v", err)
	}

	if feed == nil {
		t.Fatal("Feed should not be nil")
	}

	if feed.IsConnected() {
		t.Error("Feed should not be connected initially")
	}

	if feed.config.RadarType != "UEWR" {
		t.Errorf("Expected radar type UEWR, got %s", feed.config.RadarType)
	}

	if feed.config.FrequencyBand != "L" {
		t.Errorf("Expected frequency band L, got %s", feed.config.FrequencyBand)
	}
}

// TestSBXStatusParse tests SBX status parsing
func TestSBXStatusParse(t *testing.T) {
	feed := &SBXFeed{}

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

// TestUEWRStatusParse tests UEWR status parsing
func TestUEWRStatusParse(t *testing.T) {
	feed := &UEWRFeed{}

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

// TestSBXTargetTypeParse tests SBX target type parsing
func TestSBXTargetTypeParse(t *testing.T) {
	feed := &SBXFeed{}

	tests := []struct {
		targetType uint32
		expected   string
	}{
		{1, TargetTypeAircraft},
		{2, TargetTypeMissile},
		{3, TargetTypeUAV},
		{0, TargetTypeUnknown},
		{99, TargetTypeUnknown},
	}

	for _, tt := range tests {
		result := feed.parseTargetType(tt.targetType)
		if result != tt.expected {
			t.Errorf("parseTargetType(%d) = %s, expected %s", tt.targetType, result, tt.expected)
		}
	}
}

// TestSBXTrackValidation tests SBX track validation
func TestSBXTrackValidation(t *testing.T) {
	config := DefaultConfig()
	config.EnableFiltering = true
	feed := &SBXFeed{config: config}

	tests := []struct {
		name    string
		track   *RadarTrack
		wantErr bool
	}{
		{
			name: "valid track",
			track: &RadarTrack{
				Latitude:  38.8977,
				Longitude: -77.0365,
				Altitude:  10000.0,
				SNR:       20.0,
			},
			wantErr: false,
		},
		{
			name: "invalid latitude",
			track: &RadarTrack{
				Latitude:  100.0,
				Longitude: -77.0365,
				Altitude:  10000.0,
				SNR:       20.0,
			},
			wantErr: true,
		},
		{
			name: "low SNR",
			track: &RadarTrack{
				Latitude:  38.8977,
				Longitude: -77.0365,
				Altitude:  10000.0,
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

// TestUEWRTrackValidation tests UEWR track validation
func TestUEWRTrackValidation(t *testing.T) {
	config := DefaultConfig()
	config.MaxRange = 5000000.0
	config.EnableFiltering = true
	feed := &UEWRFeed{config: config}

	tests := []struct {
		name    string
		track   *RadarTrack
		wantErr bool
	}{
		{
			name: "valid track",
			track: &RadarTrack{
				Latitude:  38.8977,
				Longitude: -77.0365,
				Altitude:  10000.0,
				Range:     1000000.0,
			},
			wantErr: false,
		},
		{
			name: "range exceeds max",
			track: &RadarTrack{
				Latitude:  38.8977,
				Longitude: -77.0365,
				Altitude:  10000.0,
				Range:     6000000.0, // Exceeds max range
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

// TestSBXMockFeed tests SBX mock feed
func TestSBXMockFeed(t *testing.T) {
	feed := NewMockRadarFeed()

	ctx := context.Background()
	if err := feed.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	if !feed.IsConnected() {
		t.Error("Feed should be connected")
	}

	track := RadarTrack{
		ID:          "SBX-1001-123",
		TrackNumber: 1001,
		SensorID:    "SBX-1",
		Timestamp:   time.Now(),
		Latitude:    38.8977,
		Longitude:   -77.0365,
		Altitude:    10000.0,
	}

	feed.Send(track)

	select {
	case r := <-feed.Receive():
		if r.ID != "SBX-1001-123" {
			t.Errorf("Expected ID SBX-1001-123, got %s", r.ID)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for track")
	}

	if err := feed.Disconnect(); err != nil {
		t.Fatalf("Failed to disconnect: %v", err)
	}
}

// TestUEWRMockFeed tests UEWR mock feed
func TestUEWRMockFeed(t *testing.T) {
	feed := NewMockRadarFeed()

	ctx := context.Background()
	if err := feed.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	track := RadarTrack{
		ID:          "UEWR-2001-456",
		TrackNumber: 2001,
		SensorID:    "UEWR-1",
		Timestamp:   time.Now(),
		Latitude:    45.0,
		Longitude:   -120.0,
		Altitude:    50000.0,
	}

	feed.Send(track)

	select {
	case r := <-feed.Receive():
		if r.ID != "UEWR-2001-456" {
			t.Errorf("Expected ID UEWR-2001-456, got %s", r.ID)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for track")
	}

	if err := feed.Disconnect(); err != nil {
		t.Fatalf("Failed to disconnect: %v", err)
	}
}

// TestRadarConfigDefaults tests radar configuration defaults
func TestRadarConfigDefaults(t *testing.T) {
	// Test SBX defaults
	configSBX := DefaultConfig()
	configSBX.Endpoints = []string{"localhost"}
	sbx, err := NewSBXFeed(configSBX)
	if err != nil {
		t.Fatalf("Failed to create SBX feed: %v", err)
	}
	if sbx.config.RadarType != "SBX" {
		t.Errorf("Expected SBX radar type, got %s", sbx.config.RadarType)
	}
	if sbx.config.FrequencyBand != "X" {
		t.Errorf("Expected X band, got %s", sbx.config.FrequencyBand)
	}

	// Test UEWR defaults (separate config)
	configUEWR := DefaultConfig()
	configUEWR.Endpoints = []string{"localhost"}
	uewr, err := NewUEWRFeed(configUEWR)
	if err != nil {
		t.Fatalf("Failed to create UEWR feed: %v", err)
	}
	if uewr.config.RadarType != "UEWR" {
		t.Errorf("Expected UEWR radar type, got %s", uewr.config.RadarType)
	}
	if uewr.config.FrequencyBand != "L" {
		t.Errorf("Expected L band, got %s", uewr.config.FrequencyBand)
	}
	if uewr.config.MaxRange != 5000000.0 {
		t.Errorf("Expected max range 5000000, got %f", uewr.config.MaxRange)
	}
}

// TestSBXTrackCache tests SBX track caching
func TestSBXTrackCache(t *testing.T) {
	feed := &SBXFeed{
		config:     DefaultConfig(),
		trackCache: make(map[uint32]*RadarTrack),
	}

	track1 := &RadarTrack{TrackNumber: 1001, SensorID: "SBX-1"}
	track2 := &RadarTrack{TrackNumber: 1002, SensorID: "SBX-1"}

	feed.trackCache[1001] = track1
	feed.trackCache[1002] = track2

	retrieved := feed.GetTrack(1001)
	if retrieved == nil || retrieved.TrackNumber != 1001 {
		t.Error("Failed to get track from cache")
	}

	active := feed.GetActiveTracks()
	if len(active) != 2 {
		t.Errorf("Expected 2 active tracks, got %d", len(active))
	}
}

// TestUEWRTrackCache tests UEWR track caching
func TestUEWRTrackCache(t *testing.T) {
	feed := &UEWRFeed{
		config:     DefaultConfig(),
		trackCache: make(map[uint32]*RadarTrack),
	}

	track := &RadarTrack{TrackNumber: 3001, SensorID: "UEWR-1"}
	feed.trackCache[3001] = track

	retrieved := feed.GetTrack(3001)
	if retrieved == nil || retrieved.TrackNumber != 3001 {
		t.Error("Failed to get track from cache")
	}
}

// BenchmarkSBXTrackParsing benchmarks SBX track parsing
func BenchmarkSBXTrackParsing(b *testing.B) {
	// Simplified benchmark - actual implementation would use real data
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = &RadarTrack{
			ID:          "SBX-1001",
			TrackNumber: 1001,
			SensorID:    "SBX-1",
			Timestamp:   time.Now(),
			Latitude:    38.8977,
			Longitude:   -77.0365,
			Altitude:    10000.0,
		}
	}
}

// BenchmarkUEWRTrackParsing benchmarks UEWR track parsing
func BenchmarkUEWRTrackParsing(b *testing.B) {
	// Simplified benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = &RadarTrack{
			ID:          "UEWR-2001",
			TrackNumber: 2001,
			SensorID:    "UEWR-1",
			Timestamp:   time.Now(),
			Latitude:    45.0,
			Longitude:   -120.0,
			Altitude:    50000.0,
		}
	}
}
