package auth

import (
	"context"
	"testing"
	"time"
)

// TestmTLSConfig tests mTLS configuration
func TestmTLSConfig(t *testing.T) {
	config := DefaultmTLSConfig()

	if config.MinVersion != 0x0303 { // TLS 1.2
		t.Errorf("Expected TLS 1.2 minimum, got %x", config.MinVersion)
	}
}

// TestCertificateInfo tests certificate info
func TestCertificateInfo(t *testing.T) {
	info := &CertificateInfo{
		Subject:   "CN=test",
		Issuer:    "CN=CA",
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(24 * time.Hour),
	}

	if info.IsExpired() {
		t.Error("Certificate should not be expired")
	}

	if !info.ExpiresSoon(48 * time.Hour) {
		t.Error("Certificate should expire within 48 hours")
	}
}

// TestPKIConfig tests PKI configuration
func TestPKIConfig(t *testing.T) {
	config := DefaultPKIConfig()

	if config.KeySize != 4096 {
		t.Errorf("Expected key size 4096, got %d", config.KeySize)
	}

	if config.ValidityDays != 365 {
		t.Errorf("Expected validity 365 days, got %d", config.ValidityDays)
	}
}

// TestCertificateRequest tests certificate request
func TestCertificateRequest(t *testing.T) {
	req := &CertificateRequest{
		CommonName:   "test.vigil.local",
		Organization: "VIGIL",
		DNSNames:     []string{"test.vigil.local"},
		KeyType:      "RSA",
		KeySize:      2048,
		Days:         365,
		IsClient:     false,
	}

	if req.CommonName != "test.vigil.local" {
		t.Errorf("Expected CN test.vigil.local, got %s", req.CommonName)
	}
}

// TestJWTConfig tests JWT configuration
func TestJWTConfig(t *testing.T) {
	config := DefaultJWTConfig()

	if config.Expiration != 24*time.Hour {
		t.Errorf("Expected expiration 24h, got %v", config.Expiration)
	}
}

// TestClaims tests JWT claims
func TestClaims(t *testing.T) {
	claims := &Claims{
		Subject:     "user-001",
		Issuer:      "vigil",
		Audience:    []string{"vigil-api"},
		ExpiresAt:   time.Now().Add(24 * time.Hour).Unix(),
		IssuedAt:    time.Now().Unix(),
		ID:          "token-001",
		Roles:       []string{"admin", "operator"},
		Permissions: []string{"read", "write"},
	}

	if claims.Subject != "user-001" {
		t.Errorf("Expected subject user-001, got %s", claims.Subject)
	}

	if !claims.HasRole("admin") {
		t.Error("Claims should have admin role")
	}

	if !claims.HasPermission("read") {
		t.Error("Claims should have read permission")
	}

	if claims.IsExpired() {
		t.Error("Claims should not be expired")
	}

	expiresIn := claims.ExpiresIn()
	if expiresIn <= 0 || expiresIn > 24*time.Hour {
		t.Errorf("Expected ExpiresIn ~24h, got %v", expiresIn)
	}
}

// TestAPIKeyConfig tests API key configuration
func TestAPIKeyConfig(t *testing.T) {
	config := DefaultAPIKeyConfig()

	if config.KeyLength != 32 {
		t.Errorf("Expected key length 32, got %d", config.KeyLength)
	}

	if config.MaxKeysPerUser != 10 {
		t.Errorf("Expected max keys 10, got %d", config.MaxKeysPerUser)
	}
}

// TestAPIKey tests API key
func TestAPIKey(t *testing.T) {
	key := &APIKey{
		ID:          "key-001",
		Key:         "abc123",
		Name:        "Test Key",
		UserID:      "user-001",
		Roles:       []string{"operator"},
		Permissions: []string{"read", "write"},
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(365 * 24 * time.Hour),
		Enabled:     true,
	}

	if key.ID != "key-001" {
		t.Errorf("Expected ID key-001, got %s", key.ID)
	}

	if !key.HasRole("operator") {
		t.Error("Key should have operator role")
	}

	if !key.HasPermission("read") {
		t.Error("Key should have read permission")
	}

	if key.IsExpired() {
		t.Error("Key should not be expired")
	}
}

