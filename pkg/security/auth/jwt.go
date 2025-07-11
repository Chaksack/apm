package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrTokenExpired     = errors.New("token expired")
	ErrInvalidClaims    = errors.New("invalid claims")
	ErrInvalidSignature = errors.New("invalid signature")
)

// JWTManager handles JWT operations
type JWTManager struct {
	config JWTConfig
	logger *zap.Logger
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(config JWTConfig, logger *zap.Logger) *JWTManager {
	if config.Secret == "" {
		// Generate a secure random secret if not provided
		config.Secret = generateSecureSecret()
		logger.Warn("JWT secret not provided, generated random secret")
	}

	// Set defaults
	if config.AccessTokenExpiry == 0 {
		config.AccessTokenExpiry = 15 * time.Minute
	}
	if config.RefreshTokenExpiry == 0 {
		config.RefreshTokenExpiry = 7 * 24 * time.Hour
	}
	if config.Issuer == "" {
		config.Issuer = "apm-system"
	}

	return &JWTManager{
		config: config,
		logger: logger,
	}
}

// GenerateToken generates a new JWT token
func (j *JWTManager) GenerateToken(user *User) (*TokenResponse, error) {
	now := time.Now()
	expiresAt := now.Add(j.config.AccessTokenExpiry)

	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.config.Issuer,
			Subject:   user.ID,
			Audience:  j.config.Audience,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        generateTokenID(),
		},
		User:      *user,
		Roles:     user.Roles,
		TokenType: "access",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(j.config.Secret))
	if err != nil {
		j.logger.Error("failed to sign token", zap.Error(err))
		return nil, fmt.Errorf("failed to sign token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := j.generateRefreshToken(user.ID)
	if err != nil {
		j.logger.Error("failed to generate refresh token", zap.Error(err))
		return nil, err
	}

	return &TokenResponse{
		AccessToken:  tokenString,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(j.config.AccessTokenExpiry.Seconds()),
		ExpiresAt:    expiresAt,
	}, nil
}

// ValidateToken validates a JWT token
func (j *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.config.Secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		if errors.Is(err, jwt.ErrSignatureInvalid) {
			return nil, ErrInvalidSignature
		}
		return nil, ErrInvalidToken
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidClaims
	}

	// Validate issuer
	if claims.Issuer != j.config.Issuer {
		return nil, fmt.Errorf("invalid issuer: %s", claims.Issuer)
	}

	// Validate audience if configured
	if len(j.config.Audience) > 0 {
		valid := false
		// In JWT v5, audience is a slice
		if claims.Audience != nil {
			for _, expectedAud := range j.config.Audience {
				for _, claimAud := range claims.Audience {
					if claimAud == expectedAud {
						valid = true
						break
					}
				}
				if valid {
					break
				}
			}
		}
		if !valid {
			return nil, fmt.Errorf("invalid audience")
		}
	}

	return claims, nil
}

// RefreshToken refreshes an access token using a refresh token
func (j *JWTManager) RefreshToken(refreshToken string) (*TokenResponse, error) {
	// Validate refresh token
	claims, err := j.validateRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	// Get user from claims
	user := &User{
		ID:       claims.Subject,
		Username: claims.User.Username,
		Email:    claims.User.Email,
		Roles:    claims.Roles,
	}

	// Generate new tokens
	return j.GenerateToken(user)
}

// generateRefreshToken generates a refresh token
func (j *JWTManager) generateRefreshToken(userID string) (string, error) {
	now := time.Now()
	expiresAt := now.Add(j.config.RefreshTokenExpiry)

	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.config.Issuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        generateTokenID(),
		},
		TokenType: "refresh",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(j.config.Secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return tokenString, nil
}

// validateRefreshToken validates a refresh token
func (j *JWTManager) validateRefreshToken(tokenString string) (*Claims, error) {
	claims, err := j.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != "refresh" {
		return nil, fmt.Errorf("invalid token type: expected refresh, got %s", claims.TokenType)
	}

	return claims, nil
}

// generateSecureSecret generates a secure random secret
func generateSecureSecret() string {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		panic(fmt.Sprintf("failed to generate secure secret: %v", err))
	}
	return base64.StdEncoding.EncodeToString(b)
}

// generateTokenID generates a unique token ID
func generateTokenID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return fmt.Sprintf("apm-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("apm-%s", base64.RawURLEncoding.EncodeToString(b))
}
