// Package alertdissemination provides alert dissemination functionality
package alertdissemination

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Recipient represents an alert recipient
type Recipient struct {
	ID          string
	Name        string
	Type        RecipientType
	Channels    []Channel
	Active      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// RecipientType defines recipient type
type RecipientType string

const (
	RecipientUser    RecipientType = "user"
	RecipientGroup   RecipientType = "group"
	RecipientSystem  RecipientType = "system"
)

// Channel defines a delivery channel
type Channel struct {
	Type      ChannelType
	Address   string
	Priority  int
	Active    bool
}

// ChannelType defines channel type
type ChannelType string

const (
	ChannelEmail   ChannelType = "email"
	ChannelSMS     ChannelType = "sms"
	ChannelVoice   ChannelType = "voice"
	ChannelWebhook ChannelType = "webhook"
	ChannelPUSH    ChannelType = "push"
)

// DeliveryStatus represents delivery status
type DeliveryStatus struct {
	RecipientID  string
	ChannelID    string
	Status       DeliveryState
	Attempts     int
	LastAttempt  time.Time
	NextRetry    *time.Time
	Error        string
}

// DeliveryState defines delivery state
type DeliveryState string

const (
	DeliveryPending   DeliveryState = "pending"
	DeliverySent      DeliveryState = "sent"
	DeliveryDelivered DeliveryState = "delivered"
	DeliveryFailed    DeliveryState = "failed"
	DeliveryRetry     DeliveryState = "retry"
)

// RecipientManager manages recipients
type RecipientManager struct {
	mu          sync.RWMutex
	recipients  map[string]*Recipient
	statuses    map[string]*DeliveryStatus
}

// NewRecipientManager creates a new recipient manager
func NewRecipientManager() *RecipientManager {
	return &RecipientManager{
		recipients: make(map[string]*Recipient),
		statuses:   make(map[string]*DeliveryStatus),
	}
}

// AddRecipient adds a recipient
func (rm *RecipientManager) AddRecipient(ctx context.Context, recipient *Recipient) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	recipient.CreatedAt = time.Now()
	recipient.UpdatedAt = time.Now()
	rm.recipients[recipient.ID] = recipient

	return nil
}

// GetRecipient gets a recipient by ID
func (rm *RecipientManager) GetRecipient(ctx context.Context, id string) (*Recipient, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	recipient, ok := rm.recipients[id]
	if !ok {
		return nil, fmt.Errorf("recipient %s not found", id)
	}

	return recipient, nil
}

// UpdateRecipient updates a recipient
func (rm *RecipientManager) UpdateRecipient(ctx context.Context, recipient *Recipient) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if _, ok := rm.recipients[recipient.ID]; !ok {
		return fmt.Errorf("recipient %s not found", recipient.ID)
	}

	recipient.UpdatedAt = time.Now()
	rm.recipients[recipient.ID] = recipient

	return nil
}

// DeleteRecipient deletes a recipient
func (rm *RecipientManager) DeleteRecipient(ctx context.Context, id string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	delete(rm.recipients, id)
	return nil
}

// ListRecipients lists all recipients
func (rm *RecipientManager) ListRecipients(ctx context.Context) []*Recipient {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	recipients := make([]*Recipient, 0, len(rm.recipients))
	for _, r := range rm.recipients {
		recipients = append(recipients, r)
	}

	return recipients
}

// AddChannel adds a channel to a recipient
func (rm *RecipientManager) AddChannel(ctx context.Context, recipientID string, channel Channel) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	recipient, ok := rm.recipients[recipientID]
	if !ok {
		return fmt.Errorf("recipient %s not found", recipientID)
	}

	recipient.Channels = append(recipient.Channels, channel)
	recipient.UpdatedAt = time.Now()

	return nil
}

// RemoveChannel removes a channel from a recipient
func (rm *RecipientManager) RemoveChannel(ctx context.Context, recipientID string, channelID string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	recipient, ok := rm.recipients[recipientID]
	if !ok {
		return fmt.Errorf("recipient %s not found", recipientID)
	}

	for i, ch := range recipient.Channels {
		if ch.Address == channelID {
			recipient.Channels = append(recipient.Channels[:i], recipient.Channels[i+1:]...)
			recipient.UpdatedAt = time.Now()
			break
		}
	}

	return nil
}

// SetDeliveryStatus sets delivery status
func (rm *RecipientManager) SetDeliveryStatus(ctx context.Context, status *DeliveryStatus) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	key := fmt.Sprintf("%s:%s", status.RecipientID, status.ChannelID)
	rm.statuses[key] = status

	return nil
}

// GetDeliveryStatus gets delivery status
func (rm *RecipientManager) GetDeliveryStatus(ctx context.Context, recipientID, channelID string) (*DeliveryStatus, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", recipientID, channelID)
	status, ok := rm.statuses[key]
	if !ok {
		return nil, fmt.Errorf("status not found")
	}

	return status, nil
}

// GetPendingDeliveries gets all pending deliveries
func (rm *RecipientManager) GetPendingDeliveries(ctx context.Context) []*DeliveryStatus {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	pending := make([]*DeliveryStatus, 0)
	for _, status := range rm.statuses {
		if status.Status == DeliveryPending || status.Status == DeliveryRetry {
			pending = append(pending, status)
		}
	}

	return pending
}

// UpdateDeliveryStatus updates delivery status after attempt
func (rm *RecipientManager) UpdateDeliveryStatus(ctx context.Context, recipientID, channelID string, success bool, errMsg string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	key := fmt.Sprintf("%s:%s", recipientID, channelID)
	status, ok := rm.statuses[key]
	if !ok {
		status = &DeliveryStatus{
			RecipientID: recipientID,
			ChannelID:   channelID,
		}
		rm.statuses[key] = status
	}

	status.Attempts++
	status.LastAttempt = time.Now()

	if success {
		status.Status = DeliverySent
		status.Error = ""
	} else {
		if status.Attempts >= 3 {
			status.Status = DeliveryFailed
		} else {
			status.Status = DeliveryRetry
			nextRetry := time.Now().Add(time.Duration(status.Attempts*5) * time.Minute)
			status.NextRetry = &nextRetry
		}
		status.Error = errMsg
	}

	return nil
}

// GetActiveRecipients gets all active recipients
func (rm *RecipientManager) GetActiveRecipients(ctx context.Context) []*Recipient {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	active := make([]*Recipient, 0)
	for _, r := range rm.recipients {
		if r.Active {
			active = append(active, r)
		}
	}

	return active
}