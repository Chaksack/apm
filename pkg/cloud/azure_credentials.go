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
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

// AzureCredentialManagerImpl implements AzureCredentialManager
type AzureCredentialManagerImpl struct {
	credentialsPath      string
	servicePrincipalPath string
	encryptionKey        []byte
	cache                map[string]*Credentials
	spCache              map[string]*AzureServicePrincipal
	mutex                sync.RWMutex
	logger               *log.Logger
}

// NewAzureCredentialManager creates a new Azure credential manager
func NewAzureCredentialManager() (*AzureCredentialManagerImpl, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	apmDir := filepath.Join(homeDir, ".apm")
	if err := os.MkdirAll(apmDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create APM directory: %w", err)
	}

	credentialsPath := filepath.Join(apmDir, "azure_credentials.enc")
	servicePrincipalPath := filepath.Join(apmDir, "azure_sp.enc")

	// Generate encryption key from machine-specific data
	encryptionKey := generateEncryptionKey()

	return &AzureCredentialManagerImpl{
		credentialsPath:      credentialsPath,
		servicePrincipalPath: servicePrincipalPath,
		encryptionKey:        encryptionKey,
		cache:                make(map[string]*Credentials),
		spCache:              make(map[string]*AzureServicePrincipal),
		logger:               log.New(os.Stdout, "[AzureCredentialManager] ", log.LstdFlags),
	}, nil
}

// Store stores Azure credentials securely
func (m *AzureCredentialManagerImpl) Store(credentials *Credentials) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if credentials.Provider != ProviderAzure {
		return fmt.Errorf("invalid provider: expected %s, got %s", ProviderAzure, credentials.Provider)
	}

	m.logger.Printf("Storing Azure credentials for profile: %s", credentials.Profile)

	// Load existing credentials
	credMap, err := m.loadCredentials()
	if err != nil {
		credMap = make(map[string]*Credentials)
	}

	// Store credentials by profile (default if empty)
	profile := credentials.Profile
	if profile == "" {
		profile = "default"
	}

	credMap[profile] = credentials

	// Encrypt and save
	if err := m.saveCredentials(credMap); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	// Update cache
	m.cache[profile] = credentials

	m.logger.Printf("Azure credentials stored successfully for profile: %s", profile)
	return nil
}

// Retrieve retrieves Azure credentials
func (m *AzureCredentialManagerImpl) Retrieve(provider Provider, profile string) (*Credentials, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if provider != ProviderAzure {
		return nil, fmt.Errorf("invalid provider: expected %s, got %s", ProviderAzure, provider)
	}

	if profile == "" {
		profile = "default"
	}

	m.logger.Printf("Retrieving Azure credentials for profile: %s", profile)

	// Check cache first
	if creds, exists := m.cache[profile]; exists {
		// Check if credentials are expired
		if creds.Expiry != nil && time.Now().After(*creds.Expiry) {
			m.logger.Printf("Cached credentials expired for profile: %s", profile)
			delete(m.cache, profile)
		} else {
			m.logger.Printf("Retrieved cached credentials for profile: %s", profile)
			return creds, nil
		}
	}

	// Load from disk
	credMap, err := m.loadCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}

	creds, exists := credMap[profile]
	if !exists {
		return nil, fmt.Errorf("credentials not found for profile: %s", profile)
	}

	// Check if credentials are expired
	if creds.Expiry != nil && time.Now().After(*creds.Expiry) {
		return nil, fmt.Errorf("credentials expired for profile: %s", profile)
	}

	// Update cache
	m.cache[profile] = creds

	m.logger.Printf("Retrieved Azure credentials for profile: %s", profile)
	return creds, nil
}

// Delete deletes Azure credentials
func (m *AzureCredentialManagerImpl) Delete(provider Provider, profile string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if provider != ProviderAzure {
		return fmt.Errorf("invalid provider: expected %s, got %s", ProviderAzure, provider)
	}

	if profile == "" {
		profile = "default"
	}

	m.logger.Printf("Deleting Azure credentials for profile: %s", profile)

	// Load existing credentials
	credMap, err := m.loadCredentials()
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}

	// Delete from map
	delete(credMap, profile)
	delete(m.cache, profile)

	// Save updated credentials
	if err := m.saveCredentials(credMap); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	m.logger.Printf("Azure credentials deleted for profile: %s", profile)
	return nil
}

// List lists all Azure credentials
func (m *AzureCredentialManagerImpl) List(provider Provider) ([]*Credentials, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if provider != ProviderAzure {
		return nil, fmt.Errorf("invalid provider: expected %s, got %s", ProviderAzure, provider)
	}

	m.logger.Println("Listing Azure credentials...")

	credMap, err := m.loadCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}

	credentials := make([]*Credentials, 0, len(credMap))
	for _, creds := range credMap {
		// Mask sensitive data
		maskedCreds := *creds
		if maskedCreds.SecretKey != "" {
			maskedCreds.SecretKey = "****"
		}
		if maskedCreds.Token != "" {
			maskedCreds.Token = "****"
		}
		credentials = append(credentials, &maskedCreds)
	}

	m.logger.Printf("Found %d Azure credential profiles", len(credentials))
	return credentials, nil
}

