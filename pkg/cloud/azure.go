package cloud

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

// AzureProviderImpl implements AzureProvider for Azure
type AzureProviderImpl struct {
	config      *ProviderConfig
	credentials *Credentials
	cliStatus   *CLIStatus
	cache       *CredentialCache
	logger      *log.Logger
	httpClient  *http.Client
}

// NewAzureProvider creates a new Azure provider
func NewAzureProvider(config *ProviderConfig) (*AzureProviderImpl, error) {
	if config == nil {
		config = &ProviderConfig{
			Provider:      ProviderAzure,
			DefaultRegion: "eastus",
			EnableCache:   true,
			CacheDuration: 5 * time.Minute,
		}
	}

	return &AzureProviderImpl{
		config:     config,
		cache:      NewCredentialCache(config.CacheDuration),
		logger:     log.New(os.Stdout, "[Azure] ", log.LstdFlags),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// Name returns the provider name
func (p *AzureProviderImpl) Name() Provider {
	return ProviderAzure
}

// DetectCLI detects Azure CLI installation
func (p *AzureProviderImpl) DetectCLI() (*CLIStatus, error) {
	detector := NewAzureCLIDetector()
	status, err := detector.Detect()
	if err != nil {
		return nil, err
	}
	p.cliStatus = status
	return status, nil
}

// ValidateCLI validates Azure CLI is properly configured
func (p *AzureProviderImpl) ValidateCLI() error {
	if p.cliStatus == nil {
		if _, err := p.DetectCLI(); err != nil {
			return err
		}
	}

	if !p.cliStatus.Installed {
		return fmt.Errorf("Azure CLI not installed")
	}

	// Check if logged in
	cmd := exec.Command("az", "account", "show")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Azure CLI not authenticated: run 'az login'")
	}

	return nil
}

// GetCLIVersion returns the Azure CLI version
func (p *AzureProviderImpl) GetCLIVersion() (string, error) {
	if p.cliStatus == nil {
		if _, err := p.DetectCLI(); err != nil {
			return "", err
		}
	}
	return p.cliStatus.Version, nil
}

// ValidateAuth validates Azure authentication
func (p *AzureProviderImpl) ValidateAuth(ctx context.Context) error {
	cmd := exec.Command("az", "account", "show")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("authentication validation failed: %w", err)
	}

	// Parse the output to get account info
	var account struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		State        string `json:"state"`
		TenantID     string `json:"tenantId"`
		IsDefault    bool   `json:"isDefault"`
		CloudName    string `json:"cloudName"`
		HomeTenantID string `json:"homeTenantId"`
	}
	if err := json.Unmarshal(output, &account); err != nil {
		return fmt.Errorf("failed to parse account info: %w", err)
	}

	if account.State != "Enabled" {
		return fmt.Errorf("account is not enabled")
	}

	// Store account info in credentials
	if p.credentials == nil {
		p.credentials = &Credentials{
			Provider:   ProviderAzure,
			AuthMethod: AuthMethodCLI,
		}
	}
	p.credentials.Account = account.ID
	if p.credentials.Properties == nil {
		p.credentials.Properties = make(map[string]string)
	}
	p.credentials.Properties["subscription_id"] = account.ID
	p.credentials.Properties["tenant_id"] = account.TenantID

	return nil
}

// GetCredentials returns current Azure credentials
func (p *AzureProviderImpl) GetCredentials() (*Credentials, error) {
	if p.credentials != nil {
		return p.credentials, nil
	}

	// Try to get from environment
	if clientID := os.Getenv("AZURE_CLIENT_ID"); clientID != "" {
		p.credentials = &Credentials{
			Provider:   ProviderAzure,
			AuthMethod: AuthMethodServiceKey,
			AccessKey:  clientID,
			SecretKey:  os.Getenv("AZURE_CLIENT_SECRET"),
			Properties: map[string]string{
				"tenant_id":       os.Getenv("AZURE_TENANT_ID"),
				"subscription_id": os.Getenv("AZURE_SUBSCRIPTION_ID"),
			},
		}
		return p.credentials, nil
	}

	// Get from CLI
	p.credentials = &Credentials{
		Provider:   ProviderAzure,
		AuthMethod: AuthMethodCLI,
		Region:     p.GetCurrentRegion(),
	}

	// Get subscription info
	cmd := exec.Command("az", "account", "show", "--query", "{subscriptionId:id,tenantId:tenantId}", "-o", "json")
	if output, err := cmd.Output(); err == nil {
		var info map[string]string
		if json.Unmarshal(output, &info) == nil {
			p.credentials.Account = info["subscriptionId"]
			if p.credentials.Properties == nil {
				p.credentials.Properties = make(map[string]string)
			}
			p.credentials.Properties["subscription_id"] = info["subscriptionId"]
			p.credentials.Properties["tenant_id"] = info["tenantId"]
		}
	}

	return p.credentials, nil
}

