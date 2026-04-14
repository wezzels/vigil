// Package auth provides JWT validation
package auth

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Issuer     string
	Audience   string
	Expiration time.Duration
}

// DefaultJWTConfig returns default JWT configuration
func DefaultJWTConfig() *JWTConfig {
	return &JWTConfig{
		Expiration: 24 * time.Hour,
	}
}

// Claims represents JWT claims
type Claims struct {
	Subject   string            `json:"sub"`
	Issuer    string            `json:"iss"`
	Audience  []string          `json:"aud"`
	ExpiresAt int64             `json:"exp"`
	NotBefore int64             `json:"nbf"`
	IssuedAt  int64             `json:"iat"`
	ID        string            `json:"jti"`
	Roles     []string          `json:"roles,omitempty"`
	Permissions []string        `json:"permissions,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// JWTValidator validates JWT tokens
type JWTValidator struct {
	config    *JWTConfig
	keys      map[string]interface{}
	keyMutex  sync.RWMutex
	blacklist map[string]bool
	blMutex   sync.RWMutex
}

// NewJWTValidator creates a new JWT validator
func NewJWTValidator(config *JWTConfig) *JWTValidator {
	return &JWTValidator{
		config:    config,
		keys:      make(map[string]interface{}),
		blacklist: make(map[string]bool),
	}
}

// AddKey adds a verification key
func (v *JWTValidator) AddKey(kid string, key interface{}) {
	v.keyMutex.Lock()
	defer v.keyMutex.Unlock()
	v.keys[kid] = key
}

// RemoveKey removes a verification key
func (v *JWTValidator) RemoveKey(kid string) {
	v.keyMutex.Lock()
	defer v.keyMutex.Unlock()
	delete(v.keys, kid)
}

// GetKey retrieves a verification key
func (v *JWTValidator) GetKey(kid string) (interface{}, bool) {
	v.keyMutex.RLock()
	defer v.keyMutex.RUnlock()
	key, ok := v.keys[kid]
	return key, ok
}

// Validate validates a JWT token
func (v *JWTValidator) Validate(ctx context.Context, tokenString string) (*Claims, error) {
	// Parse token (simplified - in production use a proper JWT library)
	claims, err := v.parseToken(tokenString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Check blacklist
	if v.IsBlacklisted(claims.ID) {
		return nil, fmt.Errorf("token is blacklisted")
	}

	// Validate claims
	if err := v.validateClaims(claims); err != nil {
		return nil, fmt.Errorf("claims validation failed: %w", err)
	}

	return claims, nil
}

// parseToken parses the JWT token string
func (v *JWTValidator) parseToken(tokenString string) (*Claims, error) {
	// This is a simplified parser - in production use:
	// - github.com/golang-jwt/jwt/v5
	// - or similar JWT library

	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	// Placeholder - would decode and verify signature
	// In production, use proper JWT parsing
	return &Claims{
		Subject:   "user-001",
		Issuer:    v.config.Issuer,
		Audience:  []string{v.config.Audience},
		ExpiresAt: time.Now().Add(v.config.Expiration).Unix(),
		IssuedAt:  time.Now().Unix(),
		ID:        "token-001",
		Roles:     []string{"operator"},
	}, nil
}

// validateClaims validates the claims
func (v *JWTValidator) validateClaims(claims *Claims) error {
	now := time.Now().Unix()

	// Check expiration
	if claims.ExpiresAt > 0 && now > claims.ExpiresAt {
		return fmt.Errorf("token expired")
	}

	// Check not before
	if claims.NotBefore > 0 && now < claims.NotBefore {
		return fmt.Errorf("token not yet valid")
	}

	// Check issuer
	if v.config.Issuer != "" && claims.Issuer != v.config.Issuer {
		return fmt.Errorf("invalid issuer")
	}

	// Check audience
	if v.config.Audience != "" {
		found := false
		for _, aud := range claims.Audience {
			if aud == v.config.Audience {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("invalid audience")
		}
	}

	return nil
}

// Blacklist adds a token to the blacklist
func (v *JWTValidator) Blacklist(tokenID string) {
	v.blMutex.Lock()
	defer v.blMutex.Unlock()
	v.blacklist[tokenID] = true
}

// RemoveFromBlacklist removes a token from the blacklist
func (v *JWTValidator) RemoveFromBlacklist(tokenID string) {
	v.blMutex.Lock()
	defer v.blMutex.Unlock()
	delete(v.blacklist, tokenID)
}

// IsBlacklisted checks if a token is blacklisted
func (v *JWTValidator) IsBlacklisted(tokenID string) bool {
	v.blMutex.RLock()
	defer v.blMutex.RUnlock()
	return v.blacklist[tokenID]
}

// Token represents a JWT token pair
type Token struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresIn    time.Duration
}

// TokenGenerator generates JWT tokens
type TokenGenerator struct {
	config   *JWTConfig
	signingKey interface{}
	kid      string
}

// NewTokenGenerator creates a new token generator
func NewTokenGenerator(config *JWTConfig, signingKey interface{}, kid string) *TokenGenerator {
	return &TokenGenerator{
		config:     config,
		signingKey: signingKey,
		kid:        kid,
	}
}

// Generate generates a new token pair
func (g *TokenGenerator) Generate(subject string, roles, permissions []string) (*Token, error) {
	now := time.Now()

	// Generate access token
	accessClaims := &Claims{
		Subject:     subject,
		Issuer:      g.config.Issuer,
		Audience:    []string{g.config.Audience},
		ExpiresAt:   now.Add(g.config.Expiration).Unix(),
		IssuedAt:    now.Unix(),
		ID:          generateTokenID(),
		Roles:       roles,
		Permissions: permissions,
	}

	accessToken, err := g.generateToken(accessClaims)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshClaims := &Claims{
		Subject:   subject,
		Issuer:    g.config.Issuer,
		Audience:  []string{g.config.Audience},
		ExpiresAt: now.Add(g.config.Expiration * 7).Unix(),
		IssuedAt:  now.Unix(),
		ID:        generateTokenID(),
	}

	refreshToken, err := g.generateToken(refreshClaims)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    g.config.Expiration,
	}, nil
}

// generateToken generates a token string (placeholder)
func (g *TokenGenerator) generateToken(claims *Claims) (string, error) {
	// In production, use proper JWT signing:
	// - jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	// - token.SignedString(g.signingKey)

	return fmt.Sprintf("header.%s.signature", claims.ID), nil
}

// generateTokenID generates a unique token ID
func generateTokenID() string {
	return fmt.Sprintf("token-%d", time.Now().UnixNano())
}

// RefreshToken refreshes an access token using refresh token
func (v *JWTValidator) RefreshToken(ctx context.Context, refreshToken string, generator *TokenGenerator) (*Token, error) {
	// Validate refresh token
	claims, err := v.Validate(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("refresh token invalid: %w", err)
	}

	// Generate new token pair
	return generator.Generate(claims.Subject, claims.Roles, claims.Permissions)
}

// ExtractToken extracts token from Authorization header
func ExtractToken(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("missing authorization header")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", fmt.Errorf("invalid authorization header format")
	}

	return parts[1], nil
}

// HasRole checks if claims have a specific role
func (c *Claims) HasRole(role string) bool {
	for _, r := range c.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasPermission checks if claims have a specific permission
func (c *Claims) HasPermission(permission string) bool {
	for _, p := range c.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// IsExpired checks if claims are expired
func (c *Claims) IsExpired() bool {
	return time.Now().Unix() > c.ExpiresAt
}

// ExpiresIn returns time until expiration
func (c *Claims) ExpiresIn() time.Duration {
	return time.Until(time.Unix(c.ExpiresAt, 0))
}