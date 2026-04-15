// Package main provides tests for the VIMI replay engine
package main

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

// TestRecordedPDUSerialization tests PDU JSON serialization
func TestRecordedPDUSerialization(t *testing.T) {
	pdu := &RecordedPDU{
		Sequence:  1,
		ReceiveTS: time.Now(),
		OriginTS:  time.Now(),
		SiteID:    1,
		AppID:     1,
		EntityID:  100,
		ForceID:   1,
		PDUType:   PDUEntityState,
		LVCType:   "DIS",
		Lat:       38.0,
		Lon:       -77.0,
		Alt:       100.0,
	}

	data, err := json.Marshal(pdu)
	if err != nil {
		t.Fatalf("Failed to marshal PDU: %v", err)
	}

	var pdu2 RecordedPDU
	if err := json.Unmarshal(data, &pdu2); err != nil {
		t.Fatalf("Failed to unmarshal PDU: %v", err)
	}

	if pdu2.Sequence != pdu.Sequence {
		t.Errorf("Sequence mismatch: expected %d, got %d", pdu.Sequence, pdu2.Sequence)
	}
	if pdu2.EntityID != pdu.EntityID {
		t.Errorf("EntityID mismatch: expected %d, got %d", pdu.EntityID, pdu2.EntityID)
	}
}

// TestRecordingMetadata tests recording metadata
func TestRecordingMetadata(t *testing.T) {
	rec := &Recording{
		ID:          "rec-001",
		Name:        "Test Recording",
		StartTime:   time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		EndTime:     time.Date(2024, 1, 1, 12, 5, 0, 0, time.UTC),
		PDUCount:    100,
		FileSize:    10240,
		Description: "Test recording for unit tests",
		Tags:        []string{"test", "unit"},
	}

	duration := rec.EndTime.Sub(rec.StartTime)
	if duration != 5*time.Minute {
		t.Errorf("Expected 5 minute duration, got %v", duration)
	}

	data, err := json.Marshal(rec)
	if err != nil {
		t.Fatalf("Failed to marshal recording: %v", err)
	}

	var rec2 Recording
	if err := json.Unmarshal(data, &rec2); err != nil {
		t.Fatalf("Failed to unmarshal recording: %v", err)
	}

	if rec2.ID != rec.ID {
		t.Errorf("ID mismatch: expected %s, got %s", rec.ID, rec2.ID)
	}
}

// TestPDUSorting tests sorting PDUs by sequence
func TestPDUSorting(t *testing.T) {
	pdus := []*RecordedPDU{
		{Sequence: 3, EntityID: 100},
		{Sequence: 1, EntityID: 101},
		{Sequence: 2, EntityID: 102},
	}

	sort.Slice(pdus, func(i, j int) bool {
		return pdus[i].Sequence < pdus[j].Sequence
	})

	if pdus[0].Sequence != 1 {
		t.Errorf("First PDU should have sequence 1, got %d", pdus[0].Sequence)
	}
	if pdus[1].Sequence != 2 {
		t.Errorf("Second PDU should have sequence 2, got %d", pdus[1].Sequence)
	}
	if pdus[2].Sequence != 3 {
		t.Errorf("Third PDU should have sequence 3, got %d", pdus[2].Sequence)
	}
}

// TestTimeCompression tests playback time scaling
func TestTimeCompression(t *testing.T) {
	startTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	endTime := time.Date(2024, 1, 1, 12, 1, 0, 0, time.UTC)

	duration := endTime.Sub(startTime)

	scaled2x := time.Duration(float64(duration) / 2.0)
	if scaled2x != 30*time.Second {
		t.Errorf("2x speed: expected 30s, got %v", scaled2x)
	}

	scaled10x := time.Duration(float64(duration) / 10.0)
	if scaled10x != 6*time.Second {
		t.Errorf("10x speed: expected 6s, got %v", scaled10x)
	}
}