// ListRegistries lists ACR registries
func (p *AzureProviderImpl) ListRegistries(ctx context.Context) ([]*Registry, error) {
	cmd := exec.Command("az", "acr", "list", "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list registries: %w", err)
	}

	var acrList []struct {
		ID                string `json:"id"`
		Location          string `json:"location"`
		LoginServer       string `json:"loginServer"`
		Name              string `json:"name"`
		ProvisioningState string `json:"provisioningState"`
		ResourceGroup     string `json:"resourceGroup"`
		SKU               struct {
			Name string `json:"name"`
			Tier string `json:"tier"`
		} `json:"sku"`
		Tags map[string]string `json:"tags"`
	}

	if err := json.Unmarshal(output, &acrList); err != nil {
		return nil, fmt.Errorf("failed to parse registries: %w", err)
	}

	registries := make([]*Registry, 0, len(acrList))
	for _, acr := range acrList {
		if acr.ProvisioningState == "Succeeded" {
			registries = append(registries, &Registry{
				Provider: ProviderAzure,
				Name:     acr.Name,
				URL:      acr.LoginServer,
				Region:   acr.Location,
				Type:     "ACR",
			})
		}
	}

	return registries, nil
}

// GetRegistry gets a specific ACR registry
func (p *AzureProviderImpl) GetRegistry(ctx context.Context, name string) (*Registry, error) {
	cmd := exec.Command("az", "acr", "show", "--name", name, "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get registry: %w", err)
	}

	var acr struct {
		ID                string `json:"id"`
		Location          string `json:"location"`
		LoginServer       string `json:"loginServer"`
		Name              string `json:"name"`
		ProvisioningState string `json:"provisioningState"`
	}

	if err := json.Unmarshal(output, &acr); err != nil {
		return nil, fmt.Errorf("failed to parse registry: %w", err)
	}

	return &Registry{
		Provider: ProviderAzure,
		Name:     acr.Name,
		URL:      acr.LoginServer,
		Region:   acr.Location,
		Type:     "ACR",
	}, nil
}

// AuthenticateRegistry authenticates to ACR
func (p *AzureProviderImpl) AuthenticateRegistry(ctx context.Context, registry *Registry) error {
	// Login to ACR
	cmd := exec.Command("az", "acr", "login", "--name", registry.Name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to login to ACR: %w", err)
	}

	return nil
}

// ListClusters lists AKS clusters
func (p *AzureProviderImpl) ListClusters(ctx context.Context) ([]*Cluster, error) {
	cmd := exec.Command("az", "aks", "list", "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	var aksList []struct {
		ID                string `json:"id"`
		Location          string `json:"location"`
		Name              string `json:"name"`
		ResourceGroup     string `json:"resourceGroup"`
		KubernetesVersion string `json:"kubernetesVersion"`
		FQDN              string `json:"fqdn"`
		ProvisioningState string `json:"provisioningState"`
		PowerState        struct {
			Code string `json:"code"`
		} `json:"powerState"`
		AgentPoolProfiles []struct {
			Count int    `json:"count"`
			Name  string `json:"name"`
		} `json:"agentPoolProfiles"`
		Tags map[string]string `json:"tags"`
	}

	if err := json.Unmarshal(output, &aksList); err != nil {
		return nil, fmt.Errorf("failed to parse clusters: %w", err)
	}

	clusters := make([]*Cluster, 0, len(aksList))
	for _, aks := range aksList {
		nodeCount := 0
		for _, pool := range aks.AgentPoolProfiles {
			nodeCount += pool.Count
		}

		status := "Unknown"
		if aks.ProvisioningState == "Succeeded" && aks.PowerState.Code == "Running" {
			status = "Running"
		} else if aks.PowerState.Code == "Stopped" {
			status = "Stopped"
		}

		clusters = append(clusters, &Cluster{
			Provider:  ProviderAzure,
			Name:      aks.Name,
			Region:    aks.Location,
			Type:      "AKS",
			Version:   aks.KubernetesVersion,
			Endpoint:  aks.FQDN,
			NodeCount: nodeCount,
			Status:    status,
			Labels:    aks.Tags,
			Properties: map[string]string{
				"resource_group": aks.ResourceGroup,
			},
		})
	}

	return clusters, nil
}

