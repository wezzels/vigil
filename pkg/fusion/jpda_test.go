package fusion

import (
	"math"
	"testing"
)

// TestMahalanobisDistance tests distance calculation
func TestMahalanobisDistance(t *testing.T) {
	tm := NewTrackManager()

	tests := []struct {
		name      string
		track     *Track
		meas      *Measurement
		wantDist  float64
		tolerance float64
	}{
		{
			name: "same position",
			track: &Track{
				ID:     1,
				Lat:    38.0,
				Lon:    -77.0,
				Alt:    100.0,
				VarLat: 0.001,
				VarLon: 0.001,
				VarAlt: 10.0,
			},
			meas: &Measurement{
				ID:     100,
				Lat:    38.0,
				Lon:    -77.0,
				Alt:    100.0,
				VarLat: 0.001,
				VarLon: 0.001,
				VarAlt: 10.0,
			},
			wantDist:  0.0,
			tolerance: 0.01,
		},
		{
			name: "different position",
			track: &Track{
				ID:     1,
				Lat:    38.0,
				Lon:    -77.0,
				Alt:    100.0,
				VarLat: 0.0001,
				VarLon: 0.0001,
				VarAlt: 1.0,
			},
			meas: &Measurement{
				ID:     100,
				Lat:    38.1, // ~11km different
				Lon:    -77.0,
				Alt:    100.0,
				VarLat: 0.0001,
				VarLon: 0.0001,
				VarAlt: 1.0,
			},
			wantDist:  100.0, // Large distance
			tolerance: 50.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dist := tm.MahalanobisDistance(tt.meas, tt.track)
			if math.Abs(dist-tt.wantDist) > tt.tolerance {
				t.Errorf("Expected distance %.2f, got %.2f", tt.wantDist, dist)
			}
		})
	}
}

// TestJPDAAssociate tests association logic
func TestJPDAAssociate(t *testing.T) {
	tm := NewTrackManager()
	tm.GatingLimit = 100.0 // Loose gate for test

	// Add existing tracks
	track1 := &Track{
		ID:     1,
		Lat:    38.0,
		Lon:    -77.0,
		Alt:    100.0,
		VarLat: 0.0001,
		VarLon: 0.0001,
		VarAlt: 1.0,
	}
	track2 := &Track{
		ID:     2,
		Lat:    39.0,
		Lon:    -78.0,
		Alt:    200.0,
		VarLat: 0.0001,
		VarLon: 0.0001,
		VarAlt: 1.0,
	}
	tm.Tracks[1] = track1
	tm.Tracks[2] = track2

	// Measurements close to track 1
	measurements := []*Measurement{
		{ID: 101, Lat: 38.0, Lon: -77.0, Alt: 100.0, VarLat: 0.0001, VarLon: 0.0001, VarAlt: 1.0},
		{ID: 102, Lat: 38.001, Lon: -77.001, Alt: 101.0, VarLat: 0.0001, VarLon: 0.0001, VarAlt: 1.0},
	}

	associations := tm.JPDAAssociate(measurements)

	// Should associate both measurements with track 1
	if len(associations) < 2 {
		t.Errorf("Expected at least 2 associations, got %d", len(associations))
	}

	// Check that associations are with track 1
	for _, a := range associations {
		if a.TrackID != 1 {
			t.Errorf("Expected association with track 1, got track %d", a.TrackID)
		}
		if a.Probability <= 0 || a.Probability > 1 {
			t.Errorf("Invalid probability: %.4f", a.Probability)
		}
	}
}

// TestSingleTargetSingleMeasurement tests 1-on-1 association
func TestSingleTargetSingleMeasurement(t *testing.T) {
	tm := NewTrackManager()

	track := &Track{
		ID:     1,
		Lat:    0.0,
		Lon:    0.0,
		Alt:    0.0,
		VarLat: 0.01,
		VarLon: 0.01,
		VarAlt: 100.0,
	}
	tm.Tracks[1] = track

	measurements := []*Measurement{
		{ID: 101, Lat: 0.0, Lon: 0.0, Alt: 0.0, VarLat: 0.01, VarLon: 0.01, VarAlt: 100.0},
	}

	associations := tm.JPDAAssociate(measurements)

	if len(associations) != 1 {
		t.Errorf("Expected 1 association, got %d", len(associations))
	}

	if associations[0].Probability < 0.9 {
		t.Errorf("Expected high probability for exact match, got %.4f", associations[0].Probability)
	}
}

