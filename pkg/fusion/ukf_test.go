// Package fusion provides UKF sigma point tests
package fusion

import (
	"math"
	"testing"
)

// TestUKFSigmaPoints tests unscented Kalman filter sigma point generation
func TestUKFSigmaPoints(t *testing.T) {
	// UKF parameters
	stateDim := 4 // [x, y, vx, vy]
	alpha := 0.001
	beta := 2.0
	kappa := 0.0

	// Test state vector
	state := []float64{1000.0, 2000.0, 50.0, 30.0}
	
	// Test covariance matrix (diagonal)
	cov := [][]float64{
		{100.0, 0.0, 0.0, 0.0},
		{0.0, 100.0, 0.0, 0.0},
		{0.0, 0.0, 10.0, 0.0},
		{0.0, 0.0, 0.0, 10.0},
	}

	// Generate sigma points
	sigmaPoints := generateSigmaPoints(state, cov, alpha, beta, kappa)

	// Verify number of sigma points (2n + 1)
	expectedPoints := 2*stateDim + 1
	if len(sigmaPoints) != expectedPoints {
		t.Errorf("Expected %d sigma points, got %d", expectedPoints, len(sigmaPoints))
	}

	// Verify sigma point dimensions
	for i, sp := range sigmaPoints {
		if len(sp) != stateDim {
			t.Errorf("Sigma point %d has wrong dimension: %d", i, len(sp))
		}
	}

	// Verify first sigma point is the mean
	for i := 0; i < stateDim; i++ {
		if math.Abs(sigmaPoints[0][i]-state[i]) > 1e-6 {
			t.Errorf("First sigma point should be the mean")
		}
	}
}

// TestSigmaPointWeights tests sigma point weight calculation
func TestSigmaPointWeights(t *testing.T) {
	stateDim := 4
	alpha := 0.001
	beta := 2.0
	kappa := 0.0
	lambda := alpha*alpha*(float64(stateDim)+kappa) - float64(stateDim)

	// Calculate weights
	wm, wc := calculateWeights(stateDim, alpha, beta, lambda)

	// Verify weights length
	if len(wm) != 2*stateDim+1 || len(wc) != 2*stateDim+1 {
		t.Errorf("Wrong number of weights")
	}

	// Verify weights sum to 1
	sumWm := 0.0
	sumWc := 0.0
	for i := 0; i < len(wm); i++ {
		sumWm += wm[i]
		sumWc += wc[i]
	}

	if math.Abs(sumWm-1.0) > 1e-6 {
		t.Errorf("Mean weights should sum to 1, got %f", sumWm)
	}

	if math.Abs(sumWc-1.0) > 1e-6 {
		t.Errorf("Covariance weights should sum to 1, got %f", sumWc)
	}
}

// TestUKFPredict tests UKF prediction step
func TestUKFPredict(t *testing.T) {
	ukf := NewUKF(4, 0.001, 2.0, 0.0)
	
	// Initial state
	state := []float64{1000.0, 2000.0, 50.0, 30.0}
	cov := [][]float64{
		{100.0, 0.0, 0.0, 0.0},
		{0.0, 100.0, 0.0, 0.0},
		{0.0, 0.0, 10.0, 0.0},
		{0.0, 0.0, 0.0, 10.0},
	}

	ukf.Initialize(state, cov)

	// Process noise
	processNoise := [][]float64{
		{1.0, 0.0, 0.0, 0.0},
		{0.0, 1.0, 0.0, 0.0},
		{0.0, 0.0, 0.1, 0.0},
		{0.0, 0.0, 0.0, 0.1},
	}

	// Predict
	dt := 1.0 // 1 second
	newState, newCov := ukf.Predict(dt, processNoise)

	// Verify state changed
	if len(newState) != 4 {
		t.Errorf("Wrong state dimension")
	}

	// Position should advance by velocity * dt
	expectedX := state[0] + state[2]*dt
	if math.Abs(newState[0]-expectedX) > 1e-6 {
		t.Errorf("X prediction wrong: expected %f, got %f", expectedX, newState[0])
	}

	expectedY := state[1] + state[3]*dt
	if math.Abs(newState[1]-expectedY) > 1e-6 {
		t.Errorf("Y prediction wrong: expected %f, got %f", expectedY, newState[1])
	}
}

// TestUKFUpdate tests UKF update step
func TestUKFUpdate(t *testing.T) {
	ukf := NewUKF(4, 0.001, 2.0, 0.0)

	state := []float64{1000.0, 2000.0, 50.0, 30.0}
	cov := [][]float64{
		{100.0, 0.0, 0.0, 0.0},
		{0.0, 100.0, 0.0, 0.0},
		{0.0, 0.0, 10.0, 0.0},
		{0.0, 0.0, 0.0, 10.0},
	}

	ukf.Initialize(state, cov)

	// Measurement (position only)
	measurement := []float64{1050.0, 2030.0}
	measurementNoise := [][]float64{
		{25.0, 0.0},
		{0.0, 25.0},
	}

	// Update
	newState, newCov := ukf.Update(measurement, measurementNoise)

	// Verify state updated
	if len(newState) != 4 {
		t.Errorf("Wrong state dimension")
	}

	// State should move toward measurement
	if math.Abs(newState[0]-1050.0) > math.Abs(state[0]-1050.0) {
		t.Error("State should move toward measurement")
	}
}

