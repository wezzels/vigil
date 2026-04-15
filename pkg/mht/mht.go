// Package mht implements Multi-Hypothesis Tracking for sensor fusion
// MHT maintains multiple track hypotheses and prunes low-probability hypotheses
// Reference: "Multi-Target Multi-Hypothesis Tracking" by Reid (1979)
package mht

import (
	"container/heap"
	"sync"
	"time"
)

// Hypothesis represents a single track hypothesis
type Hypothesis struct {
	ID           uint64        `json:"id"`
	TrackNumber  uint32        `json:"track_number"`
	ParentID     uint64        `json:"parent_id"` // Parent hypothesis
	Children     []uint64      `json:"children"`  // Child hypotheses
	Track        *TrackState   `json:"track"`
	Associations []Association `json:"associations"` // Measurement associations
	Score        float64       `json:"score"`        // Hypothesis score (log-likelihood)
	Probability  float64       `json:"probability"`  // Normalized probability
	Depth        int           `json:"depth"`        // Depth in hypothesis tree
	CreatedAt    time.Time     `json:"created_at"`
	LastUpdate   time.Time     `json:"last_update"`
	NMisses      int           `json:"n_misses"` // Consecutive missed updates
}

// TrackState represents the state of a track
type TrackState struct {
	ID              uint32        `json:"id"`
	Position        [3]float64    `json:"position"`     // x, y, z (meters)
	Velocity        [3]float64    `json:"velocity"`     // vx, vy, vz (m/s)
	Acceleration    [3]float64    `json:"acceleration"` // ax, ay, az (m/s²)
	Covariance      [6][6]float64 `json:"covariance"`   // State covariance
	LastMeasurement time.Time     `json:"last_measurement"`
	NUpdates        int           `json:"n_updates"`
	Status          string        `json:"status"`
}

// Association represents a measurement association
type Association struct {
	MeasurementID uint64      `json:"measurement_id"`
	Measurement   Measurement `json:"measurement"`
	Timestamp     time.Time   `json:"timestamp"`
	Score         float64     `json:"score"`
}

// Measurement represents a sensor measurement
type Measurement struct {
	ID        uint64     `json:"id"`
	SourceID  string     `json:"source_id"`
	Position  [3]float64 `json:"position"`
	Variance  [3]float64 `json:"variance"`
	Timestamp time.Time  `json:"timestamp"`
	Quality   float64    `json:"quality"`
}

// MHTConfig holds configuration for MHT
type MHTConfig struct {
	// Hypothesis management
	MaxHypotheses int           `json:"max_hypotheses"` // Maximum hypotheses per track
	MaxDepth      int           `json:"max_depth"`      // Maximum depth of hypothesis tree
	MinScore      float64       `json:"min_score"`      // Minimum score to keep hypothesis
	PruneInterval time.Duration `json:"prune_interval"` // Interval between pruning

	// Scoring
	AssociationScore float64 `json:"association_score"` // Score for successful association
	MissScore        float64 `json:"miss_score"`        // Score penalty for missed update
	FalseAlarmRate   float64 `json:"false_alarm_rate"`  // False alarm probability
	NewTrackScore    float64 `json:"new_track_score"`   // Score for new track initiation

	// Track management
	MaxMisses             int           `json:"max_misses"`             // Max misses before track drop
	CoastTime             time.Duration `json:"coast_time"`             // Time before coasting
	DropTime              time.Duration `json:"drop_time"`              // Time before dropping
	ConfirmationThreshold float64       `json:"confirmation_threshold"` // Score threshold for confirmation
}

// DefaultMHTConfig returns default MHT configuration
func DefaultMHTConfig() *MHTConfig {
	return &MHTConfig{
		MaxHypotheses:         100,
		MaxDepth:              5,
		MinScore:              -100.0,
		PruneInterval:         1 * time.Second,
		AssociationScore:      10.0,
		MissScore:             -5.0,
		FalseAlarmRate:        0.01,
		NewTrackScore:         0.0,
		MaxMisses:             5,
		CoastTime:             10 * time.Second,
		DropTime:              30 * time.Second,
		ConfirmationThreshold: 20.0,
	}
}

