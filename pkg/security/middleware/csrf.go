package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// CSRFConfig represents CSRF protection configuration
type CSRFConfig struct {
	// Token length
	TokenLength int `yaml:"token_length" json:"token_length"`

	// Token expiration
	TokenExpiration time.Duration `yaml:"token_expiration" json:"token_expiration"`

	// Cookie name for double-submit pattern
	CookieName string `yaml:"cookie_name" json:"cookie_name"`

	// Header name for token
	HeaderName string `yaml:"header_name" json:"header_name"`

	// Form field name for token
	FormFieldName string `yaml:"form_field_name" json:"form_field_name"`

	// Methods to protect
	ProtectedMethods []string `yaml:"protected_methods" json:"protected_methods"`

	// Paths to exclude from CSRF protection
	ExcludePaths []string `yaml:"exclude_paths" json:"exclude_paths"`

	// Cookie settings
	CookieSecure   bool   `yaml:"cookie_secure" json:"cookie_secure"`
	CookieHTTPOnly bool   `yaml:"cookie_httponly" json:"cookie_httponly"`
	CookieSameSite string `yaml:"cookie_samesite" json:"cookie_samesite"`
	CookiePath     string `yaml:"cookie_path" json:"cookie_path"`
}

// DefaultCSRFConfig provides default CSRF configuration
var DefaultCSRFConfig = CSRFConfig{
	TokenLength:      32,
	TokenExpiration:  1 * time.Hour,
	CookieName:       "csrf_token",
	HeaderName:       "X-CSRF-Token",
	FormFieldName:    "csrf_token",
	ProtectedMethods: []string{"POST", "PUT", "PATCH", "DELETE"},
	ExcludePaths:     []string{"/api/auth/login", "/api/auth/register"},
	CookieSecure:     true,
	CookieHTTPOnly:   true,
	CookieSameSite:   "Strict",
	CookiePath:       "/",
}

// csrfToken represents a CSRF token
type csrfToken struct {
	Value     string
	ExpiresAt time.Time
}

// CSRFMiddleware provides CSRF protection
type CSRFMiddleware struct {
	config     CSRFConfig
	tokenStore map[string]*csrfToken
	storeMu    sync.RWMutex
	logger     *zap.Logger
}

// NewCSRFMiddleware creates a new CSRF middleware
func NewCSRFMiddleware(config CSRFConfig, logger *zap.Logger) *CSRFMiddleware {
	// Apply defaults
	if config.TokenLength == 0 {
		config.TokenLength = DefaultCSRFConfig.TokenLength
	}
	if config.TokenExpiration == 0 {
		config.TokenExpiration = DefaultCSRFConfig.TokenExpiration
	}
	if config.CookieName == "" {
		config.CookieName = DefaultCSRFConfig.CookieName
	}
	if config.HeaderName == "" {
		config.HeaderName = DefaultCSRFConfig.HeaderName
	}
	if config.FormFieldName == "" {
		config.FormFieldName = DefaultCSRFConfig.FormFieldName
	}
	if len(config.ProtectedMethods) == 0 {
		config.ProtectedMethods = DefaultCSRFConfig.ProtectedMethods
	}
	if config.CookiePath == "" {
		config.CookiePath = DefaultCSRFConfig.CookiePath
	}

	m := &CSRFMiddleware{
		config:     config,
		tokenStore: make(map[string]*csrfToken),
		logger:     logger,
	}

	// Start cleanup goroutine
	go m.cleanupExpiredTokens()

	return m
}

// Apply returns the CSRF protection middleware handler
func (m *CSRFMiddleware) Apply() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check if path should be excluded
		path := c.Path()
		for _, excludePath := range m.config.ExcludePaths {
			if strings.HasPrefix(path, excludePath) {
				return c.Next()
			}
		}

		// Check if method should be protected
		method := c.Method()
		protected := false
		for _, protectedMethod := range m.config.ProtectedMethods {
			if method == protectedMethod {
				protected = true
				break
			}
		}

		// Generate token for GET requests or if no token exists
		if method == fiber.MethodGet || method == fiber.MethodHead {
			token := m.getOrCreateToken(c)
			c.Locals("csrf_token", token)
			return c.Next()
		}

		// Validate token for protected methods
		if protected {
			if err := m.validateToken(c); err != nil {
				m.logger.Warn("CSRF validation failed",
					zap.String("ip", c.IP()),
					zap.String("path", path),
					zap.Error(err))

				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error":   "csrf_validation_failed",
					"message": "invalid CSRF token",
				})
			}
		}

		return c.Next()
	}
}

// GetToken returns the current CSRF token for the request
func (m *CSRFMiddleware) GetToken(c *fiber.Ctx) string {
	// Check locals first
	if token, ok := c.Locals("csrf_token").(string); ok && token != "" {
		return token
	}

	// Get from cookie
	sessionID := m.getSessionID(c)
	if sessionID == "" {
		return ""
	}

	m.storeMu.RLock()
	token, exists := m.tokenStore[sessionID]
	m.storeMu.RUnlock()

	if exists && time.Now().Before(token.ExpiresAt) {
		return token.Value
	}

	return ""
}

