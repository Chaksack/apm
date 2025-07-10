package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// GCPProvider implements CloudProvider for Google Cloud
type GCPProvider struct {
	config      *ProviderConfig
	credentials *Credentials
	cliStatus   *CLIStatus
	cache       *CredentialCache
	projectID   string
	region      string
	zone        string
}

// NewGCPProvider creates a new GCP provider
func NewGCPProvider(config *ProviderConfig) (*GCPProvider, error) {
	if config == nil {
		config = &ProviderConfig{
			Provider:      ProviderGCP,
			DefaultRegion: "us-central1",
			EnableCache:   true,
			CacheDuration: 5 * time.Minute,
		}
	}

	return &GCPProvider{
		config: config,
		cache:  NewCredentialCache(config.CacheDuration),
	}, nil
}

// Name returns the provider name
func (p *GCPProvider) Name() Provider {
	return ProviderGCP
}

// DetectCLI detects Google Cloud CLI installation
func (p *GCPProvider) DetectCLI() (*CLIStatus, error) {
	detector := NewGCPCLIDetector()
	status, err := detector.Detect()
	if err != nil {
		return nil, err
	}
	p.cliStatus = status
	return status, nil
}

// ValidateCLI validates Google Cloud CLI is properly configured
func (p *GCPProvider) ValidateCLI() error {
	if p.cliStatus == nil {
		if _, err := p.DetectCLI(); err != nil {
			return err
		}
	}

	if !p.cliStatus.Installed {
		return fmt.Errorf("Google Cloud CLI not installed")
	}

	// Check if authenticated
	cmd := exec.Command("gcloud", "auth", "list", "--filter=status:ACTIVE", "--format=json")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("Google Cloud CLI not authenticated: run 'gcloud auth login'")
	}

	var accounts []interface{}
	if err := json.Unmarshal(output, &accounts); err != nil || len(accounts) == 0 {
		return fmt.Errorf("no active Google Cloud authentication found")
	}

	return nil
}

// GetCLIVersion returns the Google Cloud CLI version
func (p *GCPProvider) GetCLIVersion() (string, error) {
	if p.cliStatus == nil {
		if _, err := p.DetectCLI(); err != nil {
			return "", err
		}
	}
	return p.cliStatus.Version, nil
}

// ValidateAuth validates GCP authentication
func (p *GCPProvider) ValidateAuth(ctx context.Context) error {
	// Get active account
	cmd := exec.Command("gcloud", "config", "get-value", "account")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get active account: %w", err)
	}
	account := strings.TrimSpace(string(output))

	// Get project
	cmd = exec.Command("gcloud", "config", "get-value", "project")
	output, err = cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get active project: %w", err)
	}
	project := strings.TrimSpace(string(output))

	if account == "" || project == "" {
		return fmt.Errorf("no active account or project configured")
	}

	// Store account info in credentials
	if p.credentials == nil {
		p.credentials = &Credentials{
			Provider:   ProviderGCP,
			AuthMethod: AuthMethodCLI,
		}
	}
	p.credentials.Account = account
	if p.credentials.Properties == nil {
		p.credentials.Properties = make(map[string]string)
	}
	p.credentials.Properties["project"] = project

	return nil
}

// GetCredentials returns current GCP credentials
func (p *GCPProvider) GetCredentials() (*Credentials, error) {
	if p.credentials != nil {
		return p.credentials, nil
	}

	// Try to get from environment
	if keyFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); keyFile != "" {
		p.credentials = &Credentials{
			Provider:   ProviderGCP,
			AuthMethod: AuthMethodServiceKey,
			Properties: map[string]string{
				"key_file": keyFile,
				"project":  os.Getenv("GOOGLE_CLOUD_PROJECT"),
			},
		}
		return p.credentials, nil
	}

	// Get from CLI
	p.credentials = &Credentials{
		Provider:   ProviderGCP,
		AuthMethod: AuthMethodCLI,
		Region:     p.GetCurrentRegion(),
	}

	// Get account and project
	cmd := exec.Command("gcloud", "config", "get-value", "account")
	if output, err := cmd.Output(); err == nil {
		p.credentials.Account = strings.TrimSpace(string(output))
	}

	cmd = exec.Command("gcloud", "config", "get-value", "project")
	if output, err := cmd.Output(); err == nil {
		if p.credentials.Properties == nil {
			p.credentials.Properties = make(map[string]string)
		}
		p.credentials.Properties["project"] = strings.TrimSpace(string(output))
	}

	return p.credentials, nil
}

