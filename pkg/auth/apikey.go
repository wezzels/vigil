// Package auth provides API key management
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

// APIKey represents an API key
type APIKey struct {
	ID          string
	Key         string
	Name        string
	Description string
	UserID      string
	Roles       []string
	Permissions []string
	CreatedAt   time.Time
	ExpiresAt   time.Time
	LastUsed    time.Time
	Enabled     bool
	Metadata    map[string]string
}

// APIKeyConfig holds API key configuration
type APIKeyConfig struct {
	KeyLength      int
	DefaultTTL     time.Duration
	MaxKeysPerUser int
}

// DefaultAPIKeyConfig returns default API key configuration
func DefaultAPIKeyConfig() *APIKeyConfig {
	return &APIKeyConfig{
		KeyLength:      32,
		DefaultTTL:     365 * 24 * time.Hour,
		MaxKeysPerUser: 10,
	}
}

// APIKeyManager manages API keys
type APIKeyManager struct {
	config   *APIKeyConfig
	keys     map[string]*APIKey
	userKeys map[string][]string // userID -> keyIDs
	mu       sync.RWMutex
}

// NewAPIKeyManager creates a new API key manager
func NewAPIKeyManager(config *APIKeyConfig) *APIKeyManager {
	return &APIKeyManager{
		config:   config,
		keys:     make(map[string]*APIKey),
		userKeys: make(map[string][]string),
	}
}

// Generate generates a new API key
func (m *APIKeyManager) Generate(ctx context.Context, userID, name, description string, roles, permissions []string) (*APIKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check max keys per user
	if m.config.MaxKeysPerUser > 0 {
		if len(m.userKeys[userID]) >= m.config.MaxKeysPerUser {
			return nil, fmt.Errorf("maximum number of API keys reached for user")
		}
	}

	// Generate key
	keyBytes := make([]byte, m.config.KeyLength)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	now := time.Now()
	apiKey := &APIKey{
		ID:          generateKeyID(),
		Key:         base64.URLEncoding.EncodeToString(keyBytes),
		Name:        name,
		Description: description,
		UserID:      userID,
		Roles:       roles,
		Permissions: permissions,
		CreatedAt:   now,
		ExpiresAt:   now.Add(m.config.DefaultTTL),
		LastUsed:    now,
		Enabled:     true,
		Metadata:    make(map[string]string),
	}

	// Store key
	m.keys[apiKey.ID] = apiKey
	m.userKeys[userID] = append(m.userKeys[userID], apiKey.ID)

	return apiKey, nil
}

// Validate validates an API key
func (m *APIKeyManager) Validate(ctx context.Context, keyString string) (*APIKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Find key by key string
	var found *APIKey
	for _, key := range m.keys {
		if key.Key == keyString {
			found = key
			break
		}
	}

	if found == nil {
		return nil, fmt.Errorf("API key not found")
	}

	// Check if enabled
	if !found.Enabled {
		return nil, fmt.Errorf("API key is disabled")
	}

	// Check expiration
	if !found.ExpiresAt.IsZero() && time.Now().After(found.ExpiresAt) {
		return nil, fmt.Errorf("API key expired")
	}

	// Update last used
	m.mu.RUnlock()
	m.mu.Lock()
	found.LastUsed = time.Now()
	m.mu.Unlock()
	m.mu.RLock()

	return found, nil
}

// Get retrieves an API key by ID
func (m *APIKeyManager) Get(ctx context.Context, keyID string) (*APIKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key, ok := m.keys[keyID]
	if !ok {
		return nil, fmt.Errorf("API key not found")
	}

	return key, nil
}

// GetByUser retrieves all API keys for a user
func (m *APIKeyManager) GetByUser(ctx context.Context, userID string) ([]*APIKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keyIDs := m.userKeys[userID]
	keys := make([]*APIKey, 0, len(keyIDs))

	for _, id := range keyIDs {
		if key, ok := m.keys[id]; ok {
			keys = append(keys, key)
		}
	}

	return keys, nil
}

// Update updates an API key
func (m *APIKeyManager) Update(ctx context.Context, keyID string, updates func(*APIKey)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key, ok := m.keys[keyID]
	if !ok {
		return fmt.Errorf("API key not found")
	}

	updates(key)
	return nil
}

