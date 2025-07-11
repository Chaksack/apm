package auth

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// AuthType represents the type of authentication
type AuthType string

const (
	AuthTypeJWT    AuthType = "jwt"
	AuthTypeAPIKey AuthType = "api_key"
	AuthTypeBearer AuthType = "bearer"
)

// User represents an authenticated user
type User struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
}

// Claims represents JWT claims
type Claims struct {
	jwt.RegisteredClaims
	User      User     `json:"user"`
	Roles     []string `json:"roles"`
	TokenType string   `json:"token_type"`
}

// APIKey represents an API key
type APIKey struct {
	ID         string    `json:"id"`
	Key        string    `json:"key"`
	Name       string    `json:"name"`
	UserID     string    `json:"user_id"`
	Roles      []string  `json:"roles"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at"`
	ExpiresAt  time.Time `json:"expires_at,omitempty"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	JWT       JWTConfig    `yaml:"jwt" json:"jwt"`
	APIKey    APIKeyConfig `yaml:"api_key" json:"api_key"`
	EnableJWT bool         `yaml:"enable_jwt" json:"enable_jwt"`
	EnableAPI bool         `yaml:"enable_api_key" json:"enable_api_key"`
}

// JWTConfig represents JWT configuration
type JWTConfig struct {
	Secret             string        `yaml:"secret" json:"secret"`
	Issuer             string        `yaml:"issuer" json:"issuer"`
	Audience           []string      `yaml:"audience" json:"audience"`
	AccessTokenExpiry  time.Duration `yaml:"access_token_expiry" json:"access_token_expiry"`
	RefreshTokenExpiry time.Duration `yaml:"refresh_token_expiry" json:"refresh_token_expiry"`
}

// APIKeyConfig represents API key configuration
type APIKeyConfig struct {
	HeaderName string            `yaml:"header_name" json:"header_name"`
	QueryParam string            `yaml:"query_param" json:"query_param"`
	Keys       map[string]APIKey `yaml:"keys" json:"keys"`
}

// TokenResponse represents authentication token response
type TokenResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int64     `json:"expires_in"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// AuthContext represents the authentication context in fiber
type AuthContext struct {
	User      *User
	AuthType  AuthType
	Token     string
	Claims    *Claims
	RequestID string
}

// GetAuthContext retrieves auth context from fiber context
func GetAuthContext(c *fiber.Ctx) *AuthContext {
	ctx := c.Locals("auth_context")
	if ctx == nil {
		return nil
	}
	authCtx, ok := ctx.(*AuthContext)
	if !ok {
		return nil
	}
	return authCtx
}

// SetAuthContext sets auth context in fiber context
func SetAuthContext(c *fiber.Ctx, authCtx *AuthContext) {
	c.Locals("auth_context", authCtx)
}