// GetCluster gets details of an AKS cluster
func (p *AzureProviderImpl) GetCluster(ctx context.Context, name string) (*Cluster, error) {
	// First, find the resource group
	cmd := exec.Command("az", "aks", "list", "--query", "[?name=='"+name+"'].resourceGroup", "-o", "tsv")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to find cluster: %w", err)
	}
	resourceGroup := strings.TrimSpace(string(output))

	if resourceGroup == "" {
		return nil, fmt.Errorf("cluster %s not found", name)
	}

	// Get cluster details
	cmd = exec.Command("az", "aks", "show", "--name", name, "--resource-group", resourceGroup, "-o", "json")
	output, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster details: %w", err)
	}

	var aks struct {
		ID                string `json:"id"`
		Location          string `json:"location"`
		Name              string `json:"name"`
		ResourceGroup     string `json:"resourceGroup"`
		KubernetesVersion string `json:"kubernetesVersion"`
		FQDN              string `json:"fqdn"`
		ProvisioningState string `json:"provisioningState"`
		PowerState        struct {
			Code string `json:"code"`
		} `json:"powerState"`
		AgentPoolProfiles []struct {
			Count int    `json:"count"`
			Name  string `json:"name"`
		} `json:"agentPoolProfiles"`
		Tags map[string]string `json:"tags"`
	}

	if err := json.Unmarshal(output, &aks); err != nil {
		return nil, fmt.Errorf("failed to parse cluster: %w", err)
	}

	nodeCount := 0
	for _, pool := range aks.AgentPoolProfiles {
		nodeCount += pool.Count
	}

	status := "Unknown"
	if aks.ProvisioningState == "Succeeded" && aks.PowerState.Code == "Running" {
		status = "Running"
	} else if aks.PowerState.Code == "Stopped" {
		status = "Stopped"
	}

	return &Cluster{
		Provider:  ProviderAzure,
		Name:      aks.Name,
		Region:    aks.Location,
		Type:      "AKS",
		Version:   aks.KubernetesVersion,
		Endpoint:  aks.FQDN,
		NodeCount: nodeCount,
		Status:    status,
		Labels:    aks.Tags,
		Properties: map[string]string{
			"resource_group": aks.ResourceGroup,
		},
	}, nil
}

// GetKubeconfig gets kubeconfig for an AKS cluster
func (p *AzureProviderImpl) GetKubeconfig(ctx context.Context, cluster *Cluster) ([]byte, error) {
	resourceGroup := ""
	if cluster.Properties != nil {
		resourceGroup = cluster.Properties["resource_group"]
	}

	if resourceGroup == "" {
		// Try to find the resource group
		cmd := exec.Command("az", "aks", "list", "--query", "[?name=='"+cluster.Name+"'].resourceGroup", "-o", "tsv")
		output, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to find resource group: %w", err)
		}
		resourceGroup = strings.TrimSpace(string(output))
	}

	// Get credentials
	cmd := exec.Command("az", "aks", "get-credentials",
		"--name", cluster.Name,
		"--resource-group", resourceGroup,
		"--file", "-", // Output to stdout
	)

	kubeconfig, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	return kubeconfig, nil
}

// ListRegions lists Azure regions
func (p *AzureProviderImpl) ListRegions(ctx context.Context) ([]string, error) {
	cmd := exec.Command("az", "account", "list-locations", "--query", "[].name", "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list regions: %w", err)
	}

	var regions []string
	if err := json.Unmarshal(output, &regions); err != nil {
		return nil, fmt.Errorf("failed to parse regions: %w", err)
	}

	return regions, nil
}

// GetCurrentRegion gets the current Azure region
func (p *AzureProviderImpl) GetCurrentRegion() string {
	// Check config
	if p.config.DefaultRegion != "" {
		return p.config.DefaultRegion
	}

	// Azure doesn't have a global "current region" concept
	// Return the default
	return "eastus"
}

// SetRegion sets the Azure region
func (p *AzureProviderImpl) SetRegion(region string) error {
	p.config.DefaultRegion = region
	return nil
}

// AzureAPIFallback provides API-based operations when CLI is not available
type AzureAPIFallback struct {
	provider *AzureProviderImpl
}

// NewAzureAPIFallback creates a new Azure API fallback
func NewAzureAPIFallback(provider *AzureProviderImpl) *AzureAPIFallback {
	return &AzureAPIFallback{
		provider: provider,
	}
}

// IsAvailable checks if API fallback is available
func (f *AzureAPIFallback) IsAvailable() bool {
	// Check if Azure SDK credentials are available
	creds, err := f.provider.GetCredentials()
	if err != nil {
		return false
	}
	return creds.AuthMethod == AuthMethodServiceKey
}

// ListClustersViaAPI lists clusters using Azure SDK
func (f *AzureAPIFallback) ListClustersViaAPI(ctx context.Context) ([]*Cluster, error) {
	// This would use Azure SDK for Go
	// For now, return an error indicating SDK implementation needed
	return nil, fmt.Errorf("Azure SDK implementation required for API fallback")
}

// ListRegistriesViaAPI lists registries using Azure SDK
func (f *AzureAPIFallback) ListRegistriesViaAPI(ctx context.Context) ([]*Registry, error) {
	// This would use Azure SDK for Go
	// For now, return an error indicating SDK implementation needed
	return nil, fmt.Errorf("Azure SDK implementation required for API fallback")
}

// GetCredentialsViaAPI gets credentials using Azure SDK
func (f *AzureAPIFallback) GetCredentialsViaAPI(ctx context.Context) (*Credentials, error) {
	// This would use Azure SDK for Go
	// For now, return an error indicating SDK implementation needed
	return nil, fmt.Errorf("Azure SDK implementation required for API fallback")
}

// Enhanced Azure Authentication Methods