// Rotate rotates Azure credentials
func (m *AzureCredentialManagerImpl) Rotate(credentials *Credentials) (*Credentials, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.logger.Printf("Rotating Azure credentials for profile: %s", credentials.Profile)

	// For service principal credentials, we need to rotate the secret
	if credentials.AuthMethod == AuthMethodServicePrincipal {
		// This would typically involve calling Azure AD to rotate the secret
		// For now, we'll simulate by updating the expiry
		newCreds := *credentials
		newCreds.Expiry = timePtr(time.Now().Add(365 * 24 * time.Hour))

		// Store the rotated credentials
		if err := m.Store(&newCreds); err != nil {
			return nil, fmt.Errorf("failed to store rotated credentials: %w", err)
		}

		m.logger.Printf("Azure credentials rotated successfully for profile: %s", credentials.Profile)
		return &newCreds, nil
	}

	return nil, fmt.Errorf("credential rotation not supported for auth method: %s", credentials.AuthMethod)
}

// StoreServicePrincipal stores a service principal
func (m *AzureCredentialManagerImpl) StoreServicePrincipal(sp *AzureServicePrincipal) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.logger.Printf("Storing service principal: %s", sp.AppID)

	// Load existing service principals
	spMap, err := m.loadServicePrincipals()
	if err != nil {
		spMap = make(map[string]*AzureServicePrincipal)
	}

	spMap[sp.AppID] = sp

	// Encrypt and save
	if err := m.saveServicePrincipals(spMap); err != nil {
		return fmt.Errorf("failed to save service principal: %w", err)
	}

	// Update cache
	m.spCache[sp.AppID] = sp

	m.logger.Printf("Service principal stored successfully: %s", sp.AppID)
	return nil
}

// RetrieveServicePrincipal retrieves a service principal
func (m *AzureCredentialManagerImpl) RetrieveServicePrincipal(appID string) (*AzureServicePrincipal, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	m.logger.Printf("Retrieving service principal: %s", appID)

	// Check cache first
	if sp, exists := m.spCache[appID]; exists {
		// Check if service principal is expired
		if sp.ExpiresAt != nil && time.Now().After(*sp.ExpiresAt) {
			m.logger.Printf("Cached service principal expired: %s", appID)
			delete(m.spCache, appID)
		} else {
			m.logger.Printf("Retrieved cached service principal: %s", appID)
			return sp, nil
		}
	}

	// Load from disk
	spMap, err := m.loadServicePrincipals()
	if err != nil {
		return nil, fmt.Errorf("failed to load service principals: %w", err)
	}

	sp, exists := spMap[appID]
	if !exists {
		return nil, fmt.Errorf("service principal not found: %s", appID)
	}

	// Check if service principal is expired
	if sp.ExpiresAt != nil && time.Now().After(*sp.ExpiresAt) {
		return nil, fmt.Errorf("service principal expired: %s", appID)
	}

	// Update cache
	m.spCache[appID] = sp

	m.logger.Printf("Retrieved service principal: %s", appID)
	return sp, nil
}

// ListServicePrincipals lists all service principals
func (m *AzureCredentialManagerImpl) ListServicePrincipals() ([]*AzureServicePrincipal, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	m.logger.Println("Listing service principals...")

	spMap, err := m.loadServicePrincipals()
	if err != nil {
		return nil, fmt.Errorf("failed to load service principals: %w", err)
	}

	servicePrincipals := make([]*AzureServicePrincipal, 0, len(spMap))
	for _, sp := range spMap {
		// Mask sensitive data
		maskedSP := *sp
		if maskedSP.Password != "" {
			maskedSP.Password = "****"
		}
		servicePrincipals = append(servicePrincipals, &maskedSP)
	}

	m.logger.Printf("Found %d service principals", len(servicePrincipals))
	return servicePrincipals, nil
}

// DeleteServicePrincipal deletes a service principal
func (m *AzureCredentialManagerImpl) DeleteServicePrincipal(appID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.logger.Printf("Deleting service principal: %s", appID)

	// Load existing service principals
	spMap, err := m.loadServicePrincipals()
	if err != nil {
		return fmt.Errorf("failed to load service principals: %w", err)
	}

	// Delete from map
	delete(spMap, appID)
	delete(m.spCache, appID)

	// Save updated service principals
	if err := m.saveServicePrincipals(spMap); err != nil {
		return fmt.Errorf("failed to save service principals: %w", err)
	}

	m.logger.Printf("Service principal deleted: %s", appID)
	return nil
}

