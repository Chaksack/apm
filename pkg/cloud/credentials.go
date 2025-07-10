package cloud

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

// SecureCredentialManager manages credentials securely
type SecureCredentialManager struct {
	storePath string
	key       []byte
	mu        sync.RWMutex
}

// NewSecureCredentialManager creates a new secure credential manager
func NewSecureCredentialManager(storePath string) (*SecureCredentialManager, error) {
	// Create store directory if it doesn't exist
	if err := os.MkdirAll(storePath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create credential store: %w", err)
	}

	// Derive encryption key from machine-specific data
	key := deriveEncryptionKey()

	return &SecureCredentialManager{
		storePath: storePath,
		key:       key,
	}, nil
}

// Store securely stores credentials
func (m *SecureCredentialManager) Store(creds *Credentials) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate credentials
	if err := validateCredentials(creds); err != nil {
		return err
	}

	// Serialize credentials
	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	// Encrypt data
	encrypted, err := m.encrypt(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	// Generate filename
	filename := m.getCredentialFile(creds.Provider, creds.Profile)

	// Write to file with restricted permissions
	if err := os.WriteFile(filename, encrypted, 0600); err != nil {
		return fmt.Errorf("failed to write credentials: %w", err)
	}

	return nil
}

// Retrieve retrieves stored credentials
func (m *SecureCredentialManager) Retrieve(provider Provider, profile string) (*Credentials, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	filename := m.getCredentialFile(provider, profile)

	// Read encrypted data
	encrypted, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("credentials not found for %s/%s", provider, profile)
		}
		return nil, fmt.Errorf("failed to read credentials: %w", err)
	}

	// Decrypt data
	decrypted, err := m.decrypt(encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	// Deserialize credentials
	var creds Credentials
	if err := json.Unmarshal(decrypted, &creds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	// Check expiry
	if creds.Expiry != nil && creds.Expiry.Before(time.Now()) {
		return nil, fmt.Errorf("credentials expired")
	}

	return &creds, nil
}

// Delete removes stored credentials
func (m *SecureCredentialManager) Delete(provider Provider, profile string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	filename := m.getCredentialFile(provider, profile)

	if err := os.Remove(filename); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to delete credentials: %w", err)
	}

	return nil
}

// List lists all stored credentials for a provider
func (m *SecureCredentialManager) List(provider Provider) ([]*Credentials, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pattern := filepath.Join(m.storePath, fmt.Sprintf("%s-*.cred", provider))
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list credentials: %w", err)
	}

	var credentials []*Credentials
	for _, file := range files {
		// Extract profile from filename
		base := filepath.Base(file)
		profile := extractProfile(base, provider)

		creds, err := m.Retrieve(provider, profile)
		if err != nil {
			continue // Skip invalid or expired credentials
		}

		credentials = append(credentials, creds)
	}

	return credentials, nil
}

// Rotate rotates credentials (provider-specific implementation needed)
func (m *SecureCredentialManager) Rotate(creds *Credentials) (*Credentials, error) {
	// This would require provider-specific implementation
	// For now, return an error
	return nil, fmt.Errorf("credential rotation not implemented for %s", creds.Provider)
}

