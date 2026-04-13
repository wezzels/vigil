// Package radar provides track correlation for multi-source fusion
package radar

import (
	"math"
	"sync"
	"time"
)

// TrackCorrelator correlates tracks from multiple radar sources
type TrackCorrelator struct {
	config          *CorrelationConfig
	tracks          map[uint32]*CorrelatedTrack
	nextTrackNumber uint32
	mu              sync.RWMutex
}

// CorrelationConfig holds configuration for track correlation
type CorrelationConfig struct {
	// Correlation parameters
	MaxDistance     float64       `json:"max_distance"`     // Maximum distance for correlation (m)
	MaxVelocityDiff float64       `json:"max_velocity_diff"` // Maximum velocity difference (m/s)
	MaxTimeDiff     time.Duration `json:"max_time_diff"`     // Maximum time difference
	MinConfidence   float64       `json:"min_confidence"`    // Minimum confidence for correlation
	
	// Scoring weights
	DistanceWeight   float64 `json:"distance_weight"`
	VelocityWeight   float64 `json:"velocity_weight"`
	ConfidenceWeight  float64 `json:"confidence_weight"`
	ContinuityWeight  float64 `json:"continuity_weight"`
	
	// Track management
	MaxTrackAge      time.Duration `json:"max_track_age"`      // Maximum age before drop
	MinUpdates       int           `json:"min_updates"`         // Minimum updates for confirmed track
	CoastTime        time.Duration `json:"coast_time"`         // Time before coasting
	DropTime         time.Duration `json:"drop_time"`          // Time before dropping
}

// CorrelatedTrack represents a correlated track from multiple sources
type CorrelatedTrack struct {
	TrackNumber   uint32         `json:"track_number"`
	SourceTracks  map[string]*RadarTrack `json:"source_tracks"` // Source ID -> Track
	Position      Position       `json:"position"`
	Velocity      Velocity       `json:"velocity"`
	Covariance    [6][6]float64  `json:"covariance"` // State covariance
	Score         float64        `json:"score"`
	Confidence    float64        `json:"confidence"`
	NUpdates      int            `json:"n_updates"`
	Status        string         `json:"status"`
	FirstUpdate   time.Time      `json:"first_update"`
	LastUpdate    time.Time      `json:"last_update"`
}

// Position represents a 3D position
type Position struct {
	Lat float64 `json:"lat"` // Degrees
	Lon float64 `json:"lon"` // Degrees
	Alt float64 `json:"alt"` // Meters
}

// Velocity represents a 3D velocity
type Velocity struct {
	N float64 `json:"n"` // North velocity (m/s)
	E float64 `json:"e"` // East velocity (m/s)
	U float64 `json:"u"` // Up velocity (m/s)
}

// CorrelationResult holds correlation results
type CorrelationResult struct {
	TrackNumber    uint32   `json:"track_number"`
	SourceTracks   []string `json:"source_tracks"`
	Score          float64  `json:"score"`
	IsNew         bool     `json:"is_new"`
	IsCorrelated  bool     `json:"is_correlated"`
}

// DefaultCorrelationConfig returns default correlation configuration
func DefaultCorrelationConfig() *CorrelationConfig {
	return &CorrelationConfig{
		MaxDistance:      5000.0,    // 5 km
		MaxVelocityDiff:  100.0,     // 100 m/s
		MaxTimeDiff:      5 * time.Second,
		MinConfidence:    0.5,
		DistanceWeight:   0.4,
		VelocityWeight:   0.3,
		ConfidenceWeight: 0.2,
		ContinuityWeight: 0.1,
		MaxTrackAge:      60 * time.Second,
		MinUpdates:       3,
		CoastTime:        10 * time.Second,
		DropTime:         30 * time.Second,
	}
}

// NewTrackCorrelator creates a new track correlator
func NewTrackCorrelator(config *CorrelationConfig) *TrackCorrelator {
	if config == nil {
		config = DefaultCorrelationConfig()
	}
	
	return &TrackCorrelator{
		config:          config,
		tracks:          make(map[uint32]*CorrelatedTrack),
		nextTrackNumber: 1000,
	}
}

// Correlate correlates a new track with existing tracks
func (tc *TrackCorrelator) Correlate(track *RadarTrack) *CorrelationResult {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	
	// Try to correlate with existing tracks
	bestScore := 0.0
	bestTrackNum := uint32(0)
	
	for trackNum, existing := range tc.tracks {
		score := tc.calculateScore(track, existing)
		if score > bestScore && score >= tc.config.MinConfidence {
			bestScore = score
			bestTrackNum = trackNum
		}
	}
	
	// If found a match
	if bestTrackNum > 0 {
		tc.updateCorrelatedTrack(bestTrackNum, track, bestScore)
		return &CorrelationResult{
			TrackNumber:   bestTrackNum,
			SourceTracks: tc.getSourceTrackIDs(bestTrackNum),
			Score:        bestScore,
			IsNew:        false,
			IsCorrelated: true,
		}
	}
	
	// Create new track
	trackNum := tc.createNewTrack(track)
	return &CorrelationResult{
		TrackNumber:   trackNum,
		SourceTracks:  []string{track.SensorID},
		Score:         1.0,
		IsNew:         true,
		IsCorrelated: false,
	}
}

