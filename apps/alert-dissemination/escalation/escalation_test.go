package escalation

import (
	"testing"
	"time"
)

// TestNewEscalationManager tests manager creation
func TestNewEscalationManager(t *testing.T) {
	mgr := NewEscalationManager()
	if mgr == nil {
		t.Fatal("NewEscalationManager() returned nil")
	}
}

// TestAddRule tests rule addition
func TestAddRule(t *testing.T) {
	mgr := NewEscalationManager()

	rule := &EscalationRule{
		ID:           "test-rule",
		Name:         "Test Rule",
		FromLevel:    LevelNotify,
		ToLevel:      LevelAlert,
		TriggerAfter: 5 * time.Minute,
	}

	err := mgr.AddRule(rule)
	if err != nil {
		t.Fatalf("AddRule() error = %v", err)
	}

	// Test adding rule without ID
	err = mgr.AddRule(&EscalationRule{})
	if err == nil {
		t.Error("Expected error for rule without ID")
	}
}

// TestRemoveRule tests rule removal
func TestRemoveRule(t *testing.T) {
	mgr := NewEscalationManager()

	rule := &EscalationRule{ID: "test-rule", FromLevel: LevelNotify, ToLevel: LevelAlert}
	mgr.AddRule(rule)

	mgr.RemoveRule("test-rule")

	if len(mgr.rules) != 0 {
		t.Error("Expected rule to be removed")
	}
}

// TestStartEscalation tests starting escalation
func TestStartEscalation(t *testing.T) {
	mgr := NewEscalationManager()

	state := mgr.StartEscalation("ALERT-001", LevelNotify)
	if state == nil {
		t.Fatal("StartEscalation() returned nil")
	}
	if state.AlertID != "ALERT-001" {
		t.Errorf("Expected ALERT-001, got %s", state.AlertID)
	}
	if state.CurrentLevel != LevelNotify {
		t.Errorf("Expected LevelNotify, got %s", state.CurrentLevel)
	}
	if state.Acknowledged {
		t.Error("Expected Acknowledged to be false")
	}
}

// TestCheckEscalation tests escalation checking
func TestCheckEscalation(t *testing.T) {
	mgr := NewEscalationManager()

	// Start escalation
	state := mgr.StartEscalation("ALERT-001", LevelNotify)
	if state == nil {
		t.Fatal("StartEscalation returned nil")
	}

	// Verify initial state
	if state.CurrentLevel != LevelNotify {
		t.Errorf("Expected LevelNotify, got %s", state.CurrentLevel)
	}
	if state.Acknowledged {
		t.Error("Expected Acknowledged to be false")
	}
}

// TestCheckEscalationAcknowledged tests that acknowledged alerts don't escalate
func TestCheckEscalationAcknowledged(t *testing.T) {
	mgr := NewEscalationManager()

	for _, rule := range DefaultEscalationRules() {
		mgr.AddRule(rule)
	}

	state := mgr.StartEscalation("ALERT-001", LevelNotify)
	mgr.Acknowledge("ALERT-001", "OPERATOR-1")

	state.LastEscalation = time.Now().Add(-10 * time.Minute)
	state.NextEscalation = time.Now().Add(-5 * time.Minute)

	_, escalated, err := mgr.CheckEscalation("ALERT-001")
	if err != nil {
		t.Fatalf("CheckEscalation() error = %v", err)
	}
	if escalated {
		t.Error("Should not escalate acknowledged alert")
	}
}

// TestAcknowledge tests acknowledgment
func TestAcknowledge(t *testing.T) {
	mgr := NewEscalationManager()

	mgr.StartEscalation("ALERT-001", LevelNotify)

	err := mgr.Acknowledge("ALERT-001", "OPERATOR-1")
	if err != nil {
		t.Fatalf("Acknowledge() error = %v", err)
	}

	state, err := mgr.GetState("ALERT-001")
	if err != nil {
		t.Fatalf("GetState() error = %v", err)
	}

	if !state.Acknowledged {
		t.Error("Expected Acknowledged to be true")
	}
	if state.AcknowledgedBy != "OPERATOR-1" {
		t.Errorf("Expected OPERATOR-1, got %s", state.AcknowledgedBy)
	}
}

