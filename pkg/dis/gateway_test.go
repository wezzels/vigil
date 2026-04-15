package dis

import (
	"net"
	"testing"
	"time"
)

// TestDefaultUDPConfig tests default configuration
func TestDefaultUDPConfig(t *testing.T) {
	config := DefaultUDPConfig()

	if config.Address != "239.1.2.3" {
		t.Errorf("Expected address 239.1.2.3, got %s", config.Address)
	}
	if config.Port != 3000 {
		t.Errorf("Expected port 3000, got %d", config.Port)
	}
	if config.BufferSize != 65536 {
		t.Errorf("Expected buffer size 65536, got %d", config.BufferSize)
	}
}

// TestNewUDPReceiver tests receiver creation
func TestNewUDPReceiver(t *testing.T) {
	receiver := NewUDPReceiver(nil)

	if receiver == nil {
		t.Fatal("Receiver should not be nil")
	}

	if receiver.config.Port != 3000 {
		t.Error("Default config should be used")
	}
}

// TestNewUDPTransmitter tests transmitter creation
func TestNewUDPTransmitter(t *testing.T) {
	transmitter := NewUDPTransmitter(nil)

	if transmitter == nil {
		t.Fatal("Transmitter should not be nil")
	}
}

// TestNewUDPGateway tests gateway creation
func TestNewUDPGateway(t *testing.T) {
	gateway := NewUDPGateway(nil)

	if gateway == nil {
		t.Fatal("Gateway should not be nil")
	}

	if gateway.receiver == nil {
		t.Error("Gateway receiver should not be nil")
	}

	if gateway.transmitter == nil {
		t.Error("Gateway transmitter should not be nil")
	}
}

// TestUDPReceiverStartStop tests start/stop
func TestUDPReceiverStartStop(t *testing.T) {
	config := &UDPConfig{
		Address:     "239.1.2.3",
		Port:        30000, // Use different port for test
		BufferSize:  65536,
		ReadTimeout: 10 * time.Millisecond,
	}

	receiver := NewUDPReceiver(config)

	err := receiver.Start()
	if err != nil {
		t.Errorf("Start failed: %v", err)
	}

	if !receiver.running {
		t.Error("Receiver should be running after start")
	}

	// Starting again should fail
	err = receiver.Start()
	if err != ErrAlreadyRunning {
		t.Errorf("Expected ErrAlreadyRunning, got %v", err)
	}

	err = receiver.Stop()
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	if receiver.running {
		t.Error("Receiver should not be running after stop")
	}
}

// TestUDPTransmitterStartStop tests start/stop
func TestUDPTransmitterStartStop(t *testing.T) {
	config := &UDPConfig{
		Address:      "239.1.2.3",
		Port:         30001,
		BufferSize:   65536,
		WriteTimeout: 10 * time.Millisecond,
	}

	transmitter := NewUDPTransmitter(config)

	err := transmitter.Start()
	if err != nil {
		t.Errorf("Start failed: %v", err)
	}

	if !transmitter.running {
		t.Error("Transmitter should be running after start")
	}

	// Starting again should fail
	err = transmitter.Start()
	if err != ErrAlreadyRunning {
		t.Errorf("Expected ErrAlreadyRunning, got %v", err)
	}

	err = transmitter.Stop()
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	if transmitter.running {
		t.Error("Transmitter should not be running after stop")
	}
}

// TestUDPGatewayStartStop tests gateway start/stop
func TestUDPGatewayStartStop(t *testing.T) {
	config := &UDPConfig{
		Address:      "239.1.2.3",
		Port:         30002,
		BufferSize:   65536,
		ReadTimeout:  10 * time.Millisecond,
		WriteTimeout: 10 * time.Millisecond,
	}

	gateway := NewUDPGateway(config)

	err := gateway.Start()
	if err != nil {
		t.Errorf("Start failed: %v", err)
	}

	err = gateway.Stop()
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}
}

// TestUDPReceiverStats tests statistics
func TestUDPReceiverStats(t *testing.T) {
	receiver := NewUDPReceiver(nil)

	stats := receiver.Stats()

	if stats.PacketsReceived != 0 {
		t.Error("Initial packets received should be 0")
	}

	if stats.BytesReceived != 0 {
		t.Error("Initial bytes received should be 0")
	}
}

// TestUDPTransmitterStats tests transmitter statistics
func TestUDPTransmitterStats(t *testing.T) {
	transmitter := NewUDPTransmitter(nil)

	stats := transmitter.Stats()

	if stats.PacketsSent != 0 {
		t.Error("Initial packets sent should be 0")
	}

	if stats.BytesSent != 0 {
		t.Error("Initial bytes sent should be 0")
	}
}

