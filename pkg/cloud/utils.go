package cloud

import (
	"bufio"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

// CLIDetectionUtils provides utilities for detecting and validating cloud CLIs
type CLIDetectionUtils struct{}

// NewCLIDetectionUtils creates a new CLI detection utils instance
func NewCLIDetectionUtils() *CLIDetectionUtils {
	return &CLIDetectionUtils{}
}

// DetectCLI detects if a CLI tool is installed and gets its version
func (u *CLIDetectionUtils) DetectCLI(cliName string, versionArgs []string) (*CLIStatus, error) {
	status := &CLIStatus{
		Installed:   false,
		Version:     "",
		Path:        "",
		ConfigPath:  "",
		IsSupported: false,
	}

	// Find CLI path
	path, err := exec.LookPath(cliName)
	if err != nil {
		return status, nil // CLI not found, but not an error
	}

	status.Installed = true
	status.Path = path

	// Get version
	if len(versionArgs) > 0 {
		cmd := exec.Command(path, versionArgs...)
		output, err := cmd.Output()
		if err != nil {
			return status, fmt.Errorf("failed to get version for %s: %w", cliName, err)
		}

		version := u.extractVersion(string(output), cliName)
		status.Version = version
	}

	// Set config path based on CLI
	status.ConfigPath = u.getConfigPath(cliName)

	return status, nil
}

// ValidateMinVersion validates if the installed version meets minimum requirements
func (u *CLIDetectionUtils) ValidateMinVersion(installedVersion, minVersion string) bool {
	if installedVersion == "" || minVersion == "" {
		return false
	}

	installed := u.parseVersion(installedVersion)
	minimum := u.parseVersion(minVersion)

	return u.compareVersions(installed, minimum) >= 0
}

// extractVersion extracts version number from CLI output
func (u *CLIDetectionUtils) extractVersion(output, cliName string) string {
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		switch cliName {
		case "aws":
			if strings.Contains(line, "aws-cli/") {
				parts := strings.Split(line, "aws-cli/")
				if len(parts) > 1 {
					version := strings.Split(parts[1], " ")[0]
					return version
				}
			}
		case "az":
			if strings.Contains(line, "\"azure-cli\":") {
				// JSON format: {"azure-cli": "2.x.x", ...}
				var azureVersion map[string]string
				if err := json.Unmarshal([]byte(line), &azureVersion); err == nil {
					if version, exists := azureVersion["azure-cli"]; exists {
						return version
					}
				}
			}
		case "gcloud":
			if strings.Contains(line, "Google Cloud SDK") {
				parts := strings.Fields(line)
				for i, part := range parts {
					if part == "SDK" && i+1 < len(parts) {
						return parts[i+1]
					}
				}
			}
		}
	}

	return ""
}

// parseVersion parses a version string into comparable parts
func (u *CLIDetectionUtils) parseVersion(version string) []int {
	// Remove common prefixes
	version = strings.TrimPrefix(version, "v")
	version = strings.TrimPrefix(version, "version")
	version = strings.TrimSpace(version)

	// Split by dots and parse integers
	parts := strings.Split(version, ".")
	result := make([]int, len(parts))

	for i, part := range parts {
		// Remove any non-numeric suffixes (like "-beta", "-rc1")
		numPart := ""
		for _, char := range part {
			if char >= '0' && char <= '9' {
				numPart += string(char)
			} else {
				break
			}
		}

		if num, err := strconv.Atoi(numPart); err == nil {
			result[i] = num
		}
	}

	return result
}

// compareVersions compares two version arrays (-1: v1 < v2, 0: v1 == v2, 1: v1 > v2)
func (u *CLIDetectionUtils) compareVersions(v1, v2 []int) int {
	maxLen := len(v1)
	if len(v2) > maxLen {
		maxLen = len(v2)
	}

	for i := 0; i < maxLen; i++ {
		val1 := 0
		val2 := 0

		if i < len(v1) {
			val1 = v1[i]
		}
		if i < len(v2) {
			val2 = v2[i]
		}

		if val1 < val2 {
			return -1
		}
		if val1 > val2 {
			return 1
		}
	}

	return 0
}

