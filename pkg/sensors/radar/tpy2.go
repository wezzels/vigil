// Package radar provides AN/TPY-2 radar data ingestion
// AN/TPY-2 is an X-band, phased array, transportable radar
// Used for ballistic missile defense and surveillance
package radar

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"sync"
	"time"
)

// TPY2Feed implements RadarDataFeed for AN/TPY-2 radar
type TPY2Feed struct {
	config     *RadarConfig
	conn       net.Conn
	tracks     chan RadarTrack
	errors     chan error
	connected  bool
	stats      FeedStats
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	trackCache map[uint32]*RadarTrack
}

// NewTPY2Feed creates a new AN/TPY-2 feed
func NewTPY2Feed(config *RadarConfig) (*TPY2Feed, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &TPY2Feed{
		config:     config,
		tracks:     make(chan RadarTrack, config.BufferSize),
		errors:     make(chan error, 100),
		trackCache: make(map[uint32]*RadarTrack),
	}, nil
}

// Connect establishes connection to AN/TPY-2 data feed
func (t *TPY2Feed) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.connected {
		return nil
	}

	t.ctx, t.cancel = context.WithCancel(ctx)

	var conn net.Conn
	var err error

	for attempt := 0; attempt <= t.config.MaxRetries; attempt++ {
		conn, err = t.dial()
		if err == nil {
			break
		}

		if attempt < t.config.MaxRetries {
			delay := t.calculateBackoff(attempt)
			select {
			case <-time.After(delay):
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	if err != nil {
		return NewConnectionError(
			fmt.Sprintf("failed to connect after %d attempts: %v", t.config.MaxRetries, err),
			false,
		)
	}

	t.conn = conn
	t.connected = true
	t.stats.Connected = true
	t.stats.ReconnectCount++

	go t.receiveLoop()

	return nil
}

// dial establishes connection with TLS
func (t *TPY2Feed) dial() (net.Conn, error) {
	if len(t.config.Endpoints) == 0 {
		return nil, &RadarError{
			Code:    ErrCodeValidation,
			Message: "no endpoints configured",
		}
	}

	endpoint := t.config.Endpoints[0]
	address := fmt.Sprintf("%s:%d", endpoint, t.config.Port)

	if t.config.CertFile != "" {
		return t.dialTLS(address)
	}

	conn, err := net.DialTimeout("tcp", address, t.config.ConnectTimeout)
	if err != nil {
		return nil, err
	}

	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}

	return conn, nil
}

// dialTLS establishes TLS connection
func (t *TPY2Feed) dialTLS(address string) (net.Conn, error) {
	cert, err := tls.LoadX509KeyPair(t.config.CertFile, t.config.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	caCert, err := os.ReadFile(t.config.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS12,
	}

	dialer := &net.Dialer{Timeout: t.config.ConnectTimeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", address, tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("TLS connection failed: %w", err)
	}

	return conn, nil
}

// calculateBackoff calculates exponential backoff
func (t *TPY2Feed) calculateBackoff(attempt int) time.Duration {
	delay := t.config.RetryDelay * time.Duration(1<<uint(attempt))
	if delay > t.config.MaxRetryDelay {
		delay = t.config.MaxRetryDelay
	}
	return delay
}

// Disconnect closes the connection
func (t *TPY2Feed) Disconnect() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.connected {
		return nil
	}

	t.connected = false
	t.stats.Connected = false

	if t.cancel != nil {
		t.cancel()
	}

	if t.conn != nil {
		return t.conn.Close()
	}

	return nil
}

// Receive returns the tracks channel
func (t *TPY2Feed) Receive() <-chan RadarTrack {
	return t.tracks
}

// Errors returns the errors channel
func (t *TPY2Feed) Errors() <-chan error {
	return t.errors
}

// IsConnected returns connection status
func (t *TPY2Feed) IsConnected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.connected
}

// Stats returns feed statistics
func (t *TPY2Feed) Stats() FeedStats {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.stats
}

// receiveLoop continuously receives data
func (t *TPY2Feed) receiveLoop() {
	for {
		select {
		case <-t.ctx.Done():
			return
		default:
			if err := t.receiveOne(); err != nil {
				t.handleError(err)
			}
		}
	}
}

