package doctrine

import (
	"math"
	"testing"
)

// TestDetermineAlertLevelNone tests no alert case
func TestDetermineAlertLevelNone(t *testing.T) {
	alert := &Alert{
		Confidence: 0.1,
	}

	level := DetermineAlertLevel(alert, nil)
	if level != AlertNone {
		t.Errorf("Expected AlertNone for low confidence, got %v", level)
	}
}

// TestDetermineAlertLevelCONOPREP tests CONOPREP alert
func TestDetermineAlertLevelCONOPREP(t *testing.T) {
	alert := &Alert{
		Confidence: 0.6,
		LaunchTime: 1000000,
		ImpactTime: 160000, // 160 seconds = 2.67 min
	}

	level := DetermineAlertLevel(alert, nil)
	if level != AlertCONOPREP {
		t.Errorf("Expected CONOPREP for confidence 0.6, got %v", level)
	}
}

// TestDetermineAlertLevelIMMINENT tests IMMINENT alert
func TestDetermineAlertLevelIMMINENT(t *testing.T) {
	alert := &Alert{
		Confidence: 0.75,
		LaunchTime: 1000000,
		ImpactTime: 1100000, // 100 seconds
	}

	level := DetermineAlertLevel(alert, nil)
	if level != AlertIMMINENT {
		t.Errorf("Expected IMMINENT for confidence 0.75, got %v", level)
	}
}

// TestDetermineAlertLevelINCOMING tests INCOMING alert
func TestDetermineAlertLevelINCOMING(t *testing.T) {
	alert := &Alert{
		Confidence: 0.9,
		LaunchTime: 1000000,
		ImpactTime: 1025000, // 25 seconds
	}

	level := DetermineAlertLevel(alert, nil)
	if level != AlertINCOMING {
		t.Errorf("Expected INCOMING for confidence 0.9, got %v", level)
	}
}

// TestDetermineAlertLevelHOSTILE tests HOSTILE alert
func TestDetermineAlertLevelHOSTILE(t *testing.T) {
	alert := &Alert{
		Confidence: 0.97,
		LaunchTime: 1000000,
		ImpactTime: 1005000, // 5 seconds
	}

	level := DetermineAlertLevel(alert, nil)
	if level != AlertHOSTILE {
		t.Errorf("Expected HOSTILE for confidence 0.97, got %v", level)
	}
}

// TestAlertEscalation tests alert escalation
func TestAlertEscalation(t *testing.T) {
	tests := []struct {
		current  AlertLevel
		new      AlertLevel
		expected AlertLevel
	}{
		{AlertNone, AlertCONOPREP, AlertCONOPREP},
		{AlertCONOPREP, AlertIMMINENT, AlertIMMINENT},
		{AlertIMMINENT, AlertINCOMING, AlertINCOMING},
		{AlertINCOMING, AlertHOSTILE, AlertHOSTILE},
		{AlertHOSTILE, AlertIMMINENT, AlertHOSTILE},   // No deescalation
		{AlertIMMINENT, AlertCONOPREP, AlertIMMINENT}, // No deescalation
	}

	for _, tt := range tests {
		result := EscalateAlert(tt.current, tt.new)
		if result != tt.expected {
			t.Errorf("EscalateAlert(%v, %v) = %v, expected %v",
				tt.current, tt.new, result, tt.expected)
		}
	}
}

// TestAlertDeescalation tests alert deescalation
func TestAlertDeescalation(t *testing.T) {
	tests := []struct {
		current    AlertLevel
		confidence float64
		expected   AlertLevel
	}{
		{AlertHOSTILE, 0.5, AlertHOSTILE},  // High confidence -> stay
		{AlertHOSTILE, 0.2, AlertINCOMING}, // Low confidence -> deescalate
		{AlertINCOMING, 0.2, AlertIMMINENT},
		{AlertIMMINENT, 0.2, AlertCONOPREP},
		{AlertCONOPREP, 0.2, AlertNone},
		{AlertNone, 0.1, AlertNone},
	}

	for _, tt := range tests {
		result := DeescalateAlert(tt.current, tt.confidence)
		if result != tt.expected {
			t.Errorf("DeescalateAlert(%v, %.1f) = %v, expected %v",
				tt.current, tt.confidence, result, tt.expected)
		}
	}
}

