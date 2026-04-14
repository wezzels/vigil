package c2bmc

import (
	"context"
	"fmt"
	"math"
	"time"
)

// TrackCorrelator handles track correlation with C2BMC
type TrackCorrelator struct {
	client      C2BMCClient
	threshold   float64 // Correlation threshold (0.0-1.0)
	maxDistance float64 // Max distance for correlation (meters)
	timeWindow  time.Duration
}

// NewTrackCorrelator creates a new track correlator
func NewTrackCorrelator(client C2BMCClient, threshold float64) *TrackCorrelator {
	return &TrackCorrelator{
		client:      client,
		threshold:   threshold,
		maxDistance: 5000.0,  // 5km default
		timeWindow: 30 * time.Second,
	}
}

// CorrelationResult represents the result of a correlation attempt
type CorrelationResult struct {
	PrimaryTrack    string
	SecondaryTrack   string
	CorrelationScore float64
	IsCorrelated    bool
	Reason         string
}

// CorrelateByPosition correlates tracks by position proximity
func (tc *TrackCorrelator) CorrelateByPosition(ctx context.Context, primary *TrackData, secondary *TrackData) (*CorrelationResult, error) {
	// Calculate distance between tracks
	distance := calculateDistance(primary.Position, secondary.Position)

	// Check time window
	timeDiff := primary.LastUpdate.Sub(secondary.LastUpdate)
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}

	if timeDiff > tc.timeWindow {
		return &CorrelationResult{
			PrimaryTrack:    primary.TrackNumber,
			SecondaryTrack:   secondary.TrackNumber,
			CorrelationScore: 0,
			IsCorrelated:    false,
			Reason:         fmt.Sprintf("Time difference %v exceeds window %v", timeDiff, tc.timeWindow),
		}, nil
	}

	// Calculate correlation score
	score := 1.0 - (distance / tc.maxDistance)
	if score < 0 {
		score = 0
	}

	// Check velocity similarity
	velocityDiff := calculateVelocityDiff(primary.Velocity, secondary.Velocity)
	velocityScore := 1.0 - velocityDiff/500.0 // 500 m/s threshold
	if velocityScore < 0 {
		velocityScore = 0
	}

	// Combine scores
	finalScore := (score * 0.7) + (velocityScore * 0.3)
	isCorrelated := finalScore >= tc.threshold

	reason := "Distance-based correlation"
	if isCorrelated {
		reason = fmt.Sprintf("Correlated: distance=%.1fm, score=%.2f", distance, finalScore)
	} else {
		reason = fmt.Sprintf("Not correlated: score %.2f below threshold %.2f", finalScore, tc.threshold)
	}

	return &CorrelationResult{
		PrimaryTrack:    primary.TrackNumber,
		SecondaryTrack:   secondary.TrackNumber,
		CorrelationScore: finalScore,
		IsCorrelated:    isCorrelated,
		Reason:         reason,
	}, nil
}

// CorrelateAndSubmit correlates tracks and submits to C2BMC
func (tc *TrackCorrelator) CorrelateAndSubmit(ctx context.Context, primary *TrackData, secondaries []*TrackData) (*TrackCorrelationResponse, error) {
	secondaryIDs := make([]string, len(secondaries))
	for i, t := range secondaries {
		secondaryIDs[i] = t.TrackNumber
	}

	req := &TrackCorrelationRequest{
		PrimaryTrack:    primary.TrackNumber,
		SecondaryTracks: secondaryIDs,
		SourceSystem:   "VIGIL",
		Timestamp:      time.Now(),
	}

	return tc.client.CorrelateTracks(ctx, req)
}

// BatchCorrelate correlates multiple tracks
func (tc *TrackCorrelator) BatchCorrelate(ctx context.Context, tracks []*TrackData) ([]*CorrelationResult, error) {
	results := make([]*CorrelationResult, 0)

	for i := 0; i < len(tracks); i++ {
		for j := i + 1; j < len(tracks); j++ {
			result, err := tc.CorrelateByPosition(ctx, tracks[i], tracks[j])
			if err != nil {
				return nil, fmt.Errorf("correlation failed: %w", err)
			}
			results = append(results, result)
		}
	}

	return results, nil
}