// TestUDPReceiverAddHandler tests adding handlers
func TestUDPReceiverAddHandler(t *testing.T) {
	receiver := NewUDPReceiver(nil)

	called := false
	receiver.AddHandlerFunc(func(data []byte, addr net.Addr) {
		called = true
	})

	if len(receiver.handlers) != 1 {
		t.Error("Handler should be added")
	}

	// Call handler manually
	receiver.handlers[0].HandlePDU([]byte("test"), nil)

	if !called {
		t.Error("Handler should have been called")
	}
}

// TestUDPTransmitterSendNotRunning tests sending when not running
func TestUDPTransmitterSendNotRunning(t *testing.T) {
	transmitter := NewUDPTransmitter(nil)

	err := transmitter.Send([]byte("test"))
	if err != ErrNotRunning {
		t.Errorf("Expected ErrNotRunning, got %v", err)
	}
}

// TestUDPTransmitterSendToNotRunning tests sending when not running
func TestUDPTransmitterSendToNotRunning(t *testing.T) {
	transmitter := NewUDPTransmitter(nil)

	err := transmitter.SendTo([]byte("test"), "239.1.2.3:3000")
	if err != ErrNotRunning {
		t.Errorf("Expected ErrNotRunning, got %v", err)
	}
}

// TestPDUHandlerFunc tests handler function
func TestPDUHandlerFunc(t *testing.T) {
	called := false
	handler := PDUHandlerFunc(func(data []byte, addr net.Addr) {
		called = true
	})

	handler.HandlePDU([]byte("test"), nil)

	if !called {
		t.Error("Handler should have been called")
	}
}

// TestDISError tests DIS error
func TestDISError(t *testing.T) {
	err := ErrAlreadyRunning

	if err.Code != "ALREADY_RUNNING" {
		t.Errorf("Error code should be ALREADY_RUNNING, got %s", err.Code)
	}

	if err.Error() == "" {
		t.Error("Error message should not be empty")
	}
}

// TestUDPGatewayAddHandler tests gateway handler
func TestUDPGatewayAddHandler(t *testing.T) {
	gateway := NewUDPGateway(nil)

	called := false
	gateway.AddHandlerFunc(func(data []byte, addr net.Addr) {
		called = true
	})

	if len(gateway.receiver.handlers) != 1 {
		t.Error("Handler should be added to receiver")
	}

	gateway.receiver.handlers[0].HandlePDU([]byte("test"), nil)

	if !called {
		t.Error("Handler should have been called")
	}
}

// TestUDPGatewaySend tests gateway send
func TestUDPGateway(t *testing.T) {
	config := &UDPConfig{
		Address:      "239.1.2.3",
		Port:         30003,
		BufferSize:   65536,
		ReadTimeout:  10 * time.Millisecond,
		WriteTimeout: 10 * time.Millisecond,
	}

	// Start receiver
	receiver := NewUDPReceiver(config)
	err := receiver.Start()
	if err != nil {
		t.Fatalf("Receiver start failed: %v", err)
	}

	// Start transmitter
	transmitter := NewUDPTransmitter(config)
	err = transmitter.Start()
	if err != nil {
		receiver.Stop()
		t.Fatalf("Transmitter start failed: %v", err)
	}

	// Give time for sockets to bind
	time.Sleep(10 * time.Millisecond)

	// Stop
	receiver.Stop()
	transmitter.Stop()
}

// BenchmarkUDPReceiverAddHandler benchmarks adding handlers
func BenchmarkUDPReceiverAddHandler(b *testing.B) {
	receiver := NewUDPReceiver(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		receiver.AddHandlerFunc(func(data []byte, addr net.Addr) {})
	}
}

// BenchmarkUDPTransmitterSend benchmarks sending (not running)
func BenchmarkUDPTransmitterSend(b *testing.B) {
	transmitter := NewUDPTransmitter(nil)
	transmitter.Start()
	data := make([]byte, 1500)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		transmitter.Send(data)
	}

	transmitter.Stop()
}

// BenchmarkPDUHandlerFunc benchmarks handler function
func BenchmarkPDUHandlerFunc(b *testing.B) {
	handler := PDUHandlerFunc(func(data []byte, addr net.Addr) {
		// Empty handler
	})

	data := []byte("test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.HandlePDU(data, nil)
	}
}
