// Package radar provides radar sensor data ingestion
// for AN/TPY-2, SBX, and UEWR radars
package radar

import (
	"context"
	"time"
)

// RadarDataFeed defines the interface for radar data feeds
type RadarDataFeed interface {
	// Connect establishes connection to the radar data source
	Connect(ctx context.Context) error

	// Disconnect closes the connection
	Disconnect() error

	// Receive returns a channel for receiving tracks
	Receive() <-chan RadarTrack

	// Errors returns a channel for receiving errors
	Errors() <-chan error

	// IsConnected returns connection status
	IsConnected() bool

	// Stats returns feed statistics
	Stats() FeedStats
}

// RadarTrack represents a radar track
type RadarTrack struct {
	// Identification
	ID          string `json:"id"`
	TrackNumber uint32 `json:"track_number"`
	SensorID    string `json:"sensor_id"`

	// Timestamps
	Timestamp  time.Time `json:"timestamp"`   // Track time
	ReceivedAt time.Time `json:"received_at"` // Reception time

	// Position (WGS84)
	Latitude  float64 `json:"latitude"`  // Degrees (-90 to 90)
	Longitude float64 `json:"longitude"` // Degrees (-180 to 180)
	Altitude  float64 `json:"altitude"`  // Meters above WGS84

	// Velocity
	VelocityN float64 `json:"velocity_n"` // North velocity (m/s)
	VelocityE float64 `json:"velocity_e"` // East velocity (m/s)
	VelocityU float64 `json:"velocity_u"` // Up velocity (m/s)
	Speed     float64 `json:"speed"`      // Speed magnitude (m/s)
	Heading   float64 `json:"heading"`    // Heading (degrees, 0-360)

	// Tracking Quality
	TrackQuality uint8  `json:"track_quality"` // Track quality (0-7)
	TrackStatus  string `json:"track_status"`  // TRACK, COAST, DROP
	NUpdates     uint32 `json:"n_updates"`     // Number of updates

	// Covariance (uncertainty)
	CovPosition [3][3]float64 `json:"cov_position"` // Position covariance (lat, lon, alt)
	CovVelocity [3][3]float64 `json:"cov_velocity"` // Velocity covariance

	// Classification
	TargetType string  `json:"target_type"` // AIRCRAFT, MISSILE, UNKNOWN
	TargetSize float64 `json:"target_size"` // RCS (m²)

	// Radar Parameters
	RCS       float64 `json:"rcs"`        // Radar cross-section (dBsm)
	SNR       float64 `json:"snr"`        // Signal-to-noise ratio (dB)
	Range     float64 `json:"range"`      // Range to target (m)
	RangeRate float64 `json:"range_rate"` // Range rate (m/s)
	Azimuth   float64 `json:"azimuth"`    // Azimuth angle (degrees)
	Elevation float64 `json:"elevation"`  // Elevation angle (degrees)

	// Metadata
	Mode       string  `json:"mode"`        // Search, Track, Track-While-Scan
	BeamNumber uint16  `json:"beam_number"` // Beam identifier
	PRF        float64 `json:"prf"`         // Pulse repetition frequency (Hz)
	SourceIP   string  `json:"source_ip"`   // Source IP address
}

// RadarConfig holds configuration for radar feeds
type RadarConfig struct {
	// Connection
	Endpoints []string `json:"endpoints"` // Data feed endpoints
	Port      int      `json:"port"`      // Port number
	Protocol  string   `json:"protocol"`  // "tcp", "udp", "serial"

	// Authentication
	Username string `json:"username"`  // Authentication username
	Password string `json:"password"`  // Authentication password
	CertFile string `json:"cert_file"` // TLS certificate file
	KeyFile  string `json:"key_file"`  // TLS key file
	CAFile   string `json:"ca_file"`   // CA certificate file

	// Timeouts
	ConnectTimeout time.Duration `json:"connect_timeout"`
	ReadTimeout    time.Duration `json:"read_timeout"`
	WriteTimeout   time.Duration `json:"write_timeout"`

	// Retry
	MaxRetries    int           `json:"max_retries"`
	RetryDelay    time.Duration `json:"retry_delay"`
	MaxRetryDelay time.Duration `json:"max_retry_delay"`

	// Buffering
	BufferSize   int           `json:"buffer_size"`
	BatchSize    int           `json:"batch_size"`
	BatchTimeout time.Duration `json:"batch_timeout"`

	// Radar Parameters
	RadarType     string  `json:"radar_type"`     // "TPY2", "SBX", "UEWR"
	FrequencyBand string  `json:"frequency_band"` // "X", "C", "L"
	MaxRange      float64 `json:"max_range"`      // Maximum range (m)
	MinRange      float64 `json:"min_range"`      // Minimum range (m)

	// Filtering
	EnableFiltering bool    `json:"enable_filtering"`
	MinSNR          float64 `json:"min_snr"`        // Minimum SNR (dB)
	MinRCS          float64 `json:"min_rcs"`        // Minimum RCS (dBsm)
	MaxRangeRate    float64 `json:"max_range_rate"` // Maximum range rate (m/s)

	// Correlation
	CorrelationWindow time.Duration `json:"correlation_window"`
	MaxTrackAge       time.Duration `json:"max_track_age"`
}

