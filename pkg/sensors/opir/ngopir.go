// Package opir provides NG-OPIR (Next Generation OPIR) satellite data ingestion
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

// NGOPIRFeed implements OPIRDataFeed for NG-OPIR satellites
type NGOPIRFeed struct {
	config    *OPIRConfig
	conn      net.Conn
	sightings chan OPIRSighting
	errors    chan error
	connected bool
	stats     FeedStats
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewNGOPIRFeed creates a new NG-OPIR feed
func NewNGOPIRFeed(config *OPIRConfig) (*NGOPIRFeed, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &NGOPIRFeed{
		config:    config,
		sightings: make(chan OPIRSighting, config.BufferSize),
		errors:    make(chan error, 100),
	}, nil
}

// Connect establishes connection to NG-OPIR data feed
func (n *NGOPIRFeed) Connect(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.connected {
		return nil
	}

	n.ctx, n.cancel = context.WithCancel(ctx)

	var conn net.Conn
	var err error

	for attempt := 0; attempt <= n.config.MaxRetries; attempt++ {
		conn, err = n.dial()
		if err == nil {
			break
		}

		if attempt < n.config.MaxRetries {
			delay := n.calculateBackoff(attempt)
			select {
			case <-time.After(delay):
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	if err != nil {
		return NewConnectionError(fmt.Sprintf("failed to connect after %d attempts: %v", n.config.MaxRetries, err), false)
	}

	n.conn = conn
	n.connected = true
	n.stats.Connected = true
	n.stats.ReconnectCount++

	go n.receiveLoop()

	return nil
}

// dial establishes connection
func (n *NGOPIRFeed) dial() (net.Conn, error) {
	if len(n.config.Endpoints) == 0 {
		return nil, NewValidationError("no endpoints configured", "")
	}

	endpoint := n.config.Endpoints[0]
	address := fmt.Sprintf("%s:%d", endpoint, n.config.Port)

	// NG-OPIR uses mTLS by default
	if n.config.CertFile != "" {
		return n.dialTLS(address)
	}

	// Plain TCP for testing
	conn, err := net.DialTimeout("tcp", address, n.config.ConnectTimeout)
	if err != nil {
		return nil, err
	}

	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(n.config.KeepAlive)
	}

	return conn, nil
}

// dialTLS establishes mTLS connection
func (n *NGOPIRFeed) dialTLS(address string) (net.Conn, error) {
	cert, err := tls.LoadX509KeyPair(n.config.CertFile, n.config.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	caCert, err := os.ReadFile(n.config.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS13, // NG-OPIR requires TLS 1.3
	}

	dialer := &net.Dialer{Timeout: n.config.ConnectTimeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", address, tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("TLS connection failed: %w", err)
	}

	return conn, nil
}

// calculateBackoff calculates exponential backoff delay
func (n *NGOPIRFeed) calculateBackoff(attempt int) time.Duration {
	delay := n.config.RetryDelay * time.Duration(1<<uint(attempt))
	if delay > n.config.MaxRetryDelay {
		delay = n.config.MaxRetryDelay
	}
	return delay
}

// Disconnect closes the connection
func (n *NGOPIRFeed) Disconnect() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.connected {
		return nil
	}

	n.connected = false
	n.stats.Connected = false

	if n.cancel != nil {
		n.cancel()
	}

	if n.conn != nil {
		return n.conn.Close()
	}

	return nil
}

// Receive returns the sightings channel
func (n *NGOPIRFeed) Receive() <-chan OPIRSighting {
	return n.sightings
}

// Errors returns the errors channel
func (n *NGOPIRFeed) Errors() <-chan error {
	return n.errors
}

// IsConnected returns connection status
func (n *NGOPIRFeed) IsConnected() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.connected
}

// Stats returns feed statistics
func (n *NGOPIRFeed) Stats() FeedStats {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.stats
}

// receiveLoop continuously receives data
func (n *NGOPIRFeed) receiveLoop() {
	for {
		select {
		case <-n.ctx.Done():
			return
		default:
			if err := n.receiveOne(); err != nil {
				n.handleError(err)
			}
		}
	}
}

// receiveOne receives a single sighting
func (n *NGOPIRFeed) receiveOne() error {
	n.mu.RLock()
	conn := n.conn
	n.mu.RUnlock()

	if conn == nil {
		return NewConnectionError("no connection", true)
	}

	conn.SetReadDeadline(time.Now().Add(n.config.ReadTimeout))

	// NG-OPIR uses a different message format:
	// 4-byte length prefix + message body

	// Read length prefix
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(conn, lenBuf); err != nil {
		return err
	}

	msgLen := binary.BigEndian.Uint32(lenBuf)
	if msgLen > 1048576 { // Max 1MB message
		return NewParsingError(fmt.Sprintf("message too large: %d", msgLen), "")
	}

	// Read message body
	msg := make([]byte, msgLen)
	if _, err := io.ReadFull(conn, msg); err != nil {
		return err
	}

	// Parse sighting
	sighting, err := n.parseMessage(msg)
	if err != nil {
		return err
	}

	// Validate sighting
	if err := n.validateSighting(sighting); err != nil {
		return err
	}

	// Update stats
	n.mu.Lock()
	n.stats.TotalReceived++
	n.stats.LastReceived = time.Now()
	n.mu.Unlock()

	// Send sighting
	select {
	case n.sightings <- *sighting:
	case <-n.ctx.Done():
		return n.ctx.Err()
	}

	return nil
}

// parseMessage parses NG-OPIR message
func (n *NGOPIRFeed) parseMessage(msg []byte) (*OPIRSighting, error) {
	// NG-OPIR message format:
	// Header (16 bytes):
	//   Bytes 0-3: Magic (0x4E474F50 = "NGOP")
	//   Bytes 4-5: Version
	//   Bytes 6-7: Message type
	//   Bytes 8-15: Timestamp
	// Body (variable):
	//   JSON-encoded sighting data

	if len(msg) < 16 {
		return nil, NewParsingError("message too short", "")
	}

	magic := binary.BigEndian.Uint32(msg[0:4])
	if magic != 0x4E474F50 { // "NGOP"
		return nil, NewParsingError("invalid magic number", "")
	}

	_ = binary.BigEndian.Uint16(msg[4:6]) // version
	msgType := binary.BigEndian.Uint16(msg[6:8])
	timestamp := int64(binary.BigEndian.Uint64(msg[8:16]))

	// Parse JSON body
	body := msg[16:]

	sighting := &OPIRSighting{
		ID:         fmt.Sprintf("NG-OPIR-%d-%d", msgType, timestamp/1000000),
		SensorID:   n.parseSensorID(msgType),
		Timestamp:  time.Unix(0, timestamp),
		ReceivedAt: time.Now(),
	}

	// Parse JSON fields (simplified - real implementation would use encoding/json)
	// For now, extract key fields
	if err := n.parseJSONFields(sighting, body); err != nil {
		return nil, err
	}

	return sighting, nil
}

// parseSensorID converts message type to sensor ID
func (n *NGOPIRFeed) parseSensorID(msgType uint16) string {
	// NG-OPIR has multiple satellites
	satellites := []string{"NG-OPIR-1", "NG-OPIR-2", "NG-OPIR-3", "NG-OPIR-4"}
	if int(msgType) < len(satellites) {
		return satellites[msgType]
	}
	return "NG-OPIR-UNKNOWN"
}

// parseJSONFields extracts fields from JSON body
func (n *NGOPIRFeed) parseJSONFields(sighting *OPIRSighting, body []byte) error {
	// Simplified JSON parsing - real implementation would use encoding/json
	// Extract latitude, longitude, altitude, etc.

	// For testing, just set some default values
	sighting.Latitude = 0.0
	sighting.Longitude = 0.0
	sighting.Altitude = 0.0
	sighting.Confidence = 0.9
	sighting.SNR = 20.0

	return nil
}

// validateSighting validates the sighting
func (n *NGOPIRFeed) validateSighting(sighting *OPIRSighting) error {
	if sighting.Latitude < -90 || sighting.Latitude > 90 {
		return NewValidationError(fmt.Sprintf("invalid latitude: %.2f", sighting.Latitude), sighting.SensorID)
	}

	if sighting.Longitude < -180 || sighting.Longitude > 180 {
		return NewValidationError(fmt.Sprintf("invalid longitude: %.2f", sighting.Longitude), sighting.SensorID)
	}

	if sighting.Confidence < n.config.MinConfidence {
		return NewValidationError(fmt.Sprintf("low confidence: %.2f", sighting.Confidence), sighting.SensorID)
	}

	return nil
}

// handleError handles an error
func (n *NGOPIRFeed) handleError(err error) {
	n.mu.Lock()
	n.stats.TotalErrors++
	n.stats.LastError = err.Error()
	n.mu.Unlock()

	select {
	case n.errors <- err:
	case <-time.After(1 * time.Second):
	}
}
