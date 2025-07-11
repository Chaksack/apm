package middleware

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/yourusername/apm/pkg/security/validator"
)

// ValidationMiddleware provides input validation middleware
type ValidationMiddleware struct {
	validator *validator.Validator
	sanitizer *validator.Sanitizer
	logger    *zap.Logger
}

// NewValidationMiddleware creates a new validation middleware
func NewValidationMiddleware(logger *zap.Logger) *ValidationMiddleware {
	return &ValidationMiddleware{
		validator: validator.NewValidator(logger),
		sanitizer: validator.NewSanitizer(logger),
		logger:    logger,
	}
}

// ValidateRequest validates request against rules
func (m *ValidationMiddleware) ValidateRequest(rules validator.RequestValidationRules) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var validationErrors []validator.ValidationError

		// Validate body
		if len(rules.Body) > 0 && c.Method() != fiber.MethodGet {
			var body map[string]interface{}
			if err := c.BodyParser(&body); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error":   "invalid_request",
					"message": "invalid request body",
					"details": err.Error(),
				})
			}

			if err := m.validator.ValidateMap(body, rules.Body); err != nil {
				if vErr, ok := err.(validator.ValidationErrors); ok {
					validationErrors = append(validationErrors, vErr...)
				}
			}
		}

		// Validate query parameters
		if len(rules.Query) > 0 {
			queryParams := make(map[string]interface{})
			c.Request().URI().QueryArgs().VisitAll(func(key, value []byte) {
				queryParams[string(key)] = string(value)
			})

			if err := m.validator.ValidateMap(queryParams, rules.Query); err != nil {
				if vErr, ok := err.(validator.ValidationErrors); ok {
					for _, e := range vErr {
						e.Field = "query." + e.Field
						validationErrors = append(validationErrors, e)
					}
				}
			}
		}

		// Validate route parameters
		if len(rules.Params) > 0 {
			params := make(map[string]interface{})
			for key, _ := range rules.Params {
				if value := c.Params(key); value != "" {
					params[key] = value
				}
			}

			if err := m.validator.ValidateMap(params, rules.Params); err != nil {
				if vErr, ok := err.(validator.ValidationErrors); ok {
					for _, e := range vErr {
						e.Field = "param." + e.Field
						validationErrors = append(validationErrors, e)
					}
				}
			}
		}

		// Validate headers
		if len(rules.Headers) > 0 {
			headers := make(map[string]interface{})
			c.Request().Header.VisitAll(func(key, value []byte) {
				headers[string(key)] = string(value)
			})

			if err := m.validator.ValidateMap(headers, rules.Headers); err != nil {
				if vErr, ok := err.(validator.ValidationErrors); ok {
					for _, e := range vErr {
						e.Field = "header." + e.Field
						validationErrors = append(validationErrors, e)
					}
				}
			}
		}

		// Return validation errors if any
		if len(validationErrors) > 0 {
			m.logger.Debug("validation failed",
				zap.String("path", c.Path()),
				zap.String("method", c.Method()),
				zap.Any("errors", validationErrors))

			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "validation_failed",
				"message": "request validation failed",
				"errors":  validationErrors,
			})
		}

		return c.Next()
	}
}

// SanitizeInput sanitizes input data
func (m *ValidationMiddleware) SanitizeInput() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Sanitize query parameters
		c.Request().URI().QueryArgs().VisitAll(func(key, value []byte) {
			sanitized := m.sanitizeValue(string(value))
			c.Request().URI().QueryArgs().Set(string(key), sanitized)
		})

		// Sanitize route parameters
		for _, param := range c.Route().Params {
			value := c.Params(param)
			if value != "" {
				sanitized := m.sanitizeValue(value)
				// Store sanitized value in locals
				c.Locals(fmt.Sprintf("param_%s", param), sanitized)
			}
		}

		// Sanitize headers
		c.Request().Header.VisitAll(func(key, value []byte) {
			// Skip content-type and similar headers
			keyStr := string(key)
			if !strings.HasPrefix(strings.ToLower(keyStr), "content-") &&
				!strings.HasPrefix(strings.ToLower(keyStr), "accept") {
				sanitized := m.sanitizeValue(string(value))
				c.Request().Header.Set(keyStr, sanitized)
			}
		})

		return c.Next()
	}
}

