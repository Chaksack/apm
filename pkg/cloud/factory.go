package cloud

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// ProviderFactory creates cloud providers
type ProviderFactory struct {
	configs   map[Provider]*ProviderConfig
	detectors map[Provider]CLIDetector
	authMgr   AuthManager
	configMgr ConfigManager
	mu        sync.RWMutex
}

// NewProviderFactory creates a new provider factory
func NewProviderFactory() *ProviderFactory {
	factory := &ProviderFactory{
		configs:   make(map[Provider]*ProviderConfig),
		detectors: make(map[Provider]CLIDetector),
	}

	// Initialize CLI detectors
	factory.detectors[ProviderAWS] = NewAWSCLIDetector()
	factory.detectors[ProviderAzure] = NewAzureCLIDetector()
	factory.detectors[ProviderGCP] = NewGCPCLIDetector()

	return factory
}

// NewProviderFactoryWithManagers creates a new provider factory with auth and config managers
func NewProviderFactoryWithManagers(authMgr AuthManager, configMgr ConfigManager) *ProviderFactory {
	factory := NewProviderFactory()
	factory.authMgr = authMgr
	factory.configMgr = configMgr
	return factory
}

// RegisterConfig registers a provider configuration
func (f *ProviderFactory) RegisterConfig(provider Provider, config *ProviderConfig) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.configs[provider] = config
}

// DetectProvider detects if a provider is available and configured
func (f *ProviderFactory) DetectProvider(provider Provider) (*ProviderDetectionResult, error) {
	f.mu.RLock()
	detector, exists := f.detectors[provider]
	f.mu.RUnlock()

	if !exists {
		return &ProviderDetectionResult{
			Provider:  provider,
			Available: false,
			Reason:    "No detector available for provider",
		}, nil
	}

	// Detect CLI
	cliStatus, err := detector.Detect()
	if err != nil {
		return &ProviderDetectionResult{
			Provider:  provider,
			Available: false,
			Reason:    fmt.Sprintf("CLI detection failed: %v", err),
		}, nil
	}

	// Check if CLI is installed and supported
	if !cliStatus.Installed {
		return &ProviderDetectionResult{
			Provider:     provider,
			Available:    false,
			Reason:       "CLI not installed",
			CLIStatus:    cliStatus,
			Instructions: detector.GetInstallInstructions(),
		}, nil
	}

	if !cliStatus.IsSupported {
		return &ProviderDetectionResult{
			Provider:  provider,
			Available: false,
			Reason:    fmt.Sprintf("CLI version %s not supported (minimum: %s)", cliStatus.Version, cliStatus.MinVersion),
			CLIStatus: cliStatus,
		}, nil
	}

	// Load configuration
	var config *ProviderConfig
	var configErr error

	if f.configMgr != nil {
		config, configErr = f.configMgr.LoadConfig(provider)
	} else {
		config = f.getDefaultConfig(provider)
	}

	// Check authentication status
	authStatus := "unknown"
	if f.authMgr != nil {
		if authenticated, err := f.authMgr.IsAuthenticated(context.Background(), provider); err == nil {
			if authenticated {
				authStatus = "authenticated"
			} else {
				authStatus = "not_authenticated"
			}
		}
	}

	return &ProviderDetectionResult{
		Provider:      provider,
		Available:     true,
		CLIStatus:     cliStatus,
		Config:        config,
		ConfigError:   configErr,
		AuthStatus:    authStatus,
		Capabilities:  f.getProviderCapabilities(provider),
		Compatibility: f.getPlatformCompatibility(provider),
	}, nil
}

// DetectAllProviders detects all available providers
func (f *ProviderFactory) DetectAllProviders() map[Provider]*ProviderDetectionResult {
	results := make(map[Provider]*ProviderDetectionResult)

	providers := []Provider{ProviderAWS, ProviderAzure, ProviderGCP}
	for _, provider := range providers {
		result, err := f.DetectProvider(provider)
		if err != nil {
			result = &ProviderDetectionResult{
				Provider:  provider,
				Available: false,
				Reason:    fmt.Sprintf("Detection error: %v", err),
			}
		}
		results[provider] = result
	}

	return results
}

// LoadConfiguration loads configuration for a provider
func (f *ProviderFactory) LoadConfiguration(provider Provider, environment string) (*ProviderConfig, error) {
	if f.configMgr == nil {
		return f.getDefaultConfig(provider), nil
	}

	// Try environment-specific config first
	if environment != "" {
		if config, err := f.configMgr.LoadEnvironmentConfig(provider, environment); err == nil {
			return config, nil
		}
	}

	// Fall back to default config
	return f.configMgr.LoadConfig(provider)
}

