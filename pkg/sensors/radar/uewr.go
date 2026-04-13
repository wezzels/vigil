// Package radar provides UEWR (Upgraded Early Warning Radar) data ingestion
// UEWR is an L-band, phased array radar for early warning and surveillance
// Used for ballistic missile detection and tracking
package radar

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"
)

// UEWRFeed implements RadarDataFeed for UEWR radar
type UEWRFeed struct {
	config      *RadarConfig
	conn        net.Conn
	tracks      chan RadarTrack
	errors      chan error
	connected   bool
	stats       FeedStats
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	trackCache  map[uint32]*RadarTrack
}

// NewUEWRFeed creates a new UEWR feed
func NewUEWRFeed(config *RadarConfig) (*UEWRFeed, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	
	// Set UEWR-specific defaults
	if config.RadarType == "" {
		config.RadarType = "UEWR"
	}
	if config.FrequencyBand == "" {
		config.FrequencyBand = "L"
	}
	if config.MaxRange == 0 {
		config.MaxRange = 5000000.0 // 5000 km typical for UEWR
	}
	
	return &UEWRFeed{
		config:     config,
		tracks:     make(chan RadarTrack, config.BufferSize),
		errors:     make(chan error, 100),
		trackCache: make(map[uint32]*RadarTrack),
	}, nil
}

// Connect establishes connection to UEWR data feed
func (u *UEWRFeed) Connect(ctx context.Context) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	
	if u.connected {
		return nil
	}
	
	u.ctx, u.cancel = context.WithCancel(ctx)
	
	var conn net.Conn
	var err error
	
	for attempt := 0; attempt <= u.config.MaxRetries; attempt++ {
		conn, err = u.dial()
		if err == nil {
			break
		}
		
		if attempt < u.config.MaxRetries {
			delay := u.calculateBackoff(attempt)
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
			fmt.Sprintf("UEWR connection failed after %d attempts: %v", u.config.MaxRetries, err),
			false,
		)
	}
	
	u.conn = conn
	u.connected = true
	u.stats.Connected = true
	u.stats.ReconnectCount++
	
	go u.receiveLoop()
	
	return nil
}

