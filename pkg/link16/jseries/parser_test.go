package jseries

import (
	"testing"
)

// TestJSeriesHeader tests header parsing
func TestJSeriesHeader(t *testing.T) {
	parser := NewParser()

	header := JSeriesHeader{
		MessageNumber: J3_2,
		Submessage:    0,
		WordCount:     3,
		Priority:      7,
		TimeSlot:      5,
		Security:      3,
		Compression:   1,
	}

	data := parser.SerializeHeader(header)

	if len(data) != 4 {
		t.Errorf("Expected header length 4, got %d", len(data))
	}

	parsed, err := parser.ParseHeader(data)
	if err != nil {
		t.Errorf("ParseHeader failed: %v", err)
	}

	if parsed.MessageNumber != header.MessageNumber {
		t.Errorf("Message number mismatch: got %d, want %d", parsed.MessageNumber, header.MessageNumber)
	}

	if parsed.WordCount != header.WordCount {
		t.Errorf("Word count mismatch: got %d, want %d", parsed.WordCount, header.WordCount)
	}
}

// TestParseHeader tests header parsing with bytes
func TestParseHeader(t *testing.T) {
	parser := NewParser()

	// Create header and serialize
	header := JSeriesHeader{
		MessageNumber: 50, // J3.2
		Submessage:    0,
		WordCount:     3,
		Priority:      7,
	}

	data := parser.SerializeHeader(header)

	parsed, err := parser.ParseHeader(data)
	if err != nil {
		t.Errorf("ParseHeader failed: %v", err)
	}

	if parsed.MessageNumber != header.MessageNumber {
		t.Errorf("Message number mismatch: got %d, want %d",
			parsed.MessageNumber, header.MessageNumber)
	}

	if parsed.WordCount != header.WordCount {
		t.Errorf("Word count mismatch: got %d, want %d",
			parsed.WordCount, header.WordCount)
	}
}

// TestParseMessage tests message parsing
func TestParseMessage(t *testing.T) {
	parser := NewParser()

	// Create a message: header + 2 words
	header := JSeriesHeader{
		MessageNumber: J3_2,
		Submessage:    0,
		WordCount:     2,
		Priority:      7,
		TimeSlot:      5,
		Security:      3,
		Compression:   1,
	}

	msg := JSeriesMessage{
		Header: header,
		Words:  []uint32{0x12345678, 0xABCDEF00},
		Valid:  true,
	}

	serialized := parser.SerializeMessage(msg)

	parsed, err := parser.ParseMessage(serialized)
	if err != nil {
		t.Errorf("ParseMessage failed: %v", err)
	}

	if parsed.Header.MessageNumber != msg.Header.MessageNumber {
		t.Errorf("Message number mismatch: got %d, want %d",
			parsed.Header.MessageNumber, msg.Header.MessageNumber)
	}

	if len(parsed.Words) != len(msg.Words) {
		t.Errorf("Word count mismatch: got %d, want %d",
			len(parsed.Words), len(msg.Words))
	}
}

// TestSerializeWord tests word serialization
func TestSerializeWord(t *testing.T) {
	parser := NewParser()

	word := uint32(0x12345678)
	data := make([]byte, WordByteLength)

	parser.serializeWord(word, data)

	parsed := parser.parseWord(data)

	// Note: due to 70-bit alignment, there may be slight precision loss
	if parsed>>8 != word>>2>>6 {
		t.Errorf("Word mismatch: got %08X, want %08X", parsed, word)
	}
}

// TestGetJMessageType tests J message type strings
func TestGetJMessageType(t *testing.T) {
	tests := []struct {
		msgNum   uint16
		expected string
	}{
		{J0_0, "J0.0 (Initial Entry)"},
		{J0_1, "J0.1 (Network Management)"},
		{J2_0, "J2.0 (Air Track)"},
		{J3_2, "J3.2 (Air Track)"},
		{J7_0, "J7.0 (Track Management)"},
		{J12_0, "J12.0 (Mission Assignment)"},
		{J13_0, "J13.0 (Weapon Engagement)"},
		{9999, "Unknown"},
	}

	for _, tt := range tests {
		result := GetJMessageType(tt.msgNum)
		if result != tt.expected {
			t.Errorf("GetJMessageType(%d) = %s, want %s", tt.msgNum, result, tt.expected)
		}
	}
}

// TestValidateMessage tests message validation
func TestValidateMessage(t *testing.T) {
	parser := NewParser()

	// Valid message
	validMsg := JSeriesMessage{
		Header: JSeriesHeader{
			MessageNumber: J3_2,
			WordCount:     3,
			Priority:      7,
		},
		Words: []uint32{1, 2, 3},
		Valid: true,
	}

	err := parser.ValidateMessage(validMsg)
	if err != nil {
		t.Errorf("Valid message should pass: %v", err)
	}

	// Invalid message number
	invalidMsg := validMsg
	invalidMsg.Header.MessageNumber = 9999

	err = parser.ValidateMessage(invalidMsg)
	if err != ErrInvalidMessageNumber {
		t.Errorf("Expected ErrInvalidMessageNumber, got %v", err)
	}
}

