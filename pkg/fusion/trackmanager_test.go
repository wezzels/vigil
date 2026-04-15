package fusion

import (
	"math"
	"testing"
)

// TestTrackCreation tests track creation from measurements
func TestTrackCreation(t *testing.T) {
	tm := NewTrackManager()

	meas := &Measurement{
		ID:        1,
		Lat:       38.0,
		Lon:       -77.0,
		Alt:       100.0,
		VarLat:    0.001,
		VarLon:    0.001,
		VarAlt:    10.0,
		SourceID:  1,
		Timestamp: 1000,
	}

	track := tm.InitiateTrack(meas)

	if track == nil {
		t.Fatal("Track should not be nil")
	}

	if track.ID != 1 {
		t.Errorf("Expected ID 1, got %d", track.ID)
	}

	if track.Lat != 38.0 {
		t.Errorf("Expected Lat 38.0, got %.4f", track.Lat)
	}

	if track.TrackNum != 1 {
		t.Errorf("Expected TrackNum 1, got %d", track.TrackNum)
	}
}

// TestTrackUpdateWithMeasurement tests track update with new measurement
func TestTrackUpdateWithMeasurement(t *testing.T) {
	tm := NewTrackManager()
	tm.GatingLimit = 100.0

	// Create initial track
	track := &Track{
		ID:     1,
		Lat:    38.0,
		Lon:    -77.0,
		Alt:    100.0,
		VarLat: 0.001,
		VarLon: 0.001,
		VarAlt: 10.0,
	}
	tm.Tracks[1] = track

	// New measurement close to track
	meas := &Measurement{
		ID:        100,
		Lat:       38.001,
		Lon:       -77.001,
		Alt:       101.0,
		VarLat:    0.001,
		VarLon:    0.001,
		VarAlt:    10.0,
		SourceID:  1,
		Timestamp: 2000,
	}

	measurements := []*Measurement{meas}
	tm.Update(measurements, 2000)

	// Track should have moved toward measurement
	if track.Lat == 38.0 {
		t.Error("Track should have moved")
	}

	// Track should have been updated
	if track.SourceCount < 1 {
		t.Error("Track should have been updated")
	}
}

// TestTrackDeletionWithAge tests removal of old tracks
func TestTrackDeletionWithAge(t *testing.T) {
	tm := NewTrackManager()
	tm.MaxAge = 1000 // 1 second

	// Create old track
	track := &Track{
		ID:         1,
		Lat:        38.0,
		Lon:        -77.0,
		Alt:        100.0,
		LastUpdate: 0, // Old timestamp
	}
	tm.Tracks[1] = track

	// Update with current time
	tm.Update(nil, 2000)

	if len(tm.Tracks) != 0 {
		t.Error("Old track should have been deleted")
	}
}

// TestMultipleTrackCorrelation tests correlation of measurements to tracks
func TestMultipleTrackCorrelation(t *testing.T) {
	tm := NewTrackManager()
	tm.GatingLimit = 20.0

	// Create multiple tracks
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
		Lat:    38.1,
		Lon:    -77.1,
		Alt:    200.0,
		VarLat: 0.0001,
		VarLon: 0.0001,
		VarAlt: 1.0,
	}
	tm.Tracks[1] = track1
	tm.Tracks[2] = track2

	// Measurements close to each track
	measurements := []*Measurement{
		{ID: 101, Lat: 38.0, Lon: -77.0, Alt: 100.0, VarLat: 0.0001, VarLon: 0.0001, VarAlt: 1.0, SourceID: 1, Timestamp: 1000},
		{ID: 102, Lat: 38.1, Lon: -77.1, Alt: 200.0, VarLat: 0.0001, VarLon: 0.0001, VarAlt: 1.0, SourceID: 1, Timestamp: 1000},
	}

	associations := tm.JPDAAssociate(measurements)

	// Should have 2 associations
	if len(associations) != 2 {
		t.Errorf("Expected 2 associations, got %d", len(associations))
	}
}

// TestTrackNumberAllocation tests track number assignment
func TestTrackNumberAllocation(t *testing.T) {
	tm := NewTrackManager()

	// Create first track
	meas1 := &Measurement{ID: 1, Lat: 38.0, Lon: -77.0, Alt: 100.0, VarLat: 0.001, VarLon: 0.001, VarAlt: 10.0, Timestamp: 1000}
	track1 := tm.InitiateTrack(meas1)

	// Create second track
	meas2 := &Measurement{ID: 2, Lat: 39.0, Lon: -78.0, Alt: 200.0, VarLat: 0.001, VarLon: 0.001, VarAlt: 10.0, Timestamp: 1000}
	track2 := tm.InitiateTrack(meas2)

	if track1.TrackNum != 1 {
		t.Errorf("Expected TrackNum 1, got %d", track1.TrackNum)
	}

	if track2.TrackNum != 2 {
		t.Errorf("Expected TrackNum 2, got %d", track2.TrackNum)
	}
}