// ListRegistries lists GCR registries
func (p *GCPProvider) ListRegistries(ctx context.Context) ([]*Registry, error) {
	// Get current project
	cmd := exec.Command("gcloud", "config", "get-value", "project")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get current project: %w", err)
	}
	project := strings.TrimSpace(string(output))

	// GCR has predefined registry locations
	registryLocations := []struct {
		name   string
		url    string
		region string
	}{
		{"gcr.io", fmt.Sprintf("gcr.io/%s", project), "global"},
		{"us.gcr.io", fmt.Sprintf("us.gcr.io/%s", project), "us"},
		{"eu.gcr.io", fmt.Sprintf("eu.gcr.io/%s", project), "eu"},
		{"asia.gcr.io", fmt.Sprintf("asia.gcr.io/%s", project), "asia"},
	}

	registries := make([]*Registry, 0, len(registryLocations))
	for _, loc := range registryLocations {
		registries = append(registries, &Registry{
			Provider: ProviderGCP,
			Name:     loc.name,
			URL:      loc.url,
			Region:   loc.region,
			Type:     "GCR",
		})
	}

	// Also check for Artifact Registry repositories
	cmd = exec.Command("gcloud", "artifacts", "repositories", "list", "--format=json")
	if output, err := cmd.Output(); err == nil {
		var repos []struct {
			Name        string `json:"name"`
			Location    string `json:"location"`
			Format      string `json:"format"`
			Description string `json:"description"`
		}
		if json.Unmarshal(output, &repos) == nil {
			for _, repo := range repos {
				if repo.Format == "DOCKER" {
					// Extract repository name from full name
					parts := strings.Split(repo.Name, "/")
					repoName := parts[len(parts)-1]

					registries = append(registries, &Registry{
						Provider: ProviderGCP,
						Name:     repoName,
						URL:      fmt.Sprintf("%s-docker.pkg.dev/%s/%s", repo.Location, project, repoName),
						Region:   repo.Location,
						Type:     "Artifact Registry",
					})
				}
			}
		}
	}

	return registries, nil
}

// GetRegistry gets a specific GCR registry
func (p *GCPProvider) GetRegistry(ctx context.Context, name string) (*Registry, error) {
	registries, err := p.ListRegistries(ctx)
	if err != nil {
		return nil, err
	}

	for _, registry := range registries {
		if registry.Name == name {
			return registry, nil
		}
	}

	return nil, fmt.Errorf("registry %s not found", name)
}

// AuthenticateRegistry authenticates to GCR/Artifact Registry
func (p *GCPProvider) AuthenticateRegistry(ctx context.Context, registry *Registry) error {
	// Determine registry type and configure accordingly
	if registry.Type == "Artifact Registry" {
		// Configure Docker for Artifact Registry
		region := registry.Region
		cmd := exec.Command("gcloud", "auth", "configure-docker", fmt.Sprintf("%s-docker.pkg.dev", region))
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to configure Docker authentication for Artifact Registry: %w", err)
		}
	} else {
		// Configure Docker for GCR
		gcrHosts := []string{"gcr.io", "us.gcr.io", "eu.gcr.io", "asia.gcr.io"}
		for _, host := range gcrHosts {
			cmd := exec.Command("gcloud", "auth", "configure-docker", host)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to configure Docker authentication for %s: %w", host, err)
			}
		}
	}

	// Verify authentication by getting access token
	cmd := exec.Command("gcloud", "auth", "print-access-token")
	if _, err := cmd.Output(); err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	return nil
}

// ListClusters lists GKE clusters
func (p *GCPProvider) ListClusters(ctx context.Context) ([]*Cluster, error) {
	cmd := exec.Command("gcloud", "container", "clusters", "list", "--format=json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	var gkeClusters []struct {
		Name                 string            `json:"name"`
		Location             string            `json:"location"`
		Status               string            `json:"status"`
		CurrentMasterVersion string            `json:"currentMasterVersion"`
		Endpoint             string            `json:"endpoint"`
		CurrentNodeCount     int               `json:"currentNodeCount"`
		ResourceLabels       map[string]string `json:"resourceLabels"`
		CreateTime           string            `json:"createTime"`
	}

	if err := json.Unmarshal(output, &gkeClusters); err != nil {
		return nil, fmt.Errorf("failed to parse clusters: %w", err)
	}

	clusters := make([]*Cluster, 0, len(gkeClusters))
	for _, gke := range gkeClusters {
		clusters = append(clusters, &Cluster{
			Provider:  ProviderGCP,
			Name:      gke.Name,
			Region:    gke.Location,
			Type:      "GKE",
			Version:   gke.CurrentMasterVersion,
			Endpoint:  gke.Endpoint,
			NodeCount: gke.CurrentNodeCount,
			Status:    gke.Status,
			Labels:    gke.ResourceLabels,
		})
	}

	return clusters, nil
}

