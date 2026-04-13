// Package fusion implements Kalman filtering for track estimation
package fusion

import (
	"math"
)

// KalmanState represents the state vector and covariance
// State: [lat, lon, alt, vLat, vLon, vAlt]
type KalmanState struct {
	X     [6]float64 // State vector
	P     [6][6]float64 // Covariance matrix
	Time  int64      // Timestamp
}

// KalmanFilter implements a 6-state EKF for track estimation
type KalmanFilter struct {
	// Process noise (Q)
	Q [6][6]float64
	
	// Measurement noise (R) - will be set from measurement
	// State transition matrix (F) - computed dynamically
	// Measurement matrix (H) - identity for direct measurement
}

// NewKalmanFilter creates a new Kalman filter with default noise parameters
func NewKalmanFilter() *KalmanFilter {
	kf := &KalmanFilter{}
	
	// Process noise (how much we expect state to change)
	// Position process noise
	kf.Q[0][0] = 0.0001 * 0.0001 // lat variance
	kf.Q[1][1] = 0.0001 * 0.0001 // lon variance
	kf.Q[2][2] = 10.0 * 10.0    // alt variance
	// Velocity process noise
	kf.Q[3][3] = 0.001 * 0.001 // vLat variance
	kf.Q[4][4] = 0.001 * 0.001 // vLon variance
	kf.Q[5][5] = 1.0 * 1.0    // vAlt variance
	
	return kf
}

// Predict propagates state forward using constant velocity model
func (kf *KalmanFilter) Predict(state *KalmanState, dt float64) {
	// State transition: x' = F * x
	// F = [1 0 0 dt 0  0 ]
	//     [0 1 0 0  dt 0 ]
	//     [0 0 1 0  0  dt]
	//     [0 0 0 1  0  0 ]
	//     [0 0 0 0  1  0 ]
	//     [0 0 0 0  0  1 ]
	
	// Predict state
	state.X[0] += state.X[3] * dt // lat += vLat * dt
	state.X[1] += state.X[4] * dt // lon += vLon * dt
	state.X[2] += state.X[5] * dt // alt += vAlt * dt
	
	// Predict covariance: P' = F * P * F' + Q
	// For constant velocity model, we can compute this efficiently
	
	// Add process noise
	for i := 0; i < 6; i++ {
		for j := 0; j < 6; j++ {
			state.P[i][j] += kf.Q[i][j]
		}
	}
	
	// Propagate uncertainty through F
	// P[i][j] += dt * (P[i][j+3] + P[i+3][j]) for i,j < 3
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			state.P[i][j] += dt * (state.P[i][j+3] + state.P[i+3][j])
			state.P[i][j] += dt * dt * state.P[i+3][j+3]
		}
	}
	
	// Cross terms
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			state.P[i][j+3] += dt * state.P[i+3][j+3]
			state.P[j+3][i] = state.P[i][j+3] // Symmetric
		}
	}
}

// Update incorporates a measurement using standard Kalman update
func (kf *KalmanFilter) Update(state *KalmanState, z [3]float64, R [3][3]float64) {
	// Measurement: z = [lat, lon, alt]
	// H = [1 0 0 0 0 0]
	//     [0 1 0 0 0 0]
	//     [0 0 1 0 0 0]
	
	// Innovation: y = z - H*x
	y := [3]float64{
		z[0] - state.X[0],
		z[1] - state.X[1],
		z[2] - state.X[2],
	}
	
	// Innovation covariance: S = H * P * H' + R
	// Since H is identity for first 3 elements, S = P[0:3][0:3] + R
	S := [3][3]float64{}
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			S[i][j] = state.P[i][j] + R[i][j]
		}
	}
	
	// Kalman gain: K = P * H' * S^-1
	// K = P[:, 0:3] * S^-1
	Sinv := inverse3x3(S)
	K := [6][3]float64{}
	for i := 0; i < 6; i++ {
		for j := 0; j < 3; j++ {
			for k := 0; k < 3; k++ {
				K[i][j] += state.P[i][k] * Sinv[k][j]
			}
		}
	}
	
	// Update state: x = x + K * y
	for i := 0; i < 6; i++ {
		for j := 0; j < 3; j++ {
			state.X[i] += K[i][j] * y[j]
		}
	}
	
	// Update covariance: P = (I - K*H) * P
	// P = P - K * S * K'
	Pnew := [6][6]float64{}
	for i := 0; i < 6; i++ {
		for j := 0; j < 6; j++ {
			for k := 0; k < 3; k++ {
				Pnew[i][j] -= K[i][k] * S[k][k] * K[j][k]
			}
			Pnew[i][j] += state.P[i][j]
		}
	}
	state.P = Pnew
}

// inverse3x3 computes the inverse of a 3x3 matrix
func inverse3x3(m [3][3]float64) [3][3]float64 {
	// Using cofactor expansion
	det := m[0][0]*(m[1][1]*m[2][2]-m[1][2]*m[2][1]) -
		m[0][1]*(m[1][0]*m[2][2]-m[1][2]*m[2][0]) +
		m[0][2]*(m[1][0]*m[2][1]-m[1][1]*m[2][0])
	
	if math.Abs(det) < 1e-15 {
		// Singular matrix, return identity
		return [3][3]float64{
			{1, 0, 0},
			{0, 1, 0},
			{0, 0, 1},
		}
	}
	
	invDet := 1.0 / det
	
	var inv [3][3]float64
	inv[0][0] = (m[1][1]*m[2][2] - m[1][2]*m[2][1]) * invDet
	inv[0][1] = (m[0][2]*m[2][1] - m[0][1]*m[2][2]) * invDet
	inv[0][2] = (m[0][1]*m[1][2] - m[0][2]*m[1][1]) * invDet
	inv[1][0] = (m[1][2]*m[2][0] - m[1][0]*m[2][2]) * invDet
	inv[1][1] = (m[0][0]*m[2][2] - m[0][2]*m[2][0]) * invDet
	inv[1][2] = (m[0][2]*m[1][0] - m[0][0]*m[1][2]) * invDet
	inv[2][0] = (m[1][0]*m[2][1] - m[1][1]*m[2][0]) * invDet
	inv[2][1] = (m[0][1]*m[2][0] - m[0][0]*m[2][1]) * invDet
	inv[2][2] = (m[0][0]*m[1][1] - m[0][1]*m[1][0]) * invDet
	
	return inv
}

