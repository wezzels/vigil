package fusion

import (
	"testing"
	"time"
)

// TestScorerConfig tests scorer configuration
func TestScorerConfig(t *testing.T) {
	config := DefaultScorerConfig()

	if config.InitialScore != 0.0 {
		t.Errorf("Expected initial score 0, got %f", config.InitialScore)
	}
	if config.UpdateIncrement != 10.0 {
		t.Errorf("Expected update increment 10, got %f", config.UpdateIncrement)
	}
	if config.MissDecrement != 5.0 {
		t.Errorf("Expected miss decrement 5, got %f", config.MissDecrement)
	}
	if config.ConfirmationScore != 30.0 {
		t.Errorf("Expected confirmation score 30, got %f", config.ConfirmationScore)
	}
}

// TestNewTrackScorer tests scorer creation
func TestNewTrackScorer(t *testing.T) {
	scorer := NewTrackScorer(nil)

	if scorer == nil {
		t.Fatal("Scorer should not be nil")
	}

	if scorer.config.UpdateIncrement != 10.0 {
		t.Error("Default config should be used")
	}
}

// TestInitializeTrack tests track initialization
func TestInitializeTrack(t *testing.T) {
	scorer := NewTrackScorer(nil)

	now := time.Now()
	score := scorer.InitializeTrack(1001, now)

	if score == nil {
		t.Fatal("Score should not be nil")
	}

	if score.TrackNumber != 1001 {
		t.Errorf("Expected track number 1001, got %d", score.TrackNumber)
	}

	if score.Score != 0.0 {
		t.Errorf("Expected initial score 0, got %f", score.Score)
	}

	if score.Status != "TENTATIVE" {
		t.Errorf("Expected status TENTATIVE, got %s", score.Status)
	}
}

// TestUpdateTrack tests track update
func TestUpdateTrack(t *testing.T) {
	scorer := NewTrackScorer(nil)

	now := time.Now()
	scorer.InitializeTrack(1001, now)

	// Update with quality 0.9
	score := scorer.UpdateTrack(1001, 0.9, now.Add(1*time.Second))

	if score.Score <= 0 {
		t.Errorf("Score should increase after update, got %f", score.Score)
	}

	if score.NUpdates != 1 {
		t.Errorf("Expected 1 update, got %d", score.NUpdates)
	}

	if score.NMisses != 0 {
		t.Errorf("Expected 0 misses, got %d", score.NMisses)
	}
}

// TestMissTrack tests miss handling
func TestMissTrack(t *testing.T) {
	scorer := NewTrackScorer(nil)

	now := time.Now()
	scorer.InitializeTrack(1001, now)
	scorer.UpdateTrack(1001, 0.9, now.Add(1*time.Second))

	// Record miss
	score := scorer.MissTrack(1001, now.Add(2*time.Second))

	if score.Score < 0 {
		// Score should decrease after miss
		t.Logf("Score after miss: %f", score.Score)
	}

	if score.NMisses != 1 {
		t.Errorf("Expected 1 miss, got %d", score.NMisses)
	}
}

// TestTrackConfirmation tests track confirmation
func TestTrackConfirmation(t *testing.T) {
	config := DefaultScorerConfig()
	config.ConfirmationScore = 30.0
	scorer := NewTrackScorer(config)

	now := time.Now()
	scorer.InitializeTrack(1001, now)

	// Update multiple times
	for i := 0; i < 5; i++ {
		scorer.UpdateTrack(1001, 0.9, now.Add(time.Duration(i+1)*time.Second))
	}

	score := scorer.GetScore(1001)

	if score.Status != "CONFIRMED" {
		t.Errorf("Expected status CONFIRMED, got %s", score.Status)
	}
}

// TestDecay tests score decay
func TestDecay(t *testing.T) {
	scorer := NewTrackScorer(nil)

	now := time.Now()
	scorer.InitializeTrack(1001, now)
	scorer.UpdateTrack(1001, 0.9, now)

	initialScore := scorer.GetScore(1001).Score

	// Wait and check decay
	later := now.Add(60 * time.Second)
	score := scorer.MissTrack(1001, later)

	// Score should have decayed
	if score.Score >= initialScore {
		t.Errorf("Score should decay over time: %.2f -> %.2f", initialScore, score.Score)
	}
}

