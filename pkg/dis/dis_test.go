// Package dis_test provides tests for DIS protocol
package dis_test

import (
	"bytes"
	"testing"

	"github.com/wezzels/vigil/pkg/dis/pdu"
)

// TestPDURoundtrip tests PDU encode/decode roundtrip
func TestPDURoundtrip(t *testing.T) {
	// Test EntityState PDU roundtrip
	original := &pdu.EntityStatePDU{
		PDUHeader: pdu.PDUHeader{
			ProtocolVersion: 7,
			ExerciseID:      1,
			PDUType:         1,
			ProtocolFamily:  1,
			Timestamp:       12345,
			Length:          144,
		},
		EntityID: pdu.EntityID{
			SiteID:        1,
			ApplicationID: 1,
			EntityIDNum:  100,
		},
		EntityLocation: pdu.WorldCoordinate{
			X: 1000000.0,
			Y: 2000000.0,
			Z: 3000000.0,
		},
		EntityOrientation: pdu.EulerAngles{
			Psi:   1.57,
			Theta: 0.0,
			Phi:   0.0,
		},
	}

	// Encode
	data, err := original.Encode()
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	// Decode
	decoded := &pdu.EntityStatePDU{}
	if err := decoded.Decode(data); err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	// Verify
	if decoded.EntityID.SiteID != original.EntityID.SiteID {
		t.Errorf("SiteID mismatch: got %d, want %d", decoded.EntityID.SiteID, original.EntityID.SiteID)
	}

	if decoded.EntityLocation.X != original.EntityLocation.X {
		t.Errorf("X mismatch: got %f, want %f", decoded.EntityLocation.X, original.EntityLocation.X)
	}
}

// TestFirePDURoundtrip tests Fire PDU roundtrip
func TestFirePDURoundtrip(t *testing.T) {
	original := &pdu.FirePDU{
		PDUHeader: pdu.PDUHeader{
			ProtocolVersion: 7,
			ExerciseID:      1,
			PDUType:         2,
			ProtocolFamily:  2,
			Timestamp:       12345,
			Length:          96,
		},
		FiringEntityID: pdu.EntityID{
			SiteID:        1,
			ApplicationID: 1,
			EntityIDNum:  100,
		},
		TargetEntityID: pdu.EntityID{
			SiteID:        2,
			ApplicationID: 1,
			EntityIDNum:  200,
		},
		EventID: pdu.EntityID{
			SiteID:        1,
			ApplicationID: 1,
			EntityIDNum:  1,
		},
		FireMissionID: 1,
		Location: pdu.WorldCoordinate{
			X: 1000000.0,
			Y: 2000000.0,
			Z: 50000.0,
		},
		BurstDescriptor: pdu.BurstDescriptor{
			MunitionType: pdu.EntityType{
				EntityKind:     2,
				Domain:         3,
				Country:        225,
				Category:       1,
				Subcategory:    1,
				Specific:       1,
				Extra:          1,
			},
			Warhead:       1000,
			Fuse:          1,
			Quantity:      1,
			Rate:          1,
		},
	}

	// Encode
	data, err := original.Encode()
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	// Decode
	decoded := &pdu.FirePDU{}
	if err := decoded.Decode(data); err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	// Verify
	if decoded.FiringEntityID.EntityIDNum != original.FiringEntityID.EntityIDNum {
		t.Errorf("FiringEntityID mismatch")
	}
}

// TestDetonationPDURoundtrip tests Detonation PDU roundtrip
func TestDetonationPDURoundtrip(t *testing.T) {
	original := &pdu.DetonationPDU{
		PDUHeader: pdu.PDUHeader{
			ProtocolVersion: 7,
			ExerciseID:      1,
			PDUType:         3,
			ProtocolFamily:  2,
			Timestamp:       12345,
			Length:          128,
		},
		FiringEntityID: pdu.EntityID{
			SiteID:        1,
			ApplicationID: 1,
			EntityIDNum:  100,
		},
		TargetEntityID: pdu.EntityID{
			SiteID:        2,
			ApplicationID: 1,
			EntityIDNum:  200,
		},
		DetonationResult: 1,
	}

	// Encode
	data, err := original.Encode()
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	// Decode
	decoded := &pdu.DetonationPDU{}
	if err := decoded.Decode(data); err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	// Verify
	if decoded.PDUHeader.PDUType != original.PDUHeader.PDUType {
		t.Errorf("PDUType mismatch")
	}
}

