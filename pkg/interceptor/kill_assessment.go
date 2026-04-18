// Package interceptor provides BMD interceptor guidance, kinematics, and kill assessment
package interceptor

import (
	"math"
	"time"
)

// KillAssessment evaluates whether an intercept was successful
// References:
// - Thiessen, "Kill Assessment Methodology"
// - JIAMO ("Joint Integrated Air and Missile Defense")
type KillAssessment struct{}

// KillLevel represents post-intercept assessment result
type KillLevel int

const (
	KILL_LEVEL_NONE KillLevel = iota // No assessment yet
	KILL_LEVEL_FAW                   // Fall-away warhead (debris, no warhead)
	KILL_LEVEL_PKW                   // Propellant kill (warhead destroyed but not neutralized)
	KILL_LEVEL_NTK                   // Neutralization kill (warhead disabled)
	KILL_LEVEL_CATK                  // Cockpit/area kill (complete destruction)
)

// Assessment input parameters
type InterceptEvent struct {
	Time              time.Time
	InterceptPosition [3]float64 // ECEF, meters
	TargetPosition    [3]float64
	TargetVelocity    [3]float64
	MissDistance      float64 // meters
	InterceptorType   InterceptorType
	TargetType        ThreatType
}

// ThreatType classifies threat for assessment
type ThreatType int

const (
	THREAT_ICBM ThreatType = iota
	THREAT_IRBM
	THREAT_MRBM
	THREAT_SRBM
	THREAT_CRUISE
	THREAT_TBM // Tactical Ballistic Missile (anti-ballistic)
)

// DefaultKillAssessment returns configured kill assessor
func DefaultKillAssessment() *KillAssessment {
	return &KillAssessment{}
}

// Assess performs kill assessment based on intercept geometry
// Returns KillLevel and confidence (0-1)
func (ka *KillAssessment) Assess(event *InterceptEvent) (KillLevel, float64) {
	// Calculate key parameters
	rng := ka.distance(event.InterceptPosition, event.TargetPosition)

	// Blast radius based on threat type (NATO kill mechanism)
	blastRadius := ka.blastRadius(event.TargetType)

	// Conditioned kill probability based on miss distance
	Pk := ka.conditionedKillProbability(rng, blastRadius)

	// Determine kill level based on probability
	var level KillLevel
	var confidence float64

	if Pk >= 0.95 {
		level = KILL_LEVEL_CATK
		confidence = Pk
	} else if Pk >= 0.75 {
		level = KILL_LEVEL_NTK
		confidence = Pk
	} else if Pk >= 0.50 {
		level = KILL_LEVEL_PKW
		confidence = Pk
	} else if Pk >= 0.20 {
		level = KILL_LEVEL_FAW
		confidence = Pk
	} else {
		level = KILL_LEVEL_NONE
		confidence = 0.1
	}

	return level, confidence
}

// distance calculates 3D Euclidean distance
func (ka *KillAssessment) distance(p1, p2 [3]float64) float64 {
	return math.Sqrt(
		(p2[0]-p1[0])*(p2[0]-p1[0]) +
			(p2[1]-p1[1])*(p2[1]-p1[1]) +
			(p2[2]-p1[2])*(p2[2]-p1[2]),
	)
}

// blastRadius returns lethal radius for threat type (meters)
// Based on typical re-entry vehicle and explosive fill
func (ka *KillAssessment) blastRadius(threat ThreatType) float64 {
	switch threat {
	case THREAT_ICBM:
		return 50.0 // Large RV with heavy warhead
	case THREAT_IRBM:
		return 40.0
	case THREAT_MRBM:
		return 30.0
	case THREAT_SRBM:
		return 20.0
	case THREAT_CRUISE:
		return 15.0
	case THREAT_TBM:
		return 25.0
	default:
		return 30.0
	}
}

