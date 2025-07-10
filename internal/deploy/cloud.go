package deploy

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// CloudProvider represents a cloud provider
type CloudProvider string

const (
	AWS   CloudProvider = "aws"
	Azure CloudProvider = "azure"
	GCP   CloudProvider = "gcp"
)

// CloudDeployer interface for cloud-specific deployments
type CloudDeployer interface {
	Deploy(ctx context.Context) error
	CheckAuthentication(ctx context.Context) error
	GetDeploymentStatus(ctx context.Context, deploymentID string) (string, error)
}

// CloudConfig holds cloud deployment configuration
type CloudConfig struct {
	Provider      CloudProvider
	Region        string
	ProjectID     string
	ResourceGroup string
	ClusterName   string
	ServiceName   string
	ImageURL      string
	APMConfig     APMConfig
}

// NewCloudDeployer creates a cloud deployer based on provider
func NewCloudDeployer(config CloudConfig) (CloudDeployer, error) {
	switch config.Provider {
	case AWS:
		return &AWSDeployer{config: config}, nil
	case Azure:
		return &AzureDeployer{config: config}, nil
	case GCP:
		return &GCPDeployer{config: config}, nil
	default:
		return nil, fmt.Errorf("unsupported cloud provider: %s", config.Provider)
	}
}

// AWSDeployer handles AWS deployments
type AWSDeployer struct {
	config CloudConfig
}

// Deploy deploys to AWS ECS or EKS
func (d *AWSDeployer) Deploy(ctx context.Context) error {
	// Check if deploying to ECS or EKS
	if d.config.ClusterName != "" {
		return d.deployToEKS(ctx)
	}
	return d.deployToECS(ctx)
}

// deployToECS deploys to AWS ECS
func (d *AWSDeployer) deployToECS(ctx context.Context) error {
	// 1. Create task definition with APM
	taskDef := d.createECSTaskDefinition()

	// Register task definition
	cmd := exec.CommandContext(ctx, "aws", "ecs", "register-task-definition",
		"--family", d.config.ServiceName,
		"--region", d.config.Region,
		"--cli-input-json", taskDef,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to register task definition: %w", err)
	}

	// 2. Create or update service
	serviceCmd := exec.CommandContext(ctx, "aws", "ecs", "update-service",
		"--cluster", "default",
		"--service", d.config.ServiceName,
		"--task-definition", d.config.ServiceName,
		"--region", d.config.Region,
	)

	if err := serviceCmd.Run(); err != nil {
		// Service doesn't exist, create it
		createCmd := exec.CommandContext(ctx, "aws", "ecs", "create-service",
			"--cluster", "default",
			"--service-name", d.config.ServiceName,
			"--task-definition", d.config.ServiceName,
			"--desired-count", "1",
			"--region", d.config.Region,
		)

		if err := createCmd.Run(); err != nil {
			return fmt.Errorf("failed to create service: %w", err)
		}
	}

	return nil
}

// deployToEKS deploys to AWS EKS
func (d *AWSDeployer) deployToEKS(ctx context.Context) error {
	// Update kubeconfig for EKS
	cmd := exec.CommandContext(ctx, "aws", "eks", "update-kubeconfig",
		"--name", d.config.ClusterName,
		"--region", d.config.Region,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update kubeconfig: %w", err)
	}

	// Use Kubernetes deployer
	k8sConfig := KubernetesConfig{
		Namespace:      "default",
		APMConfig:      d.config.APMConfig,
		InjectSidecars: true,
	}

	k8sDeployer := NewKubernetesDeployer(k8sConfig)
	return k8sDeployer.Deploy(ctx)
}

// createECSTaskDefinition creates an ECS task definition with APM
func (d *AWSDeployer) createECSTaskDefinition() string {
	envVars := fmt.Sprintf(`
		{
			"name": "OTEL_SERVICE_NAME",
			"value": "%s"
		},
		{
			"name": "OTEL_ENVIRONMENT",
			"value": "%s"
		},
		{
			"name": "OTEL_EXPORTER_OTLP_ENDPOINT",
			"value": "%s"
		}
	`, d.config.ServiceName, d.config.APMConfig.Environment, d.config.APMConfig.Endpoint)

	return fmt.Sprintf(`{
		"family": "%s",
		"networkMode": "awsvpc",
		"requiresCompatibilities": ["FARGATE"],
		"cpu": "256",
		"memory": "512",
		"containerDefinitions": [
			{
				"name": "%s",
				"image": "%s",
				"essential": true,
				"environment": [%s],
				"logConfiguration": {
					"logDriver": "awslogs",
					"options": {
						"awslogs-group": "/ecs/%s",
						"awslogs-region": "%s",
						"awslogs-stream-prefix": "ecs"
					}
				}
			}
		]
	}`, d.config.ServiceName, d.config.ServiceName, d.config.ImageURL, envVars,
		d.config.ServiceName, d.config.Region)
}

// CheckAuthentication checks AWS authentication
func (d *AWSDeployer) CheckAuthentication(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "aws", "sts", "get-caller-identity")
	return cmd.Run()
}