// TestThreatTypeString tests threat type string conversion
func TestThreatTypeString(t *testing.T) {
	tests := []struct {
		t        ThreatType
		expected string
	}{
		{ThreatUnknown, "UNKNOWN"},
		{ThreatBallistic, "BALLISTIC"},
		{ThreatCruise, "CRUISE"},
		{ThreatAir, "AIRCRAFT"},
		{ThreatUAV, "UAV"},
		{ThreatArtillery, "ARTILLERY"},
	}

	for _, tt := range tests {
		result := tt.t.String()
		if result != tt.expected {
			t.Errorf("ThreatType(%d).String() = %s, expected %s",
				tt.t, result, tt.expected)
		}
	}
}

// TestAlertLevelString tests alert level string conversion
func TestAlertLevelString(t *testing.T) {
	tests := []struct {
		level    AlertLevel
		expected string
	}{
		{AlertNone, "NONE"},
		{AlertCONOPREP, "CONOPREP"},
		{AlertIMMINENT, "IMMINENT"},
		{AlertINCOMING, "INCOMING"},
		{AlertHOSTILE, "HOSTILE"},
	}

	for _, tt := range tests {
		result := tt.level.String()
		if result != tt.expected {
			t.Errorf("AlertLevel(%d).String() = %s, expected %s",
				tt.level, result, tt.expected)
		}
	}
}

// TestShouldAlert tests alert decision
func TestShouldAlert(t *testing.T) {
	tests := []struct {
		confidence  float64
		shouldAlert bool
	}{
		{0.1, false}, // Too low
		{0.5, true},  // Meets CONOPREP threshold
		{0.7, true},  // Meets IMMINENT threshold
		{0.9, true},  // Meets INCOMING threshold
	}

	for _, tt := range tests {
		alert := &Alert{
			Confidence: tt.confidence,
			LaunchTime: 1000000,
			ImpactTime: 1100000, // 100 seconds
		}

		result := ShouldAlert(alert, nil)
		if result != tt.shouldAlert {
			t.Errorf("ShouldAlert(conf=%.1f) = %v, expected %v",
				tt.confidence, result, tt.shouldAlert)
		}
	}
}

// TestEstimateTimeToImpact tests TTI estimation
func TestEstimateTimeToImpact(t *testing.T) {
	tests := []struct {
		launch   int64
		impact   int64
		expected float64
	}{
		{1000000, 1060000, 60.0},  // 60 seconds
		{1000000, 1300000, 300.0}, // 5 minutes
		{1000000, 1600000, 600.0}, // 10 minutes
		{0, 1000, 1.0},            // No launch time
	}

	for _, tt := range tests {
		alert := &Alert{
			LaunchTime: tt.launch,
			ImpactTime: tt.impact,
		}

		result := EstimateTimeToImpact(alert)
		if math.Abs(result-tt.expected) > 0.1 {
			t.Errorf("EstimateTimeToImpact(%d, %d) = %.1f, expected %.1f",
				tt.launch, tt.impact, result, tt.expected)
		}
	}
}

// TestEstimateConfidence tests confidence estimation
func TestEstimateConfidence(t *testing.T) {
	tests := []struct {
		sources    int
		ageSeconds int64
		expected   float64
	}{
		{1, 0, 0.2},   // Single source, fresh
		{5, 0, 1.0},   // Multiple sources, fresh
		{1, 60, 0.07}, // Single source, old
		{5, 60, 0.37}, // Multiple sources, old
	}

	for _, tt := range tests {
		result := EstimateConfidence(tt.sources, tt.ageSeconds*1000)
		// Allow 10% tolerance
		if math.Abs(result-tt.expected) > 0.1 {
			t.Errorf("EstimateConfidence(%d, %d) = %.2f, expected %.2f",
				tt.sources, tt.ageSeconds, result, tt.expected)
		}
	}
}