// conditionedKillProbability calculates P(kill | intercept geometry)
// Uses exponential falloff model for kill probability vs miss distance
// Reference: Ballistic Missile Defense, AIAA 2010
func (ka *KillAssessment) conditionedKillProbability(missDist, lethalRadius float64) float64 {
	// Probability that target is within lethal radius
	// Uses exponential falloff based on miss distance ratio

	if missDist < lethalRadius {
		// Direct hit or within lethal radius
		return 0.98
	}

	// Exponential falloff
	ratio := missDist / lethalRadius
	Pk := math.Exp(-0.5 * (ratio - 1) * (ratio - 1))

	return math.Max(0, math.Min(1.0, Pk))
}

// HitAssessment determines if intercept was a hit (for battle damage assessment)
func (ka *KillAssessment) HitAssessment(event *InterceptEvent) bool {
	configs := DefaultInterceptorConfigs()
	config := configs[event.InterceptorType]

	rng := ka.distance(event.InterceptPosition, event.TargetPosition)

	// Hit criteria based on interceptor type
	// CEV (Circular Error Probable) based
	hittableRadius := 0.0
	switch config.Type {
	case GMD_GBI:
		hittableRadius = 50.0 // GBI has larger CEV
	case SM3_IIA, SM3_IB:
		hittableRadius = 30.0
	case THAAD:
		hittableRadius = 20.0
	case PATRIOT_PAC3:
		hittableRadius = 10.0
	}

	return rng <= hittableRadius
}

// EngagementZone determines optimal engagement zone for intercept
func (ka *KillAssessment) EngagementZone(
	interceptorType InterceptorType,
	targetAlt float64,
	targetRange float64,
	timeToImpact float64,
) string {

	configs := DefaultInterceptorConfigs()
	config := configs[interceptorType]

	// Check altitude
	if targetAlt > config.MaxAltitude*0.8 {
		return "NO_ENGAGE" // Target too high
	}
	if targetAlt < config.MaxAltitude*0.2 && config.EngagementZone == "TERMINAL" {
		return "TERMINAL"
	}

	// Check range
	if targetRange < config.MinRange*1.5 {
		return "NO_ENGAGE" // Too close
	}
	if targetRange > config.MaxRange*0.9 {
		return "NO_ENGAGE" // Too far
	}

	// Based on time to impact and interceptor type
	switch config.EngagementZone {
	case "BOOST":
		if timeToImpact > 300 { // > 5 min
			return "BOOST"
		}
		return "NO_ENGAGE"
	case "MIDCOURSE":
		if timeToImpact > 120 { // > 2 min
			return "MIDCOURSE"
		}
		return "TERMINAL"
	case "TERMINAL":
		return "TERMINAL"
	}

	return "NO_ENGAGE"
}

// SelectBestInterceptor chooses optimal interceptor for engagement
func SelectBestInterceptor(
	target ThreatType,
	targetAlt, targetRange, timeToImpact float64,
	available []InterceptorType,
) (InterceptorType, float64) {

	ka := DefaultKillAssessment()

	var best InterceptorType
	bestProb := 0.0

	for _, intType := range available {
		zone := ka.EngagementZone(intType, targetAlt, targetRange, timeToImpact)
		if zone == "NO_ENGAGE" {
			continue
		}

		// Estimate miss distance based on interceptor accuracy
		configs := DefaultInterceptorConfigs()
		config := configs[intType]

		// Simplified: use interceptor CEV
		var CEV float64
		switch config.Type {
		case GMD_GBI:
			CEV = 100.0 // Larger CEV for exo-atmospheric
		case SM3_IIA:
			CEV = 30.0
		case SM3_IB:
			CEV = 40.0
		case THAAD:
			CEV = 20.0
		case PATRIOT_PAC3:
			CEV = 10.0
		default:
			CEV = 50.0
		}

		// Simplified probability based on range, time, CEV
		prob := config.HitProbability * (1.0 - CEV/200.0)

		if timeToImpact < 30 {
			prob *= 0.8 // Terminal pressure
		}

		if prob > bestProb {
			bestProb = prob
			best = intType
		}
	}

	return best, bestProb
}