// encrypt encrypts data using AES-GCM
func (m *SecureCredentialManager) encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(m.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decrypt decrypts data using AES-GCM
func (m *SecureCredentialManager) decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(m.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// getCredentialFile returns the filename for storing credentials
func (m *SecureCredentialManager) getCredentialFile(provider Provider, profile string) string {
	if profile == "" {
		profile = "default"
	}
	filename := fmt.Sprintf("%s-%s.cred", provider, profile)
	return filepath.Join(m.storePath, filename)
}

// deriveEncryptionKey derives an encryption key from machine-specific data
func deriveEncryptionKey() []byte {
	// Combine multiple sources for key derivation
	hostname, _ := os.Hostname()
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME")
	}

	// Use machine ID on Linux
	machineID := ""
	if runtime.GOOS == "linux" {
		if data, err := os.ReadFile("/etc/machine-id"); err == nil {
			machineID = string(data)
		}
	}

	// Combine all sources
	combined := fmt.Sprintf("%s-%s-%s-%s", hostname, username, machineID, runtime.GOOS)

	// Use PBKDF2 to derive key
	salt := []byte("apm-cloud-credentials")
	return pbkdf2.Key([]byte(combined), salt, 10000, 32, sha256.New)
}

// validateCredentials validates credential fields
func validateCredentials(creds *Credentials) error {
	if creds.Provider == "" {
		return fmt.Errorf("provider is required")
	}

	switch creds.AuthMethod {
	case AuthMethodAccessKey:
		if creds.AccessKey == "" || creds.SecretKey == "" {
			return fmt.Errorf("access key and secret key are required for access key auth")
		}
	case AuthMethodServiceKey:
		if creds.Token == "" {
			return fmt.Errorf("service key is required for service key auth")
		}
	case AuthMethodCLI, AuthMethodSDK, AuthMethodIAMRole:
		// These methods don't require stored credentials
	default:
		return fmt.Errorf("unsupported auth method: %s", creds.AuthMethod)
	}

	return nil
}

// extractProfile extracts profile name from filename
func extractProfile(filename string, provider Provider) string {
	prefix := fmt.Sprintf("%s-", provider)
	suffix := ".cred"

	if len(filename) > len(prefix)+len(suffix) {
		start := len(prefix)
		end := len(filename) - len(suffix)
		return filename[start:end]
	}

	return "default"
}

// CredentialHelper provides helper functions for credential management
type CredentialHelper struct {
	manager CredentialManager
}

// NewCredentialHelper creates a new credential helper
func NewCredentialHelper(manager CredentialManager) *CredentialHelper {
	return &CredentialHelper{
		manager: manager,
	}
}

// GetActiveCredentials gets the active credentials for a provider
func (h *CredentialHelper) GetActiveCredentials(ctx context.Context, provider Provider) (*Credentials, error) {
	// First, try environment variables
	if creds := h.getEnvCredentials(provider); creds != nil {
		return creds, nil
	}

	// Then, try CLI credentials
	if creds := h.getCLICredentials(provider); creds != nil {
		return creds, nil
	}

	// Finally, try stored credentials
	return h.manager.Retrieve(provider, "default")
}

// getEnvCredentials gets credentials from environment variables
func (h *CredentialHelper) getEnvCredentials(provider Provider) *Credentials {
	switch provider {
	case ProviderAWS:
		accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
		secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
		if accessKey != "" && secretKey != "" {
			return &Credentials{
				Provider:   provider,
				AuthMethod: AuthMethodAccessKey,
				AccessKey:  accessKey,
				SecretKey:  secretKey,
				Token:      os.Getenv("AWS_SESSION_TOKEN"),
				Region:     os.Getenv("AWS_REGION"),
			}
		}
	case ProviderAzure:
		clientID := os.Getenv("AZURE_CLIENT_ID")
		clientSecret := os.Getenv("AZURE_CLIENT_SECRET")
		if clientID != "" && clientSecret != "" {
			return &Credentials{
				Provider:   provider,
				AuthMethod: AuthMethodServiceKey,
				AccessKey:  clientID,
				SecretKey:  clientSecret,
				Properties: map[string]string{
					"tenant_id":       os.Getenv("AZURE_TENANT_ID"),
					"subscription_id": os.Getenv("AZURE_SUBSCRIPTION_ID"),
				},
			}
		}
	case ProviderGCP:
		if keyFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); keyFile != "" {
			return &Credentials{
				Provider:   provider,
				AuthMethod: AuthMethodServiceKey,
				Properties: map[string]string{
					"key_file": keyFile,
					"project":  os.Getenv("GOOGLE_CLOUD_PROJECT"),
				},
			}
		}
	}
	return nil
}

// getCLICredentials gets credentials from CLI configuration
func (h *CredentialHelper) getCLICredentials(provider Provider) *Credentials {
	// This would require parsing CLI config files
	// For now, return nil
	return nil
}

// CredentialCache provides in-memory caching of credentials
type CredentialCache struct {
	cache map[string]*cachedCredential
	mu    sync.RWMutex
	ttl   time.Duration
}

type cachedCredential struct {
	creds  *Credentials
	expiry time.Time
}

// NewCredentialCache creates a new credential cache
func NewCredentialCache(ttl time.Duration) *CredentialCache {
	return &CredentialCache{
		cache: make(map[string]*cachedCredential),
		ttl:   ttl,
	}
}

// Get retrieves credentials from cache
func (c *CredentialCache) Get(key string) (*Credentials, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, ok := c.cache[key]
	if !ok {
		return nil, false
	}

	if time.Now().After(cached.expiry) {
		return nil, false
	}

	return cached.creds, true
}

// Set stores credentials in cache
func (c *CredentialCache) Set(key string, creds *Credentials) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[key] = &cachedCredential{
		creds:  creds,
		expiry: time.Now().Add(c.ttl),
	}
}

// Clear clears the cache
func (c *CredentialCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*cachedCredential)
}
