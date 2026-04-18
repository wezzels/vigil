// Package interceptor implements BMD interceptor physics and guidance
// Includes proportional navigation guidance, interceptor kinematics,
// and kill assessment for GBI, SM-3, THAAD, and Patriot interceptors
package interceptor

import (
	"math"
	"time"
)

// InterceptorType represents supported interceptor types
type InterceptorType int

const (
	GMD_GBI InterceptorType = iota // Ground-based Midcourse Defense (GBI)
	SM3_IIA                        // Standard Missile 3 (Block IIA)
	SM3_IB                         // Standard Missile 3 (Block IB)
	THAAD                          // Terminal High Altitude Area Defense
	PATRIOT_PAC3                   // Patriot PAC-3
)

// InterceptorConfig holds interceptor performance parameters
type InterceptorConfig struct {
	Type            InterceptorType
	Name            string
	MaxAltitude     float64 // meters
	MaxVelocity     float64 // m/s
	MaxAcceleration float64 // m/s² (g * g)
	MinRange        float64 // meters
	MaxRange        float64 // meters
	EngagementZone  string  // \"BOOST\", \"MIDCOURSE\", \"TERMINAL\"
	HitProbability  float64 // Single-shot probability (0-1)
}

// DefaultInterceptorConfigs returns default configs for all interceptor types
func DefaultInterceptorConfigs() map[InterceptorType]*InterceptorConfig {
	return map[InterceptorType]*InterceptorConfig{
		GMD_GBI: {
			Type:            GMD_GBI,
			Name:            "GBI (Ground-based Interceptor)",
			MaxAltitude:     2000e3, // 2000 km (exo-atmospheric)
			MaxVelocity:     10e3,    // 10 km/s
			MaxAcceleration: 10 * 9.81,
			MinRange:        1000e3,  // 1000 km
			MaxRange:        5500e3,  // 5500 km (ICBM class)
			EngagementZone:  "MIDCOURSE",
			HitProbability:  0.56, // GBI documented intercept probability
		},
		SM3_IIA: {
			Type:            SM3_IIA,
			Name:            "SM-3 IIA (Standard Missile 3)",
			MaxAltitude:     500e3,  // 500 km
			MaxVelocity:     4.5e3,  // 4.5 km/s
			MaxAcceleration: 10 * 9.81,
			MinRange:         100e3, // 100 km
			MaxRange:         2500e3,
			EngagementZone:  "MIDCOURSE",
			HitProbability:  0.68,
		},
		SM3_IB: {
			Type:            SM3_IB,
			Name:            "SM-3 IB",
			MaxAltitude:     300e3, // 300 km
			MaxVelocity:     3.5e3,
			MaxAcceleration: 10 * 9.81,
			MinRange:        100e3,
			MaxRange:        1200e3,
			EngagementZone:  "MIDCOURSE",
			HitProbability:  0.65,
		},
		THAAD: {
			Type:            THAAD,
			Name:            "THAAD (Terminal High Altitude Area Defense)",
			MaxAltitude:     150e3, // 150 km
			MaxVelocity:     2.8e3,
			MaxAcceleration: 15 * 9.81,
			MinRange:         35e3,  // 35 km
			MaxRange:         300e3, // 300 km
			EngagementZone:  "TERMINAL",
			HitProbability:  0.89, // THAAD documented intercept rate
		},
		PATRIOT_PAC3: {
			Type:            PATRIOT_PAC3,
			Name:            "Patriot PAC-3",
			MaxAltitude:     25e3,  // 25 km
			MaxVelocity:     1.7e3,
			MaxAcceleration: 20 * 9.81,
			MinRange:        3e3,   // 3 km
			MaxRange:        100e3, // 100 km
			EngagementZone:  "TERMINAL",
			HitProbability:  0.85,
		},
	}
}

