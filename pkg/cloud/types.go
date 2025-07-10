package cloud

import (
	"context"
	"time"
)

// Provider represents a cloud provider type
type Provider string

const (
	ProviderAWS   Provider = "aws"
	ProviderAzure Provider = "azure"
	ProviderGCP   Provider = "gcp"
)

// AuthMethod represents the authentication method
type AuthMethod string

const (
	AuthMethodCLI              AuthMethod = "cli"
	AuthMethodSDK              AuthMethod = "sdk"
	AuthMethodIAMRole          AuthMethod = "iam-role"
	AuthMethodAccessKey        AuthMethod = "access-key"
	AuthMethodServiceKey       AuthMethod = "service-key"
	AuthMethodBrowser          AuthMethod = "browser"
	AuthMethodDeviceCode       AuthMethod = "device-code"
	AuthMethodManagedIdentity  AuthMethod = "managed-identity"
	AuthMethodServicePrincipal AuthMethod = "service-principal"
)

// CLIStatus represents the status of a cloud CLI
type CLIStatus struct {
	Installed   bool   `json:"installed"`
	Version     string `json:"version"`
	Path        string `json:"path"`
	ConfigPath  string `json:"config_path"`
	MinVersion  string `json:"min_version"`
	IsSupported bool   `json:"is_supported"`
}

// Credentials represents cloud provider credentials
type Credentials struct {
	Provider   Provider          `json:"provider"`
	AuthMethod AuthMethod        `json:"auth_method"`
	Profile    string            `json:"profile,omitempty"`
	AccessKey  string            `json:"access_key,omitempty"`
	SecretKey  string            `json:"secret_key,omitempty"`
	Token      string            `json:"token,omitempty"`
	Region     string            `json:"region,omitempty"`
	Account    string            `json:"account,omitempty"`
	Expiry     *time.Time        `json:"expiry,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
}

// Registry represents a container registry
type Registry struct {
	Provider Provider `json:"provider"`
	Name     string   `json:"name"`
	URL      string   `json:"url"`
	Region   string   `json:"region"`
	Type     string   `json:"type"` // ECR, ACR, GCR
}

// Cluster represents a Kubernetes cluster
type Cluster struct {
	Provider   Provider          `json:"provider"`
	Name       string            `json:"name"`
	Region     string            `json:"region"`
	Type       string            `json:"type"` // EKS, AKS, GKE
	Version    string            `json:"version"`
	Endpoint   string            `json:"endpoint"`
	NodeCount  int               `json:"node_count"`
	Status     string            `json:"status"`
	Labels     map[string]string `json:"labels,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
}

