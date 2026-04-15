// Package hla provides HLA (High Level Architecture) subscription support
package hla

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Subscriber handles HLA object class subscription
type Subscriber struct {
	mu          sync.RWMutex
	federation  *Federation
	objectClass string
	handlers    []ObjectHandler
	objects     map[string]*DiscoveredObject
}

// ObjectHandler handles discovered objects
type ObjectHandler func(ctx context.Context, obj *DiscoveredObject) error

// DiscoveredObject represents a discovered HLA object
type DiscoveredObject struct {
	ID                string
	Class             string
	Attributes        map[string]interface{}
	DiscoverTime      time.Time
	LastUpdateTime    time.Time
	ProducingFederate string
}

// NewSubscriber creates a new HLA subscriber
func NewSubscriber(federation *Federation, objectClass string) *Subscriber {
	return &Subscriber{
		federation:  federation,
		objectClass: objectClass,
		handlers:    make([]ObjectHandler, 0),
		objects:     make(map[string]*DiscoveredObject),
	}
}

// Subscribe subscribes to an object class
func (s *Subscriber) Subscribe(ctx context.Context, attributes []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Register subscription with RTI
	if err := s.federation.subscribeObjectClass(s.objectClass, attributes); err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	return nil
}

// OnObjectDiscover registers a handler for discovered objects
func (s *Subscriber) OnObjectDiscover(handler ObjectHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers = append(s.handlers, handler)
}

// DiscoverObject handles object discovery
func (s *Subscriber) DiscoverObject(ctx context.Context, objectID, producingFederate string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already discovered
	if _, ok := s.objects[objectID]; ok {
		return nil
	}

	// Create discovered object
	obj := &DiscoveredObject{
		ID:                objectID,
		Class:             s.objectClass,
		Attributes:        make(map[string]interface{}),
		DiscoverTime:      time.Now(),
		LastUpdateTime:    time.Now(),
		ProducingFederate: producingFederate,
	}
	s.objects[objectID] = obj

	// Call handlers
	for _, handler := range s.handlers {
		if err := handler(ctx, obj); err != nil {
			return fmt.Errorf("handler failed: %w", err)
		}
	}

	return nil
}

// ReflectAttributes handles attribute reflection
func (s *Subscriber) ReflectAttributes(ctx context.Context, objectID string, attributes map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	obj, ok := s.objects[objectID]
	if !ok {
		// Object not discovered yet, create it
		obj = &DiscoveredObject{
			ID:             objectID,
			Class:          s.objectClass,
			Attributes:     make(map[string]interface{}),
			DiscoverTime:   time.Now(),
			LastUpdateTime: time.Now(),
		}
		s.objects[objectID] = obj
	}

	// Update attributes
	for name, value := range attributes {
		obj.Attributes[name] = value
	}
	obj.LastUpdateTime = time.Now()

	// Notify handlers
	for _, handler := range s.handlers {
		if err := handler(ctx, obj); err != nil {
			return fmt.Errorf("handler failed: %w", err)
		}
	}

	return nil
}

// RemoveObject handles object removal
func (s *Subscriber) RemoveObject(ctx context.Context, objectID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.objects[objectID]; !ok {
		return nil
	}

	delete(s.objects, objectID)
	return nil
}

// GetDiscoveredObjects returns all discovered objects
func (s *Subscriber) GetDiscoveredObjects() []*DiscoveredObject {
	s.mu.RLock()
	defer s.mu.RUnlock()

	objects := make([]*DiscoveredObject, 0, len(s.objects))
	for _, obj := range s.objects {
		objects = append(objects, obj)
	}
	return objects
}

// GetObject returns a specific discovered object
func (s *Subscriber) GetObject(objectID string) (*DiscoveredObject, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	obj, ok := s.objects[objectID]
	if !ok {
		return nil, fmt.Errorf("object %s not discovered", objectID)
	}
	return obj, nil
}

// RequestAttributeUpdate requests attribute update from producing federate
func (s *Subscriber) RequestAttributeUpdate(ctx context.Context, objectID string, attributes []string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	obj, ok := s.objects[objectID]
	if !ok {
		return fmt.Errorf("object %s not discovered", objectID)
	}

	// Request update via RTI
	return s.federation.requestAttributeUpdate(objectID, obj.ProducingFederate, attributes)
}

// Unsubscribe unsubscribes from the object class
func (s *Subscriber) Unsubscribe(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Unsubscribe via RTI
	if err := s.federation.unsubscribeObjectClass(s.objectClass); err != nil {
		return fmt.Errorf("failed to unsubscribe: %w", err)
	}

	// Clear objects
	s.objects = make(map[string]*DiscoveredObject)

	return nil
}

// AttributeSubscription handles attribute-level subscription
type AttributeSubscription struct {
	mu          sync.RWMutex
	federation  *Federation
	objectClass string
	attributes  map[string]bool
}

// NewAttributeSubscription creates attribute subscription
func NewAttributeSubscription(federation *Federation, objectClass string) *AttributeSubscription {
	return &AttributeSubscription{
		federation:  federation,
		objectClass: objectClass,
		attributes:  make(map[string]bool),
	}
}

// AddAttribute adds an attribute to the subscription
func (as *AttributeSubscription) AddAttribute(ctx context.Context, name string) error {
	as.mu.Lock()
	defer as.mu.Unlock()

	as.attributes[name] = true
	return nil
}

// RemoveAttribute removes an attribute from the subscription
func (as *AttributeSubscription) RemoveAttribute(ctx context.Context, name string) error {
	as.mu.Lock()
	defer as.mu.Unlock()

	delete(as.attributes, name)
	return nil
}

// GetSubscribedAttributes returns subscribed attributes
func (as *AttributeSubscription) GetSubscribedAttributes() []string {
	as.mu.RLock()
	defer as.mu.RUnlock()

	attrs := make([]string, 0, len(as.attributes))
	for attr := range as.attributes {
		attrs = append(attrs, attr)
	}
	return attrs
}
