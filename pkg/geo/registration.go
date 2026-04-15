// Package geo provides geospatial utilities for sensor registration
package geo

import (
	"math"
	"sync"
	"time"
)

// SensorRegistration handles sensor bias estimation and correction
type SensorRegistration struct {
	config    *RegistrationConfig
	sensors   map[string]*SensorBias
	residuals map[string][]Residual
	mu        sync.RWMutex
}

// RegistrationConfig holds configuration for sensor registration
type RegistrationConfig struct {
	// Bias estimation
	MinSamples      int     `json:"min_samples"`      // Minimum samples for bias estimation
	MaxSamples      int     `json:"max_samples"`      // Maximum samples to retain
	BiasThreshold   float64 `json:"bias_threshold"`   // Threshold for bias correction (meters)
	ConfidenceLevel float64 `json:"confidence_level"` // Confidence level for estimation (0-1)

	// Adaptive estimation
	AdaptiveRate       float64 `json:"adaptive_rate"`       // Learning rate for bias updates
	MaxBiasDrift       float64 `json:"max_bias_drift"`      // Maximum allowed drift per update (meters)
	StabilityThreshold float64 `json:"stability_threshold"` // Threshold for stable estimation

	// Coordinate systems
	UseGeodetic bool `json:"use_geodetic"` // Use geodetic coordinates (lat/lon/alt)
	UseECEF     bool `json:"use_ecef"`     // Use ECEF coordinates
}

// SensorBias represents estimated bias for a sensor
type SensorBias struct {
	SensorID      string     `json:"sensor_id"`
	PositionBias  [3]float64 `json:"position_bias"`   // x, y, z bias (meters)
	VelocityBias  [3]float64 `json:"velocity_bias"`   // vx, vy, vz bias (m/s)
	RangeBias     float64    `json:"range_bias"`      // Range bias (meters)
	RangeRateBias float64    `json:"range_rate_bias"` // Range rate bias (m/s)
	AngleBias     [2]float64 `json:"angle_bias"`      // Azimuth, elevation bias (radians)
	LastUpdate    time.Time  `json:"last_update"`
	NumSamples    int        `json:"num_samples"`
	Variance      [6]float64 `json:"variance"`   // Position and velocity variance
	Status        string     `json:"status"`     // ESTIMATING, STABLE, UNSTABLE
	Confidence    float64    `json:"confidence"` // Estimation confidence (0-1)
}

// Residual represents measurement residual
type Residual struct {
	SensorID     string     `json:"sensor_id"`
	Timestamp    time.Time  `json:"timestamp"`
	MeasuredPos  [3]float64 `json:"measured_pos"`
	EstimatedPos [3]float64 `json:"estimated_pos"`
	ResidualPos  [3]float64 `json:"residual_pos"`
	Weight       float64    `json:"weight"`
	UsedForBias  bool       `json:"used_for_bias"`
}

// DefaultRegistrationConfig returns default configuration
func DefaultRegistrationConfig() *RegistrationConfig {
	return &RegistrationConfig{
		MinSamples:         10,
		MaxSamples:         100,
		BiasThreshold:      50.0, // 50 meters
		ConfidenceLevel:    0.95,
		AdaptiveRate:       0.1,
		MaxBiasDrift:       10.0, // 10 meters per update
		StabilityThreshold: 5.0,  // 5 meters
		UseGeodetic:        false,
		UseECEF:            true,
	}
}

// NewSensorRegistration creates a new sensor registration manager
func NewSensorRegistration(config *RegistrationConfig) *SensorRegistration {
	if config == nil {
		config = DefaultRegistrationConfig()
	}

	return &SensorRegistration{
		config:    config,
		sensors:   make(map[string]*SensorBias),
		residuals: make(map[string][]Residual),
	}
}

// InitializeSensor initializes bias estimation for a sensor
func (sr *SensorRegistration) InitializeSensor(sensorID string, now time.Time) *SensorBias {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	bias := &SensorBias{
		SensorID:      sensorID,
		PositionBias:  [3]float64{0, 0, 0},
		VelocityBias:  [3]float64{0, 0, 0},
		RangeBias:     0,
		RangeRateBias: 0,
		AngleBias:     [2]float64{0, 0},
		LastUpdate:    now,
		NumSamples:    0,
		Variance:      [6]float64{100, 100, 100, 10, 10, 10},
		Status:        "ESTIMATING",
		Confidence:    0.0,
	}

	sr.sensors[sensorID] = bias
	sr.residuals[sensorID] = make([]Residual, 0)

	return bias
}

