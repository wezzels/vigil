// Package hla provides HLA (High Level Architecture) federation support
package hla

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Publisher handles HLA object class publishing
type Publisher struct {
	mu          sync.RWMutex
	federation  *Federation
	objectClass string
	objects     map[string]*PublishedObject
}

// PublishedObject represents a published HLA object
type PublishedObject struct {
	ID         string
	Class      string
	Attributes map[string]interface{}
	Owned      bool
	UpdatedAt  time.Time
}

// NewPublisher creates a new HLA publisher
func NewPublisher(federation *Federation, objectClass string) *Publisher {
	return &Publisher{
		federation:  federation,
		objectClass: objectClass,
		objects:     make(map[string]*PublishedObject),
	}
}

// Publish publishes an object instance
func (p *Publisher) Publish(ctx context.Context, objectID string, attributes map[string]interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Register with RTI
	if err := p.federation.registerObjectInstance(objectID, p.objectClass); err != nil {
		return fmt.Errorf("failed to register object: %w", err)
	}

	// Store object
	obj := &PublishedObject{
		ID:         objectID,
		Class:      p.objectClass,
		Attributes: attributes,
		Owned:      true,
		UpdatedAt:  time.Now(),
	}
	p.objects[objectID] = obj

	// Update attributes
	return p.updateAttributes(ctx, objectID, attributes)
}

// UpdateAttribute updates a single attribute
func (p *Publisher) UpdateAttribute(ctx context.Context, objectID string, name string, value interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	obj, ok := p.objects[objectID]
	if !ok {
		return fmt.Errorf("object %s not found", objectID)
	}

	obj.Attributes[name] = value
	obj.UpdatedAt = time.Now()

	return p.updateAttributes(ctx, objectID, map[string]interface{}{name: value})
}

// UpdateAttributes updates multiple attributes
func (p *Publisher) UpdateAttributes(ctx context.Context, objectID string, attributes map[string]interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	obj, ok := p.objects[objectID]
	if !ok {
		return fmt.Errorf("object %s not found", objectID)
	}

	for name, value := range attributes {
		obj.Attributes[name] = value
	}
	obj.UpdatedAt = time.Now()

	return p.updateAttributes(ctx, objectID, attributes)
}

// DeleteObject deletes a published object
func (p *Publisher) DeleteObject(ctx context.Context, objectID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.objects[objectID]; !ok {
		return fmt.Errorf("object %s not found", objectID)
	}

	// Delete from RTI
	if err := p.federation.deleteObjectInstance(objectID); err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	delete(p.objects, objectID)
	return nil
}

// TransferOwnership transfers ownership of an object
func (p *Publisher) TransferOwnership(ctx context.Context, objectID string, newOwner string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	obj, ok := p.objects[objectID]
	if !ok {
		return fmt.Errorf("object %s not found", objectID)
	}

	if !obj.Owned {
		return fmt.Errorf("object %s not owned by this federate", objectID)
	}

	// Initiate ownership transfer
	if err := p.federation.requestOwnershipTransfer(objectID, newOwner); err != nil {
		return fmt.Errorf("ownership transfer failed: %w", err)
	}

	obj.Owned = false
	return nil
}

// GetPublishedObjects returns all published objects
func (p *Publisher) GetPublishedObjects() []*PublishedObject {
	p.mu.RLock()
	defer p.mu.RUnlock()

	objects := make([]*PublishedObject, 0, len(p.objects))
	for _, obj := range p.objects {
		objects = append(objects, obj)
	}
	return objects
}

// GetObject returns a specific published object
func (p *Publisher) GetObject(objectID string) (*PublishedObject, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	obj, ok := p.objects[objectID]
	if !ok {
		return nil, fmt.Errorf("object %s not found", objectID)
	}
	return obj, nil
}

// internal method to update attributes via RTI
func (p *Publisher) updateAttributes(ctx context.Context, objectID string, attributes map[string]interface{}) error {
	// In production, this would call RTI updateAttributeValues
	return p.federation.updateAttributeValues(objectID, attributes)
}

// OwnershipManager handles ownership management
type OwnershipManager struct {
	mu       sync.RWMutex
	owned    map[string]bool
	requests map[string]*OwnershipRequest
}

// OwnershipRequest represents a pending ownership request
type OwnershipRequest struct {
	ObjectID string
	From     string
	To       string
	Status   string
	Created  time.Time
}

// NewOwnershipManager creates a new ownership manager
func NewOwnershipManager() *OwnershipManager {
	return &OwnershipManager{
		owned:    make(map[string]bool),
		requests: make(map[string]*OwnershipRequest),
	}
}

// RequestOwnership requests ownership of an object
func (om *OwnershipManager) RequestOwnership(objectID, federate string) error {
	om.mu.Lock()
	defer om.mu.Unlock()

	om.requests[objectID] = &OwnershipRequest{
		ObjectID: objectID,
		From:     "",
		To:       federate,
		Status:   "pending",
		Created:  time.Now(),
	}

	return nil
}

// AcceptOwnership accepts ownership transfer
func (om *OwnershipManager) AcceptOwnership(objectID string) error {
	om.mu.Lock()
	defer om.mu.Unlock()

	req, ok := om.requests[objectID]
	if !ok {
		return fmt.Errorf("no pending request for %s", objectID)
	}

	req.Status = "accepted"
	om.owned[objectID] = true
	delete(om.requests, objectID)

	return nil
}

// ReleaseOwnership releases ownership
func (om *OwnershipManager) ReleaseOwnership(objectID string) {
	om.mu.Lock()
	defer om.mu.Unlock()

	delete(om.owned, objectID)
}