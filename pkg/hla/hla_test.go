// Package hla_test provides tests for HLA federation support
package hla_test

import (
	"context"
	"testing"
	"time"

	"github.com/wezzels/vigil/pkg/hla"
)

// TestFederationJoin tests federation join
func TestFederationJoin(t *testing.T) {
	federation := hla.NewFederation("TestFederation", "TestFederate")

	ctx := context.Background()

	// Test join
	err := federation.Join(ctx)
	if err != nil {
		t.Fatalf("Failed to join federation: %v", err)
	}

	if !federation.IsJoined() {
		t.Error("Federation should be joined")
	}

	// Test double join
	err = federation.Join(ctx)
	if err == nil {
		t.Error("Expected error on double join")
	}
}

// TestFederationResign tests federation resign
func TestFederationResign(t *testing.T) {
	federation := hla.NewFederation("TestFederation", "TestFederate")

	ctx := context.Background()
	federation.Join(ctx)

	// Test resign
	err := federation.Resign(ctx)
	if err != nil {
		t.Fatalf("Failed to resign from federation: %v", err)
	}

	if federation.IsJoined() {
		t.Error("Federation should not be joined after resign")
	}

	// Test resign when not joined
	err = federation.Resign(ctx)
	if err == nil {
		t.Error("Expected error on resign when not joined")
	}
}

// TestSyncPoint tests synchronization points
func TestSyncPoint(t *testing.T) {
	federation := hla.NewFederation("TestFederation", "TestFederate")

	ctx := context.Background()
	federation.Join(ctx)

	// Test sync point registration
	err := federation.RegisterSyncPoint(ctx, "ReadyToRun")
	if err != nil {
		t.Fatalf("Failed to register sync point: %v", err)
	}

	// Test achieve sync point
	err = federation.AchieveSyncPoint(ctx, "ReadyToRun")
	if err != nil {
		t.Fatalf("Failed to achieve sync point: %v", err)
	}

	// Test unregistered sync point
	err = federation.AchieveSyncPoint(ctx, "UnknownPoint")
	if err == nil {
		t.Error("Expected error for unregistered sync point")
	}
}

// TestPublisher tests HLA publisher
func TestPublisher(t *testing.T) {
	federation := hla.NewFederation("TestFederation", "TestFederate")
	ctx := context.Background()
	federation.Join(ctx)

	publisher := hla.NewPublisher(federation, "Entity")

	// Test publish
	attrs := map[string]interface{}{
		"position": map[string]float64{"x": 100, "y": 200, "z": 300},
		"velocity": map[string]float64{"x": 10, "y": 20, "z": 5},
	}

	err := publisher.Publish(ctx, "entity-001", attrs)
	if err != nil {
		t.Fatalf("Failed to publish object: %v", err)
	}

	// Test update
	err = publisher.UpdateAttribute(ctx, "entity-001", "position", map[string]float64{"x": 110, "y": 210, "z": 310})
	if err != nil {
		t.Fatalf("Failed to update attribute: %v", err)
	}

	// Test get object
	obj, err := publisher.GetObject("entity-001")
	if err != nil {
		t.Fatalf("Failed to get object: %v", err)
	}

	if obj.ID != "entity-001" {
		t.Errorf("Expected entity-001, got %s", obj.ID)
	}

	// Test delete
	err = publisher.DeleteObject(ctx, "entity-001")
	if err != nil {
		t.Fatalf("Failed to delete object: %v", err)
	}

	// Verify deleted
	_, err = publisher.GetObject("entity-001")
	if err == nil {
		t.Error("Expected error for deleted object")
	}
}

// TestSubscriber tests HLA subscriber
func TestSubscriber(t *testing.T) {
	federation := hla.NewFederation("TestFederation", "TestFederate")
	ctx := context.Background()
	federation.Join(ctx)

	subscriber := hla.NewSubscriber(federation, "Entity")

	// Test subscribe
	err := subscriber.Subscribe(ctx, []string{"position", "velocity"})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Test object discovery
	discovered := false
	subscriber.OnObjectDiscover(func(ctx context.Context, obj *hla.DiscoveredObject) error {
		discovered = true
		return nil
	})

	err = subscriber.DiscoverObject(ctx, "entity-001", "OtherFederate")
	if err != nil {
		t.Fatalf("Failed to discover object: %v", err)
	}

	if !discovered {
		t.Error("Handler should have been called")
	}

	// Test attribute reflection
	attrs := map[string]interface{}{
		"position": map[string]float64{"x": 100, "y": 200, "z": 300},
	}

	err = subscriber.ReflectAttributes(ctx, "entity-001", attrs)
	if err != nil {
		t.Fatalf("Failed to reflect attributes: %v", err)
	}

	// Test get object
	obj, err := subscriber.GetObject("entity-001")
	if err != nil {
		t.Fatalf("Failed to get object: %v", err)
	}

	if obj.ID != "entity-001" {
		t.Errorf("Expected entity-001, got %s", obj.ID)
	}

	// Test unsubscribe
	err = subscriber.Unsubscribe(ctx)
	if err != nil {
		t.Fatalf("Failed to unsubscribe: %v", err)
	}
}

