// Package radar provides SBX (Sea-Based X-band) radar data ingestion
// SBX is a mobile X-band radar mounted on a sea-based platform
// Used for ballistic missile defense testing and operations
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

// SBXFeed implements RadarDataFeed for SBX radar
type SBXFeed struct {
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

// NewSBXFeed creates a new SBX feed
func NewSBXFeed(config *RadarConfig) (*SBXFeed, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	
	// Set SBX-specific defaults
	if config.RadarType == "" {
		config.RadarType = "SBX"
	}
	if config.FrequencyBand == "" {
		config.FrequencyBand = "X"
	}
	
	return &SBXFeed{
		config:     config,
		tracks:     make(chan RadarTrack, config.BufferSize),
		errors:     make(chan error, 100),
		trackCache: make(map[uint32]*RadarTrack),
	}, nil
}

// Connect establishes connection to SBX data feed
func (s *SBXFeed) Connect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.connected {
		return nil
	}
	
	s.ctx, s.cancel = context.WithCancel(ctx)
	
	var conn net.Conn
	var err error
	
	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		conn, err = s.dial()
		if err == nil {
			break
		}
		
		if attempt < s.config.MaxRetries {
			delay := s.calculateBackoff(attempt)
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
			fmt.Sprintf("SBX connection failed after %d attempts: %v", s.config.MaxRetries, err),
			false,
		)
	}
	
	s.conn = conn
	s.connected = true
	s.stats.Connected = true
	s.stats.ReconnectCount++
	
	go s.receiveLoop()
	
	return nil
}

// dial establishes connection
func (s *SBXFeed) dial() (net.Conn, error) {
	if len(s.config.Endpoints) == 0 {
		return nil, &RadarError{
			Code:    ErrCodeValidation,
			Message: "no endpoints configured",
		}
	}
	
	endpoint := s.config.Endpoints[0]
	address := fmt.Sprintf("%s:%d", endpoint, s.config.Port)
	
	if s.config.CertFile != "" {
		return s.dialTLS(address)
	}
	
	conn, err := net.DialTimeout("tcp", address, s.config.ConnectTimeout)
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
func (s *SBXFeed) dialTLS(address string) (net.Conn, error) {
	cert, err := tls.LoadX509KeyPair(s.config.CertFile, s.config.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}
	
	caCert, err := os.ReadFile(s.config.CAFile)
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
	
	dialer := &net.Dialer{Timeout: s.config.ConnectTimeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", address, tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("TLS connection failed: %w", err)
	}
	
	return conn, nil
}

// calculateBackoff calculates exponential backoff
func (s *SBXFeed) calculateBackoff(attempt int) time.Duration {
	delay := s.config.RetryDelay * time.Duration(1<<uint(attempt))
	if delay > s.config.MaxRetryDelay {
		delay = s.config.MaxRetryDelay
	}
	return delay
}

// Disconnect closes the connection
func (s *SBXFeed) Disconnect() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.connected {
		return nil
	}
	
	s.connected = false
	s.stats.Connected = false
	
	if s.cancel != nil {
		s.cancel()
	}
	
	if s.conn != nil {
		return s.conn.Close()
	}
	
	return nil
}

// Receive returns the tracks channel
func (s *SBXFeed) Receive() <-chan RadarTrack {
	return s.tracks
}

// Errors returns the errors channel
func (s *SBXFeed) Errors() <-chan error {
	return s.errors
}

// IsConnected returns connection status
func (s *SBXFeed) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connected
}

// Stats returns feed statistics
func (s *SBXFeed) Stats() FeedStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats
}

// receiveLoop continuously receives data
func (s *SBXFeed) receiveLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			if err := s.receiveOne(); err != nil {
				s.handleError(err)
			}
		}
	}
}