// ValidateConfiguration validates provider configuration
func (f *ProviderFactory) ValidateConfiguration(provider Provider, config *ProviderConfig) (*ValidationResult, error) {
	if f.configMgr == nil {
		return &ValidationResult{Valid: true}, nil
	}

	return f.configMgr.ValidateConfig(config)
}

// CreateProvider creates a cloud provider instance
func (f *ProviderFactory) CreateProvider(provider Provider) (CloudProvider, error) {
	f.mu.RLock()
	config := f.configs[provider]
	f.mu.RUnlock()

	// If no config cached, try to load it
	if config == nil && f.configMgr != nil {
		loadedConfig, err := f.configMgr.LoadConfig(provider)
		if err == nil {
			config = loadedConfig
			f.RegisterConfig(provider, config)
		}
	}

	// Use default config if still nil
	if config == nil {
		config = f.getDefaultConfig(provider)
	}

	switch provider {
	case ProviderAWS:
		return NewAWSProvider(config)
	case ProviderAzure:
		return NewAzureProvider(config)
	case ProviderGCP:
		return NewGCPProvider(config)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// getDefaultConfig returns default configuration for a provider
func (f *ProviderFactory) getDefaultConfig(provider Provider) *ProviderConfig {
	config := &ProviderConfig{
		Provider:      provider,
		EnableCache:   true,
		CacheDuration: 15 * time.Minute,
	}

	switch provider {
	case ProviderAWS:
		config.DefaultRegion = "us-east-1"
		config.CLIPath = f.getCLIPath("aws")
	case ProviderAzure:
		config.DefaultRegion = "eastus"
		config.CLIPath = f.getCLIPath("az")
	case ProviderGCP:
		config.DefaultRegion = "us-central1"
		config.CLIPath = f.getCLIPath("gcloud")
	}

	return config
}

// getCLIPath returns the path to a CLI tool
func (f *ProviderFactory) getCLIPath(cliName string) string {
	if runtime.GOOS == "windows" {
		cliName += ".exe"
	}

	// Check common paths
	commonPaths := []string{
		"/usr/local/bin",
		"/usr/bin",
		"/bin",
		"/opt/homebrew/bin",
	}

	if runtime.GOOS == "windows" {
		commonPaths = []string{
			"C:\\Program Files\\Amazon\\AWSCLIV2",
			"C:\\Program Files (x86)\\Microsoft SDKs\\Azure\\CLI2\\wbin",
			"C:\\Program Files (x86)\\Google\\Cloud SDK\\google-cloud-sdk\\bin",
		}
	}

	for _, path := range commonPaths {
		fullPath := filepath.Join(path, cliName)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath
		}
	}

	return cliName // Return bare name, let PATH resolve it
}

// getProviderCapabilities returns capabilities for a provider
func (f *ProviderFactory) getProviderCapabilities(provider Provider) []string {
	switch provider {
	case ProviderAWS:
		return []string{"ecr", "eks", "s3", "iam", "sts", "cloudformation"}
	case ProviderAzure:
		return []string{"acr", "aks", "storage", "keyvault", "arm", "ad"}
	case ProviderGCP:
		return []string{"gcr", "artifact-registry", "gke", "gcs", "iam", "deployment-manager"}
	default:
		return []string{}
	}
}

// getPlatformCompatibility returns platform compatibility info
func (f *ProviderFactory) getPlatformCompatibility(provider Provider) *PlatformCompatibility {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	switch provider {
	case ProviderAWS:
		return &PlatformCompatibility{
			OS:              goos,
			Arch:            goarch,
			CLICommand:      "aws",
			ConfigLocations: f.getAWSConfigLocations(),
			EnvVars:         []string{"AWS_PROFILE", "AWS_REGION", "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"},
		}
	case ProviderAzure:
		return &PlatformCompatibility{
			OS:              goos,
			Arch:            goarch,
			CLICommand:      "az",
			ConfigLocations: f.getAzureConfigLocations(),
			EnvVars:         []string{"AZURE_CLIENT_ID", "AZURE_CLIENT_SECRET", "AZURE_TENANT_ID"},
		}
	case ProviderGCP:
		return &PlatformCompatibility{
			OS:              goos,
			Arch:            goarch,
			CLICommand:      "gcloud",
			ConfigLocations: f.getGCPConfigLocations(),
			EnvVars:         []string{"GOOGLE_APPLICATION_CREDENTIALS", "GCLOUD_PROJECT"},
		}
	default:
		return &PlatformCompatibility{OS: goos, Arch: goarch}
	}
}

// getAWSConfigLocations returns AWS config file locations
func (f *ProviderFactory) getAWSConfigLocations() []string {
	homeDir, _ := os.UserHomeDir()
	if runtime.GOOS == "windows" {
		return []string{
			filepath.Join(homeDir, ".aws", "config"),
			filepath.Join(homeDir, ".aws", "credentials"),
		}
	}
	return []string{
		filepath.Join(homeDir, ".aws", "config"),
		filepath.Join(homeDir, ".aws", "credentials"),
	}
}

// getAzureConfigLocations returns Azure config file locations
func (f *ProviderFactory) getAzureConfigLocations() []string {
	homeDir, _ := os.UserHomeDir()
	if runtime.GOOS == "windows" {
		return []string{
			filepath.Join(homeDir, ".azure", "config"),
			filepath.Join(homeDir, ".azure", "azureProfile.json"),
		}
	}
	return []string{
		filepath.Join(homeDir, ".azure", "config"),
		filepath.Join(homeDir, ".azure", "azureProfile.json"),
	}
}

// getGCPConfigLocations returns GCP config file locations
func (f *ProviderFactory) getGCPConfigLocations() []string {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".config", "gcloud")
	if runtime.GOOS == "windows" {
		configDir = filepath.Join(homeDir, "AppData", "Roaming", "gcloud")
	}
	return []string{
		filepath.Join(configDir, "configurations", "config_default"),
		filepath.Join(configDir, "credentials.db"),
		filepath.Join(configDir, "access_tokens.db"),
	}
}