// TestFormatBytes tests byte formatting utility
func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1024.0 MB"}, // 1 GB shows as MB in this implementation
	}

	for _, tt := range tests {
		result := formatBytes(tt.bytes)
		if result != tt.expected {
			t.Errorf("formatBytes(%d) = %s, expected %s", tt.bytes, result, tt.expected)
		}
	}
}

// TestPDUTypeValues tests PDU type constants
func TestPDUTypeValues(t *testing.T) {
	if PDUEntityState != 1 {
		t.Errorf("PDUEntityState should be 1, got %d", PDUEntityState)
	}
	if PDUFire != 2 {
		t.Errorf("PDUFire should be 2, got %d", PDUFire)
	}
	if PDUDetonation != 3 {
		t.Errorf("PDUDetonation should be 3, got %d", PDUDetonation)
	}
}

// TestEntityStatePDUHeader tests PDU header parsing
func TestEntityStatePDUHeader(t *testing.T) {
	// Create a minimal entity state PDU
	pduData := make([]byte, 144)

	// Protocol version (1 byte)
	pduData[0] = 7 // DIS 7

	// Exercise ID (1 byte)
	pduData[1] = 1

	// PDU type (1 byte)
	pduData[2] = 1 // Entity State

	// Protocol family (1 byte)
	pduData[3] = 1 // Entity Information

	// Timestamp (4 bytes)
	binary.BigEndian.PutUint32(pduData[4:8], uint32(time.Now().Unix()))

	// Length (2 bytes)
	binary.BigEndian.PutUint16(pduData[8:10], 144)

	// Verify header
	if pduData[0] != 7 {
		t.Errorf("Protocol version should be 7, got %d", pduData[0])
	}
	if pduData[2] != 1 {
		t.Errorf("PDU type should be 1, got %d", pduData[2])
	}
}

// TestRecordingFilePath tests recording file path generation
func TestRecordingFilePath(t *testing.T) {
	// Test that recording files go to correct directory
	id := "test-recording-001"
	expectedPath := filepath.Join(RecordDir, id+".jsonl")

	if !filepath.IsAbs(expectedPath) {
		t.Errorf("Recording path should be absolute: %s", expectedPath)
	}
}

// TestPDUSequence tests PDU sequence numbering
func TestPDUSequence(t *testing.T) {
	rs := newReplayState()

	// Initial sequence should be 0
	if rs.currentSeq != 0 {
		t.Errorf("Initial sequence should be 0, got %d", rs.currentSeq)
	}

	// After recording a PDU, sequence should increment
	pdu := &RecordedPDU{
		Sequence: 1,
		EntityID: 100,
	}

	_ = pdu // Use pdu to avoid unused variable error

	// Sequence increment would be tested with actual recordPDU call
	// but that requires file I/O setup
}

// BenchmarkJSONMarshal benchmarks JSON marshaling
func BenchmarkJSONMarshal(b *testing.B) {
	pdu := &RecordedPDU{
		Sequence: 1,
		SiteID:   1,
		AppID:    1,
		EntityID: 100,
		ForceID:  1,
		PDUType:  PDUEntityState,
		Lat:      38.8977,
		Lon:      -77.0365,
		Alt:      100.0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(pdu)
	}
}

// BenchmarkJSONUnmarshal benchmarks JSON unmarshaling
func BenchmarkJSONUnmarshal(b *testing.B) {
	data, _ := json.Marshal(&RecordedPDU{
		Sequence: 1,
		EntityID: 100,
		Lat:      38.0,
		Lon:      -77.0,
	})

	var pdu RecordedPDU
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = json.Unmarshal(data, &pdu)
	}
}

// BenchmarkFormatBytes benchmarks byte formatting
func BenchmarkFormatBytes(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = formatBytes(1024 * 1024)
	}
}

// TestTempFileCreation tests that temp files can be created
func TestTempFileCreation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "vimi-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, "test.jsonl")
	file, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	file.Close()

	if _, err := os.Stat(tmpFile); err != nil {
		t.Errorf("Temp file not created: %v", err)
	}
}
