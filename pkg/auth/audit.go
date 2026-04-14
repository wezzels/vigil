// Package auth provides audit logging
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// AuditEventType represents the type of audit event
type AuditEventType string

const (
	EventTypeLogin        AuditEventType = "login"
	EventTypeLogout       AuditEventType = "logout"
	EventTypeAccess       AuditEventType = "access"
	EventTypeCreate       AuditEventType = "create"
	EventTypeUpdate       AuditEventType = "update"
	EventTypeDelete       AuditEventType = "delete"
	EventTypeAuthenticate AuditEventType = "authenticate"
	EventTypeAuthorize    AuditEventType = "authorize"
	EventTypeKeyGenerate  AuditEventType = "key_generate"
	EventTypeKeyRotate    AuditEventType = "key_rotate"
	EventTypeKeyRevoke    AuditEventType = "key_revoke"
	EventTypeRoleAssign   AuditEventType = "role_assign"
	EventTypeRoleRevoke   AuditEventType = "role_revoke"
)

// AuditResult represents the result of an audit event
type AuditResult string

const (
	ResultSuccess AuditResult = "success"
	ResultFailure AuditResult = "failure"
	ResultDenied  AuditResult = "denied"
)

// AuditEvent represents an audit event
type AuditEvent struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	Type        AuditEventType         `json:"type"`
	Result      AuditResult            `json:"result"`
	UserID      string                 `json:"user_id,omitempty"`
	ActorID     string                 `json:"actor_id,omitempty"`
	Resource    string                 `json:"resource,omitempty"`
	Action      string                 `json:"action,omitempty"`
	IPAddress   string                 `json:"ip_address,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	Method      string                 `json:"method,omitempty"`
	Path        string                 `json:"path,omitempty"`
	StatusCode  int                    `json:"status_code,omitempty"`
	Duration    time.Duration          `json:"duration,omitempty"`
	Message     string                 `json:"message,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Metadata    map[string]string      `json:"metadata,omitempty"`
}

// AuditConfig holds audit configuration
type AuditConfig struct {
	MaxEvents      int
	FlushInterval  time.Duration
	RetentionDays  int
	IncludeHeaders bool
}

// DefaultAuditConfig returns default audit configuration
func DefaultAuditConfig() *AuditConfig {
	return &AuditConfig{
		MaxEvents:      10000,
		FlushInterval:  5 * time.Second,
		RetentionDays:  90,
		IncludeHeaders: false,
	}
}