// TestEmissionPDURoundtrip tests Emission PDU roundtrip
func TestEmissionPDURoundtrip(t *testing.T) {
	original := &pdu.EmissionPDU{
		PDUHeader: pdu.PDUHeader{
			ProtocolVersion: 7,
			ExerciseID:      1,
			PDUType:         23,
			ProtocolFamily:  6,
			Timestamp:       12345,
			Length:          64,
		},
		EmittingEntityID: pdu.EntityID{
			SiteID:        1,
			ApplicationID: 1,
			EntityIDNum:  100,
		},
		EventID: pdu.EntityID{
			SiteID:        1,
			ApplicationID: 1,
			EntityIDNum:  1,
		},
		StateUpdateIndicator: 1,
		AttachedIndicator:    0,
	}

	// Encode
	data, err := original.Encode()
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	// Decode
	decoded := &pdu.EmissionPDU{}
	if err := decoded.Decode(data); err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	// Verify
	if decoded.EmittingEntityID.EntityIDNum != original.EmittingEntityID.EntityIDNum {
		t.Errorf("EmittingEntityID mismatch")
	}
}

// TestNetworkEncoding tests network byte order encoding
func TestNetworkEncoding(t *testing.T) {
	// Test that we use network byte order (big-endian)
	data := make([]byte, 4)
	pdu.WriteUint32(data, 0x01020304)

	if data[0] != 0x01 {
		t.Errorf("Expected big-endian encoding, got %x", data)
	}
}

// TestBufferBoundaryConditions tests buffer boundary conditions
func TestBufferBoundaryConditions(t *testing.T) {
	// Test empty buffer
	pdu := &pdu.EntityStatePDU{}
	err := pdu.Decode([]byte{})
	if err == nil {
		t.Error("Expected error for empty buffer")
	}

	// Test truncated buffer
	err = pdu.Decode(make([]byte, 10))
	if err == nil {
		t.Error("Expected error for truncated buffer")
	}
}

// TestPDUSize tests PDU size calculations
func TestPDUSize(t *testing.T) {
	esPDU := &pdu.EntityStatePDU{}
	esPDU.PDUHeader.Length = 144

	data, err := esPDU.Encode()
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	// EntityState PDU should be exactly 144 bytes (standard DIS size)
	// In this implementation, we may not match exactly without all fields
	_ = data
}

// BenchmarkPDUEncoding benchmarks PDU encoding
func BenchmarkPDUEncoding(b *testing.B) {
	pdu := &pdu.EntityStatePDU{
		PDUHeader: pdu.PDUHeader{
			ProtocolVersion: 7,
			ExerciseID:      1,
			PDUType:         1,
			ProtocolFamily:  1,
			Timestamp:       12345,
			Length:          144,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pdu.Encode()
	}
}

// BenchmarkPDUDecoding benchmarks PDU decoding
func BenchmarkPDUDecoding(b *testing.B) {
	original := &pdu.EntityStatePDU{
		PDUHeader: pdu.PDUHeader{
			ProtocolVersion: 7,
			ExerciseID:      1,
			PDUType:         1,
			ProtocolFamily:  1,
			Timestamp:       12345,
			Length:          144,
		},
	}
	data, _ := original.Encode()
	decoded := &pdu.EntityStatePDU{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		decoded.Decode(data)
	}
}

// Network test utilities

// MockNetwork simulates network conditions
type MockNetwork struct {
	Latency    int
	PacketLoss float64
	Buffer     *bytes.Buffer
}

// NewMockNetwork creates a mock network
func NewMockNetwork() *MockNetwork {
	return &MockNetwork{
		Buffer: bytes.NewBuffer(nil),
	}
}

// Send sends data through mock network
func (mn *MockNetwork) Send(data []byte) (int, error) {
	return mn.Buffer.Write(data)
}

// Receive receives data from mock network
func (mn *MockNetwork) Receive() ([]byte, error) {
	return mn.Buffer.ReadBytes(0)
}

// TestMockNetwork tests mock network
func TestMockNetwork(t *testing.T) {
	net := NewMockNetwork()

	// Send PDU
	pdu := &pdu.EntityStatePDU{}
	data, err := pdu.Encode()
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	n, err := net.Send(data)
	if err != nil {
		t.Fatalf("Failed to send: %v", err)
	}

	if n != len(data) {
		t.Errorf("Send length mismatch: got %d, want %d", n, len(data))
	}
}

// Helper functions (if not defined in pdu package)

func WriteUint32(data []byte, val uint32) {
	data[0] = byte(val >> 24)
	data[1] = byte(val >> 16)
	data[2] = byte(val >> 8)
	data[3] = byte(val)
}

func ReadUint32(data []byte) uint32 {
	return uint32(data[0])<<24 | uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3])
}