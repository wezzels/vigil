// Package hla provides High Level Architecture (HLA) RTI integration
// HLA is IEEE 1516 standard for distributed simulation
package hla

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"sync"
	"time"
)

// RTIType represents the type of RTI implementation
type RTIType int

const (
	RTIPortico RTIType = iota // Portico open-source RTI
	RTIMak                     // MAK RTI
	RTIPitch                   // Pitch RTI
)

// String returns string representation of RTI type
func (r RTIType) String() string {
	switch r {
	case RTIPortico:
		return "Portico"
	case RTIMak:
		return "MAK"
	case RTIPitch:
		return "Pitch"
	default:
		return "Unknown"
	}
}

// RTIConfig holds configuration for RTI connection
type RTIConfig struct {
	RTIType           RTIType       `json:"rti_type"`
	FederationName    string        `json:"federation_name"`
	FederateName      string        `json:"federate_name"`
	FOMFile           string        `json:"fom_file"`
	SOMFile           string        `json:"som_file"`
	RTIHost           string        `json:"rti_host"`
	RTIPort           int           `json:"rti_port"`
	ConnectionTimeout time.Duration `json:"connection_timeout"`
	EnableCallbacks   bool          `json:"enable_callbacks"`
}

// DefaultRTIConfig returns default RTI configuration
func DefaultRTIConfig() *RTIConfig {
	return &RTIConfig{
		RTIType:           RTIPortico,
		FederationName:    "VIGIL",
		FederateName:      "vigil-federate",
		FOMFile:           "RPR_FOM.xml",
		RTIHost:           "localhost",
		RTIPort:           8649,
		ConnectionTimeout: 30 * time.Second,
		EnableCallbacks:   true,
	}
}

// ObjectClassHandle represents an HLA object class handle
type ObjectClassHandle uint64

// AttributeHandle represents an HLA attribute handle
type AttributeHandle uint64

// ObjectInstanceHandle represents an HLA object instance handle
type ObjectInstanceHandle uint64

// InteractionClassHandle represents an HLA interaction class handle
type InteractionClassHandle uint64

// ParameterHandle represents an HLA parameter handle
type ParameterHandle uint64

// FederateAmbassador defines callback interface for federate
type FederateAmbassador interface {
	DiscoverObjectInstance(instance ObjectInstanceHandle, class ObjectClassHandle, name string)
	ReflectAttributeValues(instance ObjectInstanceHandle, attributes map[AttributeHandle][]byte)
	RemoveObjectInstance(instance ObjectInstanceHandle)
	ReceiveInteraction(interaction InteractionClassHandle, parameters map[ParameterHandle][]byte)
	TimeAdvanceGrant(t time.Time)
	AnnounceSynchronizationPoint(label string, tag string)
	FederationSynchronized(label string)
}

// RTIAmbassador is the main RTI ambassador implementation
type RTIAmbassador struct {
	config          *RTIConfig
	connected       bool
	federationName  string
	federateName    string
	timeRegulated   bool
	timeConstrained bool
	federateTime    time.Time
	lookahead       time.Duration
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	callbacks       FederateAmbassador
	objectClasses   map[string]ObjectClassHandle
	interactions    map[string]InteractionClassHandle
	attributes      map[string]AttributeHandle
	parameters      map[string]ParameterHandle
}

// NewRTIAmbassador creates a new RTI ambassador
func NewRTIAmbassador(config *RTIConfig) *RTIAmbassador {
	if config == nil {
		config = DefaultRTIConfig()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &RTIAmbassador{
		config:        config,
		ctx:           ctx,
		cancel:        cancel,
		objectClasses: make(map[string]ObjectClassHandle),
		interactions:  make(map[string]InteractionClassHandle),
		attributes:    make(map[string]AttributeHandle),
		parameters:    make(map[string]ParameterHandle),
	}
}

// SetCallbacks sets the federate ambassador callbacks
func (r *RTIAmbassador) SetCallbacks(callbacks FederateAmbassador) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.callbacks = callbacks
}

// CreateFederation creates a new federation
func (r *RTIAmbassador) CreateFederation(name string, fomFiles []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if r.connected {
		return ErrFederationExists
	}
	
	for _, file := range fomFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			return fmt.Errorf("FOM file not found: %s", file)
		}
	}
	
	r.federationName = name
	r.connected = true
	
	return nil
}