// GetCluster gets details of a GKE cluster
func (p *GCPProvider) GetCluster(ctx context.Context, name string) (*Cluster, error) {
	// First, try to find the cluster in any zone/region
	cmd := exec.Command("gcloud", "container", "clusters", "list",
		"--filter", fmt.Sprintf("name=%s", name),
		"--format=json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to find cluster: %w", err)
	}

	var gkeClusters []struct {
		Name                 string            `json:"name"`
		Location             string            `json:"location"`
		Status               string            `json:"status"`
		CurrentMasterVersion string            `json:"currentMasterVersion"`
		Endpoint             string            `json:"endpoint"`
		CurrentNodeCount     int               `json:"currentNodeCount"`
		ResourceLabels       map[string]string `json:"resourceLabels"`
		CreateTime           string            `json:"createTime"`
	}

	if err := json.Unmarshal(output, &gkeClusters); err != nil {
		return nil, fmt.Errorf("failed to parse cluster: %w", err)
	}

	if len(gkeClusters) == 0 {
		return nil, fmt.Errorf("cluster %s not found", name)
	}

	gke := gkeClusters[0]
	return &Cluster{
		Provider:  ProviderGCP,
		Name:      gke.Name,
		Region:    gke.Location,
		Type:      "GKE",
		Version:   gke.CurrentMasterVersion,
		Endpoint:  gke.Endpoint,
		NodeCount: gke.CurrentNodeCount,
		Status:    gke.Status,
		Labels:    gke.ResourceLabels,
	}, nil
}

// GetKubeconfig gets kubeconfig for a GKE cluster
func (p *GCPProvider) GetKubeconfig(ctx context.Context, cluster *Cluster) ([]byte, error) {
	// Get cluster credentials
	cmd := exec.Command("gcloud", "container", "clusters", "get-credentials",
		cluster.Name,
		"--location", cluster.Region,
		"--format=json")

	// This command updates the kubeconfig file, we need to extract it
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get cluster credentials: %w", err)
	}

	// Read the kubeconfig
	home, _ := os.UserHomeDir()
	kubeconfigPath := filepath.Join(home, ".kube", "config")

	kubeconfig, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read kubeconfig: %w", err)
	}

	return kubeconfig, nil
}

// ListRegions lists GCP regions
func (p *GCPProvider) ListRegions(ctx context.Context) ([]string, error) {
	cmd := exec.Command("gcloud", "compute", "regions", "list", "--format=value(name)")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list regions: %w", err)
	}

	regions := strings.Split(strings.TrimSpace(string(output)), "\n")
	return regions, nil
}

// GetCurrentRegion gets the current GCP region
func (p *GCPProvider) GetCurrentRegion() string {
	// Check config
	if p.config.DefaultRegion != "" {
		return p.config.DefaultRegion
	}

	// Try to get from gcloud config
	cmd := exec.Command("gcloud", "config", "get-value", "compute/region")
	if output, err := cmd.Output(); err == nil {
		if region := strings.TrimSpace(string(output)); region != "" {
			return region
		}
	}

	// Default to us-central1
	return "us-central1"
}

// SetRegion sets the GCP region
func (p *GCPProvider) SetRegion(region string) error {
	p.config.DefaultRegion = region
	p.region = region

	// Also set in gcloud config
	cmd := exec.Command("gcloud", "config", "set", "compute/region", region)
	return cmd.Run()
}

// GetAdvancedOperations returns the advanced operations manager
func (p *GCPProvider) GetAdvancedOperations() *GCPAdvancedOperations {
	return NewGCPAdvancedOperations(p)
}

// SetProjectID sets the current project ID
func (p *GCPProvider) SetProjectID(projectID string) error {
	p.projectID = projectID

	// Also set in gcloud config
	cmd := exec.Command("gcloud", "config", "set", "project", projectID)
	return cmd.Run()
}

// GetProjectID returns the current project ID
func (p *GCPProvider) GetProjectID() string {
	if p.projectID == "" {
		// Try to get from gcloud config
		cmd := exec.Command("gcloud", "config", "get-value", "project")
		if output, err := cmd.Output(); err == nil {
			p.projectID = strings.TrimSpace(string(output))
		}
	}
	return p.projectID
}

// GetZone returns the current zone
func (p *GCPProvider) GetZone() string {
	if p.zone == "" {
		// Try to get from gcloud config
		cmd := exec.Command("gcloud", "config", "get-value", "compute/zone")
		if output, err := cmd.Output(); err == nil {
			p.zone = strings.TrimSpace(string(output))
		}
	}
	return p.zone
}

// SetZone sets the current zone
func (p *GCPProvider) SetZone(zone string) error {
	p.zone = zone

	// Also set in gcloud config
	cmd := exec.Command("gcloud", "config", "set", "compute/zone", zone)
	return cmd.Run()
}

// ValidateGCloudCLI validates that gcloud CLI is properly set up
func (p *GCPProvider) ValidateGCloudCLI() error {
	// Check if gcloud is installed
	if _, err := exec.LookPath("gcloud"); err != nil {
		return fmt.Errorf("gcloud CLI not found: %w", err)
	}

	// Check version
	cmd := exec.Command("gcloud", "version", "--format=json")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get gcloud version: %w", err)
	}

	var versionInfo struct {
		GoogleCloudSDK string `json:"Google Cloud SDK"`
	}
	if err := json.Unmarshal(output, &versionInfo); err != nil {
		return fmt.Errorf("failed to parse gcloud version: %w", err)
	}

	// Update CLI status
	p.cliStatus = &CLIStatus{
		Installed:   true,
		Version:     versionInfo.GoogleCloudSDK,
		Path:        "gcloud",
		IsSupported: true,
	}

	return nil
}

