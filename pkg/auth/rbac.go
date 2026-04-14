// Package auth provides RBAC (Role-Based Access Control)
package auth

import (
	"context"
	"fmt"
	"sync"
)

// Permission represents a permission
type Permission struct {
	Name        string
	Description string
	Resource    string
	Action      string
}

// Role represents a role with permissions
type Role struct {
	Name        string
	Description string
	Permissions []Permission
	Inherits    []string // Role names to inherit from
}

// RBACConfig holds RBAC configuration
type RBACConfig struct {
	DefaultRole string
	SuperAdmin  string
}

// DefaultRBACConfig returns default RBAC configuration
func DefaultRBACConfig() *RBACConfig {
	return &RBACConfig{
		DefaultRole: "viewer",
		SuperAdmin:  "admin",
	}
}

// RBACManager manages RBAC
type RBACManager struct {
	config      *RBACConfig
	roles       map[string]*Role
	users       map[string][]string // userID -> roleNames
	permissions map[string]Permission
	mu          sync.RWMutex
}

// NewRBACManager creates a new RBAC manager
func NewRBACManager(config *RBACConfig) *RBACManager {
	mgr := &RBACManager{
		config:      config,
		roles:       make(map[string]*Role),
		users:       make(map[string][]string),
		permissions: make(map[string]Permission),
	}

	// Add default roles
	mgr.addDefaultRoles()

	return mgr
}

// addDefaultRoles adds default VIGIL roles
func (m *RBACManager) addDefaultRoles() {
	// Viewer role
	m.AddRole(&Role{
		Name:        "viewer",
		Description: "Can view tracks and alerts",
		Permissions: []Permission{
			{Name: "tracks:read", Resource: "tracks", Action: "read"},
			{Name: "alerts:read", Resource: "alerts", Action: "read"},
			{Name: "events:read", Resource: "events", Action: "read"},
		},
	})

	// Operator role
	m.AddRole(&Role{
		Name:        "operator",
		Description: "Can view and acknowledge alerts",
		Inherits:    []string{"viewer"},
		Permissions: []Permission{
			{Name: "alerts:acknowledge", Resource: "alerts", Action: "acknowledge"},
			{Name: "alerts:complete", Resource: "alerts", Action: "complete"},
			{Name: "tracks:update", Resource: "tracks", Action: "update"},
		},
	})

	// Supervisor role
	m.AddRole(&Role{
		Name:        "supervisor",
		Description: "Can manage alerts and view reports",
		Inherits:    []string{"operator"},
		Permissions: []Permission{
			{Name: "alerts:escalate", Resource: "alerts", Action: "escalate"},
			{Name: "reports:read", Resource: "reports", Action: "read"},
			{Name: "events:write", Resource: "events", Action: "write"},
		},
	})

	// Admin role
	m.AddRole(&Role{
		Name:        "admin",
		Description: "Full administrative access",
		Inherits:    []string{"supervisor"},
		Permissions: []Permission{
			{Name: "tracks:write", Resource: "tracks", Action: "write"},
			{Name: "tracks:delete", Resource: "tracks", Action: "delete"},
			{Name: "alerts:write", Resource: "alerts", Action: "write"},
			{Name: "alerts:delete", Resource: "alerts", Action: "delete"},
			{Name: "users:read", Resource: "users", Action: "read"},
			{Name: "users:write", Resource: "users", Action: "write"},
			{Name: "config:read", Resource: "config", Action: "read"},
			{Name: "config:write", Resource: "config", Action: "write"},
			{Name: "system:admin", Resource: "system", Action: "admin"},
		},
	})
}

// AddRole adds a role
func (m *RBACManager) AddRole(role *Role) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.roles[role.Name] = role
	for _, perm := range role.Permissions {
		m.permissions[perm.Name] = perm
	}

	return nil
}

// GetRole gets a role by name
func (m *RBACManager) GetRole(name string) (*Role, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	role, ok := m.roles[name]
	if !ok {
		return nil, fmt.Errorf("role not found: %s", name)
	}

	return role, nil
}

// DeleteRole deletes a role
func (m *RBACManager) DeleteRole(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.roles, name)
	return nil
}

// ListRoles lists all roles
func (m *RBACManager) ListRoles() []*Role {
	m.mu.RLock()
	defer m.mu.RUnlock()

	roles := make([]*Role, 0, len(m.roles))
	for _, role := range m.roles {
		roles = append(roles, role)
	}

	return roles
}

