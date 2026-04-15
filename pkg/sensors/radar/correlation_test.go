package radar

import (
	"math"
	"testing"
	"time"
)

// TestCorrelationConfig tests correlation configuration
func TestCorrelationConfig(t *testing.T) {
	config := DefaultCorrelationConfig()

	if config.MaxDistance != 5000.0 {
		t.Errorf("Expected max distance 5000, got %f", config.MaxDistance)
	}
	if config.MaxVelocityDiff != 100.0 {
		t.Errorf("Expected max velocity diff 100, got %f", config.MaxVelocityDiff)
	}
	if config.MinConfidence != 0.5 {
		t.Errorf("Expected min confidence 0.5, got %f", config.MinConfidence)
	}
}

// TestNewTrackCorrelator tests correlator creation
func TestNewTrackCorrelator(t *testing.T) {
	corr := NewTrackCorrelator(nil)

	if corr == nil {
		t.Fatal("Correlator should not be nil")
	}

	if corr.config.MaxDistance != 5000.0 {
		t.Error("Default config should be used")
	}
}

// TestCorrelateNewTrack tests correlating a new track
func TestCorrelateNewTrack(t *testing.T) {
	corr := NewTrackCorrelator(nil)

	track := &RadarTrack{
		TrackNumber:  1001,
		SensorID:     "TPY2-1",
		Timestamp:    time.Now(),
		Latitude:     38.8977,
		Longitude:    -77.0365,
		Altitude:     10000.0,
		VelocityN:    300.0,
		VelocityE:    0.0,
		VelocityU:    0.0,
		TrackQuality: 5,
	}

	result := corr.Correlate(track)

	if !result.IsNew {
		t.Error("Track should be new")
	}
	if result.TrackNumber < 1000 {
		t.Errorf("Track number should be >= 1000, got %d", result.TrackNumber)
	}
	if len(result.SourceTracks) != 1 {
		t.Errorf("Expected 1 source track, got %d", len(result.SourceTracks))
	}
}

// TestCorrelateExistingTrack tests correlating with existing track
func TestCorrelateExistingTrack(t *testing.T) {
	corr := NewTrackCorrelator(nil)

	// Create first track
	track1 := &RadarTrack{
		TrackNumber:  1001,
		SensorID:     "TPY2-1",
		Timestamp:    time.Now(),
		Latitude:     38.8977,
		Longitude:    -77.0365,
		Altitude:     10000.0,
		VelocityN:    300.0,
		VelocityE:    0.0,
		VelocityU:    0.0,
		TrackQuality: 5,
	}

	result1 := corr.Correlate(track1)

	// Create second track close to first
	track2 := &RadarTrack{
		TrackNumber:  1002,
		SensorID:     "TPY2-2",
		Timestamp:    time.Now(),
		Latitude:     38.8978, // Very close
		Longitude:    -77.0366,
		Altitude:     10010.0,
		VelocityN:    301.0,
		VelocityE:    1.0,
		VelocityU:    0.0,
		TrackQuality: 5,
	}

	result2 := corr.Correlate(track2)

	if !result2.IsCorrelated {
		t.Error("Track should be correlated")
	}
	if result2.TrackNumber != result1.TrackNumber {
		t.Errorf("Track should correlate to same track number: %d vs %d",
			result2.TrackNumber, result1.TrackNumber)
	}
	if len(result2.SourceTracks) != 2 {
		t.Errorf("Expected 2 source tracks, got %d", len(result2.SourceTracks))
	}
}

// TestPositionDistance tests distance calculation
func TestPositionDistance(t *testing.T) {
	corr := NewTrackCorrelator(nil)

	tests := []struct {
		name             string
		lat1, lon1, alt1 float64
		lat2, lon2, alt2 float64
		expected         float64
		tolerance        float64
	}{
		{
			name: "same position",
			lat1: 38.8977, lon1: -77.0365, alt1: 10000,
			lat2: 38.8977, lon2: -77.0365, alt2: 10000,
			expected:  0.0,
			tolerance: 1.0,
		},
		{
			name: "100m apart",
			lat1: 38.8977, lon1: -77.0365, alt1: 10000,
			lat2: 38.8986, lon2: -77.0365, alt2: 10000, // ~100m north
			expected:  100.0,
			tolerance: 10.0,
		},
		{
			name: "1km apart",
			lat1: 38.8977, lon1: -77.0365, alt1: 10000,
			lat2: 38.9067, lon2: -77.0365, alt2: 10000, // ~1km north
			expected:  1000.0,
			tolerance: 100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			distance := corr.positionDistance(
				tt.lat1, tt.lon1, tt.alt1,
				tt.lat2, tt.lon2, tt.alt2,
			)

			diff := math.Abs(distance - tt.expected)
			if diff > tt.tolerance {
				t.Errorf("Distance = %.1f, expected %.1f (±%.1f)",
					distance, tt.expected, tt.tolerance)
			}
		})
	}
}

