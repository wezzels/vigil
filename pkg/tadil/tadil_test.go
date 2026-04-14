// Package tadil_test provides tests for TADIL message formats
package tadil_test

import (
	"testing"
	"time"

	"github.com/wezzels/vigil/pkg/tadil"
)

// TestTADILAFormat tests TADIL-A message formatting
func TestTADILAFormat(t *testing.T) {
	formatter := tadil.NewTADILAFormatter()

	msg := &tadil.TADILAMessage{
		Preamble:    "LINK1",
		MessageType: "TRK",
		Originator:  "UNIT1",
		Destination: "UNIT2",
		Data:        []string{"DATA1", "DATA2"},
		Timestamp:   time.Now(),
	}

	// Format message
	str, err := formatter.Format(msg)
	if err != nil {
		t.Fatalf("Failed to format: %v", err)
	}

	if str == "" {
		t.Error("Formatted message should not be empty")
	}

	// Verify format
	t.Logf("Formatted message: %s", str)
}

// TestTADILAParse tests TADIL-A message parsing
func TestTADILAParse(t *testing.T) {
	formatter := tadil.NewTADILAFormatter()

	// Create and format message
	original := &tadil.TADILAMessage{
		Preamble:    "LINK1",
		MessageType: "TRK",
		Originator:  "UNIT1",
		Destination: "UNIT2",
		Data:        []string{"DATA1", "DATA2"},
		Timestamp:   time.Now(),
	}

	str, err := formatter.Format(original)
	if err != nil {
		t.Fatalf("Failed to format: %v", err)
	}

	// Parse message
	parsed, err := formatter.Parse(str)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Verify
	if parsed.Preamble != original.Preamble {
		t.Errorf("Preamble mismatch: got %s, want %s", parsed.Preamble, original.Preamble)
	}

	if parsed.MessageType != original.MessageType {
		t.Errorf("MessageType mismatch: got %s, want %s", parsed.MessageType, original.MessageType)
	}
}

// TestTADILJFormatJ2 tests TADIL-J J2 message formatting
func TestTADILJFormatJ2(t *testing.T) {
	formatter := tadil.NewTADILJFormatter()

	msg := &tadil.TADILJMessage{
		MessageNumber: "J2.2",
		TrackNumber:   "T0001",
		TrackQuality:  12,
		Position: tadil.Position3D{
			Latitude:  34.0522,
			Longitude: -118.2437,
			Altitude:  10000.0,
		},
		Velocity: tadil.Velocity3D{
			X: 100.0,
			Y: 200.0,
			Z: 50.0,
		},
		Identity: 2,
	}

	data, err := formatter.FormatJ2(msg)
	if err != nil {
		t.Fatalf("Failed to format: %v", err)
	}

	if len(data) != 64 {
		t.Errorf("Expected 64 bytes, got %d", len(data))
	}
}

// TestTADILJParseJ2 tests TADIL-J J2 message parsing
func TestTADILJParseJ2(t *testing.T) {
	formatter := tadil.NewTADILJFormatter()

	original := &tadil.TADILJMessage{
		MessageNumber: "J2.2",
		TrackNumber:   "T0001",
		TrackQuality:  12,
		Position: tadil.Position3D{
			Latitude:  34.0522,
			Longitude: -118.2437,
			Altitude:  10000.0,
		},
		Velocity: tadil.Velocity3D{
			X: 100.0,
			Y: 200.0,
			Z: 50.0,
		},
		Identity: 2,
	}

	data, err := formatter.FormatJ2(original)
	if err != nil {
		t.Fatalf("Failed to format: %v", err)
	}

	parsed, err := formatter.ParseJ2(data)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Verify
	if parsed.MessageNumber != original.MessageNumber {
		t.Errorf("MessageNumber mismatch")
	}

	if parsed.TrackNumber != original.TrackNumber {
		t.Errorf("TrackNumber mismatch")
	}
}

// TestVMFFormat tests VMF message formatting
func TestVMFFormat(t *testing.T) {
	formatter := tadil.NewVMFFormatter()

	msg := &tadil.VMFMessage{
		MessageHeader: tadil.VMFHeader{
			Originator:    "UNIT1",
			Destination:   "UNIT2",
			MessageType:   "TRACK",
			Precedence:    "PRIORITY",
			SecurityLevel: "UNCLASSIFIED",
		},
		MessageBody: "Test message body",
		Timestamp:   time.Now(),
	}

	str, err := formatter.Format(msg)
	if err != nil {
		t.Fatalf("Failed to format: %v", err)
	}

	if str == "" {
		t.Error("Formatted message should not be empty")
	}

	// Verify format
	t.Logf("Formatted message: %s", str)
}

// TestVMFParse tests VMF message parsing
func TestVMFParse(t *testing.T) {
	formatter := tadil.NewVMFFormatter()

	original := &tadil.VMFMessage{
		MessageHeader: tadil.VMFHeader{
			Originator:    "UNIT1",
			Destination:   "UNIT2",
			MessageType:   "TRACK",
			Precedence:    "PRIORITY",
			SecurityLevel: "UNCLASSIFIED",
		},
		MessageBody: "Test message body",
		Timestamp:   time.Now(),
	}

	str, err := formatter.Format(original)
	if err != nil {
		t.Fatalf("Failed to format: %v", err)
	}

	parsed, err := formatter.Parse(str)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Verify
	if parsed.MessageHeader.Originator != original.MessageHeader.Originator {
		t.Errorf("Originator mismatch")
	}

	if parsed.MessageBody != original.MessageBody {
		t.Errorf("Body mismatch")
	}
}