// CloudManager manages multiple cloud providers
type CloudManager struct {
	factory   *ProviderFactory
	providers map[Provider]CloudProvider
	credMgr   CredentialManager
	mu        sync.RWMutex
}

// NewCloudManager creates a new cloud manager
func NewCloudManager(credentialStorePath string) (*CloudManager, error) {
	credMgr, err := NewSecureCredentialManager(credentialStorePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create credential manager: %w", err)
	}

	return &CloudManager{
		factory:   NewProviderFactory(),
		providers: make(map[Provider]CloudProvider),
		credMgr:   credMgr,
	}, nil
}

// RegisterProvider registers a cloud provider with configuration
func (m *CloudManager) RegisterProvider(provider Provider, config *ProviderConfig) error {
	m.factory.RegisterConfig(provider, config)

	// Create and cache the provider
	p, err := m.factory.CreateProvider(provider)
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.providers[provider] = p
	m.mu.Unlock()

	return nil
}

// GetProvider gets or creates a cloud provider
func (m *CloudManager) GetProvider(provider Provider) (CloudProvider, error) {
	m.mu.RLock()
	p, exists := m.providers[provider]
	m.mu.RUnlock()

	if exists {
		return p, nil
	}

	// Create new provider
	p, err := m.factory.CreateProvider(provider)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.providers[provider] = p
	m.mu.Unlock()

	return p, nil
}

// ValidateEnvironment validates the environment for all registered providers
func (m *CloudManager) ValidateEnvironment(ctx context.Context) map[Provider]*ValidationResult {
	results := make(map[Provider]*ValidationResult)

	providers := []Provider{ProviderAWS, ProviderAzure, ProviderGCP}
	for _, provider := range providers {
		result, err := ValidateCLIEnvironment(provider)
		if err != nil {
			result = &ValidationResult{
				Valid:  false,
				Errors: []string{fmt.Sprintf("Validation error: %v", err)},
			}
		}
		results[provider] = result
	}

	return results
}

// DetectAvailableProviders detects which cloud providers are available
func (m *CloudManager) DetectAvailableProviders(ctx context.Context) []Provider {
	var available []Provider

	providers := []Provider{ProviderAWS, ProviderAzure, ProviderGCP}
	for _, provider := range providers {
		p, err := m.GetProvider(provider)
		if err != nil {
			continue
		}

		// Check if CLI is installed
		if status, err := p.DetectCLI(); err == nil && status.Installed {
			available = append(available, provider)
		}
	}

	return available
}

// GetCredentials gets credentials for a provider
func (m *CloudManager) GetCredentials(provider Provider, profile string) (*Credentials, error) {
	// First try to get from credential manager
	creds, err := m.credMgr.Retrieve(provider, profile)
	if err == nil {
		return creds, nil
	}

	// If not found, try to get from provider
	p, err := m.GetProvider(provider)
	if err != nil {
		return nil, err
	}

	return p.GetCredentials()
}

