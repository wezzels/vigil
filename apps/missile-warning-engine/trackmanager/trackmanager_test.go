// Package trackmanager provides track management for missile warning
package trackmanager

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestTrackCreation tests track creation
func TestTrackCreation(t *testing.T) {
	tm := NewTrackManager()
	ctx := context.Background()

	track := &Track{
		ID:        "track-001",
		Position:  Position{Lat: 34.0522, Lon: -118.2437, Alt: 10000},
		Velocity:  Velocity{X: 100, Y: 200, Z: 50},
		Source:    "OPIR",
		CreatedAt: time.Now(),
	}

	err := tm.CreateTrack(ctx, track)
	if err != nil {
		t.Fatalf("Failed to create track: %v", err)
	}

	retrieved, err := tm.GetTrack(ctx, "track-001")
	if err != nil {
		t.Fatalf("Failed to get track: %v", err)
	}

	if retrieved.ID != track.ID {
		t.Errorf("Expected ID %s, got %s", track.ID, retrieved.ID)
	}
}

// TestTrackUpdate tests track update
func TestTrackUpdate(t *testing.T) {
	tm := NewTrackManager()
	ctx := context.Background()

	// Create initial track
	track := &Track{
		ID:        "track-001",
		Position:  Position{Lat: 34.0522, Lon: -118.2437, Alt: 10000},
		Velocity:  Velocity{X: 100, Y: 200, Z: 50},
		Source:    "OPIR",
		CreatedAt: time.Now(),
	}
	tm.CreateTrack(ctx, track)

	// Update track
	update := &TrackUpdate{
		Position: &Position{Lat: 34.0530, Lon: -118.2440, Alt: 10100},
		Velocity: &Velocity{X: 105, Y: 205, Z: 55},
	}

	err := tm.UpdateTrack(ctx, "track-001", update)
	if err != nil {
		t.Fatalf("Failed to update track: %v", err)
	}

	retrieved, _ := tm.GetTrack(ctx, "track-001")
	if retrieved.Position.Lat != 34.0530 {
		t.Errorf("Expected lat 34.0530, got %f", retrieved.Position.Lat)
	}
}

// TestTrackDeletion tests track deletion
func TestTrackDeletion(t *testing.T) {
	tm := NewTrackManager()
	ctx := context.Background()

	track := &Track{
		ID:        "track-001",
		Position:  Position{Lat: 34.0522, Lon: -118.2437, Alt: 10000},
		Source:    "OPIR",
		CreatedAt: time.Now(),
	}
	tm.CreateTrack(ctx, track)

	err := tm.DeleteTrack(ctx, "track-001")
	if err != nil {
		t.Fatalf("Failed to delete track: %v", err)
	}

	_, err = tm.GetTrack(ctx, "track-001")
	if err == nil {
		t.Error("Expected error getting deleted track")
	}
}

// TestTrackCorrelation tests track correlation
func TestTrackCorrelation(t *testing.T) {
	tm := NewTrackManager()
	ctx := context.Background()

	// Create two tracks from different sensors
	track1 := &Track{
		ID:        "track-opir-001",
		Position:  Position{Lat: 34.0522, Lon: -118.2437, Alt: 10000},
		Source:    "OPIR",
		CreatedAt: time.Now(),
	}
	track2 := &Track{
		ID:        "track-radar-001",
		Position:  Position{Lat: 34.0523, Lon: -118.2438, Alt: 10050}, // Close to track1
		Source:    "RADAR",
		CreatedAt: time.Now(),
	}

	tm.CreateTrack(ctx, track1)
	tm.CreateTrack(ctx, track2)

	// Correlate tracks
	correlated := tm.CorrelateTracks(ctx, "track-opir-001", "track-radar-001")
	if !correlated {
		t.Error("Expected tracks to be correlated")
	}
}

