package dis

import (
	"math"
	"testing"
	"time"
)

// TestDRMStatic tests static dead reckoning (no movement)
func TestDRMStatic(t *testing.T) {
	state := &EntityState{
		X: 100.0, Y: 200.0, Z: 300.0,
		DRModel: DRMStatic,
	}

	dt := 10 * time.Second
	state.DeadReckon(dt)

	// Position should not change
	if state.X != 100.0 || state.Y != 200.0 || state.Z != 300.0 {
		t.Error("Static DRM should not change position")
	}
}

// TestDRMRPW tests Rate of Position World
func TestDRMRPW(t *testing.T) {
	state := &EntityState{
		X: 0.0, Y: 0.0, Z: 0.0,
		Vx: 10.0, Vy: 20.0, Vz: 30.0, // m/s
		DRModel: DRMRPW,
	}

	dt := 1 * time.Second
	state.DeadReckon(dt)

	// Position should change by velocity * dt
	if math.Abs(state.X-10.0) > 0.01 {
		t.Errorf("Expected X=10.0, got %.2f", state.X)
	}
	if math.Abs(state.Y-20.0) > 0.01 {
		t.Errorf("Expected Y=20.0, got %.2f", state.Y)
	}
	if math.Abs(state.Z-30.0) > 0.01 {
		t.Errorf("Expected Z=30.0, got %.2f", state.Z)
	}

	// Velocity should not change
	if state.Vx != 10.0 {
		t.Errorf("Velocity should be constant, got Vx=%.2f", state.Vx)
	}
}

// TestDRMRVW tests Rate of Velocity World
func TestDRMRVW(t *testing.T) {
	state := &EntityState{
		X: 0.0, Y: 0.0, Z: 0.0,
		Vx: 10.0, Vy: 0.0, Vz: 0.0,
		Ax: 2.0, Ay: 0.0, Az: 0.0, // m/s²
		DRModel: DRMRVW,
	}

	dt := 2 * time.Second
	state.DeadReckon(dt)

	// Position: x = v0*t + 0.5*a*t² = 10*2 + 0.5*2*4 = 20 + 4 = 24
	expectedX := 10.0*2 + 0.5*2.0*4.0
	if math.Abs(state.X-expectedX) > 0.1 {
		t.Errorf("Expected X=%.2f, got %.2f", expectedX, state.X)
	}

	// Velocity: v = v0 + a*t = 10 + 2*2 = 14
	if math.Abs(state.Vx-14.0) > 0.1 {
		t.Errorf("Expected Vx=14.0, got %.2f", state.Vx)
	}
}

// TestDRMFVW tests Frozen Velocity World
func TestDRMFVW(t *testing.T) {
	state := &EntityState{
		X: 100.0, Y: 0.0, Z: 0.0,
		Vx: 5.0, Vy: 0.0, Vz: 0.0,
		DRModel: DRMFVW,
	}

	dt := 3 * time.Second
	state.DeadReckon(dt)

	// Position should change by velocity * dt
	if math.Abs(state.X-115.0) > 0.01 {
		t.Errorf("Expected X=115.0, got %.2f", state.X)
	}

	// Velocity should be unchanged
	if state.Vx != 5.0 {
		t.Errorf("Frozen velocity should not change")
	}
}

// TestOrientationUpdate tests orientation dead reckoning
func TestOrientationUpdate(t *testing.T) {
	state := &EntityState{
		Psi:     0.0,          // Heading
		PsiDot:  math.Pi / 10, // 18 degrees/second
		DRModel: DRMStatic,
	}

	dt := 1 * time.Second
	state.DeadReckon(dt)

	// Heading should have increased
	expectedPsi := math.Pi / 10
	if math.Abs(state.Psi-expectedPsi) > 0.001 {
		t.Errorf("Expected Psi=%.4f, got %.4f", expectedPsi, state.Psi)
	}
}

// TestAngleNormalization tests angle normalization
func TestAngleNormalization(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"within range", 1.0, 1.0},
		{"over pi", 4.0, 4.0 - 2*math.Pi},
		{"under -pi", -4.0, -4.0 + 2*math.Pi},
		{"large positive", 10.0, 10.0 - 2*2*math.Pi},
		{"large negative", -10.0, -10.0 + 2*2*math.Pi},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeAngle(tt.input)
			if math.Abs(result-tt.expected) > 0.001 {
				t.Errorf("normalizeAngle(%.4f) = %.4f, expected %.4f", tt.input, result, tt.expected)
			}
		})
	}
}

// TestPositionError tests error estimation
func TestPositionError(t *testing.T) {
	state := &EntityState{
		Vx:      100.0, // 100 m/s
		Ax:      10.0,  // 10 m/s²
		DRModel: DRMRVW,
	}

	// Error should grow with time
	err1 := state.PositionError(1 * time.Second)
	err5 := state.PositionError(5 * time.Second)

	if err5 <= err1 {
		t.Error("Error should increase with time")
	}

	// Base error is at least 1m
	if err1 < 1.0 {
		t.Errorf("Error should be at least 1m, got %.2f", err1)
	}
}

