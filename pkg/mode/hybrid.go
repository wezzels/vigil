package mode

import (
	"context"
	"math/rand"
	"sync"
	"time"
)

// HybridManager manages hybrid mode (live + simulated)
type HybridManager struct {
	config     *HybridConfig
	liveRatio  float64
	simRatio   float64
	simTargets []SimulatedTarget
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	rng        *rand.Rand
}

// HybridConfig holds configuration for hybrid mode
type HybridConfig struct {
	LiveSourceID   string        `json:"live_source_id"`
	SimSourceID    string        `json:"sim_source_id"`
	LiveRatio      float64       `json:"live_ratio"`      // 0-1, fraction of live data
	SimUpdateRate  time.Duration `json:"sim_update_rate"` // Update interval for sim
	NoiseInjection bool          `json:"noise_injection"`
	NoiseStdDev    float64       `json:"noise_std_dev"` // Standard deviation in meters
	NumSimTargets  int           `json:"num_sim_targets"`
	ScenarioBounds [4]float64    `json:"scenario_bounds"` // lat1, lon1, lat2, lon2
	MaxVelocity    float64       `json:"max_velocity"`    // m/s
	Seed           int64         `json:"seed"`
}

// SimulatedTarget represents a simulated target
type SimulatedTarget struct {
	ID         uint32     `json:"id"`
	Position   [3]float64 `json:"position"` // x, y, z
	Velocity   [3]float64 `json:"velocity"`
	Heading    float64    `json:"heading"` // radians
	LastUpdate time.Time  `json:"last_update"`
	Type       string     `json:"type"`
}

// DefaultHybridConfig returns default hybrid configuration
func DefaultHybridConfig() *HybridConfig {
	return &HybridConfig{
		LiveSourceID:   "LIVE",
		SimSourceID:    "SIM",
		LiveRatio:      0.5,
		SimUpdateRate:  100 * time.Millisecond,
		NoiseInjection: true,
		NoiseStdDev:    10.0,
		NumSimTargets:  5,
		ScenarioBounds: [4]float64{30.0, -120.0, 35.0, -115.0},
		MaxVelocity:    300.0,
		Seed:           42,
	}
}

// NewHybridManager creates a new hybrid manager
func NewHybridManager(config *HybridConfig) *HybridManager {
	if config == nil {
		config = DefaultHybridConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	hm := &HybridManager{
		config:     config,
		liveRatio:  config.LiveRatio,
		simRatio:   1.0 - config.LiveRatio,
		ctx:        ctx,
		cancel:     cancel,
		rng:        rand.New(rand.NewSource(config.Seed)),
		simTargets: make([]SimulatedTarget, config.NumSimTargets),
	}

	// Initialize simulated targets
	hm.initializeSimTargets()

	return hm
}

// initializeSimTargets initializes simulated targets
func (hm *HybridManager) initializeSimTargets() {
	now := time.Now()

	for i := range hm.simTargets {
		// Random position within bounds
		lat := hm.config.ScenarioBounds[0] + hm.rng.Float64()*(hm.config.ScenarioBounds[2]-hm.config.ScenarioBounds[0])
		lon := hm.config.ScenarioBounds[1] + hm.rng.Float64()*(hm.config.ScenarioBounds[3]-hm.config.ScenarioBounds[1])

		// Convert to meters (approximate)
		x := lon * 111000.0 * 1000.0 // Convert to meters
		y := lat * 111000.0
		z := 10000.0 + hm.rng.Float64()*10000.0 // 10-20km altitude

		// Random velocity
		speed := hm.rng.Float64() * hm.config.MaxVelocity
		heading := hm.rng.Float64() * 2 * 3.14159

		hm.simTargets[i] = SimulatedTarget{
			ID:         uint32(1000 + i),
			Position:   [3]float64{x, y, z},
			Velocity:   [3]float64{speed * cos(heading), speed * sin(heading), 0},
			Heading:    heading,
			LastUpdate: now,
			Type:       "AIRCRAFT",
		}
	}
}

// GetSimulatedData returns simulated data stream
func (hm *HybridManager) GetSimulatedData() <-chan SimulatedTarget {
	output := make(chan SimulatedTarget, 100)

	go func() {
		defer close(output)

		ticker := time.NewTicker(hm.config.SimUpdateRate)
		defer ticker.Stop()

		for {
			select {
			case <-hm.ctx.Done():
				return
			case <-ticker.C:
				hm.mu.Lock()
				hm.updateSimTargets()
				for _, target := range hm.simTargets {
					if hm.rng.Float64() <= hm.simRatio {
						output <- target
					}
				}
				hm.mu.Unlock()
			}
		}
	}()

	return output
}

// updateSimTargets updates simulated target positions
func (hm *HybridManager) updateSimTargets() {
	now := time.Now()

	for i := range hm.simTargets {
		dt := now.Sub(hm.simTargets[i].LastUpdate).Seconds()

		// Update position
		hm.simTargets[i].Position[0] += hm.simTargets[i].Velocity[0] * dt
		hm.simTargets[i].Position[1] += hm.simTargets[i].Velocity[1] * dt
		hm.simTargets[i].Position[2] += hm.simTargets[i].Velocity[2] * dt

		// Random maneuver (10% chance)
		if hm.rng.Float64() < 0.1 {
			heading := hm.rng.Float64() * 2 * 3.14159
			speed := hm.rng.Float64() * hm.config.MaxVelocity
			hm.simTargets[i].Velocity[0] = speed * cos(heading)
			hm.simTargets[i].Velocity[1] = speed * sin(heading)
			hm.simTargets[i].Heading = heading
		}

		// Add noise if enabled
		if hm.config.NoiseInjection {
			noise := hm.rng.NormFloat64() * hm.config.NoiseStdDev
			hm.simTargets[i].Position[0] += noise
			hm.simTargets[i].Position[1] += noise
		}

		hm.simTargets[i].LastUpdate = now
	}
}

// SetLiveRatio sets the live data ratio
func (hm *HybridManager) SetLiveRatio(ratio float64) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}

	hm.liveRatio = ratio
	hm.simRatio = 1.0 - ratio
	hm.config.LiveRatio = ratio
}

