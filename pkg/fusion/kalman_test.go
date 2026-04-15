package fusion

import (
	"math"
	"testing"
)

// TestKalmanPredict tests the predict step
func TestKalmanPredict(t *testing.T) {
	kf := NewKalmanFilter()

	state := &KalmanState{
		X: [6]float64{38.0, -77.0, 100.0, 0.001, 0.001, 1.0}, // Position + velocity
		P: [6][6]float64{
			{0.0001, 0, 0, 0, 0, 0},
			{0, 0.0001, 0, 0, 0, 0},
			{0, 0, 10.0, 0, 0, 0},
			{0, 0, 0, 0.0001, 0, 0},
			{0, 0, 0, 0, 0.0001, 0},
			{0, 0, 0, 0, 0, 1.0},
		},
	}

	dt := 1.0 // 1 second

	// Predict forward
	lat0 := state.X[0]
	lon0 := state.X[1]
	alt0 := state.X[2]
	_ = lon0 // Avoid unused variable error
	_ = alt0

	kf.Predict(state, dt)

	// Position should have changed by velocity * dt
	expectedLat := lat0 + 0.001*dt
	expectedLon := lon0 + 0.001*dt
	expectedAlt := alt0 + 1.0*dt

	if math.Abs(state.X[0]-expectedLat) > 0.00001 {
		t.Errorf("Lat prediction wrong: expected %.6f, got %.6f", expectedLat, state.X[0])
	}
	if math.Abs(state.X[1]-expectedLon) > 0.00001 {
		t.Errorf("Lon prediction wrong: expected %.6f, got %.6f", expectedLon, state.X[1])
	}
	if math.Abs(state.X[2]-expectedAlt) > 0.1 {
		t.Errorf("Alt prediction wrong: expected %.1f, got %.1f", expectedAlt, state.X[2])
	}

	// Velocity should remain constant
	if state.X[3] != 0.001 {
		t.Errorf("Velocity should be constant, got %.6f", state.X[3])
	}

	// Covariance should increase
	if state.P[0][0] <= 0.0001 {
		t.Error("Covariance should increase after predict")
	}
}

// TestKalmanUpdate tests the update step
func TestKalmanUpdate(t *testing.T) {
	kf := NewKalmanFilter()

	state := &KalmanState{
		X: [6]float64{38.0, -77.0, 100.0, 0.0, 0.0, 0.0},
		P: [6][6]float64{
			{0.01, 0, 0, 0, 0, 0},
			{0, 0.01, 0, 0, 0, 0},
			{0, 0, 100.0, 0, 0, 0},
			{0, 0, 0, 0.001, 0, 0},
			{0, 0, 0, 0, 0.001, 0},
			{0, 0, 0, 0, 0, 1.0},
		},
	}

	// Measurement at slightly different position
	z := [3]float64{38.1, -77.1, 110.0}
	R := [3][3]float64{
		{0.001, 0, 0},
		{0, 0.001, 0},
		{0, 0, 10.0},
	}

	lat0 := state.X[0]
	lon0 := state.X[1]
	alt0 := state.X[2]
	_ = lon0 // Avoid unused error
	_ = alt0

	kf.Update(state, z, R)

	// State should have moved toward measurement
	if state.X[0] <= lat0 {
		t.Errorf("Lat should increase toward measurement")
	}
	if state.X[0] >= z[0] {
		t.Errorf("Lat should not pass measurement (Kalman gain < 1)")
	}

	// Covariance should decrease
	if state.P[0][0] >= 0.01 {
		t.Error("Covariance should decrease after update")
	}
}

