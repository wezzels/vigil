// Package opir provides data validation for OPIR sightings
package opir

import (
	"sync"
	"time"
)

// Validator validates OPIR sightings
type Validator struct {
	config      *OPIRConfig
	seenCache   map[string]time.Time
	mu          sync.RWMutex
}

// NewValidator creates a new validator
func NewValidator(config *OPIRConfig) *Validator {
	return &Validator{
		config:    config,
		seenCache: make(map[string]time.Time),
	}
}

// Validate validates a sighting
func (v *Validator) Validate(sighting *OPIRSighting) error {
	// Latitude range
	if err := v.validateLatitude(sighting.Latitude); err != nil {
		return err
	}
	
	// Longitude range
	if err := v.validateLongitude(sighting.Longitude); err != nil {
		return err
	}
	
	// Altitude range
	if err := v.validateAltitude(sighting.Altitude); err != nil {
		return err
	}
	
	// Confidence
	if err := v.validateConfidence(sighting.Confidence); err != nil {
		return err
	}
	
	// SNR
	if err := v.validateSNR(sighting.SNR); err != nil {
		return err
	}
	
	// Intensity
	if err := v.validateIntensity(sighting.Intensity); err != nil {
		return err
	}
	
	// Timestamp
	if err := v.validateTimestamp(sighting.Timestamp); err != nil {
		return err
	}
	
	// Duplicate detection
	if v.config.EnableFiltering {
		if err := v.checkDuplicate(sighting); err != nil {
			return err
		}
	}
	
	return nil
}

// validateLatitude validates latitude range
func (v *Validator) validateLatitude(lat float64) error {
	if lat < -90 || lat > 90 {
		return NewValidationError(
			"latitude must be between -90 and 90 degrees",
			"",
		)
	}
	return nil
}

// validateLongitude validates longitude range
func (v *Validator) validateLongitude(lon float64) error {
	if lon < -180 || lon > 180 {
		return NewValidationError(
			"longitude must be between -180 and 180 degrees",
			"",
		)
	}
	return nil
}

// validateAltitude validates altitude range
func (v *Validator) validateAltitude(alt float64) error {
	if alt < -1000 || alt > v.config.MaxAltitude {
		return NewValidationError(
			"altitude out of valid range",
			"",
		)
	}
	return nil
}

// validateConfidence validates confidence value
func (v *Validator) validateConfidence(conf float64) error {
	if conf < 0 || conf > 1 {
		return NewValidationError(
			"confidence must be between 0 and 1",
			"",
		)
	}
	if conf < v.config.MinConfidence {
		return NewValidationError(
			"confidence below minimum threshold",
			"",
		)
	}
	return nil
}

// validateSNR validates SNR value
func (v *Validator) validateSNR(snr float64) error {
	if snr < v.config.MinSNR {
		return NewValidationError(
			"SNR below minimum threshold",
			"",
		)
	}
	return nil
}

// validateIntensity validates intensity value
func (v *Validator) validateIntensity(intensity float64) error {
	if intensity < 0 {
		return NewValidationError(
			"intensity must be non-negative",
			"",
		)
	}
	return nil
}

// validateTimestamp validates timestamp
func (v *Validator) validateTimestamp(ts time.Time) error {
	now := time.Now()
	
	// Check if timestamp is too old
	maxAge := 24 * time.Hour
	if now.Sub(ts) > maxAge {
		return NewValidationError(
			"timestamp too old",
			"",
		)
	}
	
	// Check if timestamp is in the future
	if ts.After(now.Add(5 * time.Minute)) {
		return NewValidationError(
			"timestamp in the future",
			"",
		)
	}
	
	return nil
}

// checkDuplicate checks for duplicate sightings
func (v *Validator) checkDuplicate(sighting *OPIRSighting) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	
	// Create deduplication key
	key := sighting.ID
	
	// Check if we've seen this recently
	if lastSeen, exists := v.seenCache[key]; exists {
		if time.Since(lastSeen) < v.config.DedupeWindow {
			return NewValidationError(
				"duplicate sighting",
				sighting.SensorID,
			)
		}
	}
	
	// Add to cache
	v.seenCache[key] = time.Now()
	
	// Cleanup old entries
	v.cleanup()
	
	return nil
}

