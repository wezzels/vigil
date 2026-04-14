// Package external provides external system interfaces
package external

import (
	"fmt"
	"strings"
	"time"
)

// JTAGSMessage represents a JTAGS message
type JTAGSMessage struct {
	MessageType    string
	Priority       string
	Originator     string
	TrackData      JTAGSTrackData
	Timestamp      time.Time
}

// JTAGSTrackData represents track data in JTAGS format
type JTAGSTrackData struct {
	TrackNumber    string
	Latitude       float64
	Longitude      float64
	Altitude       float64
	VelocityKts    float64
	Heading        float64
	TrackQuality   int
}

// JTAGSFormatter formats JTAGS messages
type JTAGSFormatter struct{}

// NewJTAGSFormatter creates a new formatter
func NewJTAGSFormatter() *JTAGSFormatter {
	return &JTAGSFormatter{}
}

// Format formats a JTAGS message
func (f *JTAGSFormatter) Format(msg *JTAGSMessage) (string, error) {
	var sb strings.Builder

	// Header
	sb.WriteString("JTAGS/")
	sb.WriteString(msg.MessageType)
	sb.WriteString("/")
	sb.WriteString(msg.Priority)
	sb.WriteString("/")

	// Track data
	sb.WriteString(msg.TrackData.TrackNumber)
	sb.WriteString(",")
	sb.WriteString(fmt.Sprintf("%.6f", msg.TrackData.Latitude))
	sb.WriteString(",")
	sb.WriteString(fmt.Sprintf("%.6f", msg.TrackData.Longitude))
	sb.WriteString(",")
	sb.WriteString(fmt.Sprintf("%.1f", msg.TrackData.Altitude))
	sb.WriteString(",")
	sb.WriteString(fmt.Sprintf("%.1f", msg.TrackData.VelocityKts))
	sb.WriteString(",")
	sb.WriteString(fmt.Sprintf("%.1f", msg.TrackData.Heading))
	sb.WriteString(",")
	sb.WriteString(fmt.Sprintf("%d", msg.TrackData.TrackQuality))
	sb.WriteString("/")

	// Timestamp
	sb.WriteString(msg.Timestamp.Format("20060102T150405Z"))

	return sb.String(), nil
}

// Parse parses a JTAGS message
func (f *JTAGSFormatter) Parse(data string) (*JTAGSMessage, error) {
	parts := strings.Split(data, "/")
	if len(parts) < 5 {
		return nil, fmt.Errorf("invalid JTAGS message format")
	}

	msg := &JTAGSMessage{
		MessageType: parts[1],
		Priority:    parts[2],
	}

	// Parse track data
	trackParts := strings.Split(parts[3], ",")
	if len(trackParts) >= 7 {
		msg.TrackData.TrackNumber = trackParts[0]
		fmt.Sscanf(trackParts[1], "%f", &msg.TrackData.Latitude)
		fmt.Sscanf(trackParts[2], "%f", &msg.TrackData.Longitude)
		fmt.Sscanf(trackParts[3], "%f", &msg.TrackData.Altitude)
		fmt.Sscanf(trackParts[4], "%f", &msg.TrackData.VelocityKts)
		fmt.Sscanf(trackParts[5], "%f", &msg.TrackData.Heading)
		fmt.Sscanf(trackParts[6], "%d", &msg.TrackData.TrackQuality)
	}

	// Parse timestamp
	if len(parts) > 4 {
		ts, err := time.Parse(time.RFC3339, parts[4])
		if err == nil {
			msg.Timestamp = ts
		}
	}

	return msg, nil
}

// JTAGSConnection handles JTAGS network connections
type JTAGSConnection struct {
	Host     string
	Port     int
	Timeout  time.Duration
	connected bool
}

// NewJTAGSConnection creates a new connection
func NewJTAGSConnection(host string, port int) *JTAGSConnection {
	return &JTAGSConnection{
		Host:    host,
		Port:    port,
		Timeout: 30 * time.Second,
	}
}

// Connect establishes connection
func (c *JTAGSConnection) Connect() error {
	// In production, would establish TCP connection
	c.connected = true
	return nil
}

// Disconnect closes connection
func (c *JTAGSConnection) Disconnect() error {
	c.connected = false
	return nil
}

// Send sends a message
func (c *JTAGSConnection) Send(msg *JTAGSMessage) error {
	if !c.connected {
		return fmt.Errorf("not connected")
	}

	// In production, would send over socket
	return nil
}

// Receive receives a message
func (c *JTAGSConnection) Receive() (*JTAGSMessage, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected")
	}

	// In production, would receive from socket
	return &JTAGSMessage{}, nil
}

// IsConnected returns connection status
func (c *JTAGSConnection) IsConnected() bool {
	return c.connected
}

// Validate validates a JTAGS message
func (f *JTAGSFormatter) Validate(msg *JTAGSMessage) error {
	validTypes := map[string]bool{
		"TRACK":    true,
		"ALERT":    true,
		"STATUS":   true,
		"CONTROL":  true,
	}

	if !validTypes[msg.MessageType] {
		return fmt.Errorf("invalid message type: %s", msg.MessageType)
	}

	return nil
}