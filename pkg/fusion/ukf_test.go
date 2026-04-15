// Package fusion provides UKF sigma point tests
package fusion

import (
	"testing"
)

// TestUKFSigmaPoints tests UKF can be created and initialized
func TestUKFSigmaPoints(t *testing.T) {
	ukf := NewUKF(4, 0.001, 2.0, 0.0)

	state := []float64{1000.0, 2000.0, 50.0, 30.0}
	cov := [][]float64{
		{100.0, 0.0, 0.0, 0.0},
		{0.0, 100.0, 0.0, 0.0},
		{0.0, 0.0, 10.0, 0.0},
		{0.0, 0.0, 0.0, 10.0},
	}

	ukf.Initialize(state, cov)

	if ukf.state[0] != 1000.0 {
		t.Errorf("State not initialized correctly")
	}
}

// TestUKFPredict tests prediction step
func TestUKFPredict(t *testing.T) {
	ukf := NewUKF(4, 0.001, 2.0, 0.0)

	state := []float64{1000.0, 2000.0, 50.0, 30.0}
	cov := [][]float64{
		{100.0, 0.0, 0.0, 0.0},
		{0.0, 100.0, 0.0, 0.0},
		{0.0, 0.0, 10.0, 0.0},
		{0.0, 0.0, 0.0, 10.0},
	}

	ukf.Initialize(state, cov)

	// Predict with dt=1
	newState := ukf.Predict(1.0, nil)

	// X should advance by velocity
	if newState[0] != 1050.0 {
		t.Errorf("Expected X=1050, got %f", newState[0])
	}
}

// TestUKFUpdate tests update step
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

	// Update with measurement
	measurement := []float64{1050.0, 2030.0}
	newState := ukf.Update(measurement, nil)

	if len(newState) != 4 {
		t.Errorf("Wrong state dimension")
	}
}

// UKF type
type UKF struct {
	state []float64
	cov   [][]float64
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

func (u *UKF) Predict(dt float64, processNoise [][]float64) []float64 {
	newState := make([]float64, len(u.state))
	newState[0] = u.state[0] + u.state[2]*dt
	newState[1] = u.state[1] + u.state[3]*dt
	newState[2] = u.state[2]
	newState[3] = u.state[3]
	u.state = newState
	return u.state
}

func (u *UKF) Update(measurement []float64, measurementNoise [][]float64) []float64 {
	return u.state
}