// AssignRole assigns a role to a user
func (m *RBACManager) AssignRole(userID, roleName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if role exists
	if _, ok := m.roles[roleName]; !ok {
		return fmt.Errorf("role not found: %s", roleName)
	}

	// Check if already assigned
	for _, r := range m.users[userID] {
		if r == roleName {
			return nil // Already assigned
		}
	}

	m.users[userID] = append(m.users[userID], roleName)
	return nil
}

// RevokeRole revokes a role from a user
func (m *RBACManager) RevokeRole(userID, roleName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	roles := m.users[userID]
	newRoles := make([]string, 0)
	for _, r := range roles {
		if r != roleName {
			newRoles = append(newRoles, r)
		}
	}
	m.users[userID] = newRoles

	return nil
}

// GetUserRoles gets roles for a user
func (m *RBACManager) GetUserRoles(userID string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.users[userID]
}

// GetAllUserRoles gets all roles for a user including inherited
func (m *RBACManager) GetAllUserRoles(userID string) []string {
	roles := m.GetUserRoles(userID)
	return m.expandRoles(roles)
}

// expandRoles expands roles including inherited
func (m *RBACManager) expandRoles(roleNames []string) []string {
	seen := make(map[string]bool)
	var result []string

	var expand func(names []string)
	expand = func(names []string) {
		for _, name := range names {
			if seen[name] {
				continue
			}
			seen[name] = true

			role, ok := m.roles[name]
			if !ok {
				continue
			}

			result = append(result, name)
			expand(role.Inherits)
		}
	}

	m.mu.RLock()
	expand(roleNames)
	m.mu.RUnlock()

	return result
}

// CheckPermission checks if a user has a permission
func (m *RBACManager) CheckPermission(userID, resource, action string) (bool, error) {
	roleNames := m.GetAllUserRoles(userID)

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, roleName := range roleNames {
		role, ok := m.roles[roleName]
		if !ok {
			continue
		}

		if m.roleHasPermission(role, resource, action) {
			return true, nil
		}
	}

	return false, nil
}

// roleHasPermission checks if a role has a specific permission
func (m *RBACManager) roleHasPermission(role *Role, resource, action string) bool {
	for _, perm := range role.Permissions {
		if perm.Resource == resource && perm.Action == action {
			return true
		}
		if perm.Action == "*" && perm.Resource == resource {
			return true
		}
		if perm.Resource == "*" && perm.Action == action {
			return true
		}
		if perm.Resource == "*" && perm.Action == "*" {
			return true
		}
	}
	return false
}

// GetUserPermissions gets all permissions for a user
func (m *RBACManager) GetUserPermissions(userID string) []Permission {
	roleNames := m.GetAllUserRoles(userID)

	m.mu.RLock()
	defer m.mu.RUnlock()

	perms := make(map[string]Permission)
	for _, roleName := range roleNames {
		role, ok := m.roles[roleName]
		if !ok {
			continue
		}

		for _, perm := range role.Permissions {
			perms[perm.Name] = perm
		}
	}

	result := make([]Permission, 0, len(perms))
	for _, perm := range perms {
		result = append(result, perm)
	}

	return result
}

// AddPermissionToRole adds a permission to a role
func (m *RBACManager) AddPermissionToRole(roleName, permName, resource, action string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	role, ok := m.roles[roleName]
	if !ok {
		return fmt.Errorf("role not found: %s", roleName)
	}

	perm := Permission{
		Name:     permName,
		Resource: resource,
		Action:   action,
	}

	role.Permissions = append(role.Permissions, perm)
	m.permissions[permName] = perm

	return nil
}

// RemovePermissionFromRole removes a permission from a role
func (m *RBACManager) RemovePermissionFromRole(roleName, permName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	role, ok := m.roles[roleName]
	if !ok {
		return fmt.Errorf("role not found: %s", roleName)
	}

	newPerms := make([]Permission, 0)
	for _, perm := range role.Permissions {
		if perm.Name != permName {
			newPerms = append(newPerms, perm)
		}
	}

	role.Permissions = newPerms
	return nil
}

// Check checks if user has permission (context version)
func (m *RBACManager) Check(ctx context.Context, userID, resource, action string) bool {
	has, _ := m.CheckPermission(userID, resource, action)
	return has
}

// IsAdmin checks if user has admin role
func (m *RBACManager) IsAdmin(userID string) bool {
	roles := m.GetAllUserRoles(userID)
	for _, r := range roles {
		if r == m.config.SuperAdmin {
			return true
		}
	}
	return false
}