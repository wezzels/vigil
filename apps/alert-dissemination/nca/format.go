// Package nca provides Nuclear Control Authority message formatting
// for alert dissemination in accordance with CONOPREP protocols
package nca

import (
	"fmt"
	"time"
)

// AlertPriority defines NCA alert priority levels
type AlertPriority int

const (
	PriorityRoutine       AlertPriority = 0
	PriorityPriority      AlertPriority = 1
	PriorityFlash         AlertPriority = 2
	PriorityFlashOverride AlertPriority = 3
)

// AlertCategory defines NCA alert categories
type AlertCategory int

const (
	CategoryThreat   AlertCategory = 0
	CategoryLaunch   AlertCategory = 1
	CategoryImpact   AlertCategory = 2
	CategoryDefense  AlertCategory = 3
	CategoryExercise AlertCategory = 4
)

// AlertAction defines required actions
type AlertAction int

const (
	ActionNone     AlertAction = 0
	ActionMonitor  AlertAction = 1
	ActionPrepare  AlertAction = 2
	ActionDefend   AlertAction = 3
	ActionEvacuate AlertAction = 4
)

// CONOPREPMessage represents a CONOPREP formatted message
type CONOPREPMessage struct {
	// Header
	MessageID      string        `json:"message_id"`
	Classification string        `json:"classification"`
	Priority       AlertPriority `json:"priority"`
	Category       AlertCategory `json:"category"`
	Originator     string        `json:"originator"`
	Timestamp      time.Time     `json:"timestamp"`
	ExpiresAt      time.Time     `json:"expires_at,omitempty"`

	// Content
	Heading        string      `json:"heading"`
	Content        string      `json:"content"`
	RequiredAction AlertAction `json:"required_action"`

	// Track data (if applicable)
	TrackData *TrackSummary `json:"track_data,omitempty"`

	// Impact data (if applicable)
	ImpactData *ImpactSummary `json:"impact_data,omitempty"`

	// Authentication
	AuthenticationCode string `json:"auth_code,omitempty"`
	ReleaseAuthority   string `json:"release_authority,omitempty"`
}

// TrackSummary represents summarized track information
type TrackSummary struct {
	TrackNumber     string    `json:"track_number"`
	LaunchLocation  string    `json:"launch_location,omitempty"`
	LaunchTime      time.Time `json:"launch_time,omitempty"`
	CurrentPosition Position  `json:"current_position"`
	PredictedImpact *Position `json:"predicted_impact,omitempty"`
	Confidence      float64   `json:"confidence"`
}

// Position represents a geographic position
type Position struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude"`
}

// ImpactSummary represents impact prediction data
type ImpactSummary struct {
	Location      Position  `json:"location"`
	EstimatedTime time.Time `json:"estimated_time"`
	Confidence    float64   `json:"confidence"`
	Warnings      []string  `json:"warnings,omitempty"`
}

// IMMINENTMessage represents an IMMINENT alert
type IMMINENTMessage struct {
	MessageID       string      `json:"message_id"`
	Classification  string      `json:"classification"`
	ThreatID        string      `json:"threat_id"`
	LaunchLocation  Position    `json:"launch_location"`
	LaunchTime      time.Time   `json:"launch_time"`
	PredictedImpact Position    `json:"predicted_impact"`
	EstimatedTime   time.Time   `json:"estimated_time"`
	Confidence      float64     `json:"confidence"`
	RequiredAction  AlertAction `json:"required_action"`
	Timestamp       time.Time   `json:"timestamp"`
	ExpiresAt       time.Time   `json:"expires_at"`
}

// INCOMINGMessage represents an INCOMING alert
type INCOMINGMessage struct {
	MessageID       string        `json:"message_id"`
	Classification  string        `json:"classification"`
	ThreatID        string        `json:"threat_id"`
	ThreatType      string        `json:"threat_type"`
	CurrentPosition Position      `json:"current_position"`
	Velocity        Velocity      `json:"velocity"`
	PredictedImpact Position      `json:"predicted_impact"`
	TimeToImpact    time.Duration `json:"time_to_impact"`
	Confidence      float64       `json:"confidence"`
	RequiredAction  AlertAction   `json:"required_action"`
	Timestamp       time.Time     `json:"timestamp"`
}