// calculateScore calculates correlation score between track and existing track
func (tc *TrackCorrelator) calculateScore(newTrack *RadarTrack, existing *CorrelatedTrack) float64 {
	// Position distance
	posDistance := tc.positionDistance(
		newTrack.Latitude, newTrack.Longitude, newTrack.Altitude,
		existing.Position.Lat, existing.Position.Lon, existing.Position.Alt,
	)
	
	// Velocity difference
	velDiff := tc.velocityDifference(
		newTrack.VelocityN, newTrack.VelocityE, newTrack.VelocityU,
		existing.Velocity.N, existing.Velocity.E, existing.Velocity.U,
	)
	
	// Time difference
	timeDiff := time.Since(existing.LastUpdate)
	
	// Check thresholds
	if posDistance > tc.config.MaxDistance {
		return 0.0
	}
	if velDiff > tc.config.MaxVelocityDiff {
		return 0.0
	}
	if timeDiff > tc.config.MaxTimeDiff {
		return 0.0
	}
	
	// Calculate score components
	distanceScore := 1.0 - (posDistance / tc.config.MaxDistance)
	velocityScore := 1.0 - (velDiff / tc.config.MaxVelocityDiff)
	confidenceScore := float64(newTrack.TrackQuality) / 7.0 // Track quality is 0-7
	continuityScore := 1.0 / (1.0 + float64(existing.NUpdates)) // More updates = better
	
	// Weighted sum
	totalScore := tc.config.DistanceWeight*distanceScore +
		tc.config.VelocityWeight*velocityScore +
		tc.config.ConfidenceWeight*confidenceScore +
		tc.config.ContinuityWeight*continuityScore
	
	return totalScore
}

// positionDistance calculates distance between two positions (simplified)
func (tc *TrackCorrelator) positionDistance(lat1, lon1, alt1, lat2, lon2, alt2 float64) float64 {
	// Approximate distance calculation (haversine + altitude)
	// This is a simplified version - real implementation would use proper geodesic
	
	dLat := (lat2 - lat1) * 111000.0 // meters per degree latitude
	dLon := (lon2 - lon1) * 111000.0 * math.Cos(lat1*math.Pi/180.0)
	dAlt := alt2 - alt1
	
	return math.Sqrt(dLat*dLat + dLon*dLon + dAlt*dAlt)
}

// velocityDifference calculates velocity difference magnitude
func (tc *TrackCorrelator) velocityDifference(vn1, ve1, vu1, vn2, ve2, vu2 float64) float64 {
	dn := vn2 - vn1
	de := ve2 - ve1
	du := vu2 - vu1
	return math.Sqrt(dn*dn + de*de + du*du)
}

// createNewTrack creates a new correlated track
func (tc *TrackCorrelator) createNewTrack(track *RadarTrack) uint32 {
	trackNum := tc.nextTrackNumber
	tc.nextTrackNumber++
	
	now := time.Now()
	
	corrTrack := &CorrelatedTrack{
		TrackNumber: trackNum,
		SourceTracks: map[string]*RadarTrack{
			track.SensorID: track,
		},
		Position: Position{
			Lat: track.Latitude,
			Lon: track.Longitude,
			Alt: track.Altitude,
		},
		Velocity: Velocity{
			N: track.VelocityN,
			E: track.VelocityE,
			U: track.VelocityU,
		},
		Score:       1.0,
		Confidence:  float64(track.TrackQuality) / 7.0,
		NUpdates:    1,
		Status:       TrackStatusInit,
		FirstUpdate:  now,
		LastUpdate:   now,
	}
	
	// Initialize covariance
	tc.initializeCovariance(corrTrack, track)
	
	tc.tracks[trackNum] = corrTrack
	return trackNum
}

// updateCorrelatedTrack updates an existing correlated track
func (tc *TrackCorrelator) updateCorrelatedTrack(trackNum uint32, track *RadarTrack, score float64) {
	corrTrack := tc.tracks[trackNum]
	
	// Add source track
	corrTrack.SourceTracks[track.SensorID] = track
	
	// Weighted position update
	nSources := float64(len(corrTrack.SourceTracks))
	weight := 1.0 / nSources
	
	corrTrack.Position.Lat = corrTrack.Position.Lat*(1-weight) + track.Latitude*weight
	corrTrack.Position.Lon = corrTrack.Position.Lon*(1-weight) + track.Longitude*weight
	corrTrack.Position.Alt = corrTrack.Position.Alt*(1-weight) + track.Altitude*weight
	
	// Weighted velocity update
	corrTrack.Velocity.N = corrTrack.Velocity.N*(1-weight) + track.VelocityN*weight
	corrTrack.Velocity.E = corrTrack.Velocity.E*(1-weight) + track.VelocityE*weight
	corrTrack.Velocity.U = corrTrack.Velocity.U*(1-weight) + track.VelocityU*weight
	
	// Update score and confidence
	corrTrack.Score = score
	corrTrack.Confidence = math.Min(1.0, corrTrack.Confidence+0.1)
	
	// Update tracking
	corrTrack.NUpdates++
	corrTrack.LastUpdate = time.Now()
	
	// Update status
	if corrTrack.NUpdates >= tc.config.MinUpdates {
		corrTrack.Status = TrackStatusTrack
	}
}

