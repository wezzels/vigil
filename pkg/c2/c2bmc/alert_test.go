package c2bmc

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// TestAlertFormatter tests alert formatting
func TestAlertFormatter(t *testing.T) {
	formatter := NewAlertFormatter("OPIR-SBIRS")

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

	// Test launch alert
	launchAlert := formatter.FormatLaunchAlert(track)
	if launchAlert.AlertType != AlertTypeLaunch {
		t.Errorf("Expected LAUNCH, got %d", launchAlert.AlertType)
	}
	if launchAlert.SourceSystem != "OPIR-SBIRS" {
		t.Errorf("Expected OPIR-SBIRS, got %s", launchAlert.SourceSystem)
	}
	if launchAlert.Priority != AlertPriorityCritical {
		t.Errorf("Expected CRITICAL (confidence > 0.9), got %d", launchAlert.Priority)
	}

	// Test impact alert
	impact := &Position{
		Latitude:  35.0,
		Longitude: -100.0,
		Altitude:  0.0,
	}
	impactAlert := formatter.FormatImpactAlert(track, impact)
	if impactAlert.AlertType != AlertTypeImpact {
		t.Errorf("Expected IMPACT, got %d", impactAlert.AlertType)
	}
	if impactAlert.TrackData.PredictedImpact == nil {
		t.Error("Expected predicted impact position")
	}
	if impactAlert.Priority != AlertPriorityImminent {
		t.Errorf("Expected IMMINENT, got %d", impactAlert.Priority)
	}

	// Test track alert
	trackAlert := formatter.FormatTrackAlert(track, "Hostile track detected")
	if trackAlert.AlertType != AlertTypeTrack {
		t.Errorf("Expected TRACK, got %d", trackAlert.AlertType)
	}
	if trackAlert.Priority != AlertPriorityHigh {
		t.Errorf("Expected HIGH (hostile), got %d", trackAlert.Priority)
	}

	// Test system alert
	sysAlert := formatter.FormatSystemAlert(AlertPriorityCritical, "System degraded")
	if sysAlert.AlertType != AlertTypeSystem {
		t.Errorf("Expected SYSTEM, got %d", sysAlert.AlertType)
	}
	if sysAlert.Message != "System degraded" {
		t.Errorf("Expected 'System degraded', got %s", sysAlert.Message)
	}
}

// TestAlertSubmitter tests alert submission
func TestAlertSubmitter(t *testing.T) {
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

	submitter := NewAlertSubmitter(client, 30*time.Second)

	req := &AlertRequest{
		AlertType:    AlertTypeLaunch,
		Priority:     AlertPriorityCritical,
		SourceSystem: "OPIR",
	}

	ctx := context.Background()
	resp, err := submitter.Submit(ctx, req)
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}

	if resp.Status != AlertStatusAcknowledged {
		t.Errorf("Expected ACKNOWLEDGED, got %d", resp.Status)
	}

	if submitter.GetPending() != 1 {
		t.Errorf("Expected 1 pending, got %d", submitter.GetPending())
	}
}

// TestAcknowledgmentHandler tests acknowledgment handling
func TestAcknowledgmentHandler(t *testing.T) {
	acknowledged := ""
	rejected := ""
	completed := ""

	handler := NewAcknowledgmentHandler().
		OnAcknowledge(func(id, by string) {
			acknowledged = id
		}).
		OnReject(func(id, reason string) {
			rejected = id
		}).
		OnComplete(func(id string) {
			completed = id
		})

	// Test acknowledgment
	handler.Process(&AlertResponse{
		AlertID:        "ALERT-001",
		Status:         AlertStatusAcknowledged,
		AcknowledgedBy: "OPERATOR-1",
	})

	if acknowledged != "ALERT-001" {
		t.Errorf("Expected ALERT-001, got %s", acknowledged)
	}

	// Test completion
	handler.Process(&AlertResponse{
		AlertID: "ALERT-002",
		Status:  AlertStatusComplete,
	})

	if completed != "ALERT-002" {
		t.Errorf("Expected ALERT-002, got %s", completed)
	}

	// Test rejection
	handler.Process(&AlertResponse{
		AlertID: "ALERT-003",
		Status:  AlertStatusFailed,
		Message: "Invalid track",
	})

	if rejected != "ALERT-003" {
		t.Errorf("Expected ALERT-003, got %s", rejected)
	}
}