// TestKalmanConvergence tests that filter converges to true state
func TestKalmanConvergence(t *testing.T) {
	kf := NewKalmanFilter()

	// True state
	trueLat := 38.0
	trueLon := -77.0
	trueAlt := 100.0

	// Initial estimate (wrong)
	state := &KalmanState{
		X: [6]float64{38.5, -77.5, 150.0, 0.0, 0.0, 0.0},
		P: [6][6]float64{
			{1.0, 0, 0, 0, 0, 0},
			{0, 1.0, 0, 0, 0, 0},
			{0, 0, 1000.0, 0, 0, 0},
			{0, 0, 0, 0.1, 0, 0},
			{0, 0, 0, 0, 0.1, 0},
			{0, 0, 0, 0, 0, 10.0},
		},
	}

	R := [3][3]float64{
		{0.0001, 0, 0},
		{0, 0.0001, 0},
		{0, 0, 1.0},
	}

	// Apply many measurements near true state
	for i := 0; i < 100; i++ {
		// Noisy measurement near true state
		offset := 0.01 * (float64(i%10) - 4.5) / 10.0 // Small noise
		z := [3]float64{
			trueLat + offset,
			trueLon + offset,
			trueAlt + offset*10.0,
		}
		kf.Update(state, z, R)
	}

	// Should converge to near true state
	tolerance := 0.1
	if math.Abs(state.X[0]-trueLat) > tolerance {
		t.Errorf("Lat did not converge: expected %.1f ± %.1f, got %.4f", trueLat, tolerance, state.X[0])
	}
	if math.Abs(state.X[1]-trueLon) > tolerance {
		t.Errorf("Lon did not converge: expected %.1f ± %.1f, got %.4f", trueLon, tolerance, state.X[1])
	}
	if math.Abs(state.X[2]-trueAlt) > 20.0 {
		t.Errorf("Alt did not converge: expected %.0f ± 20, got %.1f", trueAlt, state.X[2])
	}
}

// TestInverse3x3 tests matrix inversion
func TestInverse3x3(t *testing.T) {
	tests := []struct {
		name string
		m    [3][3]float64
	}{
		{"identity", [3][3]float64{
			{1, 0, 0},
			{0, 1, 0},
			{0, 0, 1},
		}},
		{"diagonal", [3][3]float64{
			{2, 0, 0},
			{0, 3, 0},
			{0, 0, 4},
		}},
		{"full", [3][3]float64{
			{1, 2, 3},
			{4, 5, 6},
			{7, 8, 10},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := inverse3x3(tt.m)

			// Verify: m * inv = I
			for i := 0; i < 3; i++ {
				for j := 0; j < 3; j++ {
					sum := 0.0
					for k := 0; k < 3; k++ {
						sum += tt.m[i][k] * inv[k][j]
					}
					expected := 0.0
					if i == j {
						expected = 1.0
					}
					if math.Abs(sum-expected) > 0.001 {
						t.Errorf("m * inv[%d][%d] = %.6f, expected %.1f", i, j, sum, expected)
					}
				}
			}
		})
	}
}

// TestProcessNoise tests that process noise increases uncertainty
func TestProcessNoise(t *testing.T) {
	kf := NewKalmanFilter()

	state := &KalmanState{
		X: [6]float64{0, 0, 0, 0, 0, 0},
		P: [6][6]float64{}, // Zero covariance initially
	}

	p00 := state.P[0][0]

	kf.Predict(state, 1.0)

	// Process noise should have been added
	if state.P[0][0] <= p00 {
		t.Error("Process noise should increase covariance")
	}
}

// TestMeasurementNoise tests that measurement noise affects update
func TestMeasurementNoise(t *testing.T) {
	kf := NewKalmanFilter()

	state := &KalmanState{
		X: [6]float64{0, 0, 0, 0, 0, 0},
		P: [6][6]float64{
			{1, 0, 0, 0, 0, 0},
			{0, 1, 0, 0, 0, 0},
			{0, 0, 1, 0, 0, 0},
			{0, 0, 0, 1, 0, 0},
			{0, 0, 0, 0, 1, 0},
			{0, 0, 0, 0, 0, 1},
		},
	}

	// High measurement noise
	Rhigh := [3][3]float64{{100, 0, 0}, {0, 100, 0}, {0, 0, 100}}

	// Low measurement noise
	Rlow := [3][3]float64{{0.01, 0, 0}, {0, 0.01, 0}, {0, 0, 0.01}}

	// Same measurement
	z := [3]float64{1.0, 0, 0}

	state1 := *state
	state2 := *state

	kf.Update(&state1, z, Rhigh)
	kf.Update(&state2, z, Rlow)

	// With higher measurement noise, we should trust our prior more
	// So state1 should move less than state2
	if math.Abs(state1.X[0]) >= math.Abs(state2.X[0]) {
		t.Error("Higher noise should result in less update")
	}
}