// AddResidual adds a measurement residual for bias estimation
func (sr *SensorRegistration) AddResidual(sensorID string, measured, estimated [3]float64, timestamp time.Time) {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	// Initialize sensor if needed
	if _, exists := sr.sensors[sensorID]; !exists {
		sr.sensors[sensorID] = &SensorBias{
			SensorID:   sensorID,
			LastUpdate: timestamp,
			Status:     "ESTIMATING",
		}
		sr.residuals[sensorID] = make([]Residual, 0)
	}

	// Calculate residual
	residual := Residual{
		SensorID:     sensorID,
		Timestamp:    timestamp,
		MeasuredPos:  measured,
		EstimatedPos: estimated,
		ResidualPos: [3]float64{
			estimated[0] - measured[0],
			estimated[1] - measured[1],
			estimated[2] - measured[2],
		},
		Weight:      1.0,
		UsedForBias: true,
	}

	// Add to residuals
	sr.residuals[sensorID] = append(sr.residuals[sensorID], residual)

	// Limit residuals
	if len(sr.residuals[sensorID]) > sr.config.MaxSamples {
		sr.residuals[sensorID] = sr.residuals[sensorID][1:]
	}

	// Update bias estimation
	sr.updateBiasEstimation(sensorID)
}

// updateBiasEstimation updates bias estimation for a sensor
func (sr *SensorRegistration) updateBiasEstimation(sensorID string) {
	residuals := sr.residuals[sensorID]
	bias := sr.sensors[sensorID]

	if len(residuals) < sr.config.MinSamples {
		bias.NumSamples = len(residuals)
		return
	}

	// Calculate weighted mean residual
	sumWeight := 0.0
	sumResidual := [3]float64{0, 0, 0}

	for _, r := range residuals {
		// Weight based on recency
		age := time.Since(r.Timestamp).Seconds()
		weight := math.Exp(-age / 60.0) // Decay over 60 seconds

		sumWeight += weight
		sumResidual[0] += r.ResidualPos[0] * weight
		sumResidual[1] += r.ResidualPos[1] * weight
		sumResidual[2] += r.ResidualPos[2] * weight
	}

	if sumWeight == 0 {
		return
	}

	meanResidual := [3]float64{
		sumResidual[0] / sumWeight,
		sumResidual[1] / sumWeight,
		sumResidual[2] / sumWeight,
	}

	// Calculate variance
	variance := [3]float64{0, 0, 0}
	for _, r := range residuals {
		variance[0] += math.Pow(r.ResidualPos[0]-meanResidual[0], 2)
		variance[1] += math.Pow(r.ResidualPos[1]-meanResidual[1], 2)
		variance[2] += math.Pow(r.ResidualPos[2]-meanResidual[2], 2)
	}
	variance[0] /= float64(len(residuals))
	variance[1] /= float64(len(residuals))
	variance[2] /= float64(len(residuals))

	// Adaptive bias update
	biasChange := [3]float64{
		(meanResidual[0] - bias.PositionBias[0]) * sr.config.AdaptiveRate,
		(meanResidual[1] - bias.PositionBias[1]) * sr.config.AdaptiveRate,
		(meanResidual[2] - bias.PositionBias[2]) * sr.config.AdaptiveRate,
	}

	// Limit maximum change
	for i := 0; i < 3; i++ {
		if math.Abs(biasChange[i]) > sr.config.MaxBiasDrift {
			biasChange[i] = math.Copysign(sr.config.MaxBiasDrift, biasChange[i])
		}
	}

	// Update bias
	bias.PositionBias[0] += biasChange[0]
	bias.PositionBias[1] += biasChange[1]
	bias.PositionBias[2] += biasChange[2]
	bias.Variance[0] = variance[0]
	bias.Variance[1] = variance[1]
	bias.Variance[2] = variance[2]
	bias.NumSamples = len(residuals)
	bias.LastUpdate = time.Now()

	// Update status
	magnitude := math.Sqrt(bias.PositionBias[0]*bias.PositionBias[0] +
		bias.PositionBias[1]*bias.PositionBias[1] +
		bias.PositionBias[2]*bias.PositionBias[2])

	if magnitude < sr.config.StabilityThreshold {
		bias.Status = "STABLE"
		bias.Confidence = sr.config.ConfidenceLevel
	} else if magnitude > sr.config.BiasThreshold {
		bias.Status = "UNSTABLE"
		bias.Confidence = 0.0
	}
}

// CorrectPosition applies bias correction to a position
func (sr *SensorRegistration) CorrectPosition(sensorID string, position [3]float64) [3]float64 {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	bias, exists := sr.sensors[sensorID]
	if !exists || bias.Status == "ESTIMATING" {
		return position
	}

	// Apply correction (subtract bias)
	return [3]float64{
		position[0] - bias.PositionBias[0],
		position[1] - bias.PositionBias[1],
		position[2] - bias.PositionBias[2],
	}
}

