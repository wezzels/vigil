// Package fusion implements track correlation and sensor fusion algorithms
package fusion

import (
	"math"
)

// Track represents a correlated track from multiple sensors
type Track struct {
	ID          uint64
	Lat         float64
	Lon         float64
	Alt         float64
	VelocityN   float64 // m/s North
	VelocityE   float64 // m/s East
	VelocityU   float64 // m/s Up
	VarLat      float64 // Variance
	VarLon      float64
	VarAlt      float64
	TrackNum    uint32
	SourceCount int
	LastUpdate  int64
}

// Measurement represents a sensor observation
type Measurement struct {
	ID        uint64
	Lat       float64
	Lon       float64
	Alt       float64
	VarLat    float64
	VarLon    float64
	VarAlt    float64
	SourceID  int
	Timestamp int64
}

// Association represents a track-measurement association hypothesis
type Association struct {
	TrackID       uint64
	MeasurementID uint64
	Probability    float64
	Mahalanobis   float64
}

// TrackManager manages track creation, updates, and deletion
type TrackManager struct {
	Tracks       map[uint64]*Track
	NextTrackNum uint32
	MaxTracks    int
	MaxAge       int64 // Max age in milliseconds
	GatingLimit  float64 // Mahalanobis distance threshold
}

// NewTrackManager creates a new track manager
func NewTrackManager() *TrackManager {
	return &TrackManager{
		Tracks:       make(map[uint64]*Track),
		NextTrackNum: 1,
		MaxTracks:    100000,
		MaxAge:       300000, // 5 minutes
		GatingLimit:  9.0,    // 3-sigma squared
	}
}

// MahalanobisDistance calculates the Mahalanobis distance between measurement and track
func (tm *TrackManager) MahalanobisDistance(m *Measurement, t *Track) float64 {
	// Convert lat/lon to meters for distance calculation
	// Approximate: 1 degree = 111km
	mLatM := m.Lat * 111000
	mLonM := m.Lon * 111000 * math.Cos(m.Lat*math.Pi/180)
	tLatM := t.Lat * 111000
	tLonM := t.Lon * 111000 * math.Cos(t.Lat*math.Pi/180)
	
	// Position differences
	dLat := mLatM - tLatM
	dLon := mLonM - tLonM
	dAlt := m.Alt - t.Alt
	
	// Combined variance (approximate as diagonal)
	vLat := t.VarLat*111000*111000 + m.VarLat*111000*111000
	vLon := t.VarLon*111000*111000*math.Cos(t.Lat*math.Pi/180)*math.Cos(t.Lat*math.Pi/180) +
		m.VarLon*111000*111000*math.Cos(m.Lat*math.Pi/180)*math.Cos(m.Lat*math.Pi/180)
	vAlt := t.VarAlt + m.VarAlt
	
	// Mahalanobis distance: d^T * S^-1 * d
	dSquared := dLat*dLat/vLat + dLon*dLon/vLon + dAlt*dAlt/vAlt
	
	return dSquared
}

// JPDAAssociate performs Joint Probabilistic Data Association
// Returns associations for all track-measurement pairs within gating limit
func (tm *TrackManager) JPDAAssociate(measurements []*Measurement) []*Association {
	associations := make([]*Association, 0)
	
	for _, m := range measurements {
		for _, t := range tm.Tracks {
			dist := tm.MahalanobisDistance(m, t)
			
			if dist < tm.GatingLimit {
				// Calculate probability based on Mahalanobis distance
				prob := math.Exp(-dist / 2.0)
				
				assoc := &Association{
					TrackID:       t.ID,
					MeasurementID: m.ID,
					Probability:   prob,
					Mahalanobis:    dist,
				}
				associations = append(associations, assoc)
			}
		}
	}
	
	return associations
}

// Update updates tracks based on measurements using JPDA
func (tm *TrackManager) Update(measurements []*Measurement, timestamp int64) {
	// Get associations
	associations := tm.JPDAAssociate(measurements)
	
	// Group associations by track
	trackAssocs := make(map[uint64][]*Association)
	for _, a := range associations {
		trackAssocs[a.TrackID] = append(trackAssocs[a.TrackID], a)
	}
	
	// Update each track with weighted combination
	for trackID, assocs := range trackAssocs {
		track := tm.Tracks[trackID]
		if track == nil {
			continue
		}
		
		// Normalize probabilities
		totalProb := 0.0
		for _, a := range assocs {
			totalProb += a.Probability
		}
		if totalProb == 0 {
			continue
		}
		
		// Weighted update
		newLat := 0.0
		newLon := 0.0
		newAlt := 0.0
		
		for _, a := range assocs {
			m := findMeasurement(measurements, a.MeasurementID)
			if m == nil {
				continue
			}
			weight := a.Probability / totalProb
			newLat += m.Lat * weight
			newLon += m.Lon * weight
			newAlt += m.Alt * weight
		}
		
		// Simple weighted average update (could use Kalman)
		alpha := 0.3 // Update rate
		track.Lat = track.Lat*(1-alpha) + newLat*alpha
		track.Lon = track.Lon*(1-alpha) + newLon*alpha
		track.Alt = track.Alt*(1-alpha) + newAlt*alpha
		track.LastUpdate = timestamp
		track.SourceCount++
	}
	
	// Initiate new tracks for unassociated measurements
	for _, m := range measurements {
		associated := false
		for _, a := range associations {
			if a.MeasurementID == m.ID {
				associated = true
				break
			}
		}
		
		if !associated {
			tm.InitiateTrack(m)
		}
	}
	
	// Remove old tracks
	tm.Cleanup(timestamp)
}

// InitiateTrack creates a new track from a measurement
func (tm *TrackManager) InitiateTrack(m *Measurement) *Track {
	if len(tm.Tracks) >= tm.MaxTracks {
		return nil
	}
	
	track := &Track{
		ID:          m.ID,
		Lat:         m.Lat,
		Lon:         m.Lon,
		Alt:         m.Alt,
		VarLat:      m.VarLat,
		VarLon:      m.VarLon,
		VarAlt:      m.VarAlt,
		TrackNum:    tm.NextTrackNum,
		SourceCount: 1,
		LastUpdate:  m.Timestamp,
	}
	
	tm.NextTrackNum++
	tm.Tracks[track.ID] = track
	
	return track
}

// Cleanup removes tracks older than MaxAge
func (tm *TrackManager) Cleanup(timestamp int64) {
	for id, t := range tm.Tracks {
		if timestamp-t.LastUpdate > tm.MaxAge {
			delete(tm.Tracks, id)
		}
	}
}

// Helper function to find measurement by ID
func findMeasurement(measurements []*Measurement, id uint64) *Measurement {
	for _, m := range measurements {
		if m.ID == id {
			return m
		}
	}
	return nil
}