// receiveOne receives a single track
func (t *TPY2Feed) receiveOne() error {
	t.mu.RLock()
	conn := t.conn
	t.mu.RUnlock()

	if conn == nil {
		return NewConnectionError("no connection", true)
	}

	conn.SetReadDeadline(time.Now().Add(t.config.ReadTimeout))

	// AN/TPY-2 message format:
	// Bytes 0-3: Sync word (0x54505932 = "TPY2")
	// Bytes 4-5: Version
	// Bytes 6-7: Message type
	// Bytes 8-11: Length
	// Bytes 12+: Track data

	header := make([]byte, 16)
	if _, err := io.ReadFull(conn, header); err != nil {
		return err
	}

	sync := binary.BigEndian.Uint32(header[0:4])
	if sync != 0x54505932 { // "TPY2"
		return &RadarError{
			Code:    ErrCodeParsing,
			Message: fmt.Sprintf("invalid sync word: 0x%08X", sync),
		}
	}

	_ = binary.BigEndian.Uint16(header[4:6]) // version
	msgType := binary.BigEndian.Uint16(header[6:8])
	length := binary.BigEndian.Uint32(header[8:12])
	_ = binary.BigEndian.Uint32(header[12:16]) // timestamp

	if length > 1048576 { // Max 1MB
		return &RadarError{
			Code:    ErrCodeParsing,
			Message: fmt.Sprintf("message too large: %d", length),
		}
	}

	// Read track data
	data := make([]byte, length)
	if _, err := io.ReadFull(conn, data); err != nil {
		return err
	}

	// Parse track
	track, err := t.parseTrack(msgType, data)
	if err != nil {
		return err
	}

	// Validate track
	if err := t.validateTrack(track); err != nil {
		return err
	}

	// Update stats
	t.mu.Lock()
	t.stats.TotalReceived++
	t.stats.LastReceived = time.Now()
	t.stats.TracksActive = uint32(len(t.trackCache))
	t.mu.Unlock()

	// Send track
	select {
	case t.tracks <- *track:
	case <-t.ctx.Done():
		return t.ctx.Err()
	}

	return nil
}

// parseTrack parses AN/TPY-2 track data
func (t *TPY2Feed) parseTrack(msgType uint16, data []byte) (*RadarTrack, error) {
	// Track message format:
	// Bytes 0-3: Track number
	// Bytes 4-11: Timestamp (Unix nanoseconds)
	// Bytes 12-19: Latitude (double)
	// Bytes 20-27: Longitude (double)
	// Bytes 28-35: Altitude (double)
	// Bytes 36-43: Velocity North (double)
	// Bytes 44-51: Velocity East (double)
	// Bytes 52-59: Velocity Up (double)
	// Bytes 60-63: Track quality
	// Bytes 64-67: Track status
	// Bytes 68-75: Range (double)
	// Bytes 76-83: Range rate (double)
	// Bytes 84-91: Azimuth (double)
	// Bytes 92-99: Elevation (double)
	// Bytes 100-107: RCS (double)
	// Bytes 108-115: SNR (double)

	if len(data) < 116 {
		return nil, &RadarError{
			Code:    ErrCodeParsing,
			Message: "track data too short",
		}
	}

	trackNum := binary.BigEndian.Uint32(data[0:4])
	timestamp := int64(binary.BigEndian.Uint64(data[4:12]))
	lat := float64(binary.BigEndian.Uint64(data[12:20]))
	lon := float64(binary.BigEndian.Uint64(data[20:28]))
	alt := float64(binary.BigEndian.Uint64(data[28:36]))
	velN := float64(binary.BigEndian.Uint64(data[36:44]))
	velE := float64(binary.BigEndian.Uint64(data[44:52]))
	velU := float64(binary.BigEndian.Uint64(data[52:60]))
	quality := binary.BigEndian.Uint32(data[60:64])
	status := binary.BigEndian.Uint32(data[64:68])
	trackRange := float64(binary.BigEndian.Uint64(data[68:76]))
	rangeRate := float64(binary.BigEndian.Uint64(data[76:84]))
	azimuth := float64(binary.BigEndian.Uint64(data[84:92]))
	elevation := float64(binary.BigEndian.Uint64(data[92:100]))
	rcs := float64(binary.BigEndian.Uint64(data[100:108]))
	snr := float64(binary.BigEndian.Uint64(data[108:116]))

	track := &RadarTrack{
		ID:           fmt.Sprintf("TPY2-%d-%d", trackNum, timestamp/1000000),
		TrackNumber:  trackNum,
		SensorID:     fmt.Sprintf("TPY2-%d", msgType),
		Timestamp:    time.Unix(0, timestamp),
		ReceivedAt:   time.Now(),
		Latitude:     lat,
		Longitude:    lon,
		Altitude:     alt,
		VelocityN:    velN,
		VelocityE:    velE,
		VelocityU:    velU,
		TrackQuality: uint8(quality & 0x07),
		TrackStatus:  t.parseStatus(status),
		Range:        trackRange,
		RangeRate:    rangeRate,
		Azimuth:      azimuth,
		Elevation:    elevation,
		RCS:          rcs,
		SNR:          snr,
		Mode:         ModeTrack,
	}

	// Calculate speed
	track.Speed = (velN*velN + velE*velE + velU*velU)
	if track.Speed > 0 {
		track.Speed = (track.Speed)
	}

	// Calculate heading from velocity
	track.Heading = t.calculateHeading(velE, velN)

	// Update track cache
	t.mu.Lock()
	t.trackCache[trackNum] = track
	t.mu.Unlock()

	return track, nil
}

