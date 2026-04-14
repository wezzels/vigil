package jreap

import (
	"testing"
)

// TestJREAPType tests JREAP type string representation
func TestJREAPType(t *testing.T) {
	tests := []struct {
		jreapType JREAPType
		want      string
	}{
		{JREAPTypeA, "JREAP-A"},
		{JREAPTypeB, "JREAP-B"},
		{JREAPTypeC, "JREAP-C"},
	}
	
	for _, tt := range tests {
		if got := tt.jreapType.String(); got != tt.want {
			t.Errorf("JREAPType(%d).String() = %s, want %s", tt.jreapType, got, tt.want)
		}
	}
}

// TestDefaultJREAPConfig tests default configuration
func TestDefaultJREAPConfig(t *testing.T) {
	config := DefaultJREAPConfig()
	
	if config.Type != JREAPTypeB {
		t.Errorf("Expected JREAP-B, got %v", config.Type)
	}
	if config.Port != 15000 {
		t.Errorf("Expected port 15000, got %d", config.Port)
	}
	if config.BufferSize != 65536 {
		t.Errorf("Expected buffer size 65536, got %d", config.BufferSize)
	}
}

// TestNewJREAPBridge tests bridge creation
func TestNewJREAPBridge(t *testing.T) {
	bridge := NewJREAPBridge(nil)
	
	if bridge == nil {
		t.Fatal("Bridge should not be nil")
	}
	
	if bridge.config.Type != JREAPTypeB {
		t.Error("Default type should be JREAP-B")
	}
}

// TestCalculateChecksum tests checksum calculation
func TestCalculateChecksum(t *testing.T) {
	// Test empty data
	result := CalculateChecksum([]byte{})
	if result != 0xFFFF {
		t.Errorf("CalculateChecksum(empty) = %04X, want FFFF", result)
	}
	
	// Test simple data - checksum is XOR of complement
	// The calculation sums bytes and XORs with 0xFFFF
	data := []byte{0x01, 0x02, 0x03}
	result = CalculateChecksum(data)
	// Sum = 0x01 + 0x02 + 0x03 = 0x06
	// XOR with 0xFFFF = 0xFFF9
	if result != 0xFFF9 {
		t.Errorf("CalculateChecksum([1,2,3]) = %04X, want FFF9", result)
	}
}

// TestBuildMessage tests message building
func TestBuildMessage(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04}
	msg := BuildMessage(data)
	
	if msg.Header.SyncByte != 0x55 {
		t.Errorf("Expected sync byte 0x55, got %02X", msg.Header.SyncByte)
	}
	
	if msg.Header.Version != 0x01 {
		t.Errorf("Expected version 0x01, got %02X", msg.Header.Version)
	}
	
	if msg.Header.MessageLength != 4 {
		t.Errorf("Expected message length 4, got %d", msg.Header.MessageLength)
	}
	
	if len(msg.Data) != 4 {
		t.Errorf("Expected data length 4, got %d", len(msg.Data))
	}
	
	if !msg.Valid {
		t.Error("Message should be valid")
	}
}

// TestSerializeMessage tests message serialization
func TestSerializeMessage(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04}
	msg := BuildMessage(data)
	
	serialized := msg.Serialize()
	
	if len(serialized) != 12 { // 8 header + 4 data
		t.Errorf("Expected serialized length 12, got %d", len(serialized))
	}
	
	if serialized[0] != 0x55 {
		t.Errorf("Expected sync byte at position 0, got %02X", serialized[0])
	}
	
	if serialized[1] != 0x01 {
		t.Errorf("Expected version at position 1, got %02X", serialized[1])
	}
}

// TestParseMessage tests message parsing
func TestParseMessage(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04}
	msg := BuildMessage(data)
	serialized := msg.Serialize()
	
	parsed, remaining, err := ParseMessage(serialized)
	if err != nil {
		t.Errorf("ParseMessage failed: %v", err)
	}
	
	if len(remaining) != 0 {
		t.Errorf("Expected no remaining data, got %d bytes", len(remaining))
	}
	
	if parsed.Header.SyncByte != 0x55 {
		t.Errorf("Expected sync byte 0x55, got %02X", parsed.Header.SyncByte)
	}
	
	if parsed.Header.MessageLength != 4 {
		t.Errorf("Expected message length 4, got %d", parsed.Header.MessageLength)
	}
}

// TestParseMessageTooShort tests parsing with insufficient data
func TestParseMessageTooShort(t *testing.T) {
	data := []byte{0x55, 0x01, 0x00}
	
	_, _, err := ParseMessage(data)
	if err != ErrMessageTooShort {
		t.Errorf("Expected ErrMessageTooShort, got %v", err)
	}
}

