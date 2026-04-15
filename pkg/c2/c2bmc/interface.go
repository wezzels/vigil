// Package c2bmc provides C2BMC (Command and Control, Battle Management and Communications) integration
// C2BMC is the US missile defense command system
package c2bmc

import (
	"context"
	"time"
)

// AlertPriority defines alert priority levels
type AlertPriority int

const (
	AlertPriorityLow      AlertPriority = 0
	AlertPriorityNormal   AlertPriority = 1
	AlertPriorityHigh     AlertPriority = 2
	AlertPriorityCritical AlertPriority = 3
	AlertPriorityImminent AlertPriority = 4
)

// AlertType defines the type of alert
type AlertType int

const (
	AlertTypeTrack       AlertType = 0 // Track alert
	AlertTypeLaunch      AlertType = 1 // Launch alert
	AlertTypeImpact      AlertType = 2 // Impact prediction
	AlertTypeCorrelation AlertType = 3 // Track correlation
	AlertTypeSystem      AlertType = 4 // System alert
)

// AlertStatus defines alert status
type AlertStatus int

const (
	AlertStatusPending      AlertStatus = 0
	AlertStatusAcknowledged AlertStatus = 1
	AlertStatusProcessing   AlertStatus = 2
	AlertStatusComplete     AlertStatus = 3
	AlertStatusFailed       AlertStatus = 4
	AlertStatusCanceled    AlertStatus = 5
)

// TrackQuality defines track quality levels
type TrackQuality int

const (
	TrackQualityUnknown   TrackQuality = 0
	TrackQualityPoor      TrackQuality = 1
	TrackQualityFair      TrackQuality = 2
	TrackQualityGood      TrackQuality = 3
	TrackQualityExcellent TrackQuality = 4
)

// TrackIdentity defines track identity
type TrackIdentity int

const (
	TrackIdentityUnknown        TrackIdentity = 0
	TrackIdentityPending        TrackIdentity = 1
	TrackIdentityFriendly       TrackIdentity = 2
	TrackIdentityHostile        TrackIdentity = 3
	TrackIdentityNeutral        TrackIdentity = 4
	TrackIdentityAssumedHostile TrackIdentity = 5
)

// Position represents a 3D position
type Position struct {
	Latitude  float64 `json:"latitude"`  // Degrees
	Longitude float64 `json:"longitude"` // Degrees
	Altitude  float64 `json:"altitude"`  // Meters
}

// Velocity represents 3D velocity
type Velocity struct {
	Vx float64 `json:"vx"` // m/s
	Vy float64 `json:"vy"` // m/s
	Vz float64 `json:"vz"` // m/s
}

// TrackData represents track information for C2BMC
type TrackData struct {
	TrackNumber     string        `json:"track_number"`
	TrackID         string        `json:"track_id"`
	Position        Position      `json:"position"`
	Velocity        Velocity      `json:"velocity"`
	Identity        TrackIdentity `json:"identity"`
	Quality         TrackQuality  `json:"quality"`
	Source          string        `json:"source"` // Sensor source
	FirstDetect     time.Time     `json:"first_detect"`
	LastUpdate      time.Time     `json:"last_update"`
	PredictedImpact *Position     `json:"predicted_impact,omitempty"`
	Confidence      float64       `json:"confidence"` // 0.0-1.0
}

// AlertRequest represents an alert submission request
type AlertRequest struct {
	AlertID        string        `json:"alert_id"`
	AlertType      AlertType     `json:"alert_type"`
	Priority       AlertPriority `json:"priority"`
	TrackData      *TrackData    `json:"track_data,omitempty"`
	Message        string        `json:"message,omitempty"`
	SourceSystem   string        `json:"source_system"`
	Timestamp      time.Time     `json:"timestamp"`
	ExpiresAt      time.Time     `json:"expires_at,omitempty"`
	EscalationPath []string      `json:"escalation_path,omitempty"`
}

// AlertResponse represents the response to an alert submission
type AlertResponse struct {
	AlertID        string      `json:"alert_id"`
	Status         AlertStatus `json:"status"`
	AcknowledgedBy string      `json:"acknowledged_by,omitempty"`
	AcknowledgedAt time.Time   `json:"acknowledged_at,omitempty"`
	Message        string      `json:"message,omitempty"`
}

// TrackCorrelationRequest represents a track correlation request
type TrackCorrelationRequest struct {
	PrimaryTrack    string    `json:"primary_track"`
	SecondaryTracks []string  `json:"secondary_tracks"`
	SourceSystem    string    `json:"source_system"`
	Timestamp       time.Time `json:"timestamp"`
}

// TrackCorrelationResponse represents the response to a correlation request
type TrackCorrelationResponse struct {
	PrimaryTrack     string      `json:"primary_track"`
	CorrelatedTracks []string    `json:"correlated_tracks"`
	Confidence       float64     `json:"confidence"`
	Status           AlertStatus `json:"status"`
	Message          string      `json:"message,omitempty"`
}