// CorrectVelocity applies bias correction to a velocity
func (sr *SensorRegistration) CorrectVelocity(sensorID string, velocity [3]float64) [3]float64 {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	bias, exists := sr.sensors[sensorID]
	if !exists || bias.Status == "ESTIMATING" {
		return velocity
	}

	return [3]float64{
		velocity[0] - bias.VelocityBias[0],
		velocity[1] - bias.VelocityBias[1],
		velocity[2] - bias.VelocityBias[2],
	}
}

// CorrectRange applies range bias correction
func (sr *SensorRegistration) CorrectRange(sensorID string, rangeVal float64) float64 {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	bias, exists := sr.sensors[sensorID]
	if !exists || bias.Status == "ESTIMATING" {
		return rangeVal
	}

	return rangeVal - bias.RangeBias
}

// CorrectAngles applies angle bias correction
func (sr *SensorRegistration) CorrectAngles(sensorID string, azimuth, elevation float64) (float64, float64) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	bias, exists := sr.sensors[sensorID]
	if !exists || bias.Status == "ESTIMATING" {
		return azimuth, elevation
	}

	return azimuth - bias.AngleBias[0], elevation - bias.AngleBias[1]
}

// GetBias returns current bias estimate for a sensor
func (sr *SensorRegistration) GetBias(sensorID string) *SensorBias {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	return sr.sensors[sensorID]
}

// GetAllBiases returns all sensor biases
func (sr *SensorRegistration) GetAllBiases() map[string]*SensorBias {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	result := make(map[string]*SensorBias)
	for k, v := range sr.sensors {
		result[k] = v
	}
	return result
}

// GetResiduals returns residuals for a sensor
func (sr *SensorRegistration) GetResiduals(sensorID string) []Residual {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	return sr.residuals[sensorID]
}

// GetStableSensors returns sensors with stable bias estimates
func (sr *SensorRegistration) GetStableSensors() []string {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	stable := make([]string, 0)
	for sensorID, bias := range sr.sensors {
		if bias.Status == "STABLE" {
			stable = append(stable, sensorID)
		}
	}
	return stable
}

// ResetBias resets bias estimation for a sensor
func (sr *SensorRegistration) ResetBias(sensorID string) {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	if bias, exists := sr.sensors[sensorID]; exists {
		bias.PositionBias = [3]float64{0, 0, 0}
		bias.VelocityBias = [3]float64{0, 0, 0}
		bias.RangeBias = 0
		bias.RangeRateBias = 0
		bias.AngleBias = [2]float64{0, 0}
		bias.NumSamples = 0
		bias.Status = "ESTIMATING"
		bias.Confidence = 0.0
	}

	sr.residuals[sensorID] = make([]Residual, 0)
}

// RemoveSensor removes a sensor from registration
func (sr *SensorRegistration) RemoveSensor(sensorID string) {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	delete(sr.sensors, sensorID)
	delete(sr.residuals, sensorID)
}

// Stats returns registration statistics
func (sr *SensorRegistration) Stats() RegistrationStats {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	estimating := 0
	stable := 0
	unstable := 0

	for _, bias := range sr.sensors {
		switch bias.Status {
		case "ESTIMATING":
			estimating++
		case "STABLE":
			stable++
		case "UNSTABLE":
			unstable++
		}
	}

	return RegistrationStats{
		TotalSensors: len(sr.sensors),
		Estimating:   estimating,
		Stable:       stable,
		Unstable:     unstable,
	}
}

// RegistrationStats holds registration statistics
type RegistrationStats struct {
	TotalSensors int `json:"total_sensors"`
	Estimating   int `json:"estimating"`
	Stable       int `json:"stable"`
	Unstable     int `json:"unstable"`
}

// CalculateRMS calculates RMS of position residuals
func (sr *SensorRegistration) CalculateRMS(sensorID string) float64 {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	residuals := sr.residuals[sensorID]
	if len(residuals) == 0 {
		return 0
	}

	sumSq := 0.0
	for _, r := range residuals {
		sumSq += r.ResidualPos[0]*r.ResidualPos[0] +
			r.ResidualPos[1]*r.ResidualPos[1] +
			r.ResidualPos[2]*r.ResidualPos[2]
	}

	return math.Sqrt(sumSq / float64(len(residuals)))
}

// CalculateBiasMagnitude calculates magnitude of position bias
func (sr *SensorRegistration) CalculateBiasMagnitude(sensorID string) float64 {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	bias, exists := sr.sensors[sensorID]
	if !exists {
		return 0
	}

	return math.Sqrt(bias.PositionBias[0]*bias.PositionBias[0] +
		bias.PositionBias[1]*bias.PositionBias[1] +
		bias.PositionBias[2]*bias.PositionBias[2])
}
