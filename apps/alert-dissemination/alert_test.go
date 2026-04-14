// Package alertdissemination_test provides tests for alert dissemination
package alertdissemination_test

import (
	"context"
	"testing"
	"time"

	"github.com/wezzels/vigil/apps/alert-dissemination"
)

// TestRecipientManagement tests recipient CRUD operations
func TestRecipientManagement(t *testing.T) {
	rm := alertdissemination.NewRecipientManager()
	ctx := context.Background()

	recipient := &alertdissemination.Recipient{
		ID:     "rec-001",
		Name:   "Test User",
		Type:   alertdissemination.RecipientUser,
		Active: true,
		Channels: []alertdissemination.Channel{
			{Type: alertdissemination.ChannelEmail, Address: "test@example.com", Priority: 1},
		},
	}

	// Test Add
	err := rm.AddRecipient(ctx, recipient)
	if err != nil {
		t.Fatalf("Failed to add recipient: %v", err)
	}

	// Test Get
	retrieved, err := rm.GetRecipient(ctx, "rec-001")
	if err != nil {
		t.Fatalf("Failed to get recipient: %v", err)
	}

	if retrieved.Name != recipient.Name {
		t.Errorf("Name mismatch: got %s, want %s", retrieved.Name, recipient.Name)
	}

	// Test Update
	recipient.Name = "Updated User"
	err = rm.UpdateRecipient(ctx, recipient)
	if err != nil {
		t.Fatalf("Failed to update recipient: %v", err)
	}

	// Test Delete
	err = rm.DeleteRecipient(ctx, "rec-001")
	if err != nil {
		t.Fatalf("Failed to delete recipient: %v", err)
	}

	// Verify deleted
	_, err = rm.GetRecipient(ctx, "rec-001")
	if err == nil {
		t.Error("Expected error for deleted recipient")
	}
}

// TestChannelManagement tests channel operations
func TestChannelManagement(t *testing.T) {
	rm := alertdissemination.NewRecipientManager()
	ctx := context.Background()

	recipient := &alertdissemination.Recipient{
		ID:     "rec-001",
		Name:   "Test User",
		Type:   alertdissemination.RecipientUser,
		Active: true,
	}
	rm.AddRecipient(ctx, recipient)

	// Add channel
	channel := alertdissemination.Channel{
		Type:     alertdissemination.ChannelSMS,
		Address:  "+1234567890",
		Priority: 2,
	}
	err := rm.AddChannel(ctx, "rec-001", channel)
	if err != nil {
		t.Fatalf("Failed to add channel: %v", err)
	}

	// Verify channel added
	rec, _ := rm.GetRecipient(ctx, "rec-001")
	if len(rec.Channels) != 1 {
		t.Errorf("Expected 1 channel, got %d", len(rec.Channels))
	}

	// Remove channel
	err = rm.RemoveChannel(ctx, "rec-001", "+1234567890")
	if err != nil {
		t.Fatalf("Failed to remove channel: %v", err)
	}

	// Verify channel removed
	rec, _ = rm.GetRecipient(ctx, "rec-001")
	if len(rec.Channels) != 0 {
		t.Errorf("Expected 0 channels, got %d", len(rec.Channels))
	}
}

// TestDeliveryStatus tests delivery status tracking
func TestDeliveryStatus(t *testing.T) {
	rm := alertdissemination.NewRecipientManager()
	ctx := context.Background()

	recipient := &alertdissemination.Recipient{
		ID:     "rec-001",
		Name:   "Test User",
		Type:   alertdissemination.RecipientUser,
		Active: true,
		Channels: []alertdissemination.Channel{
			{Type: alertdissemination.ChannelEmail, Address: "test@example.com"},
		},
	}
	rm.AddRecipient(ctx, recipient)

	// Set delivery status
	status := &alertdissemination.DeliveryStatus{
		RecipientID: "rec-001",
		ChannelID:   "test@example.com",
		Status:       alertdissemination.DeliveryPending,
		Attempts:     0,
	}

	err := rm.SetDeliveryStatus(ctx, status)
	if err != nil {
		t.Fatalf("Failed to set delivery status: %v", err)
	}

	// Get delivery status
	retrieved, err := rm.GetDeliveryStatus(ctx, "rec-001", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to get delivery status: %v", err)
	}

	if retrieved.Status != alertdissemination.DeliveryPending {
		t.Errorf("Status mismatch: got %s, want %s", retrieved.Status, alertdissemination.DeliveryPending)
	}
}

// TestDeliveryStatusUpdate tests delivery status updates
func TestDeliveryStatusUpdate(t *testing.T) {
	rm := alertdissemination.NewRecipientManager()
	ctx := context.Background()

	// Test successful delivery
	err := rm.UpdateDeliveryStatus(ctx, "rec-001", "test@example.com", true, "")
	if err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}

	status, _ := rm.GetDeliveryStatus(ctx, "rec-001", "test@example.com")
	if status.Status != alertdissemination.DeliverySent {
		t.Errorf("Expected sent status, got %s", status.Status)
	}

	// Test failed delivery
	err = rm.UpdateDeliveryStatus(ctx, "rec-002", "sms", false, "Connection timeout")
	if err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}

	status, _ = rm.GetDeliveryStatus(ctx, "rec-002", "sms")
	if status.Status != alertdissemination.DeliveryRetry {
		t.Errorf("Expected retry status, got %s", status.Status)
	}
}