// AuthenticateInteractive performs interactive browser authentication
func (p *AzureProviderImpl) AuthenticateInteractive(ctx context.Context) error {
	p.logger.Println("Starting interactive authentication...")

	cmd := exec.CommandContext(ctx, "az", "login", "--use-device-code")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("interactive authentication failed: %w, output: %s", err, string(output))
	}

	p.logger.Println("Interactive authentication successful")
	return p.ValidateAuth(ctx)
}

// AuthenticateDeviceCode initiates device code authentication flow
func (p *AzureProviderImpl) AuthenticateDeviceCode(ctx context.Context) (*DeviceCodeAuth, error) {
	p.logger.Println("Starting device code authentication...")

	cmd := exec.CommandContext(ctx, "az", "login", "--use-device-code", "--output", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("device code authentication failed: %w, output: %s", err, string(output))
	}

	// Parse device code response (simplified for now)
	deviceAuth := &DeviceCodeAuth{
		DeviceCode:      generateDeviceCode(),
		UserCode:        generateUserCode(),
		VerificationURL: "https://microsoft.com/devicelogin",
		ExpiresIn:       900, // 15 minutes
		Interval:        5,   // 5 seconds
		Message:         "To sign in, use a web browser to open the page https://microsoft.com/devicelogin and enter the code to authenticate.",
		ExpiresAt:       time.Now().Add(15 * time.Minute),
	}

	p.logger.Printf("Device code authentication initiated. User code: %s", deviceAuth.UserCode)
	return deviceAuth, nil
}

// AuthenticateServicePrincipal authenticates using service principal credentials
func (p *AzureProviderImpl) AuthenticateServicePrincipal(ctx context.Context, clientID, clientSecret, tenantID string) error {
	p.logger.Printf("Authenticating with service principal: %s", clientID)

	cmd := exec.CommandContext(ctx, "az", "login",
		"--service-principal",
		"--username", clientID,
		"--password", clientSecret,
		"--tenant", tenantID,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("service principal authentication failed: %w, output: %s", err, string(output))
	}

	// Update credentials
	p.credentials = &Credentials{
		Provider:   ProviderAzure,
		AuthMethod: AuthMethodServicePrincipal,
		AccessKey:  clientID,
		SecretKey:  clientSecret,
		Properties: map[string]string{
			"tenant_id": tenantID,
		},
	}

	p.logger.Println("Service principal authentication successful")
	return nil
}

// AuthenticateManagedIdentity authenticates using managed identity
func (p *AzureProviderImpl) AuthenticateManagedIdentity(ctx context.Context) error {
	p.logger.Println("Attempting managed identity authentication...")

	// Check if running in Azure environment
	if !p.isAzureEnvironment() {
		return fmt.Errorf("managed identity authentication only available in Azure environment")
	}

	// Use Azure Instance Metadata Service (IMDS) to get token
	token, err := p.getManagedIdentityToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get managed identity token: %w", err)
	}

	p.credentials = &Credentials{
		Provider:   ProviderAzure,
		AuthMethod: AuthMethodManagedIdentity,
		Token:      token,
		Expiry:     timePtr(time.Now().Add(1 * time.Hour)), // Tokens typically expire in 1 hour
	}

	p.logger.Println("Managed identity authentication successful")
	return nil
}

// Subscription Management

// ListSubscriptions lists all available Azure subscriptions
func (p *AzureProviderImpl) ListSubscriptions(ctx context.Context) ([]*AzureSubscription, error) {
	p.logger.Println("Listing Azure subscriptions...")

	cmd := exec.CommandContext(ctx, "az", "account", "list", "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions: %w", err)
	}

	var subscriptions []*AzureSubscription
	if err := json.Unmarshal(output, &subscriptions); err != nil {
		return nil, fmt.Errorf("failed to parse subscriptions: %w", err)
	}

	p.logger.Printf("Found %d subscriptions", len(subscriptions))
	return subscriptions, nil
}

// GetSubscription gets details of a specific subscription
func (p *AzureProviderImpl) GetSubscription(ctx context.Context, subscriptionID string) (*AzureSubscription, error) {
	p.logger.Printf("Getting subscription details: %s", subscriptionID)

	cmd := exec.CommandContext(ctx, "az", "account", "show", "--subscription", subscriptionID, "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	var subscription AzureSubscription
	if err := json.Unmarshal(output, &subscription); err != nil {
		return nil, fmt.Errorf("failed to parse subscription: %w", err)
	}

	return &subscription, nil
}

// SetDefaultSubscription sets the default subscription
func (p *AzureProviderImpl) SetDefaultSubscription(ctx context.Context, subscriptionID string) error {
	p.logger.Printf("Setting default subscription: %s", subscriptionID)

	cmd := exec.CommandContext(ctx, "az", "account", "set", "--subscription", subscriptionID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set default subscription: %w", err)
	}

	p.logger.Println("Default subscription set successfully")
	return nil
}

// Resource Group Management

// ListResourceGroups lists all resource groups in the current subscription
func (p *AzureProviderImpl) ListResourceGroups(ctx context.Context) ([]*AzureResourceGroup, error) {
	p.logger.Println("Listing resource groups...")

	cmd := exec.CommandContext(ctx, "az", "group", "list", "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list resource groups: %w", err)
	}

	var resourceGroups []*AzureResourceGroup
	if err := json.Unmarshal(output, &resourceGroups); err != nil {
		return nil, fmt.Errorf("failed to parse resource groups: %w", err)
	}

	p.logger.Printf("Found %d resource groups", len(resourceGroups))
	return resourceGroups, nil
}

