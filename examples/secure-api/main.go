package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"go.uber.org/zap"

	"github.com/yourusername/apm/pkg/security"
	"github.com/yourusername/apm/pkg/security/auth"
	"github.com/yourusername/apm/pkg/security/middleware"
	"github.com/yourusername/apm/pkg/security/validator"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Sync()

	// Load security configuration
	securityConfig := security.DefaultConfig()

	// Create Fiber app with security configuration
	app := fiber.New(fiber.Config{
		ErrorHandler:          middleware.ErrorHandler(logger),
		DisableStartupMessage: false,
		AppName:               "Secure APM API",
		BodyLimit:             securityConfig.APISecurity.MaxRequestBodySize,
	})

	// Initialize security middleware
	authMiddleware := middleware.NewAuthMiddleware(securityConfig.Auth, logger)
	authzMiddleware := middleware.NewAuthorizationMiddleware(securityConfig.RBAC, logger)
	validationMiddleware := middleware.NewValidationMiddleware(logger)
	headersMiddleware := middleware.NewSecurityHeadersMiddleware(securityConfig.Headers, logger)
	corsMiddleware := middleware.NewCORSMiddleware(securityConfig.CORS, logger)
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(securityConfig.RateLimit, logger)
	auditMiddleware := middleware.NewAuditMiddleware(securityConfig.Audit, logger)
	csrfMiddleware := middleware.NewCSRFMiddleware(securityConfig.CSRF, logger)
	apiSecurityMiddleware := middleware.NewAPISecurityMiddleware(securityConfig.APISecurity, logger)

	// Apply global middleware in security-conscious order

	// 1. Recovery middleware (catch panics)
	app.Use(recover.New())

	// 2. Request ID tracking
	app.Use(apiSecurityMiddleware.RequestIDTracking())

	// 3. Request timeout
	app.Use(apiSecurityMiddleware.RequestTimeout())

	// 4. Request size limits
	app.Use(apiSecurityMiddleware.RequestSizeLimits())

	// 5. Security headers
	app.Use(headersMiddleware.Apply())

	// 6. CORS
	app.Use(corsMiddleware.Apply())

	// 7. Rate limiting and DDoS protection
	app.Use(rateLimitMiddleware.Apply())
	app.Use(rateLimitMiddleware.DDoSProtection())

	// 8. Input sanitization
	app.Use(validationMiddleware.SanitizeInput())

	// 9. SQL/Command injection prevention
	app.Use(validationMiddleware.PreventSQLInjection())
	app.Use(validationMiddleware.PreventCommandInjection())

	// 10. Request timing
	app.Use(apiSecurityMiddleware.RequestTiming())

	// 11. Security context
	app.Use(apiSecurityMiddleware.SecurityContext())

	// 12. Audit logging (capture body for audit)
	app.Use(middleware.CaptureRequestBody())
	app.Use(auditMiddleware.Apply())

	// 13. Authentication (for protected routes)
	// Applied per route or route group

	// Health check endpoint (public)
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"service": "apm-api",
		})
	})

	// CSRF token endpoint
	app.Get("/api/csrf-token", middleware.CSRFTokenHandler(csrfMiddleware))

	// API routes
	api := app.Group("/api")

	// Authentication routes (public)
	authRoutes := api.Group("/auth")
	authRoutes.Post("/login",
		validationMiddleware.ValidateRequest(validator.RequestValidationRules{
			Body: map[string][]validator.ValidationRule{
				"username": {validator.NameValidation},
				"password": {{Required: true, MinLength: 8}},
			},
		}),
		loginHandler(authMiddleware, auditMiddleware),
	)

	authRoutes.Post("/refresh",
		refreshTokenHandler(authMiddleware),
	)

	// Protected routes
	protected := api.Group("")
	protected.Use(authMiddleware.Authenticate())
	protected.Use(csrfMiddleware.Apply())

	// User management (admin only)
	users := protected.Group("/users")
	users.Use(middleware.RequireRoles("admin"))

	users.Get("/",
		authzMiddleware.RequirePermission(string(auth.ResourceUsers), string(auth.ActionList)),
		listUsersHandler(),
	)

	users.Post("/",
		authzMiddleware.RequirePermission(string(auth.ResourceUsers), string(auth.ActionCreate)),
		validationMiddleware.ValidateRequest(validator.RequestValidationRules{
			Body: map[string][]validator.ValidationRule{
				"username": {validator.NameValidation},
				"email":    {validator.EmailValidation},
				"roles":    {{Required: true}},
			},
		}),
		createUserHandler(),
	)

	// Deployment routes
	deployments := protected.Group("/deployments")
	deploymentPerms := authzMiddleware.ForResource(string(auth.ResourceDeployments))

	deployments.Get("/",
		deploymentPerms.List(),
		listDeploymentsHandler(),
	)

	deployments.Post("/",
		deploymentPerms.Create(),
		validationMiddleware.ValidateRequest(validator.DeploymentValidationRules),
		createDeploymentHandler(auditMiddleware),
	)

	deployments.Put("/:id",
		deploymentPerms.Update(),
		validationMiddleware.ValidateRequest(validator.RequestValidationRules{
			Params: map[string][]validator.ValidationRule{
				"id": {validator.IDValidation},
			},
			Body: validator.DeploymentValidationRules.Body,
		}),
		updateDeploymentHandler(auditMiddleware),
	)

	// Configuration routes
	configs := protected.Group("/configurations")
	configPerms := authzMiddleware.ForResource(string(auth.ResourceConfig))

	configs.Get("/",
		configPerms.List(),
		listConfigurationsHandler(),
	)

	configs.Post("/",
		configPerms.Create(),
		validationMiddleware.ValidateRequest(validator.ConfigValidationRules),
		createConfigurationHandler(auditMiddleware),
	)

	// API Key management
	apiKeys := protected.Group("/api-keys")
	apiKeys.Use(middleware.RequireRoles("admin", "operator"))

	apiKeys.Get("/",
		authzMiddleware.RequirePermission(string(auth.ResourceAPIKeys), string(auth.ActionList)),
		listAPIKeysHandler(),
	)

	apiKeys.Post("/",
		authzMiddleware.RequirePermission(string(auth.ResourceAPIKeys), string(auth.ActionCreate)),
		validationMiddleware.ValidateRequest(validator.APIKeyValidationRules),
		createAPIKeyHandler(authMiddleware),
	)

	// Start server
	logger.Info("Starting secure APM API server on :8080")
	if err := app.Listen(":8080"); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}