// InterceptorState represents current interceptor state
type InterceptorState struct {
	Type InterceptorType

	// Position (ECEF, meters)
	Position [3]float64

	// Velocity (ECEF, m/s)
	Velocity [3]float64

	// Attitude ( Euler angles, radians)
	Yaw, Pitch, Roll float64

	// Navigation state
	Stage         string // \"LAUNCH\", \"MIDCOURSE\", \"TERMINAL\", \"intercept\"
	TimeSinceLaunch time.Duration
	RemainingFuel  float64 // 0-1

	// Target info (for guidance)
	TargetPosition [3]float64
	TargetVelocity [3]float64
}

// NewInterceptorState creates initial interceptor state
func NewInterceptorState(t InterceptorType, launchPos, launchVel [3]float64) *InterceptorState {
	return &InterceptorState{
		Type:            t,
		Position:        launchPos,
		Velocity:        launchVel,
		Stage:           "LAUNCH",
		TimeSinceLaunch: 0,
		RemainingFuel:   1.0,
	}
}

// ProportionalNavigation implements PNG guidance law
// Standard algorithm used in GBI, SM-3, THAAD
// Reference: \"Proportional Navigation\" Zarchan (2012)
type ProportionalNavigation struct {
	NAVGain float64 // Navigation constant (typically 3-5)
}

// DefaultPNG returns standard PNG configuration
func DefaultPNG() *ProportionalNavigation {
	return &ProportionalNavigation{NAVGain: 3.0}
}

// GuidanceCommand calculates interceptor guidance command
// Returns required acceleration vector in m/s²
func (png *ProportionalNavigation) GuidanceCommand(state *InterceptorState, targetPos, targetVel [3]float64, dt time.Duration) [3]float64 {
	// Line-of-sight (LOS) vector to target
	los := [3]float64{
		targetPos[0] - state.Position[0],
		targetPos[1] - state.Position[1],
		targetPos[2] - state.Position[2],
	}

	// Range to target
	rng := math.Sqrt(los[0]*los[0] + los[1]*los[1] + los[2]*los[2])

	if rng < 1.0 {
		return [3]float64{0, 0, 0} // Contact
	}

	// Unit LOS
	losUnit := [3]float64{los[0] / rng, los[1] / rng, los[2] / rng}

	// Closing velocity (positive = closing)
	relVel := [3]float64{
		state.Velocity[0] - targetVel[0],
		state.Velocity[1] - targetVel[1],
		state.Velocity[2] - targetVel[2],
	}
	Vc := -(relVel[0]*losUnit[0] + relVel[1]*losUnit[1] + relVel[2]*losUnit[2])

	if Vc <= 0 {
		// Opening - no intercept possible
		return [3]float64{0, 0, 0}
	}

	// LOS rate (angular velocity of LOS)
	// Approximation: use cross-product of relative velocity and LOS
	// d(LOS)/dt = (V_rel - (V_rel·LOS)LOS) / |LOS|
	losRate := [3]float64{
		(relVel[0] - relVel[0]*losUnit[0]*losUnit[0]) / rng,
		(relVel[1] - relVel[1]*losUnit[1]*losUnit[1]) / rng,
		(relVel[2] - relVel[2]*losUnit[2]*losUnit[2]) / rng,
	}
	losRateMag := math.Sqrt(losRate[0]*losRate[0] + losRate[1]*losRate[1] + losRate[2]*losRate[2])

	// PNG command: N * Vc * LOS_rate × LOS_unit
	// This produces acceleration perpendicular to LOS
	cmd := [3]float64{
		png.NAVGain * Vc * losRate[1]*losUnit[2] / (losRateMag + 1e-10),
		png.NAVGain * Vc * losRate[2]*losUnit[0] / (losRateMag + 1e-10),
		png.NAVGain * Vc * losRate[0]*losUnit[1] / (losRateMag + 1e-10),
	}

	// Clamp to max acceleration
	cmdMag := math.Sqrt(cmd[0]*cmd[0] + cmd[1]*cmd[1] + cmd[2]*cmd[2])
	if cmdMag > 10*9.81 { // 10g max
		scale := 10 * 9.81 / cmdMag
		cmd = [3]float64{cmd[0] * scale, cmd[1] * scale, cmd[2] * scale}
	}

	return cmd
}