// Velocity represents 3D velocity
type Velocity struct {
	Vx float64 `json:"vx"`
	Vy float64 `json:"vy"`
	Vz float64 `json:"vz"`
}

// CONOPREPFormatter formats messages to CONOPREP standard
type CONOPREPFormatter struct {
	classification   string
	originator       string
	releaseAuthority string
}

// NewCONOPREPFormatter creates a new formatter
func NewCONOPREPFormatter(classification, originator, releaseAuthority string) *CONOPREPFormatter {
	return &CONOPREPFormatter{
		classification:   classification,
		originator:       originator,
		releaseAuthority: releaseAuthority,
	}
}

// FormatThreatAlert formats a threat detection alert
func (f *CONOPREPFormatter) FormatThreatAlert(heading, content string, track *TrackSummary) *CONOPREPMessage {
	priority := PriorityPriority
	if track != nil && track.Confidence > 0.9 {
		priority = PriorityFlash
	}

	return &CONOPREPMessage{
		MessageID:        generateMessageID(),
		Classification:   f.classification,
		Priority:         priority,
		Category:         CategoryThreat,
		Originator:       f.originator,
		Timestamp:        time.Now(),
		Heading:          heading,
		Content:          content,
		RequiredAction:   ActionMonitor,
		TrackData:        track,
		ReleaseAuthority: f.releaseAuthority,
	}
}

// FormatLaunchAlert formats a launch detection alert
func (f *CONOPREPFormatter) FormatLaunchAlert(launchLocation, launchTime string, track *TrackSummary) *CONOPREPMessage {
	content := fmt.Sprintf("Launch detected from %s at %s", launchLocation, launchTime)
	if track != nil {
		content = fmt.Sprintf("Launch detected from %s at %s. Track: %s",
			launchLocation, launchTime, track.TrackNumber)
	}

	return &CONOPREPMessage{
		MessageID:        generateMessageID(),
		Classification:   f.classification,
		Priority:         PriorityFlash,
		Category:         CategoryLaunch,
		Originator:       f.originator,
		Timestamp:        time.Now(),
		Heading:          "LAUNCH DETECTED",
		Content:          content,
		RequiredAction:   ActionPrepare,
		TrackData:        track,
		ReleaseAuthority: f.releaseAuthority,
	}
}

// FormatImpactAlert formats an impact prediction alert
func (f *CONOPREPFormatter) FormatImpactAlert(impact *ImpactSummary, action AlertAction) *CONOPREPMessage {
	content := fmt.Sprintf("Impact predicted at %s", formatPosition(impact.Location))
	warnings := ""
	if len(impact.Warnings) > 0 {
		warnings = ". WARNINGS: " + impact.Warnings[0]
		for _, w := range impact.Warnings[1:] {
			warnings += ", " + w
		}
	}

	return &CONOPREPMessage{
		MessageID:        generateMessageID(),
		Classification:   f.classification,
		Priority:         PriorityFlashOverride,
		Category:         CategoryImpact,
		Originator:       f.originator,
		Timestamp:        time.Now(),
		Heading:          "IMPACT PREDICTION",
		Content:          content + warnings,
		RequiredAction:   action,
		ImpactData:       impact,
		ReleaseAuthority: f.releaseAuthority,
	}
}

// FormatDefenseAlert formats a defense action alert
func (f *CONOPREPFormatter) FormatDefenseAlert(heading, content string, action AlertAction) *CONOPREPMessage {
	return &CONOPREPMessage{
		MessageID:        generateMessageID(),
		Classification:   f.classification,
		Priority:         PriorityFlash,
		Category:         CategoryDefense,
		Originator:       f.originator,
		Timestamp:        time.Now(),
		Heading:          heading,
		Content:          content,
		RequiredAction:   action,
		ReleaseAuthority: f.releaseAuthority,
	}
}

