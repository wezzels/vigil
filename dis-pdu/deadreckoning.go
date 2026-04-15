// Package dis implements Dead Reckoning algorithms for entity state propagation
package dis

import (
	"math"
	"time"
)

// DeadReckoningModel defines the dead reckoning algorithm type
type DeadReckoningModel int

const (
	DRMStatic   DeadReckoningModel = 0 // No dead reckoning
	DRMFPW      DeadReckoningModel = 1 // World, Frozen Position (World)
	DRMRPW      DeadReckoningModel = 2 // World, Rate of Position (World)
	DRMRVW      DeadReckoningModel = 3 // World, Rate of Velocity (World)
	DRMFVW      DeadReckoningModel = 4 // World, Frozen Velocity (World)
	DRMFPB      DeadReckoningModel = 5 // Body, Frozen Position (Body)
	DRMRPB      DeadReckoningModel = 6 // Body, Rate of Position (Body)
	DRMRVB      DeadReckoningModel = 7 // Body, Rate of Velocity (Body)
	DRMFVB      DeadReckoningModel = 8 // Body, Frozen Velocity (Body)
	DRMRPWOrbit DeadReckoningModel = 9 // World, Orbit (RPW)
)

// EntityState represents entity state for dead reckoning
type EntityState struct {
	// Position (ECEF meters)
	X, Y, Z float64

	// Velocity (m/s)
	Vx, Vy, Vz float64

	// Acceleration (m/s²)
	Ax, Ay, Az float64

	// Orientation (radians)
	Psi, Theta, Phi float64 // Heading, Pitch, Roll

	// Angular velocity (rad/s)
	PsiDot, ThetaDot, PhiDot float64

	// Last update time
	LastUpdate time.Time

	// Dead reckoning model
	DRModel DeadReckoningModel

	// DR parameters from PDU
	DRLinearAccelX, DRLinearAccelY, DRLinearAccelZ float64
	DRAngularVelX, DRAngularVelY, DRAngularVelZ    float64
}

// DeadReckon propagates entity state forward in time
func (s *EntityState) DeadReckon(dt time.Duration) {
	seconds := dt.Seconds()

	switch s.DRModel {
	case DRMStatic:
		// No movement

	case DRMFPW:
		// Frozen position - no dead reckoning
		// Entity stays at last known position

	case DRMRPW:
		// Rate of Position World
		// Position += Velocity * dt
		s.X += s.Vx * seconds
		s.Y += s.Vy * seconds
		s.Z += s.Vz * seconds

	case DRMRVW:
		// Rate of Velocity World
		// Position += Velocity * dt + 0.5 * Acceleration * dt²
		s.X += s.Vx*seconds + 0.5*s.Ax*seconds*seconds
		s.Y += s.Vy*seconds + 0.5*s.Ay*seconds*seconds
		s.Z += s.Vz*seconds + 0.5*s.Az*seconds*seconds
		// Velocity += Acceleration * dt
		s.Vx += s.Ax * seconds
		s.Vy += s.Ay * seconds
		s.Vz += s.Az * seconds

	case DRMFVW:
		// Frozen Velocity World
		// Position += Velocity * dt (velocity unchanged)
		s.X += s.Vx * seconds
		s.Y += s.Vy * seconds
		s.Z += s.Vz * seconds

	case DRMFVB:
		// Frozen Velocity Body
		// Transform body velocity to world, apply
		s.deadReckonBodyFrozen(seconds)

	case DRMRVB:
		// Rate of Velocity Body
		// Transform body acceleration, apply
		s.deadReckonBodyRate(seconds)

	case DRMRPWOrbit:
		// Orbit model - simplified circular orbit
		s.deadReckonOrbit(seconds)
	}

	// Update orientation (all models)
	s.Psi += s.PsiDot * seconds
	s.Theta += s.ThetaDot * seconds
	s.Phi += s.PhiDot * seconds

	// Normalize angles
	s.Psi = normalizeAngle(s.Psi)
	s.Theta = normalizeAngle(s.Theta)
	s.Phi = normalizeAngle(s.Phi)
}