// StoreCredentials stores credentials for a provider
func (m *CloudManager) StoreCredentials(creds *Credentials) error {
	return m.credMgr.Store(creds)
}

// ListAllClusters lists clusters across all providers
func (m *CloudManager) ListAllClusters(ctx context.Context) (map[Provider][]*Cluster, error) {
	results := make(map[Provider][]*Cluster)
	errors := make(map[Provider]error)

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, provider := range m.DetectAvailableProviders(ctx) {
		wg.Add(1)
		go func(p Provider) {
			defer wg.Done()

			cloudProvider, err := m.GetProvider(p)
			if err != nil {
				mu.Lock()
				errors[p] = err
				mu.Unlock()
				return
			}

			clusters, err := cloudProvider.ListClusters(ctx)
			mu.Lock()
			if err != nil {
				errors[p] = err
			} else {
				results[p] = clusters
			}
			mu.Unlock()
		}(provider)
	}

	wg.Wait()

	// Check if all providers failed
	if len(errors) == len(m.DetectAvailableProviders(ctx)) && len(errors) > 0 {
		return nil, fmt.Errorf("failed to list clusters from any provider")
	}

	return results, nil
}

// ListAllRegistries lists registries across all providers
func (m *CloudManager) ListAllRegistries(ctx context.Context) (map[Provider][]*Registry, error) {
	results := make(map[Provider][]*Registry)
	errors := make(map[Provider]error)

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, provider := range m.DetectAvailableProviders(ctx) {
		wg.Add(1)
		go func(p Provider) {
			defer wg.Done()

			cloudProvider, err := m.GetProvider(p)
			if err != nil {
				mu.Lock()
				errors[p] = err
				mu.Unlock()
				return
			}

			registries, err := cloudProvider.ListRegistries(ctx)
			mu.Lock()
			if err != nil {
				errors[p] = err
			} else {
				results[p] = registries
			}
			mu.Unlock()
		}(provider)
	}

	wg.Wait()

	// Check if all providers failed
	if len(errors) == len(m.DetectAvailableProviders(ctx)) && len(errors) > 0 {
		return nil, fmt.Errorf("failed to list registries from any provider")
	}

	return results, nil
}

// MultiCloudOperations provides operations across multiple cloud providers
type MultiCloudOperations struct {
	manager *CloudManager
}

// NewMultiCloudOperations creates a new multi-cloud operations instance
func NewMultiCloudOperations(manager *CloudManager) *MultiCloudOperations {
	return &MultiCloudOperations{
		manager: manager,
	}
}

// FindCluster finds a cluster by name across all providers
func (o *MultiCloudOperations) FindCluster(ctx context.Context, name string) (*Cluster, Provider, error) {
	for _, provider := range o.manager.DetectAvailableProviders(ctx) {
		p, err := o.manager.GetProvider(provider)
		if err != nil {
			continue
		}

		cluster, err := p.GetCluster(ctx, name)
		if err == nil {
			return cluster, provider, nil
		}
	}

	return nil, "", fmt.Errorf("cluster %s not found in any provider", name)
}

// FindRegistry finds a registry by name across all providers
func (o *MultiCloudOperations) FindRegistry(ctx context.Context, name string) (*Registry, Provider, error) {
	for _, provider := range o.manager.DetectAvailableProviders(ctx) {
		p, err := o.manager.GetProvider(provider)
		if err != nil {
			continue
		}

		registry, err := p.GetRegistry(ctx, name)
		if err == nil {
			return registry, provider, nil
		}
	}

	return nil, "", fmt.Errorf("registry %s not found in any provider", name)
}

// AuthenticateAllRegistries authenticates to all available registries
func (o *MultiCloudOperations) AuthenticateAllRegistries(ctx context.Context) error {
	registries, err := o.manager.ListAllRegistries(ctx)
	if err != nil {
		return err
	}

	var errors []error
	for provider, regs := range registries {
		p, err := o.manager.GetProvider(provider)
		if err != nil {
			errors = append(errors, fmt.Errorf("%s: %w", provider, err))
			continue
		}

		for _, registry := range regs {
			if err := p.AuthenticateRegistry(ctx, registry); err != nil {
				errors = append(errors, fmt.Errorf("%s/%s: %w", provider, registry.Name, err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("authentication errors: %v", errors)
	}

	return nil
}