// DestroyFederation destroys the federation
func (r *RTIAmbassador) DestroyFederation(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if !r.connected {
		return ErrNotConnected
	}
	
	if r.federationName != name {
		return ErrInvalidFederation
	}
	
	r.connected = false
	r.federationName = ""
	return nil
}

// JoinFederation joins an existing federation
func (r *RTIAmbassador) JoinFederation(name string, federateName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if r.connected {
		return ErrAlreadyConnected
	}
	
	r.federationName = name
	r.federateName = federateName
	r.connected = true
	r.federateTime = time.Time{}
	
	return nil
}

// ResignFederation resigns from the federation
func (r *RTIAmbassador) ResignFederation() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if !r.connected {
		return ErrNotConnected
	}
	
	r.connected = false
	r.timeRegulated = false
	r.timeConstrained = false
	return nil
}

// RegisterObjectClass registers an object class
func (r *RTIAmbassador) RegisterObjectClass(name string) (ObjectClassHandle, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if !r.connected {
		return 0, ErrNotConnected
	}
	
	handle := ObjectClassHandle(len(r.objectClasses) + 1)
	r.objectClasses[name] = handle
	
	return handle, nil
}

// PublishObjectClass publishes an object class
func (r *RTIAmbassador) PublishObjectClass(handle ObjectClassHandle, attributes []AttributeHandle) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if !r.connected {
		return ErrNotConnected
	}
	
	return nil
}

// SubscribeObjectClass subscribes to an object class
func (r *RTIAmbassador) SubscribeObjectClass(handle ObjectClassHandle, attributes []AttributeHandle) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if !r.connected {
		return ErrNotConnected
	}
	
	return nil
}

// RegisterObjectInstance registers a new object instance
func (r *RTIAmbassador) RegisterObjectInstance(handle ObjectClassHandle) (ObjectInstanceHandle, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if !r.connected {
		return 0, ErrNotConnected
	}
	
	instanceHandle := ObjectInstanceHandle(time.Now().UnixNano())
	
	return instanceHandle, nil
}

// UpdateAttributeValues updates attribute values for an object
func (r *RTIAmbassador) UpdateAttributeValues(instance ObjectInstanceHandle, values map[AttributeHandle][]byte) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if !r.connected {
		return ErrNotConnected
	}
	
	return nil
}

// DeleteObjectInstance deletes an object instance
func (r *RTIAmbassador) DeleteObjectInstance(instance ObjectInstanceHandle) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if !r.connected {
		return ErrNotConnected
	}
	
	return nil
}

// RegisterInteractionClass registers an interaction class
func (r *RTIAmbassador) RegisterInteractionClass(name string) (InteractionClassHandle, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if !r.connected {
		return 0, ErrNotConnected
	}
	
	handle := InteractionClassHandle(len(r.interactions) + 1)
	r.interactions[name] = handle
	
	return handle, nil
}

// PublishInteractionClass publishes an interaction class
func (r *RTIAmbassador) PublishInteractionClass(handle InteractionClassHandle) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if !r.connected {
		return ErrNotConnected
	}
	
	return nil
}

// SubscribeInteractionClass subscribes to an interaction class
func (r *RTIAmbassador) SubscribeInteractionClass(handle InteractionClassHandle) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if !r.connected {
		return ErrNotConnected
	}
	
	return nil
}

// SendInteraction sends an interaction
func (r *RTIAmbassador) SendInteraction(handle InteractionClassHandle, parameters map[ParameterHandle][]byte) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if !r.connected {
		return ErrNotConnected
	}
	
	return nil
}

// EnableTimeRegulation enables time regulation
func (r *RTIAmbassador) EnableTimeRegulation(lookahead time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if !r.connected {
		return ErrNotConnected
	}
	
	r.timeRegulated = true
	r.lookahead = lookahead
	return nil
}

// DisableTimeRegulation disables time regulation
func (r *RTIAmbassador) DisableTimeRegulation() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.timeRegulated = false
	return nil
}

// EnableTimeConstrained enables time constrained mode
func (r *RTIAmbassador) EnableTimeConstrained() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if !r.connected {
		return ErrNotConnected
	}
	
	r.timeConstrained = true
	return nil
}

// DisableTimeConstrained disables time constrained mode
func (r *RTIAmbassador) DisableTimeConstrained() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.timeConstrained = false
	return nil
}

