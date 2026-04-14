package c2bmc

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// TestTrackCorrelator tests track correlation
func TestTrackCorrelator(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		resp := TrackCorrelationResponse{
			PrimaryTrack:    "T-001",
			CorrelatedTracks: []string{"T-002"},
			Confidence:      0.95,
			Status:          AlertStatusComplete,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	server, url, err := mockServer(handler)
	if err != nil {
		t.Fatalf("Failed to create mock server: %v", err)
	}
	defer server.Close()

	config := &C2BMCConfig{
		Endpoint: url,
		Timeout:  5 * time.Second,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	correlator := NewTrackCorrelator(client, 0.7)

	track1 := &TrackData{
		TrackNumber: "T-001",
		Position: Position{
			Latitude:  45.0,
			Longitude: -120.0,
			Altitude:  10000.0,
		},
		Velocity: Velocity{
			Vx: 100.0,
			Vy: 50.0,
			Vz: 10.0,
		},
		LastUpdate: time.Now(),
	}

	track2 := &TrackData{
		TrackNumber: "T-002",
		Position: Position{
			Latitude:  45.001,
			Longitude: -120.001,
			Altitude:  10000.0,
		},
		Velocity: Velocity{
			Vx: 100.0,
			Vy: 50.0,
			Vz: 10.0,
		},
		LastUpdate: time.Now(),
	}

	ctx := context.Background()
	result, err := correlator.CorrelateByPosition(ctx, track1, track2)
	if err != nil {
		t.Fatalf("CorrelateByPosition() error = %v", err)
	}

	if result.PrimaryTrack != "T-001" {
		t.Errorf("PrimaryTrack = %s, want T-001", result.PrimaryTrack)
	}

	if result.SecondaryTrack != "T-002" {
		t.Errorf("SecondaryTrack = %s, want T-002", result.SecondaryTrack)
	}

	// Close tracks should correlate
	if !result.IsCorrelated {
		t.Errorf("Expected correlated, got %s", result.Reason)
	}
}

// TestTrackCorrelatorDistant tests correlation with distant tracks
func TestTrackCorrelatorDistant(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	server, url, err := mockServer(handler)
	if err != nil {
		t.Fatalf("Failed to create mock server: %v", err)
	}
	defer server.Close()

	config := &C2BMCConfig{
		Endpoint: url,
		Timeout:  5 * time.Second,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	correlator := NewTrackCorrelator(client, 0.7)

	track1 := &TrackData{
		TrackNumber: "T-001",
		Position: Position{
			Latitude:  45.0,
			Longitude: -120.0,
			Altitude:  10000.0,
		},
		Velocity: Velocity{
			Vx: 100.0,
			Vy: 50.0,
			Vz: 10.0,
		},
		LastUpdate: time.Now(),
	}

	// Distant track
	track2 := &TrackData{
		TrackNumber: "T-002",
		Position: Position{
			Latitude:  46.0, // ~111 km away
			Longitude: -121.0,
			Altitude:  10000.0,
		},
		Velocity: Velocity{
			Vx: 100.0,
			Vy: 50.0,
			Vz: 10.0,
		},
		LastUpdate: time.Now(),
	}

	ctx := context.Background()
	result, err := correlator.CorrelateByPosition(ctx, track1, track2)
	if err != nil {
		t.Fatalf("CorrelateByPosition() error = %v", err)
	}

	// Distant tracks should not correlate
	if result.IsCorrelated {
		t.Errorf("Expected not correlated, got %s", result.Reason)
	}
}

// TestTrackCorrelatorTimeWindow tests correlation with time window
func TestTrackCorrelatorTimeWindow(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	server, url, err := mockServer(handler)
	if err != nil {
		t.Fatalf("Failed to create mock server: %v", err)
	}
	defer server.Close()

	config := &C2BMCConfig{
		Endpoint: url,
		Timeout:  5 * time.Second,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	correlator := NewTrackCorrelator(client, 0.7)

	track1 := &TrackData{
		TrackNumber: "T-001",
		Position: Position{
			Latitude:  45.0,
			Longitude: -120.0,
			Altitude:  10000.0,
		},
		LastUpdate: time.Now(),
	}

	// Track with old timestamp
	track2 := &TrackData{
		TrackNumber: "T-002",
		Position: Position{
			Latitude:  45.0,
			Longitude: -120.0,
			Altitude:  10000.0,
		},
		LastUpdate: time.Now().Add(-60 * time.Second), // 60s ago
	}

	ctx := context.Background()
	result, err := correlator.CorrelateByPosition(ctx, track1, track2)
	if err != nil {
		t.Fatalf("CorrelateByPosition() error = %v", err)
	}

	// Should not correlate due to time difference
	if result.IsCorrelated {
		t.Error("Expected not correlated due to time difference")
	}
}

// TestBatchCorrelate tests batch correlation
func TestBatchCorrelate(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	server, url, err := mockServer(handler)
	if err != nil {
		t.Fatalf("Failed to create mock server: %v", err)
	}
	defer server.Close()

	config := &C2BMCConfig{
		Endpoint: url,
		Timeout:  5 * time.Second,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	correlator := NewTrackCorrelator(client, 0.7)

	tracks := []*TrackData{
		{
			TrackNumber: "T-001",
			Position:    Position{Latitude: 45.0, Longitude: -120.0, Altitude: 10000.0},
			Velocity:    Velocity{Vx: 100.0, Vy: 50.0, Vz: 10.0},
			LastUpdate:  time.Now(),
		},
		{
			TrackNumber: "T-002",
			Position:    Position{Latitude: 45.001, Longitude: -120.001, Altitude: 10000.0},
			Velocity:    Velocity{Vx: 100.0, Vy: 50.0, Vz: 10.0},
			LastUpdate:  time.Now(),
		},
		{
			TrackNumber: "T-003",
			Position:    Position{Latitude: 46.0, Longitude: -121.0, Altitude: 10000.0},
			Velocity:    Velocity{Vx: 100.0, Vy: 50.0, Vz: 10.0},
			LastUpdate:  time.Now(),
		},
	}

	ctx := context.Background()
	results, err := correlator.BatchCorrelate(ctx, tracks)
	if err != nil {
		t.Fatalf("BatchCorrelate() error = %v", err)
	}

	// 3 tracks = 3 pairs (C(3,2) = 3)
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}
}

// TestCorrelateAndSubmit tests correlation submission
func TestCorrelateAndSubmit(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		resp := TrackCorrelationResponse{
			PrimaryTrack:    "T-001",
			CorrelatedTracks: []string{"T-002", "T-003"},
			Confidence:      0.95,
			Status:          AlertStatusComplete,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	server, url, err := mockServer(handler)
	if err != nil {
		t.Fatalf("Failed to create mock server: %v", err)
	}
	defer server.Close()

	config := &C2BMCConfig{
		Endpoint: url,
		Timeout:  5 * time.Second,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	correlator := NewTrackCorrelator(client, 0.7)

	primary := &TrackData{
		TrackNumber: "T-001",
		Position:    Position{Latitude: 45.0, Longitude: -120.0, Altitude: 10000.0},
		LastUpdate:  time.Now(),
	}

	secondaries := []*TrackData{
		{TrackNumber: "T-002"},
		{TrackNumber: "T-003"},
	}

	ctx := context.Background()
	resp, err := correlator.CorrelateAndSubmit(ctx, primary, secondaries)
	if err != nil {
		t.Fatalf("CorrelateAndSubmit() error = %v", err)
	}

	if resp.PrimaryTrack != "T-001" {
		t.Errorf("PrimaryTrack = %s, want T-001", resp.PrimaryTrack)
	}

	if len(resp.CorrelatedTracks) != 2 {
		t.Errorf("Expected 2 correlated tracks, got %d", len(resp.CorrelatedTracks))
	}
}

// TestTrackUpdateHandler tests track update batching
func TestTrackUpdateHandler(t *testing.T) {
	received := make([]*TrackData, 0)

	handler := func(w http.ResponseWriter, r *http.Request) {
		var track TrackData
		json.NewDecoder(r.Body).Decode(&track)
		received = append(received, &track)
		w.WriteHeader(http.StatusOK)
	}

	server, url, err := mockServer(handler)
	if err != nil {
		t.Fatalf("Failed to create mock server: %v", err)
	}
	defer server.Close()

	config := &C2BMCConfig{
		Endpoint: url,
		Timeout:  5 * time.Second,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	updateHandler := NewTrackUpdateHandler(client, 10, 100*time.Millisecond)
	ctx := context.Background()
	updateHandler.Start(ctx)

	// Submit tracks
	for i := 0; i < 5; i++ {
		updateHandler.Submit(&TrackData{
			TrackNumber: string(rune('A' + i)),
			Position:    Position{Latitude: float64(i), Longitude: float64(i), Altitude: 10000.0},
		})
	}

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	if len(received) != 5 {
		t.Errorf("Expected 5 tracks, got %d", len(received))
	}
}

// TestTrackStatusQuery tests track status queries
func TestTrackStatusQuery(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		track := &TrackData{
			TrackNumber: "T-001",
			TrackID:     "TRACK-001",
			Position: Position{
				Latitude:  45.0,
				Longitude: -120.0,
				Altitude:  10000.0,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(track)
	}

	server, url, err := mockServer(handler)
	if err != nil {
		t.Fatalf("Failed to create mock server: %v", err)
	}
	defer server.Close()

	config := &C2BMCConfig{
		Endpoint: url,
		Timeout:  5 * time.Second,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	query := NewTrackStatusQuery(client)

	ctx := context.Background()
	track, err := query.Query(ctx, "TRACK-001")
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	if track.TrackNumber != "T-001" {
		t.Errorf("TrackNumber = %s, want T-001", track.TrackNumber)
	}
}

// BenchmarkTrackCorrelation benchmarks correlation
func BenchmarkTrackCorrelation(b *testing.B) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	server, url, err := mockServer(handler)
	if err != nil {
		b.Fatalf("Failed to create mock server: %v", err)
	}
	defer server.Close()

	config := &C2BMCConfig{
		Endpoint: url,
		Timeout:  5 * time.Second,
	}

	client, err := NewClient(config)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}

	correlator := NewTrackCorrelator(client, 0.7)

	track1 := &TrackData{
		TrackNumber: "T-001",
		Position:    Position{Latitude: 45.0, Longitude: -120.0, Altitude: 10000.0},
		Velocity:    Velocity{Vx: 100.0, Vy: 50.0, Vz: 10.0},
		LastUpdate:  time.Now(),
	}

	track2 := &TrackData{
		TrackNumber: "T-002",
		Position:    Position{Latitude: 45.001, Longitude: -120.001, Altitude: 10000.0},
		Velocity:    Velocity{Vx: 100.0, Vy: 50.0, Vz: 10.0},
		LastUpdate:  time.Now(),
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		correlator.CorrelateByPosition(ctx, track1, track2)
	}
}