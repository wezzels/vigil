package geo

import (
	"math"
	"testing"
	"time"
)

// TestRegistrationConfig tests registration configuration
func TestRegistrationConfig(t *testing.T) {
	config := DefaultRegistrationConfig()
	
	if config.MinSamples != 10 {
		t.Errorf("Expected min samples 10, got %d", config.MinSamples)
	}
	if config.BiasThreshold != 50.0 {
		t.Errorf("Expected bias threshold 50, got %f", config.BiasThreshold)
	}
	if config.AdaptiveRate != 0.1 {
		t.Errorf("Expected adaptive rate 0.1, got %f", config.AdaptiveRate)
	}
}

// TestNewSensorRegistration tests registration creation
func TestNewSensorRegistration(t *testing.T) {
	sr := NewSensorRegistration(nil)
	
	if sr == nil {
		t.Fatal("Registration should not be nil")
	}
	
	if sr.config.MinSamples != 10 {
		t.Error("Default config should be used")
	}
}

// TestInitializeSensor tests sensor initialization
func TestInitializeSensor(t *testing.T) {
	sr := NewSensorRegistration(nil)
	
	now := time.Now()
	bias := sr.InitializeSensor("TPY2-1", now)
	
	if bias == nil {
		t.Fatal("Bias should not be nil")
	}
	
	if bias.SensorID != "TPY2-1" {
		t.Errorf("Expected sensor ID TPY2-1, got %s", bias.SensorID)
	}
	
	if bias.Status != "ESTIMATING" {
		t.Errorf("Expected status ESTIMATING, got %s", bias.Status)
	}
}

// TestAddResidual tests adding residuals
func TestAddResidual(t *testing.T) {
	sr := NewSensorRegistration(nil)
	
	now := time.Now()
	sr.InitializeSensor("TPY2-1", now)
	
	measured := [3]float64{1000, 0, 10000}
	estimated := [3]float64{1010, 5, 10015}
	
	sr.AddResidual("TPY2-1", measured, estimated, now)
	
	residuals := sr.GetResiduals("TPY2-1")
	
	if len(residuals) != 1 {
		t.Errorf("Expected 1 residual, got %d", len(residuals))
	}
	
	r := residuals[0]
	if r.ResidualPos[0] != 10 {
		t.Errorf("Expected residual x=10, got %.2f", r.ResidualPos[0])
	}
}

// TestBiasEstimation tests bias estimation with multiple residuals
func TestBiasEstimation(t *testing.T) {
	config := DefaultRegistrationConfig()
	config.MinSamples = 5 // Lower for testing
	config.AdaptiveRate = 0.5 // Faster adaptation
	sr := NewSensorRegistration(config)
	
	now := time.Now()
	sr.InitializeSensor("TPY2-1", now)
	
	// Add consistent residuals (10m bias)
	for i := 0; i < 20; i++ {
		measured := [3]float64{1000, 0, 10000}
		estimated := [3]float64{1010, 0, 10000} // 10m bias in x
		sr.AddResidual("TPY2-1", measured, estimated, now.Add(time.Duration(i)*time.Second))
	}
	
	bias := sr.GetBias("TPY2-1")
	
	if bias.NumSamples < 5 {
		t.Errorf("Expected at least 5 samples, got %d", bias.NumSamples)
	}
	
	// Bias should be estimated (direction correct, not necessarily exact value)
	if bias.PositionBias[0] < 5 {
		t.Errorf("Bias should track residuals, got %.2f (expected ~10m)", bias.PositionBias[0])
	}
}