// MHTTracker implements Multi-Hypothesis Tracking
type MHTTracker struct {
	config       *MHTConfig
	hypotheses   map[uint64]*Hypothesis
	trackStates  map[uint32]*TrackState
	nextHypID    uint64
	nextTrackNum uint32
	mu           sync.RWMutex
}

// NewMHTTracker creates a new MHT tracker
func NewMHTTracker(config *MHTConfig) *MHTTracker {
	if config == nil {
		config = DefaultMHTConfig()
	}

	return &MHTTracker{
		config:       config,
		hypotheses:   make(map[uint64]*Hypothesis),
		trackStates:  make(map[uint32]*TrackState),
		nextHypID:    1,
		nextTrackNum: 1000,
	}
}

// ProcessMeasurements processes a batch of measurements
func (mht *MHTTracker) ProcessMeasurements(measurements []Measurement) []*Hypothesis {
	mht.mu.Lock()
	defer mht.mu.Unlock()

	now := time.Now()
	newHypotheses := make([]*Hypothesis, 0)

	// For each existing hypothesis, generate child hypotheses
	for _, hyp := range mht.hypotheses {
		children := mht.generateChildHypotheses(hyp, measurements, now)
		newHypotheses = append(newHypotheses, children...)
	}

	// Add new track hypotheses for unassociated measurements
	for _, meas := range measurements {
		if !mht.isMeasurementAssociated(meas.ID) {
			newHyp := mht.createNewTrackHypothesis(meas, now)
			newHypotheses = append(newHypotheses, newHyp)
		}
	}

	// Add all new hypotheses
	for _, hyp := range newHypotheses {
		mht.hypotheses[hyp.ID] = hyp

		// Link to parent
		if hyp.ParentID > 0 {
			if parent, exists := mht.hypotheses[hyp.ParentID]; exists {
				parent.Children = append(parent.Children, hyp.ID)
			}
		}
	}

	// Prune hypotheses
	mht.pruneHypotheses()

	return newHypotheses
}

// generateChildHypotheses generates child hypotheses for a parent hypothesis
func (mht *MHTTracker) generateChildHypotheses(parent *Hypothesis, measurements []Measurement, now time.Time) []*Hypothesis {
	children := make([]*Hypothesis, 0, len(measurements)+1)

	// Hypothesis for missed detection
	missHyp := mht.createMissHypothesis(parent, now)
	children = append(children, missHyp)

	// Hypothesis for each measurement association
	for _, meas := range measurements {
		score := mht.calculateAssociationScore(parent.Track, meas)
		if score >= mht.config.MinScore {
			assocHyp := mht.createAssociationHypothesis(parent, meas, score, now)
			children = append(children, assocHyp)
		}
	}

	return children
}

// createMissHypothesis creates a hypothesis for missed detection
func (mht *MHTTracker) createMissHypothesis(parent *Hypothesis, now time.Time) *Hypothesis {
	hypID := mht.nextHypID
	mht.nextHypID++

	newTrack := mht.predictTrack(parent.Track, now.Sub(parent.LastUpdate))

	return &Hypothesis{
		ID:           hypID,
		TrackNumber:  parent.TrackNumber,
		ParentID:     parent.ID,
		Children:     make([]uint64, 0),
		Track:        newTrack,
		Associations: parent.Associations,
		Score:        parent.Score + mht.config.MissScore,
		Depth:        parent.Depth + 1,
		CreatedAt:    now,
		LastUpdate:   now,
		NMisses:      parent.NMisses + 1,
	}
}

// createAssociationHypothesis creates a hypothesis for measurement association
func (mht *MHTTracker) createAssociationHypothesis(parent *Hypothesis, meas Measurement, score float64, now time.Time) *Hypothesis {
	hypID := mht.nextHypID
	mht.nextHypID++

	// Update track with measurement
	newTrack := mht.updateTrackWithMeasurement(parent.Track, meas, now)

	// Copy associations
	assocs := make([]Association, len(parent.Associations))
	copy(assocs, parent.Associations)
	assocs = append(assocs, Association{
		MeasurementID: meas.ID,
		Measurement:   meas,
		Timestamp:     now,
		Score:         score,
	})

	return &Hypothesis{
		ID:           hypID,
		TrackNumber:  parent.TrackNumber,
		ParentID:     parent.ID,
		Children:     make([]uint64, 0),
		Track:        newTrack,
		Associations: assocs,
		Score:        parent.Score + score + mht.config.AssociationScore,
		Depth:        parent.Depth + 1,
		CreatedAt:    now,
		LastUpdate:   now,
		NMisses:      0,
	}
}

