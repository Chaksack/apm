package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"golang.org/x/crypto/pbkdf2"
)

// CredentialManager handles secure storage of credentials
type CredentialManager struct {
	key []byte
}

// NewCredentialManager creates a new credential manager
func NewCredentialManager() (*CredentialManager, error) {
	key, err := deriveSecureKey()
	if err != nil {
		return nil, fmt.Errorf("failed to derive encryption key: %w", err)
	}

	return &CredentialManager{
		key: key,
	}, nil
}

// deriveSecureKey generates a secure encryption key
func deriveSecureKey() ([]byte, error) {
	// Get machine-specific entropy sources
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	// Read random bytes for salt
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Store salt in a secure location (e.g., OS keyring or secure file)
	saltPath := getSaltPath()
	if err := storeSalt(saltPath, salt); err != nil {
		// Try to read existing salt
		existingSalt, readErr := readSalt(saltPath)
		if readErr != nil {
			return nil, fmt.Errorf("failed to store/read salt: %w", err)
		}
		salt = existingSalt
	}

	// Combine multiple entropy sources
	username := os.Getenv("USER")
	if username == "" {
		username = "default"
	}

	// Get process-specific entropy
	pid := os.Getpid()

	// Combine entropy sources
	combined := fmt.Sprintf("%s-%s-%d-%s-%d", hostname, username, pid, runtime.GOOS, runtime.GOARCH)

	// Use PBKDF2 with higher iteration count and random salt
	const iterations = 100000 // Much higher than before
	key := pbkdf2.Key([]byte(combined), salt, iterations, 32, sha256.New)

	return key, nil
}

// Encrypt encrypts data using AES-GCM
func (cm *CredentialManager) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(cm.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts data using AES-GCM
func (cm *CredentialManager) Decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(cm.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// EncryptString encrypts a string and returns base64 encoded result
func (cm *CredentialManager) EncryptString(plaintext string) (string, error) {
	encrypted, err := cm.Encrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// DecryptString decrypts a base64 encoded string
func (cm *CredentialManager) DecryptString(ciphertext string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	decrypted, err := cm.Decrypt(decoded)
	if err != nil {
		return "", err
	}

	return string(decrypted), nil
}

// getSaltPath returns the path where salt should be stored
func getSaltPath() string {
	// Store in user's home directory with restricted permissions
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "/tmp"
	}
	return fmt.Sprintf("%s/.apm/.salt", homeDir)
}

// storeSalt stores the salt with restricted permissions
func storeSalt(path string, salt []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write with restricted permissions (owner read/write only)
	if err := os.WriteFile(path, salt, 0600); err != nil {
		return fmt.Errorf("failed to write salt: %w", err)
	}

	return nil
}

// readSalt reads the stored salt
func readSalt(path string) ([]byte, error) {
	salt, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read salt: %w", err)
	}

	if len(salt) != 32 {
		return nil, fmt.Errorf("invalid salt length")
	}

	return salt, nil
}

// GenerateSecurePassword generates a cryptographically secure password
func GenerateSecurePassword(length int) (string, error) {
	if length < 12 {
		length = 12 // Minimum secure length
	}

	// Character set for password generation
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+-=[]{}|;:,.<>?"

	b := make([]byte, length)
	for i := range b {
		randomByte := make([]byte, 1)
		if _, err := rand.Read(randomByte); err != nil {
			return "", fmt.Errorf("failed to generate random byte: %w", err)
		}
		b[i] = charset[int(randomByte[0])%len(charset)]
	}

	return string(b), nil
}

// ClearString overwrites a string in memory
func ClearString(s *string) {
	if s != nil {
		// Overwrite the string with zeros
		bytes := []byte(*s)
		for i := range bytes {
			bytes[i] = 0
		}
		*s = ""
	}
}