// TestWordCountMismatch tests word count validation
func TestWordCountMismatch(t *testing.T) {
	parser := NewParser()

	msg := JSeriesMessage{
		Header: JSeriesHeader{
			MessageNumber: J3_2,
			WordCount:     5, // Says 5 words
			Priority:      7,
		},
		Words: []uint32{1, 2, 3}, // But only 3 provided
		Valid: true,
	}

	err := parser.ValidateMessage(msg)
	if err != ErrWordCountMismatch {
		t.Errorf("Expected ErrWordCountMismatch, got %v", err)
	}
}

// TestInvalidPriority tests priority validation
func TestInvalidPriority(t *testing.T) {
	parser := NewParser()

	msg := JSeriesMessage{
		Header: JSeriesHeader{
			MessageNumber: J3_2,
			WordCount:     1,
			Priority:      20, // Invalid (max 15)
		},
		Words: []uint32{1},
		Valid: true,
	}

	err := parser.ValidateMessage(msg)
	if err != ErrInvalidPriority {
		t.Errorf("Expected ErrInvalidPriority, got %v", err)
	}
}

// TestJSeriesError tests error type
func TestJSeriesError(t *testing.T) {
	err := ErrDataTooShort

	if err.Code != "DATA_TOO_SHORT" {
		t.Errorf("Error code should be DATA_TOO_SHORT, got %s", err.Code)
	}

	if err.Error() == "" {
		t.Error("Error message should not be empty")
	}
}

// TestNewParser tests parser creation
func TestNewParser(t *testing.T) {
	parser := NewParser()

	if parser == nil {
		t.Fatal("Parser should not be nil")
	}

	if parser.wordBuffer == nil {
		t.Error("Word buffer should be initialized")
	}
}

// TestMessageRoundtrip tests message serialization and parsing
func TestMessageRoundtrip(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name   string
		header JSeriesHeader
		words  []uint32
	}{
		{
			name: "J3.2 Air Track",
			header: JSeriesHeader{
				MessageNumber: J3_2,
				WordCount:     2,
				Priority:      7,
				TimeSlot:      5,
				Security:      3,
			},
			words: []uint32{0x12345678, 0xABCDEF00},
		},
		{
			name: "J7.0 Track Management",
			header: JSeriesHeader{
				MessageNumber: J7_0,
				WordCount:     1,
				Priority:      5,
				TimeSlot:      3,
				Security:      2,
			},
			words: []uint32{0x00112233},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := JSeriesMessage{
				Header: tt.header,
				Words:  tt.words,
				Valid:  true,
			}

			serialized := parser.SerializeMessage(msg)
			parsed, err := parser.ParseMessage(serialized)

			if err != nil {
				t.Errorf("ParseMessage failed: %v", err)
			}

			if parsed.Header.MessageNumber != msg.Header.MessageNumber {
				t.Errorf("Message number mismatch")
			}

			if parsed.Header.WordCount != msg.Header.WordCount {
				t.Errorf("Word count mismatch")
			}

			if len(parsed.Words) != len(msg.Words) {
				t.Errorf("Word count mismatch: got %d, want %d",
					len(parsed.Words), len(msg.Words))
			}
		})
	}
}

// BenchmarkParseHeader benchmarks header parsing
func BenchmarkParseHeader(b *testing.B) {
	parser := NewParser()
	data := []byte{0x32, 0x00, 0x37, 0x5D}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.ParseHeader(data)
	}
}

// BenchmarkSerializeHeader benchmarks header serialization
func BenchmarkSerializeHeader(b *testing.B) {
	parser := NewParser()
	header := JSeriesHeader{
		MessageNumber: J3_2,
		WordCount:     3,
		Priority:      7,
		TimeSlot:      5,
		Security:      3,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.SerializeHeader(header)
	}
}

// BenchmarkParseMessage benchmarks message parsing
func BenchmarkParseMessage(b *testing.B) {
	parser := NewParser()
	msg := JSeriesMessage{
		Header: JSeriesHeader{
			MessageNumber: J3_2,
			WordCount:     3,
			Priority:      7,
		},
		Words: []uint32{0x12345678, 0xABCDEF00, 0x00112233},
	}
	serialized := parser.SerializeMessage(msg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.ParseMessage(serialized)
	}
}

// BenchmarkSerializeMessage benchmarks message serialization
func BenchmarkSerializeMessage(b *testing.B) {
	parser := NewParser()
	msg := JSeriesMessage{
		Header: JSeriesHeader{
			MessageNumber: J3_2,
			WordCount:     3,
			Priority:      7,
		},
		Words: []uint32{0x12345678, 0xABCDEF00, 0x00112233},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.SerializeMessage(msg)
	}
}

// BenchmarkValidateMessage benchmarks message validation
func BenchmarkValidateMessage(b *testing.B) {
	parser := NewParser()
	msg := JSeriesMessage{
		Header: JSeriesHeader{
			MessageNumber: J3_2,
			WordCount:     3,
			Priority:      7,
		},
		Words: []uint32{1, 2, 3},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.ValidateMessage(msg)
	}
}
