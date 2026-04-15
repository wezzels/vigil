// Package external_test provides tests for external message formats
package external_test

import (
	"testing"
	"time"

	"github.com/wezzels/vigil/pkg/external"
)

// TestJTAGSFormat tests JTAGS message formatting
func TestJTAGSFormat(t *testing.T) {
	formatter := external.NewJTAGSFormatter()

	msg := &external.JTAGSMessage{
		MessageType: "TRACK",
		Priority:    "HIGH",
		Originator:  "UNIT1",
		TrackData: external.JTAGSTrackData{
			TrackNumber:  "T0001",
			Latitude:     34.0522,
			Longitude:    -118.2437,
			Altitude:     10000.0,
			VelocityKts:  500.0,
			Heading:      270.0,
			TrackQuality: 12,
		},
		Timestamp: time.Now(),
	}

	str, err := formatter.Format(msg)
	if err != nil {
		t.Fatalf("Failed to format: %v", err)
	}

	t.Logf("JTAGS message: %s", str)
}

// TestJTAGSParse tests JTAGS message parsing
func TestJTAGSParse(t *testing.T) {
	formatter := external.NewJTAGSFormatter()

	original := &external.JTAGSMessage{
		MessageType: "TRACK",
		Priority:    "HIGH",
		TrackData: external.JTAGSTrackData{
			TrackNumber: "T0001",
			Latitude:    34.0522,
			Longitude:   -118.2437,
		},
		Timestamp: time.Now(),
	}

	str, _ := formatter.Format(original)
	parsed, err := formatter.Parse(str)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if parsed.MessageType != original.MessageType {
		t.Error("Roundtrip failed")
	}
}

// TestJTAGSConnection tests JTAGS connection
func TestJTAGSConnection(t *testing.T) {
	conn := external.NewJTAGSConnection("localhost", 5000)

	if conn.IsConnected() {
		t.Error("Should not be connected initially")
	}

	err := conn.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	if !conn.IsConnected() {
		t.Error("Should be connected")
	}

	err = conn.Disconnect()
	if err != nil {
		t.Fatalf("Failed to disconnect: %v", err)
	}

	if conn.IsConnected() {
		t.Error("Should be disconnected")
	}
}

// TestUSMTFFormat tests USMTF message formatting
func TestUSMTFFormat(t *testing.T) {
	formatter := external.NewUSMTFFormatter()

	msg := &external.USMTFMessage{
		Header: external.USMTFHeader{
			Originator:     "UNIT1",
			Destination:    "UNIT2",
			MessageType:    "TRACK",
			Precedence:     "PRIORITY",
			Classification: "UNCLASSIFIED",
		},
		Body:      "Track data here",
		Timestamp: time.Now(),
	}

	str, err := formatter.Format(msg)
	if err != nil {
		t.Fatalf("Failed to format: %v", err)
	}

	t.Logf("USMTF message: %s", str)
}

// TestUSMTFParse tests USMTF message parsing
func TestUSMTFParse(t *testing.T) {
	formatter := external.NewUSMTFFormatter()

	original := &external.USMTFMessage{
		Header: external.USMTFHeader{
			Originator:     "UNIT1",
			Destination:    "UNIT2",
			MessageType:    "TRACK",
			Precedence:     "PRIORITY",
			Classification: "UNCLASSIFIED",
		},
		Body:      "Track data",
		Timestamp: time.Now(),
	}

	str, _ := formatter.Format(original)
	parsed, err := formatter.Parse(str)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if parsed.Header.Originator != original.Header.Originator {
		t.Error("Roundtrip failed")
	}
}

// TestUSMTFValidation tests USMTF validation
func TestUSMTFValidation(t *testing.T) {
	formatter := external.NewUSMTFFormatter()

	// Valid message
	validMsg := &external.USMTFMessage{
		Header: external.USMTFHeader{
			Originator: "UNIT1",
			Precedence: "PRIORITY",
		},
	}

	err := formatter.Validate(validMsg)
	if err != nil {
		t.Errorf("Valid message failed: %v", err)
	}

	// Invalid precedence
	invalidMsg := &external.USMTFMessage{
		Header: external.USMTFHeader{
			Originator: "UNIT1",
			Precedence: "INVALID",
		},
	}

	err = formatter.Validate(invalidMsg)
	if err == nil {
		t.Error("Expected error for invalid precedence")
	}
}