// CloudResource represents a generic cloud resource
type CloudResource struct {
	Provider   Provider          `json:"provider"`
	Type       string            `json:"type"`
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Region     string            `json:"region"`
	Status     string            `json:"status"`
	CreatedAt  time.Time         `json:"created_at"`
	Tags       map[string]string `json:"tags,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
}

// Logger interface for logging
type Logger func(string)

// ProviderConfig holds configuration for a cloud provider
type ProviderConfig struct {
	Provider        Provider          `json:"provider"`
	DefaultRegion   string            `json:"default_region"`
	DefaultProfile  string            `json:"default_profile,omitempty"`
	CLIPath         string            `json:"cli_path,omitempty"`
	ConfigPath      string            `json:"config_path,omitempty"`
	EnableCache     bool              `json:"enable_cache"`
	CacheDuration   time.Duration     `json:"cache_duration"`
	CustomEndpoints map[string]string `json:"custom_endpoints,omitempty"`
	Logger          Logger            `json:"-"` // Logger function for debugging
}

// CloudProvider interface for all cloud providers
type CloudProvider interface {
	// Provider info
	Name() Provider
	ValidateAuth(ctx context.Context) error
	GetCredentials() (*Credentials, error)

	// CLI operations
	DetectCLI() (*CLIStatus, error)
	ValidateCLI() error
	GetCLIVersion() (string, error)

	// Registry operations
	ListRegistries(ctx context.Context) ([]*Registry, error)
	GetRegistry(ctx context.Context, name string) (*Registry, error)
	AuthenticateRegistry(ctx context.Context, registry *Registry) error

	// Cluster operations
	ListClusters(ctx context.Context) ([]*Cluster, error)
	GetCluster(ctx context.Context, name string) (*Cluster, error)
	GetKubeconfig(ctx context.Context, cluster *Cluster) ([]byte, error)

	// Region operations
	ListRegions(ctx context.Context) ([]string, error)
	GetCurrentRegion() string
	SetRegion(region string) error
}

// CLIDetector interface for detecting cloud CLIs
type CLIDetector interface {
	Detect() (*CLIStatus, error)
	ValidateVersion(version string) bool
	GetMinVersion() string
	GetInstallInstructions() string
}

// CredentialManager interface for managing credentials
type CredentialManager interface {
	Store(credentials *Credentials) error
	Retrieve(provider Provider, profile string) (*Credentials, error)
	Delete(provider Provider, profile string) error
	List(provider Provider) ([]*Credentials, error)
	Rotate(credentials *Credentials) (*Credentials, error)
}

// RegistryManager interface for container registry operations
type RegistryManager interface {
	Authenticate(ctx context.Context, registry *Registry) error
	PushImage(ctx context.Context, registry *Registry, image string) error
	PullImage(ctx context.Context, registry *Registry, image string) error
	ListImages(ctx context.Context, registry *Registry) ([]string, error)
	DeleteImage(ctx context.Context, registry *Registry, image string) error
}

// ClusterManager interface for Kubernetes cluster operations
type ClusterManager interface {
	Connect(ctx context.Context, cluster *Cluster) error
	Disconnect() error
	GetNodes(ctx context.Context) ([]string, error)
	GetNamespaces(ctx context.Context) ([]string, error)
	DeployHelm(ctx context.Context, chart string, namespace string) error
}

// APIFallback interface for API-based operations when CLI is not available
type APIFallback interface {
	IsAvailable() bool
	ListClustersViaAPI(ctx context.Context) ([]*Cluster, error)
	ListRegistriesViaAPI(ctx context.Context) ([]*Registry, error)
	GetCredentialsViaAPI(ctx context.Context) (*Credentials, error)
}

// PlatformCompatibility represents platform-specific compatibility
type PlatformCompatibility struct {
	OS              string   `json:"os"`
	Arch            string   `json:"arch"`
	CLICommand      string   `json:"cli_command"`
	ConfigLocations []string `json:"config_locations"`
	EnvVars         []string `json:"env_vars"`
}

// ValidationResult represents the result of a validation check
type ValidationResult struct {
	Valid    bool              `json:"valid"`
	Errors   []string          `json:"errors,omitempty"`
	Warnings []string          `json:"warnings,omitempty"`
	Details  map[string]string `json:"details,omitempty"`
}

// ProviderDetectionResult represents the result of provider detection
type ProviderDetectionResult struct {
	Provider      Provider               `json:"provider"`
	Available     bool                   `json:"available"`
	Reason        string                 `json:"reason,omitempty"`
	CLIStatus     *CLIStatus             `json:"cli_status,omitempty"`
	Config        *ProviderConfig        `json:"config,omitempty"`
	ConfigError   error                  `json:"config_error,omitempty"`
	AuthStatus    string                 `json:"auth_status,omitempty"`
	Capabilities  []string               `json:"capabilities,omitempty"`
	Compatibility *PlatformCompatibility `json:"compatibility,omitempty"`
	Instructions  string                 `json:"instructions,omitempty"`
}

// Azure-specific types

// AzureSubscription represents an Azure subscription
type AzureSubscription struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	State        string            `json:"state"`
	TenantID     string            `json:"tenant_id"`
	IsDefault    bool              `json:"is_default"`
	CloudName    string            `json:"cloud_name"`
	HomeTenantID string            `json:"home_tenant_id"`
	Tags         map[string]string `json:"tags,omitempty"`
}

// AzureResourceGroup represents an Azure resource group
type AzureResourceGroup struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	Location          string            `json:"location"`
	SubscriptionID    string            `json:"subscription_id"`
	Tags              map[string]string `json:"tags,omitempty"`
	ProvisioningState string            `json:"provisioning_state"`
}

// AzureServicePrincipal represents an Azure service principal
type AzureServicePrincipal struct {
	AppID       string     `json:"app_id"`
	DisplayName string     `json:"display_name"`
	Password    string     `json:"password,omitempty"`
	Tenant      string     `json:"tenant"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	KeyID       string     `json:"key_id,omitempty"`
	Certificate string     `json:"certificate,omitempty"`
}

