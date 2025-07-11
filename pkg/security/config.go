package security

import (
	"github.com/yourusername/apm/pkg/security/auth"
	"github.com/yourusername/apm/pkg/security/middleware"
)

// Config represents the complete security configuration
type Config struct {
	// Authentication configuration
	Auth auth.AuthConfig `yaml:"auth" json:"auth"`

	// RBAC configuration
	RBAC auth.RBACConfig `yaml:"rbac" json:"rbac"`

	// Security headers configuration
	Headers middleware.SecurityHeadersConfig `yaml:"headers" json:"headers"`

	// CORS configuration
	CORS middleware.CORSConfig `yaml:"cors" json:"cors"`

	// Rate limiting configuration
	RateLimit middleware.RateLimitConfig `yaml:"rate_limit" json:"rate_limit"`

	// Audit logging configuration
	Audit middleware.AuditConfig `yaml:"audit" json:"audit"`

	// CSRF protection configuration
	CSRF middleware.CSRFConfig `yaml:"csrf" json:"csrf"`

	// API security configuration
	APISecurity middleware.APISecurityConfig `yaml:"api_security" json:"api_security"`
}

// DefaultConfig returns a secure default configuration
func DefaultConfig() Config {
	return Config{
		Auth: auth.AuthConfig{
			EnableJWT: true,
			EnableAPI: true,
			JWT: auth.JWTConfig{
				Issuer:   "apm-system",
				Audience: []string{"apm-api"},
			},
			APIKey: auth.APIKeyConfig{
				HeaderName: "X-API-Key",
				QueryParam: "api_key",
			},
		},
		RBAC: auth.RBACConfig{
			Roles:       auth.DefaultRoles,
			DefaultRole: "viewer",
		},
		Headers:     middleware.DefaultSecurityHeadersConfig,
		CORS:        middleware.DefaultCORSConfig,
		RateLimit:   middleware.DefaultRateLimitConfig,
		Audit:       middleware.DefaultAuditConfig,
		CSRF:        middleware.DefaultCSRFConfig,
		APISecurity: middleware.DefaultAPISecurityConfig,
	}
}
