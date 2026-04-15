// Package delivery provides delivery tracking and confirmation for alert dissemination
package delivery

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DeliveryStatus represents the delivery status
type DeliveryStatus int

const (
	StatusPending DeliveryStatus = iota
	StatusSent
	StatusDelivered
	StatusAcknowledged
	StatusFailed
	StatusTimeout
)

// String returns string representation of status
func (s DeliveryStatus) String() string {
	switch s {
	case StatusPending:
		return "PENDING"
	case StatusSent:
		return "SENT"
	case StatusDelivered:
		return "DELIVERED"
	case StatusAcknowledged:
		return "ACKNOWLEDGED"
	case StatusFailed:
		return "FAILED"
	case StatusTimeout:
		return "TIMEOUT"
	default:
		return "UNKNOWN"
	}
}

// DeliveryRecord represents a delivery attempt
type DeliveryRecord struct {
	AlertID        string         `json:"alert_id"`
	Recipient      string         `json:"recipient"`
	Status         DeliveryStatus `json:"status"`
	Attempts       int            `json:"attempts"`
	MaxAttempts    int            `json:"max_attempts"`
	FirstAttempt   time.Time      `json:"first_attempt"`
	LastAttempt    time.Time      `json:"last_attempt"`
	DeliveredAt    time.Time      `json:"delivered_at,omitempty"`
	AcknowledgedAt time.Time      `json:"acknowledged_at,omitempty"`
	LastError      string         `json:"last_error,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// DeliveryConfig holds delivery configuration
type DeliveryConfig struct {
	MaxAttempts       int           `json:"max_attempts"`
	InitialDelay      time.Duration `json:"initial_delay"`
	RetryDelay        time.Duration `json:"retry_delay"`
	Timeout           time.Duration `json:"timeout"`
	EnableRetry       bool          `json:"enable_retry"`
	RetryBackoff      bool          `json:"retry_backoff"`
	BackoffMultiplier float64       `json:"backoff_multiplier"`
}

// DefaultDeliveryConfig returns default configuration
func DefaultDeliveryConfig() *DeliveryConfig {
	return &DeliveryConfig{
		MaxAttempts:       3,
		InitialDelay:      1 * time.Second,
		RetryDelay:        5 * time.Second,
		Timeout:           30 * time.Second,
		EnableRetry:       true,
		RetryBackoff:      true,
		BackoffMultiplier: 2.0,
	}
}

// DeliveryTracker tracks delivery status
type DeliveryTracker struct {
	records  map[string]*DeliveryRecord
	config   *DeliveryConfig
	mutex    sync.RWMutex
	notifyCh chan *DeliveryRecord
}

// NewDeliveryTracker creates a new delivery tracker
func NewDeliveryTracker(config *DeliveryConfig) *DeliveryTracker {
	if config == nil {
		config = DefaultDeliveryConfig()
	}

	return &DeliveryTracker{
		records:  make(map[string]*DeliveryRecord),
		config:   config,
		notifyCh: make(chan *DeliveryRecord, 100),
	}
}

// Register registers a new delivery
func (t *DeliveryTracker) Register(alertID, recipient string) *DeliveryRecord {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	key := fmt.Sprintf("%s:%s", alertID, recipient)
	now := time.Now()

	record := &DeliveryRecord{
		AlertID:     alertID,
		Recipient:   recipient,
		Status:      StatusPending,
		Attempts:    0,
		MaxAttempts: t.config.MaxAttempts,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	t.records[key] = record
	return record
}

// MarkSent marks a delivery as sent
func (t *DeliveryTracker) MarkSent(alertID, recipient string) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	key := fmt.Sprintf("%s:%s", alertID, recipient)
	record, exists := t.records[key]
	if !exists {
		return ErrRecordNotFound
	}

	now := time.Now()
	record.Status = StatusSent
	record.Attempts++
	if record.FirstAttempt.IsZero() {
		record.FirstAttempt = now
	}
	record.LastAttempt = now
	record.UpdatedAt = now

	t.notify(record)
	return nil
}

// MarkDelivered marks a delivery as delivered
func (t *DeliveryTracker) MarkDelivered(alertID, recipient string) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	key := fmt.Sprintf("%s:%s", alertID, recipient)
	record, exists := t.records[key]
	if !exists {
		return ErrRecordNotFound
	}

	now := time.Now()
	record.Status = StatusDelivered
	record.DeliveredAt = now
	record.UpdatedAt = now

	t.notify(record)
	return nil
}

// MarkAcknowledged marks a delivery as acknowledged
func (t *DeliveryTracker) MarkAcknowledged(alertID, recipient string) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	key := fmt.Sprintf("%s:%s", alertID, recipient)
	record, exists := t.records[key]
	if !exists {
		return ErrRecordNotFound
	}

	now := time.Now()
	record.Status = StatusAcknowledged
	record.AcknowledgedAt = now
	record.UpdatedAt = now

	t.notify(record)
	return nil
}

// MarkFailed marks a delivery as failed
func (t *DeliveryTracker) MarkFailed(alertID, recipient string, errStr string) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	key := fmt.Sprintf("%s:%s", alertID, recipient)
	record, exists := t.records[key]
	if !exists {
		return ErrRecordNotFound
	}

	now := time.Now()
	record.LastError = errStr
	record.UpdatedAt = now

	if record.Attempts >= record.MaxAttempts {
		record.Status = StatusFailed
		t.notify(record)
	}

	return nil
}

// ShouldRetry checks if a delivery should be retried
func (t *DeliveryTracker) ShouldRetry(alertID, recipient string) (bool, time.Duration) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	key := fmt.Sprintf("%s:%s", alertID, recipient)
	record, exists := t.records[key]
	if !exists {
		return false, 0
	}

	if !t.config.EnableRetry {
		return false, 0
	}

	if record.Attempts >= record.MaxAttempts {
		return false, 0
	}

	if record.Status == StatusDelivered || record.Status == StatusAcknowledged {
		return false, 0
	}

	// Calculate retry delay
	delay := t.config.RetryDelay
	if t.config.RetryBackoff {
		for i := 0; i < record.Attempts; i++ {
			delay = time.Duration(float64(delay) * t.config.BackoffMultiplier)
		}
	}

	return true, delay
}

// NextRetry returns the next retry time
func (t *DeliveryTracker) NextRetry(alertID, recipient string) (time.Time, error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	key := fmt.Sprintf("%s:%s", alertID, recipient)
	record, exists := t.records[key]
	if !exists {
		return time.Time{}, ErrRecordNotFound
	}

	if record.Attempts == 0 {
		return record.CreatedAt.Add(t.config.InitialDelay), nil
	}

	shouldRetry, delay := t.ShouldRetry(alertID, recipient)
	if !shouldRetry {
		return time.Time{}, ErrMaxAttemptsExceeded
	}

	return record.LastAttempt.Add(delay), nil
}

// GetRecord returns a delivery record
func (t *DeliveryTracker) GetRecord(alertID, recipient string) (*DeliveryRecord, error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	key := fmt.Sprintf("%s:%s", alertID, recipient)
	record, exists := t.records[key]
	if !exists {
		return nil, ErrRecordNotFound
	}

	return record, nil
}

// GetRecordsByAlert returns all records for an alert
func (t *DeliveryTracker) GetRecordsByAlert(alertID string) []*DeliveryRecord {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	var result []*DeliveryRecord
	for _, record := range t.records {
		if record.AlertID == alertID {
			result = append(result, record)
		}
	}
	return result
}

// GetRecordsByRecipient returns all records for a recipient
func (t *DeliveryTracker) GetRecordsByRecipient(recipient string) []*DeliveryRecord {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	var result []*DeliveryRecord
	for _, record := range t.records {
		if record.Recipient == recipient {
			result = append(result, record)
		}
	}
	return result
}

// GetPending returns all pending deliveries
func (t *DeliveryTracker) GetPending() []*DeliveryRecord {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	var result []*DeliveryRecord
	for _, record := range t.records {
		if record.Status == StatusPending || record.Status == StatusSent {
			result = append(result, record)
		}
	}
	return result
}

// GetFailed returns all failed deliveries
func (t *DeliveryTracker) GetFailed() []*DeliveryRecord {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	var result []*DeliveryRecord
	for _, record := range t.records {
		if record.Status == StatusFailed {
			result = append(result, record)
		}
	}
	return result
}

// CheckTimeouts checks for timed out deliveries
func (t *DeliveryTracker) CheckTimeouts() []*DeliveryRecord {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	now := time.Now()
	var timedOut []*DeliveryRecord

	for _, record := range t.records {
		if record.Status == StatusPending || record.Status == StatusSent {
			if now.Sub(record.CreatedAt) > t.config.Timeout {
				record.Status = StatusTimeout
				record.UpdatedAt = now
				timedOut = append(timedOut, record)
				t.notify(record)
			}
		}
	}

	return timedOut
}

// Stats returns delivery statistics
func (t *DeliveryTracker) Stats() DeliveryStats {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	stats := DeliveryStats{}
	for _, record := range t.records {
		switch record.Status {
		case StatusPending:
			stats.Pending++
		case StatusSent:
			stats.Sent++
		case StatusDelivered:
			stats.Delivered++
		case StatusAcknowledged:
			stats.Acknowledged++
		case StatusFailed:
			stats.Failed++
		case StatusTimeout:
			stats.Timeout++
		}
	}

	return stats
}

// DeliveryStats holds delivery statistics
type DeliveryStats struct {
	Pending      int `json:"pending"`
	Sent         int `json:"sent"`
	Delivered    int `json:"delivered"`
	Acknowledged int `json:"acknowledged"`
	Failed       int `json:"failed"`
	Timeout      int `json:"timeout"`
}

// Notifications returns the notification channel
func (t *DeliveryTracker) Notifications() <-chan *DeliveryRecord {
	return t.notifyCh
}

// notify sends a notification
func (t *DeliveryTracker) notify(record *DeliveryRecord) {
	select {
	case t.notifyCh <- record:
	default:
		// Channel full, drop notification
	}
}

// Clear removes all records
func (t *DeliveryTracker) Clear() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.records = make(map[string]*DeliveryRecord)
}

// Retrier handles retry logic
type Retrier struct {
	tracker  *DeliveryTracker
	sendFunc func(alertID, recipient string) error
}

// NewRetrier creates a new retrier
func NewRetrier(tracker *DeliveryTracker, sendFunc func(alertID, recipient string) error) *Retrier {
	return &Retrier{
		tracker:  tracker,
		sendFunc: sendFunc,
	}
}

// Retry retries delivery
func (r *Retrier) Retry(ctx context.Context, alertID, recipient string) error {
	shouldRetry, delay := r.tracker.ShouldRetry(alertID, recipient)
	if !shouldRetry {
		return ErrMaxAttemptsExceeded
	}

	// Wait for delay or context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
	}

	// Mark as sent (attempt)
	if err := r.tracker.MarkSent(alertID, recipient); err != nil {
		return err
	}

	// Send
	err := r.sendFunc(alertID, recipient)
	if err != nil {
		r.tracker.MarkFailed(alertID, recipient, err.Error())
		return err
	}

	// Mark as delivered
	return r.tracker.MarkDelivered(alertID, recipient)
}

// RetryAll retries all pending deliveries
func (r *Retrier) RetryAll(ctx context.Context) error {
	pending := r.tracker.GetPending()
	for _, record := range pending {
		shouldRetry, _ := r.tracker.ShouldRetry(record.AlertID, record.Recipient)
		if shouldRetry {
			if err := r.Retry(ctx, record.AlertID, record.Recipient); err != nil {
				// Continue with other deliveries
				continue
			}
		}
	}
	return nil
}

// Errors
var (
	ErrRecordNotFound      = &DeliveryError{Code: "RECORD_NOT_FOUND", Message: "delivery record not found"}
	ErrMaxAttemptsExceeded = &DeliveryError{Code: "MAX_ATTEMPTS", Message: "maximum attempts exceeded"}
)

// DeliveryError represents a delivery error
type DeliveryError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *DeliveryError) Error() string {
	return e.Message
}