// GenerateToken generates a new CSRF token
func (m *CSRFMiddleware) GenerateToken(c *fiber.Ctx) string {
	token := m.generateRandomToken()
	sessionID := m.getOrCreateSessionID(c)

	m.storeMu.Lock()
	m.tokenStore[sessionID] = &csrfToken{
		Value:     token,
		ExpiresAt: time.Now().Add(m.config.TokenExpiration),
	}
	m.storeMu.Unlock()

	// Set cookie with token
	m.setTokenCookie(c, token)

	return token
}

// validateToken validates the CSRF token
func (m *CSRFMiddleware) validateToken(c *fiber.Ctx) error {
	// Get session token
	sessionID := m.getSessionID(c)
	if sessionID == "" {
		return fmt.Errorf("no session found")
	}

	m.storeMu.RLock()
	storedToken, exists := m.tokenStore[sessionID]
	m.storeMu.RUnlock()

	if !exists {
		return fmt.Errorf("no CSRF token found for session")
	}

	if time.Now().After(storedToken.ExpiresAt) {
		return fmt.Errorf("CSRF token expired")
	}

	// Get submitted token from header or form
	submittedToken := c.Get(m.config.HeaderName)
	if submittedToken == "" {
		submittedToken = c.FormValue(m.config.FormFieldName)
	}

	if submittedToken == "" {
		return fmt.Errorf("no CSRF token provided")
	}

	// Compare tokens using constant-time comparison
	if !m.compareTokens(storedToken.Value, submittedToken) {
		return fmt.Errorf("CSRF token mismatch")
	}

	// For double-submit cookie pattern, also validate cookie
	cookieToken := c.Cookies(m.config.CookieName)
	if cookieToken != "" && !m.compareTokens(storedToken.Value, cookieToken) {
		return fmt.Errorf("CSRF cookie token mismatch")
	}

	return nil
}

// getOrCreateToken gets existing token or creates new one
func (m *CSRFMiddleware) getOrCreateToken(c *fiber.Ctx) string {
	token := m.GetToken(c)
	if token == "" {
		token = m.GenerateToken(c)
	}
	return token
}

// getSessionID gets session ID from auth context or cookie
func (m *CSRFMiddleware) getSessionID(c *fiber.Ctx) string {
	// Try to get from auth context
	if authCtx, ok := c.Locals("auth_context").(interface{ GetSessionID() string }); ok {
		if sessionID := authCtx.GetSessionID(); sessionID != "" {
			return sessionID
		}
	}

	// Fallback to session cookie
	return c.Cookies("session_id")
}

// getOrCreateSessionID gets or creates session ID
func (m *CSRFMiddleware) getOrCreateSessionID(c *fiber.Ctx) string {
	sessionID := m.getSessionID(c)
	if sessionID == "" {
		// Generate new session ID
		sessionID = m.generateRandomToken()

		// Set session cookie
		cookie := &fiber.Cookie{
			Name:     "session_id",
			Value:    sessionID,
			Expires:  time.Now().Add(24 * time.Hour),
			HTTPOnly: true,
			Secure:   m.config.CookieSecure,
			SameSite: m.config.CookieSameSite,
			Path:     m.config.CookiePath,
		}
		c.Cookie(cookie)
	}
	return sessionID
}

// setTokenCookie sets CSRF token cookie
func (m *CSRFMiddleware) setTokenCookie(c *fiber.Ctx, token string) {
	cookie := &fiber.Cookie{
		Name:     m.config.CookieName,
		Value:    token,
		Expires:  time.Now().Add(m.config.TokenExpiration),
		HTTPOnly: m.config.CookieHTTPOnly,
		Secure:   m.config.CookieSecure,
		SameSite: m.config.CookieSameSite,
		Path:     m.config.CookiePath,
	}
	c.Cookie(cookie)
}

// generateRandomToken generates a cryptographically secure random token
func (m *CSRFMiddleware) generateRandomToken() string {
	b := make([]byte, m.config.TokenLength)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based token
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return base64.URLEncoding.EncodeToString(b)
}

// compareTokens compares tokens using constant-time comparison
func (m *CSRFMiddleware) compareTokens(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// cleanupExpiredTokens periodically removes expired tokens
func (m *CSRFMiddleware) cleanupExpiredTokens() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.storeMu.Lock()
		now := time.Now()
		for sessionID, token := range m.tokenStore {
			if now.After(token.ExpiresAt) {
				delete(m.tokenStore, sessionID)
			}
		}
		m.storeMu.Unlock()
	}
}

// CSRFTokenHandler provides an endpoint to get CSRF token
func CSRFTokenHandler(csrf *CSRFMiddleware) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := csrf.GetToken(c)
		if token == "" {
			token = csrf.GenerateToken(c)
		}

		return c.JSON(fiber.Map{
			"csrf_token": token,
		})
	}
}