// FormatExerciseAlert formats an exercise alert
func (f *CONOPREPFormatter) FormatExerciseAlert(heading, content string) *CONOPREPMessage {
	return &CONOPREPMessage{
		MessageID:        generateMessageID(),
		Classification:   f.classification,
		Priority:         PriorityRoutine,
		Category:         CategoryExercise,
		Originator:       f.originator,
		Timestamp:        time.Now(),
		Heading:          "EXERCISE " + heading,
		Content:          "EXERCISE EXERCISE EXERCISE - " + content,
		RequiredAction:   ActionNone,
		ReleaseAuthority: f.releaseAuthority,
	}
}

// IMMINENTFormatter formats IMMINENT messages
type IMMINENTFormatter struct {
	classification string
}

// NewIMMINENTFormatter creates a new IMMINENT formatter
func NewIMMINENTFormatter(classification string) *IMMINENTFormatter {
	return &IMMINENTFormatter{classification: classification}
}

// Format formats an IMMINENT message
func (f *IMMINENTFormatter) Format(threatID string, launchPos, impactPos Position,
	launchTime, impactTime time.Time, confidence float64) *IMMINENTMessage {

	action := ActionPrepare
	if confidence < 0.7 {
		action = ActionMonitor
	} else if confidence >= 0.95 {
		action = ActionDefend
	}

	return &IMMINENTMessage{
		MessageID:       generateMessageID(),
		Classification:  f.classification,
		ThreatID:        threatID,
		LaunchLocation:  launchPos,
		LaunchTime:      launchTime,
		PredictedImpact: impactPos,
		EstimatedTime:   impactTime,
		Confidence:      confidence,
		RequiredAction:  action,
		Timestamp:       time.Now(),
		ExpiresAt:       time.Now().Add(15 * time.Minute),
	}
}

// INCOMINGFormatter formats INCOMING messages
type INCOMINGFormatter struct {
	classification string
}

// NewINCOMINGFormatter creates a new INCOMING formatter
func NewINCOMINGFormatter(classification string) *INCOMINGFormatter {
	return &INCOMINGFormatter{classification: classification}
}

// Format formats an INCOMING message
func (f *INCOMINGFormatter) Format(threatID, threatType string,
	currentPos Position, velocity Velocity, impactPos Position,
	timeToImpact time.Duration, confidence float64) *INCOMINGMessage {

	action := ActionDefend
	if timeToImpact < 5*time.Minute {
		action = ActionEvacuate
	}

	return &INCOMINGMessage{
		MessageID:       generateMessageID(),
		Classification:  f.classification,
		ThreatID:        threatID,
		ThreatType:      threatType,
		CurrentPosition: currentPos,
		Velocity:        velocity,
		PredictedImpact: impactPos,
		TimeToImpact:    timeToImpact,
		Confidence:      confidence,
		RequiredAction:  action,
		Timestamp:       time.Now(),
	}
}

// ToText formats a CONOPREP message to text format
func (m *CONOPREPMessage) ToText() string {
	return fmt.Sprintf(`MSGID: %s
CLASS: %s
PRIORITY: %s
CATEGORY: %s
ORIGINATOR: %s
TIME: %s
HEADING: %s
CONTENT: %s
ACTION: %s%s%s`,
		m.MessageID,
		m.Classification,
		getPriorityString(m.Priority),
		getCategoryString(m.Category),
		m.Originator,
		m.Timestamp.Format("2006-01-02 15:04:05 UTC"),
		m.Heading,
		m.Content,
		getActionString(m.RequiredAction),
		formatTrackSection(m.TrackData),
		formatImpactSection(m.ImpactData),
	)
}

