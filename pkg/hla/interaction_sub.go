// Package hla provides HLA interaction subscription support
package hla

import (
	"context"
	"fmt"
	"sync"
)

// InteractionSubscriber handles HLA interaction subscription
type InteractionSubscriber struct {
	mu               sync.RWMutex
	federation       *Federation
	interactionClass string
	handlers         []InteractionHandler
}

// InteractionHandler handles received interactions
type InteractionHandler func(ctx context.Context, interaction *Interaction) error

// NewInteractionSubscriber creates a new interaction subscriber
func NewInteractionSubscriber(federation *Federation, interactionClass string) *InteractionSubscriber {
	return &InteractionSubscriber{
		federation:       federation,
		interactionClass: interactionClass,
		handlers:         make([]InteractionHandler, 0),
	}
}

// Subscribe subscribes to an interaction class
func (is *InteractionSubscriber) Subscribe(ctx context.Context) error {
	is.mu.Lock()
	defer is.mu.Unlock()

	// Register subscription with RTI
	if err := is.federation.subscribeInteractionClass(is.interactionClass); err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	return nil
}

// OnInteraction registers a handler for interactions
func (is *InteractionSubscriber) OnInteraction(handler InteractionHandler) {
	is.mu.Lock()
	defer is.mu.Unlock()
	is.handlers = append(is.handlers, handler)
}

// ReceiveInteraction handles received interaction
func (is *InteractionSubscriber) ReceiveInteraction(ctx context.Context, interaction *Interaction) error {
	is.mu.RLock()
	defer is.mu.RUnlock()

	// Call handlers
	for _, handler := range is.handlers {
		if err := handler(ctx, interaction); err != nil {
			return fmt.Errorf("handler failed: %w", err)
		}
	}

	return nil
}

// Unsubscribe unsubscribes from the interaction class
func (is *InteractionSubscriber) Unsubscribe(ctx context.Context) error {
	is.mu.Lock()
	defer is.mu.Unlock()

	// Unsubscribe via RTI
	return is.federation.unsubscribeInteractionClass(is.interactionClass)
}

// ParameterExtractor extracts parameters from interactions
type ParameterExtractor struct {
	mu      sync.RWMutex
	formats map[string]ParameterFormat
}

// ParameterFormat defines parameter extraction format
type ParameterFormat struct {
	Name     string
	Type     string
	Offset   int
	Length   int
	Encoding string
}

// NewParameterExtractor creates a parameter extractor
func NewParameterExtractor() *ParameterExtractor {
	return &ParameterExtractor{
		formats: make(map[string]ParameterFormat),
	}
}

// RegisterFormat registers a parameter format
func (pe *ParameterExtractor) RegisterFormat(name string, format ParameterFormat) error {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	pe.formats[name] = format
	return nil
}

// Extract extracts parameters from raw interaction data
func (pe *ParameterExtractor) Extract(data []byte) (map[string]interface{}, error) {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	result := make(map[string]interface{})

	for name, format := range pe.formats {
		if format.Offset+format.Length > len(data) {
			continue
		}

		value := pe.decodeValue(data[format.Offset:format.Offset+format.Length], format.Type, format.Encoding)
		result[name] = value
	}

	return result, nil
}

// DecodeValue decodes a value from bytes
func (pe *ParameterExtractor) decodeValue(data []byte, valueType, encoding string) interface{} {
	switch valueType {
	case "int16":
		return int16(data[0])<<8 | int16(data[1])
	case "int32":
		return int32(data[0])<<24 | int32(data[1])<<16 | int32(data[2])<<8 | int32(data[3])
	case "float32":
		// IEEE 754 encoding
		bits := uint32(data[0])<<24 | uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3])
		return float32(bits)
	case "float64":
		// IEEE 754 encoding
		bits := uint64(data[0])<<56 | uint64(data[1])<<48 | uint64(data[2])<<40 | uint64(data[3])<<32 |
			uint64(data[4])<<24 | uint64(data[5])<<16 | uint64(data[6])<<8 | uint64(data[7])
		return float64(bits)
	case "string":
		return string(data)
	default:
		return data
	}
}

// EncodeValue encodes a value to bytes
func (pe *ParameterExtractor) EncodeValue(value interface{}, valueType string) ([]byte, error) {
	switch valueType {
	case "int16":
		v, ok := value.(int16)
		if !ok {
			return nil, fmt.Errorf("invalid type for int16")
		}
		return []byte{byte(v >> 8), byte(v)}, nil
	case "int32":
		v, ok := value.(int32)
		if !ok {
			return nil, fmt.Errorf("invalid type for int32")
		}
		return []byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}, nil
	case "string":
		v, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("invalid type for string")
		}
		return []byte(v), nil
	default:
		return nil, fmt.Errorf("unknown type: %s", valueType)
	}
}

// Add federation methods for interaction subscription
func (f *Federation) subscribeInteractionClass(interactionClass string) error {
	// In production, call RTIambassador.subscribeInteractionClass
	return nil
}

func (f *Federation) unsubscribeInteractionClass(interactionClass string) error {
	// In production, call RTIambassador.unsubscribeInteractionClass
	return nil
}

// InteractionReceivedCallback handles interaction received callback
func (f *Federation) InteractionReceivedCallback(ctx context.Context, interaction *Interaction) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.callbacks.OnInteractionReceive != nil {
		f.callbacks.OnInteractionReceive(ctx, interaction)
	}
}