// AzureStorageAccount represents an Azure storage account
type AzureStorageAccount struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	ResourceGroup     string            `json:"resource_group"`
	Location          string            `json:"location"`
	Kind              string            `json:"kind"`
	AccessTier        string            `json:"access_tier"`
	ProvisioningState string            `json:"provisioning_state"`
	PrimaryEndpoints  map[string]string `json:"primary_endpoints"`
	Tags              map[string]string `json:"tags,omitempty"`
}

// AzureMonitorMetric represents an Azure Monitor metric
type AzureMonitorMetric struct {
	Name       string                   `json:"name"`
	Unit       string                   `json:"unit"`
	Timeseries []AzureMonitorTimeseries `json:"timeseries"`
}

// AzureMonitorTimeseries represents Azure Monitor time series data
type AzureMonitorTimeseries struct {
	MetadataValues []AzureMonitorMetadata  `json:"metadata_values"`
	Data           []AzureMonitorDataPoint `json:"data"`
}

// AzureMonitorMetadata represents Azure Monitor metadata
type AzureMonitorMetadata struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// AzureMonitorDataPoint represents a single data point
type AzureMonitorDataPoint struct {
	TimeStamp time.Time `json:"timestamp"`
	Average   *float64  `json:"average,omitempty"`
	Minimum   *float64  `json:"minimum,omitempty"`
	Maximum   *float64  `json:"maximum,omitempty"`
	Total     *float64  `json:"total,omitempty"`
	Count     *float64  `json:"count,omitempty"`
}

// AzureApplicationInsights represents an Application Insights resource
type AzureApplicationInsights struct {
	ID                 string            `json:"id"`
	Name               string            `json:"name"`
	ResourceGroup      string            `json:"resource_group"`
	Location           string            `json:"location"`
	InstrumentationKey string            `json:"instrumentation_key"`
	ConnectionString   string            `json:"connection_string"`
	ApplicationID      string            `json:"application_id"`
	ApplicationType    string            `json:"application_type"`
	ProvisioningState  string            `json:"provisioning_state"`
	Tags               map[string]string `json:"tags,omitempty"`
}

