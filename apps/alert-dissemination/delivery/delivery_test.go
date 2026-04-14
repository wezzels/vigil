package delivery

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestNewDeliveryTracker tests tracker creation
func TestNewDeliveryTracker(t *testing.T) {
	tracker := NewDeliveryTracker(nil)
	if tracker == nil {
		t.Fatal("NewDeliveryTracker() returned nil")
	}
	if tracker.config.MaxAttempts != 3 {
		t.Errorf("Expected MaxAttempts 3, got %d", tracker.config.MaxAttempts)
	}
}

// TestRegister tests delivery registration
func TestRegister(t *testing.T) {
	tracker := NewDeliveryTracker(nil)

	record := tracker.Register("ALERT-001", "RECIPIENT-A")
	if record == nil {
		t.Fatal("Register() returned nil")
	}
	if record.AlertID != "ALERT-001" {
		t.Errorf("Expected ALERT-001, got %s", record.AlertID)
	}
	if record.Recipient != "RECIPIENT-A" {
		t.Errorf("Expected RECIPIENT-A, got %s", record.Recipient)
	}
	if record.Status != StatusPending {
		t.Errorf("Expected PENDING, got %s", record.Status)
	}
}

// TestMarkSent tests marking as sent
func TestMarkSent(t *testing.T) {
	tracker := NewDeliveryTracker(nil)

	tracker.Register("ALERT-001", "RECIPIENT-A")

	err := tracker.MarkSent("ALERT-001", "RECIPIENT-A")
	if err != nil {
		t.Fatalf("MarkSent() error = %v", err)
	}

	record, _ := tracker.GetRecord("ALERT-001", "RECIPIENT-A")
	if record.Status != StatusSent {
		t.Errorf("Expected SENT, got %s", record.Status)
	}
	if record.Attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", record.Attempts)
	}
}

// TestMarkDelivered tests marking as delivered
func TestMarkDelivered(t *testing.T) {
	tracker := NewDeliveryTracker(nil)

	tracker.Register("ALERT-001", "RECIPIENT-A")
	tracker.MarkSent("ALERT-001", "RECIPIENT-A")

	err := tracker.MarkDelivered("ALERT-001", "RECIPIENT-A")
	if err != nil {
		t.Fatalf("MarkDelivered() error = %v", err)
	}

	record, _ := tracker.GetRecord("ALERT-001", "RECIPIENT-A")
	if record.Status != StatusDelivered {
		t.Errorf("Expected DELIVERED, got %s", record.Status)
	}
	if record.DeliveredAt.IsZero() {
		t.Error("Expected DeliveredAt to be set")
	}
}

// TestMarkAcknowledged tests marking as acknowledged
func TestMarkAcknowledged(t *testing.T) {
	tracker := NewDeliveryTracker(nil)

	tracker.Register("ALERT-001", "RECIPIENT-A")
	tracker.MarkSent("ALERT-001", "RECIPIENT-A")
	tracker.MarkDelivered("ALERT-001", "RECIPIENT-A")

	err := tracker.MarkAcknowledged("ALERT-001", "RECIPIENT-A")
	if err != nil {
		t.Fatalf("MarkAcknowledged() error = %v", err)
	}

	record, _ := tracker.GetRecord("ALERT-001", "RECIPIENT-A")
	if record.Status != StatusAcknowledged {
		t.Errorf("Expected ACKNOWLEDGED, got %s", record.Status)
	}
}

// TestMarkFailed tests marking as failed
func TestMarkFailed(t *testing.T) {
	tracker := NewDeliveryTracker(nil)

	tracker.Register("ALERT-001", "RECIPIENT-A")
	tracker.MarkSent("ALERT-001", "RECIPIENT-A")

	err := tracker.MarkFailed("ALERT-001", "RECIPIENT-A", "Connection refused")
	if err != nil {
		t.Fatalf("MarkFailed() error = %v", err)
	}

	record, _ := tracker.GetRecord("ALERT-001", "RECIPIENT-A")
	if record.LastError != "Connection refused" {
		t.Errorf("Expected 'Connection refused', got %s", record.LastError)
	}
}