// TestAPIKeyManager tests API key manager
func TestAPIKeyManager(t *testing.T) {
	config := DefaultAPIKeyConfig()
	mgr := NewAPIKeyManager(config)
	ctx := context.Background()

	// Generate key
	key, err := mgr.Generate(ctx, "user-001", "Test Key", "Test description", []string{"operator"}, []string{"read"})
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	if key.UserID != "user-001" {
		t.Errorf("Expected user ID user-001, got %s", key.UserID)
	}

	// Validate key
	validated, err := mgr.Validate(ctx, key.Key)
	if err != nil {
		t.Fatalf("Failed to validate key: %v", err)
	}

	if validated.ID != key.ID {
		t.Errorf("Expected ID %s, got %s", key.ID, validated.ID)
	}

	// Disable key
	if err := mgr.Disable(ctx, key.ID); err != nil {
		t.Fatalf("Failed to disable key: %v", err)
	}

	// Validate should fail
	_, err = mgr.Validate(ctx, key.Key)
	if err == nil {
		t.Error("Expected validation error for disabled key")
	}

	// Enable key
	if err := mgr.Enable(ctx, key.ID); err != nil {
		t.Fatalf("Failed to enable key: %v", err)
	}

	// Rotate key
	rotated, err := mgr.Rotate(ctx, key.ID)
	if err != nil {
		t.Fatalf("Failed to rotate key: %v", err)
	}

	if rotated.ID != key.ID {
		t.Errorf("Rotated key should have same ID")
	}

	// Delete key
	if err := mgr.Delete(ctx, key.ID); err != nil {
		t.Fatalf("Failed to delete key: %v", err)
	}

	// Get should fail
	_, err = mgr.Get(ctx, key.ID)
	if err == nil {
		t.Error("Expected error getting deleted key")
	}
}

// TestRBACConfig tests RBAC configuration
func TestRBACConfig(t *testing.T) {
	config := DefaultRBACConfig()

	if config.DefaultRole != "viewer" {
		t.Errorf("Expected default role viewer, got %s", config.DefaultRole)
	}

	if config.SuperAdmin != "admin" {
		t.Errorf("Expected super admin admin, got %s", config.SuperAdmin)
	}
}

// TestRBACManager tests RBAC manager
func TestRBACManager(t *testing.T) {
	config := DefaultRBACConfig()
	mgr := NewRBACManager(config)

	// Check default roles exist
	roles := mgr.ListRoles()
	if len(roles) < 4 {
		t.Errorf("Expected at least 4 default roles, got %d", len(roles))
	}

	// Get role
	role, err := mgr.GetRole("admin")
	if err != nil {
		t.Fatalf("Failed to get admin role: %v", err)
	}

	if role.Name != "admin" {
		t.Errorf("Expected role name admin, got %s", role.Name)
	}

	// Assign role
	if err := mgr.AssignRole("user-001", "operator"); err != nil {
		t.Fatalf("Failed to assign role: %v", err)
	}

	// Check user roles
	userRoles := mgr.GetUserRoles("user-001")
	if len(userRoles) != 1 || userRoles[0] != "operator" {
		t.Errorf("Expected 1 role (operator), got %v", userRoles)
	}

	// Check permission
	has, err := mgr.CheckPermission("user-001", "alerts", "acknowledge")
	if err != nil {
		t.Fatalf("Failed to check permission: %v", err)
	}

	if !has {
		t.Error("User should have alerts:acknowledge permission")
	}

	// Revoke role
	if err := mgr.RevokeRole("user-001", "operator"); err != nil {
		t.Fatalf("Failed to revoke role: %v", err)
	}

	userRoles = mgr.GetUserRoles("user-001")
	if len(userRoles) != 0 {
		t.Errorf("Expected 0 roles, got %v", userRoles)
	}
}

// TestPermission tests permission
func TestPermission(t *testing.T) {
	perm := Permission{
		Name:        "tracks:read",
		Description: "Read track data",
		Resource:    "tracks",
		Action:      "read",
	}

	if perm.Name != "tracks:read" {
		t.Errorf("Expected name tracks:read, got %s", perm.Name)
	}

	if perm.Resource != "tracks" {
		t.Errorf("Expected resource tracks, got %s", perm.Resource)
	}

	if perm.Action != "read" {
		t.Errorf("Expected action read, got %s", perm.Action)
	}
}

