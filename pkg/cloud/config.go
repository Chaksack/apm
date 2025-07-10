package cloud

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DefaultConfigManager implements the ConfigManager interface
type DefaultConfigManager struct {
	baseDir     string
	fileManager *ConfigFileManager
	mu          sync.RWMutex
	cache       map[string]*ProviderConfig
	cacheExpiry map[string]time.Time
	cacheTTL    time.Duration
}

// NewDefaultConfigManager creates a new default config manager
func NewDefaultConfigManager(baseDir string) (*DefaultConfigManager, error) {
	fileManager, err := NewConfigFileManager(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create file manager: %w", err)
	}

	return &DefaultConfigManager{
		baseDir:     baseDir,
		fileManager: fileManager,
		cache:       make(map[string]*ProviderConfig),
		cacheExpiry: make(map[string]time.Time),
		cacheTTL:    15 * time.Minute,
	}, nil
}

// LoadConfig loads configuration for a provider
func (dcm *DefaultConfigManager) LoadConfig(provider Provider) (*ProviderConfig, error) {
	return dcm.LoadEnvironmentConfig(provider, "")
}

// SaveConfig saves configuration for a provider
func (dcm *DefaultConfigManager) SaveConfig(provider Provider, config *ProviderConfig) error {
	return dcm.SaveEnvironmentConfig(provider, "", config)
}

// DeleteConfig deletes configuration for a provider
func (dcm *DefaultConfigManager) DeleteConfig(provider Provider) error {
	// Delete from cache
	dcm.mu.Lock()
	delete(dcm.cache, dcm.getCacheKey(provider, ""))
	delete(dcm.cacheExpiry, dcm.getCacheKey(provider, ""))
	dcm.mu.Unlock()

	// Delete from file system
	return dcm.fileManager.Delete(provider, "")
}

// LoadEnvironmentConfig loads environment-specific configuration
func (dcm *DefaultConfigManager) LoadEnvironmentConfig(provider Provider, environment string) (*ProviderConfig, error) {
	cacheKey := dcm.getCacheKey(provider, environment)

	// Check cache first
	dcm.mu.RLock()
	if config, exists := dcm.cache[cacheKey]; exists {
		if expiry, hasExpiry := dcm.cacheExpiry[cacheKey]; hasExpiry && time.Now().Before(expiry) {
			dcm.mu.RUnlock()
			return dcm.cloneConfig(config), nil
		}
	}
	dcm.mu.RUnlock()

	// Load from file
	config, err := dcm.fileManager.Load(provider, environment)
	if err != nil {
		// If environment-specific config not found, try default
		if environment != "" {
			if defaultConfig, defaultErr := dcm.fileManager.Load(provider, ""); defaultErr == nil {
				config = defaultConfig
				err = nil
			}
		}

		if err != nil {
			return nil, fmt.Errorf("failed to load config for %s: %w", provider, err)
		}
	}

	// Validate loaded config
	if validationResult := dcm.validateConfigInternal(config); !validationResult.Valid {
		return nil, fmt.Errorf("invalid configuration: %v", validationResult.Errors)
	}

	// Cache the config
	dcm.mu.Lock()
	dcm.cache[cacheKey] = dcm.cloneConfig(config)
	dcm.cacheExpiry[cacheKey] = time.Now().Add(dcm.cacheTTL)
	dcm.mu.Unlock()

	return config, nil
}

