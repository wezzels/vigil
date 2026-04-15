// Package geo provides time alignment utilities for sensor fusion
package geo

import (
	"math"
	"time"
)

// TimeAligner handles time alignment and interpolation for sensor data
type TimeAligner struct {
	config *TimeAlignConfig
}

// TimeAlignConfig holds configuration for time alignment
type TimeAlignConfig struct {
	// Interpolation
	MaxInterpolationGap time.Duration `json:"max_interpolation_gap"` // Maximum gap for interpolation
	ExtrapolationLimit  time.Duration `json:"extrapolation_limit"`   // Maximum extrapolation time
	InterpolationMethod string        `json:"interpolation_method"`  // "linear", "cubic", "spline"

	// Prediction
	MaxVelocity     float64       `json:"max_velocity"`     // Maximum velocity for prediction (m/s)
	MaxAcceleration float64       `json:"max_acceleration"` // Maximum acceleration (m/s²)
	CoastTime       time.Duration `json:"coast_time"`       // Time before coasting

	// Accuracy
	PositionTolerance float64 `json:"position_tolerance"` // Position tolerance (meters)
	VelocityTolerance float64 `json:"velocity_tolerance"` // Velocity tolerance (m/s)
}

// TimeSeriesPoint represents a point in a time series
type TimeSeriesPoint struct {
	Timestamp time.Time  `json:"timestamp"`
	Position  [3]float64 `json:"position"`
	Velocity  [3]float64 `json:"velocity"`
	Accuracy  float64    `json:"accuracy"`
}

// DefaultTimeAlignConfig returns default configuration
func DefaultTimeAlignConfig() *TimeAlignConfig {
	return &TimeAlignConfig{
		MaxInterpolationGap: 5 * time.Second,
		ExtrapolationLimit:  2 * time.Second,
		InterpolationMethod: "linear",
		MaxVelocity:         1000.0, // 1000 m/s
		MaxAcceleration:     100.0,  // 100 m/s²
		CoastTime:           10 * time.Second,
		PositionTolerance:   10.0, // 10 meters
		VelocityTolerance:   5.0,  // 5 m/s
	}
}

// NewTimeAligner creates a new time aligner
func NewTimeAligner(config *TimeAlignConfig) *TimeAligner {
	if config == nil {
		config = DefaultTimeAlignConfig()
	}
	return &TimeAligner{config: config}
}

// InterpolatePosition interpolates position at a given time
func (ta *TimeAligner) InterpolatePosition(points []TimeSeriesPoint, targetTime time.Time) *TimeSeriesPoint {
	if len(points) == 0 {
		return nil
	}

	// Single point - extrapolate from it
	if len(points) == 1 {
		// Check if forward or backward extrapolation
		if targetTime.After(points[0].Timestamp) {
			return ta.extrapolateForward(&points[0], targetTime)
		} else {
			return ta.extrapolateBackward(&points[0], targetTime)
		}
	}

	// Find bracketing points
	var before, after *TimeSeriesPoint
	for i := range points {
		if points[i].Timestamp.Before(targetTime) || points[i].Timestamp.Equal(targetTime) {
			before = &points[i]
		}
		if points[i].Timestamp.After(targetTime) && after == nil {
			after = &points[i]
		}
	}

	// Exact match
	if before != nil && before.Timestamp.Equal(targetTime) {
		return before
	}

	// Extrapolation before first point
	if before == nil && after != nil {
		return ta.extrapolateBackward(after, targetTime)
	}

	// Extrapolation after last point
	if before != nil && after == nil {
		return ta.extrapolateForward(before, targetTime)
	}

	// Interpolation between points
	return ta.interpolateLinear(before, after, targetTime)
}

// interpolateLinear performs linear interpolation
func (ta *TimeAligner) interpolateLinear(before, after *TimeSeriesPoint, targetTime time.Time) *TimeSeriesPoint {
	// Check gap
	gap := after.Timestamp.Sub(before.Timestamp)
	if gap > ta.config.MaxInterpolationGap {
		return nil // Gap too large
	}

	// Calculate interpolation factor
	totalGap := after.Timestamp.Sub(before.Timestamp)
	if totalGap == 0 {
		return before
	}

	elapsed := targetTime.Sub(before.Timestamp)
	alpha := float64(elapsed) / float64(totalGap)

	// Interpolate position
	position := [3]float64{
		before.Position[0] + alpha*(after.Position[0]-before.Position[0]),
		before.Position[1] + alpha*(after.Position[1]-before.Position[1]),
		before.Position[2] + alpha*(after.Position[2]-before.Position[2]),
	}

	// Interpolate velocity
	velocity := [3]float64{
		before.Velocity[0] + alpha*(after.Velocity[0]-before.Velocity[0]),
		before.Velocity[1] + alpha*(after.Velocity[1]-before.Velocity[1]),
		before.Velocity[2] + alpha*(after.Velocity[2]-before.Velocity[2]),
	}

	// Interpolate accuracy (worst case)
	accuracy := math.Max(before.Accuracy, after.Accuracy)

	return &TimeSeriesPoint{
		Timestamp: targetTime,
		Position:  position,
		Velocity:  velocity,
		Accuracy:  accuracy,
	}
}