// TestAlertStats tests alert statistics
func TestAlertStats(t *testing.T) {
	stats := NewAlertStats()

	// Record some alerts
	stats.Record(&AlertRequest{
		AlertType: AlertTypeLaunch,
		Priority:  AlertPriorityCritical,
	}, &AlertResponse{
		Status: AlertStatusAcknowledged,
	})

	stats.Record(&AlertRequest{
		AlertType: AlertTypeTrack,
		Priority:  AlertPriorityHigh,
	}, &AlertResponse{
		Status: AlertStatusFailed,
	})

	stats.Record(&AlertRequest{
		AlertType: AlertTypeImpact,
		Priority:  AlertPriorityImminent,
	}, &AlertResponse{
		Status: AlertStatusProcessing,
	})

	if stats.TotalSubmitted != 3 {
		t.Errorf("Expected 3 submissions, got %d", stats.TotalSubmitted)
	}

	if stats.TotalAcknowledged != 2 {
		t.Errorf("Expected 2 acknowledged, got %d", stats.TotalAcknowledged)
	}

	if stats.TotalFailed != 1 {
		t.Errorf("Expected 1 failed, got %d", stats.TotalFailed)
	}

	if stats.ByType[AlertTypeLaunch] != 1 {
		t.Errorf("Expected 1 LAUNCH, got %d", stats.ByType[AlertTypeLaunch])
	}

	if stats.ByPriority[AlertPriorityCritical] != 1 {
		t.Errorf("Expected 1 CRITICAL, got %d", stats.ByPriority[AlertPriorityCritical])
	}
}

// TestFormatLaunchAlertWithLowConfidence tests launch alert with low confidence
func TestFormatLaunchAlertWithLowConfidence(t *testing.T) {
	formatter := NewAlertFormatter("OPIR")

	track := &TrackData{
		TrackNumber: "T-001",
		Identity:    TrackIdentityHostile,
		Confidence:  0.5, // Low confidence
	}

	alert := formatter.FormatLaunchAlert(track)
	if alert.Priority != AlertPriorityHigh {
		t.Errorf("Expected HIGH (confidence <= 0.9), got %d", alert.Priority)
	}
}

// TestFormatImpactAlertWithLowConfidence tests impact alert with low confidence
func TestFormatImpactAlertWithLowConfidence(t *testing.T) {
	formatter := NewAlertFormatter("OPIR")

	track := &TrackData{
		TrackNumber: "T-001",
		Confidence:  0.5, // Low confidence
	}

	impact := &Position{Latitude: 35.0, Longitude: -100.0, Altitude: 0}
	alert := formatter.FormatImpactAlert(track, impact)

	if alert.Priority != AlertPriorityCritical {
		t.Errorf("Expected CRITICAL (confidence < 0.7), got %d", alert.Priority)
	}
}

// TestFormatTrackAlertFriendly tests friendly track alert
func TestFormatTrackAlertFriendly(t *testing.T) {
	formatter := NewAlertFormatter("OPIR")

	track := &TrackData{
		TrackNumber: "T-001",
		Identity:    TrackIdentityFriendly,
	}

	alert := formatter.FormatTrackAlert(track, "Friendly track update")
	if alert.Priority != AlertPriorityNormal {
		t.Errorf("Expected NORMAL (friendly), got %d", alert.Priority)
	}
}

// BenchmarkAlertFormatter benchmarks alert formatting
func BenchmarkAlertFormatter(b *testing.B) {
	formatter := NewAlertFormatter("OPIR-SBIRS")

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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatter.FormatLaunchAlert(track)
	}
}

// BenchmarkAlertSubmit benchmarks alert submission
func BenchmarkAlertSubmit(b *testing.B) {
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

	submitter := NewAlertSubmitter(client, 30*time.Second)

	req := &AlertRequest{
		AlertType:    AlertTypeLaunch,
		Priority:     AlertPriorityCritical,
		SourceSystem: "OPIR",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		submitter.Submit(ctx, req)
	}
}
