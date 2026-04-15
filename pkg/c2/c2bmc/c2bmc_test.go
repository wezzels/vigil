package c2bmc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"
)

// TestNewClient tests client creation
func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *C2BMCConfig
		wantErr bool
	}{
		{
			name: "default config",
			config: &C2BMCConfig{
				Endpoint:   "https://test.example.mil:8443",
				EnableMTLS: false,
			},
		},
		{
			name: "custom config",
			config: &C2BMCConfig{
				Endpoint:   "https://test.example.mil:8443",
				Timeout:    60 * time.Second,
				MaxRetries: 5,
				RetryDelay: 2 * time.Second,
				EnableMTLS: false,
			},
		},
		{
			name: "mTLS without cert",
			config: &C2BMCConfig{
				Endpoint:   "https://test.example.mil:8443",
				EnableMTLS: true,
				CertFile:   "",
				KeyFile:    "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("NewClient() returned nil client")
			}
		})
	}
}

// TestNewClientWithMTLS tests mTLS client creation
func TestNewClientWithMTLS(t *testing.T) {
	// Skip this test as it requires valid certificates
	t.Skip("requires valid test certificates")
}

// TestPriorityStrings tests priority string conversion
func TestPriorityStrings(t *testing.T) {
	tests := []struct {
		priority AlertPriority
		expected string
	}{
		{AlertPriorityLow, "LOW"},
		{AlertPriorityNormal, "NORMAL"},
		{AlertPriorityHigh, "HIGH"},
		{AlertPriorityCritical, "CRITICAL"},
		{AlertPriorityImminent, "IMMINENT"},
		{AlertPriority(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := GetPriorityString(tt.priority)
			if result != tt.expected {
				t.Errorf("GetPriorityString(%d) = %s, want %s", tt.priority, result, tt.expected)
			}
		})
	}
}

// TestAlertTypeStrings tests alert type string conversion
func TestAlertTypeStrings(t *testing.T) {
	tests := []struct {
		alertType AlertType
		expected  string
	}{
		{AlertTypeTrack, "TRACK"},
		{AlertTypeLaunch, "LAUNCH"},
		{AlertTypeImpact, "IMPACT"},
		{AlertTypeCorrelation, "CORRELATION"},
		{AlertTypeSystem, "SYSTEM"},
		{AlertType(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := GetAlertTypeString(tt.alertType)
			if result != tt.expected {
				t.Errorf("GetAlertTypeString(%d) = %s, want %s", tt.alertType, result, tt.expected)
			}
		})
	}
}

// TestStatusStrings tests status string conversion
func TestStatusStrings(t *testing.T) {
	tests := []struct {
		status   AlertStatus
		expected string
	}{
		{AlertStatusPending, "PENDING"},
		{AlertStatusAcknowledged, "ACKNOWLEDGED"},
		{AlertStatusProcessing, "PROCESSING"},
		{AlertStatusComplete, "COMPLETE"},
		{AlertStatusFailed, "FAILED"},
		{AlertStatusCancelled, "CANCELLED"},
		{AlertStatus(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := GetStatusString(tt.status)
			if result != tt.expected {
				t.Errorf("GetStatusString(%d) = %s, want %s", tt.status, result, tt.expected)
			}
		})
	}
}

// TestIdentityStrings tests identity string conversion
func TestIdentityStrings(t *testing.T) {
	tests := []struct {
		identity TrackIdentity
		expected string
	}{
		{TrackIdentityUnknown, "UNKNOWN"},
		{TrackIdentityPending, "PENDING"},
		{TrackIdentityFriendly, "FRIENDLY"},
		{TrackIdentityHostile, "HOSTILE"},
		{TrackIdentityNeutral, "NEUTRAL"},
		{TrackIdentityAssumedHostile, "ASSUMED_HOSTILE"},
		{TrackIdentity(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := GetIdentityString(tt.identity)
			if result != tt.expected {
				t.Errorf("GetIdentityString(%d) = %s, want %s", tt.identity, result, tt.expected)
			}
		})
	}
}