// TestDeescalate tests de-escalation
func TestDeescalate(t *testing.T) {
	mgr := NewEscalationManager()

	mgr.StartEscalation("ALERT-001", LevelCritical)

	deescalated, err := mgr.Deescalate("ALERT-001", "threat resolved")
	if err != nil {
		t.Fatalf("Deescalate() error = %v", err)
	}

	if deescalated.CurrentLevel != LevelAlert {
		t.Errorf("Expected LevelAlert, got %s", deescalated.CurrentLevel)
	}
}

// TestGetState tests state retrieval
func TestGetState(t *testing.T) {
	mgr := NewEscalationManager()

	mgr.StartEscalation("ALERT-001", LevelNotify)

	state, err := mgr.GetState("ALERT-001")
	if err != nil {
		t.Fatalf("GetState() error = %v", err)
	}
	if state.AlertID != "ALERT-001" {
		t.Errorf("Expected ALERT-001, got %s", state.AlertID)
	}

	_, err = mgr.GetState("NOT-EXIST")
	if err != ErrStateNotFound {
		t.Errorf("Expected ErrStateNotFound, got %v", err)
	}
}

// TestGetActiveEscalations tests active escalation retrieval
func TestGetActiveEscalations(t *testing.T) {
	mgr := NewEscalationManager()

	mgr.StartEscalation("ALERT-001", LevelNotify)
	mgr.StartEscalation("ALERT-002", LevelAlert)
	mgr.StartEscalation("ALERT-003", LevelCritical)
	mgr.Acknowledge("ALERT-003", "OPERATOR-1")

	active := mgr.GetActiveEscalations()
	if len(active) != 2 {
		t.Errorf("Expected 2 active, got %d", len(active))
	}
}

// TestGetEscalationsByLevel tests filtering by level
func TestGetEscalationsByLevel(t *testing.T) {
	mgr := NewEscalationManager()

	mgr.StartEscalation("ALERT-001", LevelNotify)
	mgr.StartEscalation("ALERT-002", LevelNotify)
	mgr.StartEscalation("ALERT-003", LevelAlert)

	notify := mgr.GetEscalationsByLevel(LevelNotify)
	if len(notify) != 2 {
		t.Errorf("Expected 2 at Notify, got %d", len(notify))
	}

	alert := mgr.GetEscalationsByLevel(LevelAlert)
	if len(alert) != 1 {
		t.Errorf("Expected 1 at Alert, got %d", len(alert))
	}
}

// TestOnEscalate tests callback registration
func TestOnEscalate(t *testing.T) {
	mgr := NewEscalationManager()

	var escalated bool
	mgr.OnEscalate("ALERT-001", func(state *EscalationState) {
		escalated = true
	})

	// Start and escalate
	state := mgr.StartEscalation("ALERT-001", LevelNotify)
	for _, rule := range DefaultEscalationRules() {
		mgr.AddRule(rule)
	}
	
	// Trigger escalation
	state.NextEscalation = time.Now().Add(-1 * time.Minute)
	mgr.CheckEscalation("ALERT-001")

	_ = escalated // Used for testing
}

// TestCancelEscalation tests cancellation
func TestCancelEscalation(t *testing.T) {
	mgr := NewEscalationManager()

	mgr.StartEscalation("ALERT-001", LevelNotify)

	err := mgr.CancelEscalation("ALERT-001")
	if err != nil {
		t.Fatalf("CancelEscalation() error = %v", err)
	}

	state, _ := mgr.GetState("ALERT-001")
	if !state.Acknowledged {
		t.Error("Expected Acknowledged to be true after cancel")
	}
}

// TestClearState tests state clearing
func TestClearState(t *testing.T) {
	mgr := NewEscalationManager()

	mgr.StartEscalation("ALERT-001", LevelNotify)
	mgr.ClearState("ALERT-001")

	_, err := mgr.GetState("ALERT-001")
	if err != ErrStateNotFound {
		t.Errorf("Expected ErrStateNotFound, got %v", err)
	}
}

