package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/yourusername/apm/pkg/security/auth"
)

// AuthMiddleware provides authentication middleware
type AuthMiddleware struct {
	jwtManager    *auth.JWTManager
	apiKeyManager *auth.APIKeyManager
	config        auth.AuthConfig
	logger        *zap.Logger
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(config auth.AuthConfig, logger *zap.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		jwtManager:    auth.NewJWTManager(config.JWT, logger),
		apiKeyManager: auth.NewAPIKeyManager(config.APIKey, logger),
		config:        config,
		logger:        logger,
	}
}

// Authenticate returns a fiber middleware function for authentication
func (m *AuthMiddleware) Authenticate() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Generate request ID
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
			c.Set("X-Request-ID", requestID)
		}

		// Try JWT authentication first if enabled
		if m.config.EnableJWT {
			token := extractBearerToken(c)
			if token != "" {
				claims, err := m.jwtManager.ValidateToken(token)
				if err == nil {
					// JWT authentication successful
					authCtx := &auth.AuthContext{
						User: &auth.User{
							ID:       claims.Subject,
							Username: claims.User.Username,
							Email:    claims.User.Email,
							Roles:    claims.Roles,
						},
						AuthType:  auth.AuthTypeJWT,
						Token:     token,
						Claims:    claims,
						RequestID: requestID,
					}
					auth.SetAuthContext(c, authCtx)

					m.logger.Debug("JWT authentication successful",
						zap.String("user_id", claims.Subject),
						zap.String("request_id", requestID))

					return c.Next()
				}

				m.logger.Debug("JWT authentication failed",
					zap.Error(err),
					zap.String("request_id", requestID))
			}
		}

		// Try API key authentication if enabled
		if m.config.EnableAPI {
			apiKey := extractAPIKey(c, m.config.APIKey)
			if apiKey != "" {
				key, err := m.apiKeyManager.ValidateAPIKey(apiKey)
				if err == nil {
					// API key authentication successful
					authCtx := &auth.AuthContext{
						User: &auth.User{
							ID:    key.UserID,
							Roles: key.Roles,
						},
						AuthType:  auth.AuthTypeAPIKey,
						Token:     apiKey,
						RequestID: requestID,
					}
					auth.SetAuthContext(c, authCtx)

					m.logger.Debug("API key authentication successful",
						zap.String("key_id", key.ID),
						zap.String("user_id", key.UserID),
						zap.String("request_id", requestID))

					return c.Next()
				}

				m.logger.Debug("API key authentication failed",
					zap.Error(err),
					zap.String("request_id", requestID))
			}
		}

		// No valid authentication found
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":      "unauthorized",
			"message":    "valid authentication required",
			"request_id": requestID,
		})
	}
}

// RequireAuth ensures request is authenticated
func RequireAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authCtx := auth.GetAuthContext(c)
		if authCtx == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "authentication required",
			})
		}
		return c.Next()
	}
}

// RequireRoles ensures user has at least one of the specified roles
func RequireRoles(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authCtx := auth.GetAuthContext(c)
		if authCtx == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "authentication required",
			})
		}

		// Check if user has at least one required role
		hasRole := false
		for _, requiredRole := range roles {
			for _, userRole := range authCtx.User.Roles {
				if userRole == requiredRole {
					hasRole = true
					break
				}
			}
			if hasRole {
				break
			}
		}

		if !hasRole {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "forbidden",
				"message": "insufficient permissions",
			})
		}

		return c.Next()
	}
}

// extractBearerToken extracts bearer token from Authorization header
func extractBearerToken(c *fiber.Ctx) string {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}

	return parts[1]
}

// extractAPIKey extracts API key from request
func extractAPIKey(c *fiber.Ctx, config auth.APIKeyConfig) string {
	// Build headers map
	headers := make(map[string]string)
	c.Request().Header.VisitAll(func(key, value []byte) {
		headers[string(key)] = string(value)
	})

	// Build query map
	query := make(map[string]string)
	c.Request().URI().QueryArgs().VisitAll(func(key, value []byte) {
		query[string(key)] = string(value)
	})

	return auth.ExtractAPIKey(headers, query, config)
}
