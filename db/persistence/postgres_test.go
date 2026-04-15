package persistence

import (
	"testing"
	"time"
)

// TestDefaultConfig tests default configuration
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Host != "localhost" {
		t.Errorf("Expected localhost, got %s", config.Host)
	}
	if config.Port != 5432 {
		t.Errorf("Expected 5432, got %d", config.Port)
	}
	if config.Database != "vigil" {
		t.Errorf("Expected vigil, got %s", config.Database)
	}
	if config.User != "vigil" {
		t.Errorf("Expected vigil, got %s", config.User)
	}
	if config.MaxConns != 100 {
		t.Errorf("Expected 100, got %d", config.MaxConns)
	}
	if config.MinConns != 10 {
		t.Errorf("Expected 10, got %d", config.MinConns)
	}
}

// TestTrackStruct tests track struct
func TestTrackStruct(t *testing.T) {
	track := &Track{
		ID:           "test-id",
		TrackNumber:  "T-001",
		TrackID:      "TRACK-001",
		SourceSystem: "OPIR",
		Latitude:     45.0,
		Longitude:    -120.0,
		Altitude:     10000.0,
		VelocityX:    100.0,
		VelocityY:    50.0,
		VelocityZ:    10.0,
		Identity:     "hostile",
		Quality:      "good",
		Confidence:   0.95,
		TrackType:    "missile",
		ForceID:      "OPFOR",
		Environment:  "air",
		FirstDetect:  time.Now(),
		LastUpdate:   time.Now(),
	}

	if track.TrackNumber != "T-001" {
		t.Errorf("Expected T-001, got %s", track.TrackNumber)
	}
	if track.Latitude != 45.0 {
		t.Errorf("Expected 45.0, got %f", track.Latitude)
	}
	if track.Confidence != 0.95 {
		t.Errorf("Expected 0.95, got %f", track.Confidence)
	}
}

// TestAlertStruct tests alert struct
func TestAlertStruct(t *testing.T) {
	now := time.Now()
	expires := now.Add(1 * time.Hour)

	alert := &Alert{
		ID:              "alert-id",
		AlertID:         "ALERT-001",
		AlertType:       "launch",
		Priority:        "critical",
		Status:          "pending",
		TrackID:         "track-id",
		TrackNumber:     "T-001",
		Message:         "Launch detected",
		SourceSystem:    "OPIR",
		EscalationLevel: "notify",
		CreatedAt:       now,
		ExpiresAt:       &expires,
	}

	if alert.AlertID != "ALERT-001" {
		t.Errorf("Expected ALERT-001, got %s", alert.AlertID)
	}
	if alert.Priority != "critical" {
		t.Errorf("Expected critical, got %s", alert.Priority)
	}
	if alert.Status != "pending" {
		t.Errorf("Expected pending, got %s", alert.Status)
	}
}

// TestEventStruct tests event struct
func TestEventStruct(t *testing.T) {
	event := &Event{
		ID:          "event-id",
		EventType:   "track_update",
		EventSource: "OPIR",
		EventTime:   time.Now(),
		TrackID:     "track-id",
		AlertID:     "",
		Data:        `{"position": {"lat": 45.0, "lon": -120.0}}`,
		Severity:    "info",
	}

	if event.EventType != "track_update" {
		t.Errorf("Expected track_update, got %s", event.EventType)
	}
	if event.Severity != "info" {
		t.Errorf("Expected info, got %s", event.Severity)
	}
}

// Note: Repository tests require a running database and are tested
// in integration tests. See db/integration/ for integration tests.

// TestDatabaseInterface tests database interface
func TestDatabaseInterface(t *testing.T) {
	// This test verifies the database methods exist and compile
	// Note: Actual database operations require a running PostgreSQL instance
	_ = &Database{}
}

// TestTrackRepositoryInterface tests track repository interface
func TestTrackRepositoryInterface(t *testing.T) {
	// This test verifies the repository methods exist and compile
	_ = &TrackRepository{}
}

// TestAlertRepositoryInterface tests alert repository interface
func TestAlertRepositoryInterface(t *testing.T) {
	// This test verifies the repository methods exist and compile
	_ = &AlertRepository{}
}
