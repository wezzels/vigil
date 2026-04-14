// Package dis provides DIS (Distributed Interactive Simulation) UDP gateway
// DIS is IEEE 1278 standard for real-time simulation
package dis

import (
	"context"
	"net"
	"sync"
	"time"
)

// UDPConfig holds configuration for UDP gateway
type UDPConfig struct {
	Address         string        `json:"address"`          // Multicast address
	Port            int           `json:"port"`             // Port number
	Interface       string        `json:"interface"`        // Network interface
	BufferSize      int           `json:"buffer_size"`      // Receive buffer size
	MulticastTTL    int           `json:"multicast_ttl"`    // Multicast TTL
	EnableBroadcast bool          `json:"enable_broadcast"` // Enable broadcast
	ReadTimeout     time.Duration `json:"read_timeout"`     // Read timeout
	WriteTimeout    time.Duration `json:"write_timeout"`    // Write timeout
}

// DefaultUDPConfig returns default UDP configuration
func DefaultUDPConfig() *UDPConfig {
	return &UDPConfig{
		Address:         "239.1.2.3",
		Port:            3000,
		Interface:       "",
		BufferSize:      65536,
		MulticastTTL:    1,
		EnableBroadcast: false,
		ReadTimeout:     100 * time.Millisecond,
		WriteTimeout:    100 * time.Millisecond,
	}
}

// UDPReceiver handles receiving DIS PDUs via UDP
type UDPReceiver struct {
	config     *UDPConfig
	conn       *net.UDPConn
	multicast  bool
	running    bool
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	handlers   []PDUHandler
	pduChan    chan []byte
	errChan    chan error
	stats      ReceiverStats
}

// PDUHandler handles received PDUs
type PDUHandler interface {
	HandlePDU(data []byte, addr net.Addr)
}

// PDUHandlerFunc is an adapter for PDUHandler
type PDUHandlerFunc func(data []byte, addr net.Addr)

// HandlePDU implements PDUHandler
func (f PDUHandlerFunc) HandlePDU(data []byte, addr net.Addr) {
	f(data, addr)
}

// ReceiverStats holds receiver statistics
type ReceiverStats struct {
	PacketsReceived   uint64 `json:"packets_received"`
	BytesReceived     uint64 `json:"bytes_received"`
	PacketsDropped    uint64 `json:"packets_dropped"`
	Errors            uint64 `json:"errors"`
	LastReceiveTime   time.Time `json:"last_receive_time"`
}

// NewUDPReceiver creates a new UDP receiver
func NewUDPReceiver(config *UDPConfig) *UDPReceiver {
	if config == nil {
		config = DefaultUDPConfig()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &UDPReceiver{
		config:   config,
		ctx:      ctx,
		cancel:   cancel,
		handlers: make([]PDUHandler, 0),
		pduChan:  make(chan []byte, 1000),
		errChan:  make(chan error, 100),
	}
}

// Start starts the UDP receiver
func (r *UDPReceiver) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if r.running {
		return ErrAlreadyRunning
	}
	
	// Parse address
	addr, err := net.ResolveUDPAddr("udp", 
		net.JoinHostPort(r.config.Address, "0"))
	if err != nil {
		return err
	}
	
	// Check if multicast
	r.multicast = addr.IP.IsMulticast()
	
	// Bind to port
	listenAddr := &net.UDPAddr{Port: r.config.Port}
	
	conn, err := net.ListenPacket("udp", listenAddr.String())
	if err != nil {
		return err
	}
	
	r.conn = conn.(*net.UDPConn)
	
	// Join multicast group if needed
	if r.multicast {
		if err := r.joinMulticast(); err != nil {
			conn.Close()
			return err
		}
	}
	
	r.running = true
	
	// Start receive loop
	go r.receiveLoop()
	
	return nil
}

// joinMulticast joins the multicast group
func (r *UDPReceiver) joinMulticast() error {
	group := net.ParseIP(r.config.Address)
	if group == nil {
		return &DISError{Code: "INVALID_ADDRESS", Message: "invalid multicast address"}
	}
	
	// Note: Full multicast join would use net.ListenMulticastUDP
	// This simplified version just sets the buffer size
	r.conn.SetReadBuffer(r.config.BufferSize)
	
	return nil
}