// dial establishes connection
func (u *UEWRFeed) dial() (net.Conn, error) {
	if len(u.config.Endpoints) == 0 {
		return nil, &RadarError{
			Code:    ErrCodeValidation,
			Message: "no endpoints configured",
		}
	}
	
	endpoint := u.config.Endpoints[0]
	address := fmt.Sprintf("%s:%d", endpoint, u.config.Port)
	
	if u.config.CertFile != "" {
		return u.dialTLS(address)
	}
	
	conn, err := net.DialTimeout("tcp", address, u.config.ConnectTimeout)
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
func (u *UEWRFeed) dialTLS(address string) (net.Conn, error) {
	cert, err := tls.LoadX509KeyPair(u.config.CertFile, u.config.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}
	
	caCert, err := os.ReadFile(u.config.CAFile)
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
	
	dialer := &net.Dialer{Timeout: u.config.ConnectTimeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", address, tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("TLS connection failed: %w", err)
	}
	
	return conn, nil
}

// calculateBackoff calculates exponential backoff
func (u *UEWRFeed) calculateBackoff(attempt int) time.Duration {
	delay := u.config.RetryDelay * time.Duration(1<<uint(attempt))
	if delay > u.config.MaxRetryDelay {
		delay = u.config.MaxRetryDelay
	}
	return delay
}

// Disconnect closes the connection
func (u *UEWRFeed) Disconnect() error {
	u.mu.Lock()
	defer u.mu.Unlock()
	
	if !u.connected {
		return nil
	}
	
	u.connected = false
	u.stats.Connected = false
	
	if u.cancel != nil {
		u.cancel()
	}
	
	if u.conn != nil {
		return u.conn.Close()
	}
	
	return nil
}

// Receive returns the tracks channel
func (u *UEWRFeed) Receive() <-chan RadarTrack {
	return u.tracks
}

// Errors returns the errors channel
func (u *UEWRFeed) Errors() <-chan error {
	return u.errors
}

// IsConnected returns connection status
func (u *UEWRFeed) IsConnected() bool {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.connected
}

// Stats returns feed statistics
func (u *UEWRFeed) Stats() FeedStats {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.stats
}

// receiveLoop continuously receives data
func (u *UEWRFeed) receiveLoop() {
	for {
		select {
		case <-u.ctx.Done():
			return
		default:
			if err := u.receiveOne(); err != nil {
				u.handleError(err)
			}
		}
	}
}

// receiveOne receives a single track
func (u *UEWRFeed) receiveOne() error {
	u.mu.RLock()
	conn := u.conn
	u.mu.RUnlock()
	
	if conn == nil {
		return NewConnectionError("no connection", true)
	}
	
	conn.SetReadDeadline(time.Now().Add(u.config.ReadTimeout))
	
	// UEWR message format (L-band radar):
	// Bytes 0-3: Sync word (0x55455752 = "UEWR")
	// Bytes 4-5: Version
	// Bytes 6-7: Message type
	// Bytes 8-11: Length
	// Bytes 12-15: Track count
	// Bytes 16+: Track data (multiple tracks per message)
	
	header := make([]byte, 20)
	if _, err := io.ReadFull(conn, header); err != nil {
		return err
	}
	
	sync := binary.BigEndian.Uint32(header[0:4])
	if sync != 0x55455752 { // "UEWR"
		return &RadarError{
			Code:    ErrCodeParsing,
			Message: fmt.Sprintf("invalid UEWR sync word: 0x%08X", sync),
		}
	}
	
	_ = binary.BigEndian.Uint16(header[4:6])  // version
	msgType := binary.BigEndian.Uint16(header[6:8])
	length := binary.BigEndian.Uint32(header[8:12])
	trackCount := binary.BigEndian.Uint32(header[12:16])
	_ = binary.BigEndian.Uint32(header[16:20]) // timestamp
	
	if length > 1048576 {
		return &RadarError{
			Code:    ErrCodeParsing,
			Message: fmt.Sprintf("UEWR message too large: %d", length),
		}
	}
	
	data := make([]byte, length)
	if _, err := io.ReadFull(conn, data); err != nil {
		return err
	}
	
	// Parse multiple tracks from message
	offset := 0
	for i := uint32(0); i < trackCount; i++ {
		if offset+80 > len(data) {
			break
		}
		
		track, err := u.parseTrack(msgType, data[offset:offset+80])
		if err != nil {
			u.handleError(err)
			offset += 80
			continue
		}
		
		if err := u.validateTrack(track); err != nil {
			u.handleError(err)
			offset += 80
			continue
		}
		
		u.mu.Lock()
		u.stats.TotalReceived++
		u.stats.LastReceived = time.Now()
		u.mu.Unlock()
		
		select {
		case u.tracks <- *track:
		case <-u.ctx.Done():
			return u.ctx.Err()
		}
		
		offset += 80
	}
	
	u.mu.Lock()
	u.stats.TracksActive = uint32(len(u.trackCache))
	u.mu.Unlock()
	
	return nil
}

// parseTrack parses UEWR track data
func (u *UEWRFeed) parseTrack(msgType uint16, data []byte) (*RadarTrack, error) {
	if len(data) < 80 {
		return nil, &RadarError{
			Code:    ErrCodeParsing,
			Message: "UEWR track data too short",
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
	quality := binary.BigEndian.Uint16(data[60:62])
	status := binary.BigEndian.Uint16(data[62:64])
	trackRange := float64(binary.BigEndian.Uint64(data[64:72]))
	rangeRate := float64(binary.BigEndian.Uint64(data[72:80]))
	
	track := &RadarTrack{
		ID:           fmt.Sprintf("UEWR-%d-%d", trackNum, timestamp/1000000),
		TrackNumber: trackNum,
		SensorID:    fmt.Sprintf("UEWR-%d", msgType),
		Timestamp:   time.Unix(0, timestamp),
		ReceivedAt:  time.Now(),
		Latitude:    lat,
		Longitude:   lon,
		Altitude:    alt,
		VelocityN:   velN,
		VelocityE:   velE,
		VelocityU:   velU,
		TrackQuality: uint8(quality & 0x07),
		TrackStatus:  u.parseStatus(uint32(status)),
		Range:       trackRange,
		RangeRate:   rangeRate,
		Mode:        ModeSearch,
	}
	
	// Calculate speed
	track.Speed = (velN*velN + velE*velE + velU*velU)
	if track.Speed > 0 {
		track.Speed = (track.Speed)
	}
	
	// Calculate heading
	track.Heading = u.calculateHeading(velE, velN)
	
	u.mu.Lock()
	u.trackCache[trackNum] = track
	u.mu.Unlock()
	
	return track, nil
}

// parseStatus converts status code to string
func (u *UEWRFeed) parseStatus(status uint32) string {
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

// calculateHeading calculates heading from velocity
func (u *UEWRFeed) calculateHeading(velE, velN float64) float64 {
	heading := 0.0
	if velE != 0 || velN != 0 {
		heading = 90.0 - 57.29577951308232*velN/(velE*velE+velN*velN)
	}
	for heading < 0 {
		heading += 360
	}
	for heading >= 360 {
		heading -= 360
	}
	return heading
}

// validateTrack validates the track
func (u *UEWRFeed) validateTrack(track *RadarTrack) error {
	if track.Latitude < -90 || track.Latitude > 90 {
		return &RadarError{
			Code:    ErrCodeValidation,
			Message: fmt.Sprintf("invalid latitude: %.2f", track.Latitude),
		}
	}
	
	if track.Longitude < -180 || track.Longitude > 180 {
		return &RadarError{
			Code:    ErrCodeValidation,
			Message: fmt.Sprintf("invalid longitude: %.2f", track.Longitude),
		}
	}
	
	if u.config.EnableFiltering {
		if track.Range > u.config.MaxRange {
			return &RadarError{
				Code:    ErrCodeValidation,
				Message: fmt.Sprintf("range exceeds max: %.2f > %.2f", track.Range, u.config.MaxRange),
			}
		}
	}
	
	return nil
}

// handleError handles an error
func (u *UEWRFeed) handleError(err error) {
	u.mu.Lock()
	u.stats.TotalErrors++
	u.stats.LastError = err.Error()
	u.mu.Unlock()
	
	select {
	case u.errors <- err:
	case <-time.After(1 * time.Second):
	}
}

// GetTrack returns a track by track number
func (u *UEWRFeed) GetTrack(trackNum uint32) *RadarTrack {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.trackCache[trackNum]
}

// GetActiveTracks returns all active tracks
func (u *UEWRFeed) GetActiveTracks() []*RadarTrack {
	u.mu.RLock()
	defer u.mu.RUnlock()
	
	tracks := make([]*RadarTrack, 0, len(u.trackCache))
	for _, track := range u.trackCache {
		if track.TrackStatus != TrackStatusDrop {
			tracks = append(tracks, track)
		}
	}
	return tracks
}