// TestConfidenceBounds tests confidence bounds
func TestConfidenceBounds(t *testing.T) {
	scorer := NewTrackScorer(nil)

	now := time.Now()
	scorer.InitializeTrack(1001, now)

	// Update to build confidence
	for i := 0; i < 5; i++ {
		scorer.UpdateTrack(1001, 0.9, now.Add(time.Duration(i+1)*time.Second))
	}

	lower, upper := scorer.GetConfidenceBounds(1001)

	if lower > upper {
		t.Errorf("Lower bound %.2f should not exceed upper %.2f", lower, upper)
	}

	if lower < 0 {
		t.Errorf("Lower bound should not be negative, got %.2f", lower)
	}

	if upper > 1.0 {
		t.Errorf("Upper bound should not exceed 1.0, got %.2f", upper)
	}
}

// TestManeuverDetection tests maneuver detection
func TestManeuverDetection(t *testing.T) {
	scorer := NewTrackScorer(nil)

	now := time.Now()
	scorer.InitializeTrack(1001, now)
	scorer.UpdateTrack(1001, 0.9, now.Add(1*time.Second))

	initialScore := scorer.GetScore(1001).Score

	// Detect maneuver
	scorer.ManeuverDetected(1001, 5.0, now.Add(2*time.Second))

	score := scorer.GetScore(1001)
	if score.Score <= initialScore {
		t.Errorf("Score should increase after maneuver: %.2f -> %.2f", initialScore, score.Score)
	}
}

// TestDropTrack tests dropping a track
func TestDropTrack(t *testing.T) {
	scorer := NewTrackScorer(nil)

	now := time.Now()
	scorer.InitializeTrack(1001, now)

	scorer.DropTrack(1001)

	score := scorer.GetScore(1001)
	if score.Status != "DROPPED" {
		t.Errorf("Expected status DROPPED, got %s", score.Status)
	}
}

// TestRemoveTrack tests removing a track
func TestRemoveTrack(t *testing.T) {
	scorer := NewTrackScorer(nil)

	now := time.Now()
	scorer.InitializeTrack(1001, now)

	scorer.RemoveTrack(1001)

	score := scorer.GetScore(1001)
	if score != nil {
		t.Error("Track should be removed")
	}
}

// TestPruneOldTracks tests pruning old tracks
func TestPruneOldTracks(t *testing.T) {
	scorer := NewTrackScorer(nil)

	now := time.Now()
	scorer.InitializeTrack(1001, now)
	scorer.InitializeTrack(1002, now)
	scorer.InitializeTrack(1003, now)

	// Make one track old
	scorer.scores[1001].LastUpdate = now.Add(-100 * time.Second)
	scorer.scores[1001].Score = -100 // Below minimum

	pruned := scorer.PruneOldTracks(50 * time.Second)

	if pruned < 1 {
		t.Errorf("Expected at least 1 track pruned, got %d", pruned)
	}
}

// TestGetTopTracks tests getting top tracks
func TestGetTopTracks(t *testing.T) {
	scorer := NewTrackScorer(nil)

	now := time.Now()

	// Create multiple tracks with different scores
	for i := 0; i < 5; i++ {
		trackNum := uint32(1001 + i)
		scorer.InitializeTrack(trackNum, now)
		// Different number of updates = different scores
		for j := 0; j <= i; j++ {
			scorer.UpdateTrack(trackNum, 0.9, now.Add(time.Duration(j+1)*time.Second))
		}
	}

	top := scorer.GetTopTracks(3)

	if len(top) != 3 {
		t.Errorf("Expected 3 top tracks, got %d", len(top))
	}

	// Should be sorted by score descending
	for i := 1; i < len(top); i++ {
		if top[i].Score > top[i-1].Score {
			t.Errorf("Tracks not sorted: [%d] %.2f > [%d] %.2f", i, top[i].Score, i-1, top[i-1].Score)
		}
	}
}

