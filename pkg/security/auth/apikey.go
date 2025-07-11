package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

var (
	ErrAPIKeyNotFound = errors.New("API key not found")
	ErrAPIKeyExpired  = errors.New("API key expired")
	ErrAPIKeyInvalid  = errors.New("API key invalid")
)

// APIKeyManager handles API key operations
type APIKeyManager struct {
	config   APIKeyConfig
	logger   *zap.Logger
	keys     map[string]*APIKey // hashed key -> APIKey
	keysByID map[string]*APIKey // ID -> APIKey
	mu       sync.RWMutex
}

// NewAPIKeyManager creates a new API key manager
func NewAPIKeyManager(config APIKeyConfig, logger *zap.Logger) *APIKeyManager {
	// Set defaults
	if config.HeaderName == "" {
		config.HeaderName = "X-API-Key"
	}
	if config.QueryParam == "" {
		config.QueryParam = "api_key"
	}

	manager := &APIKeyManager{
		config:   config,
		logger:   logger,
		keys:     make(map[string]*APIKey),
		keysByID: make(map[string]*APIKey),
	}

	// Load configured keys
	for _, key := range config.Keys {
		keyCopy := key
		hashedKey := manager.hashKey(keyCopy.Key)
		manager.keys[hashedKey] = &keyCopy
		manager.keysByID[keyCopy.ID] = &keyCopy
	}

	return manager
}

// GenerateAPIKey generates a new API key
func (m *APIKeyManager) GenerateAPIKey(name string, userID string, roles []string, expiresIn time.Duration) (*APIKey, string, error) {
	// Generate secure random key
	rawKey := generateAPIKey()
	hashedKey := m.hashKey(rawKey)

	now := time.Now()
	apiKey := &APIKey{
		ID:         generateAPIKeyID(),
		Key:        hashedKey, // Store hashed version
		Name:       name,
		UserID:     userID,
		Roles:      roles,
		CreatedAt:  now,
		LastUsedAt: now,
	}

	if expiresIn > 0 {
		apiKey.ExpiresAt = now.Add(expiresIn)
	}

	m.mu.Lock()
	m.keys[hashedKey] = apiKey
	m.keysByID[apiKey.ID] = apiKey
	m.mu.Unlock()

	m.logger.Info("generated new API key",
		zap.String("id", apiKey.ID),
		zap.String("name", name),
		zap.String("user_id", userID),
		zap.Strings("roles", roles))

	// Return the raw key to the user (only shown once)
	return apiKey, rawKey, nil
}

// ValidateAPIKey validates an API key
func (m *APIKeyManager) ValidateAPIKey(rawKey string) (*APIKey, error) {
	hashedKey := m.hashKey(rawKey)

	m.mu.RLock()
	apiKey, exists := m.keys[hashedKey]
	m.mu.RUnlock()

	if !exists {
		m.logger.Debug("API key not found")
		return nil, ErrAPIKeyNotFound
	}

	// Check expiration
	if !apiKey.ExpiresAt.IsZero() && time.Now().After(apiKey.ExpiresAt) {
		m.logger.Debug("API key expired",
			zap.String("id", apiKey.ID),
			zap.Time("expired_at", apiKey.ExpiresAt))
		return nil, ErrAPIKeyExpired
	}

	// Update last used timestamp
	m.mu.Lock()
	apiKey.LastUsedAt = time.Now()
	m.mu.Unlock()

	return apiKey, nil
}

// RevokeAPIKey revokes an API key by ID
func (m *APIKeyManager) RevokeAPIKey(keyID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	apiKey, exists := m.keysByID[keyID]
	if !exists {
		return ErrAPIKeyNotFound
	}

	delete(m.keys, apiKey.Key)
	delete(m.keysByID, keyID)

	m.logger.Info("revoked API key",
		zap.String("id", keyID),
		zap.String("name", apiKey.Name))

	return nil
}

// ListAPIKeys lists all API keys for a user
func (m *APIKeyManager) ListAPIKeys(userID string) []*APIKey {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var keys []*APIKey
	for _, key := range m.keysByID {
		if key.UserID == userID {
			keyCopy := *key
			keyCopy.Key = "" // Don't expose hashed key
			keys = append(keys, &keyCopy)
		}
	}

	return keys
}

// GetAPIKey gets an API key by ID
func (m *APIKeyManager) GetAPIKey(keyID string) (*APIKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	apiKey, exists := m.keysByID[keyID]
	if !exists {
		return nil, ErrAPIKeyNotFound
	}

	keyCopy := *apiKey
	keyCopy.Key = "" // Don't expose hashed key
	return &keyCopy, nil
}

// hashKey hashes an API key using SHA-256
func (m *APIKeyManager) hashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// generateAPIKey generates a secure API key
func generateAPIKey() string {
	// Generate 32 bytes of random data
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		panic(fmt.Sprintf("failed to generate API key: %v", err))
	}

	// Encode as hex with prefix
	return fmt.Sprintf("apm_%s", hex.EncodeToString(b))
}

// generateAPIKeyID generates a unique API key ID
func generateAPIKeyID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return fmt.Sprintf("key_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("key_%s", hex.EncodeToString(b))
}

// ExtractAPIKey extracts API key from request
func ExtractAPIKey(headers map[string]string, query map[string]string, config APIKeyConfig) string {
	// Check header first
	if config.HeaderName != "" {
		if key, ok := headers[config.HeaderName]; ok && key != "" {
			return key
		}
		// Check with lowercase
		if key, ok := headers[strings.ToLower(config.HeaderName)]; ok && key != "" {
			return key
		}
	}

	// Check query parameter
	if config.QueryParam != "" {
		if key, ok := query[config.QueryParam]; ok && key != "" {
			return key
		}
	}

	return ""
}