// createNewTrackHypothesis creates a hypothesis for a new track
func (mht *MHTTracker) createNewTrackHypothesis(meas Measurement, now time.Time) *Hypothesis {
	hypID := mht.nextHypID
	mht.nextHypID++

	trackNum := mht.nextTrackNum
	mht.nextTrackNum++

	track := &TrackState{
		ID:              trackNum,
		Position:        meas.Position,
		Velocity:        [3]float64{0, 0, 0},
		Acceleration:    [3]float64{0, 0, 0},
		LastMeasurement: now,
		NUpdates:        1,
		Status:          "TENTATIVE",
	}

	// Initialize covariance from measurement variance
	for i := 0; i < 3; i++ {
		track.Covariance[i][i] = meas.Variance[i]
	}
	for i := 3; i < 6; i++ {
		track.Covariance[i][i] = 100.0 // Initial velocity variance
	}

	return &Hypothesis{
		ID:          hypID,
		TrackNumber: trackNum,
		ParentID:    0,
		Children:    make([]uint64, 0),
		Track:       track,
		Associations: []Association{{
			MeasurementID: meas.ID,
			Measurement:   meas,
			Timestamp:     now,
			Score:         mht.config.NewTrackScore,
		}},
		Score:      mht.config.NewTrackScore,
		Depth:      0,
		CreatedAt:  now,
		LastUpdate: now,
		NMisses:    0,
	}
}

// calculateAssociationScore calculates score for measurement association
func (mht *MHTTracker) calculateAssociationScore(track *TrackState, meas Measurement) float64 {
	// Calculate Mahalanobis distance
	posDiff := [3]float64{
		meas.Position[0] - track.Position[0],
		meas.Position[1] - track.Position[1],
		meas.Position[2] - track.Position[2],
	}

	// Use covariance for distance calculation
	invCov := mht.invertCovariance(track.Covariance)

	distance := 0.0
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			distance += posDiff[i] * invCov[i][j] * posDiff[j]
		}
	}

	// Score based on distance (negative log-likelihood)
	score := -0.5 * distance

	return score
}

// invertCovariance inverts a 6x6 covariance matrix (simplified)
func (mht *MHTTracker) invertCovariance(cov [6][6]float64) [6][6]float64 {
	inv := [6][6]float64{}

	// Simplified: only invert diagonal elements
	for i := 0; i < 6; i++ {
		if cov[i][i] > 0 {
			inv[i][i] = 1.0 / cov[i][i]
		}
	}

	return inv
}

// predictTrack predicts track state forward in time
func (mht *MHTTracker) predictTrack(track *TrackState, dt time.Duration) *TrackState {
	newTrack := &TrackState{
		ID:              track.ID,
		NUpdates:        track.NUpdates,
		LastMeasurement: track.LastMeasurement,
		Status:          track.Status,
	}

	dtSec := dt.Seconds()

	// Constant velocity prediction
	for i := 0; i < 3; i++ {
		newTrack.Position[i] = track.Position[i] + track.Velocity[i]*dtSec
		newTrack.Velocity[i] = track.Velocity[i]
		newTrack.Acceleration[i] = track.Acceleration[i]
	}

	// Propagate covariance (simplified)
	processNoise := 1.0 // Process noise variance
	for i := 0; i < 6; i++ {
		for j := 0; j < 6; j++ {
			newTrack.Covariance[i][j] = track.Covariance[i][j] + processNoise
		}
	}

	return newTrack
}

// updateTrackWithMeasurement updates track with measurement
func (mht *MHTTracker) updateTrackWithMeasurement(track *TrackState, meas Measurement, now time.Time) *TrackState {
	newTrack := mht.predictTrack(track, now.Sub(track.LastMeasurement))

	// Kalman-like update
	K := [6][3]float64{} // Kalman gain
	for i := 0; i < 3; i++ {
		variance := meas.Variance[i] + newTrack.Covariance[i][i]
		if variance > 0 {
			K[i][i] = newTrack.Covariance[i][i] / variance
		}
	}

	// Update position
	for i := 0; i < 3; i++ {
		innovation := meas.Position[i] - newTrack.Position[i]
		newTrack.Position[i] += K[i][i] * innovation
	}

	// Update covariance
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			newTrack.Covariance[i][j] *= (1.0 - K[i][i])
		}
	}

	newTrack.LastMeasurement = now
	newTrack.NUpdates++

	// Update status
	if newTrack.NUpdates >= 3 {
		newTrack.Status = "CONFIRMED"
	}

	return newTrack
}

