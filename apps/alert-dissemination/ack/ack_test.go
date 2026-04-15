package ack

import (
	"context"
	"testing"
	"time"
)

// TestNewAckHandler tests handler creation
func TestNewAckHandler(t *testing.T) {
	handler := NewAckHandler(30 * time.Second)
	if handler == nil {
		t.Fatal("NewAckHandler() returned nil")
	}
	if handler.timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", handler.timeout)
	}
}

// TestRegister tests acknowledgment registration
func TestRegister(t *testing.T) {
	handler := NewAckHandler(30 * time.Second)

	ack := handler.Register("ALERT-001", "RECIPIENT-A")
	if ack == nil {
		t.Fatal("Register() returned nil")
	}
	if ack.AlertID != "ALERT-001" {
		t.Errorf("Expected ALERT-001, got %s", ack.AlertID)
	}
	if ack.Recipient != "RECIPIENT-A" {
		t.Errorf("Expected RECIPIENT-A, got %s", ack.Recipient)
	}
	if ack.Status != AckStatusPending {
		t.Errorf("Expected PENDING, got %s", ack.Status)
	}
}

// TestAcknowledge tests acknowledgment processing
func TestAcknowledge(t *testing.T) {
	handler := NewAckHandler(30 * time.Second)

	handler.Register("ALERT-001", "RECIPIENT-A")

	err := handler.Acknowledge("ALERT-001", "RECIPIENT-A", "OPERATOR-1")
	if err != nil {
		t.Fatalf("Acknowledge() error = %v", err)
	}

	// Verify no longer pending
	_, err = handler.Get("ALERT-001", "RECIPIENT-A")
	if err == nil {
		t.Error("Expected error for acknowledged alert")
	}
}

// TestAcknowledgeNotFound tests acknowledgment of non-existent alert
func TestAcknowledgeNotFound(t *testing.T) {
	handler := NewAckHandler(30 * time.Second)

	err := handler.Acknowledge("NOT-EXIST", "RECIPIENT-A", "OPERATOR-1")
	if err != ErrAckNotFound {
		t.Errorf("Expected ErrAckNotFound, got %v", err)
	}
}

// TestReject tests rejection processing
func TestReject(t *testing.T) {
	handler := NewAckHandler(30 * time.Second)

	handler.Register("ALERT-001", "RECIPIENT-A")

	err := handler.Reject("ALERT-001", "RECIPIENT-A", "Invalid target")
	if err != nil {
		t.Fatalf("Reject() error = %v", err)
	}

	// Verify no longer pending
	_, err = handler.Get("ALERT-001", "RECIPIENT-A")
	if err == nil {
		t.Error("Expected error for rejected alert")
	}
}

// TestTimeout tests timeout handling
func TestTimeout(t *testing.T) {
	handler := NewAckHandler(100 * time.Millisecond)

	handler.Register("ALERT-001", "RECIPIENT-A")

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	timedOut := handler.Timeout()
	if len(timedOut) != 1 {
		t.Errorf("Expected 1 timeout, got %d", len(timedOut))
	}
	if timedOut[0].Status != AckStatusTimeout {
		t.Errorf("Expected TIMEOUT status, got %s", timedOut[0].Status)
	}
}

// TestIncrementAttempts tests attempt incrementing
func TestIncrementAttempts(t *testing.T) {
	handler := NewAckHandler(30 * time.Second)

	ack := handler.Register("ALERT-001", "RECIPIENT-A")
	ack.MaxAttempts = 2

	err := handler.IncrementAttempts("ALERT-001", "RECIPIENT-A")
	if err != nil {
		t.Fatalf("IncrementAttempts() error = %v", err)
	}

	if ack.Attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", ack.Attempts)
	}

	// Second increment should exceed max
	err = handler.IncrementAttempts("ALERT-001", "RECIPIENT-A")
	if err != ErrMaxAttemptsExceeded {
		t.Errorf("Expected ErrMaxAttemptsExceeded, got %v", err)
	}
}