// TestCustomDoctrine tests custom doctrine rules
func TestCustomDoctrine(t *testing.T) {
	customDoctrine := []AlertRule{
		{
			MinConfidence:   0.8,
			MaxTimeToImpact: 60,
			Level:           AlertIMMINENT,
		},
		{
			MinConfidence:   0.6,
			MaxTimeToImpact: 180,
			Level:           AlertCONOPREP,
		},
	}

	// Test with custom doctrine
	alert := &Alert{
		Confidence: 0.75,
		LaunchTime: 1000000,
		ImpactTime: 1150000, // 150 seconds
	}

	level := DetermineAlertLevel(alert, customDoctrine)
	if level != AlertCONOPREP {
		t.Errorf("Expected CONOPREP with custom doctrine, got %v", level)
	}
}

// TestAltitudeRange tests altitude filtering
func TestAltitudeRange(t *testing.T) {
	doctrine := []AlertRule{
		{
			MinConfidence: 0.7,
			MinAltitude:   10000, // 10 km
			MaxAltitude:   50000, // 50 km
			Level:         AlertIMMINENT,
		},
	}

	tests := []struct {
		altitude float64
		expected AlertLevel
	}{
		{5000, AlertNone},      // Below range
		{20000, AlertIMMINENT}, // In range
		{60000, AlertNone},     // Above range
	}

	for _, tt := range tests {
		alert := &Alert{
			Confidence: 0.8,
			Altitude:   tt.altitude,
			LaunchTime: 1000000,
			ImpactTime: 1060000,
		}

		level := DetermineAlertLevel(alert, doctrine)
		if level != tt.expected {
			t.Errorf("Altitude %.0f: expected %v, got %v",
				tt.altitude, tt.expected, level)
		}
	}
}

// TestSpeedRange tests speed filtering
func TestSpeedRange(t *testing.T) {
	doctrine := []AlertRule{
		{
			MinConfidence: 0.7,
			MinSpeed:      500,  // 500 m/s (Mach 1.5)
			MaxSpeed:      5000, // 5 km/s
			Level:         AlertIMMINENT,
		},
	}

	tests := []struct {
		speed    float64
		expected AlertLevel
	}{
		{100, AlertNone},      // Below range
		{2000, AlertIMMINENT}, // In range
		{6000, AlertNone},     // Above range
	}

	for _, tt := range tests {
		alert := &Alert{
			Confidence: 0.8,
			Speed:      tt.speed,
			LaunchTime: 1000000,
			ImpactTime: 1060000,
		}

		level := DetermineAlertLevel(alert, doctrine)
		if level != tt.expected {
			t.Errorf("Speed %.0f: expected %v, got %v",
				tt.speed, tt.expected, level)
		}
	}
}

// TestThreatTypeFilter tests threat type filtering
func TestThreatTypeFilter(t *testing.T) {
	doctrine := []AlertRule{
		{
			MinConfidence: 0.7,
			ThreatTypes:   []ThreatType{ThreatBallistic, ThreatCruise},
			Level:         AlertIMMINENT,
		},
	}

	tests := []struct {
		threatType ThreatType
		expected   AlertLevel
	}{
		{ThreatBallistic, AlertIMMINENT},
		{ThreatCruise, AlertIMMINENT},
		{ThreatAir, AlertNone}, // Not in list
		{ThreatUAV, AlertNone}, // Not in list
	}

	for _, tt := range tests {
		alert := &Alert{
			Confidence: 0.8,
			ThreatType: tt.threatType,
			LaunchTime: 1000000,
			ImpactTime: 1060000,
		}

		level := DetermineAlertLevel(alert, doctrine)
		if level != tt.expected {
			t.Errorf("ThreatType %v: expected %v, got %v",
				tt.threatType, tt.expected, level)
		}
	}
}

// BenchmarkDetermineAlertLevel benchmarks alert level determination
func BenchmarkDetermineAlertLevel(b *testing.B) {
	alert := &Alert{
		Confidence: 0.85,
		ThreatType: ThreatBallistic,
		LaunchTime: 1000000,
		ImpactTime: 1030000,
		Altitude:   30000,
		Speed:      2000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DetermineAlertLevel(alert, nil)
	}
}

// BenchmarkEstimateConfidence benchmarks confidence estimation
func BenchmarkEstimateConfidence(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EstimateConfidence(5, 30000)
	}
}