// extrapolateForward extrapolates forward from last known position
func (ta *TimeAligner) extrapolateForward(last *TimeSeriesPoint, targetTime time.Time) *TimeSeriesPoint {
	elapsed := targetTime.Sub(last.Timestamp)

	// Check extrapolation limit
	if elapsed > ta.config.ExtrapolationLimit {
		return nil // Too far to extrapolate
	}

	elapsedSec := elapsed.Seconds()

	// Constant velocity extrapolation
	position := [3]float64{
		last.Position[0] + last.Velocity[0]*elapsedSec,
		last.Position[1] + last.Velocity[1]*elapsedSec,
		last.Position[2] + last.Velocity[2]*elapsedSec,
	}

	// Uncertainty increases with extrapolation
	uncertainty := last.Accuracy * (1.0 + elapsedSec/float64(ta.config.ExtrapolationLimit.Seconds()))

	return &TimeSeriesPoint{
		Timestamp: targetTime,
		Position:  position,
		Velocity:  last.Velocity,
		Accuracy:  uncertainty,
	}
}

// extrapolateBackward extrapolates backward from first known position
func (ta *TimeAligner) extrapolateBackward(first *TimeSeriesPoint, targetTime time.Time) *TimeSeriesPoint {
	elapsed := first.Timestamp.Sub(targetTime)

	// Check extrapolation limit
	if elapsed > ta.config.ExtrapolationLimit {
		return nil
	}

	elapsedSec := elapsed.Seconds()

	// Constant velocity extrapolation backward
	position := [3]float64{
		first.Position[0] - first.Velocity[0]*elapsedSec,
		first.Position[1] - first.Velocity[1]*elapsedSec,
		first.Position[2] - first.Velocity[2]*elapsedSec,
	}

	uncertainty := first.Accuracy * (1.0 + elapsedSec/float64(ta.config.ExtrapolationLimit.Seconds()))

	return &TimeSeriesPoint{
		Timestamp: targetTime,
		Position:  position,
		Velocity:  first.Velocity,
		Accuracy:  uncertainty,
	}
}

// InterpolateTrack interpolates a full track to target times
func (ta *TimeAligner) InterpolateTrack(points []TimeSeriesPoint, targetTimes []time.Time) []TimeSeriesPoint {
	result := make([]TimeSeriesPoint, 0, len(targetTimes))

	for _, targetTime := range targetTimes {
		point := ta.InterpolatePosition(points, targetTime)
		if point != nil {
			result = append(result, *point)
		}
	}

	return result
}

// AlignTracks aligns multiple tracks to common time grid
func (ta *TimeAligner) AlignTracks(tracks map[string][]TimeSeriesPoint, targetTimes []time.Time) map[string][]TimeSeriesPoint {
	result := make(map[string][]TimeSeriesPoint)

	for sensorID, track := range tracks {
		aligned := ta.InterpolateTrack(track, targetTimes)
		if len(aligned) > 0 {
			result[sensorID] = aligned
		}
	}

	return result
}

// PredictPosition predicts position at a future time
func (ta *TimeAligner) PredictPosition(point *TimeSeriesPoint, futureTime time.Time) *TimeSeriesPoint {
	elapsed := futureTime.Sub(point.Timestamp)

	// Check prediction limit
	if elapsed > ta.config.CoastTime {
		return nil // Too far to predict
	}

	elapsedSec := elapsed.Seconds()

	// Constant velocity prediction
	position := [3]float64{
		point.Position[0] + point.Velocity[0]*elapsedSec,
		point.Position[1] + point.Velocity[1]*elapsedSec,
		point.Position[2] + point.Velocity[2]*elapsedSec,
	}

	// Uncertainty increases with prediction
	uncertainty := point.Accuracy * (1.0 + elapsedSec/float64(ta.config.CoastTime.Seconds()))

	return &TimeSeriesPoint{
		Timestamp: futureTime,
		Position:  position,
		Velocity:  point.Velocity,
		Accuracy:  uncertainty,
	}
}

// CalculateVelocity estimates velocity from consecutive points
func (ta *TimeAligner) CalculateVelocity(points []TimeSeriesPoint) []TimeSeriesPoint {
	if len(points) < 2 {
		return points
	}

	result := make([]TimeSeriesPoint, len(points))

	for i := range points {
		result[i].Timestamp = points[i].Timestamp
		result[i].Position = points[i].Position
		result[i].Accuracy = points[i].Accuracy

		if i < len(points)-1 {
			// Calculate velocity from current to next point
			dt := points[i+1].Timestamp.Sub(points[i].Timestamp).Seconds()
			if dt > 0 {
				result[i].Velocity = [3]float64{
					(points[i+1].Position[0] - points[i].Position[0]) / dt,
					(points[i+1].Position[1] - points[i].Position[1]) / dt,
					(points[i+1].Position[2] - points[i].Position[2]) / dt,
				}
			} else {
				result[i].Velocity = [3]float64{0, 0, 0}
			}
		} else {
			// Last point uses previous velocity
			result[i].Velocity = result[i-1].Velocity
		}
	}

	return result
}

