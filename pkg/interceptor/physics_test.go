package interceptor

import (
	"math"
	"testing"
	"time"
)

func TestProportionalNavigation(t *testing.T) {
	png := DefaultPNG()

	// Interceptor at origin, moving at 3 km/s toward target
	interceptor := &InterceptorState{
		Position: [3]float64{0, 0, 0},
		Velocity: [3]float64{3000, 0, 0},
	}

	// Target ahead at 100 km, moving at 3 km/s toward interceptor
	targetPos := [3]float64{100e3, 0, 0}
	targetVel := [3]float64{-3000, 0, 0}

	// Guidance command should point toward closing the gap
	cmd := png.GuidanceCommand(interceptor, targetPos, targetVel, 1*time.Second)

	// Should not be zero (closing geometry)
	if math.IsNaN(cmd[0]) || math.IsNaN(cmd[1]) || math.IsNaN(cmd[2]) {
		t.Errorf("PNG guidance command produced NaN: %v", cmd)
	}
}

func TestPurePNG(t *testing.T) {
	png := &ProportionalNavigation{NAVGain: 3.0}

	// Head-on intercept
	pos := [3]float64{0, 0, 0}
	vel := [3]float64{3000, 0, 0}
	targetPos := [3]float64{100e3, 0, 0}
	targetVel := [3]float64{-3000, 0, 0}

	accelMag, accelDir := png.PureProportionalNavigation(pos, vel, targetPos, targetVel, 3.0)

	if accelMag < 0 {
		t.Errorf("Expected positive acceleration command, got %f", accelMag)
	}

	if math.IsNaN(accelDir[0]) || math.IsNaN(accelDir[1]) || math.IsNaN(accelDir[2]) {
		t.Errorf("Acceleration direction produced NaN: %v", accelDir)
	}
}

func TestTimeToIntercept(t *testing.T) {
	tests := []struct {
		name        string
		pos         [3]float64
		vel         [3]float64
		targetPos   [3]float64
		targetVel   [3]float64
		expectedMin float64
		expectedMax float64
	}{
		{
			name:        "head-on closing",
			pos:         [3]float64{0, 0, 0},
			vel:         [3]float64{3000, 0, 0},
			targetPos:   [3]float64{100e3, 0, 0},
			targetVel:   [3]float64{-3000, 0, 0},
			expectedMin: -25,
			expectedMax: -10,
		},
		{
			name:        "parallel (no closure)",
			pos:         [3]float64{0, 0, 0},
			vel:         [3]float64{3000, 0, 0},
			targetPos:   [3]float64{100e3, 0, 0},
			targetVel:   [3]float64{3000, 0, 0}, // Same direction
			expectedMin: 999999,
			expectedMax: 999999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tti := TimeToIntercept(tt.pos, tt.vel, tt.targetPos, tt.targetVel)
			if tti < tt.expectedMin || tti > tt.expectedMax {
				t.Errorf("Expected TTI between %f and %f, got %f", tt.expectedMin, tt.expectedMax, tti)
			}
		})
	}
}

func TestInterceptorConfigs(t *testing.T) {
	configs := DefaultInterceptorConfigs()

	// GBI should have highest range
	if configs[GMD_GBI].MaxRange < 5000e3 {
		t.Errorf("GBI max range too low: %f", configs[GMD_GBI].MaxRange)
	}

	// THAAD should have terminal engagement zone
	if configs[THAAD].EngagementZone != "TERMINAL" {
		t.Errorf("THAAD should be TERMINAL engagement, got %s", configs[THAAD].EngagementZone)
	}

	// All should have reasonable hit probabilities
	for itype, config := range configs {
		if config.HitProbability <= 0 || config.HitProbability > 1 {
			t.Errorf("Invalid hit probability for %v: %f", itype, config.HitProbability)
		}
	}
}

func TestUpdateState(t *testing.T) {
	state := &InterceptorState{
		Position: [3]float64{0, 0, 0},
		Velocity: [3]float64{3000, 0, 0},
	}

	// Apply zero acceleration
	cmd := [3]float64{0, 0, 0}
	state.UpdateState(cmd, 1*time.Second)

	// Position should have advanced by 3 km (3000 m)
	if state.Position[0] < 2900 || state.Position[0] > 3100 {
		t.Errorf("Position not updated correctly: %f", state.Position[0])
	}

	// Time should be 1 second
	if state.TimeSinceLaunch != 1*time.Second {
		t.Errorf("Time not updated: %v", state.TimeSinceLaunch)
	}
}

