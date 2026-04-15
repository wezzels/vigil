// Package jreap implements JREAP (Joint Range Extension Applications Protocol)
// JREAP is defined in MIL-STD-3011 for extending data links over various media
package jreap

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"
)

// JREAPType represents the type of JREAP connection
type JREAPType int

const (
	JREAPTypeA JREAPType = iota // Serial connection
	JREAPTypeB                  // TCP/IP connection
	JREAPTypeC                  // Satellite link
)

// String returns string representation of JREAP type
func (t JREAPType) String() string {
	switch t {
	case JREAPTypeA:
		return "JREAP-A"
	case JREAPTypeB:
		return "JREAP-B"
	case JREAPTypeC:
		return "JREAP-C"
	default:
		return "Unknown"
	}
}

// JREAPConfig holds configuration for JREAP connection
type JREAPConfig struct {
	Type         JREAPType     `json:"type"`
	Address      string        `json:"address"`
	Port         int           `json:"port"`
	SerialDevice string        `json:"serial_device"`
	BaudRate     int           `json:"baud_rate"`
	BufferSize   int           `json:"buffer_size"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
}

// DefaultJREAPConfig returns default JREAP configuration
func DefaultJREAPConfig() *JREAPConfig {
	return &JREAPConfig{
		Type:         JREAPTypeB,
		Address:      "0.0.0.0",
		Port:         15000,
		BufferSize:   65536,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
}

// MessageHeader represents a JREAP message header
type MessageHeader struct {
	SyncByte      byte   `json:"sync_byte"`
	Version       byte   `json:"version"`
	Control       byte   `json:"control"`
	MessageLength uint16 `json:"message_length"`
	Checksum      uint16 `json:"checksum"`
}

// Message represents a JREAP message
type Message struct {
	Header MessageHeader `json:"header"`
	Data   []byte        `json:"data"`
	RxTime time.Time     `json:"rx_time"`
	TxTime time.Time     `json:"tx_time"`
	Valid  bool          `json:"valid"`
}

// JREAPBridge handles JREAP communication
type JREAPBridge struct {
	config   *JREAPConfig
	conn     net.Conn
	listener net.Listener
	running  bool
	mu       sync.RWMutex
	rxChan   chan Message
	txChan   chan Message
	errChan  chan error
	stats    BridgeStats
}

// BridgeStats holds bridge statistics
type BridgeStats struct {
	MessagesReceived uint64    `json:"messages_received"`
	MessagesSent     uint64    `json:"messages_sent"`
	BytesReceived    uint64    `json:"bytes_received"`
	BytesSent        uint64    `json:"bytes_sent"`
	Errors           uint64    `json:"errors"`
	LastRxTime       time.Time `json:"last_rx_time"`
	LastTxTime       time.Time `json:"last_tx_time"`
}

// NewJREAPBridge creates a new JREAP bridge
func NewJREAPBridge(config *JREAPConfig) *JREAPBridge {
	if config == nil {
		config = DefaultJREAPConfig()
	}

	return &JREAPBridge{
		config:  config,
		rxChan:  make(chan Message, 1000),
		txChan:  make(chan Message, 1000),
		errChan: make(chan error, 100),
	}
}

// Start starts the JREAP bridge
func (j *JREAPBridge) Start() error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.running {
		return ErrAlreadyRunning
	}

	switch j.config.Type {
	case JREAPTypeB:
		return j.startTCP()
	case JREAPTypeA:
		return j.startSerial()
	case JREAPTypeC:
		return j.startSatellite()
	default:
		return ErrInvalidType
	}
}

// startTCP starts TCP/IP connection (JREAP-B)
func (j *JREAPBridge) startTCP() error {
	addr := fmt.Sprintf("%s:%d", j.config.Address, j.config.Port)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	j.listener = listener
	j.running = true

	go j.acceptLoop()

	return nil
}

// acceptLoop handles incoming connections
func (j *JREAPBridge) acceptLoop() {
	for {
		conn, err := j.listener.Accept()
		if err != nil {
			if !j.running {
				return
			}
			j.errChan <- err
			continue
		}

		j.mu.Lock()
		j.conn = conn
		j.mu.Unlock()

		go j.receiveLoop()
	}
}

// receiveLoop handles receiving messages
func (j *JREAPBridge) receiveLoop() {
	buf := make([]byte, j.config.BufferSize)

	for {
		if !j.running {
			return
		}

		if j.config.ReadTimeout > 0 {
			j.conn.SetReadDeadline(time.Now().Add(j.config.ReadTimeout))
		}

		n, err := j.conn.Read(buf)
		if err != nil {
			if !j.running {
				return
			}
			j.mu.Lock()
			j.stats.Errors++
			j.mu.Unlock()
			j.errChan <- err
			continue
		}

		// Parse messages
		messages := j.parseMessages(buf[:n])
		for _, msg := range messages {
			j.mu.Lock()
			j.stats.MessagesReceived++
			j.stats.BytesReceived += uint64(len(msg.Data))
			j.stats.LastRxTime = time.Now()
			j.mu.Unlock()

			select {
			case j.rxChan <- msg:
			default:
				// Channel full, drop message
			}
		}
	}
}

// parseMessages parses received data into messages
func (j *JREAPBridge) parseMessages(data []byte) []Message {
	var messages []Message

	for len(data) > 0 {
		msg, remaining, err := ParseMessage(data)
		if err != nil {
			break
		}

		messages = append(messages, msg)
		data = remaining
	}

	return messages
}

// startSerial starts serial connection (JREAP-A)
func (j *JREAPBridge) startSerial() error {
	// Serial implementation would go here
	// Requires go.bug.st/serial package
	return ErrNotImplemented
}

// startSatellite starts satellite connection (JREAP-C)
func (j *JREAPBridge) startSatellite() error {
	// Satellite implementation would go here
	// Similar to TCP but with different framing
	return ErrNotImplemented
}

// Stop stops the JREAP bridge
func (j *JREAPBridge) Stop() error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if !j.running {
		return nil
	}

	j.running = false

	if j.conn != nil {
		j.conn.Close()
	}

	if j.listener != nil {
		j.listener.Close()
	}

	return nil
}

// Send sends a message
func (j *JREAPBridge) Send(data []byte) error {
	j.mu.RLock()
	defer j.mu.RUnlock()

	if !j.running {
		return ErrNotRunning
	}

	if j.conn == nil {
		return ErrNotConnected
	}

	// Build message
	msg := BuildMessage(data)
	msgBytes := msg.Serialize()

	if j.config.WriteTimeout > 0 {
		j.conn.SetWriteDeadline(time.Now().Add(j.config.WriteTimeout))
	}

	n, err := j.conn.Write(msgBytes)
	if err != nil {
		j.stats.Errors++
		return err
	}

	j.stats.MessagesSent++
	j.stats.BytesSent += uint64(n)
	j.stats.LastTxTime = time.Now()

	return nil
}

// Receive returns the receive channel
func (j *JREAPBridge) Receive() <-chan Message {
	return j.rxChan
}

// Errors returns the error channel
func (j *JREAPBridge) Errors() <-chan error {
	return j.errChan
}

// Stats returns bridge statistics
func (j *JREAPBridge) Stats() BridgeStats {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.stats
}

// ParseMessage parses a JREAP message from data
func ParseMessage(data []byte) (Message, []byte, error) {
	if len(data) < 8 {
		return Message{}, nil, ErrMessageTooShort
	}

	// Check sync byte
	if data[0] != 0x55 {
		return Message{}, nil, ErrInvalidSync
	}

	var msg Message
	msg.Header.SyncByte = data[0]
	msg.Header.Version = data[1]
	msg.Header.Control = data[2]
	msg.Header.MessageLength = binary.BigEndian.Uint16(data[3:5])
	msg.Header.Checksum = binary.BigEndian.Uint16(data[6:8])

	// Validate length
	if len(data) < int(msg.Header.MessageLength)+8 {
		return Message{}, nil, ErrMessageTooShort
	}

	// Extract data
	dataEnd := 8 + int(msg.Header.MessageLength)
	msg.Data = make([]byte, msg.Header.MessageLength)
	copy(msg.Data, data[8:dataEnd])

	// Validate checksum
	calculatedChecksum := CalculateChecksum(msg.Data)
	if calculatedChecksum != msg.Header.Checksum {
		msg.Valid = false
	} else {
		msg.Valid = true
	}

	msg.RxTime = time.Now()

	// Return remaining data
	remaining := data[dataEnd:]

	return msg, remaining, nil
}

// BuildMessage builds a JREAP message from data
func BuildMessage(data []byte) Message {
	msg := Message{
		Header: MessageHeader{
			SyncByte:      0x55,
			Version:       0x01,
			Control:       0x00,
			MessageLength: uint16(len(data)),
		},
		Data:   data,
		TxTime: time.Now(),
		Valid:  true,
	}

	msg.Header.Checksum = CalculateChecksum(data)

	return msg
}

// Serialize serializes a message to bytes
func (m *Message) Serialize() []byte {
	buf := make([]byte, 8+len(m.Data))

	buf[0] = m.Header.SyncByte
	buf[1] = m.Header.Version
	buf[2] = m.Header.Control
	binary.BigEndian.PutUint16(buf[3:5], m.Header.MessageLength)
	buf[5] = 0x00 // Reserved
	binary.BigEndian.PutUint16(buf[6:8], m.Header.Checksum)

	copy(buf[8:], m.Data)

	return buf
}

// CalculateChecksum calculates JREAP checksum
func CalculateChecksum(data []byte) uint16 {
	var checksum uint16

	for _, b := range data {
		checksum += uint16(b)
	}

	return checksum ^ 0xFFFF
}

// Errors
var (
	ErrAlreadyRunning  = &JREAPError{Code: "ALREADY_RUNNING", Message: "already running"}
	ErrNotRunning      = &JREAPError{Code: "NOT_RUNNING", Message: "not running"}
	ErrNotConnected    = &JREAPError{Code: "NOT_CONNECTED", Message: "not connected"}
	ErrInvalidType     = &JREAPError{Code: "INVALID_TYPE", Message: "invalid JREAP type"}
	ErrNotImplemented  = &JREAPError{Code: "NOT_IMPLEMENTED", Message: "not implemented"}
	ErrMessageTooShort = &JREAPError{Code: "MESSAGE_TOO_SHORT", Message: "message too short"}
	ErrInvalidSync     = &JREAPError{Code: "INVALID_SYNC", Message: "invalid sync byte"}
)

// JREAPError represents a JREAP error
type JREAPError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *JREAPError) Error() string {
	return e.Message
}