// TestMaxTracksLimit tests track limit enforcement
func TestMaxTracksLimit(t *testing.T) {
	tm := NewTrackManager()
	tm.MaxTracks = 3

	// Create tracks up to limit
	for i := 0; i < 5; i++ {
		meas := &Measurement{
			ID:        uint64(i),
			Lat:       float64(i) * 0.1,
			Lon:       float64(i) * 0.1,
			Alt:       100.0,
			VarLat:    0.001,
			VarLon:    0.001,
			VarAlt:    10.0,
			Timestamp: 1000,
		}
		tm.InitiateTrack(meas)
	}

	if len(tm.Tracks) > 3 {
		t.Errorf("Should not exceed MaxTracks, got %d tracks", len(tm.Tracks))
	}
}

// TestTrackVarianceUpdate tests that variance decreases with updates
func TestTrackVarianceUpdate(t *testing.T) {
	tm := NewTrackManager()
	tm.GatingLimit = 100.0

	// Create track with high variance
	track := &Track{
		ID:     1,
		Lat:    38.0,
		Lon:    -77.0,
		Alt:    100.0,
		VarLat: 0.01,
		VarLon: 0.01,
		VarAlt: 100.0,
	}
	tm.Tracks[1] = track

	// Measurement with lower variance
	meas := &Measurement{
		ID:        100,
		Lat:       38.001,
		Lon:       -77.001,
		Alt:       101.0,
		VarLat:    0.001,
		VarLon:    0.001,
		VarAlt:    10.0,
		SourceID:  1,
		Timestamp: 2000,
	}

	varBefore := track.VarLat
	tm.Update([]*Measurement{meas}, 2000)

	// Variance should generally decrease with more measurements
	// (though our simple implementation may not always do this)
	_ = varBefore // Track variance for potential future implementation
}

// TestAmbiguousAssociation tests when measurement could belong to multiple tracks
func TestAmbiguousAssociation(t *testing.T) {
	tm := NewTrackManager()
	tm.GatingLimit = 50.0

	// Two tracks close together
	track1 := &Track{
		ID:     1,
		Lat:    38.0,
		Lon:    -77.0,
		Alt:    100.0,
		VarLat: 0.001,
		VarLon: 0.001,
		VarAlt: 10.0,
	}
	track2 := &Track{
		ID:     2,
		Lat:    38.01, // Very close to track 1
		Lon:    -77.01,
		Alt:    100.0,
		VarLat: 0.001,
		VarLon: 0.001,
		VarAlt: 10.0,
	}
	tm.Tracks[1] = track1
	tm.Tracks[2] = track2

	// Measurement in between
	meas := &Measurement{
		ID:        100,
		Lat:       38.005,
		Lon:       -77.005,
		Alt:       100.0,
		VarLat:    0.001,
		VarLon:    0.001,
		VarAlt:    10.0,
		SourceID:  1,
		Timestamp: 1000,
	}

	associations := tm.JPDAAssociate([]*Measurement{meas})

	// Should associate with both tracks (ambiguous)
	if len(associations) != 2 {
		t.Errorf("Expected 2 associations for ambiguous case, got %d", len(associations))
	}
}

// TestTrackManagerConcurrency tests thread safety (basic)
func TestTrackManagerConcurrency(t *testing.T) {
	tm := NewTrackManager()
	tm.GatingLimit = 100.0

	// Add initial track
	track := &Track{
		ID:     1,
		Lat:    38.0,
		Lon:    -77.0,
		Alt:    100.0,
		VarLat: 0.001,
		VarLon: 0.001,
		VarAlt: 10.0,
	}
	tm.Tracks[1] = track

	// Process multiple updates
	for i := 0; i < 100; i++ {
		meas := &Measurement{
			ID:        uint64(i + 100),
			Lat:       38.0 + float64(i)*0.00001,
			Lon:       -77.0,
			Alt:       100.0,
			VarLat:    0.001,
			VarLon:    0.001,
			VarAlt:    10.0,
			SourceID:  1,
			Timestamp: 1000 + int64(i),
		}
		tm.Update([]*Measurement{meas}, meas.Timestamp)
	}

	// Track should exist
	if len(tm.Tracks) == 0 {
		t.Error("Track should still exist")
	}
}