// FeedStats provides statistics about the feed
type FeedStats struct {
	Connected      bool          `json:"connected"`
	Uptime         time.Duration `json:"uptime"`
	TotalReceived  uint64        `json:"total_received"`
	TotalErrors    uint64        `json:"total_errors"`
	LastError      string        `json:"last_error"`
	ReceiveRate    float64       `json:"receive_rate"`
	AvgLatency     time.Duration `json:"avg_latency"`
	ReconnectCount uint64        `json:"reconnect_count"`
	LastReceived   time.Time     `json:"last_received"`
	TracksActive   uint32        `json:"tracks_active"`
	TracksDropped  uint32        `json:"tracks_dropped"`
}

// RadarError represents radar-specific errors
type RadarError struct {
	Code      string    `json:"code"`
	Message   string    `json:"message"`
	SensorID  string    `json:"sensor_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Retryable bool      `json:"retryable"`
}

func (e *RadarError) Error() string {
	return e.Message
}

// Error codes
const (
	ErrCodeConnection     = "CONNECTION_ERROR"
	ErrCodeAuthentication = "AUTHENTICATION_ERROR"
	ErrCodeTimeout        = "TIMEOUT_ERROR"
	ErrCodeValidation     = "VALIDATION_ERROR"
	ErrCodeParsing        = "PARSING_ERROR"
	ErrCodeTrackLost      = "TRACK_LOST_ERROR"
	ErrCodeBufferFull     = "BUFFER_FULL_ERROR"
)

// NewConnectionError creates a connection error
func NewConnectionError(msg string, retryable bool) *RadarError {
	return &RadarError{
		Code:      ErrCodeConnection,
		Message:   msg,
		Timestamp: time.Now(),
		Retryable: retryable,
	}
}

// NewTrackLostError creates a track lost error
func NewTrackLostError(trackID string) *RadarError {
	return &RadarError{
		Code:      ErrCodeTrackLost,
		Message:   "track lost",
		SensorID:  trackID,
		Timestamp: time.Now(),
		Retryable: false,
	}
}

// DefaultConfig returns default radar configuration
func DefaultConfig() *RadarConfig {
	return &RadarConfig{
		Protocol:          "tcp",
		Port:              5001,
		ConnectTimeout:    30 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		MaxRetries:        10,
		RetryDelay:        1 * time.Second,
		MaxRetryDelay:     30 * time.Second,
		BufferSize:        10000,
		BatchSize:         100,
		BatchTimeout:      100 * time.Millisecond,
		MinSNR:            10.0,
		MinRCS:            -20.0,
		MaxRangeRate:      10000.0,
		EnableFiltering:   true,
		CorrelationWindow: 5 * time.Second,
		MaxTrackAge:       60 * time.Second,
	}
}

// Validate validates the configuration
func (c *RadarConfig) Validate() error {
	if len(c.Endpoints) == 0 {
		return &RadarError{
			Code:    ErrCodeValidation,
			Message: "endpoints required",
		}
	}
	if c.ConnectTimeout <= 0 {
		return &RadarError{
			Code:    ErrCodeValidation,
			Message: "connect_timeout must be positive",
		}
	}
	if c.ReadTimeout <= 0 {
		return &RadarError{
			Code:    ErrCodeValidation,
			Message: "read_timeout must be positive",
		}
	}
	return nil
}

// TrackStatus constants
const (
	TrackStatusInit    = "INIT"
	TrackStatusTrack   = "TRACK"
	TrackStatusCoast   = "COAST"
	TrackStatusDrop    = "DROP"
	TrackStatusUnknown = "UNKNOWN"
)

// TargetType constants
const (
	TargetTypeAircraft = "AIRCRAFT"
	TargetTypeMissile  = "MISSILE"
	TargetTypeUAV      = "UAV"
	TargetTypeUnknown  = "UNKNOWN"
)

// RadarMode constants
const (
	ModeSearch         = "SEARCH"
	ModeTrack          = "TRACK"
	ModeTrackWhileScan = "TWS"
	ModeBeacon         = "BEACON"
)