// AzureKeyVaultSecret represents a Key Vault secret
type AzureKeyVaultSecret struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Value       string            `json:"value,omitempty"`
	ContentType string            `json:"content_type,omitempty"`
	Enabled     bool              `json:"enabled"`
	Created     time.Time         `json:"created"`
	Updated     time.Time         `json:"updated"`
	Expires     *time.Time        `json:"expires,omitempty"`
	NotBefore   *time.Time        `json:"not_before,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
}

// AzureARMTemplate represents an Azure Resource Manager template
type AzureARMTemplate struct {
	Name           string                 `json:"name"`
	ResourceGroup  string                 `json:"resource_group"`
	Template       map[string]interface{} `json:"template"`
	Parameters     map[string]interface{} `json:"parameters"`
	Mode           string                 `json:"mode"` // Incremental or Complete
	DeploymentName string                 `json:"deployment_name"`
}

// DeviceCodeAuth represents device code authentication flow
type DeviceCodeAuth struct {
	DeviceCode      string    `json:"device_code"`
	UserCode        string    `json:"user_code"`
	VerificationURL string    `json:"verification_url"`
	ExpiresIn       int       `json:"expires_in"`
	Interval        int       `json:"interval"`
	Message         string    `json:"message"`
	ExpiresAt       time.Time `json:"expires_at"`
}

// AzureProvider interface extends CloudProvider with Azure-specific operations
type AzureProvider interface {
	CloudProvider

	// Authentication
	AuthenticateInteractive(ctx context.Context) error
	AuthenticateDeviceCode(ctx context.Context) (*DeviceCodeAuth, error)
	AuthenticateServicePrincipal(ctx context.Context, clientID, clientSecret, tenantID string) error
	AuthenticateManagedIdentity(ctx context.Context) error

	// Subscription management
	ListSubscriptions(ctx context.Context) ([]*AzureSubscription, error)
	GetSubscription(ctx context.Context, subscriptionID string) (*AzureSubscription, error)
	SetDefaultSubscription(ctx context.Context, subscriptionID string) error

	// Resource group management
	ListResourceGroups(ctx context.Context) ([]*AzureResourceGroup, error)
	CreateResourceGroup(ctx context.Context, name, location string, tags map[string]string) (*AzureResourceGroup, error)
	DeleteResourceGroup(ctx context.Context, name string) error

	// Service principal management
	CreateServicePrincipal(ctx context.Context, name string) (*AzureServicePrincipal, error)
	ListServicePrincipals(ctx context.Context) ([]*AzureServicePrincipal, error)
	DeleteServicePrincipal(ctx context.Context, appID string) error
	RotateServicePrincipalSecret(ctx context.Context, appID string) (*AzureServicePrincipal, error)

	// Azure Monitor
	GetMonitorMetrics(ctx context.Context, resourceID string, metricNames []string, timespan string) ([]*AzureMonitorMetric, error)
	CreateAlertRule(ctx context.Context, name, resourceGroup string, config map[string]interface{}) error
	ListActionGroups(ctx context.Context, resourceGroup string) ([]map[string]interface{}, error)

	// Application Insights
	CreateApplicationInsights(ctx context.Context, name, resourceGroup, location string) (*AzureApplicationInsights, error)
	ListApplicationInsights(ctx context.Context) ([]*AzureApplicationInsights, error)
	GetApplicationInsights(ctx context.Context, name, resourceGroup string) (*AzureApplicationInsights, error)

	// Storage Account
	ListStorageAccounts(ctx context.Context) ([]*AzureStorageAccount, error)
	CreateStorageAccount(ctx context.Context, name, resourceGroup, location string) (*AzureStorageAccount, error)
	GetStorageAccountKeys(ctx context.Context, name, resourceGroup string) ([]string, error)

	// Key Vault
	ListKeyVaults(ctx context.Context) ([]string, error)
	GetSecret(ctx context.Context, vaultName, secretName string) (*AzureKeyVaultSecret, error)
	SetSecret(ctx context.Context, vaultName, secretName, value string) error
	DeleteSecret(ctx context.Context, vaultName, secretName string) error

	// ARM Templates
	ValidateARMTemplate(ctx context.Context, template *AzureARMTemplate) (*ValidationResult, error)
	DeployARMTemplate(ctx context.Context, template *AzureARMTemplate) (string, error)
	GetDeploymentStatus(ctx context.Context, resourceGroup, deploymentName string) (string, error)
}

// AzureCredentialManager interface for Azure-specific credential operations
type AzureCredentialManager interface {
	CredentialManager
	StoreServicePrincipal(sp *AzureServicePrincipal) error
	RetrieveServicePrincipal(appID string) (*AzureServicePrincipal, error)
	ListServicePrincipals() ([]*AzureServicePrincipal, error)
	DeleteServicePrincipal(appID string) error
	ValidateCredentials(ctx context.Context, creds *Credentials) error
	RefreshToken(ctx context.Context, creds *Credentials) (*Credentials, error)
}