// TestVelocityDifference tests velocity difference calculation
func TestVelocityDifference(t *testing.T) {
	corr := NewTrackCorrelator(nil)

	tests := []struct {
		name          string
		vn1, ve1, vu1 float64
		vn2, ve2, vu2 float64
		expected      float64
		tolerance     float64
	}{
		{
			name: "same velocity",
			vn1:  300, ve1: 0, vu1: 0,
			vn2: 300, ve2: 0, vu2: 0,
			expected:  0.0,
			tolerance: 0.1,
		},
		{
			name: "10 m/s difference",
			vn1:  300, ve1: 0, vu1: 0,
			vn2: 310, ve2: 0, vu2: 0,
			expected:  10.0,
			tolerance: 0.1,
		},
		{
			name: "diagonal difference",
			vn1:  300, ve1: 0, vu1: 0,
			vn2: 300, ve2: 10, vu2: 10,
			expected:  14.14, // sqrt(10^2 + 10^2)
			tolerance: 0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := corr.velocityDifference(tt.vn1, tt.ve1, tt.vu1, tt.vn2, tt.ve2, tt.vu2)

			if math.Abs(diff-tt.expected) > tt.tolerance {
				t.Errorf("Velocity diff = %.2f, expected %.2f", diff, tt.expected)
			}
		})
	}
}

// TestCalculateScore tests correlation score calculation
func TestCalculateScore(t *testing.T) {
	config := DefaultCorrelationConfig()
	config.MinConfidence = 0.3
	corr := NewTrackCorrelator(config)

	// Create existing track
	existing := &CorrelatedTrack{
		TrackNumber: 1000,
		Position: Position{
			Lat: 38.8977,
			Lon: -77.0365,
			Alt: 10000.0,
		},
		Velocity: Velocity{
			N: 300.0,
			E: 0.0,
			U: 0.0,
		},
		NUpdates:   5,
		LastUpdate: time.Now(),
	}

	// Close track
	closeTrack := &RadarTrack{
		Latitude:     38.8978,
		Longitude:    -77.0366,
		Altitude:     10010,
		VelocityN:    301,
		VelocityE:    1,
		VelocityU:    0,
		TrackQuality: 5,
	}

	score := corr.calculateScore(closeTrack, existing)
	if score < config.MinConfidence {
		t.Errorf("Close track should score >= %.2f, got %.2f", config.MinConfidence, score)
	}

	// Far track
	farTrack := &RadarTrack{
		Latitude:     39.0, // ~10km away
		Longitude:    -77.0,
		Altitude:     20000,
		VelocityN:    500,
		VelocityE:    200,
		VelocityU:    100,
		TrackQuality: 5,
	}

	score = corr.calculateScore(farTrack, existing)
	if score > 0.1 {
		t.Errorf("Far track should score low, got %.2f", score)
	}
}

// TestUpdateTrackStates tests track state updates
func TestUpdateTrackStates(t *testing.T) {
	config := DefaultCorrelationConfig()
	config.MinUpdates = 3
	config.CoastTime = 10 * time.Second
	config.DropTime = 30 * time.Second

	corr := NewTrackCorrelator(config)

	// Create track with 3 updates
	track := &RadarTrack{
		TrackNumber:  1001,
		SensorID:     "TPY2-1",
		Timestamp:    time.Now(),
		Latitude:     38.8977,
		Longitude:    -77.0365,
		Altitude:     10000.0,
		TrackQuality: 5,
	}

	// Correlate 3 times
	for i := 0; i < 3; i++ {
		corr.Correlate(track)
	}

	corr.UpdateTrackStates()

	stats := corr.Stats()
	if stats.ActiveTracks != 1 {
		t.Errorf("Expected 1 active track, got %d", stats.ActiveTracks)
	}

	// Get track and check status
	corrTrack := corr.GetTrack(1000) // First track number
	if corrTrack == nil {
		t.Fatal("Track should exist")
	}
	if corrTrack.Status != TrackStatusTrack {
		t.Errorf("Expected status %s, got %s", TrackStatusTrack, corrTrack.Status)
	}
}

