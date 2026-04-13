// Package doctrine implements alert level rules for missile warning
package doctrine

import (
	"math"
)

// AlertLevel represents the severity of a threat alert
type AlertLevel int

const (
	AlertNone     AlertLevel = 0
	AlertCONOPREP AlertLevel = 1 // Contingency Preparation
	AlertIMMINENT AlertLevel = 2 // Imminent threat
	AlertINCOMING AlertLevel = 3 // Incoming threat
	AlertHOSTILE  AlertLevel = 4 // Hostile action confirmed
)

func (a AlertLevel) String() string {
	switch a {
	case AlertCONOPREP:
		return "CONOPREP"
	case AlertIMMINENT:
		return "IMMINENT"
	case AlertINCOMING:
		return "INCOMING"
	case AlertHOSTILE:
		return "HOSTILE"
	default:
		return "NONE"
	}
}

// ThreatType represents the type of threat
type ThreatType int

const (
	ThreatUnknown    ThreatType = 0
	ThreatBallistic  ThreatType = 1  // Ballistic missile
	ThreatCruise     ThreatType = 2  // Cruise missile
	ThreatAir        ThreatType = 3  // Aircraft
	ThreatUAV        ThreatType = 4  // UAV/drone
	ThreatArtillery  ThreatType = 5  // Artillery/rocket
)

func (t ThreatType) String() string {
	switch t {
	case ThreatBallistic:
		return "BALLISTIC"
	case ThreatCruise:
		return "CRUISE"
	case ThreatAir:
		return "AIRCRAFT"
	case ThreatUAV:
		return "UAV"
	case ThreatArtillery:
		return "ARTILLERY"
	default:
		return "UNKNOWN"
	}
}

// Alert represents an alert to be disseminated
type Alert struct {
	ID           uint64
	TrackNumber  uint32
	AlertLevel   AlertLevel
	ThreatType   ThreatType
	LaunchPoint  LatLonAlt
	ImpactPoint  LatLonAlt
	LaunchTime   int64   // Unix milliseconds
	ImpactTime   int64   // Unix milliseconds
	Confidence   float64 // 0.0 to 1.0
	SourceCount  int
	Heading      float64 // degrees
	Speed        float64 // m/s
	Altitude     float64 // meters
}

// LatLonAlt represents a position
type LatLonAlt struct {
	Lat float64 // degrees
	Lon float64 // degrees
	Alt float64 // meters
}

// AlertRule defines doctrine for alert level determination
type AlertRule struct {
	MinConfidence  float64
	MaxTimeToImpact float64 // seconds
	MinAltitude    float64
	MaxAltitude    float64
	MinSpeed       float64
	MaxSpeed       float64
	ThreatTypes    []ThreatType
	Level          AlertLevel
}

// DefaultDoctrine is the standard alert doctrine
var DefaultDoctrine = []AlertRule{
	// CONOPREP: Any track with confidence > 0.5
	{
		MinConfidence:  0.5,
		MaxTimeToImpact: 300, // 5 minutes
		Level:          AlertCONOPREP,
	},
	// IMMINENT: High confidence, time to impact < 2 min
	{
		MinConfidence:   0.7,
		MaxTimeToImpact: 120, // 2 minutes
		Level:           AlertIMMINENT,
	},
	// INCOMING: Very high confidence, time to impact < 30 sec
	{
		MinConfidence:   0.85,
		MaxTimeToImpact: 30, // 30 seconds
		Level:           AlertINCOMING,
	},
	// HOSTILE: Confirmed threat, impact imminent
	{
		MinConfidence:   0.95,
		MaxTimeToImpact: 10, // 10 seconds
		Level:           AlertHOSTILE,
	},
}

// DetermineAlertLevel determines the appropriate alert level for a track
func DetermineAlertLevel(alert *Alert, doctrine []AlertRule) AlertLevel {
	if doctrine == nil {
		doctrine = DefaultDoctrine
	}
	
	// Calculate time to impact
	timeToImpact := float64(alert.ImpactTime-alert.LaunchTime) / 1000.0
	if timeToImpact < 0 {
		timeToImpact = 0
	}
	
	// Check rules in order (highest to lowest)
	// Start from highest alert level
	for i := len(doctrine) - 1; i >= 0; i-- {
		rule := doctrine[i]
		
		// Check confidence threshold
		if alert.Confidence < rule.MinConfidence {
			continue
		}
		
		// Check time to impact
		if rule.MaxTimeToImpact > 0 && timeToImpact > rule.MaxTimeToImpact {
			continue
		}
		
		// Check altitude range
		if rule.MinAltitude > 0 && alert.Altitude < rule.MinAltitude {
			continue
		}
		if rule.MaxAltitude > 0 && alert.Altitude > rule.MaxAltitude {
			continue
		}
		
		// Check speed range
		if rule.MinSpeed > 0 && alert.Speed < rule.MinSpeed {
			continue
		}
		if rule.MaxSpeed > 0 && alert.Speed > rule.MaxSpeed {
			continue
		}
		
		// Check threat types
		if len(rule.ThreatTypes) > 0 {
			found := false
			for _, tt := range rule.ThreatTypes {
				if tt == alert.ThreatType {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		
		return rule.Level
	}
	
	return AlertNone
}

// EscalateAlert escalates an alert level based on new information
func EscalateAlert(current, newLevel AlertLevel) AlertLevel {
	if newLevel > current {
		return newLevel
	}
	return current
}

// DeescalateAlert deescalates an alert level based on new information
func DeescalateAlert(current AlertLevel, confidence float64) AlertLevel {
	// Only deescalate if confidence drops significantly
	if confidence < 0.3 {
		switch current {
		case AlertHOSTILE:
			return AlertINCOMING
		case AlertINCOMING:
			return AlertIMMINENT
		case AlertIMMINENT:
			return AlertCONOPREP
		case AlertCONOPREP:
			return AlertNone
		}
	}
	return current
}

// ShouldAlert determines if an alert should be generated
func ShouldAlert(alert *Alert, doctrine []AlertRule) bool {
	level := DetermineAlertLevel(alert, doctrine)
	return level != AlertNone
}

// FormatAlertMessage formats an alert for transmission
func FormatAlertMessage(alert *Alert) string {
	return formatAlert(alert)
}

func formatAlert(alert *Alert) string {
	// Format as USMTF-like message
	// Simplified format for demonstration
	msg := "ALERT\n"
	msg += "LEVEL: " + alert.AlertLevel.String() + "\n"
	msg += "TRACK: " + string(rune(alert.TrackNumber)) + "\n"
	msg += "TYPE: " + alert.ThreatType.String() + "\n"
	msg += "CONFIDENCE: " + string(rune(int(alert.Confidence*100))) + "%\n"
	return msg
}

// EstimateTimeToImpact estimates time to impact from track data
func EstimateTimeToImpact(alert *Alert) float64 {
	if alert.ImpactTime <= 0 {
		return math.Inf(1)
	}
	
	now := alert.LaunchTime
	if now <= 0 {
		// Use current time
		return float64(alert.ImpactTime) / 1000.0
	}
	
	return float64(alert.ImpactTime-now) / 1000.0
}

// EstimateConfidence estimates confidence based on track quality
func EstimateConfidence(sourceCount int, timeSinceLastUpdate int64) float64 {
	// More sources = higher confidence
	confidence := math.Min(1.0, float64(sourceCount)/5.0)
	
	// Time since last update reduces confidence
	ageSeconds := float64(timeSinceLastUpdate) / 1000.0
	ageFactor := math.Exp(-ageSeconds / 60.0) // Decay over 60 seconds
	
	return confidence * ageFactor
}