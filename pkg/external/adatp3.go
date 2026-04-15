// Package external provides ADatP-3 message formatting
package external

// ADatP3Message and formatter already defined in usmtf.go
// This file provides additional ADatP-3 specific functionality

import (
	"fmt"
	"strings"
	"time"
)

// ADatP3ReportType defines ADatP-3 report types
type ADatP3ReportType string

const (
	ADatP3Sitrep   ADatP3ReportType = "SITREP"
	ADatP3Intelrep ADatP3ReportType = "INTREP"
	ADatP3Opsrep   ADatP3ReportType = "OPSREP"
	ADatP3Trackrep ADatP3ReportType = "TRACKREP"
)

// NewADatP3Sitrep creates a SITREP message
func NewADatP3Sitrep(originator string, body string) *ADatP3Message {
	return &ADatP3Message{
		Header: ADatP3Header{
			Originator:    originator,
			ReportType:    string(ADatP3Sitrep),
			SecurityLevel: "UNCLASSIFIED",
		},
		Body:      body,
		Timestamp: time.Now(),
	}
}

// NewADatP3Trackrep creates a TRACKREP message
func NewADatP3Trackrep(originator string, tracks []string) *ADatP3Message {
	return &ADatP3Message{
		Header: ADatP3Header{
			Originator:    originator,
			ReportType:    string(ADatP3Trackrep),
			SecurityLevel: "UNCLASSIFIED",
		},
		Body:      strings.Join(tracks, "\n"),
		Timestamp: time.Now(),
	}
}

// ADatP3Parser provides parsing utilities
type ADatP3Parser struct{}

// NewADatP3Parser creates a new parser
func NewADatP3Parser() *ADatP3Parser {
	return &ADatP3Parser{}
}

// ParseTrackData parses track data from ADatP-3 message
func (p *ADatP3Parser) ParseTrackData(body string) ([]TrackReport, error) {
	tracks := make([]TrackReport, 0)
	lines := strings.Split(body, "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "TRACK:") {
			track, err := p.parseTrackLine(line)
			if err != nil {
				continue
			}
			tracks = append(tracks, track)
		}
	}

	return tracks, nil
}

// parseTrackLine parses a single track line
func (p *ADatP3Parser) parseTrackLine(line string) (TrackReport, error) {
	track := TrackReport{}
	parts := strings.Split(line, ",")

	if len(parts) < 5 {
		return track, fmt.Errorf("insufficient track data")
	}

	// Extract track number
	if strings.HasPrefix(parts[0], "TRACK:") {
		track.TrackNumber = strings.TrimSpace(strings.TrimPrefix(parts[0], "TRACK:"))
	}

	return track, nil
}

// TrackReport represents a track report
type TrackReport struct {
	TrackNumber string
	Latitude    float64
	Longitude   float64
	Altitude    float64
}

// ADatP3Validator provides validation
type ADatP3Validator struct{}

// NewADatP3Validator creates a new validator
func NewADatP3Validator() *ADatP3Validator {
	return &ADatP3Validator{}
}

// ValidateReportType validates report type
func (v *ADatP3Validator) ValidateReportType(reportType string) error {
	validTypes := map[string]bool{
		"SITREP":   true,
		"INTREP":   true,
		"OPSREP":   true,
		"TRACKREP": true,
	}

	if !validTypes[reportType] {
		return fmt.Errorf("invalid report type: %s", reportType)
	}

	return nil
}

// ValidateSecurityLevel validates security level
func (v *ADatP3Validator) ValidateSecurityLevel(level string) error {
	validLevels := map[string]bool{
		"UNCLASSIFIED": true,
		"CONFIDENTIAL": true,
		"SECRET":       true,
		"TOP SECRET":   true,
	}

	if !validLevels[level] {
		return fmt.Errorf("invalid security level: %s", level)
	}

	return nil
}