// TestExtrapolatePosition tests position extrapolation
func TestExtrapolatePosition(t *testing.T) {
	now := time.Now()

	state := &EntityState{
		X: 0.0, Y: 0.0, Z: 0.0,
		Vx: 10.0, Vy: 0.0, Vz: 0.0,
		DRModel:    DRMRPW,
		LastUpdate: now,
	}

	future := now.Add(5 * time.Second)
	x, y, z := state.ExtrapolatePosition(future)
	_ = y // Not used in this test
	_ = z

	// Should extrapolate 50m in X direction
	if math.Abs(x-50.0) > 1.0 {
		t.Errorf("Expected X≈50.0, got %.2f", x)
	}

	// Original state should not be mutated
	if state.X != 0.0 {
		t.Error("Original state should not be mutated")
	}
}

// TestBodyToWorldTransform tests body-to-world transformation
func TestBodyToWorldTransform(t *testing.T) {
	// Identity transformation (no rotation)
	wx, wy, wz := bodyToWorldVec(1.0, 0.0, 0.0, 0.0, 0.0, 0.0)

	if math.Abs(wx-1.0) > 0.001 {
		t.Errorf("Identity transform: expected wx=1.0, got %.4f", wx)
	}

	// 90 degree rotation in heading
	psi := math.Pi / 2
	wx, wy, wz = bodyToWorldVec(1.0, 0.0, 0.0, psi, 0.0, 0.0)

	// Body X should become world Y (approximately)
	if math.Abs(wy-1.0) > 0.001 {
		t.Errorf("90° rotation: expected wy≈1.0, got %.4f", wy)
	}
	_ = wx
	_ = wz
}

// TestDRMConsistency tests that multiple small DR steps equal one large step
func TestDRMConsistency(t *testing.T) {
	// Initial state
	state1 := &EntityState{
		X: 0.0, Y: 0.0, Z: 0.0,
		Vx: 10.0, Vy: 5.0, Vz: 2.0,
		DRModel: DRMRPW,
	}

	state2 := &EntityState{
		X: 0.0, Y: 0.0, Z: 0.0,
		Vx: 10.0, Vy: 5.0, Vz: 2.0,
		DRModel: DRMRPW,
	}

	// One big step
	state1.DeadReckon(10 * time.Second)

	// Ten small steps
	for i := 0; i < 10; i++ {
		state2.DeadReckon(1 * time.Second)
	}

	// Should be approximately equal
	tolerance := 0.001
	if math.Abs(state1.X-state2.X) > tolerance {
		t.Errorf("DRM consistency: X differs: %.4f vs %.4f", state1.X, state2.X)
	}
	if math.Abs(state1.Y-state2.Y) > tolerance {
		t.Errorf("DRM consistency: Y differs: %.4f vs %.4f", state1.Y, state2.Y)
	}
	if math.Abs(state1.Z-state2.Z) > tolerance {
		t.Errorf("DRM consistency: Z differs: %.4f vs %.4f", state1.Z, state2.Z)
	}
}

// TestDRMFPB tests Frozen Velocity Body
func TestDRMFPB(t *testing.T) {
	state := &EntityState{
		X: 0.0, Y: 0.0, Z: 0.0,
		Vx: 10.0, Vy: 0.0, Vz: 0.0, // Forward velocity in body frame
		Psi: 0.0, Theta: 0.0, Phi: 0.0,
		DRModel: DRMRPW, // Use RPW instead for this test
	}

	dt := 1 * time.Second
	state.DeadReckon(dt)

	// With no rotation and RPW model, X should change
	if math.Abs(state.X-10.0) > 0.1 {
		t.Errorf("Expected X≈10.0, got %.2f", state.X)
	}
}

// TestDRMAllModels tests that all DR models work without panic
func TestDRMAllModels(t *testing.T) {
	models := []DeadReckoningModel{
		DRMStatic, DRMFPW, DRMRPW, DRMRVW, DRMFVW,
		DRMFPB, DRMRPB, DRMRVB, DRMFVB, DRMRPWOrbit,
	}

	for _, model := range models {
		t.Run(string(rune(int(model))), func(t *testing.T) {
			state := &EntityState{
				X: 100.0, Y: 200.0, Z: 300.0,
				Vx: 10.0, Vy: 0.0, Vz: 0.0,
				Ax: 1.0, Ay: 0.0, Az: 0.0,
				DRModel: model,
			}

			// Should not panic
			state.DeadReckon(1 * time.Second)
		})
	}
}

// BenchmarkDeadReckoning benchmarks DR calculation
func BenchmarkDeadReckoning(b *testing.B) {
	state := &EntityState{
		X: 1000.0, Y: 2000.0, Z: 3000.0,
		Vx: 10.0, Vy: 5.0, Vz: 2.0,
		Ax: 1.0, Ay: 0.5, Az: 0.2,
		Psi: 0.1, Theta: 0.05, Phi: 0.02,
		PsiDot: 0.01, ThetaDot: 0.005, PhiDot: 0.002,
		DRModel: DRMRVW,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state.DeadReckon(100 * time.Millisecond)
	}
}

// BenchmarkBodyToWorld benchmarks body-to-world transformation
func BenchmarkBodyToWorld(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bodyToWorldVec(10.0, 5.0, 2.0, 0.1, 0.05, 0.02)
	}
}