// TestCorrectPosition tests position correction
func TestCorrectPosition(t *testing.T) {
	config := DefaultRegistrationConfig()
	config.MinSamples = 5
	config.StabilityThreshold = 20 // Allow stable at 20m
	sr := NewSensorRegistration(config)
	
	now := time.Now()
	sr.InitializeSensor("TPY2-1", now)
	
	// Add residuals to establish bias
	for i := 0; i < 10; i++ {
		measured := [3]float64{1000, 0, 10000}
		estimated := [3]float64{1010, 0, 10000}
		sr.AddResidual("TPY2-1", measured, estimated, now.Add(time.Duration(i)*time.Second))
	}
	
	// Test correction
	position := [3]float64{2000, 100, 20000}
	corrected := sr.CorrectPosition("TPY2-1", position)
	
	// Should subtract bias
	if corrected[0] >= position[0] {
		t.Errorf("Corrected x should be less than original: %.2f vs %.2f", corrected[0], position[0])
	}
}

// TestCorrectVelocity tests velocity correction
func TestCorrectVelocity(t *testing.T) {
	config := DefaultRegistrationConfig()
	config.MinSamples = 5
	sr := NewSensorRegistration(config)
	
	now := time.Now()
	bias := sr.InitializeSensor("TPY2-1", now)
	bias.VelocityBias = [3]float64{5, 0, 0}
	bias.Status = "STABLE"
	
	velocity := [3]float64{100, 50, 10}
	corrected := sr.CorrectVelocity("TPY2-1", velocity)
	
	if corrected[0] != 95 {
		t.Errorf("Expected corrected vx=95, got %.2f", corrected[0])
	}
}

// TestCorrectRange tests range correction
func TestCorrectRange(t *testing.T) {
	config := DefaultRegistrationConfig()
	config.MinSamples = 5
	sr := NewSensorRegistration(config)
	
	now := time.Now()
	bias := sr.InitializeSensor("TPY2-1", now)
	bias.RangeBias = 50 // 50m range bias
	bias.Status = "STABLE"
	
	rangeVal := 1000.0
	corrected := sr.CorrectRange("TPY2-1", rangeVal)
	
	if corrected != 950 {
		t.Errorf("Expected corrected range=950, got %.2f", corrected)
	}
}

// TestCorrectAngles tests angle correction
func TestCorrectAngles(t *testing.T) {
	sr := NewSensorRegistration(nil)
	
	now := time.Now()
	bias := sr.InitializeSensor("TPY2-1", now)
	bias.AngleBias = [2]float64{0.01, 0.005} // Azimuth and elevation bias
	bias.Status = "STABLE"
	
	azimuth := 1.0
	elevation := 0.5
	
	corrAz, corrEl := sr.CorrectAngles("TPY2-1", azimuth, elevation)
	
	if corrAz != 0.99 {
		t.Errorf("Expected corrected azimuth=0.99, got %.4f", corrAz)
	}
	if corrEl != 0.495 {
		t.Errorf("Expected corrected elevation=0.495, got %.4f", corrEl)
	}
}

// TestResetBias tests bias reset
func TestResetBias(t *testing.T) {
	sr := NewSensorRegistration(nil)
	
	now := time.Now()
	bias := sr.InitializeSensor("TPY2-1", now)
	bias.PositionBias = [3]float64{10, 5, 3}
	bias.Status = "STABLE"
	
	sr.ResetBias("TPY2-1")
	
	bias = sr.GetBias("TPY2-1")
	
	if bias.PositionBias[0] != 0 {
		t.Errorf("Expected reset bias=0, got %.2f", bias.PositionBias[0])
	}
	
	if bias.Status != "ESTIMATING" {
		t.Errorf("Expected status ESTIMATING after reset, got %s", bias.Status)
	}
}

// TestRemoveSensor tests sensor removal
func TestRemoveSensor(t *testing.T) {
	sr := NewSensorRegistration(nil)
	
	now := time.Now()
	sr.InitializeSensor("TPY2-1", now)
	
	sr.RemoveSensor("TPY2-1")
	
	bias := sr.GetBias("TPY2-1")
	if bias != nil {
		t.Error("Sensor should be removed")
	}
}