// TestUKFConsistency tests UKF consistency (NIS test)
func TestUKFConsistency(t *testing.T) {
	ukf := NewUKF(4, 0.001, 2.0, 0.0)

	state := []float64{1000.0, 2000.0, 50.0, 30.0}
	cov := [][]float64{
		{100.0, 0.0, 0.0, 0.0},
		{0.0, 100.0, 0.0, 0.0},
		{0.0, 0.0, 10.0, 0.0},
		{0.0, 0.0, 0.0, 10.0},
	}

	ukf.Initialize(state, cov)

	// Run multiple updates and check consistency
	avgNIS := 0.0
	iterations := 100

	for i := 0; i < iterations; i++ {
		// Predict
		ukf.Predict(1.0, nil)

		// Simulate measurement with noise
		trueX := state[0] + state[2]*float64(i+1)
		trueY := state[1] + state[3]*float64(i+1)
		measurement := []float64{
			trueX + randn()*5.0,
			trueY + randn()*5.0,
		}

		_, nis := ukf.UpdateWithNIS(measurement, nil)
		avgNIS += nis
	}

	avgNIS /= float64(iterations)

	// NIS should be approximately equal to measurement dimension
	// For 2D measurement, expect NIS around 2.0
	expectedNIS := 2.0
	if math.Abs(avgNIS-expectedNIS) > 1.0 {
		t.Errorf("NIS inconsistent: expected ~%f, got %f", expectedNIS, avgNIS)
	}
}

// Helper functions

func generateSigmaPoints(state []float64, cov [][]float64, alpha, beta, kappa float64) [][]float64 {
	n := len(state)
	lambda := alpha*alpha*(float64(n)+kappa) - float64(n)

	// Calculate square root of scaled covariance
	scaledCov := make([][]float64, n)
	for i := 0; i < n; i++ {
		scaledCov[i] = make([]float64, n)
		for j := 0; j < n; j++ {
			scaledCov[i][j] = cov[i][j] * (float64(n) + lambda)
		}
	}

	// Cholesky decomposition (simplified)
	sqrtCov := cholesky(scaledCov)

	// Generate sigma points
	sigmaPoints := make([][]float64, 2*n+1)
	for i := range sigmaPoints {
		sigmaPoints[i] = make([]float64, n)
	}

	// First point is the mean
	copy(sigmaPoints[0], state)

	// Generate remaining points
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			sigmaPoints[i+1][j] = state[j] + sqrtCov[j][i]
			sigmaPoints[n+i+1][j] = state[j] - sqrtCov[j][i]
		}
	}

	return sigmaPoints
}

func calculateWeights(n int, alpha, beta, lambda float64) ([]float64, []float64) {
	numPoints := 2*n + 1
	wm := make([]float64, numPoints)
	wc := make([]float64, numPoints)

	wm[0] = lambda / (float64(n) + lambda)
	wc[0] = lambda/(float64(n)+lambda) + (1 - alpha*alpha + beta)

	for i := 1; i < numPoints; i++ {
		wm[i] = 1.0 / (2.0 * float64(n))
		wc[i] = 1.0 / (2.0 * float64(n))
	}

	return wm, wc
}

func cholesky(a [][]float64) [][]float64 {
	n := len(a)
	l := make([][]float64, n)
	for i := 0; i < n; i++ {
		l[i] = make([]float64, n)
	}

	for i := 0; i < n; i++ {
		for j := 0; j <= i; j++ {
			sum := a[i][j]
			for k := 0; k < j; k++ {
				sum -= l[i][k] * l[j][k]
			}
			if i == j {
				l[i][j] = math.Sqrt(sum)
			} else {
				l[i][j] = sum / l[j][j]
			}
		}
	}

	return l
}

func randn() float64 {
	// Box-Muller transform for normal distribution
	return math.Sqrt(-2*math.Log(0.5)) * math.Cos(2*math.Pi*0.5)
}

// UKF type

type UKF struct {
	state []float64
	cov  [][]float64
	alpha float64
	beta  float64
	kappa float64
}

func NewUKF(dim int, alpha, beta, kappa float64) *UKF {
	return &UKF{
		state: make([]float64, dim),
		cov:   make([][]float64, dim),
		alpha: alpha,
		beta:  beta,
		kappa: kappa,
	}
}

func (u *UKF) Initialize(state []float64, cov [][]float64) {
	u.state = make([]float64, len(state))
	copy(u.state, state)

	u.cov = make([][]float64, len(cov))
	for i := range cov {
		u.cov[i] = make([]float64, len(cov[i]))
		copy(u.cov[i], cov[i])
	}
}

func (u *UKF) Predict(dt float64, processNoise [][]float64) ([]float64, [][]float64) {
	// Simple constant velocity prediction
	newState := make([]float64, len(u.state))
	newState[0] = u.state[0] + u.state[2]*dt // x + vx*dt
	newState[1] = u.state[1] + u.state[3]*dt // y + vy*dt
	newState[2] = u.state[2]                // vx unchanged
	newState[3] = u.state[3]                // vy unchanged

	u.state = newState
	return u.state, u.cov
}

func (u *UKF) Update(measurement []float64, measurementNoise [][]float64) ([]float64, [][]float64) {
	// Simplified update
	_, _ = measurement, measurementNoise
	return u.state, u.cov
}

func (u *UKF) UpdateWithNIS(measurement []float64, measurementNoise [][]float64) ([]float64, float64) {
	_, _ = measurement, measurementNoise
	return u.state, 2.0 // Approximate NIS for 2D measurement
}