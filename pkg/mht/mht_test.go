package mht

import (
	"container/heap"
	"testing"
	"time"
)

// TestMHTConfig tests MHT configuration
func TestMHTConfig(t *testing.T) {
	config := DefaultMHTConfig()
	
	if config.MaxHypotheses != 100 {
		t.Errorf("Expected max hypotheses 100, got %d", config.MaxHypotheses)
	}
	if config.MaxDepth != 5 {
		t.Errorf("Expected max depth 5, got %d", config.MaxDepth)
	}
	if config.AssociationScore != 10.0 {
		t.Errorf("Expected association score 10, got %f", config.AssociationScore)
	}
	if config.MaxMisses != 5 {
		t.Errorf("Expected max misses 5, got %d", config.MaxMisses)
	}
}

// TestNewMHTTracker tests MHT tracker creation
func TestNewMHTTracker(t *testing.T) {
	tracker := NewMHTTracker(nil)
	
	if tracker == nil {
		t.Fatal("Tracker should not be nil")
	}
	
	if tracker.config.MaxHypotheses != 100 {
		t.Error("Default config should be used")
	}
}

// TestProcessMeasurements tests processing measurements
func TestProcessMeasurements(t *testing.T) {
	tracker := NewMHTTracker(nil)
	
	measurements := []Measurement{
		{
			ID:        1,
			SourceID:   "TPY2-1",
			Position:   [3]float64{1000, 0, 10000},
			Variance:   [3]float64{100, 100, 100},
			Timestamp:  time.Now(),
			Quality:    0.9,
		},
	}
	
	hypotheses := tracker.ProcessMeasurements(measurements)
	
	if len(hypotheses) != 1 {
		t.Errorf("Expected 1 new hypothesis, got %d", len(hypotheses))
	}
	
	stats := tracker.Stats()
	if stats.TotalHypotheses != 1 {
		t.Errorf("Expected 1 total hypothesis, got %d", stats.TotalHypotheses)
	}
}

// TestHypothesisCreation tests hypothesis creation
func TestHypothesisCreation(t *testing.T) {
	tracker := NewMHTTracker(nil)
	
	meas := Measurement{
		ID:        1,
		SourceID:   "TPY2-1",
		Position:   [3]float64{1000, 0, 10000},
		Variance:   [3]float64{100, 100, 100},
		Timestamp:  time.Now(),
		Quality:    0.9,
	}
	
	tracker.ProcessMeasurements([]Measurement{meas})
	
	hyps := tracker.GetAllHypotheses()
	if len(hyps) != 1 {
		t.Fatalf("Expected 1 hypothesis, got %d", len(hyps))
	}
	
	hyp := hyps[0]
	if hyp.TrackNumber < 1000 {
		t.Errorf("Track number should be >= 1000, got %d", hyp.TrackNumber)
	}
	if hyp.NMisses != 0 {
		t.Errorf("New hypothesis should have 0 misses, got %d", hyp.NMisses)
	}
	if len(hyp.Associations) != 1 {
		t.Errorf("New hypothesis should have 1 association, got %d", len(hyp.Associations))
	}
}

// TestTrackPrediction tests track prediction
func TestTrackPrediction(t *testing.T) {
	tracker := NewMHTTracker(nil)
	
	track := &TrackState{
		ID:          1001,
		Position:    [3]float64{0, 0, 0},
		Velocity:    [3]float64{100, 0, 0}, // 100 m/s in x
		Acceleration: [3]float64{0, 0, 0},
	}
	
	dt := 1 * time.Second
	predicted := tracker.predictTrack(track, dt)
	
	// Should move 100m in x direction
	if predicted.Position[0] != 100.0 {
		t.Errorf("Expected x=100, got %.2f", predicted.Position[0])
	}
	
	// Velocity should remain constant
	if predicted.Velocity[0] != 100.0 {
		t.Errorf("Velocity should remain 100, got %.2f", predicted.Velocity[0])
	}
}

// TestTrackUpdate tests track update with measurement
func TestTrackUpdate(t *testing.T) {
	tracker := NewMHTTracker(nil)
	
	track := &TrackState{
		ID:          1001,
		Position:    [3]float64{0, 0, 0},
		Velocity:    [3]float64{100, 0, 0},
		Covariance:  [6][6]float64{
			{100, 0, 0, 0, 0, 0},
			{0, 100, 0, 0, 0, 0},
			{0, 0, 100, 0, 0, 0},
		},
		LastMeasurement: time.Now(),
		NUpdates:        1,
	}
	
	meas := Measurement{
		ID:        1,
		SourceID:   "TPY2-1",
		Position:   [3]float64{105, 0, 0}, // 5m from predicted
		Variance:   [3]float64{10, 10, 10},
		Timestamp:  time.Now(),
	}
	
	updated := tracker.updateTrackWithMeasurement(track, meas, time.Now())
	
	// Position should move toward measurement
	if updated.Position[0] <= 0 || updated.Position[0] >= 105 {
		t.Errorf("Position should be between 0 and 105, got %.2f", updated.Position[0])
	}
	
	if updated.NUpdates != 2 {
		t.Errorf("Expected 2 updates, got %d", updated.NUpdates)
	}
}

