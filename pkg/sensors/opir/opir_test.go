package opir

import (
	"context"
	"encoding/binary"
	"testing"
	"time"
)

// TestOPIRSighting tests sighting structure
func TestOPIRSighting(t *testing.T) {
	sighting := OPIRSighting{
		ID:          "TEST-001",
		SensorID:    "SBIRS-GEO-1",
		SequenceNum: 1,
		Timestamp:   time.Now(),
		Latitude:    38.8977,
		Longitude:   -77.0365,
		Altitude:    50000.0,
		Confidence:  0.95,
	}

	if sighting.ID != "TEST-001" {
		t.Errorf("Expected ID TEST-001, got %s", sighting.ID)
	}
	if sighting.SensorID != "SBIRS-GEO-1" {
		t.Errorf("Expected SensorID SBIRS-GEO-1, got %s", sighting.SensorID)
	}
	if sighting.Latitude != 38.8977 {
		t.Errorf("Expected Latitude 38.8977, got %f", sighting.Latitude)
	}
}

// TestOPIRConfig tests configuration
func TestOPIRConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Port != 5000 {
		t.Errorf("Expected default port 5000, got %d", config.Port)
	}
	if config.MinConfidence != 0.5 {
		t.Errorf("Expected min confidence 0.5, got %f", config.MinConfidence)
	}
	if config.ConnectTimeout != 30*time.Second {
		t.Errorf("Expected connect timeout 30s, got %v", config.ConnectTimeout)
	}
}