// Handler implementations

func loginHandler(authMiddleware *middleware.AuthMiddleware, auditMiddleware *middleware.AuditMiddleware) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid request",
			})
		}

		// TODO: Validate credentials against your user store
		// This is just an example
		if req.Username != "admin" || req.Password != "secure_password" {
			auditMiddleware.LogAuthEvent(
				auth.EventTypeAuthFailure,
				"",
				req.Username,
				false,
				map[string]interface{}{
					"ip":     c.IP(),
					"reason": "invalid credentials",
				},
			)

			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid credentials",
			})
		}

		// Create user object
		user := &auth.User{
			ID:       "user-123",
			Username: req.Username,
			Email:    "admin@example.com",
			Roles:    []string{"admin"},
		}

		// Generate tokens
		// Note: authMiddleware would need a method to access jwtManager
		// This is simplified for the example

		auditMiddleware.LogAuthEvent(
			auth.EventTypeAuthSuccess,
			user.ID,
			user.Username,
			true,
			map[string]interface{}{
				"ip": c.IP(),
			},
		)

		return c.JSON(fiber.Map{
			"message": "login successful",
			"user":    user,
			// "tokens": tokens,
		})
	}
}

func refreshTokenHandler(authMiddleware *middleware.AuthMiddleware) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// TODO: Implement token refresh logic
		return c.JSON(fiber.Map{
			"message": "token refreshed",
		})
	}
}

func listUsersHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"users": []fiber.Map{
				{
					"id":       "user-123",
					"username": "admin",
					"email":    "admin@example.com",
					"roles":    []string{"admin"},
				},
			},
		})
	}
}

func createUserHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"message": "user created",
		})
	}
}

func listDeploymentsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"deployments": []fiber.Map{},
		})
	}
}

func createDeploymentHandler(auditMiddleware *middleware.AuditMiddleware) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authCtx := auth.GetAuthContext(c)

		auditMiddleware.LogConfigChange(
			authCtx.User.ID,
			"deployment",
			"create",
			map[string]interface{}{
				"environment": "production",
			},
		)

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"message": "deployment created",
		})
	}
}

func updateDeploymentHandler(auditMiddleware *middleware.AuditMiddleware) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		authCtx := auth.GetAuthContext(c)

		auditMiddleware.LogConfigChange(
			authCtx.User.ID,
			"deployment",
			"update",
			map[string]interface{}{
				"deployment_id": id,
			},
		)

		return c.JSON(fiber.Map{
			"message": "deployment updated",
		})
	}
}

func listConfigurationsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"configurations": []fiber.Map{},
		})
	}
}

func createConfigurationHandler(auditMiddleware *middleware.AuditMiddleware) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authCtx := auth.GetAuthContext(c)

		auditMiddleware.LogConfigChange(
			authCtx.User.ID,
			"configuration",
			"create",
			map[string]interface{}{
				"type": "prometheus",
			},
		)

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"message": "configuration created",
		})
	}
}

func listAPIKeysHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authCtx := auth.GetAuthContext(c)

		return c.JSON(fiber.Map{
			"api_keys": []fiber.Map{
				{
					"id":         "key-123",
					"name":       "Test Key",
					"user_id":    authCtx.User.ID,
					"created_at": "2024-01-01T00:00:00Z",
				},
			},
		})
	}
}

func createAPIKeyHandler(authMiddleware *middleware.AuthMiddleware) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authCtx := auth.GetAuthContext(c)

		// TODO: Implement API key creation

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"message": "API key created",
			"key_id":  "key-456",
			"user_id": authCtx.User.ID,
		})
	}
}