// TrackUpdateHandler handles track updates
type TrackUpdateHandler struct {
	client      C2BMCClient
	updateChan  chan *TrackData
	errorChan   chan error
	batchSize   int
	batchWindow time.Duration
}

// NewTrackUpdateHandler creates a new track update handler
func NewTrackUpdateHandler(client C2BMCClient, batchSize int, batchWindow time.Duration) *TrackUpdateHandler {
	return &TrackUpdateHandler{
		client:      client,
		updateChan:  make(chan *TrackData, 1000),
		errorChan:   make(chan error, 100),
		batchSize:   batchSize,
		batchWindow: batchWindow,
	}
}

// Submit queues a track update for submission
func (h *TrackUpdateHandler) Submit(track *TrackData) {
	h.updateChan <- track
}

// Start starts the batch processing loop
func (h *TrackUpdateHandler) Start(ctx context.Context) {
	go h.processBatch(ctx)
}

// Errors returns the error channel
func (h *TrackUpdateHandler) Errors() <-chan error {
	return h.errorChan
}

// processBatch processes track updates in batches
func (h *TrackUpdateHandler) processBatch(ctx context.Context) {
	batch := make([]*TrackData, 0, h.batchSize)
	timer := time.NewTimer(h.batchWindow)

	for {
		select {
		case <-ctx.Done():
			// Flush remaining
			if len(batch) > 0 {
				h.flushBatch(ctx, batch)
			}
			return

		case track := <-h.updateChan:
			batch = append(batch, track)
			if len(batch) >= h.batchSize {
				h.flushBatch(ctx, batch)
				batch = batch[:0]
				timer.Reset(h.batchWindow)
			}

		case <-timer.C:
			if len(batch) > 0 {
				h.flushBatch(ctx, batch)
				batch = batch[:0]
			}
			timer.Reset(h.batchWindow)
		}
	}
}

// flushBatch flushes a batch of tracks to C2BMC
func (h *TrackUpdateHandler) flushBatch(ctx context.Context, batch []*TrackData) {
	for _, track := range batch {
		if err := h.client.SubmitTrack(ctx, track); err != nil {
			h.errorChan <- fmt.Errorf("failed to submit track %s: %w", track.TrackNumber, err)
		}
	}
}

// TrackStatusQuery queries track status from C2BMC
type TrackStatusQuery struct {
	client C2BMCClient
}

// NewTrackStatusQuery creates a new track status query
func NewTrackStatusQuery(client C2BMCClient) *TrackStatusQuery {
	return &TrackStatusQuery{client: client}
}

// Query queries a single track
func (q *TrackStatusQuery) Query(ctx context.Context, trackID string) (*TrackData, error) {
	return q.client.GetTrack(ctx, trackID)
}

// QueryMultiple queries multiple tracks
func (q *TrackStatusQuery) QueryMultiple(ctx context.Context, trackIDs []string) (map[string]*TrackData, error) {
	results := make(map[string]*TrackData)

	for _, id := range trackIDs {
		track, err := q.client.GetTrack(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to get track %s: %w", id, err)
		}
		results[id] = track
	}

	return results, nil
}

// Helper functions

// calculateDistance calculates distance between two positions in meters
func calculateDistance(p1, p2 Position) float64 {
	// Haversine formula
	const earthRadius = 6371000.0 // meters

	lat1 := p1.Latitude * math.Pi / 180
	lat2 := p2.Latitude * math.Pi / 180
	deltaLat := (p2.Latitude - p1.Latitude) * math.Pi / 180
	deltaLon := (p2.Longitude - p1.Longitude) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	horizontalDist := earthRadius * c

	// Add altitude difference
	altDiff := p2.Altitude - p1.Altitude

	return math.Sqrt(horizontalDist*horizontalDist + altDiff*altDiff)
}

// calculateVelocityDiff calculates velocity difference magnitude
func calculateVelocityDiff(v1, v2 Velocity) float64 {
	dx := v2.Vx - v1.Vx
	dy := v2.Vy - v1.Vy
	dz := v2.Vz - v1.Vz
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}