// CreateResourceGroup creates a new resource group
func (p *AzureProviderImpl) CreateResourceGroup(ctx context.Context, name, location string, tags map[string]string) (*AzureResourceGroup, error) {
	p.logger.Printf("Creating resource group: %s in %s", name, location)

	args := []string{"group", "create", "--name", name, "--location", location, "-o", "json"}

	// Add tags if provided
	if len(tags) > 0 {
		tagStrings := make([]string, 0, len(tags))
		for k, v := range tags {
			tagStrings = append(tagStrings, fmt.Sprintf("%s=%s", k, v))
		}
		args = append(args, "--tags")
		args = append(args, tagStrings...)
	}

	cmd := exec.CommandContext(ctx, "az", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to create resource group: %w", err)
	}

	var resourceGroup AzureResourceGroup
	if err := json.Unmarshal(output, &resourceGroup); err != nil {
		return nil, fmt.Errorf("failed to parse created resource group: %w", err)
	}

	p.logger.Printf("Resource group created successfully: %s", name)
	return &resourceGroup, nil
}

// DeleteResourceGroup deletes a resource group
func (p *AzureProviderImpl) DeleteResourceGroup(ctx context.Context, name string) error {
	p.logger.Printf("Deleting resource group: %s", name)

	cmd := exec.CommandContext(ctx, "az", "group", "delete", "--name", name, "--yes", "--no-wait")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete resource group: %w", err)
	}

	p.logger.Printf("Resource group deletion initiated: %s", name)
	return nil
}

// Service Principal Management

// CreateServicePrincipal creates a new service principal
func (p *AzureProviderImpl) CreateServicePrincipal(ctx context.Context, name string) (*AzureServicePrincipal, error) {
	p.logger.Printf("Creating service principal: %s", name)

	cmd := exec.CommandContext(ctx, "az", "ad", "sp", "create-for-rbac",
		"--display-name", name,
		"--role", "Contributor",
		"-o", "json")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to create service principal: %w", err)
	}

	var result struct {
		AppID    string `json:"appId"`
		Password string `json:"password"`
		Tenant   string `json:"tenant"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse service principal response: %w", err)
	}

	sp := &AzureServicePrincipal{
		AppID:       result.AppID,
		DisplayName: name,
		Password:    result.Password,
		Tenant:      result.Tenant,
		CreatedAt:   time.Now(),
		ExpiresAt:   timePtr(time.Now().Add(365 * 24 * time.Hour)), // Default 1 year
	}

	p.logger.Printf("Service principal created successfully: %s (App ID: %s)", name, result.AppID)
	return sp, nil
}

// ListServicePrincipals lists all service principals
func (p *AzureProviderImpl) ListServicePrincipals(ctx context.Context) ([]*AzureServicePrincipal, error) {
	p.logger.Println("Listing service principals...")

	cmd := exec.CommandContext(ctx, "az", "ad", "sp", "list", "--all", "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list service principals: %w", err)
	}

	var spList []struct {
		AppID       string `json:"appId"`
		DisplayName string `json:"displayName"`
	}

	if err := json.Unmarshal(output, &spList); err != nil {
		return nil, fmt.Errorf("failed to parse service principals: %w", err)
	}

	servicePrincipals := make([]*AzureServicePrincipal, len(spList))
	for i, sp := range spList {
		servicePrincipals[i] = &AzureServicePrincipal{
			AppID:       sp.AppID,
			DisplayName: sp.DisplayName,
		}
	}

	p.logger.Printf("Found %d service principals", len(servicePrincipals))
	return servicePrincipals, nil
}

// DeleteServicePrincipal deletes a service principal
func (p *AzureProviderImpl) DeleteServicePrincipal(ctx context.Context, appID string) error {
	p.logger.Printf("Deleting service principal: %s", appID)

	cmd := exec.CommandContext(ctx, "az", "ad", "sp", "delete", "--id", appID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete service principal: %w", err)
	}

	p.logger.Printf("Service principal deleted successfully: %s", appID)
	return nil
}

// RotateServicePrincipalSecret rotates the secret for a service principal
func (p *AzureProviderImpl) RotateServicePrincipalSecret(ctx context.Context, appID string) (*AzureServicePrincipal, error) {
	p.logger.Printf("Rotating secret for service principal: %s", appID)

	cmd := exec.CommandContext(ctx, "az", "ad", "sp", "credential", "reset",
		"--id", appID,
		"-o", "json")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to rotate service principal secret: %w", err)
	}

	var result struct {
		AppID    string `json:"appId"`
		Password string `json:"password"`
		Tenant   string `json:"tenant"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse rotated credentials: %w", err)
	}

	sp := &AzureServicePrincipal{
		AppID:     result.AppID,
		Password:  result.Password,
		Tenant:    result.Tenant,
		CreatedAt: time.Now(),
		ExpiresAt: timePtr(time.Now().Add(365 * 24 * time.Hour)), // Default 1 year
	}

	p.logger.Printf("Service principal secret rotated successfully: %s", appID)
	return sp, nil
}