// deadReckonBodyFrozen applies frozen velocity in body frame
func (s *EntityState) deadReckonBodyFrozen(seconds float64) {
	// Transform body velocity to world frame
	wx, wy, wz := bodyToWorldVec(s.Vx, s.Vy, s.Vz, s.Psi, s.Theta, s.Phi)

	s.X += wx * seconds
	s.Y += wy * seconds
	s.Z += wz * seconds
}

// deadReckonBodyRate applies rate of velocity in body frame
func (s *EntityState) deadReckonBodyRate(seconds float64) {
	// Transform body velocity and acceleration to world
	wx, wy, wz := bodyToWorldVec(s.Vx, s.Vy, s.Vz, s.Psi, s.Theta, s.Phi)
	ax, ay, az := bodyToWorldVec(s.Ax, s.Ay, s.Az, s.Psi, s.Theta, s.Phi)

	// Apply
	s.X += wx*seconds + 0.5*ax*seconds*seconds
	s.Y += wy*seconds + 0.5*ay*seconds*seconds
	s.Z += wz*seconds + 0.5*az*seconds*seconds

	s.Vx += s.Ax * seconds
	s.Vy += s.Ay * seconds
	s.Vz += s.Az * seconds
}

// bodyToWorldVec transforms a vector from body to world frame
func bodyToWorldVec(vx, vy, vz, psi, theta, phi float64) (wx, wy, wz float64) {
	// Rotation matrices (ZYX convention)
	// R = Rz(psi) * Ry(theta) * Rx(phi)

	cPsi := math.Cos(psi)
	sPsi := math.Sin(psi)
	cTheta := math.Cos(theta)
	sTheta := math.Sin(theta)
	cPhi := math.Cos(phi)
	sPhi := math.Sin(phi)

	wx = vx*(cPsi*cTheta) + vy*(cPsi*sTheta*sPhi-sPsi*cPhi) + vz*(cPsi*sTheta*cPhi+sPsi*sPhi)
	wy = vx*(sPsi*cTheta) + vy*(sPsi*sTheta*sPhi+cPsi*cPhi) + vz*(sPsi*sTheta*cPhi-cPsi*sPhi)
	wz = vx*(-sTheta) + vy*(cTheta*sPhi) + vz*(cTheta*cPhi)

	return
}

// deadReckonOrbit applies orbital mechanics (simplified)
func (s *EntityState) deadReckonOrbit(seconds float64) {
	// Simplified: assume circular orbit, propagate position
	// Real implementation would use orbital elements
	// For now, just use constant velocity
	s.X += s.Vx * seconds
	s.Y += s.Vy * seconds
	s.Z += s.Vz * seconds
}

// normalizeAngle normalizes an angle to [-π, π]
func normalizeAngle(a float64) float64 {
	for a > math.Pi {
		a -= 2 * math.Pi
	}
	for a < -math.Pi {
		a += 2 * math.Pi
	}
	return a
}

// PositionError estimates position error based on time since last update
func (s *EntityState) PositionError(dt time.Duration) float64 {
	// Error grows with time since last update
	// Model: error = base_error + velocity_error * dt + acceleration_error * dt²
	seconds := dt.Seconds()

	// Base error (GPS/sensor noise, ~1m)
	baseError := 1.0

	// Velocity error (m/s)
	velError := math.Sqrt(s.Vx*s.Vx+s.Vy*s.Vy+s.Vz*s.Vz) * 0.01 // 1% velocity error

	// Acceleration error
	accelError := math.Sqrt(s.Ax*s.Ax+s.Ay*s.Ay+s.Az*s.Az) * 0.1 // 10% accel error

	return baseError + velError*seconds + accelError*seconds*seconds
}

// ExtrapolatePosition returns extrapolated position at future time
func (s *EntityState) ExtrapolatePosition(future time.Time) (x, y, z float64) {
	dt := future.Sub(s.LastUpdate)
	// Create copy to avoid mutating state
	copy := *s
	copy.DeadReckon(dt)
	return copy.X, copy.Y, copy.Z
}
