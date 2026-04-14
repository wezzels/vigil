// Package cache provides session management
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Session represents a user session
type Session struct {
	ID           string            `json:"id"`
	UserID       string            `json:"user_id"`
	Username     string            `json:"username"`
	Roles        []string          `json:"roles"`
	Permissions  []string          `json:"permissions"`
	Metadata     map[string]string `json:"metadata"`
	CreatedAt    time.Time         `json:"created_at"`
	LastAccessed time.Time         `json:"last_accessed"`
	ExpiresAt    time.Time         `json:"expires_at"`
}

// SessionCache provides session management operations
type SessionCache struct {
	cache *Cache
	ttl   time.Duration
}

// NewSessionCache creates a new session cache
func NewSessionCache(cache *Cache, ttl time.Duration) *SessionCache {
	return &SessionCache{
		cache: cache,
		ttl:   ttl,
	}
}

// Create creates a new session
func (sc *SessionCache) Create(ctx context.Context, userID, username string, roles []string) (*Session, error) {
	session := &Session{
		ID:           generateSessionID(),
		UserID:       userID,
		Username:     username,
		Roles:        roles,
		Permissions:  []string{},
		Metadata:     make(map[string]string),
		CreatedAt:    time.Now(),
		LastAccessed: time.Now(),
		ExpiresAt:     time.Now().Add(sc.ttl),
	}

	if err := sc.Set(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}

// Get retrieves a session
func (sc *SessionCache) Get(ctx context.Context, sessionID string) (*Session, error) {
	key := sc.sessionKey(sessionID)
	data, err := sc.cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	// Check expiration
	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	return &session, nil
}

// Set stores a session
func (sc *SessionCache) Set(ctx context.Context, session *Session) error {
	key := sc.sessionKey(session.ID)
	session.ExpiresAt = time.Now().Add(sc.ttl)
	session.LastAccessed = time.Now()

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Store session
	if err := sc.cache.SetString(ctx, key, string(data), sc.ttl); err != nil {
		return err
	}

	// Add to user's sessions
	userKey := sc.userSessionsKey(session.UserID)
	return sc.cache.SAdd(ctx, userKey, session.ID)
}

// Delete removes a session
func (sc *SessionCache) Delete(ctx context.Context, sessionID string) error {
	// Get session to find user
	session, err := sc.Get(ctx, sessionID)
	if err != nil {
		return nil // Already deleted
	}

	// Remove from user's sessions
	userKey := sc.userSessionsKey(session.UserID)
	sc.cache.SRem(ctx, userKey, sessionID)

	// Delete session
	return sc.cache.Delete(ctx, sc.sessionKey(sessionID))
}

// Refresh refreshes a session's TTL
func (sc *SessionCache) Refresh(ctx context.Context, sessionID string) error {
	session, err := sc.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	return sc.Set(ctx, session)
}

// Validate validates a session
func (sc *SessionCache) Validate(ctx context.Context, sessionID string) (*Session, error) {
	session, err := sc.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Refresh TTL on access
	if err := sc.Refresh(ctx, sessionID); err != nil {
		return nil, err
	}

	return session, nil
}

// AddPermission adds a permission to a session
func (sc *SessionCache) AddPermission(ctx context.Context, sessionID, permission string) error {
	session, err := sc.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	for _, p := range session.Permissions {
		if p == permission {
			return nil // Already exists
		}
	}

	session.Permissions = append(session.Permissions, permission)
	return sc.Set(ctx, session)
}

// RemovePermission removes a permission from a session
func (sc *SessionCache) RemovePermission(ctx context.Context, sessionID, permission string) error {
	session, err := sc.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	newPerms := make([]string, 0)
	for _, p := range session.Permissions {
		if p != permission {
			newPerms = append(newPerms, p)
		}
	}

	session.Permissions = newPerms
	return sc.Set(ctx, session)
}

// HasPermission checks if a session has a permission
func (sc *SessionCache) HasPermission(ctx context.Context, sessionID, permission string) (bool, error) {
	session, err := sc.Get(ctx, sessionID)
	if err != nil {
		return false, err
	}

	for _, p := range session.Permissions {
		if p == permission {
			return true, nil
		}
	}

	return false, nil
}

// AddRole adds a role to a session
func (sc *SessionCache) AddRole(ctx context.Context, sessionID, role string) error {
	session, err := sc.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	for _, r := range session.Roles {
		if r == role {
			return nil // Already exists
		}
	}

	session.Roles = append(session.Roles, role)
	return sc.Set(ctx, session)
}

// RemoveRole removes a role from a session
func (sc *SessionCache) RemoveRole(ctx context.Context, sessionID, role string) error {
	session, err := sc.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	newRoles := make([]string, 0)
	for _, r := range session.Roles {
		if r != role {
			newRoles = append(newRoles, r)
		}
	}

	session.Roles = newRoles
	return sc.Set(ctx, session)
}

// HasRole checks if a session has a role
func (sc *SessionCache) HasRole(ctx context.Context, sessionID, role string) (bool, error) {
	session, err := sc.Get(ctx, sessionID)
	if err != nil {
		return false, err
	}

	for _, r := range session.Roles {
		if r == role {
			return true, nil
		}
	}

	return false, nil
}

// GetUserSessions gets all sessions for a user
func (sc *SessionCache) GetUserSessions(ctx context.Context, userID string) ([]*Session, error) {
	userKey := sc.userSessionsKey(userID)
	sessionIDs, err := sc.cache.SMembers(ctx, userKey)
	if err != nil {
		return nil, err
	}

	var sessions []*Session
	for _, sessionID := range sessionIDs {
		session, err := sc.Get(ctx, sessionID)
		if err != nil {
			sc.cache.SRem(ctx, userKey, sessionID)
			continue
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// DeleteUserSessions deletes all sessions for a user
func (sc *SessionCache) DeleteUserSessions(ctx context.Context, userID string) error {
	userKey := sc.userSessionsKey(userID)
	sessionIDs, err := sc.cache.SMembers(ctx, userKey)
	if err != nil {
		return err
	}

	for _, sessionID := range sessionIDs {
		sc.cache.Delete(ctx, sc.sessionKey(sessionID))
	}

	return sc.cache.Delete(ctx, userKey)
}

// SetMetadata sets session metadata
func (sc *SessionCache) SetMetadata(ctx context.Context, sessionID, key, value string) error {
	session, err := sc.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	if session.Metadata == nil {
		session.Metadata = make(map[string]string)
	}
	session.Metadata[key] = value

	return sc.Set(ctx, session)
}

// GetMetadata gets session metadata
func (sc *SessionCache) GetMetadata(ctx context.Context, sessionID, key string) (string, error) {
	session, err := sc.Get(ctx, sessionID)
	if err != nil {
		return "", err
	}

	if session.Metadata == nil {
		return "", fmt.Errorf("metadata not found")
	}

	value, ok := session.Metadata[key]
	if !ok {
		return "", fmt.Errorf("metadata key not found")
	}

	return value, nil
}

// sessionKey returns the cache key for a session
func (sc *SessionCache) sessionKey(sessionID string) string {
	return fmt.Sprintf("session:%s", sessionID)
}

// userSessionsKey returns the cache key for a user's sessions
func (sc *SessionCache) userSessionsKey(userID string) string {
	return fmt.Sprintf("user:%s:sessions", userID)
}

// generateSessionID generates a unique session ID
func generateSessionID() string {
	return fmt.Sprintf("sess_%d_%s", time.Now().UnixNano(), randomString(16))
}

// randomString generates a random string
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}