// TestGet tests retrieval by ID
func TestGet(t *testing.T) {
	handler := NewAckHandler(30 * time.Second)

	handler.Register("ALERT-001", "RECIPIENT-A")

	ack, err := handler.Get("ALERT-001", "RECIPIENT-A")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if ack.AlertID != "ALERT-001" {
		t.Errorf("Expected ALERT-001, got %s", ack.AlertID)
	}

	_, err = handler.Get("NOT-EXIST", "RECIPIENT-A")
	if err != ErrAckNotFound {
		t.Errorf("Expected ErrAckNotFound, got %v", err)
	}
}

// TestListPending tests listing pending acknowledgments
func TestListPending(t *testing.T) {
	handler := NewAckHandler(30 * time.Second)

	handler.Register("ALERT-001", "RECIPIENT-A")
	handler.Register("ALERT-002", "RECIPIENT-B")
	handler.Register("ALERT-003", "RECIPIENT-C")

	pending := handler.ListPending()
	if len(pending) != 3 {
		t.Errorf("Expected 3 pending, got %d", len(pending))
	}

	// Acknowledge one
	handler.Acknowledge("ALERT-001", "RECIPIENT-A", "OPERATOR-1")

	pending = handler.ListPending()
	if len(pending) != 2 {
		t.Errorf("Expected 2 pending after ack, got %d", len(pending))
	}
}

// TestListByAlert tests listing by alert ID
func TestListByAlert(t *testing.T) {
	handler := NewAckHandler(30 * time.Second)

	handler.Register("ALERT-001", "RECIPIENT-A")
	handler.Register("ALERT-001", "RECIPIENT-B")
	handler.Register("ALERT-002", "RECIPIENT-C")

	list := handler.ListByAlert("ALERT-001")
	if len(list) != 2 {
		t.Errorf("Expected 2 for ALERT-001, got %d", len(list))
	}
}

// TestCallbacks tests callback registration
func TestCallbacks(t *testing.T) {
	handler := NewAckHandler(30 * time.Second)

	var ackCalled bool
	var nackCalled bool
	var timeoutCalled bool

	handler.OnAck("ALERT-001", func(ack *Acknowledgment) {
		ackCalled = true
	})
	handler.OnNack("ALERT-002", func(ack *Acknowledgment) {
		nackCalled = true
	})
	handler.OnTimeout("ALERT-003", func(ack *Acknowledgment) {
		timeoutCalled = true
	})

	// Test ACK callback
	handler.Register("ALERT-001", "RECIPIENT-A")
	handler.Acknowledge("ALERT-001", "RECIPIENT-A", "OPERATOR-1")

	if !ackCalled {
		t.Error("Expected ack callback to be called")
	}

	// Test NACK callback
	handler.Register("ALERT-002", "RECIPIENT-B")
	handler.Reject("ALERT-002", "RECIPIENT-B", "Invalid")

	if !nackCalled {
		t.Error("Expected nack callback to be called")
	}

	// Test timeout callback
	handler.timeout = 100 * time.Millisecond
	handler.Register("ALERT-003", "RECIPIENT-C")
	time.Sleep(150 * time.Millisecond)
	handler.Timeout()

	if !timeoutCalled {
		t.Error("Expected timeout callback to be called")
	}
}

// TestStats tests statistics
func TestStats(t *testing.T) {
	handler := NewAckHandler(30 * time.Second)

	handler.Register("ALERT-001", "RECIPIENT-A")
	handler.Register("ALERT-001", "RECIPIENT-B")
	handler.Register("ALERT-002", "RECIPIENT-C")

	stats := handler.Stats()
	if stats.Pending != 3 {
		t.Errorf("Expected 3 pending, got %d", stats.Pending)
	}

	handler.Acknowledge("ALERT-001", "RECIPIENT-A", "OPERATOR-1")

	stats = handler.Stats()
	if stats.Pending != 2 {
		t.Errorf("Expected 2 pending after ack, got %d", stats.Pending)
	}
}