// GetDeploymentStatus gets ECS service status
func (d *AWSDeployer) GetDeploymentStatus(ctx context.Context, deploymentID string) (string, error) {
	cmd := exec.CommandContext(ctx, "aws", "ecs", "describe-services",
		"--cluster", "default",
		"--services", deploymentID,
		"--region", d.config.Region,
		"--query", "services[0].status",
		"--output", "text",
	)

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// AzureDeployer handles Azure deployments
type AzureDeployer struct {
	config CloudConfig
}

// Deploy deploys to Azure Container Instances or AKS
func (d *AzureDeployer) Deploy(ctx context.Context) error {
	if d.config.ClusterName != "" {
		return d.deployToAKS(ctx)
	}
	return d.deployToACI(ctx)
}

// deployToACI deploys to Azure Container Instances
func (d *AzureDeployer) deployToACI(ctx context.Context) error {
	envVars := []string{
		fmt.Sprintf("OTEL_SERVICE_NAME=%s", d.config.ServiceName),
		fmt.Sprintf("OTEL_ENVIRONMENT=%s", d.config.APMConfig.Environment),
		fmt.Sprintf("OTEL_EXPORTER_OTLP_ENDPOINT=%s", d.config.APMConfig.Endpoint),
	}

	args := []string{
		"container", "create",
		"--resource-group", d.config.ResourceGroup,
		"--name", d.config.ServiceName,
		"--image", d.config.ImageURL,
		"--cpu", "1",
		"--memory", "1",
		"--environment-variables", strings.Join(envVars, " "),
	}

	cmd := exec.CommandContext(ctx, "az", args...)
	return cmd.Run()
}

// deployToAKS deploys to Azure Kubernetes Service
func (d *AzureDeployer) deployToAKS(ctx context.Context) error {
	// Get AKS credentials
	cmd := exec.CommandContext(ctx, "az", "aks", "get-credentials",
		"--resource-group", d.config.ResourceGroup,
		"--name", d.config.ClusterName,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to get AKS credentials: %w", err)
	}

	// Use Kubernetes deployer
	k8sConfig := KubernetesConfig{
		Namespace:      "default",
		APMConfig:      d.config.APMConfig,
		InjectSidecars: true,
	}

	k8sDeployer := NewKubernetesDeployer(k8sConfig)
	return k8sDeployer.Deploy(ctx)
}

// CheckAuthentication checks Azure authentication
func (d *AzureDeployer) CheckAuthentication(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "az", "account", "show")
	return cmd.Run()
}

// GetDeploymentStatus gets Azure deployment status
func (d *AzureDeployer) GetDeploymentStatus(ctx context.Context, deploymentID string) (string, error) {
	cmd := exec.CommandContext(ctx, "az", "container", "show",
		"--resource-group", d.config.ResourceGroup,
		"--name", deploymentID,
		"--query", "instanceView.state",
		"--output", "tsv",
	)

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// GCPDeployer handles Google Cloud deployments
type GCPDeployer struct {
	config CloudConfig
}

// Deploy deploys to Cloud Run or GKE
func (d *GCPDeployer) Deploy(ctx context.Context) error {
	if d.config.ClusterName != "" {
		return d.deployToGKE(ctx)
	}
	return d.deployToCloudRun(ctx)
}

// deployToCloudRun deploys to Google Cloud Run
func (d *GCPDeployer) deployToCloudRun(ctx context.Context) error {
	args := []string{
		"run", "deploy", d.config.ServiceName,
		"--image", d.config.ImageURL,
		"--region", d.config.Region,
		"--project", d.config.ProjectID,
		"--platform", "managed",
		"--allow-unauthenticated",
		"--set-env-vars", fmt.Sprintf(
			"OTEL_SERVICE_NAME=%s,OTEL_ENVIRONMENT=%s,OTEL_EXPORTER_OTLP_ENDPOINT=%s",
			d.config.ServiceName,
			d.config.APMConfig.Environment,
			d.config.APMConfig.Endpoint,
		),
	}

	cmd := exec.CommandContext(ctx, "gcloud", args...)
	return cmd.Run()
}

// deployToGKE deploys to Google Kubernetes Engine
func (d *GCPDeployer) deployToGKE(ctx context.Context) error {
	// Get GKE credentials
	cmd := exec.CommandContext(ctx, "gcloud", "container", "clusters", "get-credentials",
		d.config.ClusterName,
		"--region", d.config.Region,
		"--project", d.config.ProjectID,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to get GKE credentials: %w", err)
	}

	// Use Kubernetes deployer
	k8sConfig := KubernetesConfig{
		Namespace:      "default",
		APMConfig:      d.config.APMConfig,
		InjectSidecars: true,
	}

	k8sDeployer := NewKubernetesDeployer(k8sConfig)
	return k8sDeployer.Deploy(ctx)
}

// CheckAuthentication checks GCP authentication
func (d *GCPDeployer) CheckAuthentication(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "gcloud", "auth", "list", "--filter=status:ACTIVE")
	return cmd.Run()
}

// GetDeploymentStatus gets Cloud Run service status
func (d *GCPDeployer) GetDeploymentStatus(ctx context.Context, deploymentID string) (string, error) {
	cmd := exec.CommandContext(ctx, "gcloud", "run", "services", "describe",
		deploymentID,
		"--region", d.config.Region,
		"--project", d.config.ProjectID,
		"--format", "value(status.conditions[0].status)",
	)

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// CheckCLITools checks if required CLI tools are installed
func CheckCLITools(provider CloudProvider) error {
	var tool string

	switch provider {
	case AWS:
		tool = "aws"
	case Azure:
		tool = "az"
	case GCP:
		tool = "gcloud"
	default:
		return fmt.Errorf("unknown provider: %s", provider)
	}

	cmd := exec.Command("which", tool)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s CLI not found. Please install it first", tool)
	}

	return nil
}
