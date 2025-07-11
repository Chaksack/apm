package middleware

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// SecurityHeadersConfig represents security headers configuration
type SecurityHeadersConfig struct {
	// XFrameOptions controls clickjacking protection
	XFrameOptions string `yaml:"x_frame_options" json:"x_frame_options"`

	// XContentTypeOptions prevents MIME type sniffing
	XContentTypeOptions string `yaml:"x_content_type_options" json:"x_content_type_options"`

	// XSSProtection enables XSS filtering
	XSSProtection string `yaml:"x_xss_protection" json:"x_xss_protection"`

	// StrictTransportSecurity enforces HTTPS
	StrictTransportSecurity string `yaml:"strict_transport_security" json:"strict_transport_security"`

	// ContentSecurityPolicy defines allowed sources
	ContentSecurityPolicy string `yaml:"content_security_policy" json:"content_security_policy"`

	// ReferrerPolicy controls referrer information
	ReferrerPolicy string `yaml:"referrer_policy" json:"referrer_policy"`

	// PermissionsPolicy controls browser features
	PermissionsPolicy string `yaml:"permissions_policy" json:"permissions_policy"`

	// CrossOriginOpenerPolicy controls window.opener access
	CrossOriginOpenerPolicy string `yaml:"cross_origin_opener_policy" json:"cross_origin_opener_policy"`

	// CrossOriginResourcePolicy controls resource loading
	CrossOriginResourcePolicy string `yaml:"cross_origin_resource_policy" json:"cross_origin_resource_policy"`

	// CrossOriginEmbedderPolicy controls embedding
	CrossOriginEmbedderPolicy string `yaml:"cross_origin_embedder_policy" json:"cross_origin_embedder_policy"`
}

// DefaultSecurityHeadersConfig provides secure defaults
var DefaultSecurityHeadersConfig = SecurityHeadersConfig{
	XFrameOptions:             "DENY",
	XContentTypeOptions:       "nosniff",
	XSSProtection:             "1; mode=block",
	StrictTransportSecurity:   "max-age=31536000; includeSubDomains",
	ContentSecurityPolicy:     "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self'; frame-ancestors 'none';",
	ReferrerPolicy:            "strict-origin-when-cross-origin",
	PermissionsPolicy:         "geolocation=(), microphone=(), camera=()",
	CrossOriginOpenerPolicy:   "same-origin",
	CrossOriginResourcePolicy: "same-origin",
	CrossOriginEmbedderPolicy: "require-corp",
}

// SecurityHeadersMiddleware provides security headers middleware
type SecurityHeadersMiddleware struct {
	config SecurityHeadersConfig
	logger *zap.Logger
}

// NewSecurityHeadersMiddleware creates a new security headers middleware
func NewSecurityHeadersMiddleware(config SecurityHeadersConfig, logger *zap.Logger) *SecurityHeadersMiddleware {
	// Apply defaults for empty values
	if config.XFrameOptions == "" {
		config.XFrameOptions = DefaultSecurityHeadersConfig.XFrameOptions
	}
	if config.XContentTypeOptions == "" {
		config.XContentTypeOptions = DefaultSecurityHeadersConfig.XContentTypeOptions
	}
	if config.XSSProtection == "" {
		config.XSSProtection = DefaultSecurityHeadersConfig.XSSProtection
	}
	if config.StrictTransportSecurity == "" {
		config.StrictTransportSecurity = DefaultSecurityHeadersConfig.StrictTransportSecurity
	}
	if config.ContentSecurityPolicy == "" {
		config.ContentSecurityPolicy = DefaultSecurityHeadersConfig.ContentSecurityPolicy
	}
	if config.ReferrerPolicy == "" {
		config.ReferrerPolicy = DefaultSecurityHeadersConfig.ReferrerPolicy
	}
	if config.PermissionsPolicy == "" {
		config.PermissionsPolicy = DefaultSecurityHeadersConfig.PermissionsPolicy
	}

	return &SecurityHeadersMiddleware{
		config: config,
		logger: logger,
	}
}

