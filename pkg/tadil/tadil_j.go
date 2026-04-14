// Package tadil provides TADIL-J (Link-16) message formatting and parsing
package tadil

import (
	"encoding/binary"
	"fmt"
	"time"
)

// TADILJMessage represents a TADIL-J (Link-16) J-Series message
type TADILJMessage struct {
	MessageNumber    string
	StationNumber    uint16
	TrackNumber      string
	TrackQuality     uint8
	Position         Position3D
	Velocity         Velocity3D
	Identity         uint8
	Environment      uint8
	TimeOnTarget     time.Time
}

// Position3D represents 3D position
type Position3D struct {
	Latitude  float64
	Longitude float64
	Altitude  float64
}

// Velocity3D represents 3D velocity
type Velocity3D struct {
	X, Y, Z float64
}

// TADILJFormatter formats TADIL-J messages
type TADILJFormatter struct{}

// NewTADILJFormatter creates a new formatter
func NewTADILJFormatter() *TADILJFormatter {
	return &TADILJFormatter{}
}

// FormatJ2 formats a J2.x (Air Track) message
func (f *TADILJFormatter) FormatJ2(msg *TADILJMessage) ([]byte, error) {
	data := make([]byte, 64)

	// Message number (4 chars)
	copy(data[0:4], f.padBytes([]byte(msg.MessageNumber), 4))

	// Track number (5 chars)
	copy(data[4:9], f.padBytes([]byte(msg.TrackNumber), 5))

	// Position (24 bytes - 3 doubles)
	binary.BigEndian.PutUint64(data[9:17], uint64(msg.Position.Latitude))
	binary.BigEndian.PutUint64(data[17:25], uint64(msg.Position.Longitude))
	binary.BigEndian.PutUint64(data[25:33], uint64(msg.Position.Altitude))

	// Velocity (24 bytes - 3 doubles)
	binary.BigEndian.PutUint64(data[33:41], uint64(msg.Velocity.X))
	binary.BigEndian.PutUint64(data[41:49], uint64(msg.Velocity.Y))
	binary.BigEndian.PutUint64(data[49:57], uint64(msg.Velocity.Z))

	// Identity (1 byte)
	data[57] = msg.Identity

	// Track quality (1 byte)
	data[58] = msg.TrackQuality

	// Environment (1 byte)
	data[59] = msg.Environment

	// Checksum (4 bytes)
	checksum := f.calculateChecksum(data[:60])
	binary.BigEndian.PutUint32(data[60:64], checksum)

	return data, nil
}

// ParseJ2 parses a J2.x message
func (f *TADILJFormatter) ParseJ2(data []byte) (*TADILJMessage, error) {
	if len(data) < 64 {
		return nil, fmt.Errorf("message too short: %d bytes", len(data))
	}

	msg := &TADILJMessage{}

	// Extract fields
	msg.MessageNumber = string(f.trimBytes(data[0:4]))
	msg.TrackNumber = string(f.trimBytes(data[4:9]))

	// Position
	msg.Position.Latitude = float64(binary.BigEndian.Uint64(data[9:17]))
	msg.Position.Longitude = float64(binary.BigEndian.Uint64(data[17:25]))
	msg.Position.Altitude = float64(binary.BigEndian.Uint64(data[25:33]))

	// Velocity
	msg.Velocity.X = float64(binary.BigEndian.Uint64(data[33:41]))
	msg.Velocity.Y = float64(binary.BigEndian.Uint64(data[41:49]))
	msg.Velocity.Z = float64(binary.BigEndian.Uint64(data[49:57]))

	// Other fields
	msg.Identity = data[57]
	msg.TrackQuality = data[58]
	msg.Environment = data[59]

	// Verify checksum
	checksum := binary.BigEndian.Uint32(data[60:64])
	expected := f.calculateChecksum(data[:60])
	if checksum != expected {
		return nil, fmt.Errorf("checksum mismatch")
	}

	return msg, nil
}

// FormatJ3 formats a J3.x (Surface Track) message
func (f *TADILJFormatter) FormatJ3(msg *TADILJMessage) ([]byte, error) {
	// Similar to J2 but with surface-specific fields
	return f.FormatJ2(msg)
}

// padBytes pads byte slice to fixed length
func (f *TADILJFormatter) padBytes(data []byte, length int) []byte {
	if len(data) >= length {
		return data[:length]
	}
	result := make([]byte, length)
	copy(result, data)
	return result
}

// trimBytes trims whitespace from byte slice
func (f *TADILJFormatter) trimBytes(data []byte) []byte {
	start := 0
	end := len(data)

	for start < end && (data[start] == ' ' || data[start] == 0) {
		start++
	}

	for end > start && (data[end-1] == ' ' || data[end-1] == 0) {
		end--
	}

	return data[start:end]
}

// calculateChecksum calculates message checksum
func (f *TADILJFormatter) calculateChecksum(data []byte) uint32 {
	var sum uint32
	for _, b := range data {
		sum += uint32(b)
	}
	return sum % 0xFFFFFFFF
}

// ValidateJ2 validates a J2 message
func (f *TADILJFormatter) ValidateJ2(msg *TADILJMessage) error {
	if msg.MessageNumber == "" {
		return fmt.Errorf("message number required")
	}

	if msg.TrackNumber == "" {
		return fmt.Errorf("track number required")
	}

	if msg.TrackQuality > 15 {
		return fmt.Errorf("track quality must be 0-15")
	}

	return nil
}