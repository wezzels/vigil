// Package escalation provides escalation logic for alert dissemination
package escalation

import (
	"fmt"
	"sync"
	"time"
)

// EscalationLevel represents escalation severity
type EscalationLevel int

const (
	LevelNone EscalationLevel = iota
	LevelNotify
	LevelAlert
	LevelCritical
	LevelEmergency
)

// String returns string representation of level
func (l EscalationLevel) String() string {
	switch l {
	case LevelNone:
		return "NONE"
	case LevelNotify:
		return "NOTIFY"
	case LevelAlert:
		return "ALERT"
	case LevelCritical:
		return "CRITICAL"
	case LevelEmergency:
		return "EMERGENCY"
	default:
		return "UNKNOWN"
	}
}

// EscalationRule defines when to escalate
type EscalationRule struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	FromLevel       EscalationLevel `json:"from_level"`
	ToLevel         EscalationLevel `json:"to_level"`
	TriggerAfter    time.Duration   `json:"trigger_after"`
	MaxAttempts     int             `json:"max_attempts"`
	RequireAck      bool            `json:"require_ack"`
	NotifyRecipients []string       `json:"notify_recipients"`
	Conditions      []Condition     `json:"conditions"`
}

// Condition defines a condition for escalation
type Condition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"` // eq, ne, gt, lt, gte, lte, contains
	Value    interface{} `json:"value"`
}

// EscalationState tracks escalation state
type EscalationState struct {
	AlertID         string          `json:"alert_id"`
	CurrentLevel    EscalationLevel `json:"current_level"`
	OriginalLevel   EscalationLevel `json:"original_level"`
	AttemptCount    int             `json:"attempt_count"`
	LastAttempt     time.Time       `json:"last_attempt"`
	LastEscalation  time.Time       `json:"last_escalation"`
	NextEscalation  time.Time       `json:"next_escalation"`
	EscalationPath  []EscalationStep `json:"escalation_path"`
	Acknowledged    bool            `json:"acknowledged"`
	AcknowledgedBy  string          `json:"acknowledged_by,omitempty"`
	StartedAt       time.Time       `json:"started_at"`
	CompletedAt     time.Time       `json:"completed_at,omitempty"`
}

// EscalationStep represents a step in escalation
type EscalationStep struct {
	Level       EscalationLevel `json:"level"`
	Timestamp   time.Time       `json:"timestamp"`
	Reason      string          `json:"reason"`
	Recipients  []string        `json:"recipients"`
}

// EscalationManager manages escalation rules and state
type EscalationManager struct {
	rules    map[string]*EscalationRule
	states   map[string]*EscalationState
	callbacks map[string][]func(*EscalationState)
	mutex    sync.RWMutex
}

// NewEscalationManager creates a new escalation manager
func NewEscalationManager() *EscalationManager {
	return &EscalationManager{
		rules:    make(map[string]*EscalationRule),
		states:   make(map[string]*EscalationState),
		callbacks: make(map[string][]func(*EscalationState)),
	}
}

// AddRule adds an escalation rule
func (m *EscalationManager) AddRule(rule *EscalationRule) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if rule.ID == "" {
		return fmt.Errorf("rule ID is required")
	}

	m.rules[rule.ID] = rule
	return nil
}

// RemoveRule removes an escalation rule
func (m *EscalationManager) RemoveRule(ruleID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.rules, ruleID)
}

// StartEscalation starts escalation for an alert
func (m *EscalationManager) StartEscalation(alertID string, initialLevel EscalationLevel) *EscalationState {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	state := &EscalationState{
		AlertID:        alertID,
		CurrentLevel:   initialLevel,
		OriginalLevel:  initialLevel,
		AttemptCount:   0,
		LastAttempt:    now,
		StartedAt:      now,
		EscalationPath: []EscalationStep{},
	}

	m.states[alertID] = state
	return state
}

// CheckEscalation checks if escalation is needed
func (m *EscalationManager) CheckEscalation(alertID string) (*EscalationState, bool, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	state, exists := m.states[alertID]
	if !exists {
		return nil, false, ErrStateNotFound
	}

	// Don't escalate if acknowledged
	if state.Acknowledged {
		return state, false, nil
	}

	// Find applicable rules
	for _, rule := range m.rules {
		if !m.isRuleApplicable(rule, state) {
			continue
		}

		// Check if escalation time has passed
		now := time.Now()
		if now.After(state.NextEscalation) && now.Sub(state.LastEscalation) >= rule.TriggerAfter {
			// Apply escalation
			state.CurrentLevel = rule.ToLevel
			state.LastEscalation = now
			state.AttemptCount++
			state.EscalationPath = append(state.EscalationPath, EscalationStep{
				Level:      rule.ToLevel,
				Timestamp:  now,
				Reason:     fmt.Sprintf("Escalated after %v", rule.TriggerAfter),
				Recipients: rule.NotifyRecipients,
			})

			// Trigger callbacks
			m.triggerCallbacks(alertID, state)

			return state, true, nil
		}
	}

	return state, false, nil
}