// parseStatus converts status code to string
func (t *TPY2Feed) parseStatus(status uint32) string {
	switch status {
	case 0:
		return TrackStatusInit
	case 1:
		return TrackStatusTrack
	case 2:
		return TrackStatusCoast
	case 3:
		return TrackStatusDrop
	default:
		return TrackStatusUnknown
	}
}

// calculateHeading calculates heading from velocity components
func (t *TPY2Feed) calculateHeading(velE, velN float64) float64 {
	// Heading is angle from North, clockwise
	// atan2(velE, velN) gives angle from North in radians
	heading := (180.0 / math.Pi) * math.Atan2(velE, velN)
	for heading < 0 {
		heading += 360
	}
	for heading >= 360 {
		heading -= 360
	}
	return heading
}

// validateTrack validates the track
func (t *TPY2Feed) validateTrack(track *RadarTrack) error {
	// Latitude
	if track.Latitude < -90 || track.Latitude > 90 {
		return &RadarError{
			Code:    ErrCodeValidation,
			Message: fmt.Sprintf("invalid latitude: %.2f", track.Latitude),
		}
	}

	// Longitude
	if track.Longitude < -180 || track.Longitude > 180 {
		return &RadarError{
			Code:    ErrCodeValidation,
			Message: fmt.Sprintf("invalid longitude: %.2f", track.Longitude),
		}
	}

	// SNR
	if t.config.EnableFiltering && track.SNR < t.config.MinSNR {
		return &RadarError{
			Code:    ErrCodeValidation,
			Message: fmt.Sprintf("low SNR: %.2f", track.SNR),
		}
	}

	// Range rate
	if t.config.EnableFiltering {
		if track.RangeRate < -t.config.MaxRangeRate || track.RangeRate > t.config.MaxRangeRate {
			return &RadarError{
				Code:    ErrCodeValidation,
				Message: fmt.Sprintf("invalid range rate: %.2f", track.RangeRate),
			}
		}
	}

	return nil
}

// handleError handles an error
func (t *TPY2Feed) handleError(err error) {
	t.mu.Lock()
	t.stats.TotalErrors++
	t.stats.LastError = err.Error()
	t.mu.Unlock()

	select {
	case t.errors <- err:
	case <-time.After(1 * time.Second):
	}
}

// GetTrack returns a track by track number
func (t *TPY2Feed) GetTrack(trackNum uint32) *RadarTrack {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.trackCache[trackNum]
}

// GetActiveTracks returns all active tracks
func (t *TPY2Feed) GetActiveTracks() []*RadarTrack {
	t.mu.RLock()
	defer t.mu.RUnlock()

	tracks := make([]*RadarTrack, 0, len(t.trackCache))
	for _, track := range t.trackCache {
		if track.TrackStatus != TrackStatusDrop {
			tracks = append(tracks, track)
		}
	}
	return tracks
}

// CleanupOldTracks removes old tracks from cache
func (t *TPY2Feed) CleanupOldTracks() {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	for trackNum, track := range t.trackCache {
		if now.Sub(track.Timestamp) > t.config.MaxTrackAge {
			delete(t.trackCache, trackNum)
			t.stats.TracksDropped++
		}
	}
}