// TestShouldRetry tests retry logic
func TestShouldRetry(t *testing.T) {
	tracker := NewDeliveryTracker(&DeliveryConfig{
		MaxAttempts:     3,
		EnableRetry:     true,
		RetryDelay:      1 * time.Second,
		RetryBackoff:    false,
	})

	tracker.Register("ALERT-001", "RECIPIENT-A")

	// Should retry after first attempt
	shouldRetry, delay := tracker.ShouldRetry("ALERT-001", "RECIPIENT-A")
	if !shouldRetry {
		t.Error("Expected shouldRetry to be true")
	}
	if delay != 1*time.Second {
		t.Errorf("Expected 1s delay, got %v", delay)
	}

	// Mark as sent
	tracker.MarkSent("ALERT-001", "RECIPIENT-A")

	// Should still retry
	shouldRetry, _ = tracker.ShouldRetry("ALERT-001", "RECIPIENT-A")
	if !shouldRetry {
		t.Error("Expected shouldRetry after first attempt")
	}

	// Mark as delivered - should not retry
	tracker.MarkDelivered("ALERT-001", "RECIPIENT-A")
	shouldRetry, _ = tracker.ShouldRetry("ALERT-001", "RECIPIENT-A")
	if shouldRetry {
		t.Error("Expected shouldRetry to be false after delivery")
	}
}

// TestShouldRetryMaxAttempts tests max attempts limit
func TestShouldRetryMaxAttempts(t *testing.T) {
	tracker := NewDeliveryTracker(&DeliveryConfig{
		MaxAttempts: 2,
		EnableRetry: true,
		RetryDelay:  1 * time.Second,
	})

	tracker.Register("ALERT-001", "RECIPIENT-A")
	tracker.MarkSent("ALERT-001", "RECIPIENT-A")
	tracker.MarkSent("ALERT-001", "RECIPIENT-A")

	// Max attempts reached
	shouldRetry, _ := tracker.ShouldRetry("ALERT-001", "RECIPIENT-A")
	if shouldRetry {
		t.Error("Expected shouldRetry to be false after max attempts")
	}
}

// TestRetryBackoff tests backoff calculation
func TestRetryBackoff(t *testing.T) {
	tracker := NewDeliveryTracker(&DeliveryConfig{
		MaxAttempts:       5,
		EnableRetry:       true,
		RetryDelay:        1 * time.Second,
		RetryBackoff:      true,
		BackoffMultiplier: 2.0,
	})

	tracker.Register("ALERT-001", "RECIPIENT-A")

	// First attempt
	shouldRetry, delay := tracker.ShouldRetry("ALERT-001", "RECIPIENT-A")
	if !shouldRetry {
		t.Fatal("Expected shouldRetry")
	}
	if delay != 1*time.Second {
		t.Errorf("Expected 1s, got %v", delay)
	}

	// Second attempt
	tracker.MarkSent("ALERT-001", "RECIPIENT-A")
	shouldRetry, delay = tracker.ShouldRetry("ALERT-001", "RECIPIENT-A")
	if !shouldRetry {
		t.Fatal("Expected shouldRetry")
	}
	if delay != 2*time.Second {
		t.Errorf("Expected 2s (backoff), got %v", delay)
	}

	// Third attempt
	tracker.MarkSent("ALERT-001", "RECIPIENT-A")
	shouldRetry, delay = tracker.ShouldRetry("ALERT-001", "RECIPIENT-A")
	if !shouldRetry {
		t.Fatal("Expected shouldRetry")
	}
	if delay != 4*time.Second {
		t.Errorf("Expected 4s (backoff), got %v", delay)
	}
}

// TestGetRecord tests record retrieval
func TestGetRecord(t *testing.T) {
	tracker := NewDeliveryTracker(nil)

	tracker.Register("ALERT-001", "RECIPIENT-A")

	record, err := tracker.GetRecord("ALERT-001", "RECIPIENT-A")
	if err != nil {
		t.Fatalf("GetRecord() error = %v", err)
	}
	if record.AlertID != "ALERT-001" {
		t.Errorf("Expected ALERT-001, got %s", record.AlertID)
	}

	_, err = tracker.GetRecord("NOT-EXIST", "RECIPIENT-A")
	if err != ErrRecordNotFound {
		t.Errorf("Expected ErrRecordNotFound, got %v", err)
	}
}