// Enable enables an API key
func (m *APIKeyManager) Enable(ctx context.Context, keyID string) error {
	return m.Update(ctx, keyID, func(k *APIKey) {
		k.Enabled = true
	})
}

// Disable disables an API key
func (m *APIKeyManager) Disable(ctx context.Context, keyID string) error {
	return m.Update(ctx, keyID, func(k *APIKey) {
		k.Enabled = false
	})
}

// Delete deletes an API key
func (m *APIKeyManager) Delete(ctx context.Context, keyID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key, ok := m.keys[keyID]
	if !ok {
		return fmt.Errorf("API key not found")
	}

	// Remove from user's keys
	userKeys := m.userKeys[key.UserID]
	newKeys := make([]string, 0, len(userKeys)-1)
	for _, id := range userKeys {
		if id != keyID {
			newKeys = append(newKeys, id)
		}
	}
	m.userKeys[key.UserID] = newKeys

	// Delete key
	delete(m.keys, keyID)

	return nil
}

// Rotate rotates an API key
func (m *APIKeyManager) Rotate(ctx context.Context, keyID string) (*APIKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key, ok := m.keys[keyID]
	if !ok {
		return nil, fmt.Errorf("API key not found")
	}

	// Generate new key
	keyBytes := make([]byte, m.config.KeyLength)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	key.Key = base64.URLEncoding.EncodeToString(keyBytes)
	key.LastUsed = time.Now()

	return key, nil
}

// SetExpiration sets the expiration time for a key
func (m *APIKeyManager) SetExpiration(ctx context.Context, keyID string, expiresAt time.Time) error {
	return m.Update(ctx, keyID, func(k *APIKey) {
		k.ExpiresAt = expiresAt
	})
}

// SetRoles sets the roles for a key
func (m *APIKeyManager) SetRoles(ctx context.Context, keyID string, roles []string) error {
	return m.Update(ctx, keyID, func(k *APIKey) {
		k.Roles = roles
	})
}

// SetPermissions sets the permissions for a key
func (m *APIKeyManager) SetPermissions(ctx context.Context, keyID string, permissions []string) error {
	return m.Update(ctx, keyID, func(k *APIKey) {
		k.Permissions = permissions
	})
}

// List lists all API keys
func (m *APIKeyManager) List(ctx context.Context) ([]*APIKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]*APIKey, 0, len(m.keys))
	for _, key := range m.keys {
		keys = append(keys, key)
	}

	return keys, nil
}

// ListEnabled lists all enabled API keys
func (m *APIKeyManager) ListEnabled(ctx context.Context) ([]*APIKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]*APIKey, 0)
	for _, key := range m.keys {
		if key.Enabled {
			keys = append(keys, key)
		}
	}

	return keys, nil
}

// ListExpired lists all expired API keys
func (m *APIKeyManager) ListExpired(ctx context.Context) ([]*APIKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()
	keys := make([]*APIKey, 0)
	for _, key := range m.keys {
		if !key.ExpiresAt.IsZero() && now.After(key.ExpiresAt) {
			keys = append(keys, key)
		}
	}

	return keys, nil
}

// Cleanup removes expired keys
func (m *APIKeyManager) Cleanup(ctx context.Context) (int, error) {
	expired, err := m.ListExpired(ctx)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, key := range expired {
		if err := m.Delete(ctx, key.ID); err != nil {
			continue
		}
		count++
	}

	return count, nil
}

// generateKeyID generates a unique key ID
func generateKeyID() string {
	return fmt.Sprintf("key-%d", time.Now().UnixNano())
}

// HasRole checks if the API key has a role
func (k *APIKey) HasRole(role string) bool {
	for _, r := range k.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasPermission checks if the API key has a permission
func (k *APIKey) HasPermission(permission string) bool {
	for _, p := range k.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// IsExpired checks if the API key is expired
func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(k.ExpiresAt)
}

// ExpiresIn returns time until expiration
func (k *APIKey) ExpiresIn() time.Duration {
	if k.ExpiresAt.IsZero() {
		return 0
	}
	return time.Until(k.ExpiresAt)
}