// TestSingleTargetMultiMeasurement tests 1-on-N association
func TestSingleTargetMultiMeasurement(t *testing.T) {
	tm := NewTrackManager()
	tm.GatingLimit = 50.0

	track := &Track{
		ID:     1,
		Lat:    0.0,
		Lon:    0.0,
		Alt:    0.0,
		VarLat: 0.01,
		VarLon: 0.01,
		VarAlt: 100.0,
	}
	tm.Tracks[1] = track

	// Multiple measurements near the track
	measurements := []*Measurement{
		{ID: 101, Lat: 0.001, Lon: 0.0, Alt: 0.0, VarLat: 0.01, VarLon: 0.01, VarAlt: 100.0},
		{ID: 102, Lat: 0.0, Lon: 0.001, Alt: 0.0, VarLat: 0.01, VarLon: 0.01, VarAlt: 100.0},
		{ID: 103, Lat: 0.001, Lon: 0.001, Alt: 0.0, VarLat: 0.01, VarLon: 0.01, VarAlt: 100.0},
	}

	associations := tm.JPDAAssociate(measurements)

	// All should associate with track 1
	if len(associations) != 3 {
		t.Errorf("Expected 3 associations, got %d", len(associations))
	}
}

// TestMultiTargetSingleMeasurement tests N-on-1 association
func TestMultiTargetSingleMeasurement(t *testing.T) {
	tm := NewTrackManager()

	// Two tracks close together
	track1 := &Track{
		ID:     1,
		Lat:    0.0,
		Lon:    0.0,
		Alt:    0.0,
		VarLat: 0.01,
		VarLon: 0.01,
		VarAlt: 100.0,
	}
	track2 := &Track{
		ID:     2,
		Lat:    0.001,
		Lon:    0.001,
		Alt:    0.0,
		VarLat: 0.01,
		VarLon: 0.01,
		VarAlt: 100.0,
	}
	tm.Tracks[1] = track1
	tm.Tracks[2] = track2

	// Measurement between them
	measurements := []*Measurement{
		{ID: 101, Lat: 0.0005, Lon: 0.0005, Alt: 0.0, VarLat: 0.01, VarLon: 0.01, VarAlt: 100.0},
	}

	associations := tm.JPDAAssociate(measurements)

	// Should associate with both tracks (ambiguous)
	if len(associations) != 2 {
		t.Errorf("Expected 2 associations (ambiguous), got %d", len(associations))
	}
}

// TestMultiTargetMultiMeasurement tests N-on-N association
func TestMultiTargetMultiMeasurement(t *testing.T) {
	tm := NewTrackManager()
	tm.GatingLimit = 20.0

	track1 := &Track{
		ID:     1,
		Lat:    0.0,
		Lon:    0.0,
		Alt:    0.0,
		VarLat: 0.0001,
		VarLon: 0.0001,
		VarAlt: 10.0,
	}
	track2 := &Track{
		ID:     2,
		Lat:    1.0, // Well separated
		Lon:    1.0,
		Alt:    0.0,
		VarLat: 0.0001,
		VarLon: 0.0001,
		VarAlt: 10.0,
	}
	tm.Tracks[1] = track1
	tm.Tracks[2] = track2

	measurements := []*Measurement{
		{ID: 101, Lat: 0.0, Lon: 0.0, Alt: 0.0, VarLat: 0.0001, VarLon: 0.0001, VarAlt: 10.0}, // Near track 1
		{ID: 102, Lat: 1.0, Lon: 1.0, Alt: 0.0, VarLat: 0.0001, VarLon: 0.0001, VarAlt: 10.0}, // Near track 2
	}

	associations := tm.JPDAAssociate(measurements)

	// Each measurement should associate with its nearest track
	if len(associations) != 2 {
		t.Errorf("Expected 2 associations, got %d", len(associations))
	}
}

// TestClutterRejection tests rejection of far measurements
func TestClutterRejection(t *testing.T) {
	tm := NewTrackManager()
	tm.GatingLimit = 5.0 // Tight gate

	track := &Track{
		ID:     1,
		Lat:    0.0,
		Lon:    0.0,
		Alt:    0.0,
		VarLat: 0.0001,
		VarLon: 0.0001,
		VarAlt: 1.0,
	}
	tm.Tracks[1] = track

	// Far measurement (clutter)
	measurements := []*Measurement{
		{ID: 101, Lat: 10.0, Lon: 10.0, Alt: 0.0, VarLat: 0.0001, VarLon: 0.0001, VarAlt: 1.0},
	}

	associations := tm.JPDAAssociate(measurements)

	// Should not associate (outside gate)
	if len(associations) != 0 {
		t.Errorf("Expected 0 associations for clutter, got %d", len(associations))
	}
}

