package nca

import (
	"testing"
	"time"
)

// TestCONOPREPFormatter tests CONOPREP message formatting
func TestCONOPREPFormatter(t *testing.T) {
	formatter := NewCONOPREPFormatter("SECRET", "OPIR-DEMO", "NCA-001")

	// Test threat alert
	msg := formatter.FormatThreatAlert("THREAT DETECTED", "Missile activity detected", nil)
	if msg == nil {
		t.Fatal("Expected non-nil message")
	}
	if msg.Classification != "SECRET" {
		t.Errorf("Expected SECRET, got %s", msg.Classification)
	}
	if msg.Category != CategoryThreat {
		t.Errorf("Expected CategoryThreat, got %d", msg.Category)
	}
	if msg.Priority != PriorityPriority {
		t.Errorf("Expected PriorityPriority, got %d", msg.Priority)
	}
}

// TestLaunchAlert tests launch alert formatting
func TestLaunchAlert(t *testing.T) {
	formatter := NewCONOPREPFormatter("SECRET", "OPIR-DEMO", "NCA-001")

	track := &TrackSummary{
		TrackNumber: "T-001",
		CurrentPosition: Position{
			Latitude:  45.0,
			Longitude: -120.0,
			Altitude:  10000.0,
		},
		Confidence: 0.95,
	}

	msg := formatter.FormatLaunchAlert("45N 120W", "12:00:00Z", track)
	if msg == nil {
		t.Fatal("Expected non-nil message")
	}
	if msg.Category != CategoryLaunch {
		t.Errorf("Expected CategoryLaunch, got %d", msg.Category)
	}
	if msg.Priority != PriorityFlash {
		t.Errorf("Expected PriorityFlash, got %d", msg.Priority)
	}
	if msg.RequiredAction != ActionPrepare {
		t.Errorf("Expected ActionPrepare, got %d", msg.RequiredAction)
	}
}

// TestImpactAlert tests impact alert formatting
func TestImpactAlert(t *testing.T) {
	formatter := NewCONOPREPFormatter("SECRET", "OPIR-DEMO", "NCA-001")

	impact := &ImpactSummary{
		Location: Position{
			Latitude:  35.0,
			Longitude: -100.0,
			Altitude:  0.0,
		},
		EstimatedTime: time.Now().Add(30 * time.Minute),
		Confidence:    0.95,
		Warnings:      []string{"POPULATED AREA", "MILITARY FACILITY"},
	}

	msg := formatter.FormatImpactAlert(impact, ActionEvacuate)
	if msg == nil {
		t.Fatal("Expected non-nil message")
	}
	if msg.Category != CategoryImpact {
		t.Errorf("Expected CategoryImpact, got %d", msg.Category)
	}
	if msg.Priority != PriorityFlashOverride {
		t.Errorf("Expected PriorityFlashOverride, got %d", msg.Priority)
	}
	if msg.RequiredAction != ActionEvacuate {
		t.Errorf("Expected ActionEvacuate, got %d", msg.RequiredAction)
	}
}

// TestDefenseAlert tests defense alert formatting
func TestDefenseAlert(t *testing.T) {
	formatter := NewCONOPREPFormatter("SECRET", "OPIR-DEMO", "NCA-001")

	msg := formatter.FormatDefenseAlert("DEFENSE ACTIVATION", "GBI battery activated", ActionDefend)
	if msg == nil {
		t.Fatal("Expected non-nil message")
	}
	if msg.Category != CategoryDefense {
		t.Errorf("Expected CategoryDefense, got %d", msg.Category)
	}
	if msg.RequiredAction != ActionDefend {
		t.Errorf("Expected ActionDefend, got %d", msg.RequiredAction)
	}
}

// TestExerciseAlert tests exercise alert formatting
func TestExerciseAlert(t *testing.T) {
	formatter := NewCONOPREPFormatter("SECRET", "OPIR-DEMO", "NCA-001")

	msg := formatter.FormatExerciseAlert("THREAT SCENARIO", "Simulated launch detected")
	if msg == nil {
		t.Fatal("Expected non-nil message")
	}
	if msg.Category != CategoryExercise {
		t.Errorf("Expected CategoryExercise, got %d", msg.Category)
	}
	if msg.Priority != PriorityRoutine {
		t.Errorf("Expected PriorityRoutine, got %d", msg.Priority)
	}
	if msg.RequiredAction != ActionNone {
		t.Errorf("Expected ActionNone, got %d", msg.RequiredAction)
	}
}

// TestIMMINENTFormatter tests IMMINENT message formatting
func TestIMMINENTFormatter(t *testing.T) {
	formatter := NewIMMINENTFormatter("SECRET")

	launchPos := Position{Latitude: 45.0, Longitude: -120.0, Altitude: 0}
	impactPos := Position{Latitude: 35.0, Longitude: -100.0, Altitude: 0}
	launchTime := time.Now().Add(-5 * time.Minute)
	impactTime := time.Now().Add(25 * time.Minute)

	msg := formatter.Format("THREAT-001", launchPos, impactPos, launchTime, impactTime, 0.95)
	if msg == nil {
		t.Fatal("Expected non-nil message")
	}
	if msg.ThreatID != "THREAT-001" {
		t.Errorf("Expected THREAT-001, got %s", msg.ThreatID)
	}
	if msg.RequiredAction != ActionDefend {
		t.Errorf("Expected ActionDefend (confidence > 0.95), got %d", msg.RequiredAction)
	}
}

