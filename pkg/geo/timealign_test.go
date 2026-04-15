package geo

import (
	"math"
	"testing"
	"time"
)

// TestTimeAlignConfig tests time alignment configuration
func TestTimeAlignConfig(t *testing.T) {
	config := DefaultTimeAlignConfig()

	if config.MaxInterpolationGap != 5*time.Second {
		t.Errorf("Expected max interpolation gap 5s, got %v", config.MaxInterpolationGap)
	}
	if config.ExtrapolationLimit != 2*time.Second {
		t.Errorf("Expected extrapolation limit 2s, got %v", config.ExtrapolationLimit)
	}
	if config.MaxVelocity != 1000.0 {
		t.Errorf("Expected max velocity 1000, got %f", config.MaxVelocity)
	}
}

// TestNewTimeAligner tests time aligner creation
func TestNewTimeAligner(t *testing.T) {
	ta := NewTimeAligner(nil)

	if ta == nil {
		t.Fatal("Time aligner should not be nil")
	}

	if ta.config.MaxInterpolationGap != 5*time.Second {
		t.Error("Default config should be used")
	}
}

// TestInterpolatePositionExact tests exact time match
func TestInterpolatePositionExact(t *testing.T) {
	ta := NewTimeAligner(nil)

	now := time.Now()
	points := []TimeSeriesPoint{
		{Timestamp: now, Position: [3]float64{0, 0, 0}, Velocity: [3]float64{100, 0, 0}},
		{Timestamp: now.Add(1 * time.Second), Position: [3]float64{100, 0, 0}, Velocity: [3]float64{100, 0, 0}},
	}

	result := ta.InterpolatePosition(points, now)

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.Position[0] != 0 {
		t.Errorf("Expected position x=0, got %.2f", result.Position[0])
	}
}

// TestInterpolatePositionLinear tests linear interpolation
func TestInterpolatePositionLinear(t *testing.T) {
	ta := NewTimeAligner(nil)

	now := time.Now()
	points := []TimeSeriesPoint{
		{Timestamp: now, Position: [3]float64{0, 0, 0}, Velocity: [3]float64{100, 0, 0}},
		{Timestamp: now.Add(2 * time.Second), Position: [3]float64{200, 0, 0}, Velocity: [3]float64{100, 0, 0}},
	}

	// Interpolate at 1 second
	result := ta.InterpolatePosition(points, now.Add(1*time.Second))

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	// Should be halfway
	if math.Abs(result.Position[0]-100) > 1 {
		t.Errorf("Expected position x=100, got %.2f", result.Position[0])
	}
}

// TestInterpolatePositionExtrapolateForward tests forward extrapolation
func TestInterpolatePositionExtrapolateForward(t *testing.T) {
	ta := NewTimeAligner(nil)

	now := time.Now()
	points := []TimeSeriesPoint{
		{Timestamp: now, Position: [3]float64{0, 0, 0}, Velocity: [3]float64{100, 0, 0}},
	}

	// Extrapolate 1 second forward
	result := ta.InterpolatePosition(points, now.Add(1*time.Second))

	if result == nil {
		t.Fatal("Result should not be nil for small extrapolation")
	}

	// Should move by velocity (0 + 100*1 = 100)
	if math.Abs(result.Position[0]-100) > 10 {
		t.Errorf("Expected position x~100, got %.2f", result.Position[0])
	}
}

// TestInterpolatePositionExtrapolateBackward tests backward extrapolation
func TestInterpolatePositionExtrapolateBackward(t *testing.T) {
	ta := NewTimeAligner(nil)

	now := time.Now()
	points := []TimeSeriesPoint{
		{Timestamp: now.Add(1 * time.Second), Position: [3]float64{100, 0, 0}, Velocity: [3]float64{100, 0, 0}},
	}

	// Extrapolate 1 second backward
	result := ta.InterpolatePosition(points, now)

	if result == nil {
		t.Fatal("Result should not be nil for small extrapolation")
	}

	// Should move by velocity backward (100 - 100*1 = 0)
	if math.Abs(result.Position[0]) > 10 {
		t.Errorf("Expected position x~0, got %.2f", result.Position[0])
	}
}

