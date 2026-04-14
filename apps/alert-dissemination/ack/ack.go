// Package ack provides acknowledgment handling for alert dissemination
package ack

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AckStatus represents the acknowledgment status
type AckStatus int

const (
	AckStatusPending AckStatus = iota
	AckStatusAcknowledged
	AckStatusRejected
	AckStatusTimeout
	AckStatusFailed
)

// String returns string representation of status
func (s AckStatus) String() string {
	switch s {
	case AckStatusPending:
		return "PENDING"
	case AckStatusAcknowledged:
		return "ACKNOWLEDGED"
	case AckStatusRejected:
		return "REJECTED"
	case AckStatusTimeout:
		return "TIMEOUT"
	case AckStatusFailed:
		return "FAILED"
	default:
		return "UNKNOWN"
	}
}

// Acknowledgment represents an alert acknowledgment
type Acknowledgment struct {
	AlertID       string     `json:"alert_id"`
	Recipient      string     `json:"recipient"`
	Status         AckStatus  `json:"status"`
	AcknowledgedBy string     `json:"acknowledged_by,omitempty"`
	AcknowledgedAt time.Time  `json:"acknowledged_at,omitempty"`
	Reason         string     `json:"reason,omitempty"`
	Attempts       int        `json:"attempts"`
	MaxAttempts    int        `json:"max_attempts"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	ExpiresAt      time.Time  `json:"expires_at"`
}

// AckHandler handles acknowledgment processing
type AckHandler struct {
	pending      map[string]*Acknowledgment
	ackCallbacks map[string][]func(*Acknowledgment)
	nackCallbacks map[string][]func(*Acknowledgment)
	timeoutCallbacks map[string][]func(*Acknowledgment)
	mutex        sync.RWMutex
	timeout      time.Duration
}

// NewAckHandler creates a new acknowledgment handler
func NewAckHandler(timeout time.Duration) *AckHandler {
	return &AckHandler{
		pending:          make(map[string]*Acknowledgment),
		ackCallbacks:     make(map[string][]func(*Acknowledgment)),
		nackCallbacks:    make(map[string][]func(*Acknowledgment)),
		timeoutCallbacks: make(map[string][]func(*Acknowledgment)),
		timeout:          timeout,
	}
}

// Register creates a pending acknowledgment
func (h *AckHandler) Register(alertID, recipient string) *Acknowledgment {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	key := fmt.Sprintf("%s:%s", alertID, recipient)
	now := time.Now()

	ack := &Acknowledgment{
		AlertID:    alertID,
		Recipient:   recipient,
		Status:      AckStatusPending,
		Attempts:    0,
		MaxAttempts: 3,
		CreatedAt:   now,
		UpdatedAt:   now,
		ExpiresAt:   now.Add(h.timeout),
	}

	h.pending[key] = ack
	return ack
}

// Acknowledge processes an acknowledgment
func (h *AckHandler) Acknowledge(alertID, recipient, acknowledgedBy string) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	key := fmt.Sprintf("%s:%s", alertID, recipient)
	ack, exists := h.pending[key]
	if !exists {
		return ErrAckNotFound
	}

	now := time.Now()
	ack.Status = AckStatusAcknowledged
	ack.AcknowledgedBy = acknowledgedBy
	ack.AcknowledgedAt = now
	ack.UpdatedAt = now

	// Trigger callbacks
	callbacks := h.ackCallbacks[alertID]
	for _, cb := range callbacks {
		cb(ack)
	}

	// Remove from pending
	delete(h.pending, key)

	return nil
}

// Reject processes a rejection
func (h *AckHandler) Reject(alertID, recipient, reason string) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	key := fmt.Sprintf("%s:%s", alertID, recipient)
	ack, exists := h.pending[key]
	if !exists {
		return ErrAckNotFound
	}

	now := time.Now()
	ack.Status = AckStatusRejected
	ack.Reason = reason
	ack.UpdatedAt = now

	// Trigger callbacks
	callbacks := h.nackCallbacks[alertID]
	for _, cb := range callbacks {
		cb(ack)
	}

	// Remove from pending
	delete(h.pending, key)

	return nil
}

// Timeout checks for timed out acknowledgments
func (h *AckHandler) Timeout() []*Acknowledgment {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	now := time.Now()
	var timedOut []*Acknowledgment

	for key, ack := range h.pending {
		if now.After(ack.ExpiresAt) {
			ack.Status = AckStatusTimeout
			ack.UpdatedAt = now

			// Trigger callbacks
			callbacks := h.timeoutCallbacks[ack.AlertID]
			for _, cb := range callbacks {
				cb(ack)
			}

			timedOut = append(timedOut, ack)
			delete(h.pending, key)
		}
	}

	return timedOut
}

// IncrementAttempts increments the attempt count
func (h *AckHandler) IncrementAttempts(alertID, recipient string) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	key := fmt.Sprintf("%s:%s", alertID, recipient)
	ack, exists := h.pending[key]
	if !exists {
		return ErrAckNotFound
	}

	ack.Attempts++
	ack.UpdatedAt = time.Now()

	if ack.Attempts >= ack.MaxAttempts {
		ack.Status = AckStatusFailed
		delete(h.pending, key)
		return ErrMaxAttemptsExceeded
	}

	// Extend expiration
	ack.ExpiresAt = time.Now().Add(h.timeout)

	return nil
}

// Get returns an acknowledgment by alert ID and recipient
func (h *AckHandler) Get(alertID, recipient string) (*Acknowledgment, error) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	key := fmt.Sprintf("%s:%s", alertID, recipient)
	ack, exists := h.pending[key]
	if !exists {
		return nil, ErrAckNotFound
	}

	return ack, nil
}

// ListPending returns all pending acknowledgments
func (h *AckHandler) ListPending() []*Acknowledgment {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	result := make([]*Acknowledgment, 0, len(h.pending))
	for _, ack := range h.pending {
		result = append(result, ack)
	}
	return result
}

// ListByAlert returns all acknowledgments for an alert
func (h *AckHandler) ListByAlert(alertID string) []*Acknowledgment {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	var result []*Acknowledgment
	for key, ack := range h.pending {
		if ack.AlertID == alertID {
			result = append(result, ack)
			_ = key // unused
		}
	}
	return result
}

// OnAck registers a callback for acknowledgment events
func (h *AckHandler) OnAck(alertID string, callback func(*Acknowledgment)) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.ackCallbacks[alertID] = append(h.ackCallbacks[alertID], callback)
}

// OnNack registers a callback for rejection events
func (h *AckHandler) OnNack(alertID string, callback func(*Acknowledgment)) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.nackCallbacks[alertID] = append(h.nackCallbacks[alertID], callback)
}

// OnTimeout registers a callback for timeout events
func (h *AckHandler) OnTimeout(alertID string, callback func(*Acknowledgment)) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.timeoutCallbacks[alertID] = append(h.timeoutCallbacks[alertID], callback)
}

// Stats returns acknowledgment statistics
func (h *AckHandler) Stats() AckStats {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	stats := AckStats{}
	for _, ack := range h.pending {
		switch ack.Status {
		case AckStatusPending:
			stats.Pending++
		case AckStatusAcknowledged:
			stats.Acknowledged++
		case AckStatusRejected:
			stats.Rejected++
		case AckStatusTimeout:
			stats.Timeout++
		case AckStatusFailed:
			stats.Failed++
		}
	}

	return stats
}

// AckStats holds acknowledgment statistics
type AckStats struct {
	Pending      int `json:"pending"`
	Acknowledged int `json:"acknowledged"`
	Rejected     int `json:"rejected"`
	Timeout      int `json:"timeout"`
	Failed       int `json:"failed"`
}

// AckTracker tracks acknowledgments across multiple handlers
type AckTracker struct {
	handlers map[string]*AckHandler
	timeout  time.Duration
	mutex    sync.RWMutex
}

// NewAckTracker creates a new acknowledgment tracker
func NewAckTracker(timeout time.Duration) *AckTracker {
	return &AckTracker{
		handlers: make(map[string]*AckHandler),
		timeout:  timeout,
	}
}

// Register registers a new acknowledgment
func (t *AckTracker) Register(alertID, recipient string) *Acknowledgment {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if _, exists := t.handlers[alertID]; !exists {
		t.handlers[alertID] = NewAckHandler(t.timeout)
	}

	return t.handlers[alertID].Register(alertID, recipient)
}

// Acknowledge acknowledges an alert
func (t *AckTracker) Acknowledge(alertID, recipient, acknowledgedBy string) error {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	handler, exists := t.handlers[alertID]
	if !exists {
		return ErrAckNotFound
	}

	return handler.Acknowledge(alertID, recipient, acknowledgedBy)
}

// Reject rejects an alert
func (t *AckTracker) Reject(alertID, recipient, reason string) error {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	handler, exists := t.handlers[alertID]
	if !exists {
		return ErrAckNotFound
	}

	return handler.Reject(alertID, recipient, reason)
}

// IsComplete checks if all acknowledgments are complete
func (t *AckTracker) IsComplete(alertID string) bool {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	handler, exists := t.handlers[alertID]
	if !exists {
		return true
	}

	return len(handler.ListPending()) == 0
}

// WaitForAll waits for all acknowledgments to complete
func (t *AckTracker) WaitForAll(ctx context.Context, alertID string) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if t.IsComplete(alertID) {
				return nil
			}
		}
	}
}

// Stats returns total statistics
func (t *AckTracker) Stats() AckStats {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	total := AckStats{}
	for _, handler := range t.handlers {
		stats := handler.Stats()
		total.Pending += stats.Pending
		total.Acknowledged += stats.Acknowledged
		total.Rejected += stats.Rejected
		total.Timeout += stats.Timeout
		total.Failed += stats.Failed
	}

	return total
}

// Errors
var (
	ErrAckNotFound        = &AckError{Code: "ACK_NOT_FOUND", Message: "acknowledgment not found"}
	ErrMaxAttemptsExceeded = &AckError{Code: "MAX_ATTEMPTS", Message: "maximum attempts exceeded"}
)

// AckError represents an acknowledgment error
type AckError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *AckError) Error() string {
	return e.Message
}