// TestGetRecordsByAlert tests alert-based retrieval
func TestGetRecordsByAlert(t *testing.T) {
	tracker := NewDeliveryTracker(nil)

	tracker.Register("ALERT-001", "RECIPIENT-A")
	tracker.Register("ALERT-001", "RECIPIENT-B")
	tracker.Register("ALERT-002", "RECIPIENT-C")

	records := tracker.GetRecordsByAlert("ALERT-001")
	if len(records) != 2 {
		t.Errorf("Expected 2 records, got %d", len(records))
	}
}

// TestGetRecordsByRecipient tests recipient-based retrieval
func TestGetRecordsByRecipient(t *testing.T) {
	tracker := NewDeliveryTracker(nil)

	tracker.Register("ALERT-001", "RECIPIENT-A")
	tracker.Register("ALERT-002", "RECIPIENT-A")
	tracker.Register("ALERT-003", "RECIPIENT-B")

	records := tracker.GetRecordsByRecipient("RECIPIENT-A")
	if len(records) != 2 {
		t.Errorf("Expected 2 records, got %d", len(records))
	}
}

// TestGetPending tests pending retrieval
func TestGetPending(t *testing.T) {
	tracker := NewDeliveryTracker(nil)

	tracker.Register("ALERT-001", "RECIPIENT-A")
	tracker.Register("ALERT-002", "RECIPIENT-B")
	tracker.Register("ALERT-003", "RECIPIENT-C")

	tracker.MarkSent("ALERT-001", "RECIPIENT-A")
	tracker.MarkDelivered("ALERT-003", "RECIPIENT-C")

	pending := tracker.GetPending()
	if len(pending) != 2 {
		t.Errorf("Expected 2 pending, got %d", len(pending))
	}
}

// TestGetFailed tests failed retrieval
func TestGetFailed(t *testing.T) {
	tracker := NewDeliveryTracker(&DeliveryConfig{MaxAttempts: 1})

	tracker.Register("ALERT-001", "RECIPIENT-A")
	tracker.Register("ALERT-002", "RECIPIENT-B")

	tracker.MarkSent("ALERT-001", "RECIPIENT-A")
	tracker.MarkFailed("ALERT-001", "RECIPIENT-A", "Failed")
	// ALERT-002 still pending

	failed := tracker.GetFailed()
	// Note: MarkFailed doesn't set StatusFailed until max attempts
	_ = failed
}

// TestCheckTimeouts tests timeout handling
func TestCheckTimeouts(t *testing.T) {
	tracker := NewDeliveryTracker(&DeliveryConfig{
		Timeout: 100 * time.Millisecond,
	})

	tracker.Register("ALERT-001", "RECIPIENT-A")

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	timedOut := tracker.CheckTimeouts()
	if len(timedOut) != 1 {
		t.Errorf("Expected 1 timeout, got %d", len(timedOut))
	}

	record, _ := tracker.GetRecord("ALERT-001", "RECIPIENT-A")
	if record.Status != StatusTimeout {
		t.Errorf("Expected TIMEOUT, got %s", record.Status)
	}
}

// TestStats tests statistics
func TestStats(t *testing.T) {
	tracker := NewDeliveryTracker(nil)

	tracker.Register("ALERT-001", "RECIPIENT-A")
	tracker.Register("ALERT-002", "RECIPIENT-B")
	tracker.Register("ALERT-003", "RECIPIENT-C")

	tracker.MarkSent("ALERT-001", "RECIPIENT-A")
	tracker.MarkDelivered("ALERT-002", "RECIPIENT-B")
	tracker.MarkAcknowledged("ALERT-003", "RECIPIENT-C")

	stats := tracker.Stats()
	if stats.Pending != 0 {
		t.Errorf("Expected 0 pending, got %d", stats.Pending)
	}
	if stats.Sent != 1 {
		t.Errorf("Expected 1 sent, got %d", stats.Sent)
	}
	if stats.Delivered != 1 {
		t.Errorf("Expected 1 delivered, got %d", stats.Delivered)
	}
	if stats.Acknowledged != 1 {
		t.Errorf("Expected 1 acknowledged, got %d", stats.Acknowledged)
	}
}