// getConfigPath returns the config path for a CLI tool
func (u *CLIDetectionUtils) getConfigPath(cliName string) string {
	homeDir, _ := os.UserHomeDir()

	switch cliName {
	case "aws":
		return filepath.Join(homeDir, ".aws")
	case "az":
		return filepath.Join(homeDir, ".azure")
	case "gcloud":
		if runtime.GOOS == "windows" {
			return filepath.Join(homeDir, "AppData", "Roaming", "gcloud")
		}
		return filepath.Join(homeDir, ".config", "gcloud")
	default:
		return ""
	}
}

// CredentialStorage provides secure credential storage utilities
type CredentialStorage struct {
	storePath string
	key       []byte
}

// NewCredentialStorage creates a new credential storage instance
func NewCredentialStorage(storePath string) (*CredentialStorage, error) {
	if err := os.MkdirAll(filepath.Dir(storePath), 0700); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	storage := &CredentialStorage{
		storePath: storePath,
	}

	// Generate or load encryption key
	if err := storage.initializeKey(); err != nil {
		return nil, fmt.Errorf("failed to initialize encryption key: %w", err)
	}

	return storage, nil
}

// Store stores encrypted credentials
func (cs *CredentialStorage) Store(provider Provider, profile string, creds *Credentials) error {
	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	encrypted, err := cs.encrypt(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	filename := fmt.Sprintf("%s_%s.enc", provider, profile)
	if profile == "" {
		filename = fmt.Sprintf("%s_default.enc", provider)
	}

	filePath := filepath.Join(filepath.Dir(cs.storePath), filename)

	if err := os.WriteFile(filePath, encrypted, 0600); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}

	return nil
}

// Retrieve retrieves and decrypts credentials
func (cs *CredentialStorage) Retrieve(provider Provider, profile string) (*Credentials, error) {
	filename := fmt.Sprintf("%s_%s.enc", provider, profile)
	if profile == "" {
		filename = fmt.Sprintf("%s_default.enc", provider)
	}

	filePath := filepath.Join(filepath.Dir(cs.storePath), filename)

	encrypted, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	data, err := cs.decrypt(encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	return &creds, nil
}

// Delete deletes stored credentials
func (cs *CredentialStorage) Delete(provider Provider, profile string) error {
	filename := fmt.Sprintf("%s_%s.enc", provider, profile)
	if profile == "" {
		filename = fmt.Sprintf("%s_default.enc", provider)
	}

	filePath := filepath.Join(filepath.Dir(cs.storePath), filename)

	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete credentials file: %w", err)
	}

	return nil
}

// List lists all stored credential profiles for a provider
func (cs *CredentialStorage) List(provider Provider) ([]string, error) {
	dir := filepath.Dir(cs.storePath)
	pattern := fmt.Sprintf("%s_*.enc", provider)

	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return nil, fmt.Errorf("failed to list credentials: %w", err)
	}

	var profiles []string
	for _, match := range matches {
		filename := filepath.Base(match)
		// Remove provider prefix and .enc suffix
		name := strings.TrimSuffix(strings.TrimPrefix(filename, string(provider)+"_"), ".enc")
		if name == "default" {
			name = "" // Default profile is represented as empty string
		}
		profiles = append(profiles, name)
	}

	return profiles, nil
}

// initializeKey initializes the encryption key
func (cs *CredentialStorage) initializeKey() error {
	keyFile := cs.storePath + ".key"

	// Try to load existing key
	if keyData, err := os.ReadFile(keyFile); err == nil {
		cs.key = keyData
		return nil
	}

	// Generate new key using machine-specific salt
	salt := cs.getMachineSpecificSalt()
	password := cs.getMachinePassword()

	cs.key = pbkdf2.Key([]byte(password), salt, 10000, 32, sha256.New)

	// Store the salt (not the key) for future use
	return os.WriteFile(keyFile, salt, 0600)
}