// ValidateJSON validates JSON request body against a struct type
func (m *ValidationMiddleware) ValidateJSON(structType interface{}) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Only validate for methods that have body
		if c.Method() == fiber.MethodGet || c.Method() == fiber.MethodHead {
			return c.Next()
		}

		// Parse body into the struct
		if err := c.BodyParser(structType); err != nil {
			m.logger.Debug("failed to parse JSON body",
				zap.String("path", c.Path()),
				zap.Error(err))

			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "invalid_json",
				"message": "invalid JSON in request body",
				"details": err.Error(),
			})
		}

		// Store parsed body for handler use
		c.Locals("parsed_body", structType)

		return c.Next()
	}
}

// PreventSQLInjection checks for common SQL injection patterns
func (m *ValidationMiddleware) PreventSQLInjection() fiber.Handler {
	sqlPatterns := []string{
		"';",
		"--",
		"/*",
		"*/",
		"xp_",
		"sp_",
		"exec",
		"execute",
		"select",
		"insert",
		"update",
		"delete",
		"drop",
		"create",
		"alter",
		"union",
	}

	return func(c *fiber.Ctx) error {
		// Check all input sources
		inputs := m.collectAllInputs(c)

		for source, value := range inputs {
			lowerValue := strings.ToLower(value)
			for _, pattern := range sqlPatterns {
				if strings.Contains(lowerValue, pattern) {
					m.logger.Warn("potential SQL injection attempt",
						zap.String("source", source),
						zap.String("pattern", pattern),
						zap.String("value", value),
						zap.String("ip", c.IP()))

					return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
						"error":   "invalid_input",
						"message": "potentially malicious input detected",
					})
				}
			}
		}

		return c.Next()
	}
}

// PreventCommandInjection checks for command injection patterns
func (m *ValidationMiddleware) PreventCommandInjection() fiber.Handler {
	cmdPatterns := []string{
		"&&",
		"||",
		";",
		"|",
		"`",
		"$(",
		"${",
		"\\n",
		"\\r",
	}

	return func(c *fiber.Ctx) error {
		// Check all input sources
		inputs := m.collectAllInputs(c)

		for source, value := range inputs {
			for _, pattern := range cmdPatterns {
				if strings.Contains(value, pattern) {
					m.logger.Warn("potential command injection attempt",
						zap.String("source", source),
						zap.String("pattern", pattern),
						zap.String("value", value),
						zap.String("ip", c.IP()))

					return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
						"error":   "invalid_input",
						"message": "potentially malicious input detected",
					})
				}
			}
		}

		return c.Next()
	}
}

// Helper methods

// sanitizeValue applies basic sanitization to a string value
func (m *ValidationMiddleware) sanitizeValue(value string) string {
	// Basic sanitization rules
	rules := []validator.SanitizationRule{
		{
			TrimSpace:    true,
			RemoveScript: true,
			ReplacePatterns: map[string]string{
				`<[^>]+>`: "", // Remove HTML tags
			},
		},
	}

	return m.sanitizer.SanitizeString(value, rules)
}

// collectAllInputs collects all inputs from various sources
func (m *ValidationMiddleware) collectAllInputs(c *fiber.Ctx) map[string]string {
	inputs := make(map[string]string)

	// Query parameters
	c.Request().URI().QueryArgs().VisitAll(func(key, value []byte) {
		inputs[fmt.Sprintf("query.%s", key)] = string(value)
	})

	// Route parameters
	for _, param := range c.Route().Params {
		if value := c.Params(param); value != "" {
			inputs[fmt.Sprintf("param.%s", param)] = value
		}
	}

	// Headers
	c.Request().Header.VisitAll(func(key, value []byte) {
		inputs[fmt.Sprintf("header.%s", key)] = string(value)
	})

	// Body (if JSON)
	if c.Method() != fiber.MethodGet && strings.Contains(c.Get("Content-Type"), "application/json") {
		var body map[string]interface{}
		if err := json.Unmarshal(c.Body(), &body); err == nil {
			m.flattenMap("body", body, inputs)
		}
	}

	return inputs
}

// flattenMap flattens a nested map for validation
func (m *ValidationMiddleware) flattenMap(prefix string, data map[string]interface{}, result map[string]string) {
	for key, value := range data {
		fullKey := fmt.Sprintf("%s.%s", prefix, key)

		switch v := value.(type) {
		case string:
			result[fullKey] = v
		case map[string]interface{}:
			m.flattenMap(fullKey, v, result)
		default:
			result[fullKey] = fmt.Sprintf("%v", v)
		}
	}
}