// TimeAdvanceRequest requests a time advance
func (r *RTIAmbassador) TimeAdvanceRequest(t time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if !r.connected {
		return ErrNotConnected
	}
	
	r.federateTime = t
	if r.callbacks != nil && !r.timeRegulated {
		go r.callbacks.TimeAdvanceGrant(t)
	}
	
	return nil
}

// QueryFederateTime returns the current federate time
func (r *RTIAmbassador) QueryFederateTime() (time.Time, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if !r.connected {
		return time.Time{}, ErrNotConnected
	}
	
	return r.federateTime, nil
}

// RegisterFederationSynchronizationPoint registers a sync point
func (r *RTIAmbassador) RegisterFederationSynchronizationPoint(label string, tag string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if !r.connected {
		return ErrNotConnected
	}
	
	return nil
}

// AchieveSynchronizationPoint achieves a sync point
func (r *RTIAmbassador) AchieveSynchronizationPoint(label string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if !r.connected {
		return ErrNotConnected
	}
	
	return nil
}

// Shutdown shuts down the RTI ambassador
func (r *RTIAmbassador) Shutdown() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if r.cancel != nil {
		r.cancel()
	}
	
	r.connected = false
	return nil
}

// IsConnected returns connection status
func (r *RTIAmbassador) IsConnected() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.connected
}

// Stats returns RTI statistics
func (r *RTIAmbassador) Stats() RTIStats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	return RTIStats{
		Connected:       r.connected,
		FederationName:  r.federationName,
		FederateName:    r.federateName,
		TimeRegulated:   r.timeRegulated,
		TimeConstrained: r.timeConstrained,
		FederateTime:    r.federateTime,
		Lookahead:       r.lookahead,
		ObjectClasses:   len(r.objectClasses),
		Interactions:    len(r.interactions),
	}
}

// RTIStats holds RTI statistics
type RTIStats struct {
	Connected       bool          `json:"connected"`
	FederationName  string        `json:"federation_name"`
	FederateName    string        `json:"federate_name"`
	TimeRegulated   bool          `json:"time_regulated"`
	TimeConstrained bool          `json:"time_constrained"`
	FederateTime    time.Time     `json:"federate_time"`
	Lookahead       time.Duration `json:"lookahead"`
	ObjectClasses   int           `json:"object_classes"`
	Interactions    int           `json:"interactions"`
}

// Errors
var (
	ErrNotConnected      = &RTIError{Code: "NOT_CONNECTED", Message: "not connected to federation"}
	ErrAlreadyConnected  = &RTIError{Code: "ALREADY_CONNECTED", Message: "already connected to federation"}
	ErrFederationExists  = &RTIError{Code: "FEDERATION_EXISTS", Message: "federation already exists"}
	ErrInvalidFederation = &RTIError{Code: "INVALID_FEDERATION", Message: "invalid federation name"}
)

// RTIError represents an RTI error
type RTIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *RTIError) Error() string {
	return e.Message
}

// FOMFile represents a Federation Object Model file
type FOMFile struct {
	XMLName        xml.Name           `xml:"objectModel"`
	Name           string             `xml:"name"`
	Version        string             `xml:"version"`
	ObjectClasses  []ObjectClassFOM   `xml:"objects>objectClass"`
	Interactions   []InteractionFOM   `xml:"interactions>interactionClass"`
}

// ObjectClassFOM represents an HLA object class in FOM
type ObjectClassFOM struct {
	Name       string             `xml:"name"`
	Attributes []AttributeFOM     `xml:"attribute"`
}

// AttributeFOM represents an HLA attribute in FOM
type AttributeFOM struct {
	Name       string `xml:"name"`
	DataType   string `xml:"dataType"`
	UpdateType string `xml:"updateType"`
}

// InteractionFOM represents an HLA interaction class in FOM
type InteractionFOM struct {
	Name       string          `xml:"name"`
	Parameters []ParameterFOM `xml:"parameter"`
}

// ParameterFOM represents an HLA parameter in FOM
type ParameterFOM struct {
	Name     string `xml:"name"`
	DataType string `xml:"dataType"`
}

// ParseFOM parses a FOM XML file
func ParseFOM(filename string) (*FOMFile, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	
	var fom FOMFile
	if err := xml.Unmarshal(data, &fom); err != nil {
		return nil, err
	}
	
	return &fom, nil
}