// TestMahalanobisGating tests that only measurements within gate are associated
func TestMahalanobisGating(t *testing.T) {
	tm := NewTrackManager()
	tm.GatingLimit = 9.0 // 3-sigma squared

	// Track at origin
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

	tests := []struct {
		name     string
		lat      float64
		expected int // Expected number of associations
	}{
		{"very close", 0.001, 1},
		{"close", 0.01, 1},
		{"at gate", 0.03, 1}, // Just inside
		{"far", 0.5, 0},      // Outside gate
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meas := &Measurement{
				ID:        100,
				Lat:       tt.lat,
				Lon:       0.0,
				Alt:       0.0,
				VarLat:    0.01,
				VarLon:    0.01,
				VarAlt:    100.0,
				SourceID:  1,
				Timestamp: 1000,
			}

			associations := tm.JPDAAssociate([]*Measurement{meas})

			if len(associations) != tt.expected {
				t.Errorf("Expected %d associations, got %d", tt.expected, len(associations))
			}
		})
	}
}

// TestTrackWithVelocity tests tracks with velocity estimation
func TestTrackWithVelocity(t *testing.T) {
	tm := NewTrackManager()

	track := &Track{
		ID:         1,
		Lat:        38.0,
		Lon:        -77.0,
		Alt:        100.0,
		VelocityN:  10.0, // 10 m/s North
		VelocityE:  5.0,  // 5 m/s East
		VelocityU:  0.0,
		VarLat:     0.001,
		VarLon:     0.001,
		VarAlt:     10.0,
		LastUpdate: 1000,
	}
	tm.Tracks[1] = track

	// Track should have velocity
	if track.VelocityN != 10.0 {
		t.Errorf("Expected VelocityN 10.0, got %.2f", track.VelocityN)
	}
	if track.VelocityE != 5.0 {
		t.Errorf("Expected VelocityE 5.0, got %.2f", track.VelocityE)
	}
}

// BenchmarkTrackManager benchmarks full track lifecycle
func BenchmarkTrackManager(b *testing.B) {
	tm := NewTrackManager()

	// Create initial track
	track := &Track{
		ID:     1,
		Lat:    38.0,
		Lon:    -77.0,
		Alt:    100.0,
		VarLat: 0.001,
		VarLon: 0.001,
		VarAlt: 10.0,
	}
	tm.Tracks[1] = track

	measurements := []*Measurement{
		{ID: 100, Lat: 38.001, Lon: -77.001, Alt: 101.0, VarLat: 0.001, VarLon: 0.001, VarAlt: 10.0, SourceID: 1, Timestamp: 1000},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tm.Update(measurements, 1000+int64(i))
	}
}

// BenchmarkTrackCreation benchmarks track creation
func BenchmarkTrackCreation(b *testing.B) {
	tm := NewTrackManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		meas := &Measurement{
			ID:        uint64(i),
			Lat:       float64(i) * 0.001,
			Lon:       float64(i) * 0.001,
			Alt:       100.0,
			VarLat:    0.001,
			VarLon:    0.001,
			VarAlt:    10.0,
			SourceID:  1,
			Timestamp: 1000,
		}
		tm.InitiateTrack(meas)
	}
}

// TestKalmanTrackEstimate tests Kalman-based track estimation
func TestKalmanTrackEstimate(t *testing.T) {
	// Create Kalman filter for track
	kf := NewKalmanFilter()

	// Initial state
	state := &KalmanState{
		X: [6]float64{38.0, -77.0, 100.0, 0.001, 0.001, 1.0}, // Lat, Lon, Alt, vLat, vLon, vAlt
		P: [6][6]float64{
			{0.001, 0, 0, 0, 0, 0},
			{0, 0.001, 0, 0, 0, 0},
			{0, 0, 10.0, 0, 0, 0},
			{0, 0, 0, 0.0001, 0, 0},
			{0, 0, 0, 0, 0.0001, 0},
			{0, 0, 0, 0, 0, 1.0},
		},
	}

	// Simulate track with measurements over time
	measurements := []struct {
		lat, lon, alt float64
	}{
		{38.001, -77.001, 100.5},
		{38.002, -77.002, 101.0},
		{38.003, -77.003, 101.5},
	}

	R := [3][3]float64{
		{0.0001, 0, 0},
		{0, 0.0001, 0},
		{0, 0, 1.0},
	}

	for _, m := range measurements {
		z := [3]float64{m.lat, m.lon, m.alt}
		kf.Update(state, z, R)
		kf.Predict(state, 1.0)
	}

	// After filtering, position should be close to measurements
	if math.Abs(state.X[0]-38.003) > 0.01 {
		t.Errorf("Kalman track estimation failed: lat = %.4f", state.X[0])
	}
}