// EnableRequiredAPIs enables all APIs required for APM integration
func (p *GCPProvider) EnableRequiredAPIs(ctx context.Context) error {
	requiredAPIs := []string{
		"compute.googleapis.com",
		"container.googleapis.com",
		"containerregistry.googleapis.com",
		"artifactregistry.googleapis.com",
		"monitoring.googleapis.com",
		"cloudtrace.googleapis.com",
		"logging.googleapis.com",
		"iam.googleapis.com",
		"cloudresourcemanager.googleapis.com",
		"storage.googleapis.com",
	}

	for _, api := range requiredAPIs {
		cmd := exec.Command("gcloud", "services", "enable", api)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to enable API %s: %w", api, err)
		}
	}

	return nil
}

// GCPAPIFallback provides API-based operations when CLI is not available
type GCPAPIFallback struct {
	provider *GCPProvider
}

// NewGCPAPIFallback creates a new GCP API fallback
func NewGCPAPIFallback(provider *GCPProvider) *GCPAPIFallback {
	return &GCPAPIFallback{
		provider: provider,
	}
}

// IsAvailable checks if API fallback is available
func (f *GCPAPIFallback) IsAvailable() bool {
	// Check if GCP SDK credentials are available
	creds, err := f.provider.GetCredentials()
	if err != nil {
		return false
	}
	return creds.AuthMethod == AuthMethodServiceKey
}

// ListClustersViaAPI lists clusters using Google Cloud SDK
func (f *GCPAPIFallback) ListClustersViaAPI(ctx context.Context) ([]*Cluster, error) {
	// This would use Google Cloud SDK for Go
	// For now, return an error indicating SDK implementation needed
	return nil, fmt.Errorf("Google Cloud SDK implementation required for API fallback")
}

// ListRegistriesViaAPI lists registries using Google Cloud SDK
func (f *GCPAPIFallback) ListRegistriesViaAPI(ctx context.Context) ([]*Registry, error) {
	// This would use Google Cloud SDK for Go
	// For now, return an error indicating SDK implementation needed
	return nil, fmt.Errorf("Google Cloud SDK implementation required for API fallback")
}

// GetCredentialsViaAPI gets credentials using Google Cloud SDK
func (f *GCPAPIFallback) GetCredentialsViaAPI(ctx context.Context) (*Credentials, error) {
	// This would use Google Cloud SDK for Go
	// For now, return an error indicating SDK implementation needed
	return nil, fmt.Errorf("Google Cloud SDK implementation required for API fallback")
}

// ServiceAccount represents a GCP service account
type ServiceAccount struct {
	Email          string            `json:"email"`
	Name           string            `json:"name"`
	DisplayName    string            `json:"displayName"`
	Description    string            `json:"description"`
	ProjectID      string            `json:"projectId"`
	UniqueID       string            `json:"uniqueId"`
	Disabled       bool              `json:"disabled"`
	OAuth2ClientID string            `json:"oauth2ClientId"`
	Tags           map[string]string `json:"tags,omitempty"`
}

// ServiceAccountKey represents a GCP service account key
type ServiceAccountKey struct {
	Name                string `json:"name"`
	PrivateKeyType      string `json:"privateKeyType"`
	PrivateKeyData      string `json:"privateKeyData"`
	PublicKeyData       string `json:"publicKeyData"`
	ValidAfterTime      string `json:"validAfterTime"`
	ValidBeforeTime     string `json:"validBeforeTime"`
	KeyAlgorithm        string `json:"keyAlgorithm"`
	KeyOrigin           string `json:"keyOrigin"`
	KeyType             string `json:"keyType"`
	ServiceAccountEmail string `json:"serviceAccountEmail"`
}

// GCPServiceAccountManager manages GCP service accounts
type GCPServiceAccountManager struct {
	provider *GCPProvider
}

// NewGCPServiceAccountManager creates a new service account manager
func NewGCPServiceAccountManager(provider *GCPProvider) *GCPServiceAccountManager {
	return &GCPServiceAccountManager{
		provider: provider,
	}
}

// ListServiceAccounts lists all service accounts in the project
func (m *GCPServiceAccountManager) ListServiceAccounts(ctx context.Context) ([]*ServiceAccount, error) {
	cmd := exec.Command("gcloud", "iam", "service-accounts", "list", "--format=json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list service accounts: %w", err)
	}

	var serviceAccounts []*ServiceAccount
	if err := json.Unmarshal(output, &serviceAccounts); err != nil {
		return nil, fmt.Errorf("failed to parse service accounts: %w", err)
	}

	return serviceAccounts, nil
}

// GetServiceAccount gets details of a specific service account
func (m *GCPServiceAccountManager) GetServiceAccount(ctx context.Context, email string) (*ServiceAccount, error) {
	cmd := exec.Command("gcloud", "iam", "service-accounts", "describe", email, "--format=json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to describe service account: %w", err)
	}

	var serviceAccount *ServiceAccount
	if err := json.Unmarshal(output, &serviceAccount); err != nil {
		return nil, fmt.Errorf("failed to parse service account: %w", err)
	}

	return serviceAccount, nil
}

