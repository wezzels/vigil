// Package fusion provides track scoring for sensor fusion
package fusion

import (
	"math"
	"sync"
	"time"
)

// TrackScorer calculates and maintains track scores
type TrackScorer struct {
	config *ScorerConfig
	scores map[uint32]*TrackScore
	mu     sync.RWMutex
}

// ScorerConfig holds configuration for track scoring
type ScorerConfig struct {
	// Score parameters
	InitialScore      float64 `json:"initial_score"`      // Score for new tracks
	UpdateIncrement   float64 `json:"update_increment"`   // Score increase per update
	MissDecrement     float64 `json:"miss_decrement"`     // Score decrease per miss
	MaxScore          float64 `json:"max_score"`          // Maximum score
	MinScore          float64 `json:"min_score"`          // Minimum score before drop
	ConfirmationScore float64 `json:"confirmation_score"` // Score to confirm track

	// Decay parameters
	DecayRate      float64       `json:"decay_rate"`      // Exponential decay rate per second
	MinDecayTime   time.Duration `json:"min_decay_time"`  // Minimum time before decay
	QualityWeight  float64       `json:"quality_weight"`  // Weight for measurement quality
	ManeuverWeight float64       `json:"maneuver_weight"` // Weight for maneuver detection

	// Confidence bounds
	ConfidenceFactor float64 `json:"confidence_factor"` // Factor for confidence calculation
	MinConfidence    float64 `json:"min_confidence"`    // Minimum confidence threshold
	MaxConfidence    float64 `json:"max_confidence"`    // Maximum confidence (1.0)
}

// TrackScore represents the score of a track
type TrackScore struct {
	TrackNumber   uint32    `json:"track_number"`
	Score         float64   `json:"score"`
	Confidence    float64   `json:"confidence"`
	NUpdates      int       `json:"n_updates"`
	NMisses       int       `json:"n_misses"`
	FirstUpdate   time.Time `json:"first_update"`
	LastUpdate    time.Time `json:"last_update"`
	LastScore     float64   `json:"last_score"`     // Score before last update
	QualitySum    float64   `json:"quality_sum"`    // Sum of measurement qualities
	ManeuverScore float64   `json:"maneuver_score"` // Maneuver detection score
	Status        string    `json:"status"`
}

// DefaultScorerConfig returns default scorer configuration
func DefaultScorerConfig() *ScorerConfig {
	return &ScorerConfig{
		InitialScore:      0.0,
		UpdateIncrement:   10.0,
		MissDecrement:     5.0,
		MaxScore:          100.0,
		MinScore:          -20.0,
		ConfirmationScore: 30.0,
		DecayRate:         0.01, // 1% per second
		MinDecayTime:      5 * time.Second,
		QualityWeight:     0.3,
		ManeuverWeight:    0.2,
		ConfidenceFactor:  0.5,
		MinConfidence:     0.1,
		MaxConfidence:     0.99,
	}
}

// NewTrackScorer creates a new track scorer
func NewTrackScorer(config *ScorerConfig) *TrackScorer {
	if config == nil {
		config = DefaultScorerConfig()
	}

	return &TrackScorer{
		config: config,
		scores: make(map[uint32]*TrackScore),
	}
}

// InitializeTrack initializes score for a new track
func (ts *TrackScorer) InitializeTrack(trackNumber uint32, now time.Time) *TrackScore {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	score := &TrackScore{
		TrackNumber:   trackNumber,
		Score:         ts.config.InitialScore,
		Confidence:    0.0,
		NUpdates:      0,
		NMisses:       0,
		FirstUpdate:   now,
		LastUpdate:    now,
		LastScore:     ts.config.InitialScore,
		QualitySum:    0.0,
		ManeuverScore: 0.0,
		Status:        "TENTATIVE",
	}

	ts.scores[trackNumber] = score
	return score
}

