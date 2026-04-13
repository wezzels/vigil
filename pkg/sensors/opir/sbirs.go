// Package opir provides SBIRS-High satellite data ingestion
package opir

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

// SBIRSFeed implements OPIRDataFeed for SBIRS-High satellites
type SBIRSFeed struct {
	config      *OPIRConfig
	conn        net.Conn
	sightings   chan OPIRSighting
	errors      chan error
	connected   bool
	stats       FeedStats
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	reconnectCh chan struct{}
}

// NewSBIRSFeed creates a new SBIRS feed
func NewSBIRSFeed(config *OPIRConfig) (*SBIRSFeed, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	
	return &SBIRSFeed{
		config:      config,
		sightings:  make(chan OPIRSighting, config.BufferSize),
		errors:     make(chan error, 100),
		reconnectCh: make(chan struct{}, 1),
	}, nil
}

// Connect establishes connection to SBIRS data feed
func (s *SBIRSFeed) Connect(ctx context.Context) error {
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
		return NewConnectionError(fmt.Sprintf("failed to connect after %d attempts: %v", s.config.MaxRetries, err), false)
	}
	
	s.conn = conn
	s.connected = true
	s.stats.Connected = true
	s.stats.ReconnectCount++
	
	// Start receive goroutine
	go s.receiveLoop()
	
	return nil
}

// dial establishes connection with TLS
func (s *SBIRSFeed) dial() (net.Conn, error) {
	if len(s.config.Endpoints) == 0 {
		return nil, NewValidationError("no endpoints configured", "")
	}
	
	endpoint := s.config.Endpoints[0]
	address := fmt.Sprintf("%s:%d", endpoint, s.config.Port)
	
	var conn net.Conn
	var err error
	
	if s.config.CertFile != "" {
		// TLS connection
		conn, err = s.dialTLS(address)
	} else {
		// Plain TCP connection
		conn, err = net.DialTimeout("tcp", address, s.config.ConnectTimeout)
	}
	
	if err != nil {
		return nil, err
	}
	
	// Set keep-alive
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(s.config.KeepAlive)
	}
	
	return conn, nil
}

// dialTLS establishes TLS connection
func (s *SBIRSFeed) dialTLS(address string) (net.Conn, error) {
	// Load client certificate
	cert, err := tls.LoadX509KeyPair(s.config.CertFile, s.config.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}
	
	// Load CA certificate
	caCert, err := os.ReadFile(s.config.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate: %w", err)
	}
	
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	
	// TLS configuration
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS12,
	}
	
	// Connect with timeout
	dialer := &net.Dialer{Timeout: s.config.ConnectTimeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", address, tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("TLS connection failed: %w", err)
	}
	
	return conn, nil
}

// calculateBackoff calculates exponential backoff delay
func (s *SBIRSFeed) calculateBackoff(attempt int) time.Duration {
	delay := s.config.RetryDelay * time.Duration(1<<uint(attempt))
	if delay > s.config.MaxRetryDelay {
		delay = s.config.MaxRetryDelay
	}
	return delay
}

// Disconnect closes the connection
func (s *SBIRSFeed) Disconnect() error {
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

// Receive returns the sightings channel
func (s *SBIRSFeed) Receive() <-chan OPIRSighting {
	return s.sightings
}

// Errors returns the errors channel
func (s *SBIRSFeed) Errors() <-chan error {
	return s.errors
}

// IsConnected returns connection status
func (s *SBIRSFeed) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connected
}

// Stats returns feed statistics
func (s *SBIRSFeed) Stats() FeedStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats
}

// receiveLoop continuously receives data
func (s *SBIRSFeed) receiveLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			if err := s.receiveOne(); err != nil {
				s.handleError(err)
				
				// Attempt reconnection
				if s.shouldReconnect(err) {
					s.triggerReconnect()
				}
			}
		}
	}
}

// receiveOne receives a single sighting
func (s *SBIRSFeed) receiveOne() error {
	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()
	
	if conn == nil {
		return NewConnectionError("no connection", true)
	}
	
	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(s.config.ReadTimeout))
	
	// Read header (fixed size)
	header := make([]byte, 32)
	if _, err := io.ReadFull(conn, header); err != nil {
		return err
	}
	
	// Parse header
	sighting, dataLen, err := s.parseHeader(header)
	if err != nil {
		return err
	}
	
	// Read data (if any)
	if dataLen > 0 {
		data := make([]byte, dataLen)
		if _, err := io.ReadFull(conn, data); err != nil {
			return err
		}
		
		// Parse data
		if err := s.parseData(sighting, data); err != nil {
			return err
		}
	}
	
	// Validate sighting
	if err := s.validateSighting(sighting); err != nil {
		return err
	}
	
	// Update stats
	s.mu.Lock()
	s.stats.TotalReceived++
	s.stats.LastReceived = time.Now()
	s.mu.Unlock()
	
	// Send sighting
	select {
	case s.sightings <- *sighting:
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
	
	return nil
}