// CreateServiceAccount creates a new service account
func (m *GCPServiceAccountManager) CreateServiceAccount(ctx context.Context, accountID, displayName, description string) (*ServiceAccount, error) {
	cmd := exec.Command("gcloud", "iam", "service-accounts", "create", accountID,
		"--display-name", displayName,
		"--description", description,
		"--format=json")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to create service account: %w", err)
	}

	var serviceAccount *ServiceAccount
	if err := json.Unmarshal(output, &serviceAccount); err != nil {
		return nil, fmt.Errorf("failed to parse created service account: %w", err)
	}

	return serviceAccount, nil
}

// CreateServiceAccountKey creates a new key for a service account
func (m *GCPServiceAccountManager) CreateServiceAccountKey(ctx context.Context, serviceAccountEmail, keyFilePath string) (*ServiceAccountKey, error) {
	cmd := exec.Command("gcloud", "iam", "service-accounts", "keys", "create", keyFilePath,
		"--iam-account", serviceAccountEmail,
		"--format=json")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to create service account key: %w", err)
	}

	var key *ServiceAccountKey
	if err := json.Unmarshal(output, &key); err != nil {
		return nil, fmt.Errorf("failed to parse created key: %w", err)
	}

	return key, nil
}

// ListServiceAccountKeys lists all keys for a service account
func (m *GCPServiceAccountManager) ListServiceAccountKeys(ctx context.Context, serviceAccountEmail string) ([]*ServiceAccountKey, error) {
	cmd := exec.Command("gcloud", "iam", "service-accounts", "keys", "list",
		"--iam-account", serviceAccountEmail,
		"--format=json")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list service account keys: %w", err)
	}

	var keys []*ServiceAccountKey
	if err := json.Unmarshal(output, &keys); err != nil {
		return nil, fmt.Errorf("failed to parse service account keys: %w", err)
	}

	return keys, nil
}

// DeleteServiceAccountKey deletes a service account key
func (m *GCPServiceAccountManager) DeleteServiceAccountKey(ctx context.Context, serviceAccountEmail, keyName string) error {
	cmd := exec.Command("gcloud", "iam", "service-accounts", "keys", "delete", keyName,
		"--iam-account", serviceAccountEmail,
		"--quiet")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete service account key: %w", err)
	}

	return nil
}

// SetServiceAccountPolicy sets IAM policy for a service account
func (m *GCPServiceAccountManager) SetServiceAccountPolicy(ctx context.Context, serviceAccountEmail, policyFile string) error {
	cmd := exec.Command("gcloud", "iam", "service-accounts", "set-iam-policy", serviceAccountEmail, policyFile)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set service account policy: %w", err)
	}
	return nil
}

// GetServiceAccountPolicy gets IAM policy for a service account
func (m *GCPServiceAccountManager) GetServiceAccountPolicy(ctx context.Context, serviceAccountEmail string) (string, error) {
	cmd := exec.Command("gcloud", "iam", "service-accounts", "get-iam-policy", serviceAccountEmail, "--format=json")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get service account policy: %w", err)
	}
	return string(output), nil
}

// GCPResourceManager manages GCP projects and resources
type GCPResourceManager struct {
	provider *GCPProvider
}

// NewGCPResourceManager creates a new resource manager
func NewGCPResourceManager(provider *GCPProvider) *GCPResourceManager {
	return &GCPResourceManager{
		provider: provider,
	}
}

// Project represents a GCP project
type Project struct {
	ProjectID      string            `json:"projectId"`
	Name           string            `json:"name"`
	ProjectNumber  string            `json:"projectNumber"`
	LifecycleState string            `json:"lifecycleState"`
	CreateTime     string            `json:"createTime"`
	Parent         map[string]string `json:"parent,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
}

// ListProjects lists all accessible projects
func (rm *GCPResourceManager) ListProjects(ctx context.Context) ([]*Project, error) {
	cmd := exec.Command("gcloud", "projects", "list", "--format=json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	var projects []*Project
	if err := json.Unmarshal(output, &projects); err != nil {
		return nil, fmt.Errorf("failed to parse projects: %w", err)
	}

	return projects, nil
}

// GetProject gets details of a specific project
func (rm *GCPResourceManager) GetProject(ctx context.Context, projectID string) (*Project, error) {
	cmd := exec.Command("gcloud", "projects", "describe", projectID, "--format=json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to describe project: %w", err)
	}

	var project *Project
	if err := json.Unmarshal(output, &project); err != nil {
		return nil, fmt.Errorf("failed to parse project: %w", err)
	}

	return project, nil
}

// SetCurrentProject sets the current active project
func (rm *GCPResourceManager) SetCurrentProject(ctx context.Context, projectID string) error {
	cmd := exec.Command("gcloud", "config", "set", "project", projectID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set current project: %w", err)
	}

	// Update provider's project ID
	rm.provider.projectID = projectID
	return nil
}

// GetCurrentProject gets the current active project
func (rm *GCPResourceManager) GetCurrentProject(ctx context.Context) (string, error) {
	cmd := exec.Command("gcloud", "config", "get-value", "project")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current project: %w", err)
	}

	projectID := strings.TrimSpace(string(output))
	rm.provider.projectID = projectID
	return projectID, nil
}

// GCPMonitoringManager manages Cloud Monitoring integration
type GCPMonitoringManager struct {
	provider *GCPProvider
}

// NewGCPMonitoringManager creates a new monitoring manager
func NewGCPMonitoringManager(provider *GCPProvider) *GCPMonitoringManager {
	return &GCPMonitoringManager{
		provider: provider,
	}
}

// MonitoringWorkspace represents a Cloud Monitoring workspace
type MonitoringWorkspace struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName"`
	Projects    []string `json:"projects"`
	CreateTime  string   `json:"createTime"`
	UpdateTime  string   `json:"updateTime"`
}