// C2BMCConfig holds configuration for C2BMC client
type C2BMCConfig struct {
	Endpoint           string        `json:"endpoint"`
	Timeout            time.Duration `json:"timeout"`
	MaxRetries         int           `json:"max_retries"`
	RetryDelay         time.Duration `json:"retry_delay"`
	EnableMTLS         bool          `json:"enable_mtls"`
	CertFile           string        `json:"cert_file"`
	KeyFile            string        `json:"key_file"`
	CAFile             string        `json:"ca_file"`
	InsecureSkipVerify bool          `json:"insecure_skip_verify"`
}

// DefaultC2BMCConfig returns default configuration
func DefaultC2BMCConfig() *C2BMCConfig {
	return &C2BMCConfig{
		Endpoint:   "https://c2bmc.example.mil:8443",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RetryDelay: 1 * time.Second,
		EnableMTLS: true,
	}
}

// C2BMCClient defines the interface for C2BMC operations
type C2BMCClient interface {
	// Alert operations
	SubmitAlert(ctx context.Context, req *AlertRequest) (*AlertResponse, error)
	GetAlertStatus(ctx context.Context, alertID string) (*AlertResponse, error)
	CancelAlert(ctx context.Context, alertID string) error

	// Track operations
	SubmitTrack(ctx context.Context, track *TrackData) error
	GetTrack(ctx context.Context, trackID string) (*TrackData, error)
	CorrelateTracks(ctx context.Context, req *TrackCorrelationRequest) (*TrackCorrelationResponse, error)

	// Status
	HealthCheck(ctx context.Context) error
	GetStatus(ctx context.Context) (*SystemStatus, error)
}

// SystemStatus represents C2BMC system status
type SystemStatus struct {
	Status        string        `json:"status"`
	Connected     bool          `json:"connected"`
	LastHeartbeat time.Time     `json:"last_heartbeat"`
	ActiveAlerts  int           `json:"active_alerts"`
	ActiveTracks  int           `json:"active_tracks"`
	Version       string        `json:"version"`
	Uptime        time.Duration `json:"uptime"`
}

// C2BMCError represents a C2BMC error
type C2BMCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

// Error implements the error interface
func (e *C2BMCError) Error() string {
	if e.Detail != "" {
		return e.Message + ": " + e.Detail
	}
	return e.Message
}

// IsRetryable returns true if the error is retryable
func (e *C2BMCError) IsRetryable() bool {
	// Retry on 5xx errors and rate limits
	return e.Code >= 500 || e.Code == 429
}

// GetPriorityString returns string representation of priority
func GetPriorityString(p AlertPriority) string {
	switch p {
	case AlertPriorityLow:
		return "LOW"
	case AlertPriorityNormal:
		return "NORMAL"
	case AlertPriorityHigh:
		return "HIGH"
	case AlertPriorityCritical:
		return "CRITICAL"
	case AlertPriorityImminent:
		return "IMMINENT"
	default:
		return "UNKNOWN"
	}
}

// GetAlertTypeString returns string representation of alert type
func GetAlertTypeString(t AlertType) string {
	switch t {
	case AlertTypeTrack:
		return "TRACK"
	case AlertTypeLaunch:
		return "LAUNCH"
	case AlertTypeImpact:
		return "IMPACT"
	case AlertTypeCorrelation:
		return "CORRELATION"
	case AlertTypeSystem:
		return "SYSTEM"
	default:
		return "UNKNOWN"
	}
}

// GetStatusString returns string representation of status
func GetStatusString(s AlertStatus) string {
	switch s {
	case AlertStatusPending:
		return "PENDING"
	case AlertStatusAcknowledged:
		return "ACKNOWLEDGED"
	case AlertStatusProcessing:
		return "PROCESSING"
	case AlertStatusComplete:
		return "COMPLETE"
	case AlertStatusFailed:
		return "FAILED"
	case AlertStatusCanceled:
		return "CANCELED"
	default:
		return "UNKNOWN"
	}
}

// GetIdentityString returns string representation of identity
func GetIdentityString(i TrackIdentity) string {
	switch i {
	case TrackIdentityUnknown:
		return "UNKNOWN"
	case TrackIdentityPending:
		return "PENDING"
	case TrackIdentityFriendly:
		return "FRIENDLY"
	case TrackIdentityHostile:
		return "HOSTILE"
	case TrackIdentityNeutral:
		return "NEUTRAL"
	case TrackIdentityAssumedHostile:
		return "ASSUMED_HOSTILE"
	default:
		return "UNKNOWN"
	}
}

// GetQualityString returns string representation of quality
func GetQualityString(q TrackQuality) string {
	switch q {
	case TrackQualityUnknown:
		return "UNKNOWN"
	case TrackQualityPoor:
		return "POOR"
	case TrackQualityFair:
		return "FAIR"
	case TrackQualityGood:
		return "GOOD"
	case TrackQualityExcellent:
		return "EXCELLENT"
	default:
		return "UNKNOWN"
	}
}