// TestADatP3Format tests ADatP-3 message formatting
func TestADatP3Format(t *testing.T) {
	formatter := external.NewADatP3Formatter()

	msg := &external.ADatP3Message{
		Header: external.ADatP3Header{
			Originator:    "UNIT1",
			ReportType:    "SITREP",
			SecurityLevel: "UNCLASSIFIED",
		},
		Body:      "Situation report",
		Timestamp: time.Now(),
	}

	str, err := formatter.Format(msg)
	if err != nil {
		t.Fatalf("Failed to format: %v", err)
	}

	t.Logf("ADatP-3 message: %s", str)
}

// TestADatP3Parse tests ADatP-3 message parsing
func TestADatP3Parse(t *testing.T) {
	formatter := external.NewADatP3Formatter()

	original := &external.ADatP3Message{
		Header: external.ADatP3Header{
			Originator:    "UNIT1",
			ReportType:    "SITREP",
			SecurityLevel: "UNCLASSIFIED",
		},
		Body:      "Report",
		Timestamp: time.Now(),
	}

	str, _ := formatter.Format(original)
	parsed, err := formatter.Parse(str)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if parsed.Header.Originator != original.Header.Originator {
		t.Error("Roundtrip failed")
	}
}

// TestADatP3Convenience tests convenience constructors
func TestADatP3Convenience(t *testing.T) {
	sitrep := external.NewADatP3Sitrep("UNIT1", "Situation normal")
	if sitrep.Header.ReportType != "SITREP" {
		t.Error("Expected SITREP")
	}

	trackrep := external.NewADatP3Trackrep("UNIT1", []string{"T001", "T002"})
	if trackrep.Header.ReportType != "TRACKREP" {
		t.Error("Expected TRACKREP")
	}
}

// TestADatP3Validator tests validation
func TestADatP3Validator(t *testing.T) {
	validator := external.NewADatP3Validator()

	err := validator.ValidateReportType("SITREP")
	if err != nil {
		t.Error("Valid report type should pass")
	}

	err = validator.ValidateReportType("INVALID")
	if err == nil {
		t.Error("Invalid report type should fail")
	}

	err = validator.ValidateSecurityLevel("UNCLASSIFIED")
	if err != nil {
		t.Error("Valid security level should pass")
	}

	err = validator.ValidateSecurityLevel("INVALID")
	if err == nil {
		t.Error("Invalid security level should fail")
	}
}

// TestFormatCompliance tests format compliance
func TestFormatCompliance(t *testing.T) {
	t.Run("JTAGS", func(t *testing.T) {
		formatter := external.NewJTAGSFormatter()
		msg := &external.JTAGSMessage{MessageType: "TRACK"}
		str, _ := formatter.Format(msg)
		if str == "" {
			t.Error("Format should not be empty")
		}
	})

	t.Run("USMTF", func(t *testing.T) {
		formatter := external.NewUSMTFFormatter()
		msg := &external.USMTFMessage{
			Header: external.USMTFHeader{Originator: "UNIT"},
		}
		str, _ := formatter.Format(msg)
		if !contains(str, "USMTF") {
			t.Error("Should contain USMTF marker")
		}
	})

	t.Run("ADatP3", func(t *testing.T) {
		formatter := external.NewADatP3Formatter()
		msg := &external.ADatP3Message{
			Header: external.ADatP3Header{Originator: "UNIT"},
		}
		str, _ := formatter.Format(msg)
		if !contains(str, "ADATP3") {
			t.Error("Should contain ADATP3 marker")
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) &&
			(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
				containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// BenchmarkJTAGSFormat benchmarks JTAGS formatting
func BenchmarkJTAGSFormat(b *testing.B) {
	formatter := external.NewJTAGSFormatter()
	msg := &external.JTAGSMessage{
		MessageType: "TRACK",
		Priority:    "HIGH",
		Timestamp:   time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatter.Format(msg)
	}
}

// BenchmarkUSMTFFormat benchmarks USMTF formatting
func BenchmarkUSMTFFormat(b *testing.B) {
	formatter := external.NewUSMTFFormatter()
	msg := &external.USMTFMessage{
		Header: external.USMTFHeader{
			Originator: "UNIT1",
			Precedence: "PRIORITY",
		},
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatter.Format(msg)
	}
}