// getMachineSpecificSalt generates a machine-specific salt
func (cs *CredentialStorage) getMachineSpecificSalt() []byte {
	// Use machine-specific information to generate salt
	hostname, _ := os.Hostname()
	homeDir, _ := os.UserHomeDir()

	info := fmt.Sprintf("%s:%s:%s", hostname, homeDir, runtime.GOOS)
	hash := sha256.Sum256([]byte(info))
	return hash[:16] // Use first 16 bytes as salt
}

// getMachinePassword generates a machine-specific password
func (cs *CredentialStorage) getMachinePassword() string {
	hostname, _ := os.Hostname()
	homeDir, _ := os.UserHomeDir()

	return fmt.Sprintf("apm-cloud-%s-%s", hostname, filepath.Base(homeDir))
}

// encrypt encrypts data using AES-GCM
func (cs *CredentialStorage) encrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(cs.key)
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

// decrypt decrypts data using AES-GCM
func (cs *CredentialStorage) decrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(cs.key)
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

// ConfigFileManager provides configuration file management utilities
type ConfigFileManager struct {
	baseDir string
}

// NewConfigFileManager creates a new config file manager
func NewConfigFileManager(baseDir string) (*ConfigFileManager, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	return &ConfigFileManager{
		baseDir: baseDir,
	}, nil
}

// Save saves configuration to file
func (cfm *ConfigFileManager) Save(provider Provider, environment string, config *ProviderConfig) error {
	filename := cfm.getConfigFilename(provider, environment)
	filePath := filepath.Join(cfm.baseDir, filename)

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Load loads configuration from file
func (cfm *ConfigFileManager) Load(provider Provider, environment string) (*ProviderConfig, error) {
	filename := cfm.getConfigFilename(provider, environment)
	filePath := filepath.Join(cfm.baseDir, filename)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ProviderConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// Delete deletes a configuration file
func (cfm *ConfigFileManager) Delete(provider Provider, environment string) error {
	filename := cfm.getConfigFilename(provider, environment)
	filePath := filepath.Join(cfm.baseDir, filename)

	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete config file: %w", err)
	}

	return nil
}

// ListEnvironments lists all environments for a provider
func (cfm *ConfigFileManager) ListEnvironments(provider Provider) ([]string, error) {
	pattern := fmt.Sprintf("%s_*.json", provider)
	matches, err := filepath.Glob(filepath.Join(cfm.baseDir, pattern))
	if err != nil {
		return nil, fmt.Errorf("failed to list environments: %w", err)
	}

	var environments []string
	for _, match := range matches {
		filename := filepath.Base(match)
		// Extract environment from filename: provider_environment.json
		parts := strings.Split(strings.TrimSuffix(filename, ".json"), "_")
		if len(parts) >= 2 {
			environment := strings.Join(parts[1:], "_")
			if environment != "default" {
				environments = append(environments, environment)
			}
		}
	}

	// Check for default config
	defaultFile := filepath.Join(cfm.baseDir, fmt.Sprintf("%s.json", provider))
	if _, err := os.Stat(defaultFile); err == nil {
		environments = append([]string{""}, environments...) // Empty string represents default
	}

	return environments, nil
}

// getConfigFilename returns the config filename for a provider and environment
func (cfm *ConfigFileManager) getConfigFilename(provider Provider, environment string) string {
	if environment == "" {
		return fmt.Sprintf("%s.json", provider)
	}
	return fmt.Sprintf("%s_%s.json", provider, environment)
}

// CrossPlatformUtils provides cross-platform compatibility utilities
type CrossPlatformUtils struct{}

// NewCrossPlatformUtils creates a new cross-platform utils instance
func NewCrossPlatformUtils() *CrossPlatformUtils {
	return &CrossPlatformUtils{}
}

// GetHomeDirectory returns the user's home directory
func (cpu *CrossPlatformUtils) GetHomeDirectory() (string, error) {
	return os.UserHomeDir()
}

// GetConfigDirectory returns the appropriate config directory for the platform
func (cpu *CrossPlatformUtils) GetConfigDirectory(appName string) (string, error) {
	homeDir, err := cpu.GetHomeDirectory()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "windows":
		configDir := os.Getenv("APPDATA")
		if configDir == "" {
			configDir = filepath.Join(homeDir, "AppData", "Roaming")
		}
		return filepath.Join(configDir, appName), nil
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", appName), nil
	default: // Linux and other Unix-like systems
		configDir := os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			configDir = filepath.Join(homeDir, ".config")
		}
		return filepath.Join(configDir, appName), nil
	}
}

