// Package tadil provides VMF (Variable Message Format) message support
package tadil

import (
	"fmt"
	"strings"
	"time"
)

// VMFMessage represents a VMF message
type VMFMessage struct {
	MessageHeader VMFHeader
	MessageBody   string
	Checksum      string
	Timestamp     time.Time
}

// VMFHeader represents VMF message header
type VMFHeader struct {
	Originator    string
	Destination   string
	MessageType   string
	Precedence    string
	SecurityLevel string
}

// VMFFormatter formats VMF messages
type VMFFormatter struct{}

// NewVMFFormatter creates a new formatter
func NewVMFFormatter() *VMFFormatter {
	return &VMFFormatter{}
}

// Format formats a VMF message
func (f *VMFFormatter) Format(msg *VMFMessage) (string, error) {
	var sb strings.Builder

	// Start delimiter
	sb.WriteString("VMF")

	// Header
	sb.WriteString(f.formatHeader(&msg.MessageHeader))

	// Body
	sb.WriteString("/")
	sb.WriteString(msg.MessageBody)

	// Timestamp
	sb.WriteString("/")
	sb.WriteString(msg.Timestamp.Format("20060102T150405Z"))

	// End delimiter and checksum
	sb.WriteString("/")
	sb.WriteString(f.calculateChecksum(sb.String()))
	sb.WriteString("/END")

	return sb.String(), nil
}

// formatHeader formats VMF header
func (f *VMFFormatter) formatHeader(header *VMFHeader) string {
	parts := []string{
		header.Originator,
		header.Destination,
		header.MessageType,
		header.Precedence,
		header.SecurityLevel,
	}
	return strings.Join(parts, ":")
}

// Parse parses a VMF message
func (f *VMFFormatter) Parse(data string) (*VMFMessage, error) {
	// Check delimiters
	if !strings.HasPrefix(data, "VMF") || !strings.HasSuffix(data, "/END") {
		return nil, fmt.Errorf("invalid VMF message format")
	}

	// Remove delimiters
	data = strings.TrimPrefix(data, "VMF")
	data = strings.TrimSuffix(data, "/END")

	// Split into parts
	parts := strings.Split(data, "/")
	if len(parts) < 3 {
		return nil, fmt.Errorf("insufficient parts in message")
	}

	msg := &VMFMessage{}

	// Parse header
	headerParts := strings.Split(parts[0], ":")
	if len(headerParts) >= 5 {
		msg.MessageHeader.Originator = headerParts[0]
		msg.MessageHeader.Destination = headerParts[1]
		msg.MessageHeader.MessageType = headerParts[2]
		msg.MessageHeader.Precedence = headerParts[3]
		msg.MessageHeader.SecurityLevel = headerParts[4]
	}

	// Parse body
	if len(parts) > 1 {
		msg.MessageBody = parts[1]
	}

	// Parse timestamp
	if len(parts) > 2 {
		ts, err := time.Parse(time.RFC3339, parts[2])
		if err == nil {
			msg.Timestamp = ts
		}
	}

	// Parse checksum
	if len(parts) > 3 {
		msg.Checksum = parts[3]
	}

	return msg, nil
}

// calculateChecksum calculates message checksum
func (f *VMFFormatter) calculateChecksum(data string) string {
	sum := 0
	for _, c := range data {
		sum += int(c)
	}
	return fmt.Sprintf("%04X", sum%0xFFFF)
}

// Validate validates a VMF message
func (f *VMFFormatter) Validate(msg *VMFMessage) error {
	if msg.MessageHeader.Originator == "" {
		return fmt.Errorf("originator required")
	}

	if msg.MessageHeader.MessageType == "" {
		return fmt.Errorf("message type required")
	}

	validPrecedence := map[string]bool{
		"FLASH":     true,
		"IMMEDIATE": true,
		"PRIORITY":  true,
		"ROUTINE":   true,
	}

	if !validPrecedence[msg.MessageHeader.Precedence] {
		return fmt.Errorf("invalid precedence: %s", msg.MessageHeader.Precedence)
	}

	return nil
}

// VMFEncoder encodes VMF messages to binary
type VMFEncoder struct{}

// NewVMFEncoder creates a new encoder
func NewVMFEncoder() *VMFEncoder {
	return &VMFEncoder{}
}

// Encode encodes message to binary
func (e *VMFEncoder) Encode(msg *VMFMessage) ([]byte, error) {
	formatter := NewVMFFormatter()
	str, err := formatter.Format(msg)
	if err != nil {
		return nil, err
	}
	return []byte(str), nil
}

// Decode decodes binary to message
func (e *VMFEncoder) Decode(data []byte) (*VMFMessage, error) {
	formatter := NewVMFFormatter()
	return formatter.Parse(string(data))
}

// VMFTrackMessage represents a VMF track message
type VMFTrackMessage struct {
	TrackNumber  string
	Position     Position3D
	Velocity     Velocity3D
	TrackQuality int
	Identity     string
}

// FormatTrack formats a VMF track report
func (f *VMFFormatter) FormatTrack(track *VMFTrackMessage) (string, error) {
	// Format as VMF Track Report
	body := fmt.Sprintf("TRK:%s,POS:%.6f,%.6f,%.1f,VEL:%.1f,%.1f,%.1f,QLT:%d,ID:%s",
		track.TrackNumber,
		track.Position.Latitude,
		track.Position.Longitude,
		track.Position.Altitude,
		track.Velocity.X,
		track.Velocity.Y,
		track.Velocity.Z,
		track.TrackQuality,
		track.Identity,
	)

	msg := &VMFMessage{
		MessageHeader: VMFHeader{
			MessageType:   "TRACK",
			Precedence:    "PRIORITY",
			SecurityLevel: "UNCLASSIFIED",
		},
		MessageBody: body,
		Timestamp:   time.Now(),
	}

	return f.Format(msg)
}

// ParseTrack parses a VMF track report
func (f *VMFFormatter) ParseTrack(data string) (*VMFTrackMessage, error) {
	msg, err := f.Parse(data)
	if err != nil {
		return nil, err
	}

	track := &VMFTrackMessage{}

	// Parse body
	body := msg.MessageBody
	parts := strings.Split(body, ",")

	for _, part := range parts {
		kv := strings.SplitN(part, ":", 2)
		if len(kv) != 2 {
			continue
		}

		switch kv[0] {
		case "TRK":
			track.TrackNumber = kv[1]
		case "POS":
			posParts := strings.Split(kv[1], ",")
			if len(posParts) >= 3 {
				fmt.Sscanf(posParts[0], "%f", &track.Position.Latitude)
				fmt.Sscanf(posParts[1], "%f", &track.Position.Longitude)
				fmt.Sscanf(posParts[2], "%f", &track.Position.Altitude)
			}
		case "VEL":
			velParts := strings.Split(kv[1], ",")
			if len(velParts) >= 3 {
				fmt.Sscanf(velParts[0], "%f", &track.Velocity.X)
				fmt.Sscanf(velParts[1], "%f", &track.Velocity.Y)
				fmt.Sscanf(velParts[2], "%f", &track.Velocity.Z)
			}
		case "QLT":
			fmt.Sscanf(kv[1], "%d", &track.TrackQuality)
		case "ID":
			track.Identity = kv[1]
		}
	}

	return track, nil
}