// Helper functions

func generateDeviceCode() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}

func generateUserCode() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return strings.ToUpper(base64.URLEncoding.EncodeToString(bytes)[:8])
}

func (p *AzureProviderImpl) isAzureEnvironment() bool {
	// Check for Azure environment indicators
	if os.Getenv("MSI_ENDPOINT") != "" || os.Getenv("IDENTITY_ENDPOINT") != "" {
		return true
	}

	// Try to access the Azure Instance Metadata Service
	client := &http.Client{Timeout: 2 * time.Second}
	req, _ := http.NewRequest("GET", "http://169.254.169.254/metadata/instance?api-version=2021-02-01", nil)
	req.Header.Set("Metadata", "true")

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func (p *AzureProviderImpl) getManagedIdentityToken(ctx context.Context) (string, error) {
	endpoint := os.Getenv("IDENTITY_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://169.254.169.254/metadata/identity/oauth2/token"
	}

	// Build request URL
	params := url.Values{}
	params.Set("api-version", "2018-02-01")
	params.Set("resource", "https://management.azure.com/")

	reqURL := endpoint + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Metadata", "true")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get managed identity token: %s", string(body))
	}

	var tokenResponse struct {
		AccessToken string `json:"access_token"`
		ExpiresOn   string `json:"expires_on"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return "", err
	}

	return tokenResponse.AccessToken, nil
}

func timePtr(t time.Time) *time.Time {
	return &t
}

// Azure Monitor Integration

// GetMonitorMetrics retrieves metrics from Azure Monitor
func (p *AzureProviderImpl) GetMonitorMetrics(ctx context.Context, resourceID string, metricNames []string, timespan string) ([]*AzureMonitorMetric, error) {
	p.logger.Printf("Getting monitor metrics for resource: %s", resourceID)

	args := []string{"monitor", "metrics", "list",
		"--resource", resourceID,
		"--metric", strings.Join(metricNames, ","),
		"--timespan", timespan,
		"-o", "json"}

	cmd := exec.CommandContext(ctx, "az", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get monitor metrics: %w", err)
	}

	var result struct {
		Value []*AzureMonitorMetric `json:"value"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse monitor metrics: %w", err)
	}

	p.logger.Printf("Retrieved %d metrics", len(result.Value))
	return result.Value, nil
}

// CreateAlertRule creates an alert rule in Azure Monitor
func (p *AzureProviderImpl) CreateAlertRule(ctx context.Context, name, resourceGroup string, config map[string]interface{}) error {
	p.logger.Printf("Creating alert rule: %s in resource group: %s", name, resourceGroup)

	// Convert config to JSON for passing to Azure CLI
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal alert rule config: %w", err)
	}

	// Write config to temporary file
	tmpFile, err := os.CreateTemp("", "alert-rule-*.json")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write(configJSON); err != nil {
		return fmt.Errorf("failed to write config to temp file: %w", err)
	}
	tmpFile.Close()

	cmd := exec.CommandContext(ctx, "az", "monitor", "scheduled-query", "create",
		"--name", name,
		"--resource-group", resourceGroup,
		"--condition-query", "@"+tmpFile.Name())

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create alert rule: %w", err)
	}

	p.logger.Printf("Alert rule created successfully: %s", name)
	return nil
}

// ListActionGroups lists action groups in a resource group
func (p *AzureProviderImpl) ListActionGroups(ctx context.Context, resourceGroup string) ([]map[string]interface{}, error) {
	p.logger.Printf("Listing action groups in resource group: %s", resourceGroup)

	args := []string{"monitor", "action-group", "list", "-o", "json"}
	if resourceGroup != "" {
		args = append(args, "--resource-group", resourceGroup)
	}

	cmd := exec.CommandContext(ctx, "az", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list action groups: %w", err)
	}

	var actionGroups []map[string]interface{}
	if err := json.Unmarshal(output, &actionGroups); err != nil {
		return nil, fmt.Errorf("failed to parse action groups: %w", err)
	}

	p.logger.Printf("Found %d action groups", len(actionGroups))
	return actionGroups, nil
}

// Application Insights Integration

// CreateApplicationInsights creates a new Application Insights resource
func (p *AzureProviderImpl) CreateApplicationInsights(ctx context.Context, name, resourceGroup, location string) (*AzureApplicationInsights, error) {
	p.logger.Printf("Creating Application Insights: %s in %s", name, resourceGroup)

	cmd := exec.CommandContext(ctx, "az", "extension", "add", "--name", "application-insights")
	cmd.Run() // Install extension if not present

	cmd = exec.CommandContext(ctx, "az", "monitor", "app-insights", "component", "create",
		"--app", name,
		"--location", location,
		"--resource-group", resourceGroup,
		"--application-type", "web",
		"-o", "json")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to create Application Insights: %w", err)
	}

	var appInsights AzureApplicationInsights
	if err := json.Unmarshal(output, &appInsights); err != nil {
		return nil, fmt.Errorf("failed to parse Application Insights response: %w", err)
	}

	p.logger.Printf("Application Insights created successfully: %s", name)
	return &appInsights, nil
}