// TestAssociationScore tests association scoring
func TestAssociationScore(t *testing.T) {
	tracker := NewMHTTracker(nil)
	
	track := &TrackState{
		ID:          1001,
		Position:    [3]float64{0, 0, 0},
		Velocity:    [3]float64{0, 0, 0},
		Covariance:  [6][6]float64{
			{100, 0, 0, 0, 0, 0},
			{0, 100, 0, 0, 0, 0},
			{0, 0, 100, 0, 0, 0},
		},
	}
	
	// Close measurement
	closeMeas := Measurement{
		ID:        1,
		Position:   [3]float64{10, 0, 0}, // 10m away
		Variance:   [3]float64{10, 10, 10},
	}
	
	closeScore := tracker.calculateAssociationScore(track, closeMeas)
	
	// Far measurement
	farMeas := Measurement{
		ID:        2,
		Position:   [3]float64{1000, 0, 0}, // 1000m away
		Variance:   [3]float64{10, 10, 10},
	}
	
	farScore := tracker.calculateAssociationScore(track, farMeas)
	
	// Close measurement should have higher score
	if closeScore <= farScore {
		t.Errorf("Close measurement should have higher score: %.2f vs %.2f", closeScore, farScore)
	}
}

// TestHypothesisPruning tests hypothesis pruning
func TestHypothesisPruning(t *testing.T) {
	config := DefaultMHTConfig()
	config.MaxHypotheses = 5
	config.MinScore = -50.0
	tracker := NewMHTTracker(config)
	
	// Create many hypotheses
	for i := 0; i < 10; i++ {
		meas := Measurement{
			ID:        uint64(i + 1),
			SourceID:   "TPY2-1",
			Position:   [3]float64{float64(i * 1000), 0, 0}, // Spread out
			Variance:   [3]float64{100, 100, 100},
			Timestamp:  time.Now(),
			Quality:    0.9,
		}
		tracker.ProcessMeasurements([]Measurement{meas})
	}
	
	stats := tracker.Stats()
	
	// Should have pruned to max hypotheses
	if stats.TotalHypotheses > config.MaxHypotheses {
		t.Errorf("Expected at most %d hypotheses, got %d", config.MaxHypotheses, stats.TotalHypotheses)
	}
}

// TestMissHypothesis tests missed detection handling
func TestMissHypothesis(t *testing.T) {
	tracker := NewMHTTracker(nil)
	
	// Create initial track
	meas := Measurement{
		ID:        1,
		SourceID:   "TPY2-1",
		Position:   [3]float64{0, 0, 0},
		Variance:   [3]float64{100, 100, 100},
		Timestamp:  time.Now(),
	}
	
	tracker.ProcessMeasurements([]Measurement{meas})
	
	// Process empty measurements (miss)
	tracker.ProcessMeasurements([]Measurement{})
	
	// Check if there's a hypothesis with misses
	hyps := tracker.GetAllHypotheses()
	found := false
	for _, hyp := range hyps {
		if hyp.NMisses > 0 {
			found = true
			if hyp.Score >= 0 {
				t.Errorf("Miss hypothesis should have lower score, got %.2f", hyp.Score)
			}
		}
	}
	
	if !found {
		t.Log("No miss hypothesis found (may have been pruned)")
	}
}

// TestMultiMeasurementAssociation tests multiple measurements
func TestMultiMeasurementAssociation(t *testing.T) {
	tracker := NewMHTTracker(nil)
	
	// Create initial track
	meas1 := Measurement{
		ID:        1,
		SourceID:   "TPY2-1",
		Position:   [3]float64{0, 0, 0},
		Variance:   [3]float64{100, 100, 100},
		Timestamp:  time.Now(),
	}
	
	tracker.ProcessMeasurements([]Measurement{meas1})
	
	// Process multiple nearby measurements
	meas2 := Measurement{
		ID:        2,
		SourceID:   "TPY2-1",
		Position:   [3]float64{10, 0, 0}, // Close
		Variance:   [3]float64{100, 100, 100},
		Timestamp:  time.Now(),
	}
	
	meas3 := Measurement{
		ID:        3,
		SourceID:   "SBX-1",
		Position:   [3]float64{15, 0, 0}, // Also close
		Variance:   [3]float64{100, 100, 100},
		Timestamp:  time.Now(),
	}
	
	hyps := tracker.ProcessMeasurements([]Measurement{meas2, meas3})
	
	// Should create multiple hypotheses
	if len(hyps) < 2 {
		t.Errorf("Expected at least 2 hypotheses, got %d", len(hyps))
	}
}