// UpdateTrack updates score for a track with measurement
func (ts *TrackScorer) UpdateTrack(trackNumber uint32, quality float64, now time.Time) *TrackScore {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	score, exists := ts.scores[trackNumber]
	if !exists {
		score = &TrackScore{
			TrackNumber: trackNumber,
			Score:       ts.config.InitialScore,
			FirstUpdate: now,
			LastUpdate:  now,
			Status:      "TENTATIVE",
		}
		ts.scores[trackNumber] = score
	}

	// Apply decay based on time since last update
	timeDiff := now.Sub(score.LastUpdate)
	decay := ts.calculateDecay(timeDiff)

	// Calculate new score
	newScore := score.Score - decay
	newScore += ts.config.UpdateIncrement
	newScore += quality * ts.config.QualityWeight * ts.config.UpdateIncrement

	// Clamp to bounds
	newScore = math.Max(ts.config.MinScore, math.Min(ts.config.MaxScore, newScore))

	// Update track
	score.LastScore = score.Score
	score.Score = newScore
	score.NUpdates++
	score.NMisses = 0
	score.LastUpdate = now
	score.QualitySum += quality

	// Update confidence
	score.Confidence = ts.calculateConfidence(score)

	// Update status
	if score.Score >= ts.config.ConfirmationScore && score.NUpdates >= 3 {
		score.Status = "CONFIRMED"
	}

	return score
}

// MissTrack updates score for a missed track
func (ts *TrackScorer) MissTrack(trackNumber uint32, now time.Time) *TrackScore {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	score, exists := ts.scores[trackNumber]
	if !exists {
		return nil
	}

	// Apply decay
	timeDiff := now.Sub(score.LastUpdate)
	decay := ts.calculateDecay(timeDiff)

	// Calculate new score
	newScore := score.Score - decay
	newScore -= ts.config.MissDecrement

	// Clamp to bounds
	newScore = math.Max(ts.config.MinScore, newScore)

	// Update track
	score.LastScore = score.Score
	score.Score = newScore
	score.NMisses++
	score.LastUpdate = now

	// Update confidence
	score.Confidence = ts.calculateConfidence(score)

	// Update status
	if score.Score < ts.config.ConfirmationScore {
		if score.NMisses >= 3 {
			score.Status = "TENTATIVE"
		}
	}
	if score.Score < 0 {
		score.Status = "COASTING"
	}

	return score
}

// ManeuverDetected updates score for detected maneuver
func (ts *TrackScorer) ManeuverDetected(trackNumber uint32, magnitude float64, now time.Time) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	score, exists := ts.scores[trackNumber]
	if !exists {
		return
	}

	// Increase score for maneuver detection (indicates real target)
	maneuverBonus := magnitude * ts.config.ManeuverWeight
	score.ManeuverScore += maneuverBonus
	score.Score = math.Min(ts.config.MaxScore, score.Score+maneuverBonus)
}

// calculateDecay calculates score decay based on time
func (ts *TrackScorer) calculateDecay(timeDiff time.Duration) float64 {
	if timeDiff < ts.config.MinDecayTime {
		return 0.0
	}

	seconds := timeDiff.Seconds()
	// Exponential decay
	decay := ts.config.DecayRate * seconds
	return decay
}

// calculateConfidence calculates confidence based on track history
func (ts *TrackScorer) calculateConfidence(score *TrackScore) float64 {
	// Confidence based on:
	// 1. Number of updates
	// 2. Score value
	// 3. Quality sum
	// 4. Time since first update

	if score.NUpdates == 0 {
		return 0.0
	}

	// Update-based confidence
	updateConf := math.Min(1.0, float64(score.NUpdates)/10.0)

	// Score-based confidence
	scoreConf := (score.Score - ts.config.MinScore) / (ts.config.MaxScore - ts.config.MinScore)
	scoreConf = math.Max(0.0, math.Min(1.0, scoreConf))

	// Quality-based confidence
	qualityConf := 0.0
	if score.NUpdates > 0 {
		qualityConf = score.QualitySum / float64(score.NUpdates)
	}

	// Weighted combination
	confidence := ts.config.ConfidenceFactor * (updateConf + scoreConf + qualityConf) / 3.0

	return math.Max(ts.config.MinConfidence, math.Min(ts.config.MaxConfidence, confidence))
}

