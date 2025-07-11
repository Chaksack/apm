package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// APISecurityConfig represents API security configuration
type APISecurityConfig struct {
	// Request timeout
	RequestTimeout time.Duration `yaml:"request_timeout" json:"request_timeout"`

	// Maximum request body size
	MaxRequestBodySize int `yaml:"max_request_body_size" json:"max_request_body_size"`

	// Maximum URL length
	MaxURLLength int `yaml:"max_url_length" json:"max_url_length"`

	// Maximum header size
	MaxHeaderSize int `yaml:"max_header_size" json:"max_header_size"`

	// Maximum number of headers
	MaxHeaderCount int `yaml:"max_header_count" json:"max_header_count"`

	// Enable request ID tracking
	EnableRequestID bool `yaml:"enable_request_id" json:"enable_request_id"`

	// Request ID header name
	RequestIDHeader string `yaml:"request_id_header" json:"request_id_header"`

	// Enable request timing
	EnableRequestTiming bool `yaml:"enable_request_timing" json:"enable_request_timing"`

	// Slow request threshold
	SlowRequestThreshold time.Duration `yaml:"slow_request_threshold" json:"slow_request_threshold"`
}

// DefaultAPISecurityConfig provides default API security configuration
var DefaultAPISecurityConfig = APISecurityConfig{
	RequestTimeout:       30 * time.Second,
	MaxRequestBodySize:   10 * 1024 * 1024, // 10MB
	MaxURLLength:         2048,
	MaxHeaderSize:        8192,
	MaxHeaderCount:       100,
	EnableRequestID:      true,
	RequestIDHeader:      "X-Request-ID",
	EnableRequestTiming:  true,
	SlowRequestThreshold: 5 * time.Second,
}

// APISecurityMiddleware provides API security enhancements
type APISecurityMiddleware struct {
	config APISecurityConfig
	logger *zap.Logger
}

// NewAPISecurityMiddleware creates a new API security middleware
func NewAPISecurityMiddleware(config APISecurityConfig, logger *zap.Logger) *APISecurityMiddleware {
	// Apply defaults
	if config.RequestTimeout == 0 {
		config.RequestTimeout = DefaultAPISecurityConfig.RequestTimeout
	}
	if config.MaxRequestBodySize == 0 {
		config.MaxRequestBodySize = DefaultAPISecurityConfig.MaxRequestBodySize
	}
	if config.MaxURLLength == 0 {
		config.MaxURLLength = DefaultAPISecurityConfig.MaxURLLength
	}
	if config.MaxHeaderSize == 0 {
		config.MaxHeaderSize = DefaultAPISecurityConfig.MaxHeaderSize
	}
	if config.MaxHeaderCount == 0 {
		config.MaxHeaderCount = DefaultAPISecurityConfig.MaxHeaderCount
	}
	if config.RequestIDHeader == "" {
		config.RequestIDHeader = DefaultAPISecurityConfig.RequestIDHeader
	}
	if config.SlowRequestThreshold == 0 {
		config.SlowRequestThreshold = DefaultAPISecurityConfig.SlowRequestThreshold
	}

	return &APISecurityMiddleware{
		config: config,
		logger: logger,
	}
}

// RequestTimeout adds request timeout middleware
func (m *APISecurityMiddleware) RequestTimeout() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Create context with timeout
		ctx, cancel := context.WithTimeout(c.Context(), m.config.RequestTimeout)
		defer cancel()

		// Replace fiber context
		c.SetUserContext(ctx)

		// Create channel for handler completion
		done := make(chan error, 1)

		// Run handler in goroutine
		go func() {
			done <- c.Next()
		}()

		// Wait for completion or timeout
		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			m.logger.Warn("request timeout",
				zap.String("path", c.Path()),
				zap.String("method", c.Method()),
				zap.String("ip", c.IP()),
				zap.Duration("timeout", m.config.RequestTimeout))

			return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{
				"error":   "request_timeout",
				"message": "request processing timeout",
			})
		}
	}
}

// RequestSizeLimits adds request size validation
func (m *APISecurityMiddleware) RequestSizeLimits() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check URL length
		if len(c.Request().URI().String()) > m.config.MaxURLLength {
			m.logger.Warn("URL too long",
				zap.Int("length", len(c.Request().URI().String())),
				zap.Int("max", m.config.MaxURLLength))

			return c.Status(fiber.StatusRequestURITooLong).JSON(fiber.Map{
				"error":   "url_too_long",
				"message": fmt.Sprintf("URL exceeds maximum length of %d", m.config.MaxURLLength),
			})
		}

		// Check header count
		headerCount := 0
		c.Request().Header.VisitAll(func(key, value []byte) {
			headerCount++
		})

		if headerCount > m.config.MaxHeaderCount {
			m.logger.Warn("too many headers",
				zap.Int("count", headerCount),
				zap.Int("max", m.config.MaxHeaderCount))

			return c.Status(fiber.StatusRequestHeaderFieldsTooLarge).JSON(fiber.Map{
				"error":   "too_many_headers",
				"message": fmt.Sprintf("header count exceeds maximum of %d", m.config.MaxHeaderCount),
			})
		}

		// Check individual header sizes
		headerSizeExceeded := false
		c.Request().Header.VisitAll(func(key, value []byte) {
			if len(key)+len(value) > m.config.MaxHeaderSize {
				headerSizeExceeded = true
			}
		})

		if headerSizeExceeded {
			return c.Status(fiber.StatusRequestHeaderFieldsTooLarge).JSON(fiber.Map{
				"error":   "header_too_large",
				"message": fmt.Sprintf("header size exceeds maximum of %d bytes", m.config.MaxHeaderSize),
			})
		}

		// Check body size
		if c.Request().Header.ContentLength() > m.config.MaxRequestBodySize {
			m.logger.Warn("request body too large",
				zap.Int("size", c.Request().Header.ContentLength()),
				zap.Int("max", m.config.MaxRequestBodySize))

			return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
				"error":   "body_too_large",
				"message": fmt.Sprintf("request body exceeds maximum size of %d bytes", m.config.MaxRequestBodySize),
			})
		}

		return c.Next()
	}
}

