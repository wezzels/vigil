// Package tadil provides TADIL-A message formatting and parsing
package tadil

import (
	"fmt"
	"strings"
	"time"
)

// TADILAMessage represents a TADIL-A (Link-11) message
type TADILAMessage struct {
	Preamble    string
	MessageType string
	Originator  string
	Destination string
	Data        []string
	Checksum    string
	Timestamp   time.Time
}

// TADILAFormatter formats TADIL-A messages
type TADILAFormatter struct{}

// NewTADILAFormatter creates a new formatter
func NewTADILAFormatter() *TADILAFormatter {
	return &TADILAFormatter{}
}

// Format formats a TADIL-A message
func (f *TADILAFormatter) Format(msg *TADILAMessage) (string, error) {
	var sb strings.Builder

	// Preamble (5 chars)
	sb.WriteString(f.padRight(msg.Preamble, 5))

	// Message type (3 chars)
	sb.WriteString(f.padRight(msg.MessageType, 3))

	// Originator (5 chars)
	sb.WriteString(f.padRight(msg.Originator, 5))

	// Destination (5 chars)
	sb.WriteString(f.padRight(msg.Destination, 5))

	// Data fields (variable)
	for _, field := range msg.Data {
		sb.WriteString(f.padRight(field, 8))
	}

	// Timestamp
	sb.WriteString(msg.Timestamp.Format("150405"))

	// Checksum
	sb.WriteString(f.calculateChecksum(sb.String()))

	return sb.String(), nil
}

// Parse parses a TADIL-A message
func (f *TADILAFormatter) Parse(data string) (*TADILAMessage, error) {
	if len(data) < 23 {
		return nil, fmt.Errorf("message too short: %d bytes", len(data))
	}

	msg := &TADILAMessage{}

	// Extract fixed fields
	msg.Preamble = strings.TrimSpace(data[0:5])
	msg.MessageType = strings.TrimSpace(data[5:8])
	msg.Originator = strings.TrimSpace(data[8:13])
	msg.Destination = strings.TrimSpace(data[13:18])

	// Extract data fields (remaining data minus timestamp and checksum)
	remaining := data[18:]
	if len(remaining) > 12 {
		dataPortion := remaining[:len(remaining)-12]
		msg.Data = f.extractFields(dataPortion, 8)

		// Extract timestamp
		tsStr := remaining[len(remaining)-12 : len(remaining)-6]
		ts, err := time.Parse("150405", tsStr)
		if err == nil {
			msg.Timestamp = ts
		}

		// Extract checksum
		msg.Checksum = remaining[len(remaining)-6:]
	}

	return msg, nil
}

// extractFields extracts fixed-width fields
func (f *TADILAFormatter) extractFields(data string, width int) []string {
	fields := make([]string, 0)
	for i := 0; i < len(data); i += width {
		end := i + width
		if end > len(data) {
			end = len(data)
		}
		fields = append(fields, strings.TrimSpace(data[i:end]))
	}
	return fields
}

// padRight pads a string to fixed width
func (f *TADILAFormatter) padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

// calculateChecksum calculates message checksum
func (f *TADILAFormatter) calculateChecksum(data string) string {
	sum := 0
	for _, c := range data {
		sum += int(c)
	}
	return fmt.Sprintf("%06X", sum%0xFFFFFF)
}

// Validate validates a TADIL-A message
func (f *TADILAFormatter) Validate(msg *TADILAMessage) error {
	if msg.Preamble == "" {
		return fmt.Errorf("preamble required")
	}

	if msg.MessageType == "" {
		return fmt.Errorf("message type required")
	}

	validTypes := map[string]bool{
		"OPR": true,
		"TRK": true,
		"COR": true,
		"WNG": true,
		"ALR": true,
	}

	if !validTypes[msg.MessageType] {
		return fmt.Errorf("invalid message type: %s", msg.MessageType)
	}

	return nil
}

// TADILAEncoder encodes TADIL-A messages to binary
type TADILAEncoder struct{}

// NewTADILAEncoder creates a new encoder
func NewTADILAEncoder() *TADILAEncoder {
	return &TADILAEncoder{}
}

// Encode encodes message to binary
func (e *TADILAEncoder) Encode(msg *TADILAMessage) ([]byte, error) {
	formatter := NewTADILAFormatter()
	str, err := formatter.Format(msg)
	if err != nil {
		return nil, err
	}

	// Convert to binary with 6-bit encoding
	data := make([]byte, 0)
	for _, c := range str {
		data = append(data, byte(c))
	}

	return data, nil
}

// Decode decodes binary to message
func (e *TADILAEncoder) Decode(data []byte) (*TADILAMessage, error) {
	str := string(data)
	formatter := NewTADILAFormatter()
	return formatter.Parse(str)
}
