// Package external provides USMTF message formatting
package external

import (
	"fmt"
	"strings"
	"time"
)

// USMTFMessage represents a USMTF message
type USMTFMessage struct {
	Header         USMTFHeader
	Body           string
	Signature      string
	Timestamp      time.Time
}

// USMTFHeader represents USMTF message header
type USMTFHeader struct {
	Originator     string
	Destination    string
	MessageType    string
	Precedence     string
	Classification string
}

// USMTFFormatter formats USMTF messages
type USMTFFormatter struct{}

// NewUSMTFFormatter creates a new formatter
func NewUSMTFFormatter() *USMTFFormatter {
	return &USMTFFormatter{}
}

// Format formats a USMTF message
func (f *USMTFFormatter) Format(msg *USMTFMessage) (string, error) {
	var sb strings.Builder

	// USMTF Header format
	sb.WriteString("USMTF\r\n")
	sb.WriteString("FROM: ")
	sb.WriteString(msg.Header.Originator)
	sb.WriteString("\r\n")
	sb.WriteString("TO: ")
	sb.WriteString(msg.Header.Destination)
	sb.WriteString("\r\n")
	sb.WriteString("TYPE: ")
	sb.WriteString(msg.Header.MessageType)
	sb.WriteString("\r\n")
	sb.WriteString("PRECEDENCE: ")
	sb.WriteString(msg.Header.Precedence)
	sb.WriteString("\r\n")
	sb.WriteString("CLASSIFICATION: ")
	sb.WriteString(msg.Header.Classification)
	sb.WriteString("\r\n")

	// Message body
	sb.WriteString("\r\n")
	sb.WriteString(msg.Body)
	sb.WriteString("\r\n")

	// Footer
	sb.WriteString("\r\n")
	sb.WriteString(msg.Timestamp.Format("020106 1504"))
	sb.WriteString("\r\n")
	sb.WriteString("ENDUSMTF\r\n")

	return sb.String(), nil
}

// Parse parses a USMTF message
func (f *USMTFFormatter) Parse(data string) (*USMTFMessage, error) {
	msg := &USMTFMessage{}
	lines := strings.Split(data, "\r\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "FROM: ") {
			msg.Header.Originator = strings.TrimPrefix(line, "FROM: ")
		} else if strings.HasPrefix(line, "TO: ") {
			msg.Header.Destination = strings.TrimPrefix(line, "TO: ")
		} else if strings.HasPrefix(line, "TYPE: ") {
			msg.Header.MessageType = strings.TrimPrefix(line, "TYPE: ")
		} else if strings.HasPrefix(line, "PRECEDENCE: ") {
			msg.Header.Precedence = strings.TrimPrefix(line, "PRECEDENCE: ")
		} else if strings.HasPrefix(line, "CLASSIFICATION: ") {
			msg.Header.Classification = strings.TrimPrefix(line, "CLASSIFICATION: ")
		}
	}

	// Extract body
	bodyStart := strings.Index(data, "\r\n\r\n") + 4
	bodyEnd := strings.Index(data, "\r\nENDUSMTF")
	if bodyStart < bodyEnd {
		msg.Body = data[bodyStart:bodyEnd]
	}

	return msg, nil
}

// Validate validates a USMTF message
func (f *USMTFFormatter) Validate(msg *USMTFMessage) error {
	if msg.Header.Originator == "" {
		return fmt.Errorf("originator required")
	}

	validPrecedence := map[string]bool{
		"FLASH":       true,
		"IMMEDIATE":   true,
		"PRIORITY":    true,
		"ROUTINE":     true,
	}

	if !validPrecedence[msg.Header.Precedence] {
		return fmt.Errorf("invalid precedence: %s", msg.Header.Precedence)
	}

	return nil
}

// ADatP3Message represents an ADatP-3 message
type ADatP3Message struct {
	Header         ADatP3Header
	Body           string
	Timestamp      time.Time
}

// ADatP3Header represents ADatP-3 message header
type ADatP3Header struct {
	Originator     string
	ReportType     string
	SecurityLevel  string
}

// ADatP3Formatter formats ADatP-3 messages
type ADatP3Formatter struct{}

// NewADatP3Formatter creates a new formatter
func NewADatP3Formatter() *ADatP3Formatter {
	return &ADatP3Formatter{}
}

// Format formats an ADatP-3 message
func (f *ADatP3Formatter) Format(msg *ADatP3Message) (string, error) {
	var sb strings.Builder

	// ADatP-3 format
	sb.WriteString("ADATP3\r\n")
	sb.WriteString("ORIGINATOR: ")
	sb.WriteString(msg.Header.Originator)
	sb.WriteString("\r\n")
	sb.WriteString("REPORTTYPE: ")
	sb.WriteString(msg.Header.ReportType)
	sb.WriteString("\r\n")
	sb.WriteString("SECURITY: ")
	sb.WriteString(msg.Header.SecurityLevel)
	sb.WriteString("\r\n\r\n")

	// Body
	sb.WriteString(msg.Body)
	sb.WriteString("\r\n")
	sb.WriteString(msg.Timestamp.Format("20060102T150405Z"))
	sb.WriteString("\r\nENDADATP3\r\n")

	return sb.String(), nil
}

// Parse parses an ADatP-3 message
func (f *ADatP3Formatter) Parse(data string) (*ADatP3Message, error) {
	msg := &ADatP3Message{}
	lines := strings.Split(data, "\r\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "ORIGINATOR: ") {
			msg.Header.Originator = strings.TrimPrefix(line, "ORIGINATOR: ")
		} else if strings.HasPrefix(line, "REPORTTYPE: ") {
			msg.Header.ReportType = strings.TrimPrefix(line, "REPORTTYPE: ")
		} else if strings.HasPrefix(line, "SECURITY: ") {
			msg.Header.SecurityLevel = strings.TrimPrefix(line, "SECURITY: ")
		}
	}

	return msg, nil
}

// Validate validates an ADatP-3 message
func (f *ADatP3Formatter) Validate(msg *ADatP3Message) error {
	if msg.Header.Originator == "" {
		return fmt.Errorf("originator required")
	}

	return nil
}