// TestInteractionPublisher tests interaction publishing
func TestInteractionPublisher(t *testing.T) {
	federation := hla.NewFederation("TestFederation", "TestFederate")
	ctx := context.Background()
	federation.Join(ctx)

	publisher := hla.NewInteractionPublisher(federation, "FireInteraction")

	// Test publish
	params := map[string]interface{}{
		"targetId": "target-001",
		"weapon":   "Missile",
		"time":     time.Now(),
	}

	err := publisher.Publish(ctx, params)
	if err != nil {
		t.Fatalf("Failed to publish interaction: %v", err)
	}

	// Test publish with timestamp
	err = publisher.PublishWithTimestamp(ctx, params, time.Now().Add(1*time.Second))
	if err != nil {
		t.Fatalf("Failed to publish interaction with timestamp: %v", err)
	}
}

// TestParameterHandling tests parameter validation
func TestParameterHandling(t *testing.T) {
	ph := hla.NewParameterHandling()

	// Register parameters
	err := ph.RegisterParameter("targetId", "string", true, nil)
	if err != nil {
		t.Fatalf("Failed to register parameter: %v", err)
	}

	err = ph.RegisterParameter("weapon", "string", false, "DefaultWeapon")
	if err != nil {
		t.Fatalf("Failed to register parameter: %v", err)
	}

	// Test validation with missing required
	err = ph.ValidateParameters(map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for missing required parameter")
	}

	// Test validation with all required
	err = ph.ValidateParameters(map[string]interface{}{
		"targetId": "target-001",
	})
	if err != nil {
		t.Fatalf("Unexpected validation error: %v", err)
	}

	// Test defaults
	result := ph.ApplyDefaults(map[string]interface{}{
		"targetId": "target-001",
	})

	if result["weapon"] != "DefaultWeapon" {
		t.Error("Expected default weapon value")
	}
}

// TestOwnershipManager tests ownership management
func TestOwnershipManager(t *testing.T) {
	om := hla.NewOwnershipManager()

	// Test request ownership
	err := om.RequestOwnership("entity-001", "FederateA")
	if err != nil {
		t.Fatalf("Failed to request ownership: %v", err)
	}

	// Test accept ownership
	err = om.AcceptOwnership("entity-001")
	if err != nil {
		t.Fatalf("Failed to accept ownership: %v", err)
	}

	// Test release
	om.ReleaseOwnership("entity-001")
}

// TestFederateCount tests federate counting
func TestFederateCount(t *testing.T) {
	federation := hla.NewFederation("TestFederation", "TestFederate")

	if federation.GetFederateCount() != 0 {
		t.Error("Federate count should be 0 before join")
	}

	ctx := context.Background()
	federation.Join(ctx)

	if federation.GetFederateCount() != 1 {
		t.Errorf("Federate count should be 1, got %d", federation.GetFederateCount())
	}

	federation.Resign(ctx)

	if federation.GetFederateCount() != 0 {
		t.Error("Federate count should be 0 after resign")
	}
}

// TestTimeRegulation tests time regulation
func TestTimeRegulation(t *testing.T) {
	federation := hla.NewFederation("TestFederation", "TestFederate")

	ctx := context.Background()
	federation.Join(ctx)

	// Enable time regulation
	err := federation.SetTimeRegulated(true)
	if err != nil {
		t.Fatalf("Failed to enable time regulation: %v", err)
	}

	// Enable time constrained
	err = federation.SetTimeConstrained(true)
	if err != nil {
		t.Fatalf("Failed to enable time constrained: %v", err)
	}

	// Disable
	err = federation.SetTimeRegulated(false)
	if err != nil {
		t.Fatalf("Failed to disable time regulation: %v", err)
	}

	err = federation.SetTimeConstrained(false)
	if err != nil {
		t.Fatalf("Failed to disable time constrained: %v", err)
	}
}