// TestThreatTypeEstimation tests threat type estimation
func TestThreatTypeEstimation(t *testing.T) {
	tm := NewTrackManager()
	ctx := context.Background()

	tests := []struct {
		name     string
		velocity Velocity
		alt      float64
		expected ThreatType
	}{
		{
			name:     "ballistic_missile",
			velocity: Velocity{X: 5000, Y: 0, Z: 2000}, // High velocity
			alt:      100000,
			expected: ThreatTypeBallistic,
		},
		{
			name:     "aircraft",
			velocity: Velocity{X: 200, Y: 100, Z: 0},
			alt:      10000,
			expected: ThreatTypeAircraft,
		},
		{
			name:     "unknown",
			velocity: Velocity{X: 50, Y: 50, Z: 0},
			alt:      5000,
			expected: ThreatTypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			track := &Track{
				ID:        "track-" + tt.name,
				Position:  Position{Lat: 34.0522, Lon: -118.2437, Alt: tt.alt},
				Velocity:  tt.velocity,
				Source:    "OPIR",
				CreatedAt: time.Now(),
			}
			tm.CreateTrack(ctx, track)

			threatType := tm.EstimateThreatType(ctx, track)
			if threatType != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, threatType)
			}
		})
	}
}

// TestAlertLevelEscalation tests alert level escalation
func TestAlertLevelEscalation(t *testing.T) {
	tm := NewTrackManager()
	ctx := context.Background()

	track := &Track{
		ID:        "track-001",
		Position:  Position{Lat: 34.0522, Lon: -118.2437, Alt: 100000},
		Velocity:  Velocity{X: 5000, Y: 0, Z: 2000},
		Source:    "OPIR",
		CreatedAt: time.Now(),
	}
	tm.CreateTrack(ctx, track)

	// Initial alert level
	level := tm.GetAlertLevel(ctx, "track-001")
	if level != AlertLevelNone {
		t.Errorf("Expected no alert, got %v", level)
	}

	// Escalate based on threat
	tm.UpdateAlertLevel(ctx, "track-001", AlertLevelImminent)
	level = tm.GetAlertLevel(ctx, "track-001")
	if level != AlertLevelImminent {
		t.Errorf("Expected imminent, got %v", level)
	}
}

// TestTrackAging tests track aging
func TestTrackAging(t *testing.T) {
	tm := NewTrackManager()
	ctx := context.Background()

	// Create track with old timestamp
	oldTime := time.Now().Add(-5 * time.Minute)
	track := &Track{
		ID:        "track-001",
		Position:  Position{Lat: 34.0522, Lon: -118.2437, Alt: 10000},
		Source:    "OPIR",
		CreatedAt: oldTime,
		UpdatedAt: oldTime,
	}
	tm.CreateTrack(ctx, track)

	age := tm.GetTrackAge(ctx, "track-001")
	if age < 5*time.Minute {
		t.Errorf("Expected age >= 5 minutes, got %v", age)
	}
}

// TestTrackCleanup tests track cleanup
func TestTrackCleanup(t *testing.T) {
	tm := NewTrackManager()
	ctx := context.Background()

	// Create old track
	oldTime := time.Now().Add(-10 * time.Minute)
	track := &Track{
		ID:        "track-old",
		Position:  Position{Lat: 34.0522, Lon: -118.2437, Alt: 10000},
		Source:    "OPIR",
		CreatedAt: oldTime,
		UpdatedAt: oldTime,
	}
	tm.CreateTrack(ctx, track)

	// Create new track
	newTrack := &Track{
		ID:        "track-new",
		Position:  Position{Lat: 34.0522, Lon: -118.2437, Alt: 10000},
		Source:    "OPIR",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	tm.CreateTrack(ctx, newTrack)

	// Clean up old tracks (> 5 minutes)
	tm.CleanupOldTracks(ctx, 5*time.Minute)

	// Old track should be deleted
	_, err := tm.GetTrack(ctx, "track-old")
	if err == nil {
		t.Error("Old track should be deleted")
	}

	// New track should remain
	_, err = tm.GetTrack(ctx, "track-new")
	if err != nil {
		t.Error("New track should still exist")
	}
}

// Types

type Track struct {
	ID        string
	Position  Position
	Velocity  Velocity
	Source    string
	CreatedAt time.Time
	UpdatedAt time.Time
	ThreatType ThreatType
	AlertLevel AlertLevel
}

type Position struct {
	Lat, Lon, Alt float64
}

type Velocity struct {
	X, Y, Z float64
}

type ThreatType int

const (
	ThreatTypeUnknown ThreatType = iota
	ThreatTypeBallistic
	ThreatTypeAircraft
	ThreatTypeCruise
	ThreatTypeUAV
)

type AlertLevel int

const (
	AlertLevelNone AlertLevel = iota
	AlertLevelWatch
	AlertLevelWarning
	AlertLevelImminent
)

type TrackUpdate struct {
	Position *Position
	Velocity *Velocity
}

// TrackManager manages tracks
type TrackManager struct {
	mu     sync.RWMutex
	tracks map[string]*Track
}

func NewTrackManager() *TrackManager {
	return &TrackManager{
		tracks: make(map[string]*Track),
	}
}

func (tm *TrackManager) CreateTrack(ctx context.Context, track *Track) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.tracks[track.ID] = track
	return nil
}