// TestStats tests scorer statistics
func TestStats(t *testing.T) {
	scorer := NewTrackScorer(nil)

	now := time.Now()
	scorer.InitializeTrack(1001, now) // TENTATIVE
	scorer.InitializeTrack(1002, now)
	scorer.InitializeTrack(1003, now)

	// Confirm 1002
	for i := 0; i < 5; i++ {
		scorer.UpdateTrack(1002, 0.9, now.Add(time.Duration(i+1)*time.Second))
	}

	// Coast 1003
	scorer.scores[1003].Status = "COASTING"

	stats := scorer.Stats()

	if stats.TotalTracks != 3 {
		t.Errorf("Expected 3 total tracks, got %d", stats.TotalTracks)
	}

	if stats.Tentative != 1 {
		t.Errorf("Expected 1 tentative, got %d", stats.Tentative)
	}

	if stats.Confirmed != 1 {
		t.Errorf("Expected 1 confirmed, got %d", stats.Confirmed)
	}

	if stats.Coasting != 1 {
		t.Errorf("Expected 1 coasting, got %d", stats.Coasting)
	}
}

// TestMultipleUpdates tests multiple rapid updates
func TestMultipleUpdates(t *testing.T) {
	scorer := NewTrackScorer(nil)

	now := time.Now()
	scorer.InitializeTrack(1001, now)

	// Rapid updates
	for i := 0; i < 100; i++ {
		scorer.UpdateTrack(1001, 0.9, now.Add(time.Duration(i)*10*time.Millisecond))
	}

	score := scorer.GetScore(1001)

	if score.NUpdates != 100 {
		t.Errorf("Expected 100 updates, got %d", score.NUpdates)
	}

	if score.Score < 0 {
		t.Errorf("Score should be positive after many updates, got %.2f", score.Score)
	}
}

// TestScoreBounds tests score bounds
func TestScoreBounds(t *testing.T) {
	config := DefaultScorerConfig()
	config.MaxScore = 100.0
	config.MinScore = -20.0
	scorer := NewTrackScorer(config)

	now := time.Now()
	scorer.InitializeTrack(1001, now)

	// Force score to exceed max
	for i := 0; i < 20; i++ {
		scorer.UpdateTrack(1001, 1.0, now.Add(time.Duration(i)*time.Second))
	}

	score := scorer.GetScore(1001)
	if score.Score > config.MaxScore {
		t.Errorf("Score %.2f exceeds max %.2f", score.Score, config.MaxScore)
	}
}

// TestConfidenceCalculation tests confidence calculation
func TestConfidenceCalculation(t *testing.T) {
	scorer := NewTrackScorer(nil)

	now := time.Now()
	scorer.InitializeTrack(1001, now)

	// Initial confidence should be 0
	score := scorer.GetScore(1001)
	if score.Confidence != 0.0 {
		t.Errorf("Initial confidence should be 0, got %.2f", score.Confidence)
	}

	// Update to build confidence
	for i := 0; i < 10; i++ {
		scorer.UpdateTrack(1001, 0.9, now.Add(time.Duration(i+1)*time.Second))
	}

	score = scorer.GetScore(1001)
	if score.Confidence <= 0 {
		t.Errorf("Confidence should increase with updates, got %.2f", score.Confidence)
	}
}

// BenchmarkUpdateTrack benchmarks track update
func BenchmarkUpdateTrack(b *testing.B) {
	scorer := NewTrackScorer(nil)
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trackNum := uint32(1000 + i%100)
		scorer.UpdateTrack(trackNum, 0.9, now.Add(time.Duration(i)*time.Millisecond))
	}
}

// BenchmarkMissTrack benchmarks miss handling
func BenchmarkMissTrack(b *testing.B) {
	scorer := NewTrackScorer(nil)
	now := time.Now()

	// Initialize some tracks
	for i := 0; i < 100; i++ {
		trackNum := uint32(1000 + i)
		scorer.InitializeTrack(trackNum, now)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trackNum := uint32(1000 + i%100)
		scorer.MissTrack(trackNum, now.Add(time.Duration(i)*time.Millisecond))
	}
}

// BenchmarkGetTopTracks benchmarks getting top tracks
func BenchmarkGetTopTracks(b *testing.B) {
	scorer := NewTrackScorer(nil)
	now := time.Now()

	// Initialize many tracks
	for i := 0; i < 100; i++ {
		trackNum := uint32(1000 + i)
		scorer.InitializeTrack(trackNum, now)
		for j := 0; j < i%10; j++ {
			scorer.UpdateTrack(trackNum, 0.9, now.Add(time.Duration(j+1)*time.Second))
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scorer.GetTopTracks(10)
	}
}
