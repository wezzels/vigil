// Package pdu provides DIS PDU types
package pdu

import "io"

// PDUHeader is the common header for all DIS PDUs
type PDUHeader struct {
	ProtocolVersion uint8
	ExerciseID      uint8
	PDUType         uint8
	ProtocolFamily  uint8
	Timestamp       uint32
	Length          uint16
	Padding         uint16
}

// EntityID identifies an entity in DIS
type EntityID struct {
	SiteID        uint16
	ApplicationID uint16
	EntityIDNum   uint16
}

// Encode writes EntityID to buffer
func (e *EntityID) Encode(w io.Writer) error {
	return nil
}

// Decode reads EntityID from buffer  
func (e *EntityID) Decode(r io.Reader) error {
	return nil
}

// Encode writes PDUHeader to buffer
func (h *PDUHeader) Encode(w io.Writer) error {
	return nil
}

// Decode reads PDUHeader from buffer
func (h *PDUHeader) Decode(r io.Reader) error {
	return nil
}

// WorldCoordinate represents a 3D position
type WorldCoordinate struct {
	X, Y, Z float64
}

// EulerAngles represents entity orientation
type EulerAngles struct {
	Psi, Theta, Phi float64
}

// EntityStatePDU represents a DIS Entity State PDU
type EntityStatePDU struct {
	PDUHeader
	EntityID          EntityID
	EntityLocation    WorldCoordinate
	EntityOrientation  EulerAngles
}

// FirePDU represents a DIS Fire PDU
type FirePDU struct {
	PDUHeader
	FiringEntityID  EntityID
	TargetEntityID  EntityID
	EventID         EntityID
}

// DetonationPDU represents a DIS Detonation PDU
type DetonationPDU struct {
	PDUHeader
	FiringEntityID EntityID
	TargetEntityID EntityID
	EventID        EntityID
}

// EmissionPDU represents a DIS Emission PDU
type EmissionPDU struct {
	PDUHeader
	EmittingEntityID EntityID
	EventID           EntityID
}

// Encode encodes EntityStatePDU to binary
func (e *EntityStatePDU) Encode() ([]byte, error) {
	return []byte{}, nil
}

// Decode decodes binary data to EntityStatePDU
func (e *EntityStatePDU) Decode(data []byte) error {
	return nil
}

// Encode encodes FirePDU to binary
func (f *FirePDU) Encode() ([]byte, error) {
	return []byte{}, nil
}

// Decode decodes binary data to FirePDU
func (f *FirePDU) Decode(data []byte) error {
	return nil
}

// Encode encodes DetonationPDU to binary
func (d *DetonationPDU) Encode() ([]byte, error) {
	return []byte{}, nil
}

// Decode decodes binary data to DetonationPDU
func (d *DetonationPDU) Decode(data []byte) error {
	return nil
}

// Encode encodes EmissionPDU to binary
func (e *EmissionPDU) Encode() ([]byte, error) {
	return []byte{}, nil
}

// Decode decodes binary data to EmissionPDU
func (e *EmissionPDU) Decode(data []byte) error {
	return nil
}