// TestExtendedKalmanFilter tests EKF creation
func TestExtendedKalmanFilter(t *testing.T) {
	ekf := NewExtendedKalmanFilter()

	if ekf.KalmanFilter == nil {
		t.Error("EKF should contain KalmanFilter")
	}
}

// TestUnscentedKalmanFilter tests UKF creation
func TestUnscentedKalmanFilter(t *testing.T) {
	ukf := NewUnscentedKalmanFilter()

	if ukf.Wm == nil || len(ukf.Wm) != 13 {
		t.Errorf("Expected 13 mean weights, got %d", len(ukf.Wm))
	}

	if ukf.Wc == nil || len(ukf.Wc) != 13 {
		t.Errorf("Expected 13 covariance weights, got %d", len(ukf.Wc))
	}
}

// TestSigmaPointsGeneration tests sigma point generation
func TestSigmaPointsGeneration(t *testing.T) {
	ukf := NewUnscentedKalmanFilter()

	state := &KalmanState{
		X: [6]float64{38.0, -77.0, 100.0, 0.001, 0.001, 1.0},
		P: [6][6]float64{
			{0.0001, 0, 0, 0, 0, 0},
			{0, 0.0001, 0, 0, 0, 0},
			{0, 0, 10.0, 0, 0, 0},
			{0, 0, 0, 0.0001, 0, 0},
			{0, 0, 0, 0, 0.0001, 0},
			{0, 0, 0, 0, 0, 1.0},
		},
	}

	sigmaPoints := ukf.GenerateSigmaPoints(state)

	// Should have 13 sigma points for 6D state
	if len(sigmaPoints) != 13 {
		t.Errorf("Expected 13 sigma points, got %d", len(sigmaPoints))
	}

	// First sigma point should be the mean
	for i := 0; i < 6; i++ {
		if sigmaPoints[0][i] != state.X[i] {
			t.Errorf("First sigma point should be mean")
		}
	}

	// Other sigma points should be spread around mean
	for i := 1; i < 13; i++ {
		spread := false
		for j := 0; j < 6; j++ {
			if sigmaPoints[i][j] != state.X[j] {
				spread = true
				break
			}
		}
		if !spread {
			t.Errorf("Sigma point %d should be spread from mean", i)
		}
	}
}

// BenchmarkKalmanPredict benchmarks prediction
func BenchmarkKalmanPredict(b *testing.B) {
	kf := NewKalmanFilter()
	state := &KalmanState{
		X: [6]float64{38.0, -77.0, 100.0, 0.001, 0.001, 1.0},
		P: [6][6]float64{
			{0.0001, 0, 0, 0, 0, 0},
			{0, 0.0001, 0, 0, 0, 0},
			{0, 0, 10.0, 0, 0, 0},
			{0, 0, 0, 0.0001, 0, 0},
			{0, 0, 0, 0, 0.0001, 0},
			{0, 0, 0, 0, 0, 1.0},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		kf.Predict(state, 0.1)
	}
}

// BenchmarkKalmanUpdate benchmarks update
func BenchmarkKalmanUpdate(b *testing.B) {
	kf := NewKalmanFilter()
	state := &KalmanState{
		X: [6]float64{38.0, -77.0, 100.0, 0.001, 0.001, 1.0},
		P: [6][6]float64{
			{0.0001, 0, 0, 0, 0, 0},
			{0, 0.0001, 0, 0, 0, 0},
			{0, 0, 10.0, 0, 0, 0},
			{0, 0, 0, 0.0001, 0, 0},
			{0, 0, 0, 0, 0.0001, 0},
			{0, 0, 0, 0, 0, 1.0},
		},
	}

	z := [3]float64{38.001, -77.001, 101.0}
	R := [3][3]float64{
		{0.0001, 0, 0},
		{0, 0.0001, 0},
		{0, 0, 1.0},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		kf.Update(state, z, R)
	}
}