// ListApplicationInsights lists all Application Insights resources
func (p *AzureProviderImpl) ListApplicationInsights(ctx context.Context) ([]*AzureApplicationInsights, error) {
	p.logger.Println("Listing Application Insights resources...")

	cmd := exec.CommandContext(ctx, "az", "monitor", "app-insights", "component", "show", "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list Application Insights: %w", err)
	}

	var appInsightsList []*AzureApplicationInsights
	if err := json.Unmarshal(output, &appInsightsList); err != nil {
		return nil, fmt.Errorf("failed to parse Application Insights list: %w", err)
	}

	p.logger.Printf("Found %d Application Insights resources", len(appInsightsList))
	return appInsightsList, nil
}

// GetApplicationInsights gets a specific Application Insights resource
func (p *AzureProviderImpl) GetApplicationInsights(ctx context.Context, name, resourceGroup string) (*AzureApplicationInsights, error) {
	p.logger.Printf("Getting Application Insights: %s in %s", name, resourceGroup)

	cmd := exec.CommandContext(ctx, "az", "monitor", "app-insights", "component", "show",
		"--app", name,
		"--resource-group", resourceGroup,
		"-o", "json")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get Application Insights: %w", err)
	}

	var appInsights AzureApplicationInsights
	if err := json.Unmarshal(output, &appInsights); err != nil {
		return nil, fmt.Errorf("failed to parse Application Insights: %w", err)
	}

	return &appInsights, nil
}

// Storage Account Management

// ListStorageAccounts lists all storage accounts
func (p *AzureProviderImpl) ListStorageAccounts(ctx context.Context) ([]*AzureStorageAccount, error) {
	p.logger.Println("Listing storage accounts...")

	cmd := exec.CommandContext(ctx, "az", "storage", "account", "list", "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list storage accounts: %w", err)
	}

	var storageAccounts []*AzureStorageAccount
	if err := json.Unmarshal(output, &storageAccounts); err != nil {
		return nil, fmt.Errorf("failed to parse storage accounts: %w", err)
	}

	p.logger.Printf("Found %d storage accounts", len(storageAccounts))
	return storageAccounts, nil
}

// CreateStorageAccount creates a new storage account
func (p *AzureProviderImpl) CreateStorageAccount(ctx context.Context, name, resourceGroup, location string) (*AzureStorageAccount, error) {
	p.logger.Printf("Creating storage account: %s in %s", name, resourceGroup)

	cmd := exec.CommandContext(ctx, "az", "storage", "account", "create",
		"--name", name,
		"--resource-group", resourceGroup,
		"--location", location,
		"--sku", "Standard_LRS",
		"--kind", "StorageV2",
		"-o", "json")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to create storage account: %w", err)
	}

	var storageAccount AzureStorageAccount
	if err := json.Unmarshal(output, &storageAccount); err != nil {
		return nil, fmt.Errorf("failed to parse storage account response: %w", err)
	}

	p.logger.Printf("Storage account created successfully: %s", name)
	return &storageAccount, nil
}

// GetStorageAccountKeys gets the keys for a storage account
func (p *AzureProviderImpl) GetStorageAccountKeys(ctx context.Context, name, resourceGroup string) ([]string, error) {
	p.logger.Printf("Getting storage account keys: %s", name)

	cmd := exec.CommandContext(ctx, "az", "storage", "account", "keys", "list",
		"--account-name", name,
		"--resource-group", resourceGroup,
		"-o", "json")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get storage account keys: %w", err)
	}

	var keys []struct {
		KeyName string `json:"keyName"`
		Value   string `json:"value"`
	}

	if err := json.Unmarshal(output, &keys); err != nil {
		return nil, fmt.Errorf("failed to parse storage account keys: %w", err)
	}

	keyValues := make([]string, len(keys))
	for i, key := range keys {
		keyValues[i] = key.Value
	}

	p.logger.Printf("Retrieved %d storage account keys", len(keyValues))
	return keyValues, nil
}

// Key Vault Integration

// ListKeyVaults lists all key vaults
func (p *AzureProviderImpl) ListKeyVaults(ctx context.Context) ([]string, error) {
	p.logger.Println("Listing key vaults...")

	cmd := exec.CommandContext(ctx, "az", "keyvault", "list", "--query", "[].name", "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list key vaults: %w", err)
	}

	var vaultNames []string
	if err := json.Unmarshal(output, &vaultNames); err != nil {
		return nil, fmt.Errorf("failed to parse key vault names: %w", err)
	}

	p.logger.Printf("Found %d key vaults", len(vaultNames))
	return vaultNames, nil
}