func (tm *TrackManager) GetTrack(ctx context.Context, id string) (*Track, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	track, ok := tm.tracks[id]
	if !ok {
		return nil, ErrTrackNotFound
	}
	return track, nil
}

func (tm *TrackManager) UpdateTrack(ctx context.Context, id string, update *TrackUpdate) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	track, ok := tm.tracks[id]
	if !ok {
		return ErrTrackNotFound
	}
	if update.Position != nil {
		track.Position = *update.Position
	}
	if update.Velocity != nil {
		track.Velocity = *update.Velocity
	}
	track.UpdatedAt = time.Now()
	return nil
}

func (tm *TrackManager) DeleteTrack(ctx context.Context, id string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	delete(tm.tracks, id)
	return nil
}

func (tm *TrackManager) CorrelateTracks(ctx context.Context, id1, id2 string) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	t1, ok1 := tm.tracks[id1]
	t2, ok2 := tm.tracks[id2]
	if !ok1 || !ok2 {
		return false
	}
	// Simple distance-based correlation
	dLat := t1.Position.Lat - t2.Position.Lat
	dLon := t1.Position.Lon - t2.Position.Lon
	dAlt := t1.Position.Alt - t2.Position.Alt
	return dLat*dLat+dLon*dLon < 0.0001 && dAlt*dAlt < 10000
}

func (tm *TrackManager) EstimateThreatType(ctx context.Context, track *Track) ThreatType {
	speed := track.Velocity.X*track.Velocity.X + track.Velocity.Y*track.Velocity.Y + track.Velocity.Z*track.Velocity.Z
	speed = float64(int(speed*100)) / 100 // sqrt approximation
	
	if track.Position.Alt > 50000 && speed > 10000000 {
		return ThreatTypeBallistic
	}
	if track.Position.Alt > 5000 && track.Position.Alt < 20000 && speed > 10000 && speed < 10000000 {
		return ThreatTypeAircraft
	}
	return ThreatTypeUnknown
}

func (tm *TrackManager) GetAlertLevel(ctx context.Context, id string) AlertLevel {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	track, ok := tm.tracks[id]
	if !ok {
		return AlertLevelNone
	}
	return track.AlertLevel
}

func (tm *TrackManager) UpdateAlertLevel(ctx context.Context, id string, level AlertLevel) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if track, ok := tm.tracks[id]; ok {
		track.AlertLevel = level
	}
}

func (tm *TrackManager) GetTrackAge(ctx context.Context, id string) time.Duration {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	track, ok := tm.tracks[id]
	if !ok {
		return 0
	}
	return time.Since(track.UpdatedAt)
}

func (tm *TrackManager) CleanupOldTracks(ctx context.Context, maxAge time.Duration) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	for id, track := range tm.tracks {
		if time.Since(track.UpdatedAt) > maxAge {
			delete(tm.tracks, id)
		}
	}
}

var ErrTrackNotFound = fmt.Errorf("track not found")