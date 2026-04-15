package c2bmc

import (
	"context"
	"fmt"
	"time"
)

// AlertFormatter formats alerts for C2BMC submission
type AlertFormatter struct {
	sourceSystem string
}

// NewAlertFormatter creates a new alert formatter
func NewAlertFormatter(sourceSystem string) *AlertFormatter {
	return &AlertFormatter{
		sourceSystem: sourceSystem,
	}
}

// FormatLaunchAlert formats a launch detection alert
func (f *AlertFormatter) FormatLaunchAlert(track *TrackData) *AlertRequest {
	priority := AlertPriorityHigh
	if track.Confidence > 0.9 {
		priority = AlertPriorityCritical
	}

	return &AlertRequest{
		AlertType:    AlertTypeLaunch,
		Priority:     priority,
		TrackData:    track,
		SourceSystem: f.sourceSystem,
		Timestamp:    time.Now(),
		Message:      fmt.Sprintf("Launch detected from %s: Track %s", track.Source, track.TrackNumber),
	}
}

// FormatImpactAlert formats an impact prediction alert
func (f *AlertFormatter) FormatImpactAlert(track *TrackData, impact *Position) *AlertRequest {
	priority := AlertPriorityImminent
	if track.Confidence < 0.7 {
		priority = AlertPriorityCritical
	}

	trackCopy := *track
	trackCopy.PredictedImpact = impact

	return &AlertRequest{
		AlertType:    AlertTypeImpact,
		Priority:     priority,
		TrackData:    &trackCopy,
		SourceSystem: f.sourceSystem,
		Timestamp:    time.Now(),
		Message:      fmt.Sprintf("Impact predicted: Lat %.2f, Lon %.2f", impact.Latitude, impact.Longitude),
	}
}

// FormatTrackAlert formats a track update alert
func (f *AlertFormatter) FormatTrackAlert(track *TrackData, reason string) *AlertRequest {
	priority := AlertPriorityNormal
	if track.Identity == TrackIdentityHostile || track.Identity == TrackIdentityAssumedHostile {
		priority = AlertPriorityHigh
	}

	return &AlertRequest{
		AlertType:    AlertTypeTrack,
		Priority:     priority,
		TrackData:    track,
		SourceSystem: f.sourceSystem,
		Timestamp:    time.Now(),
		Message:      fmt.Sprintf("Track %s: %s", track.TrackNumber, reason),
	}
}

// FormatSystemAlert formats a system-level alert
func (f *AlertFormatter) FormatSystemAlert(priority AlertPriority, message string) *AlertRequest {
	return &AlertRequest{
		AlertType:    AlertTypeSystem,
		Priority:     priority,
		SourceSystem: f.sourceSystem,
		Timestamp:    time.Now(),
		Message:      message,
	}
}

// AlertSubmitter handles alert submission with acknowledgment tracking
type AlertSubmitter struct {
	client        C2BMCClient
	pendingAlerts map[string]*AlertRequest
	timeout       time.Duration
}

// NewAlertSubmitter creates a new alert submitter
func NewAlertSubmitter(client C2BMCClient, timeout time.Duration) *AlertSubmitter {
	return &AlertSubmitter{
		client:        client,
		pendingAlerts: make(map[string]*AlertRequest),
		timeout:       timeout,
	}
}

// Submit submits an alert and tracks for acknowledgment
func (s *AlertSubmitter) Submit(ctx context.Context, req *AlertRequest) (*AlertResponse, error) {
	resp, err := s.client.SubmitAlert(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to submit alert: %w", err)
	}

	// Track pending alert
	if resp.Status == AlertStatusPending || resp.Status == AlertStatusAcknowledged {
		s.pendingAlerts[resp.AlertID] = req
	}

	return resp, nil
}