// TestParseMessageInvalidSync tests parsing with invalid sync byte
func TestParseMessageInvalidSync(t *testing.T) {
	data := []byte{0x54, 0x01, 0x00, 0x00, 0x04, 0x00, 0x00, 0xFF, 0xFF, 0x01, 0x02, 0x03, 0x04}
	
	_, _, err := ParseMessage(data)
	if err != ErrInvalidSync {
		t.Errorf("Expected ErrInvalidSync, got %v", err)
	}
}

// TestParseMultipleMessages tests parsing multiple messages
func TestParseMultipleMessages(t *testing.T) {
	bridge := NewJREAPBridge(nil)
	
	data1 := []byte{0x01, 0x02}
	data2 := []byte{0x03, 0x04}
	
	msg1 := BuildMessage(data1)
	msg2 := BuildMessage(data2)
	
	combined := append(msg1.Serialize(), msg2.Serialize()...)
	
	messages := bridge.parseMessages(combined)
	
	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}
}

// TestMessageHeader tests header fields
func TestMessageHeader(t *testing.T) {
	header := MessageHeader{
		SyncByte:      0x55,
		Version:       0x01,
		Control:       0x00,
		MessageLength: 100,
		Checksum:      0xABCD,
	}
	
	if header.SyncByte != 0x55 {
		t.Errorf("Expected sync byte 0x55, got %02X", header.SyncByte)
	}
	
	if header.MessageLength != 100 {
		t.Errorf("Expected message length 100, got %d", header.MessageLength)
	}
}

// TestJREAPBridgeStartStop tests bridge start/stop
func TestJREAPBridgeStartStop(t *testing.T) {
	config := &JREAPConfig{
		Type:     JREAPTypeB,
		Address:  "127.0.0.1",
		Port:     15001,
	}
	
	bridge := NewJREAPBridge(config)
	
	err := bridge.Start()
	if err != nil {
		t.Errorf("Start failed: %v", err)
	}
	
	if !bridge.running {
		t.Error("Bridge should be running after start")
	}
	
	// Starting again should fail
	err = bridge.Start()
	if err != ErrAlreadyRunning {
		t.Errorf("Expected ErrAlreadyRunning, got %v", err)
	}
	
	err = bridge.Stop()
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}
	
	if bridge.running {
		t.Error("Bridge should not be running after stop")
	}
}

// TestJREAPBridgeStats tests statistics
func TestJREAPBridgeStats(t *testing.T) {
	bridge := NewJREAPBridge(nil)
	
	stats := bridge.Stats()
	
	if stats.MessagesReceived != 0 {
		t.Error("Initial messages received should be 0")
	}
	
	if stats.MessagesSent != 0 {
		t.Error("Initial messages sent should be 0")
	}
}

// TestJREAPBridgeSend tests sending (not connected)
func TestJREAPBridgeSend(t *testing.T) {
	bridge := NewJREAPBridge(nil)
	
	err := bridge.Send([]byte{0x01, 0x02})
	if err != ErrNotRunning {
		t.Errorf("Expected ErrNotRunning, got %v", err)
	}
}

// TestJREAPBridgeReceive tests receive channel
func TestJREAPBridgeReceive(t *testing.T) {
	bridge := NewJREAPBridge(nil)
	
	rxChan := bridge.Receive()
	if rxChan == nil {
		t.Error("Receive channel should not be nil")
	}
}

// TestJREAPBridgeErrors tests error channel
func TestJREAPBridgeErrors(t *testing.T) {
	bridge := NewJREAPBridge(nil)
	
	errChan := bridge.Errors()
	if errChan == nil {
		t.Error("Error channel should not be nil")
	}
}

// TestJREAPErrors tests error types
func TestJREAPErrors(t *testing.T) {
	errors := []*JREAPError{
		ErrAlreadyRunning,
		ErrNotRunning,
		ErrNotConnected,
		ErrInvalidType,
		ErrNotImplemented,
		ErrMessageTooShort,
		ErrInvalidSync,
	}
	
	for _, err := range errors {
		if err.Error() == "" {
			t.Errorf("Error %s should have message", err.Code)
		}
	}
}

// BenchmarkBuildMessage benchmarks message building
func BenchmarkBuildMessage(b *testing.B) {
	data := make([]byte, 256)
	for i := range data {
		data[i] = byte(i)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BuildMessage(data)
	}
}

// BenchmarkParseMessage benchmarks message parsing
func BenchmarkParseMessage(b *testing.B) {
	data := make([]byte, 256)
	for i := range data {
		data[i] = byte(i)
	}
	
	msg := BuildMessage(data)
	serialized := msg.Serialize()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseMessage(serialized)
	}
}

// BenchmarkSerialize benchmarks message serialization
func BenchmarkSerialize(b *testing.B) {
	data := make([]byte, 256)
	for i := range data {
		data[i] = byte(i)
	}
	
	msg := BuildMessage(data)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg.Serialize()
	}
}

// BenchmarkCalculateChecksum benchmarks checksum calculation
func BenchmarkCalculateChecksum(b *testing.B) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateChecksum(data)
	}
}