// TestQualityStrings tests quality string conversion
func TestQualityStrings(t *testing.T) {
	tests := []struct {
		quality  TrackQuality
		expected string
	}{
		{TrackQualityUnknown, "UNKNOWN"},
		{TrackQualityPoor, "POOR"},
		{TrackQualityFair, "FAIR"},
		{TrackQualityGood, "GOOD"},
		{TrackQualityExcellent, "EXCELLENT"},
		{TrackQuality(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := GetQualityString(tt.quality)
			if result != tt.expected {
				t.Errorf("GetQualityString(%d) = %s, want %s", tt.quality, result, tt.expected)
			}
		})
	}
}

// TestC2BMCError tests error handling
func TestC2BMCError(t *testing.T) {
	err := &C2BMCError{
		Code:    500,
		Message: "Internal Server Error",
		Detail:  "Database connection failed",
	}

	expected := "Internal Server Error: Database connection failed"
	if err.Error() != expected {
		t.Errorf("C2BMCError.Error() = %s, want %s", err.Error(), expected)
	}

	if !err.IsRetryable() {
		t.Error("C2BMCError(500) should be retryable")
	}

	err2 := &C2BMCError{
		Code:    400,
		Message: "Bad Request",
	}
	if err2.IsRetryable() {
		t.Error("C2BMCError(400) should not be retryable")
	}

	err3 := &C2BMCError{
		Code:    429,
		Message: "Rate Limited",
	}
	if !err3.IsRetryable() {
		t.Error("C2BMCError(429) should be retryable")
	}
}

// TestDefaultConfig tests default configuration
func TestDefaultConfig(t *testing.T) {
	config := DefaultC2BMCConfig()

	if config.Endpoint == "" {
		t.Error("Default endpoint should not be empty")
	}
	if config.Timeout == 0 {
		t.Error("Default timeout should not be zero")
	}
	if config.MaxRetries == 0 {
		t.Error("Default max retries should not be zero")
	}
	if !config.EnableMTLS {
		t.Error("Default should enable mTLS")
	}
}

// TestAlertRequest tests alert request
func TestAlertRequest(t *testing.T) {
	req := &AlertRequest{
		AlertID:      "ALERT-001",
		AlertType:    AlertTypeLaunch,
		Priority:     AlertPriorityCritical,
		SourceSystem: "OPIR",
		Timestamp:    time.Now(),
	}

	if req.AlertID != "ALERT-001" {
		t.Errorf("AlertID = %s, want ALERT-001", req.AlertID)
	}
	if req.AlertType != AlertTypeLaunch {
		t.Errorf("AlertType = %d, want %d", req.AlertType, AlertTypeLaunch)
	}
}

// TestTrackData tests track data
func TestTrackData(t *testing.T) {
	track := &TrackData{
		TrackNumber: "T-001",
		TrackID:     "TRACK-001",
		Position: Position{
			Latitude:  45.0,
			Longitude: -120.0,
			Altitude:  10000.0,
		},
		Velocity: Velocity{
			Vx: 100.0,
			Vy: 50.0,
			Vz: 10.0,
		},
		Identity:   TrackIdentityHostile,
		Quality:    TrackQualityGood,
		Source:     "SBIRS",
		Confidence: 0.95,
	}

	if track.TrackNumber != "T-001" {
		t.Errorf("TrackNumber = %s, want T-001", track.TrackNumber)
	}
	if track.Identity != TrackIdentityHostile {
		t.Errorf("Identity = %d, want %d", track.Identity, TrackIdentityHostile)
	}
}

// mockServer creates a mock HTTP server for testing
func mockServer(handler http.HandlerFunc) (*http.Server, string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", err
	}

	server := &http.Server{
		Handler: handler,
	}

	go server.Serve(listener)

	addr := listener.Addr().(*net.TCPAddr)
	url := fmt.Sprintf("http://%s", addr.String())

	return server, url, nil
}