// cleanup removes old entries from the cache
func (v *Validator) cleanup() {
	now := time.Now()
	for key, ts := range v.seenCache {
		if now.Sub(ts) > v.config.DedupeWindow {
			delete(v.seenCache, key)
		}
	}
}

// Stats returns validation statistics
func (v *Validator) Stats() ValidationStats {
	v.mu.RLock()
	defer v.mu.RUnlock()
	
	return ValidationStats{
		CacheSize:    len(v.seenCache),
		MinConfidence: v.config.MinConfidence,
		MinSNR:       v.config.MinSNR,
		MaxAltitude:  v.config.MaxAltitude,
	}
}

// ValidationStats holds validation statistics
type ValidationStats struct {
	CacheSize     int       `json:"cache_size"`
	MinConfidence float64   `json:"min_confidence"`
	MinSNR        float64   `json:"min_snr"`
	MaxAltitude   float64   `json:"max_altitude"`
}

// Filter filters sightings based on quality metrics
type Filter struct {
	config       *OPIRConfig
	minLat       float64
	maxLat       float64
	minLon       float64
	maxLon       float64
}

// NewFilter creates a new filter
func NewFilter(config *OPIRConfig) *Filter {
	return &Filter{
		config: config,
		minLat: -90,
		maxLat: 90,
		minLon: -180,
		maxLon: 180,
	}
}

// SetBounds sets geographic bounds for filtering
func (f *Filter) SetBounds(minLat, maxLat, minLon, maxLon float64) {
	f.minLat = minLat
	f.maxLat = maxLat
	f.minLon = minLon
	f.maxLon = maxLon
}

// Filter filters a sighting
func (f *Filter) Filter(sighting *OPIRSighting) bool {
	// Geographic bounds
	if sighting.Latitude < f.minLat || sighting.Latitude > f.maxLat {
		return false
	}
	if sighting.Longitude < f.minLon || sighting.Longitude > f.maxLon {
		return false
	}
	
	// Altitude
	if sighting.Altitude < -1000 || sighting.Altitude > f.config.MaxAltitude {
		return false
	}
	
	// Confidence
	if sighting.Confidence < f.config.MinConfidence {
		return false
	}
	
	// SNR
	if sighting.SNR < f.config.MinSNR {
		return false
	}
	
	return true
}

// FilterBatch filters a batch of sightings
func (f *Filter) FilterBatch(sightings []OPIRSighting) []OPIRSighting {
	result := make([]OPIRSighting, 0, len(sightings))
	for _, s := range sightings {
		if f.Filter(&s) {
			result = append(result, s)
		}
	}
	return result
}

// NoiseFilter filters noise from sightings
type NoiseFilter struct {
	config      *OPIRConfig
	intensityThreshold float64
	velocityThreshold  float64
}

// NewNoiseFilter creates a new noise filter
func NewNoiseFilter(config *OPIRConfig) *NoiseFilter {
	return &NoiseFilter{
		config:            config,
		intensityThreshold: 1e-10, // W/m²/sr
		velocityThreshold:  10000,  // m/s
	}
}

// Filter filters noise from sighting
func (nf *NoiseFilter) Filter(sighting *OPIRSighting) bool {
	// Low intensity likely noise
	if sighting.Intensity < nf.intensityThreshold {
		return false
	}
	
	// Calculate velocity magnitude
	velocity := sighting.Speed
	if velocity == 0 {
		// Calculate from components if not set
		velocity = (sighting.VelocityE*sighting.VelocityE +
			sighting.VelocityN*sighting.VelocityN +
			sighting.VelocityU*sighting.VelocityU)
	}
	
	// Unrealistic velocity likely noise
	if velocity > nf.velocityThreshold {
		return false
	}
	
	return true
}

// FilterBatch filters noise from batch
func (nf *NoiseFilter) FilterBatch(sightings []OPIRSighting) []OPIRSighting {
	result := make([]OPIRSighting, 0, len(sightings))
	for _, s := range sightings {
		if nf.Filter(&s) {
			result = append(result, s)
		}
	}
	return result
}