// isRuleApplicable checks if a rule applies to a state
func (m *EscalationManager) isRuleApplicable(rule *EscalationRule, state *EscalationState) bool {
	// Check level match
	if state.CurrentLevel != rule.FromLevel {
		return false
	}

	// Check max attempts
	if rule.MaxAttempts > 0 && state.AttemptCount >= rule.MaxAttempts {
		return false
	}

	// Check require ack
	if rule.RequireAck && state.Acknowledged {
		return false
	}

	return true
}

// Acknowledge marks an alert as acknowledged
func (m *EscalationManager) Acknowledge(alertID, acknowledgedBy string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	state, exists := m.states[alertID]
	if !exists {
		return ErrStateNotFound
	}

	state.Acknowledged = true
	state.AcknowledgedBy = acknowledgedBy
	state.CompletedAt = time.Now()

	return nil
}

// Deescalate de-escalates an alert
func (m *EscalationManager) Deescalate(alertID string, reason string) (*EscalationState, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	state, exists := m.states[alertID]
	if !exists {
		return nil, ErrStateNotFound
	}

	// Can only de-escalate one level at a time
	if state.CurrentLevel > LevelNone {
		state.CurrentLevel--
		now := time.Now()
		state.EscalationPath = append(state.EscalationPath, EscalationStep{
			Level:     state.CurrentLevel,
			Timestamp: now,
			Reason:    fmt.Sprintf("De-escalated: %s", reason),
		})
	}

	return state, nil
}

// GetState returns the escalation state for an alert
func (m *EscalationManager) GetState(alertID string) (*EscalationState, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	state, exists := m.states[alertID]
	if !exists {
		return nil, ErrStateNotFound
	}

	return state, nil
}

// GetActiveEscalations returns all active escalations
func (m *EscalationManager) GetActiveEscalations() []*EscalationState {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var active []*EscalationState
	for _, state := range m.states {
		if !state.Acknowledged {
			active = append(active, state)
		}
	}
	return active
}

// GetEscalationsByLevel returns escalations at a specific level
func (m *EscalationManager) GetEscalationsByLevel(level EscalationLevel) []*EscalationState {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var result []*EscalationState
	for _, state := range m.states {
		if state.CurrentLevel == level && !state.Acknowledged {
			result = append(result, state)
		}
	}
	return result
}

// OnEscalate registers a callback for escalation events
func (m *EscalationManager) OnEscalate(alertID string, callback func(*EscalationState)) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.callbacks[alertID] = append(m.callbacks[alertID], callback)
}

// triggerCallbacks triggers escalation callbacks
func (m *EscalationManager) triggerCallbacks(alertID string, state *EscalationState) {
	callbacks := m.callbacks[alertID]
	for _, cb := range callbacks {
		cb(state)
	}
}

// CancelEscalation cancels escalation for an alert
func (m *EscalationManager) CancelEscalation(alertID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	state, exists := m.states[alertID]
	if !exists {
		return ErrStateNotFound
	}

	state.Acknowledged = true
	state.CompletedAt = time.Now()

	return nil
}

// ClearState removes state for an alert
func (m *EscalationManager) ClearState(alertID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.states, alertID)
}

// Stats returns escalation statistics
func (m *EscalationManager) Stats() EscalationStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	stats := EscalationStats{}
	for _, state := range m.states {
		if state.Acknowledged {
			stats.Completed++
		} else {
			stats.Active++
			switch state.CurrentLevel {
			case LevelNotify:
				stats.AtNotify++
			case LevelAlert:
				stats.AtAlert++
			case LevelCritical:
				stats.AtCritical++
			case LevelEmergency:
				stats.AtEmergency++
			}
		}
	}

	return stats
}

// EscalationStats holds escalation statistics
type EscalationStats struct {
	Active      int `json:"active"`
	Completed   int `json:"completed"`
	AtNotify    int `json:"at_notify"`
	AtAlert     int `json:"at_alert"`
	AtCritical  int `json:"at_critical"`
	AtEmergency int `json:"at_emergency"`
}

// DefaultEscalationRules returns default escalation rules
func DefaultEscalationRules() []*EscalationRule {
	return []*EscalationRule{
		{
			ID:           "notify-to-alert",
			Name:         "Notify to Alert",
			Description:  "Escalate from NOTIFY to ALERT after 5 minutes",
			FromLevel:    LevelNotify,
			ToLevel:      LevelAlert,
			TriggerAfter: 5 * time.Minute,
			MaxAttempts:  1,
		},
		{
			ID:           "alert-to-critical",
			Name:         "Alert to Critical",
			Description:  "Escalate from ALERT to CRITICAL after 10 minutes",
			FromLevel:    LevelAlert,
			ToLevel:      LevelCritical,
			TriggerAfter: 10 * time.Minute,
			MaxAttempts:  1,
		},
		{
			ID:           "critical-to-emergency",
			Name:         "Critical to Emergency",
			Description:  "Escalate from CRITICAL to EMERGENCY after 15 minutes",
			FromLevel:    LevelCritical,
			ToLevel:      LevelEmergency,
			TriggerAfter: 15 * time.Minute,
			MaxAttempts:  1,
		},
	}
}

// Errors
var (
	ErrStateNotFound = &EscalationError{Code: "STATE_NOT_FOUND", Message: "escalation state not found"}
)

// EscalationError represents an escalation error
type EscalationError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *EscalationError) Error() string {
	return e.Message
}