// initializeCovariance initializes state covariance
func (tc *TrackCorrelator) initializeCovariance(corrTrack *CorrelatedTrack, track *RadarTrack) {
	// Initialize with large uncertainties
	for i := 0; i < 6; i++ {
		for j := 0; j < 6; j++ {
			corrTrack.Covariance[i][j] = 0.0
		}
	}
	
	// Position variances (from track quality)
	posVar := (7.0 - float64(track.TrackQuality)) / 7.0 * 1000.0 // meters^2
	corrTrack.Covariance[0][0] = posVar
	corrTrack.Covariance[1][1] = posVar
	corrTrack.Covariance[2][2] = posVar
	
	// Velocity variances
	velVar := 100.0 // (m/s)^2
	corrTrack.Covariance[3][3] = velVar
	corrTrack.Covariance[4][4] = velVar
	corrTrack.Covariance[5][5] = velVar
}

// getSourceTrackIDs returns source track IDs
func (tc *TrackCorrelator) getSourceTrackIDs(trackNum uint32) []string {
	corrTrack := tc.tracks[trackNum]
	if corrTrack == nil {
		return nil
	}
	
	ids := make([]string, 0, len(corrTrack.SourceTracks))
	for id := range corrTrack.SourceTracks {
		ids = append(ids, id)
	}
	return ids
}

// GetTrack returns a correlated track by number
func (tc *TrackCorrelator) GetTrack(trackNum uint32) *CorrelatedTrack {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.tracks[trackNum]
}

// GetAllTracks returns all correlated tracks
func (tc *TrackCorrelator) GetAllTracks() []*CorrelatedTrack {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	
	tracks := make([]*CorrelatedTrack, 0, len(tc.tracks))
	for _, track := range tc.tracks {
		tracks = append(tracks, track)
	}
	return tracks
}

// GetActiveTracks returns all active tracks
func (tc *TrackCorrelator) GetActiveTracks() []*CorrelatedTrack {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	
	tracks := make([]*CorrelatedTrack, 0)
	now := time.Now()
	
	for _, track := range tc.tracks {
		// Skip dropped tracks
		if track.Status == TrackStatusDrop {
			continue
		}
		// Skip old tracks
		if now.Sub(track.LastUpdate) > tc.config.MaxTrackAge {
			continue
		}
		tracks = append(tracks, track)
	}
	return tracks
}

// UpdateTrackStates updates all track states
func (tc *TrackCorrelator) UpdateTrackStates() {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	
	now := time.Now()
	
	for trackNum, track := range tc.tracks {
		age := now.Sub(track.LastUpdate)
		
		switch track.Status {
		case TrackStatusInit:
			if track.NUpdates >= tc.config.MinUpdates {
				track.Status = TrackStatusTrack
			} else if age > tc.config.MaxTrackAge {
				track.Status = TrackStatusDrop
			}
		case TrackStatusTrack:
			if age > tc.config.CoastTime {
				track.Status = TrackStatusCoast
			}
		case TrackStatusCoast:
			if age > tc.config.DropTime {
				track.Status = TrackStatusDrop
			}
		}
		
		// Remove dropped tracks
		if track.Status == TrackStatusDrop {
			delete(tc.tracks, trackNum)
		}
	}
}

// PruneOldTracks removes tracks older than max age
func (tc *TrackCorrelator) PruneOldTracks() int {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	
	now := time.Now()
	count := 0
	
	for trackNum, track := range tc.tracks {
		if now.Sub(track.LastUpdate) > tc.config.MaxTrackAge {
			delete(tc.tracks, trackNum)
			count++
		}
	}
	
	return count
}

// Stats returns correlator statistics
func (tc *TrackCorrelator) Stats() CorrelatorStats {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	
	activeTracks := 0
	droppedTracks := 0
	
	for _, track := range tc.tracks {
		switch track.Status {
		case TrackStatusInit, TrackStatusTrack, TrackStatusCoast:
			activeTracks++
		case TrackStatusDrop:
			droppedTracks++
		}
	}
	
	return CorrelatorStats{
		TotalTracks:    len(tc.tracks),
		ActiveTracks:   activeTracks,
		DroppedTracks:  droppedTracks,
		NextTrackNumber: tc.nextTrackNumber,
	}
}

// CorrelatorStats holds correlator statistics
type CorrelatorStats struct {
	TotalTracks      int     `json:"total_tracks"`
	ActiveTracks     int     `json:"active_tracks"`
	DroppedTracks    int     `json:"dropped_tracks"`
	NextTrackNumber  uint32  `json:"next_track_number"`
}