// ListMonitoringWorkspaces lists Cloud Monitoring workspaces
func (mm *GCPMonitoringManager) ListMonitoringWorkspaces(ctx context.Context) ([]*MonitoringWorkspace, error) {
	// Note: This requires the Cloud Monitoring API to be enabled
	cmd := exec.Command("gcloud", "alpha", "monitoring", "workspaces", "list", "--format=json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list monitoring workspaces: %w", err)
	}

	var workspaces []*MonitoringWorkspace
	if err := json.Unmarshal(output, &workspaces); err != nil {
		return nil, fmt.Errorf("failed to parse monitoring workspaces: %w", err)
	}

	return workspaces, nil
}

// EnableMonitoringAPI enables the Cloud Monitoring API
func (mm *GCPMonitoringManager) EnableMonitoringAPI(ctx context.Context) error {
	cmd := exec.Command("gcloud", "services", "enable", "monitoring.googleapis.com")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable Cloud Monitoring API: %w", err)
	}
	return nil
}

// EnableCloudTraceAPI enables the Cloud Trace API
func (mm *GCPMonitoringManager) EnableCloudTraceAPI(ctx context.Context) error {
	cmd := exec.Command("gcloud", "services", "enable", "cloudtrace.googleapis.com")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable Cloud Trace API: %w", err)
	}
	return nil
}

// GCPStorageManager manages Cloud Storage buckets
type GCPStorageManager struct {
	provider *GCPProvider
}

// NewGCPStorageManager creates a new storage manager
func NewGCPStorageManager(provider *GCPProvider) *GCPStorageManager {
	return &GCPStorageManager{
		provider: provider,
	}
}