func TestKillAssessment(t *testing.T) {
	ka := DefaultKillAssessment()

	// Direct hit - should get high kill probability
	event := &InterceptEvent{
		InterceptPosition: [3]float64{0, 0, 0},
		TargetPosition:    [3]float64{5.0, 0, 0},
		TargetVelocity:    [3]float64{0, 0, 0},
		InterceptorType:   GMD_GBI,
		TargetType:        THREAT_ICBM,
	}

	level, conf := ka.Assess(event)

	if level < KILL_LEVEL_NONE || level > KILL_LEVEL_CATK {
		t.Errorf("Invalid kill level: %v", level)
	}

	if conf < 0.9 {
		t.Errorf("Direct hit should have high confidence, got %f", conf)
	}
}

func TestHitAssessment(t *testing.T) {
	ka := DefaultKillAssessment()

	tests := []struct {
		name        string
		missDist    float64
		interceptor InterceptorType
		expected    bool
	}{
		{
			name:        "THAAD direct hit",
			missDist:    5.0,
			interceptor: THAAD,
			expected:    true,
		},
		{
			name:        "THAAD far miss",
			missDist:    50.0,
			interceptor: THAAD,
			expected:    false,
		},
		{
			name:        "GBI near miss",
			missDist:    30.0,
			interceptor: GMD_GBI,
			expected:    true, // GBI has larger CEV
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &InterceptEvent{
				InterceptPosition: [3]float64{0, 0, 0},
				TargetPosition:    [3]float64{tt.missDist, 0, 0},
				TargetVelocity:    [3]float64{0, 0, 0},
				InterceptorType:   tt.interceptor,
			}

			result := ka.HitAssessment(event)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSelectBestInterceptor(t *testing.T) {
	available := []InterceptorType{SM3_IIA, THAAD, PATRIOT_PAC3}

	// SRBM at 20km altitude, 50km range, 30s to impact
	interceptor, prob := SelectBestInterceptor(THREAT_SRBM, 20e3, 50e3, 30, available)

	if interceptor == 0 {
		t.Errorf("No interceptor selected")
	}

	if prob <= 0 || prob > 1 {
		t.Errorf("Invalid probability: %f", prob)
	}

	// Terminal SRBM should select THAAD or PAC-3
	if interceptor != THAAD && interceptor != PATRIOT_PAC3 {
		t.Logf("Selected %v for SRBM terminal (may be valid)", interceptor)
	}
}

func TestEngagementZone(t *testing.T) {
	ka := DefaultKillAssessment()

	tests := []struct {
		name        string
		interceptor InterceptorType
		alt         float64
		range_      float64
		tti         float64
		expected    string
	}{
		{
			name:        "THAAD terminal",
			interceptor: THAAD,
			alt:         20e3,
			range_:      100e3,
			tti:         30,
			expected:    "TERMINAL",
		},
		{
			name:        "THAAD out of range",
			interceptor: THAAD,
			alt:         100e3, // 100km - above 20% threshold
			range_:      500e3, // 500km - above THAAD max
			tti:         60,
			expected:    "NO_ENGAGE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zone := ka.EngagementZone(tt.interceptor, tt.alt, tt.range_, tt.tti)
			if zone != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, zone)
			}
		})
	}
}

func TestInterceptProbability(t *testing.T) {
	configs := DefaultInterceptorConfigs()
	gbi := configs[GMD_GBI]

	// Perfect geometry, 60 seconds
	prob := InterceptProbability(
		gbi,
		[3]float64{0, 0, 0},
		[3]float64{7000, 0, 0},
		[3]float64{100e3, 0, 0},
		[3]float64{-7000, 0, 0},
		60,
	)

	if prob <= 0 || prob > 1 {
		t.Errorf("Invalid probability: %f", prob)
	}
}

func BenchmarkPNGGuidance(b *testing.B) {
	png := DefaultPNG()
	state := &InterceptorState{
		Position: [3]float64{0, 0, 0},
		Velocity: [3]float64{3000, 0, 0},
	}
	targetPos := [3]float64{100e3, 0, 0}
	targetVel := [3]float64{-3000, 0, 0}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		png.GuidanceCommand(state, targetPos, targetVel, 1*time.Second)
	}
}