// receiveOne receives a single track
func (s *SBXFeed) receiveOne() error {
	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()
	
	if conn == nil {
		return NewConnectionError("no connection", true)
	}
	
	conn.SetReadDeadline(time.Now().Add(s.config.ReadTimeout))
	
	// SBX message format (similar to TPY-2 but with different header):
	// Bytes 0-3: Sync word (0x53425800 = "SBX\0")
	// Bytes 4-5: Version
	// Bytes 6-7: Message type
	// Bytes 8-11: Length
	// Bytes 12-15: Sequence number
	// Bytes 16+: Track data
	
	header := make([]byte, 16)
	if _, err := io.ReadFull(conn, header); err != nil {
		return err
	}
	
	sync := binary.BigEndian.Uint32(header[0:4])
	if sync != 0x53425800 { // "SBX\0"
		return &RadarError{
			Code:    ErrCodeParsing,
			Message: fmt.Sprintf("invalid SBX sync word: 0x%08X", sync),
		}
	}
	
	_ = binary.BigEndian.Uint16(header[4:6])  // version
	msgType := binary.BigEndian.Uint16(header[6:8])
	length := binary.BigEndian.Uint32(header[8:12])
	_ = binary.BigEndian.Uint32(header[12:16]) // sequence
	
	if length > 1048576 {
		return &RadarError{
			Code:    ErrCodeParsing,
			Message: fmt.Sprintf("SBX message too large: %d", length),
		}
	}
	
	data := make([]byte, length)
	if _, err := io.ReadFull(conn, data); err != nil {
		return err
	}
	
	track, err := s.parseTrack(msgType, data)
	if err != nil {
		return err
	}
	
	if err := s.validateTrack(track); err != nil {
		return err
	}
	
	s.mu.Lock()
	s.stats.TotalReceived++
	s.stats.LastReceived = time.Now()
	s.stats.TracksActive = uint32(len(s.trackCache))
	s.mu.Unlock()
	
	select {
	case s.tracks <- *track:
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
	
	return nil
}

// parseTrack parses SBX track data
func (s *SBXFeed) parseTrack(msgType uint16, data []byte) (*RadarTrack, error) {
	// SBX track format (similar to TPY-2 but includes additional fields)
	if len(data) < 128 {
		return nil, &RadarError{
			Code:    ErrCodeParsing,
			Message: "SBX track data too short",
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
	// SBX-specific: beam number
	beamNum := binary.BigEndian.Uint16(data[116:118])
	// SBX-specific: PRF
	prf := float64(binary.BigEndian.Uint64(data[118:126]))
	// Target type
	targetType := binary.BigEndian.Uint32(data[126:130])
	
	track := &RadarTrack{
		ID:           fmt.Sprintf("SBX-%d-%d", trackNum, timestamp/1000000),
		TrackNumber: trackNum,
		SensorID:    fmt.Sprintf("SBX-%d", msgType),
		Timestamp:   time.Unix(0, timestamp),
		ReceivedAt:  time.Now(),
		Latitude:    lat,
		Longitude:   lon,
		Altitude:    alt,
		VelocityN:   velN,
		VelocityE:   velE,
		VelocityU:   velU,
		TrackQuality: uint8(quality & 0x07),
		TrackStatus:  s.parseStatus(status),
		Range:       trackRange,
		RangeRate:   rangeRate,
		Azimuth:     azimuth,
		Elevation:   elevation,
		RCS:         rcs,
		SNR:         snr,
		BeamNumber:  beamNum,
		PRF:         prf,
		Mode:        ModeTrackWhileScan,
		TargetType:  s.parseTargetType(targetType),
	}
	
	// Calculate speed
	track.Speed = (velN*velN + velE*velE + velU*velU)
	if track.Speed > 0 {
		track.Speed = (track.Speed)
	}
	
	// Calculate heading
	track.Heading = s.calculateHeading(velE, velN)
	
	s.mu.Lock()
	s.trackCache[trackNum] = track
	s.mu.Unlock()
	
	return track, nil
}

// parseStatus converts status code to string
func (s *SBXFeed) parseStatus(status uint32) string {
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

// parseTargetType converts target type code to string
func (s *SBXFeed) parseTargetType(targetType uint32) string {
	switch targetType {
	case 1:
		return TargetTypeAircraft
	case 2:
		return TargetTypeMissile
	case 3:
		return TargetTypeUAV
	default:
		return TargetTypeUnknown
	}
}

// calculateHeading calculates heading from velocity
func (s *SBXFeed) calculateHeading(velE, velN float64) float64 {
	heading := (180.0 / 3.14159265359) * (velE / (velE*velE + velN*velN))
	for heading < 0 {
		heading += 360
	}
	for heading >= 360 {
		heading -= 360
	}
	return heading
}

// validateTrack validates the track
func (s *SBXFeed) validateTrack(track *RadarTrack) error {
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
	
	if s.config.EnableFiltering {
		if track.SNR < s.config.MinSNR {
			return &RadarError{
				Code:    ErrCodeValidation,
				Message: fmt.Sprintf("low SNR: %.2f", track.SNR),
			}
		}
	}
	
	return nil
}

// handleError handles an error
func (s *SBXFeed) handleError(err error) {
	s.mu.Lock()
	s.stats.TotalErrors++
	s.stats.LastError = err.Error()
	s.mu.Unlock()
	
	select {
	case s.errors <- err:
	case <-time.After(1 * time.Second):
	}
}

// GetTrack returns a track by track number
func (s *SBXFeed) GetTrack(trackNum uint32) *RadarTrack {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.trackCache[trackNum]
}

// GetActiveTracks returns all active tracks
func (s *SBXFeed) GetActiveTracks() []*RadarTrack {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	tracks := make([]*RadarTrack, 0, len(s.trackCache))
	for _, track := range s.trackCache {
		if track.TrackStatus != TrackStatusDrop {
			tracks = append(tracks, track)
		}
	}
	return tracks
}