// ExtendedKalmanFilter extends Kalman filter for nonlinear measurements
type ExtendedKalmanFilter struct {
	*KalmanFilter
}

// NewExtendedKalmanFilter creates an EKF
func NewExtendedKalmanFilter() *ExtendedKalmanFilter {
	return &ExtendedKalmanFilter{
		KalmanFilter: NewKalmanFilter(),
	}
}

// PredictECEF predicts state in ECEF coordinates
func (ekf *ExtendedKalmanFilter) PredictECEF(state *KalmanState, dt float64) {
	// In ECEF, we need to account for Earth rotation
	// For simplicity, use the same constant velocity model
	ekf.KalmanFilter.Predict(state, dt)
}

// UpdateECEF updates with ECEF measurements (nonlinear due to coordinate transform)
func (ekf *ExtendedKalmanFilter) UpdateECEF(state *KalmanState, x, y, z float64, R [3][3]float64) {
	// Convert ECEF to geodetic for measurement update
	// Import from dis-pdu package would cause import cycle
	// Inline implementation
	lat, lon, alt := ecefToGeodeticInline(x, y, z)
	zMeas := [3]float64{lat, lon, alt}
	ekf.KalmanFilter.Update(state, zMeas, R)
}

// Inline ECEF to geodetic for Kalman package
func ecefToGeodeticInline(x, y, z float64) (lat, lon, alt float64) {
	const (
		a        = 6378137.0
		f        = 1.0 / 298.257223563
		e2       = 2*f - f*f
		b        = a * (1.0 - f)
		radToDeg = 180.0 / math.Pi
	)
	
	lon = math.Atan2(y, x) * radToDeg
	p := math.Sqrt(x*x + y*y)
	theta := math.Atan2(z*a, p*b)
	lat = math.Atan2(z+e2*b*math.Pow(math.Sin(theta), 3), p-e2*a*math.Pow(math.Cos(theta), 3)) * radToDeg
	sinLat := math.Sin(lat * math.Pi / 180.0)
	cosLat := math.Cos(lat * math.Pi / 180.0)
	N := a / math.Sqrt(1.0-e2*sinLat*sinLat)
	alt = p/cosLat - N
	return
}

// UnscentedKalmanFilter implements UKF using sigma points
type UnscentedKalmanFilter struct {
	*ExtendedKalmanFilter
	
	// UKF parameters
	Alpha float64 // Spread of sigma points
	Beta  float64 // Prior knowledge (2 for Gaussian)
	Kappa float64 // Secondary scaling parameter
	
	// Weights
	Wm []float64 // Mean weights
	Wc []float64 // Covariance weights
}

// NewUnscentedKalmanFilter creates a UKF
func NewUnscentedKalmanFilter() *UnscentedKalmanFilter {
	ukf := &UnscentedKalmanFilter{
		ExtendedKalmanFilter: NewExtendedKalmanFilter(),
		Alpha: 0.001,
		Beta:  2.0,
		Kappa: 0.0,
	}
	
	// Compute weights
	n := 6 // State dimension
	lambda := ukf.Alpha*ukf.Alpha*(float64(n)+ukf.Kappa) - float64(n)
	
	ukf.Wm = make([]float64, 2*n+1)
	ukf.Wc = make([]float64, 2*n+1)
	
	ukf.Wm[0] = lambda / (float64(n) + lambda)
	ukf.Wc[0] = ukf.Wm[0] + (1 - ukf.Alpha*ukf.Alpha + ukf.Beta)
	
	for i := 1; i < 2*n+1; i++ {
		ukf.Wm[i] = 1.0 / (2.0 * (float64(n) + lambda))
		ukf.Wc[i] = ukf.Wm[i]
	}
	
	return ukf
}

// GenerateSigmaPoints creates sigma points from state and covariance
func (ukf *UnscentedKalmanFilter) GenerateSigmaPoints(state *KalmanState) [13][6]float64 {
	n := 6
	lambda := ukf.Alpha*ukf.Alpha*(float64(n)+ukf.Kappa) - float64(n)
	
	// Compute sqrt of scaled covariance
	// For simplicity, use diagonal approximation
	sqrtP := [6][6]float64{}
	for i := 0; i < 6; i++ {
		sqrtP[i][i] = math.Sqrt((float64(n) + lambda) * state.P[i][i])
	}
	
	// Generate sigma points
	var sigmaPoints [13][6]float64
	
	// First sigma point is the mean
	sigmaPoints[0] = state.X
	
	// Sigma points from +sqrt(P)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			sigmaPoints[i+1][j] = state.X[j] + sqrtP[j][i]
		}
	}
	
	// Sigma points from -sqrt(P)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			sigmaPoints[i+n+1][j] = state.X[j] - sqrtP[j][i]
		}
	}
	
	return sigmaPoints
}

// PredictUKF performs UKF prediction step
func (ukf *UnscentedKalmanFilter) PredictUKF(state *KalmanState, dt float64) {
	// For UKF, we'd transform sigma points through the process model
	// For constant velocity, this is linear so we can use regular predict
	ukf.KalmanFilter.Predict(state, dt)
}