// TestGetBestHypotheses tests getting best hypotheses
func TestGetBestHypotheses(t *testing.T) {
	tracker := NewMHTTracker(nil)
	
	// Create two tracks
	meas1 := Measurement{
		ID:        1,
		SourceID:   "TPY2-1",
		Position:   [3]float64{0, 0, 0},
		Variance:   [3]float64{100, 100, 100},
		Timestamp:  time.Now(),
	}
	
	meas2 := Measurement{
		ID:        2,
		SourceID:   "SBX-1",
		Position:   [3]float64{1000, 0, 0}, // Far from first
		Variance:   [3]float64{100, 100, 100},
		Timestamp:  time.Now(),
	}
	
	tracker.ProcessMeasurements([]Measurement{meas1, meas2})
	
	best := tracker.GetBestHypotheses()
	
	if len(best) != 2 {
		t.Errorf("Expected 2 best hypotheses, got %d", len(best))
	}
}

// TestHypothesisDepth tests hypothesis depth limits
func TestHypothesisDepth(t *testing.T) {
	config := DefaultMHTConfig()
	config.MaxDepth = 3
	tracker := NewMHTTracker(config)
	
	meas := Measurement{
		ID:        1,
		SourceID:   "TPY2-1",
		Position:   [3]float64{0, 0, 0},
		Variance:   [3]float64{100, 100, 100},
		Timestamp:  time.Now(),
	}
	
	// Process multiple times
	for i := 0; i < 10; i++ {
		tracker.ProcessMeasurements([]Measurement{meas})
	}
	
	// Should have pruned deep hypotheses
	for _, hyp := range tracker.GetAllHypotheses() {
		if hyp.Depth > config.MaxDepth {
			t.Errorf("Hypothesis depth %d exceeds max %d", hyp.Depth, config.MaxDepth)
		}
	}
}

// TestTrackConfirmation tests track confirmation
func TestTrackConfirmation(t *testing.T) {
	tracker := NewMHTTracker(nil)
	
	meas := Measurement{
		ID:        1,
		SourceID:   "TPY2-1",
		Position:   [3]float64{0, 0, 0},
		Variance:   [3]float64{100, 100, 100},
		Timestamp:  time.Now(),
	}
	
	// Process 3 times to confirm track
	for i := 0; i < 3; i++ {
		tracker.ProcessMeasurements([]Measurement{meas})
	}
	
	best := tracker.GetBestHypotheses()
	
	for trackNum, hyp := range best {
		_ = trackNum
		if hyp.Track.Status != "CONFIRMED" {
			t.Errorf("Track status should be CONFIRMED after 3 updates, got %s", hyp.Track.Status)
		}
	}
}

// TestHypothesisHeap tests hypothesis heap
func TestHypothesisHeap(t *testing.T) {
	h := &HypothesisHeap{}
	
	// Push to empty heap
	hyp1 := &Hypothesis{ID: 1, Score: 10.0}
	hyp2 := &Hypothesis{ID: 2, Score: 5.0}
	hyp3 := &Hypothesis{ID: 3, Score: 20.0}
	hyp4 := &Hypothesis{ID: 4, Score: 15.0}
	
	heap.Push(h, hyp1)
	heap.Push(h, hyp2)
	heap.Push(h, hyp3)
	heap.Push(h, hyp4)
	
	// Pop should return highest score first
	top := heap.Pop(h).(*Hypothesis)
	if top.Score != 20.0 {
		t.Errorf("Expected top score 20.0, got %.2f", top.Score)
	}
	
	top = heap.Pop(h).(*Hypothesis)
	if top.Score != 15.0 {
		t.Errorf("Expected second score 15.0, got %.2f", top.Score)
	}
}

// BenchmarkMHTProcessing benchmarks MHT processing
func BenchmarkMHTProcessing(b *testing.B) {
	tracker := NewMHTTracker(nil)
	
	measurements := make([]Measurement, 10)
	for i := range measurements {
		measurements[i] = Measurement{
			ID:        uint64(i + 1),
			SourceID:   "TPY2-1",
			Position:   [3]float64{float64(i * 100), float64(i * 100), 10000},
			Variance:   [3]float64{100, 100, 100},
			Timestamp:  time.Now(),
			Quality:    0.9,
		}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.ProcessMeasurements(measurements)
	}
}

// BenchmarkAssociationScore benchmarks association scoring
func BenchmarkAssociationScore(b *testing.B) {
	tracker := NewMHTTracker(nil)
	
	track := &TrackState{
		ID:          1001,
		Position:    [3]float64{0, 0, 0},
		Velocity:    [3]float64{100, 0, 0},
		Covariance:  [6][6]float64{
			{100, 0, 0, 0, 0, 0},
			{0, 100, 0, 0, 0, 0},
			{0, 0, 100, 0, 0, 0},
		},
	}
	
	meas := Measurement{
		ID:        1,
		Position:   [3]float64{10, 0, 0},
		Variance:   [3]float64{10, 10, 10},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.calculateAssociationScore(track, meas)
	}
}

// BenchmarkTrackPrediction benchmarks track prediction
func BenchmarkTrackPrediction(b *testing.B) {
	tracker := NewMHTTracker(nil)
	
	track := &TrackState{
		ID:          1001,
		Position:    [3]float64{0, 0, 0},
		Velocity:    [3]float64{100, 50, 10},
	}
	
	dt := 1 * time.Second
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.predictTrack(track, dt)
	}
}