// TestIMMINENTFormatterLowConfidence tests IMMINENT with low confidence
func TestIMMINENTFormatterLowConfidence(t *testing.T) {
	formatter := NewIMMINENTFormatter("SECRET")

	launchPos := Position{Latitude: 45.0, Longitude: -120.0, Altitude: 0}
	impactPos := Position{Latitude: 35.0, Longitude: -100.0, Altitude: 0}

	msg := formatter.Format("THREAT-001", launchPos, impactPos,
		time.Now(), time.Now().Add(30*time.Minute), 0.5)
	if msg == nil {
		t.Fatal("Expected non-nil message")
	}
	if msg.RequiredAction != ActionMonitor {
		t.Errorf("Expected ActionMonitor (confidence < 0.7), got %d", msg.RequiredAction)
	}
}

// TestINCOMINGFormatter tests INCOMING message formatting
func TestINCOMINGFormatter(t *testing.T) {
	formatter := NewINCOMINGFormatter("SECRET")

	currentPos := Position{Latitude: 40.0, Longitude: -110.0, Altitude: 50000}
	velocity := Velocity{Vx: 5000, Vy: 1000, Vz: -2000}
	impactPos := Position{Latitude: 35.0, Longitude: -100.0, Altitude: 0}

	msg := formatter.Format("THREAT-001", "IRBM", currentPos, velocity, impactPos,
		10*time.Minute, 0.95)
	if msg == nil {
		t.Fatal("Expected non-nil message")
	}
	if msg.ThreatID != "THREAT-001" {
		t.Errorf("Expected THREAT-001, got %s", msg.ThreatID)
	}
	if msg.ThreatType != "IRBM" {
		t.Errorf("Expected IRBM, got %s", msg.ThreatType)
	}
	if msg.RequiredAction != ActionDefend {
		t.Errorf("Expected ActionDefend, got %d", msg.RequiredAction)
	}
}

// TestINCOMINGFormatterShortTime tests INCOMING with short time to impact
func TestINCOMINGFormatterShortTime(t *testing.T) {
	formatter := NewINCOMINGFormatter("SECRET")

	currentPos := Position{Latitude: 40.0, Longitude: -110.0, Altitude: 50000}
	velocity := Velocity{Vx: 5000, Vy: 1000, Vz: -2000}
	impactPos := Position{Latitude: 35.0, Longitude: -100.0, Altitude: 0}

	msg := formatter.Format("THREAT-001", "SRBM", currentPos, velocity, impactPos,
		3*time.Minute, 0.95)
	if msg == nil {
		t.Fatal("Expected non-nil message")
	}
	if msg.RequiredAction != ActionEvacuate {
		t.Errorf("Expected ActionEvacuate (time < 5min), got %d", msg.RequiredAction)
	}
}

// TestToText tests text formatting
func TestToText(t *testing.T) {
	formatter := NewCONOPREPFormatter("SECRET", "OPIR-DEMO", "NCA-001")

	track := &TrackSummary{
		TrackNumber: "T-001",
		CurrentPosition: Position{
			Latitude:  45.0,
			Longitude: -120.0,
			Altitude:  10000.0,
		},
		Confidence: 0.95,
	}

	msg := formatter.FormatLaunchAlert("45N 120W", "12:00:00Z", track)
	text := msg.ToText()

	if text == "" {
		t.Error("Expected non-empty text")
	}

	// Check for key fields
	if !contains(text, "MSGID:") {
		t.Error("Expected MSGID in text")
	}
	if !contains(text, "CLASS: SECRET") {
		t.Error("Expected CLASS: SECRET in text")
	}
	if !contains(text, "PRIORITY: FLASH") {
		t.Error("Expected PRIORITY: FLASH in text")
	}
	if !contains(text, "TRACK: T-001") {
		t.Error("Expected TRACK: T-001 in text")
	}
}

// TestIMMINENTToText tests IMMINENT text formatting
func TestIMMINENTToText(t *testing.T) {
	formatter := NewIMMINENTFormatter("SECRET")

	launchPos := Position{Latitude: 45.0, Longitude: -120.0, Altitude: 0}
	impactPos := Position{Latitude: 35.0, Longitude: -100.0, Altitude: 0}

	msg := formatter.Format("THREAT-001", launchPos, impactPos,
		time.Now(), time.Now().Add(30*time.Minute), 0.95)
	text := msg.ToText()

	if text == "" {
		t.Error("Expected non-empty text")
	}

	if !contains(text, "TYPE: IMMINENT") {
		t.Error("Expected TYPE: IMMINENT in text")
	}
	if !contains(text, "THREAT: THREAT-001") {
		t.Error("Expected THREAT: THREAT-001 in text")
	}
}