// Apply returns the security headers middleware handler
func (m *SecurityHeadersMiddleware) Apply() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Apply security headers
		c.Set("X-Frame-Options", m.config.XFrameOptions)
		c.Set("X-Content-Type-Options", m.config.XContentTypeOptions)
		c.Set("X-XSS-Protection", m.config.XSSProtection)

		// Only set HSTS for HTTPS connections
		if c.Protocol() == "https" {
			c.Set("Strict-Transport-Security", m.config.StrictTransportSecurity)
		}

		// Content Security Policy
		if m.config.ContentSecurityPolicy != "" {
			c.Set("Content-Security-Policy", m.config.ContentSecurityPolicy)
		}

		// Referrer Policy
		if m.config.ReferrerPolicy != "" {
			c.Set("Referrer-Policy", m.config.ReferrerPolicy)
		}

		// Permissions Policy (formerly Feature Policy)
		if m.config.PermissionsPolicy != "" {
			c.Set("Permissions-Policy", m.config.PermissionsPolicy)
		}

		// Cross-Origin policies
		if m.config.CrossOriginOpenerPolicy != "" {
			c.Set("Cross-Origin-Opener-Policy", m.config.CrossOriginOpenerPolicy)
		}
		if m.config.CrossOriginResourcePolicy != "" {
			c.Set("Cross-Origin-Resource-Policy", m.config.CrossOriginResourcePolicy)
		}
		if m.config.CrossOriginEmbedderPolicy != "" {
			c.Set("Cross-Origin-Embedder-Policy", m.config.CrossOriginEmbedderPolicy)
		}

		// Remove potentially dangerous headers
		c.Response().Header.Del("X-Powered-By")
		c.Response().Header.Del("Server")

		return c.Next()
	}
}

// CORSConfig represents CORS configuration
type CORSConfig struct {
	AllowOrigins     []string `yaml:"allow_origins" json:"allow_origins"`
	AllowMethods     []string `yaml:"allow_methods" json:"allow_methods"`
	AllowHeaders     []string `yaml:"allow_headers" json:"allow_headers"`
	ExposeHeaders    []string `yaml:"expose_headers" json:"expose_headers"`
	AllowCredentials bool     `yaml:"allow_credentials" json:"allow_credentials"`
	MaxAge           int      `yaml:"max_age" json:"max_age"`
}

// DefaultCORSConfig provides secure CORS defaults
var DefaultCORSConfig = CORSConfig{
	AllowOrigins:     []string{"https://localhost:3000"},
	AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
	ExposeHeaders:    []string{"X-Request-ID"},
	AllowCredentials: true,
	MaxAge:           86400, // 24 hours
}

// CORSMiddleware provides CORS middleware
type CORSMiddleware struct {
	config CORSConfig
	logger *zap.Logger
}

// NewCORSMiddleware creates a new CORS middleware
func NewCORSMiddleware(config CORSConfig, logger *zap.Logger) *CORSMiddleware {
	// Apply defaults for empty values
	if len(config.AllowOrigins) == 0 {
		config.AllowOrigins = DefaultCORSConfig.AllowOrigins
	}
	if len(config.AllowMethods) == 0 {
		config.AllowMethods = DefaultCORSConfig.AllowMethods
	}
	if len(config.AllowHeaders) == 0 {
		config.AllowHeaders = DefaultCORSConfig.AllowHeaders
	}
	if config.MaxAge == 0 {
		config.MaxAge = DefaultCORSConfig.MaxAge
	}

	return &CORSMiddleware{
		config: config,
		logger: logger,
	}
}

// Apply returns the CORS middleware handler
func (m *CORSMiddleware) Apply() fiber.Handler {
	return func(c *fiber.Ctx) error {
		origin := c.Get("Origin")

		// Check if origin is allowed
		allowedOrigin := m.isOriginAllowed(origin)
		if allowedOrigin != "" {
			c.Set("Access-Control-Allow-Origin", allowedOrigin)
		}

		// Set other CORS headers
		c.Set("Access-Control-Allow-Methods", strings.Join(m.config.AllowMethods, ", "))
		c.Set("Access-Control-Allow-Headers", strings.Join(m.config.AllowHeaders, ", "))

		if len(m.config.ExposeHeaders) > 0 {
			c.Set("Access-Control-Expose-Headers", strings.Join(m.config.ExposeHeaders, ", "))
		}

		if m.config.AllowCredentials {
			c.Set("Access-Control-Allow-Credentials", "true")
		}

		// Handle preflight requests
		if c.Method() == fiber.MethodOptions {
			c.Set("Access-Control-Max-Age", fmt.Sprintf("%d", m.config.MaxAge))
			return c.SendStatus(fiber.StatusNoContent)
		}

		return c.Next()
	}
}

// isOriginAllowed checks if origin is allowed
func (m *CORSMiddleware) isOriginAllowed(origin string) string {
	if origin == "" {
		return ""
	}

	for _, allowed := range m.config.AllowOrigins {
		// Exact match
		if allowed == origin {
			return origin
		}

		// Wildcard match
		if allowed == "*" {
			// For security, don't use wildcard with credentials
			if m.config.AllowCredentials {
				m.logger.Warn("wildcard origin with credentials is insecure")
				continue
			}
			return "*"
		}

		// Subdomain wildcard match (e.g., https://*.example.com)
		if strings.HasPrefix(allowed, "https://*.") || strings.HasPrefix(allowed, "http://*.") {
			domain := strings.TrimPrefix(strings.TrimPrefix(allowed, "https://*."), "http://*.")
			if strings.HasSuffix(origin, domain) {
				return origin
			}
		}
	}

	return ""
}