// ToIMMINENTText formats an IMMINENT message to text
func (m *IMMINENTMessage) ToText() string {
	return fmt.Sprintf(`MSGID: %s
CLASS: %s
TYPE: IMMINENT
THREAT: %s
LAUNCH: %s at %s
IMPACT: %s at %s
CONFIDENCE: %.2f
ACTION: %s
TIME: %s
EXPIRES: %s`,
		m.MessageID,
		m.Classification,
		m.ThreatID,
		formatPosition(m.LaunchLocation),
		m.LaunchTime.Format("15:04:05"),
		formatPosition(m.PredictedImpact),
		m.EstimatedTime.Format("15:04:05"),
		m.Confidence,
		getActionString(m.RequiredAction),
		m.Timestamp.Format("2006-01-02 15:04:05"),
		m.ExpiresAt.Format("2006-01-02 15:04:05"),
	)
}

// ToINCOMINGText formats an INCOMING message to text
func (m *INCOMINGMessage) ToText() string {
	return fmt.Sprintf(`MSGID: %s
CLASS: %s
TYPE: INCOMING
THREAT: %s (%s)
POSITION: %s
VELOCITY: %.0f,%.0f,%.0f m/s
IMPACT: %s
TIME TO IMPACT: %s
CONFIDENCE: %.2f
ACTION: %s
TIME: %s`,
		m.MessageID,
		m.Classification,
		m.ThreatID,
		m.ThreatType,
		formatPosition(m.CurrentPosition),
		m.Velocity.Vx, m.Velocity.Vy, m.Velocity.Vz,
		formatPosition(m.PredictedImpact),
		m.TimeToImpact,
		m.Confidence,
		getActionString(m.RequiredAction),
		m.Timestamp.Format("15:04:05"),
	)
}

// Helper functions

func generateMessageID() string {
	return fmt.Sprintf("NCA-%d-%06d", time.Now().Year(), time.Now().UnixNano()%1000000)
}

func formatPosition(p Position) string {
	latDir := "N"
	if p.Latitude < 0 {
		latDir = "S"
	}
	lonDir := "E"
	if p.Longitude < 0 {
		lonDir = "W"
	}
	return fmt.Sprintf("%.4f%s %.4f%s %.0fm",
		abs(p.Latitude), latDir,
		abs(p.Longitude), lonDir,
		p.Altitude)
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func formatTrackSection(track *TrackSummary) string {
	if track == nil {
		return ""
	}
	return fmt.Sprintf("\nTRACK: %s\nPOSITION: %s\nCONFIDENCE: %.2f",
		track.TrackNumber,
		formatPosition(track.CurrentPosition),
		track.Confidence)
}

func formatImpactSection(impact *ImpactSummary) string {
	if impact == nil {
		return ""
	}
	return fmt.Sprintf("\nIMPACT: %s\nTIME: %s\nCONFIDENCE: %.2f",
		formatPosition(impact.Location),
		impact.EstimatedTime.Format("15:04:05"),
		impact.Confidence)
}

func getPriorityString(p AlertPriority) string {
	switch p {
	case PriorityRoutine:
		return "ROUTINE"
	case PriorityPriority:
		return "PRIORITY"
	case PriorityFlash:
		return "FLASH"
	case PriorityFlashOverride:
		return "FLASH OVERRIDE"
	default:
		return "ROUTINE"
	}
}

func getCategoryString(c AlertCategory) string {
	switch c {
	case CategoryThreat:
		return "THREAT"
	case CategoryLaunch:
		return "LAUNCH"
	case CategoryImpact:
		return "IMPACT"
	case CategoryDefense:
		return "DEFENSE"
	case CategoryExercise:
		return "EXERCISE"
	default:
		return "UNKNOWN"
	}
}

func getActionString(a AlertAction) string {
	switch a {
	case ActionNone:
		return "NONE"
	case ActionMonitor:
		return "MONITOR"
	case ActionPrepare:
		return "PREPARE"
	case ActionDefend:
		return "DEFEND"
	case ActionEvacuate:
		return "EVACUATE"
	default:
		return "NONE"
	}
}