// TestINCOMINGToText tests INCOMING text formatting
func TestINCOMINGToText(t *testing.T) {
	formatter := NewINCOMINGFormatter("SECRET")

	currentPos := Position{Latitude: 40.0, Longitude: -110.0, Altitude: 50000}
	velocity := Velocity{Vx: 5000, Vy: 1000, Vz: -2000}
	impactPos := Position{Latitude: 35.0, Longitude: -100.0, Altitude: 0}

	msg := formatter.Format("THREAT-001", "IRBM", currentPos, velocity, impactPos,
		10*time.Minute, 0.95)
	text := msg.ToText()

	if text == "" {
		t.Error("Expected non-empty text")
	}

	if !contains(text, "TYPE: INCOMING") {
		t.Error("Expected TYPE: INCOMING in text")
	}
	if !contains(text, "THREAT: THREAT-001") {
		t.Error("Expected THREAT: THREAT-001 in text")
	}
	if !contains(text, "IRBM") {
		t.Error("Expected IRBM in text")
	}
}

// TestPriorityStrings tests priority string conversion
func TestPriorityStrings(t *testing.T) {
	tests := []struct {
		priority AlertPriority
		expected string
	}{
		{PriorityRoutine, "ROUTINE"},
		{PriorityPriority, "PRIORITY"},
		{PriorityFlash, "FLASH"},
		{PriorityFlashOverride, "FLASH OVERRIDE"},
	}

	for _, tt := range tests {
		result := getPriorityString(tt.priority)
		if result != tt.expected {
			t.Errorf("getPriorityString(%d) = %s, want %s", tt.priority, result, tt.expected)
		}
	}
}

// TestCategoryStrings tests category string conversion
func TestCategoryStrings(t *testing.T) {
	tests := []struct {
		category AlertCategory
		expected string
	}{
		{CategoryThreat, "THREAT"},
		{CategoryLaunch, "LAUNCH"},
		{CategoryImpact, "IMPACT"},
		{CategoryDefense, "DEFENSE"},
		{CategoryExercise, "EXERCISE"},
	}

	for _, tt := range tests {
		result := getCategoryString(tt.category)
		if result != tt.expected {
			t.Errorf("getCategoryString(%d) = %s, want %s", tt.category, result, tt.expected)
		}
	}
}

// TestActionStrings tests action string conversion
func TestActionStrings(t *testing.T) {
	tests := []struct {
		action   AlertAction
		expected string
	}{
		{ActionNone, "NONE"},
		{ActionMonitor, "MONITOR"},
		{ActionPrepare, "PREPARE"},
		{ActionDefend, "DEFEND"},
		{ActionEvacuate, "EVACUATE"},
	}

	for _, tt := range tests {
		result := getActionString(tt.action)
		if result != tt.expected {
			t.Errorf("getActionString(%d) = %s, want %s", tt.action, result, tt.expected)
		}
	}
}

// TestFormatPosition tests position formatting
func TestFormatPosition(t *testing.T) {
	tests := []struct {
		pos      Position
		contains string
	}{
		{Position{Latitude: 45.5, Longitude: -120.25, Altitude: 10000}, "45.5000"},
		{Position{Latitude: -35.5, Longitude: 100.0, Altitude: 0}, "35.5000S"},
		{Position{Latitude: 0, Longitude: 0, Altitude: 500}, "0m"},
	}

	for _, tt := range tests {
		result := formatPosition(tt.pos)
		if !contains(result, tt.contains) {
			t.Errorf("formatPosition(%v) = %s, want to contain %s", tt.pos, result, tt.contains)
		}
	}
}

// BenchmarkCONOPREPFormat benchmarks CONOPREP formatting
func BenchmarkCONOPREPFormat(b *testing.B) {
	formatter := NewCONOPREPFormatter("SECRET", "OPIR-DEMO", "NCA-001")

	track := &TrackSummary{
		TrackNumber: "T-001",
		CurrentPosition: Position{
			Latitude:  45.0,
			Longitude: -120.0,
			Altitude:  10000.0,
		},
		Confidence: 0.95,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatter.FormatLaunchAlert("45N 120W", "12:00:00Z", track)
	}
}

// BenchmarkIMMINENTFormat benchmarks IMMINENT formatting
func BenchmarkIMMINENTFormat(b *testing.B) {
	formatter := NewIMMINENTFormatter("SECRET")

	launchPos := Position{Latitude: 45.0, Longitude: -120.0, Altitude: 0}
	impactPos := Position{Latitude: 35.0, Longitude: -100.0, Altitude: 0}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatter.Format("THREAT-001", launchPos, impactPos,
			time.Now(), time.Now().Add(30*time.Minute), 0.95)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(substr)] == substr ||
		(len(s) > len(substr) && contains(s[1:], substr))
}
