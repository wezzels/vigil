package mode

import (
	"math"
	"testing"
	"time"
)

// TestModeString tests mode string representation
func TestModeString(t *testing.T) {
	tests := []struct {
		mode Mode
		want string
	}{
		{ModeLive, "LIVE"},
		{ModeReplay, "REPLAY"},
		{ModeSimulation, "SIMULATION"},
		{ModeHybrid, "HYBRID"},
	}

	for _, tt := range tests {
		if got := tt.mode.String(); got != tt.want {
			t.Errorf("Mode(%d).String() = %s, want %s", tt.mode, got, tt.want)
		}
	}
}

// TestModeMarshalText tests mode marshaling
func TestModeMarshalText(t *testing.T) {
	m := ModeLive
	text, err := m.MarshalText()
	if err != nil {
		t.Errorf("MarshalText failed: %v", err)
	}
	if string(text) != "LIVE" {
		t.Errorf("MarshalText = %s, want LIVE", text)
	}
}

// TestModeUnmarshalText tests mode unmarshaling
func TestModeUnmarshalText(t *testing.T) {
	tests := []string{"LIVE", "REPLAY", "SIMULATION", "HYBRID", "live", "replay"}
	expected := []Mode{ModeLive, ModeReplay, ModeSimulation, ModeHybrid, ModeLive, ModeReplay}

	for i, tt := range tests {
		var m Mode
		err := m.UnmarshalText([]byte(tt))
		if err != nil {
			t.Errorf("UnmarshalText(%s) failed: %v", tt, err)
		}
		if m != expected[i] {
			t.Errorf("UnmarshalText(%s) = %v, want %v", tt, m, expected[i])
		}
	}
}

// TestModeConfig tests default mode configuration
func TestModeConfig(t *testing.T) {
	config := DefaultModeConfig()

	if config.CurrentMode != ModeLive {
		t.Errorf("Expected default mode LIVE, got %v", config.CurrentMode)
	}
	if config.SwitchCooldown != 5*time.Second {
		t.Errorf("Expected switch cooldown 5s, got %v", config.SwitchCooldown)
	}
	if config.HybridRatio != 0.5 {
		t.Errorf("Expected hybrid ratio 0.5, got %f", config.HybridRatio)
	}
}

// TestNewModeManager tests mode manager creation
func TestNewModeManager(t *testing.T) {
	mm := NewModeManager(nil)

	if mm == nil {
		t.Fatal("Mode manager should not be nil")
	}

	if mm.GetMode() != ModeLive {
		t.Error("Default mode should be LIVE")
	}
}

// TestSetMode tests mode switching
func TestSetMode(t *testing.T) {
	mm := NewModeManager(nil)

	err := mm.SetMode(ModeSimulation)
	if err != nil {
		t.Errorf("SetMode failed: %v", err)
	}

	if mm.GetMode() != ModeSimulation {
		t.Errorf("Mode should be SIMULATION, got %v", mm.GetMode())
	}
}

// TestSetModeSame tests setting same mode
func TestSetModeSame(t *testing.T) {
	mm := NewModeManager(nil)

	// Set to LIVE (already default)
	err := mm.SetMode(ModeLive)
	if err != nil {
		t.Errorf("SetMode to same mode should succeed: %v", err)
	}
}

// TestModeChangeCallback tests mode change callbacks
func TestModeChangeCallback(t *testing.T) {
	mm := NewModeManager(nil)

	callbackCalled := false
	var callbackOld, callbackNew Mode

	mm.RegisterCallback(func(old, new_ Mode) {
		callbackCalled = true
		callbackOld = old
		callbackNew = new_
	})

	// Give callback time to register
	time.Sleep(10 * time.Millisecond)

	mm.SetMode(ModeReplay)

	// Wait for callback
	time.Sleep(10 * time.Millisecond)

	if !callbackCalled {
		t.Error("Callback should have been called")
	}

	if callbackOld != ModeLive {
		t.Errorf("Callback old mode should be LIVE, got %v", callbackOld)
	}

	if callbackNew != ModeReplay {
		t.Errorf("Callback new mode should be REPLAY, got %v", callbackNew)
	}
}

// TestRestorePreviousMode tests restoring previous mode
func TestRestorePreviousMode(t *testing.T) {
	mm := NewModeManager(nil)

	mm.SetMode(ModeSimulation)
	mm.SetMode(ModeReplay)

	err := mm.RestorePreviousMode()
	if err != nil {
		t.Errorf("RestorePreviousMode failed: %v", err)
	}

	if mm.GetMode() != ModeSimulation {
		t.Errorf("Mode should be SIMULATION, got %v", mm.GetMode())
	}
}

