// Package opir provides OPIR (Overhead Persistent Infrared) sensor data ingestion
// for SBIRS-High and NG-OPIR satellites
package opir

import (
	"context"
	"time"
)

// OPIRDataFeed defines the interface for OPIR sensor data feeds
type OPIRDataFeed interface {
	// Connect establishes connection to the OPIR data source
	Connect(ctx context.Context) error
	
	// Disconnect closes the connection
	Disconnect() error
	
	// Receive returns a channel for receiving sightings
	Receive() <-chan OPIRSighting
	
	// Errors returns a channel for receiving errors
	Errors() <-chan error
	
	// IsConnected returns connection status
	IsConnected() bool
	
	// Stats returns feed statistics
	Stats() FeedStats
}

// OPIRSighting represents a single OPIR detection
type OPIRSighting struct {
	// Identification
	ID           string    `json:"id"`
	SensorID     string    `json:"sensor_id"`     // e.g., "SBIRS-GEO-1", "NG-OPIR-1"
	SequenceNum  uint64    `json:"sequence_num"`   // Sequence number from sensor
	
	// Timestamps
	Timestamp    time.Time `json:"timestamp"`      // Detection time (UTC)
	ReceivedAt   time.Time `json:"received_at"`    // Reception time (UTC)
	
	// Location (WGS84)
	Latitude    float64   `json:"latitude"`      // Degrees (-90 to 90)
	Longitude   float64   `json:"longitude"`      // Degrees (-180 to 180)
	Altitude    float64   `json:"altitude"`       // Meters above WGS84
	
	// Motion
	VelocityE   float64   `json:"velocity_e"`     // East velocity (m/s)
	VelocityN   float64   `json:"velocity_n"`     // North velocity (m/s)
	VelocityU   float64   `json:"velocity_u"`     // Up velocity (m/s)
	Heading     float64   `json:"heading"`        // Heading (degrees, 0-360)
	Speed       float64   `json:"speed"`          // Speed (m/s)
	
	// Detection Quality
	Confidence  float64   `json:"confidence"`     // Detection confidence (0.0-1.0)
	SNR         float64   `json:"snr"`            // Signal-to-noise ratio (dB)
	Intensity   float64   `json:"intensity"`      // IR intensity (W/m²/sr)
	
	// Target Classification
	TargetType  string    `json:"target_type"`    // BOOSTER, DEBRIS, UNKNOWN
	Signature   string    `json:"signature"`      // IR signature classification
	
	// Covariance (uncertainty)
	CovLat      float64   `json:"cov_lat"`        // Latitude variance (deg²)
	CovLon      float64   `json:"cov_lon"`        // Longitude variance (deg²)
	CovAlt      float64   `json:"cov_alt"`        // Altitude variance (m²)
	
	// Metadata
	SourceIP    string    `json:"source_ip"`      // Source IP address
	Flags       uint32    `json:"flags"`         // Status flags
}

// OPIRConfig holds configuration for OPIR feeds
type OPIRConfig struct {
	// Connection
	Endpoints    []string      `json:"endpoints"`     // Data feed endpoints
	Port         int           `json:"port"`          // Port number
	Protocol     string        `json:"protocol"`      // "tcp", "udp", "http"
	
	// Authentication
	Username     string        `json:"username"`      // Authentication username
	Password     string        `json:"password"`      // Authentication password
	CertFile     string        `json:"cert_file"`     // TLS certificate file
	KeyFile      string        `json:"key_file"`      // TLS key file
	CAFile       string        `json:"ca_file"`       // CA certificate file
	
	// Timeouts
	ConnectTimeout time.Duration `json:"connect_timeout"` // Connection timeout
	ReadTimeout    time.Duration `json:"read_timeout"`   // Read timeout
	WriteTimeout   time.Duration `json:"write_timeout"`  // Write timeout
	KeepAlive      time.Duration `json:"keep_alive"`     // Keep-alive interval
	
	// Retry
	MaxRetries     int           `json:"max_retries"`     // Maximum retry attempts
	RetryDelay     time.Duration `json:"retry_delay"`     // Initial retry delay
	MaxRetryDelay   time.Duration `json:"max_retry_delay"` // Maximum retry delay
	
	// Buffering
	BufferSize     int           `json:"buffer_size"`     // Receive buffer size
	BatchSize      int           `json:"batch_size"`      // Batch size for processing
	BatchTimeout   time.Duration `json:"batch_timeout"`  // Batch timeout
	
	// Validation
	MinConfidence  float64       `json:"min_confidence"`  // Minimum confidence
	MinSNR         float64       `json:"min_snr"`         // Minimum SNR (dB)
	MaxAltitude    float64       `json:"max_altitude"`    // Maximum altitude (m)
	
	// Processing
	EnableFiltering bool         `json:"enable_filtering"` // Enable noise filtering
	DedupeWindow    time.Duration `json:"dedupe_window"`   // Deduplication window
}