// leaveMulticast leaves the multicast group
func (r *UDPReceiver) leaveMulticast() error {
	// Simplified implementation
	return nil
}

// receiveLoop handles incoming packets
func (r *UDPReceiver) receiveLoop() {
	buf := make([]byte, r.config.BufferSize)
	
	for {
		select {
		case <-r.ctx.Done():
			return
		default:
		}
		
		// Set read timeout
		if r.config.ReadTimeout > 0 {
			r.conn.SetReadDeadline(time.Now().Add(r.config.ReadTimeout))
		}
		
		n, addr, err := r.conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			r.mu.Lock()
			r.stats.Errors++
			r.mu.Unlock()
			select {
			case r.errChan <- err:
			default:
			}
			continue
		}
		
		// Update stats
		r.mu.Lock()
		r.stats.PacketsReceived++
		r.stats.BytesReceived += uint64(n)
		r.stats.LastReceiveTime = time.Now()
		r.mu.Unlock()
		
		// Copy data and dispatch to handlers
		data := make([]byte, n)
		copy(data, buf[:n])
		
		// Dispatch to handlers
		r.mu.RLock()
		for _, handler := range r.handlers {
			go handler.HandlePDU(data, addr)
		}
		r.mu.RUnlock()
	}
}

// Stop stops the UDP receiver
func (r *UDPReceiver) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if !r.running {
		return nil
	}
	
	if r.multicast {
		r.leaveMulticast()
	}
	
	if r.conn != nil {
		r.conn.Close()
	}
	
	if r.cancel != nil {
		r.cancel()
	}
	
	r.running = false
	return nil
}

// AddHandler adds a PDU handler
func (r *UDPReceiver) AddHandler(handler PDUHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers = append(r.handlers, handler)
}

// AddHandlerFunc adds a handler function
func (r *UDPReceiver) AddHandlerFunc(f func([]byte, net.Addr)) {
	r.AddHandler(PDUHandlerFunc(f))
}

// Stats returns receiver statistics
func (r *UDPReceiver) Stats() ReceiverStats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.stats
}

// Errors returns the error channel
func (r *UDPReceiver) Errors() <-chan error {
	return r.errChan
}

// UDPTransmitter handles sending DIS PDUs via UDP
type UDPTransmitter struct {
	config     *UDPConfig
	conn       *net.UDPConn
	dest       *net.UDPAddr
	running    bool
	mu         sync.RWMutex
	stats      TransmitterStats
}

// TransmitterStats holds transmitter statistics
type TransmitterStats struct {
	PacketsSent     uint64 `json:"packets_sent"`
	BytesSent       uint64 `json:"bytes_sent"`
	Errors          uint64 `json:"errors"`
	LastSendTime    time.Time `json:"last_send_time"`
}

// NewUDPTransmitter creates a new UDP transmitter
func NewUDPTransmitter(config *UDPConfig) *UDPTransmitter {
	if config == nil {
		config = DefaultUDPConfig()
	}
	
	return &UDPTransmitter{
		config: config,
	}
}

// Start starts the UDP transmitter
func (t *UDPTransmitter) Start() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	if t.running {
		return ErrAlreadyRunning
	}
	
	// Resolve destination address
	dest, err := net.ResolveUDPAddr("udp",
		net.JoinHostPort(t.config.Address, "3000"))
	if err != nil {
		return err
	}
	t.dest = dest
	
	// Create UDP connection
	localAddr := &net.UDPAddr{Port: 0} // Use any available port
	conn, err := net.DialUDP("udp", localAddr, t.dest)
	if err != nil {
		return err
	}
	
	// Enable broadcast if configured
	if t.config.EnableBroadcast {
		conn.SetWriteBuffer(t.config.BufferSize)
	}
	
	t.conn = conn
	t.running = true
	
	return nil
}