// TestPruneOldTracks tests pruning old tracks
func TestPruneOldTracks(t *testing.T) {
	config := DefaultCorrelationConfig()
	config.MaxTrackAge = 60 * time.Second

	corr := NewTrackCorrelator(config)

	// Create track
	track := &RadarTrack{
		TrackNumber:  1001,
		SensorID:     "TPY2-1",
		Timestamp:    time.Now(),
		Latitude:     38.8977,
		Longitude:    -77.0365,
		Altitude:     10000.0,
		TrackQuality: 5,
	}

	corr.Correlate(track)

	// Should have 1 track
	stats := corr.Stats()
	if stats.TotalTracks != 1 {
		t.Errorf("Expected 1 track, got %d", stats.TotalTracks)
	}

	// Prune (should remove 0)
	pruned := corr.PruneOldTracks()
	if pruned != 0 {
		t.Errorf("Expected 0 pruned tracks, got %d", pruned)
	}
}

// TestGetAllTracks tests getting all tracks
func TestGetAllTracks(t *testing.T) {
	config := DefaultCorrelationConfig()
	config.MinConfidence = 1.0 // Require exact match for new tracks
	corr := NewTrackCorrelator(config)

	// Create multiple tracks at different positions
	for i := 0; i < 5; i++ {
		track := &RadarTrack{
			TrackNumber:  uint32(1001 + i),
			SensorID:     "TPY2-" + string(rune('1'+i)),
			Timestamp:    time.Now(),
			Latitude:     38.8977 + float64(i)*0.1, // Different positions
			Longitude:    -77.0365 + float64(i)*0.1,
			Altitude:     10000.0,
			TrackQuality: 5,
		}
		corr.Correlate(track)
	}

	allTracks := corr.GetAllTracks()
	if len(allTracks) != 5 {
		t.Errorf("Expected 5 tracks, got %d", len(allTracks))
	}
}

// TestGetActiveTracks tests getting active tracks
func TestGetActiveTracks(t *testing.T) {
	config := DefaultCorrelationConfig()
	config.MaxTrackAge = 10 * time.Second

	corr := NewTrackCorrelator(config)

	// Create track
	track := &RadarTrack{
		TrackNumber:  1001,
		SensorID:     "TPY2-1",
		Timestamp:    time.Now(),
		Latitude:     38.8977,
		Longitude:    -77.0365,
		Altitude:     10000.0,
		TrackQuality: 5,
	}

	corr.Correlate(track)

	activeTracks := corr.GetActiveTracks()
	if len(activeTracks) != 1 {
		t.Errorf("Expected 1 active track, got %d", len(activeTracks))
	}
}

// TestCorrelatorStats tests correlator statistics
func TestCorrelatorStats(t *testing.T) {
	corr := NewTrackCorrelator(nil)

	// Initial stats
	stats := corr.Stats()
	if stats.TotalTracks != 0 {
		t.Errorf("Expected 0 total tracks, got %d", stats.TotalTracks)
	}

	// Add track
	track := &RadarTrack{
		TrackNumber:  1001,
		SensorID:     "TPY2-1",
		Timestamp:    time.Now(),
		Latitude:     38.8977,
		Longitude:    -77.0365,
		Altitude:     10000.0,
		TrackQuality: 5,
	}

	corr.Correlate(track)

	stats = corr.Stats()
	if stats.TotalTracks != 1 {
		t.Errorf("Expected 1 total track, got %d", stats.TotalTracks)
	}
	if stats.ActiveTracks != 1 {
		t.Errorf("Expected 1 active track, got %d", stats.ActiveTracks)
	}
}

// TestWeightedPositionUpdate tests weighted position updates
func TestWeightedPositionUpdate(t *testing.T) {
	corr := NewTrackCorrelator(nil)

	// Create first track
	track1 := &RadarTrack{
		TrackNumber:  1001,
		SensorID:     "TPY2-1",
		Timestamp:    time.Now(),
		Latitude:     38.8977,
		Longitude:    -77.0365,
		Altitude:     10000.0,
		VelocityN:    300.0,
		VelocityE:    0.0,
		VelocityU:    0.0,
		TrackQuality: 5,
	}

	result1 := corr.Correlate(track1)

	// Create second track at different position
	track2 := &RadarTrack{
		TrackNumber:  1002,
		SensorID:     "TPY2-2",
		Timestamp:    time.Now(),
		Latitude:     38.9077, // ~1km away
		Longitude:    -77.0465,
		Altitude:     11000.0,
		VelocityN:    300.0,
		VelocityE:    0.0,
		VelocityU:    0.0,
		TrackQuality: 5,
	}

	_ = corr.Correlate(track2)

	// Get track
	corrTrack := corr.GetTrack(result1.TrackNumber)
	if corrTrack == nil {
		t.Fatal("Track should exist")
	}

	// Check that position was updated
	// Position should be somewhere between the two positions
	if corrTrack.Position.Lat == 38.8977 {
		// Position should have changed after second correlation
		// (weighted average of two positions)
		t.Error("Position should have been updated after second correlation")
	}
}