// RequestIDTracking adds request ID tracking
func (m *APISecurityMiddleware) RequestIDTracking() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !m.config.EnableRequestID {
			return c.Next()
		}

		// Get or generate request ID
		requestID := c.Get(m.config.RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Set request ID in context
		c.Locals("request_id", requestID)

		// Set response header
		c.Set(m.config.RequestIDHeader, requestID)

		// Add to logger context
		logger := m.logger.With(zap.String("request_id", requestID))
		c.Locals("logger", logger)

		return c.Next()
	}
}

// RequestTiming adds request timing middleware
func (m *APISecurityMiddleware) RequestTiming() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !m.config.EnableRequestTiming {
			return c.Next()
		}

		start := time.Now()

		// Execute handler
		err := c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Add timing header
		c.Set("X-Response-Time", fmt.Sprintf("%dms", duration.Milliseconds()))

		// Log slow requests
		if duration > m.config.SlowRequestThreshold {
			requestID, _ := c.Locals("request_id").(string)

			m.logger.Warn("slow request detected",
				zap.String("path", c.Path()),
				zap.String("method", c.Method()),
				zap.Duration("duration", duration),
				zap.Duration("threshold", m.config.SlowRequestThreshold),
				zap.String("request_id", requestID))
		}

		return err
	}
}

// SecurityContext adds security context propagation
func (m *APISecurityMiddleware) SecurityContext() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Create security context
		secCtx := &SecurityContext{
			RequestID: getRequestID(c),
			IP:        c.IP(),
			UserAgent: c.Get("User-Agent"),
			Method:    c.Method(),
			Path:      c.Path(),
			StartTime: time.Now(),
		}

		// Store in locals
		c.Locals("security_context", secCtx)

		return c.Next()
	}
}

// SecurityContext represents security context for request
type SecurityContext struct {
	RequestID string
	IP        string
	UserAgent string
	Method    string
	Path      string
	StartTime time.Time
	UserID    string
	Roles     []string
}

// GetSecurityContext retrieves security context from fiber context
func GetSecurityContext(c *fiber.Ctx) *SecurityContext {
	ctx, _ := c.Locals("security_context").(*SecurityContext)
	return ctx
}

// ErrorHandler provides secure error handling
func ErrorHandler(logger *zap.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		// Default to 500
		code := fiber.StatusInternalServerError
		message := "internal server error"

		// Check if it's a fiber error
		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
			message = e.Message
		}

		// Get request ID
		requestID := getRequestID(c)

		// Log error
		logger.Error("request error",
			zap.Error(err),
			zap.Int("status", code),
			zap.String("path", c.Path()),
			zap.String("method", c.Method()),
			zap.String("ip", c.IP()),
			zap.String("request_id", requestID))

		// Don't expose internal errors to client
		if code == fiber.StatusInternalServerError {
			message = "An internal error occurred"
		}

		// Send error response
		return c.Status(code).JSON(fiber.Map{
			"error":      getErrorType(code),
			"message":    message,
			"request_id": requestID,
			"timestamp":  time.Now().UTC(),
		})
	}
}

// Helper functions

// getRequestID gets request ID from context
func getRequestID(c *fiber.Ctx) string {
	if id, ok := c.Locals("request_id").(string); ok {
		return id
	}
	return ""
}

// getErrorType returns error type based on status code
func getErrorType(code int) string {
	switch code {
	case fiber.StatusBadRequest:
		return "bad_request"
	case fiber.StatusUnauthorized:
		return "unauthorized"
	case fiber.StatusForbidden:
		return "forbidden"
	case fiber.StatusNotFound:
		return "not_found"
	case fiber.StatusMethodNotAllowed:
		return "method_not_allowed"
	case fiber.StatusRequestTimeout:
		return "timeout"
	case fiber.StatusConflict:
		return "conflict"
	case fiber.StatusRequestEntityTooLarge:
		return "payload_too_large"
	case fiber.StatusTooManyRequests:
		return "too_many_requests"
	case fiber.StatusInternalServerError:
		return "internal_error"
	case fiber.StatusBadGateway:
		return "bad_gateway"
	case fiber.StatusServiceUnavailable:
		return "service_unavailable"
	default:
		return "error"
	}
}