// TestGetPendingDeliveries tests pending delivery retrieval
func TestGetPendingDeliveries(t *testing.T) {
	rm := alertdissemination.NewRecipientManager()
	ctx := context.Background()

	// Add pending status
	rm.SetDeliveryStatus(ctx, &alertdissemination.DeliveryStatus{
		RecipientID: "rec-001",
		ChannelID:   "email",
		Status:       alertdissemination.DeliveryPending,
	})

	// Add sent status
	rm.SetDeliveryStatus(ctx, &alertdissemination.DeliveryStatus{
		RecipientID: "rec-002",
		ChannelID:   "sms",
		Status:       alertdissemination.DeliverySent,
	})

	// Add retry status
	rm.SetDeliveryStatus(ctx, &alertdissemination.DeliveryStatus{
		RecipientID: "rec-003",
		ChannelID:   "voice",
		Status:       alertdissemination.DeliveryRetry,
	})

	pending := rm.GetPendingDeliveries(ctx)
	if len(pending) != 2 {
		t.Errorf("Expected 2 pending deliveries, got %d", len(pending))
	}
}

// TestActiveRecipients tests active recipient filtering
func TestActiveRecipients(t *testing.T) {
	rm := alertdissemination.NewRecipientManager()
	ctx := context.Background()

	// Add active recipient
	rm.AddRecipient(ctx, &alertdissemination.Recipient{
		ID:     "rec-001",
		Name:   "Active User",
		Type:   alertdissemination.RecipientUser,
		Active: true,
	})

	// Add inactive recipient
	rm.AddRecipient(ctx, &alertdissemination.Recipient{
		ID:     "rec-002",
		Name:   "Inactive User",
		Type:   alertdissemination.RecipientUser,
		Active: false,
	})

	active := rm.GetActiveRecipients(ctx)
	if len(active) != 1 {
		t.Errorf("Expected 1 active recipient, got %d", len(active))
	}

	if active[0].ID != "rec-001" {
		t.Errorf("Wrong recipient returned: %s", active[0].ID)
	}
}

// TestMultipleChannels tests recipients with multiple channels
func TestMultipleChannels(t *testing.T) {
	rm := alertdissemination.NewRecipientManager()
	ctx := context.Background()

	recipient := &alertdissemination.Recipient{
		ID:     "rec-001",
		Name:   "Multi-Channel User",
		Type:   alertdissemination.RecipientUser,
		Active: true,
		Channels: []alertdissemination.Channel{
			{Type: alertdissemination.ChannelEmail, Address: "user@example.com", Priority: 1},
			{Type: alertdissemination.ChannelSMS, Address: "+1234567890", Priority: 2},
			{Type: alertdissemination.ChannelVoice, Address: "+1234567890", Priority: 3},
		},
	}

	err := rm.AddRecipient(ctx, recipient)
	if err != nil {
		t.Fatalf("Failed to add recipient: %v", err)
	}

	retrieved, _ := rm.GetRecipient(ctx, "rec-001")
	if len(retrieved.Channels) != 3 {
		t.Errorf("Expected 3 channels, got %d", len(retrieved.Channels))
	}
}

// TestRetryLogic tests delivery retry logic
func TestRetryLogic(t *testing.T) {
	rm := alertdissemination.NewRecipientManager()
	ctx := context.Background()

	// Initial failed attempt
	rm.UpdateDeliveryStatus(ctx, "rec-001", "email", false, "Timeout")
	status, _ := rm.GetDeliveryStatus(ctx, "rec-001", "email")
	if status.Attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", status.Attempts)
	}
	if status.Status != alertdissemination.DeliveryRetry {
		t.Errorf("Expected retry status, got %s", status.Status)
	}

	// Second failed attempt
	rm.UpdateDeliveryStatus(ctx, "rec-001", "email", false, "Timeout")
	status, _ = rm.GetDeliveryStatus(ctx, "rec-001", "email")
	if status.Attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", status.Attempts)
	}

	// Third failed attempt (should fail permanently)
	rm.UpdateDeliveryStatus(ctx, "rec-001", "email", false, "Timeout")
	status, _ = rm.GetDeliveryStatus(ctx, "rec-001", "email")
	if status.Status != alertdissemination.DeliveryFailed {
		t.Errorf("Expected failed status after 3 attempts, got %s", status.Status)
	}
}

// BenchmarkRecipientAdd benchmarks recipient addition
func BenchmarkRecipientAdd(b *testing.B) {
	rm := alertdissemination.NewRecipientManager()
	ctx := context.Background()

	recipient := &alertdissemination.Recipient{
		ID:     "rec-001",
		Name:   "Test User",
		Type:   alertdissemination.RecipientUser,
		Active: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rm.AddRecipient(ctx, recipient)
	}
}

// BenchmarkStatusUpdate benchmarks status updates
func BenchmarkStatusUpdate(b *testing.B) {
	rm := alertdissemination.NewRecipientManager()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rm.UpdateDeliveryStatus(ctx, "rec-001", "email", true, "")
	}
}