// TestCycleMode tests mode cycling
func TestCycleMode(t *testing.T) {
	mm := NewModeManager(nil)

	// Cycle through all modes
	modes := []Mode{ModeLive, ModeReplay, ModeSimulation, ModeHybrid}

	for _, expected := range modes[1:] {
		newMode := mm.CycleMode()
		if newMode != expected {
			t.Errorf("CycleMode = %v, want %v", newMode, expected)
		}
	}

	// Should wrap around
	newMode := mm.CycleMode()
	if newMode != ModeLive {
		t.Errorf("CycleMode should wrap to LIVE, got %v", newMode)
	}
}

// TestHybridRatio tests hybrid ratio
func TestHybridRatio(t *testing.T) {
	mm := NewModeManager(nil)

	mm.SetHybridRatio(0.75)

	if mm.GetHybridRatio() != 0.75 {
		t.Errorf("Hybrid ratio should be 0.75, got %f", mm.GetHybridRatio())
	}
}

// TestHybridRatioBounds tests hybrid ratio bounds
func TestHybridRatioBounds(t *testing.T) {
	mm := NewModeManager(nil)

	// Below minimum
	mm.SetHybridRatio(-0.5)
	if mm.GetHybridRatio() != 0 {
		t.Errorf("Hybrid ratio should be clamped to 0, got %f", mm.GetHybridRatio())
	}

	// Above maximum
	mm.SetHybridRatio(1.5)
	if mm.GetHybridRatio() != 1 {
		t.Errorf("Hybrid ratio should be clamped to 1, got %f", mm.GetHybridRatio())
	}
}

// TestModeStats tests mode statistics
func TestModeStats(t *testing.T) {
	mm := NewModeManager(nil)

	stats := mm.Stats()

	if stats.CurrentMode != ModeLive {
		t.Errorf("Current mode should be LIVE, got %v", stats.CurrentMode)
	}

	if stats.CanSwitch != true {
		t.Error("Should be able to switch mode")
	}
}

// TestModeIsMethods tests mode check methods
func TestModeIsMethods(t *testing.T) {
	mm := NewModeManager(nil)

	if !mm.IsLive() {
		t.Error("Should be in live mode")
	}

	mm.SetMode(ModeReplay)
	if !mm.IsReplay() {
		t.Error("Should be in replay mode")
	}

	mm.SetMode(ModeSimulation)
	if !mm.IsSimulation() {
		t.Error("Should be in simulation mode")
	}

	mm.SetMode(ModeHybrid)
	if !mm.IsHybrid() {
		t.Error("Should be in hybrid mode")
	}
}

// TestReplayConfig tests default replay configuration
func TestReplayConfig(t *testing.T) {
	config := DefaultReplayConfig()

	if config.SpeedFactor != 1.0 {
		t.Errorf("Expected speed factor 1.0, got %f", config.SpeedFactor)
	}
	if config.BufferSize != 1000 {
		t.Errorf("Expected buffer size 1000, got %d", config.BufferSize)
	}
}

// TestNewReplayManager tests replay manager creation
func TestNewReplayManager(t *testing.T) {
	rm := NewReplayManager(nil)

	if rm == nil {
		t.Fatal("Replay manager should not be nil")
	}
}

// TestReplaySetSpeed tests replay speed setting
func TestReplaySetSpeed(t *testing.T) {
	rm := NewReplayManager(nil)

	rm.SetSpeed(2.0)
	if rm.GetSpeed() != 2.0 {
		t.Errorf("Speed should be 2.0, got %f", rm.GetSpeed())
	}
}

// TestReplaySpeedBounds tests replay speed bounds
func TestReplaySpeedBounds(t *testing.T) {
	rm := NewReplayManager(nil)

	// Below minimum
	rm.SetSpeed(0.05)
	if rm.GetSpeed() != 0.1 {
		t.Errorf("Speed should be clamped to 0.1, got %f", rm.GetSpeed())
	}

	// Above maximum
	rm.SetSpeed(200)
	if rm.GetSpeed() != 100 {
		t.Errorf("Speed should be clamped to 100, got %f", rm.GetSpeed())
	}
}

// TestReplayStats tests replay statistics
func TestReplayStats(t *testing.T) {
	rm := NewReplayManager(nil)

	stats := rm.GetStats()

	if stats.IsPlaying {
		t.Error("Should not be playing initially")
	}

	if stats.Speed != 1.0 {
		t.Errorf("Default speed should be 1.0, got %f", stats.Speed)
	}
}