// SaveEnvironmentConfig saves environment-specific configuration
func (dcm *DefaultConfigManager) SaveEnvironmentConfig(provider Provider, environment string, config *ProviderConfig) error {
	// Validate config before saving
	if validationResult := dcm.validateConfigInternal(config); !validationResult.Valid {
		return fmt.Errorf("invalid configuration: %v", validationResult.Errors)
	}

	// Save to file
	if err := dcm.fileManager.Save(provider, environment, config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Update cache
	cacheKey := dcm.getCacheKey(provider, environment)
	dcm.mu.Lock()
	dcm.cache[cacheKey] = dcm.cloneConfig(config)
	dcm.cacheExpiry[cacheKey] = time.Now().Add(dcm.cacheTTL)
	dcm.mu.Unlock()

	return nil
}

// ListEnvironments lists all environments for a provider
func (dcm *DefaultConfigManager) ListEnvironments(provider Provider) ([]string, error) {
	return dcm.fileManager.ListEnvironments(provider)
}

// ValidateConfig validates provider configuration
func (dcm *DefaultConfigManager) ValidateConfig(config *ProviderConfig) (*ValidationResult, error) {
	result := dcm.validateConfigInternal(config)
	return &result, nil
}

// MergeConfigs merges base and override configurations
func (dcm *DefaultConfigManager) MergeConfigs(base, override *ProviderConfig) (*ProviderConfig, error) {
	if base == nil && override == nil {
		return nil, fmt.Errorf("both base and override configs are nil")
	}

	if base == nil {
		return dcm.cloneConfig(override), nil
	}

	if override == nil {
		return dcm.cloneConfig(base), nil
	}

	// Start with a copy of base config
	merged := dcm.cloneConfig(base)

	// Override with non-zero values from override config
	if override.Provider != "" {
		merged.Provider = override.Provider
	}
	if override.DefaultRegion != "" {
		merged.DefaultRegion = override.DefaultRegion
	}
	if override.DefaultProfile != "" {
		merged.DefaultProfile = override.DefaultProfile
	}
	if override.CLIPath != "" {
		merged.CLIPath = override.CLIPath
	}
	if override.ConfigPath != "" {
		merged.ConfigPath = override.ConfigPath
	}
	if override.CacheDuration != 0 {
		merged.CacheDuration = override.CacheDuration
	}

	// Override boolean values explicitly
	merged.EnableCache = override.EnableCache

	// Merge maps
	if override.CustomEndpoints != nil {
		if merged.CustomEndpoints == nil {
			merged.CustomEndpoints = make(map[string]string)
		}
		for k, v := range override.CustomEndpoints {
			merged.CustomEndpoints[k] = v
		}
	}

	return merged, nil
}

// BackupConfig creates a backup of provider configuration
func (dcm *DefaultConfigManager) BackupConfig(provider Provider) ([]byte, error) {
	config, err := dcm.LoadConfig(provider)
	if err != nil {
		return nil, fmt.Errorf("failed to load config for backup: %w", err)
	}

	// Create backup data structure
	backup := ConfigBackup{
		Provider:   provider,
		Config:     config,
		BackupTime: time.Now(),
		Version:    "1.0",
	}

	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal backup: %w", err)
	}

	return data, nil
}

// RestoreConfig restores configuration from backup
func (dcm *DefaultConfigManager) RestoreConfig(provider Provider, data []byte) error {
	var backup ConfigBackup
	if err := json.Unmarshal(data, &backup); err != nil {
		return fmt.Errorf("failed to unmarshal backup: %w", err)
	}

	// Validate that the backup is for the correct provider
	if backup.Provider != provider {
		return fmt.Errorf("backup is for provider %s, not %s", backup.Provider, provider)
	}

	// Validate the config from backup
	if validationResult := dcm.validateConfigInternal(backup.Config); !validationResult.Valid {
		return fmt.Errorf("backup contains invalid configuration: %v", validationResult.Errors)
	}

	// Save the restored config
	return dcm.SaveConfig(provider, backup.Config)
}

// getCacheKey generates a cache key for provider and environment
func (dcm *DefaultConfigManager) getCacheKey(provider Provider, environment string) string {
	if environment == "" {
		return string(provider)
	}
	return fmt.Sprintf("%s:%s", provider, environment)
}

// cloneConfig creates a deep copy of a config
func (dcm *DefaultConfigManager) cloneConfig(config *ProviderConfig) *ProviderConfig {
	if config == nil {
		return nil
	}

	clone := *config

	// Deep copy maps
	if config.CustomEndpoints != nil {
		clone.CustomEndpoints = make(map[string]string)
		for k, v := range config.CustomEndpoints {
			clone.CustomEndpoints[k] = v
		}
	}

	return &clone
}

