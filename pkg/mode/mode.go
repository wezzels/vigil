// Package mode provides operational mode management for VIGIL
package mode

import (
	"context"
	"sync"
	"time"
)

// Mode represents the operational mode
type Mode int

const (
	ModeLive Mode = iota     // Live sensor data
	ModeReplay               // Replay from recorded data
	ModeSimulation           // Simulated data
	ModeHybrid               // Mix of live and simulated
)

// String returns string representation of mode
func (m Mode) String() string {
	switch m {
	case ModeLive:
		return "LIVE"
	case ModeReplay:
		return "REPLAY"
	case ModeSimulation:
		return "SIMULATION"
	case ModeHybrid:
		return "HYBRID"
	default:
		return "UNKNOWN"
	}
}

// MarshalText implements encoding.TextMarshaler
func (m Mode) MarshalText() ([]byte, error) {
	return []byte(m.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler
func (m *Mode) UnmarshalText(text []byte) error {
	switch string(text) {
	case "LIVE", "live":
		*m = ModeLive
	case "REPLAY", "replay":
		*m = ModeReplay
	case "SIMULATION", "simulation":
		*m = ModeSimulation
	case "HYBRID", "hybrid":
		*m = ModeHybrid
	default:
		*m = ModeLive
	}
	return nil
}

// ModeConfig holds configuration for a mode
type ModeConfig struct {
	CurrentMode     Mode          `json:"current_mode"`
	PreviousMode    Mode          `json:"previous_mode"`
	SwitchCooldown  time.Duration `json:"switch_cooldown"`
	AllowHotSwitch  bool          `json:"allow_hot_switch"`
	ReplaySource    string        `json:"replay_source"`
	SimConfig       SimConfig     `json:"sim_config"`
	HybridRatio     float64       `json:"hybrid_ratio"` // 0-1, ratio of live to simulated
}

// SimConfig holds simulation configuration
type SimConfig struct {
	NumTargets      int           `json:"num_targets"`
	ScenarioFile    string        `json:"scenario_file"`
	TimeCompression float64       `json:"time_compression"`
	InjectNoise     bool          `json:"inject_noise"`
	Seed            int64         `json:"seed"`
}

// DefaultModeConfig returns default mode configuration
func DefaultModeConfig() *ModeConfig {
	return &ModeConfig{
		CurrentMode:     ModeLive,
		PreviousMode:    ModeLive,
		SwitchCooldown:  5 * time.Second,
		AllowHotSwitch:  true,
		HybridRatio:     0.5,
		SimConfig: SimConfig{
			NumTargets:      10,
			TimeCompression: 1.0,
			InjectNoise:     true,
			Seed:            42,
		},
	}
}

// ModeManager manages operational modes
type ModeManager struct {
	config     *ModeConfig
	lastSwitch time.Time
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	callbacks  []ModeChangeCallback
}

// ModeChangeCallback is called when mode changes
type ModeChangeCallback func(oldMode, newMode Mode)

// NewModeManager creates a new mode manager
func NewModeManager(config *ModeConfig) *ModeManager {
	if config == nil {
		config = DefaultModeConfig()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &ModeManager{
		config:    config,
		ctx:       ctx,
		cancel:    cancel,
		callbacks: make([]ModeChangeCallback, 0),
	}
}

// GetMode returns current mode
func (mm *ModeManager) GetMode() Mode {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return mm.config.CurrentMode
}

// SetMode switches to a new mode
func (mm *ModeManager) SetMode(newMode Mode) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	// Check cooldown
	if time.Since(mm.lastSwitch) < mm.config.SwitchCooldown && !mm.config.AllowHotSwitch {
		return ErrSwitchCooldown
	}
	
	// Same mode
	if mm.config.CurrentMode == newMode {
		return nil
	}
	
	// Store old mode
	oldMode := mm.config.CurrentMode
	
	// Update mode
	mm.config.PreviousMode = oldMode
	mm.config.CurrentMode = newMode
	mm.lastSwitch = time.Now()
	
	// Notify callbacks
	for _, cb := range mm.callbacks {
		go cb(oldMode, newMode)
	}
	
	return nil
}

// RegisterCallback registers a mode change callback
func (mm *ModeManager) RegisterCallback(cb ModeChangeCallback) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	mm.callbacks = append(mm.callbacks, cb)
}

// GetConfig returns current configuration
func (mm *ModeManager) GetConfig() *ModeConfig {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return mm.config
}

// UpdateConfig updates configuration
func (mm *ModeManager) UpdateConfig(config *ModeConfig) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	mm.config = config
}