// FeedStats provides statistics about the feed
type FeedStats struct {
	Connected         bool          `json:"connected"`
	Uptime            time.Duration `json:"uptime"`
	TotalReceived     uint64        `json:"total_received"`
	TotalErrors       uint64        `json:"total_errors"`
	LastError         string        `json:"last_error"`
	ReceiveRate       float64       `json:"receive_rate"`     // Messages/sec
	AvgLatency        time.Duration `json:"avg_latency"`
	ReconnectCount    uint64        `json:"reconnect_count"`
	LastReceived      time.Time     `json:"last_received"`
}

// OPIRError represents OPIR-specific errors
type OPIRError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	SensorID   string `json:"sensor_id,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	Retryable  bool   `json:"retryable"`
}

func (e *OPIRError) Error() string {
	return e.Message
}

// Error codes
const (
	ErrCodeConnection    = "CONNECTION_ERROR"
	ErrCodeAuthentication = "AUTHENTICATION_ERROR"
	ErrCodeTimeout       = "TIMEOUT_ERROR"
	ErrCodeValidation   = "VALIDATION_ERROR"
	ErrCodeParsing      = "PARSING_ERROR"
	ErrCodeBufferFull   = "BUFFER_FULL_ERROR"
	ErrCodeShutdown     = "SHUTDOWN_ERROR"
)

// NewConnectionError creates a connection error
func NewConnectionError(msg string, retryable bool) *OPIRError {
	return &OPIRError{
		Code:      ErrCodeConnection,
		Message:   msg,
		Timestamp: time.Now(),
		Retryable: retryable,
	}
}

// NewAuthenticationError creates an authentication error
func NewAuthenticationError(msg string) *OPIRError {
	return &OPIRError{
		Code:      ErrCodeAuthentication,
		Message:   msg,
		Timestamp: time.Now(),
		Retryable: false,
	}
}

// NewTimeoutError creates a timeout error
func NewTimeoutError(msg string) *OPIRError {
	return &OPIRError{
		Code:      ErrCodeTimeout,
		Message:   msg,
		Timestamp: time.Now(),
		Retryable: true,
	}
}

// NewValidationError creates a validation error
func NewValidationError(msg string, sensorID string) *OPIRError {
	return &OPIRError{
		Code:      ErrCodeValidation,
		Message:   msg,
		SensorID:  sensorID,
		Timestamp: time.Now(),
		Retryable: false,
	}
}

// NewParsingError creates a parsing error
func NewParsingError(msg string, sensorID string) *OPIRError {
	return &OPIRError{
		Code:      ErrCodeParsing,
		Message:   msg,
		SensorID:  sensorID,
		Timestamp: time.Now(),
		Retryable: false,
	}
}

// DefaultConfig returns default OPIR configuration
func DefaultConfig() *OPIRConfig {
	return &OPIRConfig{
		Protocol:        "tcp",
		Port:           5000,
		ConnectTimeout:  30 * time.Second,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		KeepAlive:      30 * time.Second,
		MaxRetries:     10,
		RetryDelay:     1 * time.Second,
		MaxRetryDelay:  30 * time.Second,
		BufferSize:     10000,
		BatchSize:      100,
		BatchTimeout:   100 * time.Millisecond,
		MinConfidence:  0.5,
		MinSNR:         10.0,
		MaxAltitude:    150000.0,
		EnableFiltering: true,
		DedupeWindow:   60 * time.Second,
	}
}

// Validate validates the configuration
func (c *OPIRConfig) Validate() error {
	if len(c.Endpoints) == 0 {
		return NewValidationError("endpoints required", "")
	}
	if c.ConnectTimeout <= 0 {
		return NewValidationError("connect_timeout must be positive", "")
	}
	if c.ReadTimeout <= 0 {
		return NewValidationError("read_timeout must be positive", "")
	}
	if c.MaxRetries < 0 {
		return NewValidationError("max_retries cannot be negative", "")
	}
	if c.MinConfidence < 0 || c.MinConfidence > 1 {
		return NewValidationError("min_confidence must be between 0 and 1", "")
	}
	return nil
}