// validateConfigInternal performs internal validation of config
func (dcm *DefaultConfigManager) validateConfigInternal(config *ProviderConfig) ValidationResult {
	result := ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
		Details:  make(map[string]string),
	}

	if config == nil {
		result.Valid = false
		result.Errors = append(result.Errors, "config is nil")
		return result
	}

	// Validate provider
	if config.Provider == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "provider is required")
	} else if config.Provider != ProviderAWS && config.Provider != ProviderAzure && config.Provider != ProviderGCP {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("unsupported provider: %s", config.Provider))
	}

	// Validate region
	if config.DefaultRegion == "" {
		result.Warnings = append(result.Warnings, "default region is not set")
	}

	// Validate CLI path if specified
	if config.CLIPath != "" {
		if _, err := os.Stat(config.CLIPath); err != nil {
			if os.IsNotExist(err) {
				result.Warnings = append(result.Warnings, fmt.Sprintf("CLI path does not exist: %s", config.CLIPath))
			} else {
				result.Warnings = append(result.Warnings, fmt.Sprintf("cannot access CLI path: %s", config.CLIPath))
			}
		}
	}

	// Validate config path if specified
	if config.ConfigPath != "" {
		if _, err := os.Stat(config.ConfigPath); err != nil {
			if os.IsNotExist(err) {
				result.Warnings = append(result.Warnings, fmt.Sprintf("config path does not exist: %s", config.ConfigPath))
			} else {
				result.Warnings = append(result.Warnings, fmt.Sprintf("cannot access config path: %s", config.ConfigPath))
			}
		}
	}

	// Validate cache duration
	if config.CacheDuration < 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "cache duration cannot be negative")
	} else if config.CacheDuration > 24*time.Hour {
		result.Warnings = append(result.Warnings, "cache duration is longer than 24 hours")
	}

	// Provider-specific validations
	switch config.Provider {
	case ProviderAWS:
		dcm.validateAWSConfig(config, &result)
	case ProviderAzure:
		dcm.validateAzureConfig(config, &result)
	case ProviderGCP:
		dcm.validateGCPConfig(config, &result)
	}

	return result
}

// validateAWSConfig validates AWS-specific configuration
func (dcm *DefaultConfigManager) validateAWSConfig(config *ProviderConfig, result *ValidationResult) {
	// Validate AWS regions
	validAWSRegions := map[string]bool{
		"us-east-1": true, "us-east-2": true, "us-west-1": true, "us-west-2": true,
		"eu-west-1": true, "eu-west-2": true, "eu-west-3": true, "eu-central-1": true,
		"ap-southeast-1": true, "ap-southeast-2": true, "ap-northeast-1": true, "ap-northeast-2": true,
		"ap-south-1": true, "sa-east-1": true, "ca-central-1": true,
	}

	if config.DefaultRegion != "" && !validAWSRegions[config.DefaultRegion] {
		result.Warnings = append(result.Warnings, fmt.Sprintf("AWS region '%s' may not be valid", config.DefaultRegion))
	}

	// Check for AWS-specific endpoints
	if config.CustomEndpoints != nil {
		for service, endpoint := range config.CustomEndpoints {
			if service == "s3" || service == "ec2" || service == "iam" {
				result.Details[fmt.Sprintf("aws_%s_endpoint", service)] = endpoint
			}
		}
	}
}

// validateAzureConfig validates Azure-specific configuration
func (dcm *DefaultConfigManager) validateAzureConfig(config *ProviderConfig, result *ValidationResult) {
	// Validate Azure regions
	validAzureRegions := map[string]bool{
		"eastus": true, "eastus2": true, "westus": true, "westus2": true,
		"centralus": true, "northcentralus": true, "southcentralus": true, "westcentralus": true,
		"northeurope": true, "westeurope": true, "eastasia": true, "southeastasia": true,
		"japaneast": true, "japanwest": true, "australiaeast": true, "australiasoutheast": true,
	}

	if config.DefaultRegion != "" && !validAzureRegions[config.DefaultRegion] {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Azure region '%s' may not be valid", config.DefaultRegion))
	}
}