// TestMultipleSources tests correlation from multiple sources
func TestMultipleSources(t *testing.T) {
	corr := NewTrackCorrelator(nil)

	// Track from TPY2
	trackTPY2 := &RadarTrack{
		TrackNumber:  1001,
		SensorID:     "TPY2-1",
		Timestamp:    time.Now(),
		Latitude:     38.8977,
		Longitude:    -77.0365,
		Altitude:     10000.0,
		VelocityN:    300.0,
		VelocityE:    0.0,
		VelocityU:    0.0,
		TrackQuality: 5,
	}

	// Track from SBX
	trackSBX := &RadarTrack{
		TrackNumber:  2001,
		SensorID:     "SBX-1",
		Timestamp:    time.Now(),
		Latitude:     38.8978, // Very close
		Longitude:    -77.0366,
		Altitude:     10010.0,
		VelocityN:    301.0,
		VelocityE:    1.0,
		VelocityU:    0.0,
		TrackQuality: 5,
	}

	// Track from UEWR
	trackUEWR := &RadarTrack{
		TrackNumber:  3001,
		SensorID:     "UEWR-1",
		Timestamp:    time.Now(),
		Latitude:     38.8979, // Very close
		Longitude:    -77.0367,
		Altitude:     10020.0,
		VelocityN:    302.0,
		VelocityE:    2.0,
		VelocityU:    0.0,
		TrackQuality: 5,
	}

	// Correlate all three
	result1 := corr.Correlate(trackTPY2)
	result2 := corr.Correlate(trackSBX)
	result3 := corr.Correlate(trackUEWR)

	// All should correlate to same track
	if result1.TrackNumber != result2.TrackNumber {
		t.Errorf("TPY2 and SBX should correlate: %d vs %d",
			result1.TrackNumber, result2.TrackNumber)
	}
	if result1.TrackNumber != result3.TrackNumber {
		t.Errorf("TPY2 and UEWR should correlate: %d vs %d",
			result1.TrackNumber, result3.TrackNumber)
	}

	// Check source tracks
	corrTrack := corr.GetTrack(result1.TrackNumber)
	if len(corrTrack.SourceTracks) != 3 {
		t.Errorf("Expected 3 source tracks, got %d", len(corrTrack.SourceTracks))
	}
}

// BenchmarkCorrelation benchmarks track correlation
func BenchmarkCorrelation(b *testing.B) {
	corr := NewTrackCorrelator(nil)

	track := &RadarTrack{
		TrackNumber:  1001,
		SensorID:     "TPY2-1",
		Timestamp:    time.Now(),
		Latitude:     38.8977,
		Longitude:    -77.0365,
		Altitude:     10000.0,
		VelocityN:    300.0,
		VelocityE:    0.0,
		VelocityU:    0.0,
		TrackQuality: 5,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		track.TrackNumber = uint32(1001 + i%1000)
		corr.Correlate(track)
	}
}

// BenchmarkScoreCalculation benchmarks score calculation
func BenchmarkScoreCalculation(b *testing.B) {
	corr := NewTrackCorrelator(nil)

	existing := &CorrelatedTrack{
		TrackNumber: 1000,
		Position: Position{
			Lat: 38.8977,
			Lon: -77.0365,
			Alt: 10000.0,
		},
		Velocity: Velocity{
			N: 300.0,
			E: 0.0,
			U: 0.0,
		},
		NUpdates:   5,
		LastUpdate: time.Now(),
	}

	track := &RadarTrack{
		Latitude:     38.8978,
		Longitude:    -77.0366,
		Altitude:     10010,
		VelocityN:    301,
		VelocityE:    1,
		VelocityU:    0,
		TrackQuality: 5,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		corr.calculateScore(track, existing)
	}
}