// TestConfigValidation tests configuration validation
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *OPIRConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  &OPIRConfig{Endpoints: []string{"localhost"}, ConnectTimeout: time.Second, ReadTimeout: time.Second, MinConfidence: 0.5},
			wantErr: false,
		},
		{
			name:    "no endpoints",
			config:  &OPIRConfig{ConnectTimeout: time.Second, ReadTimeout: time.Second},
			wantErr: true,
		},
		{
			name:    "invalid confidence",
			config:  &OPIRConfig{Endpoints: []string{"localhost"}, ConnectTimeout: time.Second, ReadTimeout: time.Second, MinConfidence: 2.0},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidator tests sighting validation
func TestValidator(t *testing.T) {
	config := DefaultConfig()
	validator := NewValidator(config)

	tests := []struct {
		name     string
		sighting *OPIRSighting
		wantErr  bool
	}{
		{
			name: "valid sighting",
			sighting: &OPIRSighting{
				Latitude:   38.8977,
				Longitude:  -77.0365,
				Altitude:   50000.0,
				Confidence: 0.95,
				SNR:        20.0,
				Intensity:  1e-6,
				Timestamp:  time.Now(),
			},
			wantErr: false,
		},
		{
			name: "invalid latitude",
			sighting: &OPIRSighting{
				Latitude:   100.0,
				Longitude:  -77.0365,
				Altitude:   50000.0,
				Confidence: 0.95,
				SNR:        20.0,
				Timestamp:  time.Now(),
			},
			wantErr: true,
		},
		{
			name: "invalid longitude",
			sighting: &OPIRSighting{
				Latitude:   38.8977,
				Longitude:  200.0,
				Altitude:   50000.0,
				Confidence: 0.95,
				SNR:        20.0,
				Timestamp:  time.Now(),
			},
			wantErr: true,
		},
		{
			name: "low confidence",
			sighting: &OPIRSighting{
				Latitude:   38.8977,
				Longitude:  -77.0365,
				Altitude:   50000.0,
				Confidence: 0.1,
				SNR:        20.0,
				Timestamp:  time.Now(),
			},
			wantErr: true,
		},
		{
			name: "old timestamp",
			sighting: &OPIRSighting{
				Latitude:   38.8977,
				Longitude:  -77.0365,
				Altitude:   50000.0,
				Confidence: 0.95,
				SNR:        20.0,
				Timestamp:  time.Now().Add(-48 * time.Hour),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.sighting)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestFilter tests sighting filtering
func TestFilter(t *testing.T) {
	config := DefaultConfig()
	filter := NewFilter(config)

	sightings := []OPIRSighting{
		{Latitude: 38.8977, Longitude: -77.0365, Altitude: 50000, Confidence: 0.95, SNR: 20.0},
		{Latitude: 38.8977, Longitude: -77.0365, Altitude: 50000, Confidence: 0.3, SNR: 20.0}, // Low confidence
		{Latitude: 38.8977, Longitude: -77.0365, Altitude: 50000, Confidence: 0.95, SNR: 5.0}, // Low SNR
	}

	result := filter.FilterBatch(sightings)

	if len(result) != 1 {
		t.Errorf("Expected 1 filtered sighting, got %d", len(result))
	}
}

// TestCircuitBreaker tests circuit breaker
func TestCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(3, 1*time.Second)

	// Should start closed
	if cb.State() != StateClosed {
		t.Error("Circuit should start closed")
	}

	// Record failures
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()

	// Should be open after max failures
	if cb.State() != StateOpen {
		t.Error("Circuit should be open after max failures")
	}

	// Should not allow requests
	if cb.Allow() {
		t.Error("Circuit should not allow requests when open")
	}

	// Wait for timeout
	time.Sleep(1100 * time.Millisecond)

	// Should transition to half-open
	if !cb.Allow() {
		t.Error("Circuit should allow test request after timeout")
	}

	// Record success
	cb.RecordSuccess()
	cb.RecordSuccess()
	cb.RecordSuccess()

	// Should be closed after success threshold
	if cb.State() != StateClosed {
		t.Error("Circuit should be closed after success threshold")
	}
}

// TestReconnector tests reconnection logic
func TestReconnector(t *testing.T) {
	config := &OPIRConfig{
		MaxRetries:    5,
		RetryDelay:    100 * time.Millisecond,
		MaxRetryDelay: 1 * time.Second,
	}

	reconn := NewReconnector(config)

	// Test backoff calculation
	backoff1 := reconn.NextBackoff()
	backoff2 := reconn.NextBackoff()
	backoff3 := reconn.NextBackoff()

	if backoff2 <= backoff1 {
		t.Error("Backoff should increase")
	}
	if backoff3 <= backoff2 {
		t.Error("Backoff should increase")
	}

	// Test reset
	reconn.RecordSuccess()
	if reconn.Attempts() != 0 {
		t.Error("Attempts should be 0 after success")
	}
}

// TestOPIRError tests error types
func TestOPIRError(t *testing.T) {
	err := NewConnectionError("test error", true)

	if err.Code != ErrCodeConnection {
		t.Errorf("Expected code %s, got %s", ErrCodeConnection, err.Code)
	}
	if !err.Retryable {
		t.Error("Connection error should be retryable")
	}

	authErr := NewAuthenticationError("auth failed")
	if authErr.Retryable {
		t.Error("Auth error should not be retryable")
	}
}

// TestSBIRSFeed tests SBIRS feed creation
func TestSBIRSFeed(t *testing.T) {
	config := DefaultConfig()
	config.Endpoints = []string{"localhost"}

	feed, err := NewSBIRSFeed(config)
	if err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	if feed == nil {
		t.Fatal("Feed should not be nil")
	}

	if feed.IsConnected() {
		t.Error("Feed should not be connected initially")
	}
}

// TestNGOPIRFeed tests NG-OPIR feed creation
func TestNGOPIRFeed(t *testing.T) {
	config := DefaultConfig()
	config.Endpoints = []string{"localhost"}

	feed, err := NewNGOPIRFeed(config)
	if err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	if feed == nil {
		t.Fatal("Feed should not be nil")
	}
}

// TestFeedStats tests feed statistics
func TestFeedStats(t *testing.T) {
	stats := FeedStats{
		Connected:      true,
		TotalReceived:  1000,
		TotalErrors:    5,
		ReceiveRate:    100.5,
		ReconnectCount: 2,
	}

	if !stats.Connected {
		t.Error("Stats should show connected")
	}
	if stats.TotalReceived != 1000 {
		t.Errorf("Expected 1000 received, got %d", stats.TotalReceived)
	}
}

// TestSBIRSMessageParsing tests SBIRS message parsing
func TestSBIRSMessageParsing(t *testing.T) {
	feed := &SBIRSFeed{
		config: DefaultConfig(),
	}

	// Create test message header
	header := make([]byte, 32)
	binary.BigEndian.PutUint32(header[0:4], 0x53424952) // "SBIR"
	binary.BigEndian.PutUint16(header[4:6], 1)          // Version
	binary.BigEndian.PutUint16(header[6:8], 1)          // Type
	binary.BigEndian.PutUint16(header[8:10], 128)       // Data length
	binary.BigEndian.PutUint16(header[10:12], 1)        // Sensor ID
	binary.BigEndian.PutUint32(header[12:16], 1)        // Sequence
	binary.BigEndian.PutUint64(header[16:24], uint64(time.Now().UnixNano()))

	sighting, dataLen, err := feed.parseHeader(header)
	if err != nil {
		t.Fatalf("Failed to parse header: %v", err)
	}

	if sighting.SensorID != "SBIRS-GEO-1" {
		t.Errorf("Expected SensorID SBIRS-GEO-1, got %s", sighting.SensorID)
	}
	if dataLen != 128 {
		t.Errorf("Expected data length 128, got %d", dataLen)
	}
}

// TestMockFeed tests mock feed for testing
type MockFeed struct {
	sightings chan OPIRSighting
	errors    chan error
	connected bool
	stats     FeedStats
}

func NewMockFeed() *MockFeed {
	return &MockFeed{
		sightings: make(chan OPIRSighting, 100),
		errors:    make(chan error, 10),
	}
}

func (m *MockFeed) Connect(ctx context.Context) error {
	m.connected = true
	m.stats.Connected = true
	return nil
}

func (m *MockFeed) Disconnect() error {
	m.connected = false
	m.stats.Connected = false
	return nil
}

func (m *MockFeed) Receive() <-chan OPIRSighting {
	return m.sightings
}

func (m *MockFeed) Errors() <-chan error {
	return m.errors
}

func (m *MockFeed) IsConnected() bool {
	return m.connected
}

func (m *MockFeed) Stats() FeedStats {
	return m.stats
}

func (m *MockFeed) Send(sighting OPIRSighting) {
	m.sightings <- sighting
	m.stats.TotalReceived++
}

// TestMockFeedUsage tests using mock feed
func TestMockFeedUsage(t *testing.T) {
	feed := NewMockFeed()

	ctx := context.Background()
	if err := feed.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	if !feed.IsConnected() {
		t.Error("Feed should be connected")
	}

	// Send a sighting
	sighting := OPIRSighting{
		ID:         "TEST-001",
		SensorID:   "MOCK-1",
		Timestamp:  time.Now(),
		Latitude:   38.8977,
		Longitude:  -77.0365,
		Altitude:   50000,
		Confidence: 0.95,
	}

	feed.Send(sighting)

	// Receive sighting
	select {
	case s := <-feed.Receive():
		if s.ID != "TEST-001" {
			t.Errorf("Expected ID TEST-001, got %s", s.ID)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for sighting")
	}

	if err := feed.Disconnect(); err != nil {
		t.Fatalf("Failed to disconnect: %v", err)
	}
}

// BenchmarkValidator benchmarks validation
func BenchmarkValidator(b *testing.B) {
	config := DefaultConfig()
	validator := NewValidator(config)

	sighting := &OPIRSighting{
		Latitude:   38.8977,
		Longitude:  -77.0365,
		Altitude:   50000.0,
		Confidence: 0.95,
		SNR:        20.0,
		Intensity:  1e-6,
		Timestamp:  time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.Validate(sighting)
	}
}

// BenchmarkFilter benchmarks filtering
func BenchmarkFilter(b *testing.B) {
	config := DefaultConfig()
	filter := NewFilter(config)

	sightings := make([]OPIRSighting, 100)
	for i := range sightings {
		sightings[i] = OPIRSighting{
			Latitude:   38.8977,
			Longitude:  -77.0365,
			Altitude:   50000.0,
			Confidence: 0.95,
			SNR:        20.0,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filter.FilterBatch(sightings)
	}
}

// TestConnectionPool tests connection pool
func TestConnectionPool(t *testing.T) {
	feed1 := NewMockFeed()
	feed2 := NewMockFeed()

	pool := NewConnectionPool(feed1, feed2)

	// Test round-robin
	f1 := pool.Get()
	f2 := pool.Get()

	if f1 == f2 {
		t.Error("Pool should return different feeds")
	}

	// Test all
	all := pool.All()
	if len(all) != 2 {
		t.Errorf("Expected 2 feeds, got %d", len(all))
	}
}

// TestHealthChecker tests health checking
func TestHealthChecker(t *testing.T) {
	config := DefaultConfig()
	checker := NewHealthChecker(config)

	feed := NewMockFeed()
	feed.Connect(context.Background())

	// Health check should pass
	if !checker.IsHealthy() {
		t.Error("Health checker should be healthy initially")
	}
}