// TestClientSubmitAlert tests alert submission
func TestClientSubmitAlert(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/alerts" {
			t.Errorf("Expected /api/v1/alerts, got %s", r.URL.Path)
		}

		var req AlertRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		resp := AlertResponse{
			AlertID: req.AlertID,
			Status:  AlertStatusAcknowledged,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	server, url, err := mockServer(handler)
	if err != nil {
		t.Fatalf("Failed to create mock server: %v", err)
	}
	defer server.Close()

	config := &C2BMCConfig{
		Endpoint: url,
		Timeout:  5 * time.Second,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	req := &AlertRequest{
		AlertID:      "ALERT-001",
		AlertType:    AlertTypeLaunch,
		Priority:     AlertPriorityCritical,
		SourceSystem: "OPIR",
	}

	ctx := context.Background()
	resp, err := client.SubmitAlert(ctx, req)
	if err != nil {
		t.Fatalf("SubmitAlert() error = %v", err)
	}

	if resp.AlertID != "ALERT-001" {
		t.Errorf("AlertID = %s, want ALERT-001", resp.AlertID)
	}
	if resp.Status != AlertStatusAcknowledged {
		t.Errorf("Status = %d, want %d", resp.Status, AlertStatusAcknowledged)
	}
}

// TestClientHealthCheck tests health check
func TestClientHealthCheck(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("Expected /health, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}

	server, url, err := mockServer(handler)
	if err != nil {
		t.Fatalf("Failed to create mock server: %v", err)
	}
	defer server.Close()

	config := &C2BMCConfig{
		Endpoint: url,
		Timeout:  5 * time.Second,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	if err := client.HealthCheck(ctx); err != nil {
		t.Errorf("HealthCheck() error = %v", err)
	}
}

// Test client with retry
func TestClientRetry(t *testing.T) {
	attempts := 0

	handler := func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}

	server, url, err := mockServer(handler)
	if err != nil {
		t.Fatalf("Failed to create mock server: %v", err)
	}
	defer server.Close()

	config := &C2BMCConfig{
		Endpoint:   url,
		Timeout:    5 * time.Second,
		MaxRetries: 3,
		RetryDelay: 10 * time.Millisecond,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	if err := client.HealthCheck(ctx); err != nil {
		t.Errorf("HealthCheck() error = %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

// BenchmarkSubmitAlert benchmarks alert submission
func BenchmarkSubmitAlert(b *testing.B) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		resp := AlertResponse{
			AlertID: "ALERT-001",
			Status:  AlertStatusAcknowledged,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	server, url, err := mockServer(handler)
	if err != nil {
		b.Fatalf("Failed to create mock server: %v", err)
	}
	defer server.Close()

	config := &C2BMCConfig{
		Endpoint: url,
		Timeout:  5 * time.Second,
	}

	client, err := NewClient(config)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}

	req := &AlertRequest{
		AlertID:      "ALERT-001",
		AlertType:    AlertTypeLaunch,
		Priority:     AlertPriorityCritical,
		SourceSystem: "OPIR",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.SubmitAlert(ctx, req)
	}
}

// BenchmarkGetTrack benchmarks track retrieval
func BenchmarkGetTrack(b *testing.B) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		track := &TrackData{
			TrackNumber: "T-001",
			TrackID:     "TRACK-001",
			Position: Position{
				Latitude:  45.0,
				Longitude: -120.0,
				Altitude:  10000.0,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(track)
	}

	server, url, err := mockServer(handler)
	if err != nil {
		b.Fatalf("Failed to create mock server: %v", err)
	}
	defer server.Close()

	config := &C2BMCConfig{
		Endpoint: url,
		Timeout:  5 * time.Second,
	}

	client, err := NewClient(config)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.GetTrack(ctx, "TRACK-001")
	}
}