// GetCacheDirectory returns the appropriate cache directory for the platform
func (cpu *CrossPlatformUtils) GetCacheDirectory(appName string) (string, error) {
	homeDir, err := cpu.GetHomeDirectory()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "windows":
		cacheDir := os.Getenv("LOCALAPPDATA")
		if cacheDir == "" {
			cacheDir = filepath.Join(homeDir, "AppData", "Local")
		}
		return filepath.Join(cacheDir, appName), nil
	case "darwin":
		return filepath.Join(homeDir, "Library", "Caches", appName), nil
	default: // Linux and other Unix-like systems
		cacheDir := os.Getenv("XDG_CACHE_HOME")
		if cacheDir == "" {
			cacheDir = filepath.Join(homeDir, ".cache")
		}
		return filepath.Join(cacheDir, appName), nil
	}
}

// IsExecutable checks if a file is executable
func (cpu *CrossPlatformUtils) IsExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	if runtime.GOOS == "windows" {
		// On Windows, check file extension
		ext := strings.ToLower(filepath.Ext(path))
		return ext == ".exe" || ext == ".bat" || ext == ".cmd"
	}

	// On Unix-like systems, check execute permissions
	mode := info.Mode()
	return mode&0111 != 0
}

// RunCommand runs a command with timeout and returns output
func (cpu *CrossPlatformUtils) RunCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)

	// Set up environment
	cmd.Env = os.Environ()

	// Platform-specific configurations
	cpu.configureCommand(cmd)

	return cmd.Output()
}

// RunCommandWithInput runs a command with input and returns output
func (cpu *CrossPlatformUtils) RunCommandWithInput(ctx context.Context, input string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = strings.NewReader(input)

	// Set up environment
	cmd.Env = os.Environ()

	// Platform-specific configurations
	cpu.configureCommand(cmd)

	return cmd.Output()
}

// StreamCommand runs a command and streams output
func (cpu *CrossPlatformUtils) StreamCommand(ctx context.Context, name string, args ...string) (io.ReadCloser, error) {
	cmd := exec.CommandContext(ctx, name, args...)

	// Set up environment
	cmd.Env = os.Environ()

	// Platform-specific configurations
	cpu.configureCommand(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		stdout.Close()
		return nil, err
	}

	// Return a ReadCloser that also waits for the command to finish
	return &commandReader{
		reader: stdout,
		cmd:    cmd,
	}, nil
}

// configureCommand configures platform-specific command settings
func (cpu *CrossPlatformUtils) configureCommand(cmd *exec.Cmd) {
	// Platform-specific configurations can be added here
	// For now, we'll keep it simple and avoid Windows-specific syscall attributes
}

// commandReader wraps command output with proper cleanup
type commandReader struct {
	reader io.ReadCloser
	cmd    *exec.Cmd
}

func (cr *commandReader) Read(p []byte) (n int, err error) {
	return cr.reader.Read(p)
}

func (cr *commandReader) Close() error {
	cr.reader.Close()
	return cr.cmd.Wait()
}

// FileSystemUtils provides file system utilities
type FileSystemUtils struct{}

// NewFileSystemUtils creates a new filesystem utils instance
func NewFileSystemUtils() *FileSystemUtils {
	return &FileSystemUtils{}
}

// EnsureDirectory ensures a directory exists with proper permissions
func (fsu *FileSystemUtils) EnsureDirectory(path string, mode os.FileMode) error {
	return os.MkdirAll(path, mode)
}