// StorageBucket represents a Cloud Storage bucket
type StorageBucket struct {
	Name          string            `json:"name"`
	Location      string            `json:"location"`
	LocationType  string            `json:"locationType"`
	StorageClass  string            `json:"storageClass"`
	TimeCreated   string            `json:"timeCreated"`
	Updated       string            `json:"updated"`
	Versioning    map[string]bool   `json:"versioning,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	ProjectNumber string            `json:"projectNumber"`
}

// ListStorageBuckets lists all Cloud Storage buckets
func (sm *GCPStorageManager) ListStorageBuckets(ctx context.Context) ([]*StorageBucket, error) {
	cmd := exec.Command("gsutil", "ls", "-L", "-b", "gs://")
	output, err := cmd.Output()
	if err != nil {
		// Try alternative method with gcloud
		cmd = exec.Command("gcloud", "storage", "buckets", "list", "--format=json")
		output, err = cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to list storage buckets: %w", err)
		}
	}

	var buckets []*StorageBucket
	if err := json.Unmarshal(output, &buckets); err != nil {
		return nil, fmt.Errorf("failed to parse storage buckets: %w", err)
	}

	return buckets, nil
}

// CreateStorageBucket creates a new Cloud Storage bucket
func (sm *GCPStorageManager) CreateStorageBucket(ctx context.Context, bucketName, location, storageClass string) (*StorageBucket, error) {
	cmd := exec.Command("gcloud", "storage", "buckets", "create",
		fmt.Sprintf("gs://%s", bucketName),
		"--location", location,
		"--default-storage-class", storageClass,
		"--format=json")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to create storage bucket: %w", err)
	}

	var bucket *StorageBucket
	if err := json.Unmarshal(output, &bucket); err != nil {
		return nil, fmt.Errorf("failed to parse created bucket: %w", err)
	}

	return bucket, nil
}

// GetStorageBucket gets details of a specific bucket
func (sm *GCPStorageManager) GetStorageBucket(ctx context.Context, bucketName string) (*StorageBucket, error) {
	cmd := exec.Command("gcloud", "storage", "buckets", "describe",
		fmt.Sprintf("gs://%s", bucketName),
		"--format=json")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to describe storage bucket: %w", err)
	}

	var bucket *StorageBucket
	if err := json.Unmarshal(output, &bucket); err != nil {
		return nil, fmt.Errorf("failed to parse storage bucket: %w", err)
	}

	return bucket, nil
}

// DeleteStorageBucket deletes a Cloud Storage bucket
func (sm *GCPStorageManager) DeleteStorageBucket(ctx context.Context, bucketName string, force bool) error {
	args := []string{"storage", "buckets", "delete", fmt.Sprintf("gs://%s", bucketName)}
	if force {
		args = append(args, "--force")
	}

	cmd := exec.Command("gcloud", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete storage bucket: %w", err)
	}

	return nil
}

// GCPAuthenticationManager manages different authentication methods
type GCPAuthenticationManager struct {
	provider *GCPProvider
}

// NewGCPAuthenticationManager creates a new authentication manager
func NewGCPAuthenticationManager(provider *GCPProvider) *GCPAuthenticationManager {
	return &GCPAuthenticationManager{
		provider: provider,
	}
}

// AuthenticateWithServiceAccount authenticates using a service account key file
func (am *GCPAuthenticationManager) AuthenticateWithServiceAccount(ctx context.Context, keyFilePath string) error {
	// Set environment variable for ADC
	if err := os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", keyFilePath); err != nil {
		return fmt.Errorf("failed to set GOOGLE_APPLICATION_CREDENTIALS: %w", err)
	}

	// Activate service account in gcloud
	cmd := exec.Command("gcloud", "auth", "activate-service-account", "--key-file", keyFilePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to activate service account: %w", err)
	}

	// Update credentials
	am.provider.credentials = &Credentials{
		Provider:   ProviderGCP,
		AuthMethod: AuthMethodServiceKey,
		Properties: map[string]string{
			"key_file": keyFilePath,
		},
	}

	return nil
}

// AuthenticateWithOAuth2 authenticates using OAuth2 flow
func (am *GCPAuthenticationManager) AuthenticateWithOAuth2(ctx context.Context) error {
	// Perform OAuth2 login
	cmd := exec.Command("gcloud", "auth", "login", "--no-launch-browser")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to perform OAuth2 login: %w", err)
	}

	// Update credentials
	am.provider.credentials = &Credentials{
		Provider:   ProviderGCP,
		AuthMethod: AuthMethodCLI,
	}

	return nil
}

// AuthenticateWithApplicationDefaultCredentials uses Application Default Credentials
func (am *GCPAuthenticationManager) AuthenticateWithApplicationDefaultCredentials(ctx context.Context) error {
	// Check if ADC is available
	cmd := exec.Command("gcloud", "auth", "application-default", "print-access-token")
	if err := cmd.Run(); err != nil {
		// Try to set up ADC
		cmd = exec.Command("gcloud", "auth", "application-default", "login")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set up Application Default Credentials: %w", err)
		}
	}

	// Update credentials
	am.provider.credentials = &Credentials{
		Provider:   ProviderGCP,
		AuthMethod: AuthMethodSDK,
	}

	return nil
}

// SetupWorkloadIdentity configures Workload Identity for GKE
func (am *GCPAuthenticationManager) SetupWorkloadIdentity(ctx context.Context, projectID, clusterName, location, namespace, serviceAccountName, gcpServiceAccount string) error {
	// Enable Workload Identity on the cluster
	cmd := exec.Command("gcloud", "container", "clusters", "update", clusterName,
		"--location", location,
		"--workload-pool", fmt.Sprintf("%s.svc.id.goog", projectID))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable Workload Identity on cluster: %w", err)
	}

	// Create Kubernetes service account if it doesn't exist
	cmd = exec.Command("kubectl", "create", "serviceaccount", serviceAccountName, "-n", namespace)
	cmd.Run() // Ignore error if it already exists

	// Bind GCP service account to Kubernetes service account
	cmd = exec.Command("gcloud", "iam", "service-accounts", "add-iam-policy-binding",
		gcpServiceAccount,
		"--role", "roles/iam.workloadIdentityUser",
		"--member", fmt.Sprintf("serviceAccount:%s.svc.id.goog[%s/%s]", projectID, namespace, serviceAccountName))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to bind service accounts: %w", err)
	}

	// Annotate Kubernetes service account
	cmd = exec.Command("kubectl", "annotate", "serviceaccount", serviceAccountName, "-n", namespace,
		fmt.Sprintf("iam.gke.io/gcp-service-account=%s", gcpServiceAccount))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to annotate Kubernetes service account: %w", err)
	}

	return nil
}

// GetAccessToken gets an access token for the current authentication
func (am *GCPAuthenticationManager) GetAccessToken(ctx context.Context) (string, error) {
	cmd := exec.Command("gcloud", "auth", "print-access-token")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// GetIdentityToken gets an identity token for the current authentication
func (am *GCPAuthenticationManager) GetIdentityToken(ctx context.Context, audience string) (string, error) {
	cmd := exec.Command("gcloud", "auth", "print-identity-token", "--audiences", audience)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get identity token: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// ListActiveAccounts lists all active authenticated accounts
func (am *GCPAuthenticationManager) ListActiveAccounts(ctx context.Context) ([]string, error) {
	cmd := exec.Command("gcloud", "auth", "list", "--filter=status:ACTIVE", "--format=value(account)")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list active accounts: %w", err)
	}

	accounts := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(accounts) == 1 && accounts[0] == "" {
		return []string{}, nil
	}

	return accounts, nil
}

// SwitchAccount switches to a different authenticated account
func (am *GCPAuthenticationManager) SwitchAccount(ctx context.Context, account string) error {
	cmd := exec.Command("gcloud", "config", "set", "account", account)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to switch account: %w", err)
	}

	// Update credentials
	if am.provider.credentials == nil {
		am.provider.credentials = &Credentials{
			Provider:   ProviderGCP,
			AuthMethod: AuthMethodCLI,
		}
	}
	am.provider.credentials.Account = account

	return nil
}

// RevokeAuthentication revokes the current authentication
func (am *GCPAuthenticationManager) RevokeAuthentication(ctx context.Context, account string) error {
	cmd := exec.Command("gcloud", "auth", "revoke", account)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to revoke authentication: %w", err)
	}

	return nil
}

// GCPAdvancedOperations provides advanced GCP operations
type GCPAdvancedOperations struct {
	provider        *GCPProvider
	serviceAccount  *GCPServiceAccountManager
	resourceManager *GCPResourceManager
	monitoring      *GCPMonitoringManager
	storage         *GCPStorageManager
	authentication  *GCPAuthenticationManager
}

// NewGCPAdvancedOperations creates a new advanced operations manager
func NewGCPAdvancedOperations(provider *GCPProvider) *GCPAdvancedOperations {
	return &GCPAdvancedOperations{
		provider:        provider,
		serviceAccount:  NewGCPServiceAccountManager(provider),
		resourceManager: NewGCPResourceManager(provider),
		monitoring:      NewGCPMonitoringManager(provider),
		storage:         NewGCPStorageManager(provider),
		authentication:  NewGCPAuthenticationManager(provider),
	}
}

// GetServiceAccountManager returns the service account manager
func (ao *GCPAdvancedOperations) GetServiceAccountManager() *GCPServiceAccountManager {
	return ao.serviceAccount
}

// GetResourceManager returns the resource manager
func (ao *GCPAdvancedOperations) GetResourceManager() *GCPResourceManager {
	return ao.resourceManager
}

// GetMonitoringManager returns the monitoring manager
func (ao *GCPAdvancedOperations) GetMonitoringManager() *GCPMonitoringManager {
	return ao.monitoring
}

// GetStorageManager returns the storage manager
func (ao *GCPAdvancedOperations) GetStorageManager() *GCPStorageManager {
	return ao.storage
}

// GetAuthenticationManager returns the authentication manager
func (ao *GCPAdvancedOperations) GetAuthenticationManager() *GCPAuthenticationManager {
	return ao.authentication
}

// SetupAPMIntegration sets up complete APM integration for GCP
func (ao *GCPAdvancedOperations) SetupAPMIntegration(ctx context.Context, config APMIntegrationConfig) error {
	// Enable required APIs
	requiredAPIs := []string{
		"monitoring.googleapis.com",
		"cloudtrace.googleapis.com",
		"logging.googleapis.com",
		"container.googleapis.com",
		"artifactregistry.googleapis.com",
	}

	for _, api := range requiredAPIs {
		cmd := exec.Command("gcloud", "services", "enable", api)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to enable API %s: %w", api, err)
		}
	}

	// Create service account for APM
	if config.CreateServiceAccount {
		_, err := ao.serviceAccount.CreateServiceAccount(ctx,
			config.ServiceAccountID,
			"APM Service Account",
			"Service account for Application Performance Monitoring")
		if err != nil {
			return fmt.Errorf("failed to create APM service account: %w", err)
		}

		// Create key for service account
		keyPath := fmt.Sprintf("%s-key.json", config.ServiceAccountID)
		_, err = ao.serviceAccount.CreateServiceAccountKey(ctx,
			fmt.Sprintf("%s@%s.iam.gserviceaccount.com", config.ServiceAccountID, config.ProjectID),
			keyPath)
		if err != nil {
			return fmt.Errorf("failed to create service account key: %w", err)
		}
	}

	// Set up monitoring workspace
	if config.SetupMonitoring {
		err := ao.monitoring.EnableMonitoringAPI(ctx)
		if err != nil {
			return fmt.Errorf("failed to enable monitoring API: %w", err)
		}

		err = ao.monitoring.EnableCloudTraceAPI(ctx)
		if err != nil {
			return fmt.Errorf("failed to enable trace API: %w", err)
		}
	}

	// Set up storage bucket for logs/traces
	if config.CreateStorageBucket {
		bucketName := fmt.Sprintf("%s-apm-data", config.ProjectID)
		_, err := ao.storage.CreateStorageBucket(ctx, bucketName, config.Region, "STANDARD")
		if err != nil {
			return fmt.Errorf("failed to create storage bucket: %w", err)
		}
	}

	return nil
}

// APMIntegrationConfig configuration for APM integration
type APMIntegrationConfig struct {
	ProjectID              string
	Region                 string
	ServiceAccountID       string
	CreateServiceAccount   bool
	SetupMonitoring        bool
	CreateStorageBucket    bool
	EnableWorkloadIdentity bool
	ClusterName            string
	ClusterLocation        string
}