// TestAckTracker tests tracker operations
func TestAckTracker(t *testing.T) {
	tracker := NewAckTracker(30 * time.Second)

	// Register
	ack := tracker.Register("ALERT-001", "RECIPIENT-A")
	if ack == nil {
		t.Fatal("Register() returned nil")
	}

	// Acknowledge
	err := tracker.Acknowledge("ALERT-001", "RECIPIENT-A", "OPERATOR-1")
	if err != nil {
		t.Fatalf("Acknowledge() error = %v", err)
	}

	// Check complete
	if !tracker.IsComplete("ALERT-001") {
		t.Error("Expected IsComplete to be true")
	}
}

// TestAckTrackerReject tests tracker rejection
func TestAckTrackerReject(t *testing.T) {
	tracker := NewAckTracker(30 * time.Second)

	tracker.Register("ALERT-001", "RECIPIENT-A")

	err := tracker.Reject("ALERT-001", "RECIPIENT-A", "Invalid")
	if err != nil {
		t.Fatalf("Reject() error = %v", err)
	}

	if !tracker.IsComplete("ALERT-001") {
		t.Error("Expected IsComplete to be true after reject")
	}
}

// TestWaitForAll tests waiting for all acknowledgments
func TestWaitForAll(t *testing.T) {
	tracker := NewAckTracker(30 * time.Second)

	tracker.Register("ALERT-001", "RECIPIENT-A")

	// Start wait in goroutine
	done := make(chan bool)
	go func() {
		ctx := context.Background()
		err := tracker.WaitForAll(ctx, "ALERT-001")
		if err != nil {
			t.Errorf("WaitForAll() error = %v", err)
		}
		done <- true
	}()

	// Acknowledge after short delay
	time.Sleep(100 * time.Millisecond)
	tracker.Acknowledge("ALERT-001", "RECIPIENT-A", "OPERATOR-1")

	// Wait for completion
	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Error("WaitForAll() timeout")
	}
}

// TestAckStatus tests status string conversion
func TestAckStatus(t *testing.T) {
	tests := []struct {
		status   AckStatus
		expected string
	}{
		{AckStatusPending, "PENDING"},
		{AckStatusAcknowledged, "ACKNOWLEDGED"},
		{AckStatusRejected, "REJECTED"},
		{AckStatusTimeout, "TIMEOUT"},
		{AckStatusFailed, "FAILED"},
		{AckStatus(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		result := tt.status.String()
		if result != tt.expected {
			t.Errorf("AckStatus(%d).String() = %s, want %s", tt.status, result, tt.expected)
		}
	}
}

// TestAckTrackerStats tests tracker statistics
func TestAckTrackerStats(t *testing.T) {
	tracker := NewAckTracker(30 * time.Second)

	tracker.Register("ALERT-001", "RECIPIENT-A")
	tracker.Register("ALERT-001", "RECIPIENT-B")
	tracker.Register("ALERT-002", "RECIPIENT-C")

	stats := tracker.Stats()
	if stats.Pending != 3 {
		t.Errorf("Expected 3 pending, got %d", stats.Pending)
	}

	tracker.Acknowledge("ALERT-001", "RECIPIENT-A", "OPERATOR-1")
	tracker.Reject("ALERT-001", "RECIPIENT-B", "Invalid")

	stats = tracker.Stats()
	if stats.Pending != 1 {
		t.Errorf("Expected 1 pending, got %d", stats.Pending)
	}
}

// BenchmarkRegister benchmarks registration
func BenchmarkRegister(b *testing.B) {
	handler := NewAckHandler(30 * time.Second)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.Register(string(rune(i)), "RECIPIENT")
	}
}

// BenchmarkAcknowledge benchmarks acknowledgment
func BenchmarkAcknowledge(b *testing.B) {
	handler := NewAckHandler(30 * time.Second)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := string(rune(i))
		handler.Register(id, "RECIPIENT")
		handler.Acknowledge(id, "RECIPIENT", "OPERATOR")
	}
}