// AuditLogger logs audit events
type AuditLogger struct {
	config  *AuditConfig
	events  []*AuditEvent
	mu      sync.RWMutex
	handlers []func(*AuditEvent)
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(config *AuditConfig) *AuditLogger {
	return &AuditLogger{
		config:  config,
		events:  make([]*AuditEvent, 0, config.MaxEvents),
		handlers: make([]func(*AuditEvent), 0),
	}
}

// Log logs an audit event
func (l *AuditLogger) Log(ctx context.Context, event *AuditEvent) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Set defaults
	if event.ID == "" {
		event.ID = generateAuditID()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Add to events
	l.events = append(l.events, event)

	// Trim if needed
	if len(l.events) > l.config.MaxEvents {
		l.events = l.events[1:]
	}

	// Call handlers
	for _, handler := range l.handlers {
		handler(event)
	}

	return nil
}

// AddHandler adds an event handler
func (l *AuditLogger) AddHandler(handler func(*AuditEvent)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.handlers = append(l.handlers, handler)
}

// LogLogin logs a login event
func (l *AuditLogger) LogLogin(ctx context.Context, userID string, success bool, ipAddress, userAgent string) error {
	result := ResultSuccess
	if !success {
		result = ResultFailure
	}

	return l.Log(ctx, &AuditEvent{
		Type:      EventTypeLogin,
		Result:    result,
		UserID:    userID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Message:   fmt.Sprintf("User %s login attempt", userID),
	})
}

// LogLogout logs a logout event
func (l *AuditLogger) LogLogout(ctx context.Context, userID string) error {
	return l.Log(ctx, &AuditEvent{
		Type:    EventTypeLogout,
		Result:  ResultSuccess,
		UserID:  userID,
		Message: fmt.Sprintf("User %s logged out", userID),
	})
}

// LogAccess logs an access event
func (l *AuditLogger) LogAccess(ctx context.Context, userID, method, path string, statusCode int, duration time.Duration) error {
	result := ResultSuccess
	if statusCode >= 400 {
		result = ResultFailure
	}

	return l.Log(ctx, &AuditEvent{
		Type:       EventTypeAccess,
		Result:     result,
		UserID:     userID,
		Method:     method,
		Path:       path,
		StatusCode: statusCode,
		Duration:   duration,
	})
}

// LogCreate logs a create event
func (l *AuditLogger) LogCreate(ctx context.Context, actorID, resource string, details map[string]interface{}) error {
	return l.Log(ctx, &AuditEvent{
		Type:     EventTypeCreate,
		Result:   ResultSuccess,
		ActorID:  actorID,
		Resource: resource,
		Details:  details,
	})
}

// LogUpdate logs an update event
func (l *AuditLogger) LogUpdate(ctx context.Context, actorID, resource string, details map[string]interface{}) error {
	return l.Log(ctx, &AuditEvent{
		Type:     EventTypeUpdate,
		Result:   ResultSuccess,
		ActorID:  actorID,
		Resource: resource,
		Details:  details,
	})
}

// LogDelete logs a delete event
func (l *AuditLogger) LogDelete(ctx context.Context, actorID, resource string) error {
	return l.Log(ctx, &AuditEvent{
		Type:     EventTypeDelete,
		Result:   ResultSuccess,
		ActorID:  actorID,
		Resource: resource,
	})
}

// LogAuthentication logs an authentication event
func (l *AuditLogger) LogAuthentication(ctx context.Context, userID string, success bool, method string) error {
	result := ResultSuccess
	if !success {
		result = ResultFailure
	}

	return l.Log(ctx, &AuditEvent{
		Type:    EventTypeAuthenticate,
		Result:  result,
		UserID:  userID,
		Method:  method,
		Message: fmt.Sprintf("Authentication attempt via %s", method),
	})
}

// LogAuthorization logs an authorization event
func (l *AuditLogger) LogAuthorization(ctx context.Context, userID, resource, action string, allowed bool) error {
	result := ResultSuccess
	if !allowed {
		result = ResultDenied
	}

	return l.Log(ctx, &AuditEvent{
		Type:     EventTypeAuthorize,
		Result:   result,
		UserID:   userID,
		Resource: resource,
		Action:   action,
		Message:  fmt.Sprintf("Authorization check for %s on %s", action, resource),
	})
}

// LogKeyGenerate logs a key generation event
func (l *AuditLogger) LogKeyGenerate(ctx context.Context, actorID, keyID string) error {
	return l.Log(ctx, &AuditEvent{
		Type:     EventTypeKeyGenerate,
		Result:   ResultSuccess,
		ActorID:  actorID,
		Resource: "api_key:" + keyID,
		Message:  fmt.Sprintf("API key %s generated", keyID),
	})
}

// LogKeyRotate logs a key rotation event
func (l *AuditLogger) LogKeyRotate(ctx context.Context, actorID, keyID string) error {
	return l.Log(ctx, &AuditEvent{
		Type:     EventTypeKeyRotate,
		Result:   ResultSuccess,
		ActorID:  actorID,
		Resource: "api_key:" + keyID,
		Message:  fmt.Sprintf("API key %s rotated", keyID),
	})
}

// LogKeyRevoke logs a key revocation event
func (l *AuditLogger) LogKeyRevoke(ctx context.Context, actorID, keyID string) error {
	return l.Log(ctx, &AuditEvent{
		Type:     EventTypeKeyRevoke,
		Result:   ResultSuccess,
		ActorID:  actorID,
		Resource: "api_key:" + keyID,
		Message:  fmt.Sprintf("API key %s revoked", keyID),
	})
}

// LogRoleAssign logs a role assignment event
func (l *AuditLogger) LogRoleAssign(ctx context.Context, actorID, targetUserID, role string) error {
	return l.Log(ctx, &AuditEvent{
		Type:     EventTypeRoleAssign,
		Result:   ResultSuccess,
		ActorID:  actorID,
		UserID:   targetUserID,
		Resource: "role:" + role,
		Message:  fmt.Sprintf("Role %s assigned to user %s", role, targetUserID),
	})
}

// LogRoleRevoke logs a role revocation event
func (l *AuditLogger) LogRoleRevoke(ctx context.Context, actorID, targetUserID, role string) error {
	return l.Log(ctx, &AuditEvent{
		Type:     EventTypeRoleRevoke,
		Result:   ResultSuccess,
		ActorID:  actorID,
		UserID:   targetUserID,
		Resource: "role:" + role,
		Message:  fmt.Sprintf("Role %s revoked from user %s", role, targetUserID),
	})
}

// GetEvents retrieves events
func (l *AuditLogger) GetEvents(ctx context.Context, filter *AuditFilter) ([]*AuditEvent, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var events []*AuditEvent
	for _, event := range l.events {
		if filter.Match(event) {
			events = append(events, event)
		}
	}

	return events, nil
}

// GetByUser retrieves events for a user
func (l *AuditLogger) GetByUser(ctx context.Context, userID string) ([]*AuditEvent, error) {
	return l.GetEvents(ctx, &AuditFilter{UserID: userID})
}

// GetByType retrieves events by type
func (l *AuditLogger) GetByType(ctx context.Context, eventType AuditEventType) ([]*AuditEvent, error) {
	return l.GetEvents(ctx, &AuditFilter{Type: eventType})
}

// GetByTimeRange retrieves events in a time range
func (l *AuditLogger) GetByTimeRange(ctx context.Context, start, end time.Time) ([]*AuditEvent, error) {
	return l.GetEvents(ctx, &AuditFilter{StartTime: start, EndTime: end})
}

// AuditFilter filters audit events
type AuditFilter struct {
	UserID    string
	ActorID   string
	Type      AuditEventType
	Result    AuditResult
	Resource  string
	StartTime time.Time
	EndTime   time.Time
}

// Match checks if an event matches the filter
func (f *AuditFilter) Match(event *AuditEvent) bool {
	if f.UserID != "" && event.UserID != f.UserID {
		return false
	}
	if f.ActorID != "" && event.ActorID != f.ActorID {
		return false
	}
	if f.Type != "" && event.Type != f.Type {
		return false
	}
	if f.Result != "" && event.Result != f.Result {
		return false
	}
	if f.Resource != "" && event.Resource != f.Resource {
		return false
	}
	if !f.StartTime.IsZero() && event.Timestamp.Before(f.StartTime) {
		return false
	}
	if !f.EndTime.IsZero() && event.Timestamp.After(f.EndTime) {
		return false
	}
	return true
}

// Clear clears all events
func (l *AuditLogger) Clear(ctx context.Context) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.events = make([]*AuditEvent, 0, l.config.MaxEvents)
}

// Export exports events as JSON
func (l *AuditLogger) Export(ctx context.Context) ([]byte, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return json.Marshal(l.events)
}

// generateAuditID generates a unique audit ID
func generateAuditID() string {
	return fmt.Sprintf("audit-%d", time.Now().UnixNano())
}