// WaitForAcknowledgment waits for an alert to be acknowledged
func (s *AlertSubmitter) WaitForAcknowledgment(ctx context.Context, alertID string) (*AlertResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for acknowledgment: %w", ctx.Err())
		case <-ticker.C:
			resp, err := s.client.GetAlertStatus(ctx, alertID)
			if err != nil {
				continue // Retry on error
			}

			if resp.Status == AlertStatusAcknowledged ||
				resp.Status == AlertStatusProcessing ||
				resp.Status == AlertStatusComplete {
				delete(s.pendingAlerts, alertID)
				return resp, nil
			}

			if resp.Status == AlertStatusFailed || resp.Status == AlertStatusCanceled {
				delete(s.pendingAlerts, alertID)
				return resp, fmt.Errorf("alert failed or canceled")
			}
		}
	}
}

// Cancel cancels a pending alert
func (s *AlertSubmitter) Cancel(ctx context.Context, alertID string) error {
	if err := s.client.CancelAlert(ctx, alertID); err != nil {
		return fmt.Errorf("failed to cancel alert: %w", err)
	}

	delete(s.pendingAlerts, alertID)
	return nil
}

// GetPending returns count of pending alerts
func (s *AlertSubmitter) GetPending() int {
	return len(s.pendingAlerts)
}

// AcknowledgmentHandler handles alert acknowledgments
type AcknowledgmentHandler struct {
	onAcknowledge func(alertID string, acknowledgedBy string)
	onReject      func(alertID string, reason string)
	onTimeout     func(alertID string)
	onComplete    func(alertID string)
}

// NewAcknowledgmentHandler creates a new acknowledgment handler
func NewAcknowledgmentHandler() *AcknowledgmentHandler {
	return &AcknowledgmentHandler{}
}

// OnAcknowledge sets the acknowledgment callback
func (h *AcknowledgmentHandler) OnAcknowledge(fn func(alertID, acknowledgedBy string)) *AcknowledgmentHandler {
	h.onAcknowledge = fn
	return h
}

// OnReject sets the rejection callback
func (h *AcknowledgmentHandler) OnReject(fn func(alertID, reason string)) *AcknowledgmentHandler {
	h.onReject = fn
	return h
}

// OnTimeout sets the timeout callback
func (h *AcknowledgmentHandler) OnTimeout(fn func(alertID string)) *AcknowledgmentHandler {
	h.onTimeout = fn
	return h
}

// OnComplete sets the complete callback
func (h *AcknowledgmentHandler) OnComplete(fn func(alertID string)) *AcknowledgmentHandler {
	h.onComplete = fn
	return h
}

// Process processes an acknowledgment response
func (h *AcknowledgmentHandler) Process(resp *AlertResponse) {
	switch resp.Status {
	case AlertStatusAcknowledged, AlertStatusProcessing:
		if h.onAcknowledge != nil {
			h.onAcknowledge(resp.AlertID, resp.AcknowledgedBy)
		}
	case AlertStatusComplete:
		if h.onComplete != nil {
			h.onComplete(resp.AlertID)
		}
	case AlertStatusFailed, AlertStatusCanceled:
		if h.onReject != nil {
			h.onReject(resp.AlertID, resp.Message)
		}
	}
}

// AlertStats tracks alert statistics
type AlertStats struct {
	TotalSubmitted    int64
	TotalAcknowledged int64
	TotalFailed       int64
	TotalCanceled     int64
	ByType            map[AlertType]int64
	ByPriority        map[AlertPriority]int64
	LastSubmitTime    time.Time
}

// NewAlertStats creates new alert stats
func NewAlertStats() *AlertStats {
	return &AlertStats{
		ByType:     make(map[AlertType]int64),
		ByPriority: make(map[AlertPriority]int64),
	}
}

// Record records an alert submission
func (s *AlertStats) Record(alert *AlertRequest, resp *AlertResponse) {
	s.TotalSubmitted++
	s.ByType[alert.AlertType]++
	s.ByPriority[alert.Priority]++
	s.LastSubmitTime = time.Now()

	if resp != nil {
		switch resp.Status {
		case AlertStatusAcknowledged, AlertStatusProcessing:
			s.TotalAcknowledged++
		case AlertStatusFailed:
			s.TotalFailed++
		case AlertStatusCanceled:
			s.TotalCanceled++
		}
	}
}