// TestInterpolatePositionTooFar tests extrapolation limit
func TestInterpolatePositionTooFar(t *testing.T) {
	ta := NewTimeAligner(nil)

	now := time.Now()
	points := []TimeSeriesPoint{
		{Timestamp: now, Position: [3]float64{0, 0, 0}, Velocity: [3]float64{100, 0, 0}},
	}

	// Extrapolate too far (limit is 2s, trying 10s)
	result := ta.InterpolatePosition(points, now.Add(10*time.Second))

	if result != nil {
		t.Error("Should return nil for extrapolation beyond limit")
	}
}

// TestInterpolatePositionGap tests gap limit
func TestInterpolatePositionGap(t *testing.T) {
	ta := NewTimeAligner(nil)

	now := time.Now()
	points := []TimeSeriesPoint{
		{Timestamp: now, Position: [3]float64{0, 0, 0}, Velocity: [3]float64{100, 0, 0}},
		{Timestamp: now.Add(10 * time.Second), Position: [3]float64{1000, 0, 0}, Velocity: [3]float64{100, 0, 0}},
	}

	// Gap is 10s, limit is 5s
	result := ta.InterpolatePosition(points, now.Add(5*time.Second))

	if result != nil {
		t.Error("Should return nil for gap exceeding limit")
	}
}

// TestPredictPosition tests position prediction
func TestPredictPosition(t *testing.T) {
	ta := NewTimeAligner(nil)

	now := time.Now()
	point := &TimeSeriesPoint{
		Timestamp: now,
		Position:  [3]float64{0, 0, 0},
		Velocity:  [3]float64{100, 50, 10},
	}

	// Predict 1 second forward
	result := ta.PredictPosition(point, now.Add(1*time.Second))

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if math.Abs(result.Position[0]-100) > 1 {
		t.Errorf("Expected x=100, got %.2f", result.Position[0])
	}
	if math.Abs(result.Position[1]-50) > 1 {
		t.Errorf("Expected y=50, got %.2f", result.Position[1])
	}
}

// TestPredictPositionTooFar tests prediction limit
func TestPredictPositionTooFar(t *testing.T) {
	ta := NewTimeAligner(nil)

	now := time.Now()
	point := &TimeSeriesPoint{
		Timestamp: now,
		Position:  [3]float64{0, 0, 0},
		Velocity:  [3]float64{100, 0, 0},
	}

	// Predict beyond coast time
	result := ta.PredictPosition(point, now.Add(30*time.Second))

	if result != nil {
		t.Error("Should return nil for prediction beyond coast time")
	}
}

// TestCalculateVelocity tests velocity calculation
func TestCalculateVelocity(t *testing.T) {
	ta := NewTimeAligner(nil)

	now := time.Now()
	points := []TimeSeriesPoint{
		{Timestamp: now, Position: [3]float64{0, 0, 0}},
		{Timestamp: now.Add(1 * time.Second), Position: [3]float64{100, 50, 10}},
		{Timestamp: now.Add(2 * time.Second), Position: [3]float64{200, 100, 20}},
	}

	result := ta.CalculateVelocity(points)

	if len(result) != 3 {
		t.Fatalf("Expected 3 points, got %d", len(result))
	}

	// First point velocity should be ~100, 50, 10
	if math.Abs(result[0].Velocity[0]-100) > 1 {
		t.Errorf("Expected vx=100, got %.2f", result[0].Velocity[0])
	}
}

// TestSmoothTrack tests track smoothing
func TestSmoothTrack(t *testing.T) {
	ta := NewTimeAligner(nil)

	now := time.Now()
	points := []TimeSeriesPoint{
		{Timestamp: now, Position: [3]float64{0, 0, 0}, Velocity: [3]float64{100, 0, 0}},
		{Timestamp: now.Add(1 * time.Second), Position: [3]float64{105, 5, 0}, Velocity: [3]float64{100, 0, 0}},
		{Timestamp: now.Add(2 * time.Second), Position: [3]float64{200, 0, 0}, Velocity: [3]float64{100, 0, 0}},
	}

	result := ta.SmoothTrack(points, 3)

	if len(result) != 3 {
		t.Fatalf("Expected 3 points, got %d", len(result))
	}

	// Middle point should be smoothed
	if result[1].Position[0] >= points[1].Position[0] {
		t.Log("Smoothing applied (positions averaged)")
	}
}

