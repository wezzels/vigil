// Package pdu provides DIS Emission PDU encoding/decoding
package pdu

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// EmissionPDU represents a DIS Emission PDU
type EmissionPDU struct {
	PDUHeader
	EmittingEntityID   EntityID
	EventID            EntityID
	StateUpdateIndicator uint8
	AttachedIndicator    uint8
	Padding             uint16
	EmitterSystems      []EmitterSystem
}

// EmitterSystem represents an emitter system
type EmitterSystem struct {
	EmitterSystemName uint16
	EmitterSystemFunction uint8
	EmitterSystemLocation Location
	EmitterBeamData     EmitterBeamData
}

// Location represents a location in DIS coordinates
type Location struct {
	X float64
	Y float64
	Z float64
}

// EmitterBeamData represents emitter beam data
type EmitterBeamData struct {
	BeamDataLength     uint8
	BeamIDNumber       uint8
	BeamParameterIndex uint16
	FundamentalParameterData FundamentalParameterData
	TrackingData       []BeamTrackData
}

// FundamentalParameterData represents fundamental parameter data
type FundamentalParameterData struct {
	Frequency         float64
	FrequencyRange    float32
EffectivePower      float32
	PulseRepetitionFrequency float32
	PulseWidth       float32
	BeamCenterAzimuth float32
	BeamCenterElevation float32
}

// BeamTrackData represents beam track data
type BeamTrackData struct {
	TrackDataLength     uint8
	TrackDataCount      uint8
	TrackJammerData     TrackJammerData
}

// TrackJammerData represents track/jammer data
type TrackJammerData struct {
	TrackID             EntityID
	JammerModulation    uint16
	JammerPower        float32
	JammerEffectiveTime uint32
	JammerFrequency    float64
}

// Encode encodes the Emission PDU to binary
func (e *EmissionPDU) Encode() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Encode header
	if err := e.PDUHeader.Encode(buf); err != nil {
		return nil, fmt.Errorf("header encode failed: %w", err)
	}

	// Encode emitting entity ID
	if err := e.EmittingEntityID.Encode(buf); err != nil {
		return nil, fmt.Errorf("emitting entity ID encode failed: %w", err)
	}

	// Encode event ID
	if err := e.EventID.Encode(buf); err != nil {
		return nil, fmt.Errorf("event ID encode failed: %w", err)
	}

	// Encode state update indicator
	if err := binary.Write(buf, binary.LittleEndian, e.StateUpdateIndicator); err != nil {
		return nil, err
	}

	// Encode attached indicator
	if err := binary.Write(buf, binary.LittleEndian, e.AttachedIndicator); err != nil {
		return nil, err
	}

	// Encode padding
	if err := binary.Write(buf, binary.LittleEndian, e.Padding); err != nil {
		return nil, err
	}

	// Encode emitter systems
	for _, es := range e.EmitterSystems {
		if err := e.encodeEmitterSystem(buf, &es); err != nil {
			return nil, fmt.Errorf("emitter system encode failed: %w", err)
		}
	}

	return buf.Bytes(), nil
}

// encodeEmitterSystem encodes an emitter system
func (e *EmissionPDU) encodeEmitterSystem(buf *bytes.Buffer, es *EmitterSystem) error {
	if err := binary.Write(buf, binary.LittleEndian, es.EmitterSystemName); err != nil {
		return err
	}

	if err := binary.Write(buf, binary.LittleEndian, es.EmitterSystemFunction); err != nil {
		return err
	}

	// Encode location
	if err := binary.Write(buf, binary.LittleEndian, es.EmitterSystemLocation.X); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, es.EmitterSystemLocation.Y); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, es.EmitterSystemLocation.Z); err != nil {
		return err
	}

	// Encode beam data
	if err := e.encodeEmitterBeamData(buf, &es.EmitterBeamData); err != nil {
		return err
	}

	return nil
}

// encodeEmitterBeamData encodes emitter beam data
func (e *EmissionPDU) encodeEmitterBeamData(buf *bytes.Buffer, bd *EmitterBeamData) error {
	if err := binary.Write(buf, binary.LittleEndian, bd.BeamDataLength); err != nil {
		return err
	}

	if err := binary.Write(buf, binary.LittleEndian, bd.BeamIDNumber); err != nil {
		return err
	}

	if err := binary.Write(buf, binary.LittleEndian, bd.BeamParameterIndex); err != nil {
		return err
	}

	// Encode fundamental parameter data
	if err := binary.Write(buf, binary.LittleEndian, bd.FundamentalParameterData.Frequency); err != nil {
		return err
	}

	if err := binary.Write(buf, binary.LittleEndian, bd.FundamentalParameterData.FrequencyRange); err != nil {
		return err
	}

	return nil
}

// Decode decodes binary data to Emission PDU
func (e *EmissionPDU) Decode(data []byte) error {
	buf := bytes.NewReader(data)

	// Decode header
	if err := e.PDUHeader.Decode(buf); err != nil {
		return fmt.Errorf("header decode failed: %w", err)
	}

	// Decode emitting entity ID
	if err := e.EmittingEntityID.Decode(buf); err != nil {
		return fmt.Errorf("emitting entity ID decode failed: %w", err)
	}

	// Decode event ID
	if err := e.EventID.Decode(buf); err != nil {
		return fmt.Errorf("event ID decode failed: %w", err)
	}

	// Decode state update indicator
	if err := binary.Read(buf, binary.LittleEndian, &e.StateUpdateIndicator); err != nil {
		return err
	}

	// Decode attached indicator
	if err := binary.Read(buf, binary.LittleEndian, &e.AttachedIndicator); err != nil {
		return err
	}

	// Decode padding
	if err := binary.Read(buf, binary.LittleEndian, &e.Padding); err != nil {
		return err
	}

	// Decode emitter systems
	// In production, would read based on length fields
	e.EmitterSystems = make([]EmitterSystem, 0)

	return nil
}

// PDUHeader methods (if not already defined)

// Encode writes header to buffer
func (h *PDUHeader) Encode(w io.Writer) error {
	if err := binary.Write(w, binary.LittleEndian, h.ProtocolVersion); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, h.ExerciseID); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, h.PDUType); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, h.ProtocolFamily); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, h.Timestamp); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, h.Length); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, h.Padding); err != nil {
		return err
	}
	return nil
}

// Decode reads header from buffer
func (h *PDUHeader) Decode(r io.Reader) error {
	if err := binary.Read(r, binary.LittleEndian, &h.ProtocolVersion); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &h.ExerciseID); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &h.PDUType); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &h.ProtocolFamily); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &h.Timestamp); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &h.Length); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &h.Padding); err != nil {
		return err
	}
	return nil
}

// EntityID methods

// Encode writes entity ID to buffer
func (e *EntityID) Encode(w io.Writer) error {
	if err := binary.Write(w, binary.LittleEndian, e.SiteID); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, e.ApplicationID); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, e.EntityIDNum); err != nil {
		return err
	}
	return nil
}

// Decode reads entity ID from buffer
func (e *EntityID) Decode(r io.Reader) error {
	if err := binary.Read(r, binary.LittleEndian, &e.SiteID); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &e.ApplicationID); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &e.EntityIDNum); err != nil {
		return err
	}
	return nil
}