// TestTrackInitiation tests new track creation
func TestTrackInitiation(t *testing.T) {
	tm := NewTrackManager()

	measurements := []*Measurement{
		{ID: 101, Lat: 38.0, Lon: -77.0, Alt: 100.0, VarLat: 0.001, VarLon: 0.001, VarAlt: 10.0, Timestamp: 1000},
	}

	tm.Update(measurements, 1000)

	// Should create new track
	if len(tm.Tracks) != 1 {
		t.Errorf("Expected 1 track, got %d", len(tm.Tracks))
	}

	track := tm.Tracks[101]
	if track == nil {
		t.Fatal("Track not found")
	}

	if track.Lat != 38.0 {
		t.Errorf("Expected lat 38.0, got %.4f", track.Lat)
	}
}

// TestTrackUpdate tests track updating
func TestTrackUpdate(t *testing.T) {
	tm := NewTrackManager()

	// Initial track
	track := &Track{
		ID:         1,
		Lat:        38.0,
		Lon:        -77.0,
		Alt:        100.0,
		VarLat:     0.001,
		VarLon:     0.001,
		VarAlt:     10.0,
		LastUpdate: 1000,
	}
	tm.Tracks[1] = track

	// New measurement close to track
	measurements := []*Measurement{
		{ID: 101, Lat: 38.001, Lon: -77.001, Alt: 101.0, VarLat: 0.001, VarLon: 0.001, VarAlt: 10.0, Timestamp: 2000},
	}

	tm.Update(measurements, 2000)

	// Track should be updated (check lat changed from initial)
	if track.Lat == 38.0 {
		t.Errorf("Track was not updated")
	}

	// SourceCount incremented on update
	if track.SourceCount < 1 {
		t.Errorf("Expected source count >= 1, got %d", track.SourceCount)
	}
}

// TestTrackDeletion tests old track removal
func TestTrackDeletion(t *testing.T) {
	tm := NewTrackManager()
	tm.MaxAge = 1000 // 1 second

	track := &Track{
		ID:         1,
		Lat:        38.0,
		Lon:        -77.0,
		Alt:        100.0,
		VarLat:     0.001,
		VarLon:     0.001,
		VarAlt:     10.0,
		LastUpdate: 1000,
	}
	tm.Tracks[1] = track

	// Update with old timestamp (should remove track)
	tm.Update(nil, 5000)

	if len(tm.Tracks) != 0 {
		t.Errorf("Expected 0 tracks after cleanup, got %d", len(tm.Tracks))
	}
}

// BenchmarkJPDAAssociate benchmarks association performance
func BenchmarkJPDAAssociate(b *testing.B) {
	tm := NewTrackManager()

	// Create 100 tracks
	for i := 0; i < 100; i++ {
		track := &Track{
			ID:     uint64(i),
			Lat:    float64(i) * 0.1,
			Lon:    float64(i) * 0.1,
			Alt:    100.0,
			VarLat: 0.001,
			VarLon: 0.001,
			VarAlt: 10.0,
		}
		tm.Tracks[track.ID] = track
	}

	// Create 100 measurements
	measurements := make([]*Measurement, 100)
	for i := 0; i < 100; i++ {
		measurements[i] = &Measurement{
			ID:     uint64(i + 1000),
			Lat:    float64(i) * 0.1,
			Lon:    float64(i) * 0.1,
			Alt:    100.0,
			VarLat: 0.001,
			VarLon: 0.001,
			VarAlt: 10.0,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tm.JPDAAssociate(measurements)
	}
}

// BenchmarkMahalanobisDistance benchmarks distance calculation
func BenchmarkMahalanobisDistance(b *testing.B) {
	tm := NewTrackManager()

	track := &Track{
		ID:     1,
		Lat:    38.0,
		Lon:    -77.0,
		Alt:    100.0,
		VarLat: 0.001,
		VarLon: 0.001,
		VarAlt: 10.0,
	}

	meas := &Measurement{
		ID:     100,
		Lat:    38.001,
		Lon:    -77.001,
		Alt:    101.0,
		VarLat: 0.001,
		VarLon: 0.001,
		VarAlt: 10.0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tm.MahalanobisDistance(meas, track)
	}
}
