package auth

import (
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// Role represents a user role
type Role struct {
	Name        string       `json:"name" yaml:"name"`
	Description string       `json:"description" yaml:"description"`
	Permissions []Permission `json:"permissions" yaml:"permissions"`
}

// Permission represents a permission
type Permission struct {
	Resource string   `json:"resource" yaml:"resource"`
	Actions  []string `json:"actions" yaml:"actions"`
}

// RBACConfig represents RBAC configuration
type RBACConfig struct {
	Roles       []Role            `json:"roles" yaml:"roles"`
	DefaultRole string            `json:"default_role" yaml:"default_role"`
	RoleMapping map[string]string `json:"role_mapping" yaml:"role_mapping"`
}

// Action represents an action that can be performed
type Action string

const (
	ActionCreate Action = "create"
	ActionRead   Action = "read"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"
	ActionList   Action = "list"
	ActionDeploy Action = "deploy"
	ActionManage Action = "manage"
	ActionAll    Action = "*"
)

// Resource represents a resource type
type Resource string

const (
	ResourceTools       Resource = "tools"
	ResourceDeployments Resource = "deployments"
	ResourceConfig      Resource = "configurations"
	ResourceMetrics     Resource = "metrics"
	ResourceLogs        Resource = "logs"
	ResourceAlerts      Resource = "alerts"
	ResourceDashboards  Resource = "dashboards"
	ResourceUsers       Resource = "users"
	ResourceAPIKeys     Resource = "api_keys"
	ResourceAll         Resource = "*"
)

// DefaultRoles provides standard roles
var DefaultRoles = []Role{
	{
		Name:        "admin",
		Description: "Full system administrator",
		Permissions: []Permission{
			{Resource: string(ResourceAll), Actions: []string{string(ActionAll)}},
		},
	},
	{
		Name:        "operator",
		Description: "Can manage deployments and configurations",
		Permissions: []Permission{
			{Resource: string(ResourceTools), Actions: []string{string(ActionRead), string(ActionUpdate), string(ActionManage)}},
			{Resource: string(ResourceDeployments), Actions: []string{string(ActionAll)}},
			{Resource: string(ResourceConfig), Actions: []string{string(ActionAll)}},
			{Resource: string(ResourceMetrics), Actions: []string{string(ActionRead), string(ActionList)}},
			{Resource: string(ResourceLogs), Actions: []string{string(ActionRead), string(ActionList)}},
			{Resource: string(ResourceAlerts), Actions: []string{string(ActionRead), string(ActionUpdate), string(ActionList)}},
			{Resource: string(ResourceDashboards), Actions: []string{string(ActionRead), string(ActionList)}},
		},
	},
	{
		Name:        "viewer",
		Description: "Read-only access to monitoring data",
		Permissions: []Permission{
			{Resource: string(ResourceTools), Actions: []string{string(ActionRead), string(ActionList)}},
			{Resource: string(ResourceDeployments), Actions: []string{string(ActionRead), string(ActionList)}},
			{Resource: string(ResourceConfig), Actions: []string{string(ActionRead), string(ActionList)}},
			{Resource: string(ResourceMetrics), Actions: []string{string(ActionRead), string(ActionList)}},
			{Resource: string(ResourceLogs), Actions: []string{string(ActionRead), string(ActionList)}},
			{Resource: string(ResourceAlerts), Actions: []string{string(ActionRead), string(ActionList)}},
			{Resource: string(ResourceDashboards), Actions: []string{string(ActionRead), string(ActionList)}},
		},
	},
}

// RBACManager manages role-based access control
type RBACManager struct {
	roles  map[string]*Role
	logger *zap.Logger
	mu     sync.RWMutex
}

// NewRBACManager creates a new RBAC manager
func NewRBACManager(config RBACConfig, logger *zap.Logger) *RBACManager {
	manager := &RBACManager{
		roles:  make(map[string]*Role),
		logger: logger,
	}

	// Load default roles if no roles configured
	if len(config.Roles) == 0 {
		config.Roles = DefaultRoles
	}

	// Load roles
	for i := range config.Roles {
		role := config.Roles[i]
		manager.roles[role.Name] = &role
		logger.Info("loaded role",
			zap.String("role", role.Name),
			zap.Int("permissions", len(role.Permissions)))
	}

	return manager
}

// CheckPermission checks if roles have permission for resource and action
func (m *RBACManager) CheckPermission(roles []string, resource string, action string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, roleName := range roles {
		role, exists := m.roles[roleName]
		if !exists {
			m.logger.Debug("role not found", zap.String("role", roleName))
			continue
		}

		// Check each permission in the role
		for _, perm := range role.Permissions {
			// Check resource match (with wildcard support)
			if perm.Resource != string(ResourceAll) && perm.Resource != resource {
				continue
			}

			// Check action match (with wildcard support)
			for _, permAction := range perm.Actions {
				if permAction == string(ActionAll) || permAction == action {
					m.logger.Debug("permission granted",
						zap.String("role", roleName),
						zap.String("resource", resource),
						zap.String("action", action))
					return true
				}
			}
		}
	}

	m.logger.Debug("permission denied",
		zap.Strings("roles", roles),
		zap.String("resource", resource),
		zap.String("action", action))
	return false
}

// GetRole gets a role by name
func (m *RBACManager) GetRole(name string) (*Role, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	role, exists := m.roles[name]
	if !exists {
		return nil, fmt.Errorf("role not found: %s", name)
	}

	return role, nil
}

// ListRoles lists all available roles
func (m *RBACManager) ListRoles() []Role {
	m.mu.RLock()
	defer m.mu.RUnlock()

	roles := make([]Role, 0, len(m.roles))
	for _, role := range m.roles {
		roles = append(roles, *role)
	}

	return roles
}

// AddRole adds a new role
func (m *RBACManager) AddRole(role Role) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.roles[role.Name]; exists {
		return fmt.Errorf("role already exists: %s", role.Name)
	}

	m.roles[role.Name] = &role
	m.logger.Info("added role",
		zap.String("role", role.Name),
		zap.Int("permissions", len(role.Permissions)))

	return nil
}

// UpdateRole updates an existing role
func (m *RBACManager) UpdateRole(role Role) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.roles[role.Name]; !exists {
		return fmt.Errorf("role not found: %s", role.Name)
	}

	m.roles[role.Name] = &role
	m.logger.Info("updated role",
		zap.String("role", role.Name),
		zap.Int("permissions", len(role.Permissions)))

	return nil
}

// DeleteRole deletes a role
func (m *RBACManager) DeleteRole(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.roles[name]; !exists {
		return fmt.Errorf("role not found: %s", name)
	}

	delete(m.roles, name)
	m.logger.Info("deleted role", zap.String("role", name))

	return nil
}