// CanSwitch returns true if mode can be switched
func (mm *ModeManager) CanSwitch() bool {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	
	if mm.config.AllowHotSwitch {
		return true
	}
	
	return time.Since(mm.lastSwitch) >= mm.config.SwitchCooldown
}

// TimeUntilSwitch returns time until next switch is allowed
func (mm *ModeManager) TimeUntilSwitch() time.Duration {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	
	if mm.config.AllowHotSwitch {
		return 0
	}
	
	remaining := mm.config.SwitchCooldown - time.Since(mm.lastSwitch)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// RestorePreviousMode switches back to previous mode
func (mm *ModeManager) RestorePreviousMode() error {
	mm.mu.RLock()
	previousMode := mm.config.PreviousMode
	mm.mu.RUnlock()
	
	return mm.SetMode(previousMode)
}

// Stats returns mode manager statistics
func (mm *ModeManager) Stats() ModeStats {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	
	return ModeStats{
		CurrentMode:     mm.config.CurrentMode,
		PreviousMode:    mm.config.PreviousMode,
		LastSwitch:      mm.lastSwitch,
		CanSwitch:       mm.CanSwitch(),
		TimeUntilSwitch: mm.TimeUntilSwitch(),
	}
}

// ModeStats holds mode manager statistics
type ModeStats struct {
	CurrentMode     Mode          `json:"current_mode"`
	PreviousMode    Mode          `json:"previous_mode"`
	LastSwitch      time.Time    `json:"last_switch"`
	CanSwitch       bool         `json:"can_switch"`
	TimeUntilSwitch time.Duration `json:"time_until_switch"`
}

// Context returns the mode manager context
func (mm *ModeManager) Context() context.Context {
	return mm.ctx
}

// Shutdown shuts down the mode manager
func (mm *ModeManager) Shutdown() {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	if mm.cancel != nil {
		mm.cancel()
	}
}

// IsLive returns true if in live mode
func (mm *ModeManager) IsLive() bool {
	return mm.GetMode() == ModeLive
}

// IsReplay returns true if in replay mode
func (mm *ModeManager) IsReplay() bool {
	return mm.GetMode() == ModeReplay
}

// IsSimulation returns true if in simulation mode
func (mm *ModeManager) IsSimulation() bool {
	return mm.GetMode() == ModeSimulation
}

// IsHybrid returns true if in hybrid mode
func (mm *ModeManager) IsHybrid() bool {
	return mm.GetMode() == ModeHybrid
}

// Errors
var (
	ErrSwitchCooldown = &ModeError{Code: "SWITCH_COOLDOWN", Message: "mode switch cooldown active"}
	ErrInvalidMode    = &ModeError{Code: "INVALID_MODE", Message: "invalid mode specified"}
)

// ModeError represents a mode error
type ModeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *ModeError) Error() string {
	return e.Message
}

// CycleMode cycles through all modes
func (mm *ModeManager) CycleMode() Mode {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	modes := []Mode{ModeLive, ModeReplay, ModeSimulation, ModeHybrid}
	currentIdx := 0
	
	for i, m := range modes {
		if m == mm.config.CurrentMode {
			currentIdx = i
			break
		}
	}
	
	nextIdx := (currentIdx + 1) % len(modes)
	newMode := modes[nextIdx]
	
	oldMode := mm.config.CurrentMode
	mm.config.PreviousMode = oldMode
	mm.config.CurrentMode = newMode
	mm.lastSwitch = time.Now()
	
	for _, cb := range mm.callbacks {
		go cb(oldMode, newMode)
	}
	
	return newMode
}

// SetHybridRatio sets the hybrid ratio
func (mm *ModeManager) SetHybridRatio(ratio float64) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	
	mm.config.HybridRatio = ratio
}

// GetHybridRatio returns the hybrid ratio
func (mm *ModeManager) GetHybridRatio() float64 {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return mm.config.HybridRatio
}