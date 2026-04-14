// Package hla provides HLA (High Level Architecture) interaction support
package hla

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// InteractionPublisher handles HLA interaction publishing
type InteractionPublisher struct {
	mu              sync.RWMutex
	federation      *Federation
	interactionClass string
}

// Interaction represents an HLA interaction
type Interaction struct {
	ID         string
	Class      string
	Parameters map[string]interface{}
	Timestamp  time.Time
}

// NewInteractionPublisher creates a new interaction publisher
func NewInteractionPublisher(federation *Federation, interactionClass string) *InteractionPublisher {
	return &InteractionPublisher{
		federation:       federation,
		interactionClass: interactionClass,
	}
}

// Publish publishes an interaction
func (ip *InteractionPublisher) Publish(ctx context.Context, parameters map[string]interface{}) error {
	ip.mu.Lock()
	defer ip.mu.Unlock()

	// Create interaction
	interaction := &Interaction{
		ID:         generateID(),
		Class:      ip.interactionClass,
		Parameters: parameters,
		Timestamp:  time.Now(),
	}

	// Send via RTI
	return ip.federation.sendInteraction(interaction)
}

// PublishWithTimestamp publishes an interaction with specific timestamp
func (ip *InteractionPublisher) PublishWithTimestamp(ctx context.Context, parameters map[string]interface{}, ts time.Time) error {
	ip.mu.Lock()
	defer ip.mu.Unlock()

	interaction := &Interaction{
		ID:         generateID(),
		Class:      ip.interactionClass,
		Parameters: parameters,
		Timestamp:  ts,
	}

	return ip.federation.sendInteraction(interaction)
}

// ParameterHandling handles interaction parameters
type ParameterHandling struct {
	mu         sync.RWMutex
	parameters map[string]ParameterDefinition
}

// ParameterDefinition defines an interaction parameter
type ParameterDefinition struct {
	Name     string
	Type     string
	Required bool
	Default  interface{}
}

// NewParameterHandling creates parameter handling
func NewParameterHandling() *ParameterHandling {
	return &ParameterHandling{
		parameters: make(map[string]ParameterDefinition),
	}
}

// RegisterParameter registers a parameter definition
func (ph *ParameterHandling) RegisterParameter(name string, paramType string, required bool, defaultValue interface{}) error {
	ph.mu.Lock()
	defer ph.mu.Unlock()

	ph.parameters[name] = ParameterDefinition{
		Name:     name,
		Type:     paramType,
		Required: required,
		Default:  defaultValue,
	}
	return nil
}

// ValidateParameters validates parameters against definitions
func (ph *ParameterHandling) ValidateParameters(params map[string]interface{}) error {
	ph.mu.RLock()
	defer ph.mu.RUnlock()

	for name, def := range ph.parameters {
		if def.Required {
			if _, ok := params[name]; !ok {
				if def.Default == nil {
					return fmt.Errorf("required parameter %s missing", name)
				}
			}
		}
	}

	return nil
}

// ApplyDefaults applies default values for missing parameters
func (ph *ParameterHandling) ApplyDefaults(params map[string]interface{}) map[string]interface{} {
	ph.mu.RLock()
	defer ph.mu.RUnlock()

	result := make(map[string]interface{})
	for k, v := range params {
		result[k] = v
	}

	for name, def := range ph.parameters {
		if _, ok := result[name]; !ok && def.Default != nil {
			result[name] = def.Default
		}
	}

	return result
}

// GetParameterTypes returns parameter type definitions
func (ph *ParameterHandling) GetParameterTypes() map[string]string {
	ph.mu.RLock()
	defer ph.mu.RUnlock()

	types := make(map[string]string)
	for name, def := range ph.parameters {
		types[name] = def.Type
	}
	return types
}

// InteractionPublisherTest tests interaction publishing
func InteractionPublisherTest() {
	// Test interaction publishing
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}