// validateGCPConfig validates GCP-specific configuration
func (dcm *DefaultConfigManager) validateGCPConfig(config *ProviderConfig, result *ValidationResult) {
	// Validate GCP regions
	validGCPRegions := map[string]bool{
		"us-central1": true, "us-east1": true, "us-east4": true, "us-west1": true, "us-west2": true, "us-west3": true, "us-west4": true,
		"europe-north1": true, "europe-west1": true, "europe-west2": true, "europe-west3": true, "europe-west4": true, "europe-west6": true,
		"asia-east1": true, "asia-east2": true, "asia-northeast1": true, "asia-northeast2": true, "asia-northeast3": true,
		"asia-south1": true, "asia-southeast1": true, "asia-southeast2": true,
		"australia-southeast1": true, "northamerica-northeast1": true, "southamerica-east1": true,
	}

	if config.DefaultRegion != "" && !validGCPRegions[config.DefaultRegion] {
		result.Warnings = append(result.Warnings, fmt.Sprintf("GCP region '%s' may not be valid", config.DefaultRegion))
	}
}

// ClearCache clears the configuration cache
func (dcm *DefaultConfigManager) ClearCache() {
	dcm.mu.Lock()
	defer dcm.mu.Unlock()

	dcm.cache = make(map[string]*ProviderConfig)
	dcm.cacheExpiry = make(map[string]time.Time)
}

// SetCacheTTL sets the cache time-to-live
func (dcm *DefaultConfigManager) SetCacheTTL(ttl time.Duration) {
	dcm.mu.Lock()
	defer dcm.mu.Unlock()

	dcm.cacheTTL = ttl
}

// GetCacheStats returns cache statistics
func (dcm *DefaultConfigManager) GetCacheStats() ConfigCacheStats {
	dcm.mu.RLock()
	defer dcm.mu.RUnlock()

	now := time.Now()
	var expired int

	for _, expiry := range dcm.cacheExpiry {
		if now.After(expiry) {
			expired++
		}
	}

	return ConfigCacheStats{
		TotalEntries:   len(dcm.cache),
		ExpiredEntries: expired,
		TTL:            dcm.cacheTTL,
	}
}

// ConfigBackup represents a configuration backup
type ConfigBackup struct {
	Provider   Provider        `json:"provider"`
	Config     *ProviderConfig `json:"config"`
	BackupTime time.Time       `json:"backup_time"`
	Version    string          `json:"version"`
}

// ConfigCacheStats represents cache statistics
type ConfigCacheStats struct {
	TotalEntries   int           `json:"total_entries"`
	ExpiredEntries int           `json:"expired_entries"`
	TTL            time.Duration `json:"ttl"`
}