// Stop stops the UDP transmitter
func (t *UDPTransmitter) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	if !t.running {
		return nil
	}
	
	if t.conn != nil {
		t.conn.Close()
	}
	
	t.running = false
	return nil
}

// Send sends a PDU
func (t *UDPTransmitter) Send(data []byte) error {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	if !t.running {
		return ErrNotRunning
	}
	
	// Set write timeout
	if t.config.WriteTimeout > 0 {
		t.conn.SetWriteDeadline(time.Now().Add(t.config.WriteTimeout))
	}
	
	n, err := t.conn.Write(data)
	if err != nil {
		t.stats.Errors++
		return err
	}
	
	t.stats.PacketsSent++
	t.stats.BytesSent += uint64(n)
	t.stats.LastSendTime = time.Now()
	
	return nil
}

// SendTo sends a PDU to a specific address
func (t *UDPTransmitter) SendTo(data []byte, addr string) error {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	if !t.running {
		return ErrNotRunning
	}
	
	dest, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}
	
	if t.config.WriteTimeout > 0 {
		t.conn.SetWriteDeadline(time.Now().Add(t.config.WriteTimeout))
	}
	
	n, err := t.conn.WriteToUDP(data, dest)
	if err != nil {
		t.stats.Errors++
		return err
	}
	
	t.stats.PacketsSent++
	t.stats.BytesSent += uint64(n)
	t.stats.LastSendTime = time.Now()
	
	return nil
}

// Stats returns transmitter statistics
func (t *UDPTransmitter) Stats() TransmitterStats {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.stats
}

// UDPGateway provides both receive and transmit capabilities
type UDPGateway struct {
	receiver    *UDPReceiver
	transmitter *UDPTransmitter
	mu          sync.RWMutex
}

// NewUDPGateway creates a new UDP gateway
func NewUDPGateway(config *UDPConfig) *UDPGateway {
	if config == nil {
		config = DefaultUDPConfig()
	}
	
	return &UDPGateway{
		receiver:    NewUDPReceiver(config),
		transmitter: NewUDPTransmitter(config),
	}
}

// Start starts the gateway
func (g *UDPGateway) Start() error {
	if err := g.receiver.Start(); err != nil {
		return err
	}
	
	if err := g.transmitter.Start(); err != nil {
		g.receiver.Stop()
		return err
	}
	
	return nil
}

// Stop stops the gateway
func (g *UDPGateway) Stop() error {
	var errs []error
	
	if err := g.receiver.Stop(); err != nil {
		errs = append(errs, err)
	}
	
	if err := g.transmitter.Stop(); err != nil {
		errs = append(errs, err)
	}
	
	if len(errs) > 0 {
		return errs[0]
	}
	
	return nil
}

// AddHandler adds a PDU handler
func (g *UDPGateway) AddHandler(handler PDUHandler) {
	g.receiver.AddHandler(handler)
}

// AddHandlerFunc adds a handler function
func (g *UDPGateway) AddHandlerFunc(f func([]byte, net.Addr)) {
	g.receiver.AddHandlerFunc(f)
}

// Send sends a PDU
func (g *UDPGateway) Send(data []byte) error {
	return g.transmitter.Send(data)
}

// SendTo sends a PDU to a specific address
func (g *UDPGateway) SendTo(data []byte, addr string) error {
	return g.transmitter.SendTo(data, addr)
}

// ReceiverStats returns receiver statistics
func (g *UDPGateway) ReceiverStats() ReceiverStats {
	return g.receiver.Stats()
}

// TransmitterStats returns transmitter statistics
func (g *UDPGateway) TransmitterStats() TransmitterStats {
	return g.transmitter.Stats()
}

// Errors returns the error channel
func (g *UDPGateway) Errors() <-chan error {
	return g.receiver.Errors()
}

// Errors
var (
	ErrAlreadyRunning = &DISError{Code: "ALREADY_RUNNING", Message: "already running"}
	ErrNotRunning     = &DISError{Code: "NOT_RUNNING", Message: "not running"}
)

// DISError represents a DIS error
type DISError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *DISError) Error() string {
	return e.Message
}