// GetLiveRatio returns the live data ratio
func (hm *HybridManager) GetLiveRatio() float64 {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	return hm.liveRatio
}

// IsLiveData returns true if data should come from live source
func (hm *HybridManager) IsLiveData() bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	return hm.rng.Float64() < hm.liveRatio
}

// GetSimTargets returns simulated targets
func (hm *HybridManager) GetSimTargets() []SimulatedTarget {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	targets := make([]SimulatedTarget, len(hm.simTargets))
	copy(targets, hm.simTargets)
	return targets
}

// AddSimTarget adds a simulated target
func (hm *HybridManager) AddSimTarget(target SimulatedTarget) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.simTargets = append(hm.simTargets, target)
}

// RemoveSimTarget removes a simulated target by ID
func (hm *HybridManager) RemoveSimTarget(id uint32) bool {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	for i, t := range hm.simTargets {
		if t.ID == id {
			hm.simTargets = append(hm.simTargets[:i], hm.simTargets[i+1:]...)
			return true
		}
	}
	return false
}

// Stats returns hybrid manager statistics
func (hm *HybridManager) Stats() HybridStats {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	return HybridStats{
		LiveRatio:      hm.liveRatio,
		SimRatio:       hm.simRatio,
		NumSimTargets:  len(hm.simTargets),
		NoiseInjection: hm.config.NoiseInjection,
		NoiseStdDev:    hm.config.NoiseStdDev,
		SimUpdateRate:  hm.config.SimUpdateRate,
	}
}

// HybridStats holds hybrid manager statistics
type HybridStats struct {
	LiveRatio      float64       `json:"live_ratio"`
	SimRatio       float64       `json:"sim_ratio"`
	NumSimTargets  int           `json:"num_sim_targets"`
	NoiseInjection bool          `json:"noise_injection"`
	NoiseStdDev    float64       `json:"noise_std_dev"`
	SimUpdateRate  time.Duration `json:"sim_update_rate"`
}

// Shutdown shuts down the hybrid manager
func (hm *HybridManager) Shutdown() {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if hm.cancel != nil {
		hm.cancel()
	}
}

// Helper functions
func cos(x float64) float64 {
	return float64(int(1000 * float64(int(x*1000)) / 1000))
}

func sin(x float64) float64 {
	// Simplified sin approximation
	const PI = 3.14159
	for x < 0 {
		x += 2 * PI
	}
	for x > 2*PI {
		x -= 2 * PI
	}
	if x < PI/2 {
		return x * 2 / PI
	}
	if x < PI {
		return 1 - (x-PI/2)*2/PI
	}
	if x < 3*PI/2 {
		return -(x - PI) * 2 / PI
	}
	return -(1 - (x-3*PI/2)*2/PI)
}