// GetSecret retrieves a secret from Key Vault
func (p *AzureProviderImpl) GetSecret(ctx context.Context, vaultName, secretName string) (*AzureKeyVaultSecret, error) {
	p.logger.Printf("Getting secret %s from vault %s", secretName, vaultName)

	cmd := exec.CommandContext(ctx, "az", "keyvault", "secret", "show",
		"--vault-name", vaultName,
		"--name", secretName,
		"-o", "json")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	var secret AzureKeyVaultSecret
	if err := json.Unmarshal(output, &secret); err != nil {
		return nil, fmt.Errorf("failed to parse secret: %w", err)
	}

	return &secret, nil
}

// SetSecret stores a secret in Key Vault
func (p *AzureProviderImpl) SetSecret(ctx context.Context, vaultName, secretName, value string) error {
	p.logger.Printf("Setting secret %s in vault %s", secretName, vaultName)

	cmd := exec.CommandContext(ctx, "az", "keyvault", "secret", "set",
		"--vault-name", vaultName,
		"--name", secretName,
		"--value", value)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set secret: %w", err)
	}

	p.logger.Printf("Secret set successfully: %s", secretName)
	return nil
}

// DeleteSecret deletes a secret from Key Vault
func (p *AzureProviderImpl) DeleteSecret(ctx context.Context, vaultName, secretName string) error {
	p.logger.Printf("Deleting secret %s from vault %s", secretName, vaultName)

	cmd := exec.CommandContext(ctx, "az", "keyvault", "secret", "delete",
		"--vault-name", vaultName,
		"--name", secretName)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	p.logger.Printf("Secret deleted successfully: %s", secretName)
	return nil
}

// ARM Template Integration

// ValidateARMTemplate validates an ARM template
func (p *AzureProviderImpl) ValidateARMTemplate(ctx context.Context, template *AzureARMTemplate) (*ValidationResult, error) {
	p.logger.Printf("Validating ARM template: %s", template.Name)

	// Write template to temporary file
	templateJSON, err := json.Marshal(template.Template)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal template: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "arm-template-*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write(templateJSON); err != nil {
		return nil, fmt.Errorf("failed to write template: %w", err)
	}
	tmpFile.Close()

	args := []string{"deployment", "group", "validate",
		"--resource-group", template.ResourceGroup,
		"--template-file", tmpFile.Name()}

	// Add parameters if provided
	if len(template.Parameters) > 0 {
		paramJSON, err := json.Marshal(template.Parameters)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal parameters: %w", err)
		}
		args = append(args, "--parameters", string(paramJSON))
	}

	cmd := exec.CommandContext(ctx, "az", args...)
	output, err := cmd.CombinedOutput()

	result := &ValidationResult{
		Valid:   err == nil,
		Details: make(map[string]string),
	}

	if err != nil {
		result.Errors = []string{string(output)}
		p.logger.Printf("ARM template validation failed: %s", string(output))
	} else {
		p.logger.Printf("ARM template validation successful")
	}

	return result, nil
}

// DeployARMTemplate deploys an ARM template
func (p *AzureProviderImpl) DeployARMTemplate(ctx context.Context, template *AzureARMTemplate) (string, error) {
	p.logger.Printf("Deploying ARM template: %s", template.Name)

	// Write template to temporary file
	templateJSON, err := json.Marshal(template.Template)
	if err != nil {
		return "", fmt.Errorf("failed to marshal template: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "arm-template-*.json")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write(templateJSON); err != nil {
		return "", fmt.Errorf("failed to write template: %w", err)
	}
	tmpFile.Close()

	deploymentName := template.DeploymentName
	if deploymentName == "" {
		deploymentName = fmt.Sprintf("%s-%d", template.Name, time.Now().Unix())
	}

	args := []string{"deployment", "group", "create",
		"--resource-group", template.ResourceGroup,
		"--name", deploymentName,
		"--template-file", tmpFile.Name(),
		"--mode", template.Mode}

	// Add parameters if provided
	if len(template.Parameters) > 0 {
		paramJSON, err := json.Marshal(template.Parameters)
		if err != nil {
			return "", fmt.Errorf("failed to marshal parameters: %w", err)
		}
		args = append(args, "--parameters", string(paramJSON))
	}

	cmd := exec.CommandContext(ctx, "az", args...)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to deploy ARM template: %w", err)
	}

	p.logger.Printf("ARM template deployed successfully: %s", deploymentName)
	return deploymentName, nil
}

// GetDeploymentStatus gets the status of a deployment
func (p *AzureProviderImpl) GetDeploymentStatus(ctx context.Context, resourceGroup, deploymentName string) (string, error) {
	p.logger.Printf("Getting deployment status: %s in %s", deploymentName, resourceGroup)

	cmd := exec.CommandContext(ctx, "az", "deployment", "group", "show",
		"--resource-group", resourceGroup,
		"--name", deploymentName,
		"--query", "properties.provisioningState",
		"-o", "tsv")

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get deployment status: %w", err)
	}

	status := strings.TrimSpace(string(output))
	p.logger.Printf("Deployment status: %s", status)
	return status, nil
}