// BackupFile creates a backup of a file
func (fsu *FileSystemUtils) BackupFile(srcPath string) (string, error) {
	backupPath := srcPath + ".backup." + time.Now().Format("20060102-150405")

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		os.Remove(backupPath) // Clean up on failure
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	return backupPath, nil
}

// ReadLines reads all lines from a file
func (fsu *FileSystemUtils) ReadLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}

// WriteLines writes lines to a file
func (fsu *FileSystemUtils) WriteLines(path string, lines []string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return err
		}
	}

	return writer.Flush()
}

// FindExecutable finds an executable in the system PATH
func (fsu *FileSystemUtils) FindExecutable(name string) (string, error) {
	return exec.LookPath(name)
}

// IsFile checks if a path is a regular file
func (fsu *FileSystemUtils) IsFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}

// IsDirectory checks if a path is a directory
func (fsu *FileSystemUtils) IsDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// GetFileSize returns the size of a file in bytes
func (fsu *FileSystemUtils) GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// GetFileModTime returns the modification time of a file
func (fsu *FileSystemUtils) GetFileModTime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// EnvironmentUtils provides environment variable utilities
type EnvironmentUtils struct{}

// NewEnvironmentUtils creates a new environment utils instance
func NewEnvironmentUtils() *EnvironmentUtils {
	return &EnvironmentUtils{}
}

// GetEnvWithDefault gets environment variable with default value
func (eu *EnvironmentUtils) GetEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// SetEnv sets an environment variable
func (eu *EnvironmentUtils) SetEnv(key, value string) error {
	return os.Setenv(key, value)
}

// UnsetEnv unsets an environment variable
func (eu *EnvironmentUtils) UnsetEnv(key string) error {
	return os.Unsetenv(key)
}

// GetAllEnv returns all environment variables as a map
func (eu *EnvironmentUtils) GetAllEnv() map[string]string {
	env := make(map[string]string)
	for _, entry := range os.Environ() {
		if idx := strings.Index(entry, "="); idx > 0 {
			key := entry[:idx]
			value := entry[idx+1:]
			env[key] = value
		}
	}
	return env
}

// FilterEnv returns environment variables matching a prefix
func (eu *EnvironmentUtils) FilterEnv(prefix string) map[string]string {
	env := make(map[string]string)
	prefix = strings.ToUpper(prefix)

	for _, entry := range os.Environ() {
		if idx := strings.Index(entry, "="); idx > 0 {
			key := entry[:idx]
			value := entry[idx+1:]
			if strings.HasPrefix(strings.ToUpper(key), prefix) {
				env[key] = value
			}
		}
	}
	return env
}

// ExpandEnv expands environment variables in a string
func (eu *EnvironmentUtils) ExpandEnv(s string) string {
	return os.ExpandEnv(s)
}

// EncodingUtils provides encoding/decoding utilities
type EncodingUtils struct{}

// NewEncodingUtils creates a new encoding utils instance
func NewEncodingUtils() *EncodingUtils {
	return &EncodingUtils{}
}

// Base64Encode encodes data to base64
func (enc *EncodingUtils) Base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// Base64Decode decodes base64 data
func (enc *EncodingUtils) Base64Decode(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// JSONMarshal marshals data to JSON with proper formatting
func (enc *EncodingUtils) JSONMarshal(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

// JSONUnmarshal unmarshals JSON data
func (enc *EncodingUtils) JSONUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// JSONPrettyPrint pretty prints JSON data
func (enc *EncodingUtils) JSONPrettyPrint(data []byte) ([]byte, error) {
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	return json.MarshalIndent(obj, "", "  ")
}

// HashSHA256 computes SHA256 hash of data
func (enc *EncodingUtils) HashSHA256(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

// HashSHA256String computes SHA256 hash and returns as hex string
func (enc *EncodingUtils) HashSHA256String(data []byte) string {
	hash := enc.HashSHA256(data)
	return fmt.Sprintf("%x", hash)
}