// TestAuditConfig tests audit configuration
func TestAuditConfig(t *testing.T) {
	config := DefaultAuditConfig()

	if config.MaxEvents != 10000 {
		t.Errorf("Expected max events 10000, got %d", config.MaxEvents)
	}

	if config.RetentionDays != 90 {
		t.Errorf("Expected retention 90 days, got %d", config.RetentionDays)
	}
}

// TestAuditEvent tests audit event
func TestAuditEvent(t *testing.T) {
	event := &AuditEvent{
		ID:        "audit-001",
		Timestamp: time.Now(),
		Type:      EventTypeLogin,
		Result:    ResultSuccess,
		UserID:    "user-001",
		IPAddress: "192.168.1.1",
	}

	if event.Type != EventTypeLogin {
		t.Errorf("Expected type login, got %s", event.Type)
	}

	if event.Result != ResultSuccess {
		t.Errorf("Expected result success, got %s", event.Result)
	}
}

// TestAuditLogger tests audit logger
func TestAuditLogger(t *testing.T) {
	config := DefaultAuditConfig()
	logger := NewAuditLogger(config)
	ctx := context.Background()

	// Log login
	if err := logger.LogLogin(ctx, "user-001", true, "192.168.1.1", "Mozilla/5.0"); err != nil {
		t.Fatalf("Failed to log login: %v", err)
	}

	// Log access
	if err := logger.LogAccess(ctx, "user-001", "GET", "/api/tracks", 200, 100*time.Millisecond); err != nil {
		t.Fatalf("Failed to log access: %v", err)
	}

	// Log authentication
	if err := logger.LogAuthentication(ctx, "user-001", true, "certificate"); err != nil {
		t.Fatalf("Failed to log authentication: %v", err)
	}

	// Log authorization
	if err := logger.LogAuthorization(ctx, "user-001", "tracks", "read", true); err != nil {
		t.Fatalf("Failed to log authorization: %v", err)
	}

	// Get events
	events, err := logger.GetByUser(ctx, "user-001")
	if err != nil {
		t.Fatalf("Failed to get events: %v", err)
	}

	if len(events) != 4 {
		t.Errorf("Expected 4 events, got %d", len(events))
	}

	// Get by type
	loginEvents, err := logger.GetByType(ctx, EventTypeLogin)
	if err != nil {
		t.Fatalf("Failed to get login events: %v", err)
	}

	if len(loginEvents) != 1 {
		t.Errorf("Expected 1 login event, got %d", len(loginEvents))
	}

	// Clear
	logger.Clear(ctx)

	events, err = logger.GetByUser(ctx, "user-001")
	if err != nil {
		t.Fatalf("Failed to get events: %v", err)
	}

	if len(events) != 0 {
		t.Errorf("Expected 0 events after clear, got %d", len(events))
	}
}

// TestAuditFilter tests audit filter
func TestAuditFilter(t *testing.T) {
	event := &AuditEvent{
		ID:        "audit-001",
		Timestamp: time.Now(),
		Type:      EventTypeLogin,
		Result:    ResultSuccess,
		UserID:    "user-001",
	}

	// Match by user
	filter := &AuditFilter{UserID: "user-001"}
	if !filter.Match(event) {
		t.Error("Filter should match user")
	}

	// Match by type
	filter = &AuditFilter{Type: EventTypeLogin}
	if !filter.Match(event) {
		t.Error("Filter should match type")
	}

	// Match by result
	filter = &AuditFilter{Result: ResultSuccess}
	if !filter.Match(event) {
		t.Error("Filter should match result")
	}

	// No match
	filter = &AuditFilter{UserID: "user-002"}
	if filter.Match(event) {
		t.Error("Filter should not match different user")
	}
}

// TestExtractToken tests token extraction
func TestExtractToken(t *testing.T) {
	// Valid header
	token, err := ExtractToken("Bearer abc123")
	if err != nil {
		t.Fatalf("Failed to extract token: %v", err)
	}

	if token != "abc123" {
		t.Errorf("Expected token abc123, got %s", token)
	}

	// Invalid header
	_, err = ExtractToken("Basic abc123")
	if err == nil {
		t.Error("Expected error for non-Bearer auth")
	}

	// Empty header
	_, err = ExtractToken("")
	if err == nil {
		t.Error("Expected error for empty header")
	}
}