// PureProportionalNavigation is simplified PNG for real-time use
// Returns commanded acceleration magnitude and direction
func (png *ProportionalNavigation) PureProportionalNavigation(
	pos [3]float64, vel [3]float64,
	targetPos, targetVel [3]float64,
	navGain float64,
) (accelMag float64, accelDir [3]float64) {

	// LOS vector
	los := [3]float64{
		targetPos[0] - pos[0],
		targetPos[1] - pos[1],
		targetPos[2] - pos[2],
	}
	rng := math.Sqrt(los[0]*los[0] + los[1]*los[1] + los[2]*los[2])

	if rng < 1.0 {
		return 0, [3]float64{0, 0, 0}
	}

	// Normalize LOS
	losU := [3]float64{los[0] / rng, los[1] / rng, los[2] / rng}

	// Closing velocity
	Vc := (vel[0]-targetVel[0])*losU[0] + (vel[1]-targetVel[1])*losU[1] + (vel[2]-targetVel[2])*losU[2]
	Vc = -Vc // Make positive

	if Vc <= 0 {
		return 0, [3]float64{0, 0, 0}
	}

	// Relative velocity perpendicular to LOS
	VrPerp := [3]float64{
		(vel[0] - targetVel[0]) - ((vel[0]-targetVel[0])*losU[0])*losU[0],
		(vel[1] - targetVel[1]) - ((vel[1]-targetVel[1])*losU[1])*losU[1],
		(vel[2] - targetVel[2]) - ((vel[2]-targetVel[2])*losU[2])*losU[2],
	}

	// Command: N * Vc * VrPerp / rng
	accelMag = navGain * Vc * math.Sqrt(VrPerp[0]*VrPerp[0]+VrPerp[1]*VrPerp[1]+VrPerp[2]*VrPerp[2]) / rng

	// Direction is perpendicular to LOS (cross product of LOS and velocity)
	// For simplicity, use the perpendicular velocity direction
	if math.Sqrt(VrPerp[0]*VrPerp[0]+VrPerp[1]*VrPerp[1]+VrPerp[2]*VrPerp[2]) > 0.01 {
		accelDir = [3]float64{
			VrPerp[0] / math.Sqrt(VrPerp[0]*VrPerp[0]+VrPerp[1]*VrPerp[1]+VrPerp[2]*VrPerp[2]),
			VrPerp[1] / math.Sqrt(VrPerp[0]*VrPerp[0]+VrPerp[1]*VrPerp[1]+VrPerp[2]*VrPerp[2]),
			VrPerp[2] / math.Sqrt(VrPerp[0]*VrPerp[0]+VrPerp[1]*VrPerp[1]+VrPerp[2]*VrPerp[2]),
		}
	} else {
		accelDir = [3]float64{0, 0, 0}
	}

	return accelMag, accelDir
}

// UpdateState updates interceptor state using guidance command
func (state *InterceptorState) UpdateState(cmd [3]float64, dt time.Duration) {
	// Update position
	state.Position[0] += state.Velocity[0] * dt.Seconds()
	state.Position[1] += state.Velocity[1] * dt.Seconds()
	state.Position[2] += state.Velocity[2] * dt.Seconds()

	// Update velocity from acceleration
	state.Velocity[0] += cmd[0] * dt.Seconds()
	state.Velocity[1] += cmd[1] * dt.Seconds()
	state.Velocity[2] += cmd[2] * dt.Seconds()

	// Update time
	state.TimeSinceLaunch += dt

	// Fuel consumption (simplified)
	state.RemainingFuel -= 0.001 * dt.Seconds()
	if state.RemainingFuel < 0 {
		state.RemainingFuel = 0
	}

	// Update stage based on altitude and time
	alt := math.Sqrt(state.Position[0]*state.Position[0]+
		state.Position[1]*state.Position[1]+
		state.Position[2]*state.Position[2]) - 6371e3 // Earth radius

	if alt > 100e3 {
		state.Stage = "MIDCOURSE"
	} else if alt > 30e3 {
		state.Stage = "TERMINAL"
	}
}