// SmoothTrack applies smoothing to a track
func (ta *TimeAligner) SmoothTrack(points []TimeSeriesPoint, windowSize int) []TimeSeriesPoint {
	if len(points) < windowSize {
		return points
	}

	result := make([]TimeSeriesPoint, len(points))
	halfWindow := windowSize / 2

	for i := range points {
		result[i].Timestamp = points[i].Timestamp

		// Average position in window
		start := i - halfWindow
		if start < 0 {
			start = 0
		}
		end := i + halfWindow + 1
		if end > len(points) {
			end = len(points)
		}

		var sumX, sumY, sumZ float64
		count := 0
		for j := start; j < end; j++ {
			sumX += points[j].Position[0]
			sumY += points[j].Position[1]
			sumZ += points[j].Position[2]
			count++
		}

		result[i].Position = [3]float64{
			sumX / float64(count),
			sumY / float64(count),
			sumZ / float64(count),
		}

		// Average velocity in window
		var sumVx, sumVy, sumVz float64
		for j := start; j < end; j++ {
			sumVx += points[j].Velocity[0]
			sumVy += points[j].Velocity[1]
			sumVz += points[j].Velocity[2]
		}

		result[i].Velocity = [3]float64{
			sumVx / float64(count),
			sumVy / float64(count),
			sumVz / float64(count),
		}

		// Accuracy improves with averaging
		result[i].Accuracy = points[i].Accuracy / math.Sqrt(float64(count))
	}

	return result
}

// ResampleTrack resamples track to uniform time intervals
func (ta *TimeAligner) ResampleTrack(points []TimeSeriesPoint, interval time.Duration) []TimeSeriesPoint {
	if len(points) == 0 {
		return nil
	}

	startTime := points[0].Timestamp
	endTime := points[len(points)-1].Timestamp

	var targetTimes []time.Time
	for t := startTime; !t.After(endTime); t = t.Add(interval) {
		targetTimes = append(targetTimes, t)
	}

	return ta.InterpolateTrack(points, targetTimes)
}

// ValidateTrack checks if track is valid
func (ta *TimeAligner) ValidateTrack(points []TimeSeriesPoint) (bool, string) {
	if len(points) == 0 {
		return false, "empty track"
	}

	for i := 1; i < len(points); i++ {
		// Check time ordering
		if !points[i].Timestamp.After(points[i-1].Timestamp) {
			return false, "points not in time order"
		}

		// Check position jump
		dx := points[i].Position[0] - points[i-1].Position[0]
		dy := points[i].Position[1] - points[i-1].Position[1]
		dz := points[i].Position[2] - points[i-1].Position[2]
		distance := math.Sqrt(dx*dx + dy*dy + dz*dz)

		dt := points[i].Timestamp.Sub(points[i-1].Timestamp).Seconds()
		if dt > 0 {
			velocity := distance / dt
			if velocity > ta.config.MaxVelocity {
				return false, "velocity exceeds maximum"
			}
		}
	}

	return true, ""
}

// EstimateAccuracy estimates accuracy from track points
func (ta *TimeAligner) EstimateAccuracy(points []TimeSeriesPoint) float64 {
	if len(points) == 0 {
		return 0
	}

	var sumAccuracy float64
	for _, p := range points {
		sumAccuracy += p.Accuracy
	}

	return sumAccuracy / float64(len(points))
}

// GetInterpolationStats returns statistics about interpolation quality
func (ta *TimeAligner) GetInterpolationStats(points []TimeSeriesPoint, targetTimes []time.Time) InterpolationStats {
	stats := InterpolationStats{
		TotalTargets: len(targetTimes),
	}

	for _, t := range targetTimes {
		point := ta.InterpolatePosition(points, t)
		if point == nil {
			stats.FailedInterpolations++
		} else {
			stats.SuccessfulInterpolations++
			stats.TotalAccuracy += point.Accuracy
		}
	}

	if stats.SuccessfulInterpolations > 0 {
		stats.AverageAccuracy = stats.TotalAccuracy / float64(stats.SuccessfulInterpolations)
	}

	return stats
}

// InterpolationStats holds interpolation statistics
type InterpolationStats struct {
	TotalTargets             int     `json:"total_targets"`
	SuccessfulInterpolations int     `json:"successful_interpolations"`
	FailedInterpolations     int     `json:"failed_interpolations"`
	TotalAccuracy            float64 `json:"total_accuracy"`
	AverageAccuracy          float64 `json:"average_accuracy"`
}