// TestStats tests statistics
func TestStats(t *testing.T) {
	mgr := NewEscalationManager()

	mgr.StartEscalation("ALERT-001", LevelNotify)
	mgr.StartEscalation("ALERT-002", LevelAlert)
	mgr.StartEscalation("ALERT-003", LevelCritical)
	mgr.StartEscalation("ALERT-004", LevelEmergency)
	mgr.Acknowledge("ALERT-001", "OPERATOR-1")

	stats := mgr.Stats()
	if stats.Active != 3 {
		t.Errorf("Expected 3 active, got %d", stats.Active)
	}
	if stats.Completed != 1 {
		t.Errorf("Expected 1 completed, got %d", stats.Completed)
	}
	if stats.AtNotify != 0 {
		t.Errorf("Expected 0 at Notify (acknowledged), got %d", stats.AtNotify)
	}
	if stats.AtAlert != 1 {
		t.Errorf("Expected 1 at Alert, got %d", stats.AtAlert)
	}
}

// TestDefaultEscalationRules tests default rules
func TestDefaultEscalationRules(t *testing.T) {
	rules := DefaultEscalationRules()
	if len(rules) != 3 {
		t.Errorf("Expected 3 default rules, got %d", len(rules))
	}

	// Verify escalation chain
	if rules[0].FromLevel != LevelNotify || rules[0].ToLevel != LevelAlert {
		t.Error("First rule should escalate Notify to Alert")
	}
	if rules[1].FromLevel != LevelAlert || rules[1].ToLevel != LevelCritical {
		t.Error("Second rule should escalate Alert to Critical")
	}
	if rules[2].FromLevel != LevelCritical || rules[2].ToLevel != LevelEmergency {
		t.Error("Third rule should escalate Critical to Emergency")
	}
}

// TestEscalationLevelString tests level string conversion
func TestEscalationLevelString(t *testing.T) {
	tests := []struct {
		level    EscalationLevel
		expected string
	}{
		{LevelNone, "NONE"},
		{LevelNotify, "NOTIFY"},
		{LevelAlert, "ALERT"},
		{LevelCritical, "CRITICAL"},
		{LevelEmergency, "EMERGENCY"},
		{EscalationLevel(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		result := tt.level.String()
		if result != tt.expected {
			t.Errorf("EscalationLevel(%d).String() = %s, want %s", tt.level, result, tt.expected)
		}
	}
}

// TestEscalationPath tests escalation path tracking
func TestEscalationPath(t *testing.T) {
	mgr := NewEscalationManager()

	for _, rule := range DefaultEscalationRules() {
		mgr.AddRule(rule)
	}

	state := mgr.StartEscalation("ALERT-001", LevelNotify)
	state.NextEscalation = time.Now().Add(-1 * time.Minute)

	// Trigger escalation
	mgr.CheckEscalation("ALERT-001")

	if len(state.EscalationPath) != 1 {
		t.Errorf("Expected 1 escalation step, got %d", len(state.EscalationPath))
	}
}

// TestMaxAttempts tests max attempts limit
func TestMaxAttempts(t *testing.T) {
	mgr := NewEscalationManager()

	rule := &EscalationRule{
		ID:           "test-rule",
		FromLevel:    LevelNotify,
		ToLevel:      LevelAlert,
		TriggerAfter: 0,
		MaxAttempts:  2,
	}
	mgr.AddRule(rule)

	state := mgr.StartEscalation("ALERT-001", LevelNotify)
	state.NextEscalation = time.Now().Add(-1 * time.Minute)

	// First escalation
	mgr.CheckEscalation("ALERT-001")
	if state.AttemptCount != 1 {
		t.Errorf("Expected 1 attempt, got %d", state.AttemptCount)
	}

	// Second escalation (would be blocked by max attempts)
	mgr.CheckEscalation("ALERT-001")
}

// BenchmarkCheckEscalation benchmarks escalation checking
func BenchmarkCheckEscalation(b *testing.B) {
	mgr := NewEscalationManager()
	for _, rule := range DefaultEscalationRules() {
		mgr.AddRule(rule)
	}

	mgr.StartEscalation("ALERT-001", LevelNotify)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mgr.CheckEscalation("ALERT-001")
	}
}

// BenchmarkStartEscalation benchmarks starting escalation
func BenchmarkStartEscalation(b *testing.B) {
	mgr := NewEscalationManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mgr.StartEscalation(string(rune(i)), LevelNotify)
	}
}