// TestHybridConfig tests default hybrid configuration
func TestHybridConfig(t *testing.T) {
	config := DefaultHybridConfig()

	if config.LiveRatio != 0.5 {
		t.Errorf("Expected live ratio 0.5, got %f", config.LiveRatio)
	}
	if config.NumSimTargets != 5 {
		t.Errorf("Expected 5 sim targets, got %d", config.NumSimTargets)
	}
}

// TestNewHybridManager tests hybrid manager creation
func TestNewHybridManager(t *testing.T) {
	hm := NewHybridManager(nil)

	if hm == nil {
		t.Fatal("Hybrid manager should not be nil")
	}
}

// TestHybridSetLiveRatio tests hybrid live ratio setting
func TestHybridSetLiveRatio(t *testing.T) {
	hm := NewHybridManager(nil)

	hm.SetLiveRatio(0.8)

	if hm.GetLiveRatio() != 0.8 {
		t.Errorf("Live ratio should be 0.8, got %f", hm.GetLiveRatio())
	}
}

// TestHybridLiveRatioBounds tests hybrid live ratio bounds
func TestHybridLiveRatioBounds(t *testing.T) {
	hm := NewHybridManager(nil)

	// Below minimum
	hm.SetLiveRatio(-0.5)
	if hm.GetLiveRatio() != 0 {
		t.Errorf("Live ratio should be clamped to 0, got %f", hm.GetLiveRatio())
	}

	// Above maximum
	hm.SetLiveRatio(1.5)
	if hm.GetLiveRatio() != 1 {
		t.Errorf("Live ratio should be clamped to 1, got %f", hm.GetLiveRatio())
	}
}

// TestHybridStats tests hybrid statistics
func TestHybridStats(t *testing.T) {
	hm := NewHybridManager(nil)

	stats := hm.Stats()

	if stats.LiveRatio != 0.5 {
		t.Errorf("Live ratio should be 0.5, got %f", stats.LiveRatio)
	}

	if stats.NumSimTargets != 5 {
		t.Errorf("Num sim targets should be 5, got %d", stats.NumSimTargets)
	}
}

// TestHybridSimTargets tests simulated targets
func TestHybridSimTargets(t *testing.T) {
	config := DefaultHybridConfig()
	config.NumSimTargets = 3
	hm := NewHybridManager(config)

	targets := hm.GetSimTargets()

	if len(targets) != 3 {
		t.Errorf("Expected 3 targets, got %d", len(targets))
	}

	for i, t_ := range targets {
		if t_.ID != uint32(1000+i) {
			t.Errorf("Target ID should be %d, got %d", 1000+i, t_.ID)
		}
	}
}

// TestHybridAddRemoveTarget tests adding and removing targets
func TestHybridAddRemoveTarget(t *testing.T) {
	hm := NewHybridManager(nil)

	initialCount := len(hm.GetSimTargets())

	// Add target
	target := SimulatedTarget{
		ID:       9999,
		Position: [3]float64{0, 0, 0},
		Velocity: [3]float64{100, 0, 0},
	}
	hm.AddSimTarget(target)

	if len(hm.GetSimTargets()) != initialCount+1 {
		t.Error("Target should be added")
	}

	// Remove target
	removed := hm.RemoveSimTarget(9999)
	if !removed {
		t.Error("Target should be removed")
	}

	if len(hm.GetSimTargets()) != initialCount {
		t.Error("Target should be removed")
	}
}

// TestModeError tests mode error
func TestModeError(t *testing.T) {
	err := ErrSwitchCooldown

	if err.Code != "SWITCH_COOLDOWN" {
		t.Errorf("Error code should be SWITCH_COOLDOWN, got %s", err.Code)
	}

	if err.Error() == "" {
		t.Error("Error message should not be empty")
	}
}

// BenchmarkSetMode benchmarks mode switching
func BenchmarkSetMode(b *testing.B) {
	mm := NewModeManager(nil)
	modes := []Mode{ModeLive, ModeReplay, ModeSimulation, ModeHybrid}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mm.SetMode(modes[i%4])
	}
}

// BenchmarkCycleMode benchmarks mode cycling
func BenchmarkCycleMode(b *testing.B) {
	mm := NewModeManager(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mm.CycleMode()
	}
}

// BenchmarkHybridRatio benchmarks hybrid ratio setting
func BenchmarkHybridRatio(b *testing.B) {
	hm := NewHybridManager(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hm.SetLiveRatio(float64(i%100) / 100.0)
	}
}

// Helper
func init() {
	// Suppress unused variable warning
	_ = math.E
}
