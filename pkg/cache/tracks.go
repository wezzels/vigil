// Package cache provides track state caching
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// TrackState represents cached track state
type TrackState struct {
	TrackNumber string    `json:"track_number"`
	TrackID     string    `json:"track_id"`
	Source      string    `json:"source"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	Altitude    float64   `json:"altitude"`
	VelocityX   float64   `json:"velocity_x"`
	VelocityY   float64   `json:"velocity_y"`
	VelocityZ   float64   `json:"velocity_z"`
	Identity    string    `json:"identity"`
	Quality     string    `json:"quality"`
	Confidence  float64   `json:"confidence"`
	LastUpdate  time.Time `json:"last_update"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// TrackCache provides track caching operations
type TrackCache struct {
	cache *Cache
	ttl   time.Duration
}

// NewTrackCache creates a new track cache
func NewTrackCache(cache *Cache, ttl time.Duration) *TrackCache {
	return &TrackCache{
		cache: cache,
		ttl:   ttl,
	}
}

// Get retrieves track state from cache
func (tc *TrackCache) Get(ctx context.Context, trackID string) (*TrackState, error) {
	key := tc.trackKey(trackID)
	data, err := tc.cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var state TrackState
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal track state: %w", err)
	}

	// Check expiration
	if time.Now().After(state.ExpiresAt) {
		return nil, fmt.Errorf("track state expired")
	}

	return &state, nil
}

// Set stores track state in cache
func (tc *TrackCache) Set(ctx context.Context, state *TrackState) error {
	key := tc.trackKey(state.TrackID)
	state.ExpiresAt = time.Now().Add(tc.ttl)

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal track state: %w", err)
	}

	return tc.cache.SetString(ctx, key, string(data), tc.ttl)
}

// Delete removes track state from cache
func (tc *TrackCache) Delete(ctx context.Context, trackID string) error {
	key := tc.trackKey(trackID)
	return tc.cache.Delete(ctx, key)
}

// GetMultiple retrieves multiple track states
func (tc *TrackCache) GetMultiple(ctx context.Context, trackIDs []string) (map[string]*TrackState, error) {
	states := make(map[string]*TrackState)
	for _, trackID := range trackIDs {
		state, err := tc.Get(ctx, trackID)
		if err != nil {
			continue // Skip missing tracks
		}
		states[trackID] = state
	}
	return states, nil
}

// SetMultiple stores multiple track states
func (tc *TrackCache) SetMultiple(ctx context.Context, states []*TrackState) error {
	for _, state := range states {
		if err := tc.Set(ctx, state); err != nil {
			return err
		}
	}
	return nil
}

// GetBySource retrieves all tracks for a source
func (tc *TrackCache) GetBySource(ctx context.Context, source string) ([]*TrackState, error) {
	// Get track IDs from source set
	trackIDs, err := tc.cache.SMembers(ctx, tc.sourceKey(source))
	if err != nil {
		return nil, err
	}

	var states []*TrackState
	for _, trackID := range trackIDs {
		state, err := tc.Get(ctx, trackID)
		if err != nil {
			continue
		}
		states = append(states, state)
	}

	return states, nil
}

// AddToSource adds track to source index
func (tc *TrackCache) AddToSource(ctx context.Context, source, trackID string) error {
	return tc.cache.SAdd(ctx, tc.sourceKey(source), trackID)
}

// RemoveFromSource removes track from source index
func (tc *TrackCache) RemoveFromSource(ctx context.Context, source, trackID string) error {
	return tc.cache.SRem(ctx, tc.sourceKey(source), trackID)
}

// Invalidate invalidates track cache
func (tc *TrackCache) Invalidate(ctx context.Context, trackID string) error {
	return tc.Delete(ctx, trackID)
}

// InvalidateAll invalidates all tracks for a source
func (tc *TrackCache) InvalidateAll(ctx context.Context, source string) error {
	// Get all track IDs for source
	trackIDs, err := tc.cache.SMembers(ctx, tc.sourceKey(source))
	if err != nil {
		return err
	}

	// Delete all tracks
	for _, trackID := range trackIDs {
		tc.Delete(ctx, trackID)
	}

	// Delete source set
	return tc.cache.Delete(ctx, tc.sourceKey(source))
}

// trackKey returns the cache key for a track
func (tc *TrackCache) trackKey(trackID string) string {
	return fmt.Sprintf("track:%s", trackID)
}

// sourceKey returns the cache key for a source
func (tc *TrackCache) sourceKey(source string) string {
	return fmt.Sprintf("source:%s:tracks", source)
}

// TrackCacheStats represents cache statistics
type TrackCacheStats struct {
	TotalTracks   int64 `json:"total_tracks"`
	TotalSources  int64 `json:"total_sources"`
	CacheHits     int64 `json:"cache_hits"`
	CacheMisses   int64 `json:"cache_misses"`
	Evictions     int64 `json:"evictions"`
}

// GetStats returns cache statistics
func (tc *TrackCache) GetStats(ctx context.Context) (*TrackCacheStats, error) {
	// This would require Redis INFO command or tracking in application
	// For now, return basic stats
	return &TrackCacheStats{
		TotalTracks:  0,
		TotalSources: 0,
		CacheHits:    0,
		CacheMisses:  0,
		Evictions:    0,
	}, nil
}