// TimeToIntercept estimates time to intercept at current closure rate
func TimeToIntercept(pos [3]float64, vel [3]float64, targetPos, targetVel [3]float64) float64 {
	// Relative position and velocity
	relPos := [3]float64{
		targetPos[0] - pos[0],
		targetPos[1] - pos[1],
		targetPos[2] - pos[2],
	}
	relVel := [3]float64{
		vel[0] - targetVel[0],
		vel[1] - targetVel[1],
		vel[2] - targetVel[2],
	}

	// Relative speed squared
	a := relVel[0]*relVel[0] + relVel[1]*relVel[1] + relVel[2]*relVel[2]
	if a < 1.0 {
		return 999999 // No closure
	}

	// Relative velocity dot relative position
	b := 2 * (relVel[0]*relPos[0] + relVel[1]*relPos[1] + relVel[2]*relPos[2])

	// Relative position squared
	c := relPos[0]*relPos[0] + relPos[1]*relPos[1] + relPos[2]*relPos[2]

	// Discriminant
	disc := b*b - 4*a*c
	if disc < 0 {
		return 999999 // No solution
	}

	// Time to intercept (positive root)
	t := (-b - math.Sqrt(disc)) / (2 * a)
	if t < 0 {
		t = (-b + math.Sqrt(disc)) / (2 * a)
	}

	return t
}

// InterceptProbability calculates probability of successful intercept
// Based on engagement geometry and interceptor capability
func InterceptProbability(
	interceptor *InterceptorConfig,
	interceptorPos, interceptorVel [3]float64,
	targetPos, targetVel [3]float64,
	tti float64,
) float64 {

	// Base probability from interceptor type
	baseProb := interceptor.HitProbability

	// Time pressure factor (less time = lower probability)
	timeFactor := 1.0
	if tti > 120 { // > 2 minutes is comfortable
		timeFactor = 1.0
	} else if tti > 60 { // 1-2 minutes
		timeFactor = 0.95
	} else if tti > 30 { // 30-60 seconds
		timeFactor = 0.85
	} else if tti > 10 { // 10-30 seconds (terminal)
		timeFactor = 0.75
	} else { // < 10 seconds
		timeFactor = 0.6
	}

	// Range factor
	rng := math.Sqrt(
		math.Pow(targetPos[0]-interceptorPos[0], 2)+
			math.Pow(targetPos[1]-interceptorPos[1], 2)+
			math.Pow(targetPos[2]-interceptorPos[2], 2),
	)
	rangeFactor := 1.0
	if rng < interceptor.MinRange {
		rangeFactor = 0.3 // Too close
	} else if rng > interceptor.MaxRange {
		rangeFactor = 0.2 // Too far
	} else if rng < interceptor.MinRange*2 {
		rangeFactor = 0.9
	} else if rng > interceptor.MaxRange*0.7 {
		rangeFactor = 0.85
	}

	// Closing velocity factor
	closingVel := -(interceptorVel[0]-targetVel[0])*interceptorPos[0]/rng -
		(interceptorVel[1]-targetVel[1])*interceptorPos[1]/rng -
		(interceptorVel[2]-targetVel[2])*interceptorPos[2]/rng
	closingFactor := math.Min(1.0, closingVel/1000) // Normalize to 1 km/s

	// Combined probability
	prob := baseProb * timeFactor * rangeFactor * (0.9 + 0.1*closingFactor)

	return math.Max(0, math.Min(1.0, prob))
}
