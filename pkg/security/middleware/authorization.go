package middleware

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/yourusername/apm/pkg/security/auth"
)

// AuthorizationMiddleware provides authorization middleware
type AuthorizationMiddleware struct {
	rbacManager *auth.RBACManager
	logger      *zap.Logger
}

// NewAuthorizationMiddleware creates a new authorization middleware
func NewAuthorizationMiddleware(rbacConfig auth.RBACConfig, logger *zap.Logger) *AuthorizationMiddleware {
	return &AuthorizationMiddleware{
		rbacManager: auth.NewRBACManager(rbacConfig, logger),
		logger:      logger,
	}
}

// RequirePermission ensures user has permission for resource and action
func (m *AuthorizationMiddleware) RequirePermission(resource string, action string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authCtx := auth.GetAuthContext(c)
		if authCtx == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "authentication required",
			})
		}

		// Check permission
		if !m.rbacManager.CheckPermission(authCtx.User.Roles, resource, action) {
			m.logger.Warn("permission denied",
				zap.String("user_id", authCtx.User.ID),
				zap.Strings("roles", authCtx.User.Roles),
				zap.String("resource", resource),
				zap.String("action", action),
				zap.String("request_id", authCtx.RequestID))

			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":      "forbidden",
				"message":    "insufficient permissions",
				"resource":   resource,
				"action":     action,
				"request_id": authCtx.RequestID,
			})
		}

		return c.Next()
	}
}

// RequireResourcePermission dynamically checks permission based on route parameters
func (m *AuthorizationMiddleware) RequireResourcePermission(resourceParam string, action string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		resource := c.Params(resourceParam)
		if resource == "" {
			resource = resourceParam // Use as literal if not a param
		}

		return m.RequirePermission(resource, action)(c)
	}
}

// RequireAnyPermission ensures user has at least one of the specified permissions
func (m *AuthorizationMiddleware) RequireAnyPermission(permissions []struct {
	Resource string
	Action   string
}) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authCtx := auth.GetAuthContext(c)
		if authCtx == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "authentication required",
			})
		}

		// Check if user has any of the required permissions
		for _, perm := range permissions {
			if m.rbacManager.CheckPermission(authCtx.User.Roles, perm.Resource, perm.Action) {
				return c.Next()
			}
		}

		m.logger.Warn("all permissions denied",
			zap.String("user_id", authCtx.User.ID),
			zap.Strings("roles", authCtx.User.Roles),
			zap.Any("required_permissions", permissions),
			zap.String("request_id", authCtx.RequestID))

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":      "forbidden",
			"message":    "insufficient permissions",
			"request_id": authCtx.RequestID,
		})
	}
}

// ResourcePermissionMiddleware creates middleware for common resource operations
type ResourcePermissionMiddleware struct {
	authz    *AuthorizationMiddleware
	resource string
}

// NewResourcePermissionMiddleware creates middleware for a specific resource
func (m *AuthorizationMiddleware) ForResource(resource string) *ResourcePermissionMiddleware {
	return &ResourcePermissionMiddleware{
		authz:    m,
		resource: resource,
	}
}

// Create requires create permission
func (r *ResourcePermissionMiddleware) Create() fiber.Handler {
	return r.authz.RequirePermission(r.resource, string(auth.ActionCreate))
}

// Read requires read permission
func (r *ResourcePermissionMiddleware) Read() fiber.Handler {
	return r.authz.RequirePermission(r.resource, string(auth.ActionRead))
}

// Update requires update permission
func (r *ResourcePermissionMiddleware) Update() fiber.Handler {
	return r.authz.RequirePermission(r.resource, string(auth.ActionUpdate))
}

// Delete requires delete permission
func (r *ResourcePermissionMiddleware) Delete() fiber.Handler {
	return r.authz.RequirePermission(r.resource, string(auth.ActionDelete))
}

// List requires list permission
func (r *ResourcePermissionMiddleware) List() fiber.Handler {
	return r.authz.RequirePermission(r.resource, string(auth.ActionList))
}

// Deploy requires deploy permission
func (r *ResourcePermissionMiddleware) Deploy() fiber.Handler {
	return r.authz.RequirePermission(r.resource, string(auth.ActionDeploy))
}

// Manage requires manage permission
func (r *ResourcePermissionMiddleware) Manage() fiber.Handler {
	return r.authz.RequirePermission(r.resource, string(auth.ActionManage))
}
