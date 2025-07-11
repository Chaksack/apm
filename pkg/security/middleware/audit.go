package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/yourusername/apm/pkg/security/auth"
)

// AuditEvent represents a security audit event
type AuditEvent struct {
	ID           string                 `json:"id"`
	Timestamp    time.Time              `json:"timestamp"`
	EventType    string                 `json:"event_type"`
	Severity     string                 `json:"severity"`
	UserID       string                 `json:"user_id,omitempty"`
	Username     string                 `json:"username,omitempty"`
	IP           string                 `json:"ip"`
	UserAgent    string                 `json:"user_agent"`
	Method       string                 `json:"method"`
	Path         string                 `json:"path"`
	Query        string                 `json:"query,omitempty"`
	StatusCode   int                    `json:"status_code"`
	ResponseTime int64                  `json:"response_time_ms"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
	RequestID    string                 `json:"request_id"`
}

// EventType constants
const (
	EventTypeAuthSuccess     = "auth_success"
	EventTypeAuthFailure     = "auth_failure"
	EventTypeAuthzFailure    = "authorization_failure"
	EventTypeTokenRefresh    = "token_refresh"
	EventTypeAPIKeyUsed      = "api_key_used"
	EventTypeRateLimitHit    = "rate_limit_hit"
	EventTypeSuspiciousInput = "suspicious_input"
	EventTypeConfigChange    = "config_change"
	EventTypeDeployment      = "deployment"
	EventTypeDataAccess      = "data_access"
	EventTypeError           = "error"
)

// Severity levels
const (
	SeverityInfo     = "info"
	SeverityWarning  = "warning"
	SeverityCritical = "critical"
)

// AuditLogger interface for different audit backends
type AuditLogger interface {
	Log(event *AuditEvent) error
}

// ZapAuditLogger implements AuditLogger using zap
type ZapAuditLogger struct {
	logger *zap.Logger
}

// NewZapAuditLogger creates a new zap audit logger
func NewZapAuditLogger(logger *zap.Logger) *ZapAuditLogger {
	return &ZapAuditLogger{
		logger: logger.Named("audit"),
	}
}

// Log logs an audit event
func (l *ZapAuditLogger) Log(event *AuditEvent) error {
	fields := []zap.Field{
		zap.String("event_id", event.ID),
		zap.Time("timestamp", event.Timestamp),
		zap.String("event_type", event.EventType),
		zap.String("severity", event.Severity),
		zap.String("ip", event.IP),
		zap.String("method", event.Method),
		zap.String("path", event.Path),
		zap.Int("status_code", event.StatusCode),
		zap.Int64("response_time_ms", event.ResponseTime),
		zap.String("request_id", event.RequestID),
	}

	if event.UserID != "" {
		fields = append(fields, zap.String("user_id", event.UserID))
	}
	if event.Username != "" {
		fields = append(fields, zap.String("username", event.Username))
	}
	if event.Query != "" {
		fields = append(fields, zap.String("query", event.Query))
	}
	if event.ErrorMessage != "" {
		fields = append(fields, zap.String("error_message", event.ErrorMessage))
	}
	if event.Details != nil {
		fields = append(fields, zap.Any("details", event.Details))
	}

	switch event.Severity {
	case SeverityCritical:
		l.logger.Error("security audit event", fields...)
	case SeverityWarning:
		l.logger.Warn("security audit event", fields...)
	default:
		l.logger.Info("security audit event", fields...)
	}

	return nil
}

// AuditConfig represents audit configuration
type AuditConfig struct {
	EnableAudit       bool     `yaml:"enable_audit" json:"enable_audit"`
	LogAuthEvents     bool     `yaml:"log_auth_events" json:"log_auth_events"`
	LogAuthzFailures  bool     `yaml:"log_authz_failures" json:"log_authz_failures"`
	LogDataAccess     bool     `yaml:"log_data_access" json:"log_data_access"`
	LogErrors         bool     `yaml:"log_errors" json:"log_errors"`
	LogSensitivePaths []string `yaml:"log_sensitive_paths" json:"log_sensitive_paths"`
	ExcludePaths      []string `yaml:"exclude_paths" json:"exclude_paths"`
	MaxBodySize       int      `yaml:"max_body_size" json:"max_body_size"`
}

// DefaultAuditConfig provides default audit configuration
var DefaultAuditConfig = AuditConfig{
	EnableAudit:      true,
	LogAuthEvents:    true,
	LogAuthzFailures: true,
	LogDataAccess:    true,
	LogErrors:        true,
	LogSensitivePaths: []string{
		"/api/auth",
		"/api/users",
		"/api/deployments",
		"/api/configurations",
		"/api/api-keys",
	},
	ExcludePaths: []string{
		"/health",
		"/metrics",
		"/favicon.ico",
	},
	MaxBodySize: 1024, // 1KB
}

// AuditMiddleware provides security audit logging
type AuditMiddleware struct {
	config      AuditConfig
	auditLogger AuditLogger
	logger      *zap.Logger
}

// NewAuditMiddleware creates a new audit middleware
func NewAuditMiddleware(config AuditConfig, logger *zap.Logger) *AuditMiddleware {
	// Apply defaults
	if config.MaxBodySize == 0 {
		config.MaxBodySize = DefaultAuditConfig.MaxBodySize
	}

	return &AuditMiddleware{
		config:      config,
		auditLogger: NewZapAuditLogger(logger),
		logger:      logger,
	}
}

// Apply returns the audit logging middleware handler
func (m *AuditMiddleware) Apply() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !m.config.EnableAudit {
			return c.Next()
		}

		// Check if path should be excluded
		path := c.Path()
		for _, excludePath := range m.config.ExcludePaths {
			if strings.HasPrefix(path, excludePath) {
				return c.Next()
			}
		}

		// Start timing
		start := time.Now()

		// Get request ID
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = c.Locals("request_id").(string)
		}

		// Execute request
		err := c.Next()

		// Calculate response time
		responseTime := time.Since(start).Milliseconds()

		// Create base audit event
		event := &AuditEvent{
			ID:           fmt.Sprintf("audit_%d", time.Now().UnixNano()),
			Timestamp:    time.Now(),
			IP:           c.IP(),
			UserAgent:    c.Get("User-Agent"),
			Method:       c.Method(),
			Path:         path,
			Query:        string(c.Request().URI().QueryString()),
			StatusCode:   c.Response().StatusCode(),
			ResponseTime: responseTime,
			RequestID:    requestID,
			Details:      make(map[string]interface{}),
		}

		// Add auth context if available
		if authCtx := auth.GetAuthContext(c); authCtx != nil {
			event.UserID = authCtx.User.ID
			event.Username = authCtx.User.Username
			event.Details["auth_type"] = string(authCtx.AuthType)
			event.Details["roles"] = authCtx.User.Roles
		}

		// Determine event type and severity
		m.categorizeEvent(c, event, err)

		// Add request body for sensitive operations if configured
		if m.shouldLogBody(event) {
			m.addRequestBody(c, event)
		}

		// Log the event
		if m.shouldLogEvent(event) {
			if logErr := m.auditLogger.Log(event); logErr != nil {
				m.logger.Error("failed to log audit event", zap.Error(logErr))
			}
		}

		return err
	}
}

// LogAuthEvent logs authentication events
func (m *AuditMiddleware) LogAuthEvent(eventType string, userID string, username string, success bool, details map[string]interface{}) {
	if !m.config.EnableAudit || !m.config.LogAuthEvents {
		return
	}

	severity := SeverityInfo
	if !success {
		severity = SeverityWarning
	}

	event := &AuditEvent{
		ID:        fmt.Sprintf("audit_%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		EventType: eventType,
		Severity:  severity,
		UserID:    userID,
		Username:  username,
		Details:   details,
	}

	if err := m.auditLogger.Log(event); err != nil {
		m.logger.Error("failed to log auth event", zap.Error(err))
	}
}

// LogConfigChange logs configuration changes
func (m *AuditMiddleware) LogConfigChange(userID string, resource string, action string, details map[string]interface{}) {
	if !m.config.EnableAudit {
		return
	}

	event := &AuditEvent{
		ID:        fmt.Sprintf("audit_%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		EventType: EventTypeConfigChange,
		Severity:  SeverityInfo,
		UserID:    userID,
		Details: map[string]interface{}{
			"resource": resource,
			"action":   action,
			"details":  details,
		},
	}

	if err := m.auditLogger.Log(event); err != nil {
		m.logger.Error("failed to log config change", zap.Error(err))
	}
}

// categorizeEvent determines event type and severity
func (m *AuditMiddleware) categorizeEvent(c *fiber.Ctx, event *AuditEvent, err error) {
	statusCode := c.Response().StatusCode()

	// Authentication failures
	if statusCode == fiber.StatusUnauthorized {
		event.EventType = EventTypeAuthFailure
		event.Severity = SeverityWarning
		event.ErrorMessage = "authentication failed"
		return
	}

	// Authorization failures
	if statusCode == fiber.StatusForbidden {
		event.EventType = EventTypeAuthzFailure
		event.Severity = SeverityWarning
		event.ErrorMessage = "authorization failed"
		return
	}

	// Rate limit hits
	if statusCode == fiber.StatusTooManyRequests {
		event.EventType = EventTypeRateLimitHit
		event.Severity = SeverityWarning
		event.ErrorMessage = "rate limit exceeded"
		return
	}

	// Errors
	if statusCode >= 400 || err != nil {
		event.EventType = EventTypeError
		event.Severity = SeverityWarning
		if statusCode >= 500 {
			event.Severity = SeverityCritical
		}
		if err != nil {
			event.ErrorMessage = err.Error()
		}
		return
	}

	// Check for sensitive paths
	for _, sensitivePath := range m.config.LogSensitivePaths {
		if strings.HasPrefix(event.Path, sensitivePath) {
			event.EventType = EventTypeDataAccess
			event.Severity = SeverityInfo
			return
		}
	}

	// Deployment events
	if strings.Contains(event.Path, "/deploy") {
		event.EventType = EventTypeDeployment
		event.Severity = SeverityInfo
		return
	}

	// Default
	event.EventType = EventTypeDataAccess
	event.Severity = SeverityInfo
}

// shouldLogEvent determines if event should be logged
func (m *AuditMiddleware) shouldLogEvent(event *AuditEvent) bool {
	switch event.EventType {
	case EventTypeAuthFailure, EventTypeAuthSuccess:
		return m.config.LogAuthEvents
	case EventTypeAuthzFailure:
		return m.config.LogAuthzFailures
	case EventTypeError:
		return m.config.LogErrors
	case EventTypeDataAccess:
		return m.config.LogDataAccess
	default:
		return true
	}
}

// shouldLogBody determines if request body should be logged
func (m *AuditMiddleware) shouldLogBody(event *AuditEvent) bool {
	// Only log body for certain event types and methods
	if event.Method != fiber.MethodPost && event.Method != fiber.MethodPut {
		return false
	}

	// Check sensitive paths
	for _, sensitivePath := range m.config.LogSensitivePaths {
		if strings.HasPrefix(event.Path, sensitivePath) {
			return true
		}
	}

	return false
}

// addRequestBody adds request body to event
func (m *AuditMiddleware) addRequestBody(c *fiber.Ctx, event *AuditEvent) {
	body := c.Body()
	if len(body) == 0 {
		return
	}

	// Limit body size
	if len(body) > m.config.MaxBodySize {
		event.Details["body_truncated"] = true
		body = body[:m.config.MaxBodySize]
	}

	// Try to parse as JSON
	var jsonBody interface{}
	if err := json.Unmarshal(body, &jsonBody); err == nil {
		// Mask sensitive fields
		if mapBody, ok := jsonBody.(map[string]interface{}); ok {
			m.maskSensitiveFields(mapBody)
			event.Details["request_body"] = mapBody
		} else {
			event.Details["request_body"] = jsonBody
		}
	} else {
		// Store as string if not JSON
		event.Details["request_body"] = string(body)
	}
}

// maskSensitiveFields masks sensitive fields in request body
func (m *AuditMiddleware) maskSensitiveFields(data map[string]interface{}) {
	sensitiveFields := []string{
		"password", "secret", "token", "api_key", "private_key",
		"credit_card", "ssn", "pin", "cvv",
	}

	for key, value := range data {
		// Check if field is sensitive
		lowerKey := strings.ToLower(key)
		for _, sensitive := range sensitiveFields {
			if strings.Contains(lowerKey, sensitive) {
				data[key] = "***MASKED***"
				break
			}
		}

		// Recursively mask nested objects
		if nestedMap, ok := value.(map[string]interface{}); ok {
			m.maskSensitiveFields(nestedMap)
		}
	}
}

// CaptureRequestBody captures request body for audit logging
func CaptureRequestBody() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Only capture for non-GET requests
		if c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead {
			// Read body
			body := c.Body()
			if len(body) > 0 {
				// Store original body
				c.Locals("original_body", bytes.NewBuffer(body))

				// Restore body for further processing
				c.Request().SetBody(body)
			}
		}

		return c.Next()
	}
}