// GetScore returns score for a track
func (ts *TrackScorer) GetScore(trackNumber uint32) *TrackScore {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.scores[trackNumber]
}

// GetAllScores returns all track scores
func (ts *TrackScorer) GetAllScores() []*TrackScore {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	scores := make([]*TrackScore, 0, len(ts.scores))
	for _, score := range ts.scores {
		scores = append(scores, score)
	}
	return scores
}

// GetActiveScores returns scores for active tracks
func (ts *TrackScorer) GetActiveScores() []*TrackScore {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	scores := make([]*TrackScore, 0)
	for _, score := range ts.scores {
		if score.Status != "DROPPED" && score.Score >= ts.config.MinScore {
			scores = append(scores, score)
		}
	}
	return scores
}

// DropTrack marks a track as dropped
func (ts *TrackScorer) DropTrack(trackNumber uint32) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if score, exists := ts.scores[trackNumber]; exists {
		score.Status = "DROPPED"
	}
}

// RemoveTrack removes a track from scoring
func (ts *TrackScorer) RemoveTrack(trackNumber uint32) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	delete(ts.scores, trackNumber)
}

// PruneOldTracks removes tracks below minimum score
func (ts *TrackScorer) PruneOldTracks(maxAge time.Duration) int {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	now := time.Now()
	count := 0

	for trackNum, score := range ts.scores {
		if score.Score < ts.config.MinScore {
			delete(ts.scores, trackNum)
			count++
			continue
		}

		if now.Sub(score.LastUpdate) > maxAge {
			delete(ts.scores, trackNum)
			count++
		}
	}

	return count
}

// Stats returns scorer statistics
func (ts *TrackScorer) Stats() ScorerStats {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	tentative := 0
	confirmed := 0
	coasting := 0
	dropped := 0

	for _, score := range ts.scores {
		switch score.Status {
		case "TENTATIVE":
			tentative++
		case "CONFIRMED":
			confirmed++
		case "COASTING":
			coasting++
		case "DROPPED":
			dropped++
		}
	}

	return ScorerStats{
		TotalTracks: len(ts.scores),
		Tentative:   tentative,
		Confirmed:   confirmed,
		Coasting:    coasting,
		Dropped:     dropped,
	}
}

// ScorerStats holds scorer statistics
type ScorerStats struct {
	TotalTracks int `json:"total_tracks"`
	Tentative   int `json:"tentative"`
	Confirmed   int `json:"confirmed"`
	Coasting    int `json:"coasting"`
	Dropped     int `json:"dropped"`
}

// GetTopTracks returns top N tracks by score
func (ts *TrackScorer) GetTopTracks(n int) []*TrackScore {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	// Simple selection sort for top N
	scores := make([]*TrackScore, 0, len(ts.scores))
	for _, score := range ts.scores {
		if score.Status != "DROPPED" {
			scores = append(scores, score)
		}
	}

	// Sort by score (descending)
	for i := 0; i < len(scores) && i < n; i++ {
		maxIdx := i
		for j := i + 1; j < len(scores); j++ {
			if scores[j].Score > scores[maxIdx].Score {
				maxIdx = j
			}
		}
		if maxIdx != i {
			scores[i], scores[maxIdx] = scores[maxIdx], scores[i]
		}
	}

	if len(scores) > n {
		return scores[:n]
	}
	return scores
}

// GetConfidenceBounds returns confidence bounds for a track
func (ts *TrackScorer) GetConfidenceBounds(trackNumber uint32) (lower, upper float64) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	score, exists := ts.scores[trackNumber]
	if !exists {
		return 0.0, 1.0
	}

	// Lower bound: confidence minus factor
	lower = score.Confidence - ts.config.ConfidenceFactor
	lower = math.Max(ts.config.MinConfidence, lower)

	// Upper bound: confidence plus factor
	upper = score.Confidence + ts.config.ConfidenceFactor
	upper = math.Min(ts.config.MaxConfidence, upper)

	return lower, upper
}