// TestGetStableSensors tests getting stable sensors
func TestGetStableSensors(t *testing.T) {
	config := DefaultRegistrationConfig()
	config.MinSamples = 3
	config.StabilityThreshold = 20
	sr := NewSensorRegistration(config)
	
	now := time.Now()
	sr.InitializeSensor("TPY2-1", now)
	sr.InitializeSensor("TPY2-2", now)
	sr.InitializeSensor("SBX-1", now)
	
	// Make TPY2-1 and TPY2-2 stable
	for _, sensorID := range []string{"TPY2-1", "TPY2-2"} {
		for i := 0; i < 5; i++ {
			measured := [3]float64{1000, 0, 10000}
			estimated := [3]float64{1005, 0, 10000} // Small bias
			sr.AddResidual(sensorID, measured, estimated, now.Add(time.Duration(i)*time.Second))
		}
	}
	
	stable := sr.GetStableSensors()
	
	if len(stable) < 2 {
		t.Errorf("Expected at least 2 stable sensors, got %d", len(stable))
	}
}

// TestStats tests registration statistics
func TestStats(t *testing.T) {
	sr := NewSensorRegistration(nil)
	
	now := time.Now()
	sr.InitializeSensor("TPY2-1", now)
	sr.InitializeSensor("TPY2-2", now)
	sr.InitializeSensor("SBX-1", now)
	
	// Make one stable
	sr.sensors["TPY2-1"].Status = "STABLE"
	sr.sensors["TPY2-2"].Status = "UNSTABLE"
	
	stats := sr.Stats()
	
	if stats.TotalSensors != 3 {
		t.Errorf("Expected 3 total sensors, got %d", stats.TotalSensors)
	}
	
	if stats.Estimating != 1 {
		t.Errorf("Expected 1 estimating, got %d", stats.Estimating)
	}
	
	if stats.Stable != 1 {
		t.Errorf("Expected 1 stable, got %d", stats.Stable)
	}
	
	if stats.Unstable != 1 {
		t.Errorf("Expected 1 unstable, got %d", stats.Unstable)
	}
}

// TestCalculateRMS tests RMS calculation
func TestCalculateRMS(t *testing.T) {
	sr := NewSensorRegistration(nil)
	
	now := time.Now()
	sr.InitializeSensor("TPY2-1", now)
	
	// Add residuals with known magnitude
	for i := 0; i < 3; i++ {
		measured := [3]float64{0, 0, 0}
		estimated := [3]float64{3, 4, 0} // 5m magnitude each
		sr.AddResidual("TPY2-1", measured, estimated, now.Add(time.Duration(i)*time.Second))
	}
	
	rms := sr.CalculateRMS("TPY2-1")
	
	// RMS of (5, 5, 5) = 5
	if math.Abs(rms-5.0) > 0.1 {
		t.Errorf("Expected RMS ~5, got %.2f", rms)
	}
}

// TestCalculateBiasMagnitude tests bias magnitude calculation
func TestCalculateBiasMagnitude(t *testing.T) {
	sr := NewSensorRegistration(nil)
	
	now := time.Now()
	bias := sr.InitializeSensor("TPY2-1", now)
	bias.PositionBias = [3]float64{3, 4, 0} // 5m magnitude
	
	magnitude := sr.CalculateBiasMagnitude("TPY2-1")
	
	if math.Abs(magnitude-5.0) > 0.01 {
		t.Errorf("Expected magnitude 5, got %.2f", magnitude)
	}
}

// TestAdaptiveEstimation tests adaptive bias updates
func TestAdaptiveEstimation(t *testing.T) {
	config := DefaultRegistrationConfig()
	config.AdaptiveRate = 0.5 // Faster adaptation
	config.MinSamples = 3
	sr := NewSensorRegistration(config)
	
	now := time.Now()
	sr.InitializeSensor("TPY2-1", now)
	
	// Add residuals with increasing bias
	for i := 0; i < 10; i++ {
		measured := [3]float64{1000, 0, 10000}
		estimated := [3]float64{1000 + float64(i)*10, 0, 10000}
		sr.AddResidual("TPY2-1", measured, estimated, now.Add(time.Duration(i)*time.Second))
	}
	
	bias := sr.GetBias("TPY2-1")
	
	// Bias should track the trend (but not fully)
	if bias.PositionBias[0] < 10 {
		t.Errorf("Bias should track increasing residuals, got %.2f", bias.PositionBias[0])
	}
}