// ConfigTemplate represents a configuration template
type ConfigTemplate struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Provider    Provider               `json:"provider"`
	Config      *ProviderConfig        `json:"config"`
	Variables   map[string]string      `json:"variables,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// TemplateManager manages configuration templates
type TemplateManager struct {
	baseDir string
	mu      sync.RWMutex
}

// NewTemplateManager creates a new template manager
func NewTemplateManager(baseDir string) (*TemplateManager, error) {
	templatesDir := filepath.Join(baseDir, "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create templates directory: %w", err)
	}

	return &TemplateManager{
		baseDir: templatesDir,
	}, nil
}

// SaveTemplate saves a configuration template
func (tm *TemplateManager) SaveTemplate(template *ConfigTemplate) error {
	if template.Name == "" {
		return fmt.Errorf("template name is required")
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	filename := fmt.Sprintf("%s_%s.json", template.Provider, template.Name)
	filePath := filepath.Join(tm.baseDir, filename)

	data, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write template: %w", err)
	}

	return nil
}

// LoadTemplate loads a configuration template
func (tm *TemplateManager) LoadTemplate(provider Provider, name string) (*ConfigTemplate, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	filename := fmt.Sprintf("%s_%s.json", provider, name)
	filePath := filepath.Join(tm.baseDir, filename)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template: %w", err)
	}

	var template ConfigTemplate
	if err := json.Unmarshal(data, &template); err != nil {
		return nil, fmt.Errorf("failed to unmarshal template: %w", err)
	}

	return &template, nil
}

// ListTemplates lists all templates for a provider
func (tm *TemplateManager) ListTemplates(provider Provider) ([]*ConfigTemplate, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	pattern := fmt.Sprintf("%s_*.json", provider)
	matches, err := filepath.Glob(filepath.Join(tm.baseDir, pattern))
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}

	var templates []*ConfigTemplate
	for _, match := range matches {
		data, err := os.ReadFile(match)
		if err != nil {
			continue // Skip files that can't be read
		}

		var template ConfigTemplate
		if err := json.Unmarshal(data, &template); err != nil {
			continue // Skip files that can't be parsed
		}

		templates = append(templates, &template)
	}

	return templates, nil
}

// DeleteTemplate deletes a configuration template
func (tm *TemplateManager) DeleteTemplate(provider Provider, name string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	filename := fmt.Sprintf("%s_%s.json", provider, name)
	filePath := filepath.Join(tm.baseDir, filename)

	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	return nil
}

// ApplyTemplate applies a template to create a configuration
func (tm *TemplateManager) ApplyTemplate(provider Provider, templateName string, variables map[string]string) (*ProviderConfig, error) {
	template, err := tm.LoadTemplate(provider, templateName)
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}

	// Clone the template config
	config := &ProviderConfig{}
	data, err := json.Marshal(template.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize template config: %w", err)
	}

	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to deserialize template config: %w", err)
	}

	// Apply variable substitutions
	if err := tm.applyVariables(config, variables); err != nil {
		return nil, fmt.Errorf("failed to apply variables: %w", err)
	}

	return config, nil
}

// applyVariables applies variable substitutions to a config
func (tm *TemplateManager) applyVariables(config *ProviderConfig, variables map[string]string) error {
	// Simple variable substitution - in a real implementation, you might use a templating engine
	if variables == nil {
		return nil
	}

	// Apply to string fields
	config.DefaultRegion = tm.substituteVariables(config.DefaultRegion, variables)
	config.DefaultProfile = tm.substituteVariables(config.DefaultProfile, variables)
	config.CLIPath = tm.substituteVariables(config.CLIPath, variables)
	config.ConfigPath = tm.substituteVariables(config.ConfigPath, variables)

	// Apply to map values
	if config.CustomEndpoints != nil {
		for key, value := range config.CustomEndpoints {
			config.CustomEndpoints[key] = tm.substituteVariables(value, variables)
		}
	}

	return nil
}

// substituteVariables performs simple variable substitution
func (tm *TemplateManager) substituteVariables(input string, variables map[string]string) string {
	result := input
	for key, value := range variables {
		placeholder := fmt.Sprintf("${%s}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// GetBuiltinTemplates returns built-in configuration templates
func (tm *TemplateManager) GetBuiltinTemplates() []*ConfigTemplate {
	templates := []*ConfigTemplate{
		{
			Name:        "default",
			Description: "Default AWS configuration",
			Provider:    ProviderAWS,
			Config: &ProviderConfig{
				Provider:      ProviderAWS,
				DefaultRegion: "us-east-1",
				EnableCache:   true,
				CacheDuration: 15 * time.Minute,
			},
		},
		{
			Name:        "default",
			Description: "Default Azure configuration",
			Provider:    ProviderAzure,
			Config: &ProviderConfig{
				Provider:      ProviderAzure,
				DefaultRegion: "eastus",
				EnableCache:   true,
				CacheDuration: 15 * time.Minute,
			},
		},
		{
			Name:        "default",
			Description: "Default GCP configuration",
			Provider:    ProviderGCP,
			Config: &ProviderConfig{
				Provider:      ProviderGCP,
				DefaultRegion: "us-central1",
				EnableCache:   true,
				CacheDuration: 15 * time.Minute,
			},
		},
	}

	return templates
}