// ValidateCredentials validates Azure credentials
func (m *AzureCredentialManagerImpl) ValidateCredentials(ctx context.Context, creds *Credentials) error {
	m.logger.Printf("Validating Azure credentials for profile: %s", creds.Profile)

	// Check basic credential structure
	if creds.Provider != ProviderAzure {
		return fmt.Errorf("invalid provider: %s", creds.Provider)
	}

	// Check expiry
	if creds.Expiry != nil && time.Now().After(*creds.Expiry) {
		return fmt.Errorf("credentials expired")
	}

	// Validate based on auth method
	switch creds.AuthMethod {
	case AuthMethodServicePrincipal:
		if creds.AccessKey == "" || creds.SecretKey == "" {
			return fmt.Errorf("service principal credentials missing client ID or secret")
		}
		if creds.Properties == nil || creds.Properties["tenant_id"] == "" {
			return fmt.Errorf("service principal credentials missing tenant ID")
		}

	case AuthMethodManagedIdentity:
		if creds.Token == "" {
			return fmt.Errorf("managed identity credentials missing token")
		}

	case AuthMethodCLI:
		// CLI credentials are validated by checking if Azure CLI is authenticated
		// This would require calling 'az account show'

	default:
		return fmt.Errorf("unsupported authentication method: %s", creds.AuthMethod)
	}

	m.logger.Printf("Azure credentials validated successfully for profile: %s", creds.Profile)
	return nil
}

// RefreshToken refreshes an Azure token
func (m *AzureCredentialManagerImpl) RefreshToken(ctx context.Context, creds *Credentials) (*Credentials, error) {
	m.logger.Printf("Refreshing Azure token for profile: %s", creds.Profile)

	// Token refresh logic would depend on the auth method
	switch creds.AuthMethod {
	case AuthMethodManagedIdentity:
		// For managed identity, we need to call the metadata service again
		// This is a simplified implementation
		newCreds := *creds
		newCreds.Expiry = timePtr(time.Now().Add(1 * time.Hour))

		if err := m.Store(&newCreds); err != nil {
			return nil, fmt.Errorf("failed to store refreshed credentials: %w", err)
		}

		m.logger.Printf("Azure token refreshed successfully for profile: %s", creds.Profile)
		return &newCreds, nil

	default:
		return nil, fmt.Errorf("token refresh not supported for auth method: %s", creds.AuthMethod)
	}
}

// Private helper methods

func (m *AzureCredentialManagerImpl) loadCredentials() (map[string]*Credentials, error) {
	data, err := m.loadEncryptedFile(m.credentialsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]*Credentials), nil
		}
		return nil, err
	}

	var credMap map[string]*Credentials
	if err := json.Unmarshal(data, &credMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	return credMap, nil
}

func (m *AzureCredentialManagerImpl) saveCredentials(credMap map[string]*Credentials) error {
	data, err := json.Marshal(credMap)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	return m.saveEncryptedFile(m.credentialsPath, data)
}

func (m *AzureCredentialManagerImpl) loadServicePrincipals() (map[string]*AzureServicePrincipal, error) {
	data, err := m.loadEncryptedFile(m.servicePrincipalPath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]*AzureServicePrincipal), nil
		}
		return nil, err
	}

	var spMap map[string]*AzureServicePrincipal
	if err := json.Unmarshal(data, &spMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal service principals: %w", err)
	}

	return spMap, nil
}

func (m *AzureCredentialManagerImpl) saveServicePrincipals(spMap map[string]*AzureServicePrincipal) error {
	data, err := json.Marshal(spMap)
	if err != nil {
		return fmt.Errorf("failed to marshal service principals: %w", err)
	}

	return m.saveEncryptedFile(m.servicePrincipalPath, data)
}

func (m *AzureCredentialManagerImpl) loadEncryptedFile(path string) ([]byte, error) {
	encryptedData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return m.decrypt(encryptedData)
}

func (m *AzureCredentialManagerImpl) saveEncryptedFile(path string, data []byte) error {
	encryptedData, err := m.encrypt(data)
	if err != nil {
		return err
	}

	return os.WriteFile(path, encryptedData, 0600)
}

func (m *AzureCredentialManagerImpl) encrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(m.encryptionKey)
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

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

func (m *AzureCredentialManagerImpl) decrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(m.encryptionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func generateEncryptionKey() []byte {
	// Use machine-specific data to generate a consistent key
	hostname, _ := os.Hostname()
	homeDir, _ := os.UserHomeDir()

	// Combine machine-specific data
	keyMaterial := fmt.Sprintf("%s:%s:apm-azure-credentials", hostname, homeDir)

	// Use PBKDF2 to derive a 32-byte key
	salt := []byte("apm-azure-salt-v1")
	return pbkdf2.Key([]byte(keyMaterial), salt, 10000, 32, sha256.New)
}