// TestTADILAValidation tests TADIL-A validation
func TestTADILAValidation(t *testing.T) {
	formatter := tadil.NewTADILAFormatter()

	// Valid message
	validMsg := &tadil.TADILAMessage{
		Preamble:    "LINK1",
		MessageType: "TRK",
	}

	err := formatter.Validate(validMsg)
	if err != nil {
		t.Errorf("Valid message failed validation: %v", err)
	}

	// Invalid message type
	invalidMsg := &tadil.TADILAMessage{
		Preamble:    "LINK1",
		MessageType: "INVALID",
	}

	err = formatter.Validate(invalidMsg)
	if err == nil {
		t.Error("Expected error for invalid message type")
	}
}

// TestTADILJValidation tests TADIL-J validation
func TestTADILJValidation(t *testing.T) {
	formatter := tadil.NewTADILJFormatter()

	// Valid message
	validMsg := &tadil.TADILJMessage{
		MessageNumber: "J2.2",
		TrackNumber:   "T0001",
		TrackQuality:  12,
	}

	err := formatter.ValidateJ2(validMsg)
	if err != nil {
		t.Errorf("Valid message failed validation: %v", err)
	}

	// Invalid track quality
	invalidMsg := &tadil.TADILJMessage{
		MessageNumber: "J2.2",
		TrackNumber:   "T0001",
		TrackQuality:  20, // Invalid (> 15)
	}

	err = formatter.ValidateJ2(invalidMsg)
	if err == nil {
		t.Error("Expected error for invalid track quality")
	}
}

// TestVMFValidation tests VMF validation
func TestVMFValidation(t *testing.T) {
	formatter := tadil.NewVMFFormatter()

	// Valid message
	validMsg := &tadil.VMFMessage{
		MessageHeader: tadil.VMFHeader{
			Originator: "UNIT1",
			MessageType: "TRACK",
			Precedence: "PRIORITY",
		},
	}

	err := formatter.Validate(validMsg)
	if err != nil {
		t.Errorf("Valid message failed validation: %v", err)
	}

	// Invalid precedence
	invalidMsg := &tadil.VMFMessage{
		MessageHeader: tadil.VMFHeader{
			Originator: "UNIT1",
			MessageType: "TRACK",
			Precedence: "INVALID",
		},
	}

	err = formatter.Validate(invalidMsg)
	if err == nil {
		t.Error("Expected error for invalid precedence")
	}
}

// TestVMFTrackFormat tests VMF track message formatting
func TestVMFTrackFormat(t *testing.T) {
	formatter := tadil.NewVMFFormatter()

	track := &tadil.VMFTrackMessage{
		TrackNumber: "T0001",
		Position: tadil.Position3D{
			Latitude:  34.0522,
			Longitude: -118.2437,
			Altitude:  10000.0,
		},
		Velocity: tadil.Velocity3D{
			X: 100.0,
			Y: 200.0,
			Z: 50.0,
		},
		TrackQuality: 12,
		Identity:     "HOSTILE",
	}

	str, err := formatter.FormatTrack(track)
	if err != nil {
		t.Fatalf("Failed to format track: %v", err)
	}

	t.Logf("Track message: %s", str)
}

// TestRoundtrip tests full roundtrip for all message types
func TestRoundtrip(t *testing.T) {
	t.Run("TADIL-A", func(t *testing.T) {
		formatter := tadil.NewTADILAFormatter()
		original := &tadil.TADILAMessage{
			Preamble:    "LINK1",
			MessageType: "TRK",
			Originator:  "UNIT1",
			Destination: "UNIT2",
			Data:        []string{"DATA1", "DATA2"},
			Timestamp:   time.Now(),
		}

		str, _ := formatter.Format(original)
		parsed, err := formatter.Parse(str)
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		if parsed.MessageType != original.MessageType {
			t.Error("Roundtrip failed")
		}
	})

	t.Run("TADIL-J", func(t *testing.T) {
		formatter := tadil.NewTADILJFormatter()
		original := &tadil.TADILJMessage{
			MessageNumber: "J2.2",
			TrackNumber:   "T0001",
			TrackQuality:  12,
		}

		data, _ := formatter.FormatJ2(original)
		parsed, err := formatter.ParseJ2(data)
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		if parsed.TrackNumber != original.TrackNumber {
			t.Error("Roundtrip failed")
		}
	})

	t.Run("VMF", func(t *testing.T) {
		formatter := tadil.NewVMFFormatter()
		original := &tadil.VMFMessage{
			MessageHeader: tadil.VMFHeader{
				Originator: "UNIT1",
				MessageType: "TRACK",
				Precedence: "PRIORITY",
			},
			MessageBody: "Test",
			Timestamp:   time.Now(),
		}

		str, _ := formatter.Format(original)
		parsed, err := formatter.Parse(str)
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		if parsed.MessageHeader.Originator != original.MessageHeader.Originator {
			t.Error("Roundtrip failed")
		}
	})
}

// BenchmarkTADILAFormat benchmarks TADIL-A formatting
func BenchmarkTADILAFormat(b *testing.B) {
	formatter := tadil.NewTADILAFormatter()
	msg := &tadil.TADILAMessage{
		Preamble:    "LINK1",
		MessageType: "TRK",
		Originator:  "UNIT1",
		Destination: "UNIT2",
		Data:        []string{"DATA1", "DATA2"},
		Timestamp:   time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatter.Format(msg)
	}
}

// BenchmarkVMFFormat benchmarks VMF formatting
func BenchmarkVMFFormat(b *testing.B) {
	formatter := tadil.NewVMFFormatter()
	msg := &tadil.VMFMessage{
		MessageHeader: tadil.VMFHeader{
			Originator: "UNIT1",
			MessageType: "TRACK",
			Precedence: "PRIORITY",
		},
		MessageBody: "Test message",
		Timestamp:   time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatter.Format(msg)
	}
}