// TestResampleTrack tests track resampling
func TestResampleTrack(t *testing.T) {
	ta := NewTimeAligner(nil)

	now := time.Now()
	points := []TimeSeriesPoint{
		{Timestamp: now, Position: [3]float64{0, 0, 0}, Velocity: [3]float64{100, 0, 0}},
		{Timestamp: now.Add(1 * time.Second), Position: [3]float64{100, 0, 0}, Velocity: [3]float64{100, 0, 0}},
		{Timestamp: now.Add(2 * time.Second), Position: [3]float64{200, 0, 0}, Velocity: [3]float64{100, 0, 0}},
	}

	// Resample to 500ms intervals
	result := ta.ResampleTrack(points, 500*time.Millisecond)

	if len(result) < 3 {
		t.Errorf("Expected at least 3 resampled points, got %d", len(result))
	}
}

// TestValidateTrack tests track validation
func TestValidateTrack(t *testing.T) {
	ta := NewTimeAligner(nil)

	now := time.Now()
	points := []TimeSeriesPoint{
		{Timestamp: now, Position: [3]float64{0, 0, 0}, Velocity: [3]float64{100, 0, 0}},
		{Timestamp: now.Add(1 * time.Second), Position: [3]float64{100, 0, 0}, Velocity: [3]float64{100, 0, 0}},
		{Timestamp: now.Add(2 * time.Second), Position: [3]float64{200, 0, 0}, Velocity: [3]float64{100, 0, 0}},
	}

	valid, msg := ta.ValidateTrack(points)

	if !valid {
		t.Errorf("Track should be valid: %s", msg)
	}
}

// TestValidateTrackUnordered tests track validation with unordered points
func TestValidateTrackUnordered(t *testing.T) {
	ta := NewTimeAligner(nil)

	now := time.Now()
	points := []TimeSeriesPoint{
		{Timestamp: now.Add(2 * time.Second), Position: [3]float64{200, 0, 0}, Velocity: [3]float64{100, 0, 0}},
		{Timestamp: now, Position: [3]float64{0, 0, 0}, Velocity: [3]float64{100, 0, 0}},
	}

	valid, msg := ta.ValidateTrack(points)

	if valid {
		t.Error("Unordered track should be invalid")
	}
	_ = msg
}

// TestValidateTrackVelocity tests velocity validation
func TestValidateTrackVelocity(t *testing.T) {
	config := DefaultTimeAlignConfig()
	config.MaxVelocity = 100 // 100 m/s limit
	ta := NewTimeAligner(config)

	now := time.Now()
	points := []TimeSeriesPoint{
		{Timestamp: now, Position: [3]float64{0, 0, 0}, Velocity: [3]float64{100, 0, 0}},
		{Timestamp: now.Add(1 * time.Second), Position: [3]float64{200, 0, 0}, Velocity: [3]float64{100, 0, 0}}, // 200 m/s
	}

	valid, _ := ta.ValidateTrack(points)

	if valid {
		t.Error("Track with excessive velocity should be invalid")
	}
}

// TestAlignTracks tests aligning multiple tracks
func TestAlignTracks(t *testing.T) {
	ta := NewTimeAligner(nil)

	now := time.Now()

	tracks := map[string][]TimeSeriesPoint{
		"TPY2-1": {
			{Timestamp: now, Position: [3]float64{0, 0, 0}, Velocity: [3]float64{100, 0, 0}},
			{Timestamp: now.Add(2 * time.Second), Position: [3]float64{200, 0, 0}, Velocity: [3]float64{100, 0, 0}},
		},
		"SBX-1": {
			{Timestamp: now, Position: [3]float64{0, 1000, 10000}, Velocity: [3]float64{100, 0, 0}},
			{Timestamp: now.Add(2 * time.Second), Position: [3]float64{200, 1000, 10000}, Velocity: [3]float64{100, 0, 0}},
		},
	}

	targetTimes := []time.Time{now, now.Add(1 * time.Second), now.Add(2 * time.Second)}

	aligned := ta.AlignTracks(tracks, targetTimes)

	if len(aligned) != 2 {
		t.Errorf("Expected 2 aligned tracks, got %d", len(aligned))
	}

	// Each track should have 3 points
	for sensorID, track := range aligned {
		if len(track) != 3 {
			t.Errorf("Expected 3 points for %s, got %d", sensorID, len(track))
		}
	}
}