// TestMultipleSensors tests handling multiple sensors
func TestMultipleSensors(t *testing.T) {
	sr := NewSensorRegistration(nil)
	
	now := time.Now()
	
	// Initialize multiple sensors
	sensors := []string{"TPY2-1", "TPY2-2", "SBX-1", "UEWR-1"}
	for _, id := range sensors {
		sr.InitializeSensor(id, now)
	}
	
	// Add residuals for each
	for _, id := range sensors {
		measured := [3]float64{1000, 0, 10000}
		estimated := [3]float64{1010, 0, 10000}
		sr.AddResidual(id, measured, estimated, now)
	}
	
	// Check all sensors have residuals
	for _, id := range sensors {
		residuals := sr.GetResiduals(id)
		if len(residuals) != 1 {
			t.Errorf("Expected 1 residual for %s, got %d", id, len(residuals))
		}
	}
	
	// Check all biases
	biases := sr.GetAllBiases()
	if len(biases) != 4 {
		t.Errorf("Expected 4 sensors, got %d", len(biases))
	}
}

// TestMaxResiduals tests maximum residual limit
func TestMaxResiduals(t *testing.T) {
	config := DefaultRegistrationConfig()
	config.MaxSamples = 10
	sr := NewSensorRegistration(config)
	
	now := time.Now()
	sr.InitializeSensor("TPY2-1", now)
	
	// Add more residuals than max
	for i := 0; i < 20; i++ {
		measured := [3]float64{1000, 0, 10000}
		estimated := [3]float64{1010, 0, 10000}
		sr.AddResidual("TPY2-1", measured, estimated, now.Add(time.Duration(i)*time.Second))
	}
	
	residuals := sr.GetResiduals("TPY2-1")
	
	if len(residuals) > config.MaxSamples {
		t.Errorf("Expected at most %d residuals, got %d", config.MaxSamples, len(residuals))
	}
}

// BenchmarkAddResidual benchmarks adding residuals
func BenchmarkAddResidual(b *testing.B) {
	sr := NewSensorRegistration(nil)
	now := time.Now()
	sr.InitializeSensor("TPY2-1", now)
	
	measured := [3]float64{1000, 0, 10000}
	estimated := [3]float64{1010, 0, 10000}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sr.AddResidual("TPY2-1", measured, estimated, now.Add(time.Duration(i)*time.Millisecond))
	}
}

// BenchmarkCorrectPosition benchmarks position correction
func BenchmarkCorrectPosition(b *testing.B) {
	sr := NewSensorRegistration(nil)
	now := time.Now()
	bias := sr.InitializeSensor("TPY2-1", now)
	bias.Status = "STABLE"
	bias.PositionBias = [3]float64{10, 5, 2}
	
	position := [3]float64{1000, 500, 10000}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sr.CorrectPosition("TPY2-1", position)
	}
}

// BenchmarkBiasEstimation benchmarks bias estimation
func BenchmarkBiasEstimation(b *testing.B) {
	config := DefaultRegistrationConfig()
	config.MinSamples = 10
	sr := NewSensorRegistration(config)
	
	now := time.Now()
	sr.InitializeSensor("TPY2-1", now)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		measured := [3]float64{1000, 0, 10000}
		estimated := [3]float64{1000 + float64(i%10), 0, 10000}
		sr.AddResidual("TPY2-1", measured, estimated, now.Add(time.Duration(i)*time.Millisecond))
	}
}