// isMeasurementAssociated checks if measurement is already associated
func (mht *MHTTracker) isMeasurementAssociated(measID uint64) bool {
	for _, hyp := range mht.hypotheses {
		for _, assoc := range hyp.Associations {
			if assoc.MeasurementID == measID {
				return true
			}
		}
	}
	return false
}

// pruneHypotheses removes low-score hypotheses
func (mht *MHTTracker) pruneHypotheses() {
	// Use heap to find top hypotheses
	h := &HypothesisHeap{}
	heap.Init(h)

	for id, hyp := range mht.hypotheses {
		// Skip hypotheses that are too deep
		if hyp.Depth > mht.config.MaxDepth {
			delete(mht.hypotheses, id)
			continue
		}

		// Skip hypotheses with too many misses
		if hyp.NMisses > mht.config.MaxMisses {
			delete(mht.hypotheses, id)
			continue
		}

		// Skip low-score hypotheses
		if hyp.Score < mht.config.MinScore {
			delete(mht.hypotheses, id)
			continue
		}

		heap.Push(h, hyp)
	}

	// Keep only top N hypotheses
	for h.Len() > mht.config.MaxHypotheses {
		hyp := heap.Pop(h).(*Hypothesis)
		delete(mht.hypotheses, hyp.ID)
	}
}

// GetBestHypotheses returns the best hypothesis for each track
func (mht *MHTTracker) GetBestHypotheses() map[uint32]*Hypothesis {
	mht.mu.RLock()
	defer mht.mu.RUnlock()

	best := make(map[uint32]*Hypothesis)

	for _, hyp := range mht.hypotheses {
		existing, exists := best[hyp.TrackNumber]
		if !exists || hyp.Score > existing.Score {
			best[hyp.TrackNumber] = hyp
		}
	}

	return best
}

// GetHypothesis returns a hypothesis by ID
func (mht *MHTTracker) GetHypothesis(id uint64) *Hypothesis {
	mht.mu.RLock()
	defer mht.mu.RUnlock()
	return mht.hypotheses[id]
}

// GetAllHypotheses returns all hypotheses
func (mht *MHTTracker) GetAllHypotheses() []*Hypothesis {
	mht.mu.RLock()
	defer mht.mu.RUnlock()

	hyps := make([]*Hypothesis, 0, len(mht.hypotheses))
	for _, hyp := range mht.hypotheses {
		hyps = append(hyps, hyp)
	}
	return hyps
}

// Stats returns MHT statistics
func (mht *MHTTracker) Stats() MHTStats {
	mht.mu.RLock()
	defer mht.mu.RUnlock()

	activeTracks := make(map[uint32]bool)
	for _, hyp := range mht.hypotheses {
		if hyp.Track.Status == "CONFIRMED" {
			activeTracks[hyp.TrackNumber] = true
		}
	}

	return MHTStats{
		TotalHypotheses:  len(mht.hypotheses),
		ActiveTracks:     len(activeTracks),
		NextHypothesisID: mht.nextHypID,
		NextTrackNumber:  mht.nextTrackNum,
	}
}

// MHTStats holds MHT statistics
type MHTStats struct {
	TotalHypotheses  int    `json:"total_hypotheses"`
	ActiveTracks     int    `json:"active_tracks"`
	NextHypothesisID uint64 `json:"next_hypothesis_id"`
	NextTrackNumber  uint32 `json:"next_track_number"`
}

// HypothesisHeap implements heap.Interface for hypothesis scoring
type HypothesisHeap []*Hypothesis

func (h HypothesisHeap) Len() int           { return len(h) }
func (h HypothesisHeap) Less(i, j int) bool { return h[i].Score > h[j].Score } // Higher score = higher priority
func (h HypothesisHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *HypothesisHeap) Push(x interface{}) {
	*h = append(*h, x.(*Hypothesis))
}

func (h *HypothesisHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