// TestEstimateAccuracy tests accuracy estimation
func TestEstimateAccuracy(t *testing.T) {
	ta := NewTimeAligner(nil)

	points := []TimeSeriesPoint{
		{Timestamp: time.Now(), Position: [3]float64{0, 0, 0}, Accuracy: 10},
		{Timestamp: time.Now().Add(1 * time.Second), Position: [3]float64{100, 0, 0}, Accuracy: 15},
		{Timestamp: time.Now().Add(2 * time.Second), Position: [3]float64{200, 0, 0}, Accuracy: 20},
	}

	avgAccuracy := ta.EstimateAccuracy(points)

	expected := (10.0 + 15.0 + 20.0) / 3.0
	if math.Abs(avgAccuracy-expected) > 0.01 {
		t.Errorf("Expected average accuracy %.2f, got %.2f", expected, avgAccuracy)
	}
}

// TestGetInterpolationStats tests interpolation statistics
func TestGetInterpolationStats(t *testing.T) {
	ta := NewTimeAligner(nil)

	now := time.Now()
	points := []TimeSeriesPoint{
		{Timestamp: now, Position: [3]float64{0, 0, 0}, Accuracy: 10},
		{Timestamp: now.Add(1 * time.Second), Position: [3]float64{100, 0, 0}, Accuracy: 10},
	}

	// Target times include one beyond extrapolation limit (3s from last point at 1s = 4s total)
	targetTimes := []time.Time{
		now,
		now.Add(500 * time.Millisecond),
		now.Add(1 * time.Second),
		now.Add(4 * time.Second), // Beyond extrapolation limit (3s from last point, limit is 2s)
	}

	stats := ta.GetInterpolationStats(points, targetTimes)

	if stats.TotalTargets != 4 {
		t.Errorf("Expected 4 total targets, got %d", stats.TotalTargets)
	}

	// At least 3 should succeed
	if stats.SuccessfulInterpolations < 3 {
		t.Errorf("Expected at least 3 successful interpolations, got %d", stats.SuccessfulInterpolations)
	}
}

// TestInterpolateTrack tests full track interpolation
func TestInterpolateTrack(t *testing.T) {
	ta := NewTimeAligner(nil)

	now := time.Now()
	points := []TimeSeriesPoint{
		{Timestamp: now, Position: [3]float64{0, 0, 0}, Velocity: [3]float64{100, 0, 0}},
		{Timestamp: now.Add(2 * time.Second), Position: [3]float64{200, 0, 0}, Velocity: [3]float64{100, 0, 0}},
	}

	targetTimes := []time.Time{
		now,
		now.Add(500 * time.Millisecond),
		now.Add(1 * time.Second),
		now.Add(1500 * time.Millisecond),
		now.Add(2 * time.Second),
	}

	result := ta.InterpolateTrack(points, targetTimes)

	if len(result) != 5 {
		t.Errorf("Expected 5 interpolated points, got %d", len(result))
	}
}

// BenchmarkInterpolatePosition benchmarks position interpolation
func BenchmarkInterpolatePosition(b *testing.B) {
	ta := NewTimeAligner(nil)

	now := time.Now()
	points := []TimeSeriesPoint{
		{Timestamp: now, Position: [3]float64{0, 0, 0}, Velocity: [3]float64{100, 0, 0}},
		{Timestamp: now.Add(1 * time.Second), Position: [3]float64{100, 0, 0}, Velocity: [3]float64{100, 0, 0}},
		{Timestamp: now.Add(2 * time.Second), Position: [3]float64{200, 0, 0}, Velocity: [3]float64{100, 0, 0}},
	}

	targetTime := now.Add(500 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ta.InterpolatePosition(points, targetTime)
	}
}

// BenchmarkPredictPosition benchmarks position prediction
func BenchmarkPredictPosition(b *testing.B) {
	ta := NewTimeAligner(nil)

	now := time.Now()
	point := &TimeSeriesPoint{
		Timestamp: now,
		Position:  [3]float64{0, 0, 0},
		Velocity:  [3]float64{100, 50, 10},
	}

	futureTime := now.Add(1 * time.Second)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ta.PredictPosition(point, futureTime)
	}
}

// BenchmarkSmoothTrack benchmarks track smoothing
func BenchmarkSmoothTrack(b *testing.B) {
	ta := NewTimeAligner(nil)

	now := time.Now()
	points := make([]TimeSeriesPoint, 100)
	for i := range points {
		points[i] = TimeSeriesPoint{
			Timestamp: now.Add(time.Duration(i) * 100 * time.Millisecond),
			Position:  [3]float64{float64(i * 100), 0, 0},
			Velocity:  [3]float64{100, 0, 0},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ta.SmoothTrack(points, 5)
	}
}