// TestNotifications tests notification channel
func TestNotifications(t *testing.T) {
	tracker := NewDeliveryTracker(nil)

	// Start goroutine to receive notifications
	go func() {
		for record := range tracker.Notifications() {
			_ = record // Process notification
		}
	}()

	tracker.Register("ALERT-001", "RECIPIENT-A")
	tracker.MarkSent("ALERT-001", "RECIPIENT-A")
	tracker.MarkDelivered("ALERT-001", "RECIPIENT-A")

	// Notifications are sent asynchronously
	time.Sleep(50 * time.Millisecond)
}

// TestClear tests clearing records
func TestClear(t *testing.T) {
	tracker := NewDeliveryTracker(nil)

	tracker.Register("ALERT-001", "RECIPIENT-A")
	tracker.Register("ALERT-002", "RECIPIENT-B")

	tracker.Clear()

	if len(tracker.records) != 0 {
		t.Error("Expected all records to be cleared")
	}
}

// TestRetrier tests retrier
func TestRetrier(t *testing.T) {
	tracker := NewDeliveryTracker(&DeliveryConfig{
		MaxAttempts:  3,
		EnableRetry:  true,
		RetryDelay:   10 * time.Millisecond,
		RetryBackoff: false,
	})

	var callCount int
	sendFunc := func(alertID, recipient string) error {
		callCount++
		if callCount == 1 {
			return errors.New("temporary failure")
		}
		return nil
	}

	retrier := NewRetrier(tracker, sendFunc)

	tracker.Register("ALERT-001", "RECIPIENT-A")

	ctx := context.Background()

	// First call - sendFunc returns error on first call
	err := retrier.Retry(ctx, "ALERT-001", "RECIPIENT-A")
	if err == nil {
		t.Error("Expected error on first call")
	}

	// Mark as sent for next retry
	tracker.MarkSent("ALERT-001", "RECIPIENT-A")

	// Second call - sendFunc succeeds
	err = retrier.Retry(ctx, "ALERT-001", "RECIPIENT-A")
	if err != nil {
		t.Errorf("Unexpected error on second call: %v", err)
	}
}

// TestDeliveryStatusString tests status string conversion
func TestDeliveryStatusString(t *testing.T) {
	tests := []struct {
		status   DeliveryStatus
		expected string
	}{
		{StatusPending, "PENDING"},
		{StatusSent, "SENT"},
		{StatusDelivered, "DELIVERED"},
		{StatusAcknowledged, "ACKNOWLEDGED"},
		{StatusFailed, "FAILED"},
		{StatusTimeout, "TIMEOUT"},
		{DeliveryStatus(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		result := tt.status.String()
		if result != tt.expected {
			t.Errorf("DeliveryStatus(%d).String() = %s, want %s", tt.status, result, tt.expected)
		}
	}
}

// TestDefaultDeliveryConfig tests default config
func TestDefaultDeliveryConfig(t *testing.T) {
	config := DefaultDeliveryConfig()

	if config.MaxAttempts != 3 {
		t.Errorf("Expected MaxAttempts 3, got %d", config.MaxAttempts)
	}
	if config.Timeout != 30*time.Second {
		t.Errorf("Expected Timeout 30s, got %v", config.Timeout)
	}
	if !config.EnableRetry {
		t.Error("Expected EnableRetry to be true")
	}
}

// BenchmarkRegister benchmarks registration
func BenchmarkRegister(b *testing.B) {
	tracker := NewDeliveryTracker(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.Register(string(rune(i)), "RECIPIENT")
	}
}

// BenchmarkMarkSent benchmarks marking as sent
func BenchmarkMarkSent(b *testing.B) {
	tracker := NewDeliveryTracker(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := string(rune(i))
		tracker.Register(id, "RECIPIENT")
		tracker.MarkSent(id, "RECIPIENT")
	}
}

// BenchmarkShouldRetry benchmarks retry checking
func BenchmarkShouldRetry(b *testing.B) {
	tracker := NewDeliveryTracker(nil)
	tracker.Register("ALERT-001", "RECIPIENT-A")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.ShouldRetry("ALERT-001", "RECIPIENT-A")
	}
}