// parseHeader parses the SBIRS message header
func (s *SBIRSFeed) parseHeader(header []byte) (*OPIRSighting, uint16, error) {
	// SBIRS-High message format:
	// Bytes 0-3: Magic number (0x53424952 = "SBIR")
	// Bytes 4-5: Version (1)
	// Bytes 6-7: Message type
	// Bytes 8-9: Data length
	// Bytes 10-11: Sensor ID
	// Bytes 12-15: Sequence number
	// Bytes 16-23: Timestamp (Unix nanoseconds)
	// Bytes 24-31: Reserved
	
	magic := binary.BigEndian.Uint32(header[0:4])
	if magic != 0x53424952 { // "SBIR"
		return nil, 0, NewParsingError("invalid magic number", "")
	}
	
	version := binary.BigEndian.Uint16(header[4:6])
	if version != 1 {
		return nil, 0, NewParsingError(fmt.Sprintf("unsupported version: %d", version), "")
	}
	
	dataLen := binary.BigEndian.Uint16(header[8:10])
	sensorID := binary.BigEndian.Uint16(header[10:12])
	seqNum := binary.BigEndian.Uint32(header[12:16])
	timestamp := int64(binary.BigEndian.Uint64(header[16:24]))
	
	sighting := &OPIRSighting{
		ID:          fmt.Sprintf("SBIRS-%d-%d", sensorID, seqNum),
		SensorID:    fmt.Sprintf("SBIRS-GEO-%d", sensorID),
		SequenceNum: uint64(seqNum),
		Timestamp:   time.Unix(0, timestamp),
		ReceivedAt:  time.Now(),
	}
	
	return sighting, dataLen, nil
}

// parseData parses the SBIRS message data
func (s *SBIRSFeed) parseData(sighting *OPIRSighting, data []byte) error {
	// SBIRS sighting data format:
	// Bytes 0-7: Latitude (double)
	// Bytes 8-15: Longitude (double)
	// Bytes 16-23: Altitude (double)
	// Bytes 24-31: Velocity East (double)
	// Bytes 32-39: Velocity North (double)
	// Bytes 40-47: Velocity Up (double)
	// Bytes 48-55: Confidence (double)
	// Bytes 56-63: SNR (double)
	// Bytes 64-71: Intensity (double)
	// Bytes 72-79: Covariance Latitude (double)
	// Bytes 80-87: Covariance Longitude (double)
	// Bytes 88-95: Covariance Altitude (double)
	// Bytes 96-99: Target type (uint32)
	// Bytes 100-127: Signature (string, 28 bytes)
	
	if len(data) < 128 {
		return NewParsingError("data too short", sighting.SensorID)
	}
	
	sighting.Latitude = float64(binary.BigEndian.Uint64(data[0:8]))
	sighting.Longitude = float64(binary.BigEndian.Uint64(data[8:16]))
	sighting.Altitude = float64(binary.BigEndian.Uint64(data[16:24]))
	sighting.VelocityE = float64(binary.BigEndian.Uint64(data[24:32]))
	sighting.VelocityN = float64(binary.BigEndian.Uint64(data[32:40]))
	sighting.VelocityU = float64(binary.BigEndian.Uint64(data[40:48]))
	sighting.Confidence = float64(binary.BigEndian.Uint64(data[48:56]))
	sighting.SNR = float64(binary.BigEndian.Uint64(data[56:64]))
	sighting.Intensity = float64(binary.BigEndian.Uint64(data[64:72]))
	sighting.CovLat = float64(binary.BigEndian.Uint64(data[72:80]))
	sighting.CovLon = float64(binary.BigEndian.Uint64(data[80:88]))
	sighting.CovAlt = float64(binary.BigEndian.Uint64(data[88:96]))
	
	targetType := binary.BigEndian.Uint32(data[96:100])
	sighting.TargetType = s.targetTypeString(targetType)
	
	signature := string(data[100:128])
	for i, c := range signature {
		if c == 0 {
			sighting.Signature = string(data[100 : 100+i])
			break
		}
	}
	
	return nil
}

// targetTypeString converts target type code to string
func (s *SBIRSFeed) targetTypeString(code uint32) string {
	switch code {
	case 1:
		return "BOOSTER"
	case 2:
		return "DEBRIS"
	case 3:
		return "UNKNOWN"
	default:
		return "UNCLASSIFIED"
	}
}

// validateSighting validates the sighting
func (s *SBIRSFeed) validateSighting(sighting *OPIRSighting) error {
	// Latitude range
	if sighting.Latitude < -90 || sighting.Latitude > 90 {
		return NewValidationError(fmt.Sprintf("invalid latitude: %.2f", sighting.Latitude), sighting.SensorID)
	}
	
	// Longitude range
	if sighting.Longitude < -180 || sighting.Longitude > 180 {
		return NewValidationError(fmt.Sprintf("invalid longitude: %.2f", sighting.Longitude), sighting.SensorID)
	}
	
	// Altitude range
	if sighting.Altitude < -1000 || sighting.Altitude > s.config.MaxAltitude {
		return NewValidationError(fmt.Sprintf("invalid altitude: %.2f", sighting.Altitude), sighting.SensorID)
	}
	
	// Confidence range
	if sighting.Confidence < s.config.MinConfidence {
		return NewValidationError(fmt.Sprintf("low confidence: %.2f", sighting.Confidence), sighting.SensorID)
	}
	
	// SNR threshold
	if sighting.SNR < s.config.MinSNR {
		return NewValidationError(fmt.Sprintf("low SNR: %.2f", sighting.SNR), sighting.SensorID)
	}
	
	return nil
}

// shouldReconnect determines if reconnection should be attempted
func (s *SBIRSFeed) shouldReconnect(err error) bool {
	opirErr, ok := err.(*OPIRError)
	if !ok {
		return true
	}
	return opirErr.Retryable
}

// triggerReconnect triggers reconnection
func (s *SBIRSFeed) triggerReconnect() {
	select {
	case s.reconnectCh <- struct{}{}:
	default:
	}
}

// handleError handles an error
func (s *SBIRSFeed) handleError(err error) {
	s.mu.Lock()
	s.stats.TotalErrors++
	s.stats.LastError = err.Error()
	s.mu.Unlock()
	
	select {
	case s.errors <- err:
	case <-time.After(1 * time.Second):
		// Error channel full, drop error
	}
}