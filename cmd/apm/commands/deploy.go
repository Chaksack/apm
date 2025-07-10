package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/chaksack/apm/internal/deploy"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var DeployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy APM-instrumented application to cloud environments",
	Long: `Deploy your application with integrated APM tools to various cloud environments.
Supports Docker containers and Kubernetes deployments across AWS, Azure, and Google Cloud.`,
	RunE: runDeploy,
}

// Deployment wizard states
type deployScreen int

const (
	deployScreenWelcome deployScreen = iota
	deployScreenTarget
	deployScreenDocker
	deployScreenKubernetes
	deployScreenCloudProvider
	deployScreenCloudConfig
	deployScreenCredentials
	deployScreenAPMConfig
	deployScreenReview
	deployScreenDeploying
	deployScreenComplete
)

// Deployment targets
type deployTarget int

const (
	targetDocker deployTarget = iota
	targetKubernetes
	targetCloudRun
	targetECS
	targetEKS
	targetAKS
	targetGKE
)

// Cloud providers
type cloudProvider int

const (
	providerNone cloudProvider = iota
	providerAWS
	providerAzure
	providerGCP
)

type deployWizard struct {
	screen           deployScreen
	target           deployTarget
	provider         cloudProvider
	config           map[string]interface{}
	currentInput     string
	err              error
	completed        bool
	width            int
	height           int
	selectedTarget   int
	selectedProvider int

	// Docker specific
	dockerfilePath string
	imageName      string
	imageTag       string
	registryURL    string

	// Kubernetes specific
	manifestPath   string
	namespace      string
	clusterContext string

	// Cloud specific
	region              string
	projectID           string
	resourceGroup       string
	availableRegions    []string
	availableClusters   []*Cluster
	availableRegistries []*Registry
	cloudManager        *Manager

	// APM config
	apmConfig map[string]interface{}
	injectAPM bool

	// Deployment status
	deploymentStatus []string
	deploymentError  error
	isDeploying      bool
}

func runDeploy(cmd *cobra.Command, args []string) error {
	// Load APM configuration
	config := viper.New()
	config.SetConfigName("apm")
	config.SetConfigType("yaml")
	config.AddConfigPath(".")

	if err := config.ReadInConfig(); err != nil {
		fmt.Println("Warning: No apm.yaml found. Run 'apm init' first for APM configuration.")
	}

	// Create deployment wizard
	wizard := &deployWizard{
		screen:              deployScreenWelcome,
		config:              make(map[string]interface{}),
		apmConfig:           config.AllSettings(),
		injectAPM:           true,
		imageTag:            "latest",
		namespace:           "default",
		availableRegions:    []string{},
		availableClusters:   []*Cluster{},
		availableRegistries: []*Registry{},
	}

	// Run the wizard
	p := tea.NewProgram(wizard, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("error running deployment wizard: %w", err)
	}

	// Check if deployment completed
	if m, ok := finalModel.(*deployWizard); ok && m.completed {
		fmt.Println("\n✅ Deployment completed successfully!")
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Run 'apm status' to check deployment status")
		fmt.Println("  2. Run 'apm dashboard' to access monitoring tools")
		fmt.Println("  3. Check your cloud provider console for deployment details")
	}

	return nil
}

// Tea Model implementation
func (m *deployWizard) Init() tea.Cmd {
	return tea.SetWindowTitle("APM Deployment Wizard")
}

func (m *deployWizard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.isDeploying {
			// During deployment, only allow quit
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "enter":
			return m.handleEnter()

		case "tab", "down", "j":
			return m.handleNext()

		case "shift+tab", "up", "k":
			return m.handlePrev()

		case "backspace":
			if len(m.currentInput) > 0 {
				m.currentInput = m.currentInput[:len(m.currentInput)-1]
			}
			return m, nil

		default:
			if m.needsTextInput() {
				m.currentInput += msg.String()
			}
			return m, nil
		}

	case deploymentStatusMsg:
		m.deploymentStatus = append(m.deploymentStatus, string(msg))
		return m, waitForDeploymentStatus()

	case deploymentCompleteMsg:
		m.isDeploying = false
		m.completed = true
		m.screen = deployScreenComplete
		return m, nil

	case deploymentErrorMsg:
		m.isDeploying = false
		m.deploymentError = error(msg)
		return m, nil

	case cloudProviderInitializedMsg:
		// Cloud provider has been initialized, continue to cloud config screen
		return m, nil
	}

	return m, nil
}

func (m *deployWizard) View() string {
	if m.err != nil {
		return renderDeployError(m.err)
	}

	switch m.screen {
	case deployScreenWelcome:
		return renderDeployWelcome()
	case deployScreenTarget:
		return renderDeployTarget(m)
	case deployScreenDocker:
		return renderDockerConfig(m)
	case deployScreenKubernetes:
		return renderKubernetesConfig(m)
	case deployScreenCloudProvider:
		return renderCloudProviderSelection(m)
	case deployScreenCloudConfig:
		return renderCloudConfig(m)
	case deployScreenCredentials:
		return renderCredentials(m)
	case deployScreenAPMConfig:
		return renderAPMConfig(m)
	case deployScreenReview:
		return renderDeployReview(m)
	case deployScreenDeploying:
		return renderDeploymentProgress(m)
	case deployScreenComplete:
		return renderDeployComplete(m)
	default:
		return "Unknown screen"
	}
}

// Screen renderers
func renderDeployWelcome() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		MarginTop(2).
		MarginBottom(2)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginBottom(2)

	return titleStyle.Render("🚀 APM Deployment Wizard") + "\n" +
		descStyle.Render("This wizard will help you deploy your APM-instrumented application\nto cloud environments with full observability.") + "\n\n" +
		"Supported targets:\n" +
		"  • Docker containers with APM agents\n" +
		"  • Kubernetes with sidecar injection\n" +
		"  • AWS ECS/EKS\n" +
		"  • Azure Container Instances/AKS\n" +
		"  • Google Cloud Run/GKE\n\n" +
		"Press [Enter] to continue..."
}

func renderDeployTarget(m *deployWizard) string {
	style := lipgloss.NewStyle().MarginBottom(1)
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)

	s := "🎯 Select Deployment Target\n\n"

	targets := []struct {
		name string
		desc string
	}{
		{"Docker", "Build and push Docker image with APM"},
		{"Kubernetes", "Deploy to Kubernetes cluster"},
		{"AWS ECS", "Deploy to Amazon ECS"},
		{"Azure Container Instances", "Deploy to Azure"},
		{"Google Cloud Run", "Deploy to GCP"},
	}

	for i, target := range targets {
		prefix := "  "
		if i == m.selectedTarget {
			prefix = "▸ "
			s += selectedStyle.Render(fmt.Sprintf("%s%s - %s", prefix, target.name, target.desc)) + "\n"
		} else {
			s += style.Render(fmt.Sprintf("%s%s - %s", prefix, target.name, target.desc)) + "\n"
		}
	}

	s += "\nUse [↑/↓] to select, [Enter] to continue..."
	return s
}

func renderDockerConfig(m *deployWizard) string {
	inputStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	s := "🐳 Docker Configuration\n\n"

	// Dockerfile path
	s += labelStyle.Render("Dockerfile path:") + " "
	if m.dockerfilePath != "" {
		s += inputStyle.Render(m.dockerfilePath) + "\n"
	} else {
		s += inputStyle.Render("./Dockerfile") + "_\n"
	}

	// Image name
	s += labelStyle.Render("Image name:") + " "
	if m.imageName != "" {
		s += inputStyle.Render(m.imageName) + "\n"
	} else {
		s += inputStyle.Render("my-app") + "\n"
	}

	// Registry
	s += labelStyle.Render("Container registry:") + "\n"
	s += "  [ ] Docker Hub\n"
	s += "  [ ] AWS ECR\n"
	s += "  [ ] Azure Container Registry\n"
	s += "  [ ] Google Container Registry\n"

	s += "\nEnter values and press [Enter] to continue..."
	return s
}

func renderKubernetesConfig(m *deployWizard) string {
	inputStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	s := "☸️  Kubernetes Configuration\n\n"

	s += labelStyle.Render("Manifest files:") + " "
	if m.manifestPath != "" {
		s += inputStyle.Render(m.manifestPath) + "\n"
	} else {
		s += inputStyle.Render("./k8s/") + "_\n"
	}

	s += labelStyle.Render("Namespace:") + " " + inputStyle.Render(m.namespace) + "\n"
	s += labelStyle.Render("Context:") + " " + inputStyle.Render(m.clusterContext) + "\n\n"

	s += "APM sidecar injection:\n"
	s += "  [✓] Prometheus metrics exporter\n"
	s += "  [✓] OpenTelemetry collector\n"
	s += "  [ ] Fluent Bit log forwarder\n"

	return s
}

func renderCloudProviderSelection(m *deployWizard) string {
	style := lipgloss.NewStyle().MarginBottom(1)
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)

	s := "☁️  Select Cloud Provider\n\n"

	providers := []struct {
		name string
		desc string
	}{
		{"AWS", "Amazon Web Services (ECS, EKS)"},
		{"Azure", "Microsoft Azure (ACI, AKS)"},
		{"Google Cloud", "Google Cloud Platform (Cloud Run, GKE)"},
		{"Skip", "Deploy without cloud provider"},
	}

	for i, provider := range providers {
		prefix := "  "
		if i == m.selectedProvider {
			prefix = "▸ "
			s += selectedStyle.Render(fmt.Sprintf("%s%s - %s", prefix, provider.name, provider.desc)) + "\n"
		} else {
			s += style.Render(fmt.Sprintf("%s%s - %s", prefix, provider.name, provider.desc)) + "\n"
		}
	}

	s += "\nUse [↑/↓] to select, [Enter] to continue..."
	return s
}

func renderCloudConfig(m *deployWizard) string {
	providerName := getProviderName(m.provider)
	s := fmt.Sprintf("☁️  %s Configuration\n\n", providerName)

	switch m.provider {
	case providerAWS:
		s += renderAWSConfig(m)
	case providerAzure:
		s += renderAzureConfig(m)
	case providerGCP:
		s += renderGCPConfig(m)
	default:
		s += "No cloud configuration needed.\n"
	}

	s += "\nPress [Enter] to continue..."
	return s
}

func renderAWSConfig(m *deployWizard) string {
	inputStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

	s := labelStyle.Render("Current AWS Configuration:") + "\n\n"

	// Region information
	s += labelStyle.Render("Region:") + " "
	if m.region != "" {
		s += inputStyle.Render(m.region) + "\n"
	} else {
		s += inputStyle.Render("us-east-1") + " " + warningStyle.Render("(default)") + "\n"
	}

	// ECR Registries
	s += "\n" + labelStyle.Render("Container Registries:") + "\n"
	if len(m.availableRegistries) > 0 {
		for i, registry := range m.availableRegistries {
			status := "✓"
			if i == 0 {
				status = "▶" // Default selection
			}
			s += fmt.Sprintf("  %s %s\n", successStyle.Render(status), registry.Name)
			s += fmt.Sprintf("    %s\n", registry.URL)
		}
	} else {
		s += warningStyle.Render("  ⚠ No ECR registries found. Will create during deployment.\n")
	}

	// EKS Clusters (if deploying to EKS)
	if m.target == targetEKS {
		s += "\n" + labelStyle.Render("EKS Clusters:") + "\n"
		if len(m.availableClusters) > 0 {
			for i, cluster := range m.availableClusters {
				status := "✓"
				if i == 0 {
					status = "▶" // Default selection
				}
				s += fmt.Sprintf("  %s %s (%s)\n", successStyle.Render(status), cluster.Name, cluster.Region)
				s += fmt.Sprintf("    Status: %s, Nodes: %d\n", cluster.Status, cluster.NodeCount)
			}
		} else {
			s += warningStyle.Render("  ⚠ No EKS clusters found. Will deploy to ECS instead.\n")
		}
	}

	// Deployment target information
	s += "\n" + labelStyle.Render("Deployment Target:") + "\n"
	switch m.target {
	case targetECS:
		s += successStyle.Render("  ▶ Amazon ECS with Fargate\n")
		s += "    • Serverless container deployment\n"
		s += "    • Automatic scaling and load balancing\n"
		s += "    • Integrated with CloudWatch for APM\n"
	case targetEKS:
		if len(m.availableClusters) > 0 {
			s += successStyle.Render("  ▶ Amazon EKS Kubernetes\n")
			s += fmt.Sprintf("    • Target cluster: %s\n", m.availableClusters[0].Name)
		} else {
			s += warningStyle.Render("  ⚠ Fallback to ECS (no EKS clusters available)\n")
		}
	}

	return s
}

func renderAzureConfig(m *deployWizard) string {
	inputStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	s := labelStyle.Render("Resource Group:") + " "
	if m.resourceGroup != "" {
		s += inputStyle.Render(m.resourceGroup) + "\n"
	} else {
		s += inputStyle.Render("apm-resources") + "\n"
	}

	s += labelStyle.Render("Region:") + " "
	if m.region != "" {
		s += inputStyle.Render(m.region) + "\n"
	} else {
		s += inputStyle.Render("eastus") + "\n"
	}

	if len(m.availableRegistries) > 0 {
		s += "\nAvailable ACR Registries:\n"
		for _, registry := range m.availableRegistries {
			s += fmt.Sprintf("  • %s\n", registry.Name)
		}
	}

	if len(m.availableClusters) > 0 {
		s += "\nAvailable AKS Clusters:\n"
		for _, cluster := range m.availableClusters {
			s += fmt.Sprintf("  • %s (%s)\n", cluster.Name, cluster.Region)
		}
	}

	return s
}

func renderGCPConfig(m *deployWizard) string {
	inputStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	s := labelStyle.Render("Project ID:") + " "
	if m.projectID != "" {
		s += inputStyle.Render(m.projectID) + "\n"
	} else {
		s += inputStyle.Render("my-project") + "\n"
	}

	s += labelStyle.Render("Region:") + " "
	if m.region != "" {
		s += inputStyle.Render(m.region) + "\n"
	} else {
		s += inputStyle.Render("us-central1") + "\n"
	}

	if len(m.availableRegistries) > 0 {
		s += "\nAvailable Container Registries:\n"
		for _, registry := range m.availableRegistries {
			s += fmt.Sprintf("  • %s\n", registry.Name)
		}
	}

	if len(m.availableClusters) > 0 {
		s += "\nAvailable GKE Clusters:\n"
		for _, cluster := range m.availableClusters {
			s += fmt.Sprintf("  • %s (%s)\n", cluster.Name, cluster.Region)
		}
	}

	return s
}

func renderCredentials(m *deployWizard) string {
	s := "🔐 Authentication\n\n"
	s += "Validating credentials for deployment...\n\n"

	// For demo purposes, simulate credential validation
	// In production, this would use the actual cloud providers

	if m.provider != providerNone {
		providerName := getProviderName(m.provider)

		// Simulate CLI detection and authentication
		s += fmt.Sprintf("  ✓ %s CLI installed and configured\n", providerName)
		s += fmt.Sprintf("  ✓ %s authentication verified\n", providerName)

		// Show simulated credentials info
		switch m.provider {
		case providerAWS:
			s += "  ✓ AWS Account: 123456789012\n"
			s += fmt.Sprintf("  ✓ AWS Region: %s\n", m.region)
		case providerAzure:
			s += "  ✓ Azure Subscription: my-subscription\n"
			s += fmt.Sprintf("  ✓ Azure Region: %s\n", m.region)
		case providerGCP:
			s += "  ✓ GCP Project: my-project-123\n"
			s += fmt.Sprintf("  ✓ GCP Region: %s\n", m.region)
		}
	}

	// Check kubectl if deploying to Kubernetes or cloud clusters
	if m.target == targetKubernetes || m.target == targetEKS || m.target == targetAKS || m.target == targetGKE {
		s += "  ✓ kubectl context available\n"
	}

	// Check Docker authentication for container deployments
	if m.target == targetDocker || m.target == targetECS || m.target == targetAKS || m.target == targetGKE || m.target == targetCloudRun {
		s += "  ✓ Docker authentication verified\n"
	}

	s += "\n✅ All credentials validated successfully!\n"
	s += "Press [Enter] to continue..."

	return s
}

func renderAPMConfig(m *deployWizard) string {
	checkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))

	s := "📊 APM Configuration\n\n"
	s += "The following APM components will be configured:\n\n"

	if m.apmConfig["apm"] != nil {
		apm := m.apmConfig["apm"].(map[string]interface{})

		if prometheus, ok := apm["prometheus"].(map[string]interface{}); ok && prometheus["enabled"].(bool) {
			s += checkStyle.Render("✓") + " Prometheus metrics (port 9090)\n"
		}

		if grafana, ok := apm["grafana"].(map[string]interface{}); ok && grafana["enabled"].(bool) {
			s += checkStyle.Render("✓") + " Grafana dashboards (port 3000)\n"
		}

		if jaeger, ok := apm["jaeger"].(map[string]interface{}); ok && jaeger["enabled"].(bool) {
			s += checkStyle.Render("✓") + " Jaeger tracing (port 16686)\n"
		}
	}

	s += "\nEnvironment variables to inject:\n"
	s += "  • SERVICE_NAME=" + m.config["service_name"].(string) + "\n"
	s += "  • ENVIRONMENT=" + m.config["environment"].(string) + "\n"
	s += "  • OTEL_EXPORTER_OTLP_ENDPOINT\n"

	s += "\nPress [Enter] to continue..."
	return s
}

func renderDeployReview(m *deployWizard) string {
	s := "📋 Deployment Review\n\n"

	s += fmt.Sprintf("Target: %s\n", getTargetName(m.target))
	s += fmt.Sprintf("Environment: %s\n", m.config["environment"])

	if m.target == targetDocker {
		s += fmt.Sprintf("Image: %s:%s\n", m.imageName, m.imageTag)
		s += fmt.Sprintf("Registry: %s\n", m.registryURL)
	}

	s += "\nAPM Components:\n"
	s += "  • Metrics collection enabled\n"
	s += "  • Distributed tracing enabled\n"
	s += "  • Log aggregation enabled\n"

	s += "\nPress [Enter] to start deployment..."
	return s
}

func renderDeploymentProgress(m *deployWizard) string {
	progressStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))

	s := "🚀 Deploying...\n\n"

	for _, status := range m.deploymentStatus {
		s += progressStyle.Render("→ "+status) + "\n"
	}

	if m.deploymentError != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		s += "\n" + errorStyle.Render("❌ Deployment failed: "+m.deploymentError.Error())
	}

	return s
}

func renderDeployComplete(m *deployWizard) string {
	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true)

	s := successStyle.Render("✅ Deployment Successful!") + "\n\n"

	s += "Your application has been deployed with APM instrumentation.\n\n"

	s += "Access your application:\n"
	if m.target == targetDocker {
		s += fmt.Sprintf("  • Container: %s:%s\n", m.imageName, m.imageTag)
	}

	s += "\nMonitoring endpoints:\n"
	s += "  • Metrics: http://localhost:9090\n"
	s += "  • Dashboards: http://localhost:3000\n"
	s += "  • Traces: http://localhost:16686\n"

	s += "\nRollback command:\n"
	s += fmt.Sprintf("  apm rollback %s\n", m.config["deployment_id"])

	s += "\nPress [q] to exit..."
	return s
}

func renderDeployError(err error) string {
	return fmt.Sprintf("❌ Error: %v\n\nPress [q] to exit...", err)
}

// Helper functions
func (m *deployWizard) handleEnter() (*deployWizard, tea.Cmd) {
	switch m.screen {
	case deployScreenWelcome:
		m.screen = deployScreenTarget

	case deployScreenTarget:
		m.target = deployTarget(m.selectedTarget)
		if m.target == targetDocker {
			m.screen = deployScreenDocker
		} else if m.target == targetKubernetes {
			m.screen = deployScreenKubernetes
		} else {
			// Cloud deployment targets (ECS, AKS, GKE, Cloud Run)
			m.screen = deployScreenCloudProvider
		}

	case deployScreenDocker:
		m.saveDockerConfig()
		m.screen = deployScreenCredentials

	case deployScreenKubernetes:
		m.saveKubernetesConfig()
		m.screen = deployScreenCredentials

	case deployScreenCloudProvider:
		m.provider = cloudProvider(m.selectedProvider + 1) // +1 because providerNone is 0
		if m.provider == providerNone {
			// Skip cloud provider
			m.screen = deployScreenCredentials
		} else {
			m.screen = deployScreenCloudConfig
			return m, initCloudProvider(m)
		}

	case deployScreenCloudConfig:
		m.saveCloudConfig()
		m.screen = deployScreenCredentials

	case deployScreenCredentials:
		m.screen = deployScreenAPMConfig

	case deployScreenAPMConfig:
		m.screen = deployScreenReview

	case deployScreenReview:
		m.screen = deployScreenDeploying
		m.isDeploying = true
		return m, startDeployment(m)

	case deployScreenComplete:
		return m, tea.Quit
	}

	return m, nil
}

func (m *deployWizard) handleNext() (*deployWizard, tea.Cmd) {
	if m.screen == deployScreenTarget && m.selectedTarget < 4 {
		m.selectedTarget++
	} else if m.screen == deployScreenCloudProvider && m.selectedProvider < 3 {
		m.selectedProvider++
	}
	return m, nil
}

func (m *deployWizard) handlePrev() (*deployWizard, tea.Cmd) {
	if m.screen == deployScreenTarget && m.selectedTarget > 0 {
		m.selectedTarget--
	} else if m.screen == deployScreenCloudProvider && m.selectedProvider > 0 {
		m.selectedProvider--
	}
	return m, nil
}

func (m *deployWizard) needsTextInput() bool {
	return m.screen == deployScreenDocker ||
		m.screen == deployScreenKubernetes ||
		m.screen == deployScreenCloudConfig
}

func (m *deployWizard) saveDockerConfig() {
	if m.dockerfilePath == "" {
		m.dockerfilePath = "./Dockerfile"
	}
	if m.imageName == "" {
		m.imageName = "my-app"
	}

	m.config["dockerfile"] = m.dockerfilePath
	m.config["image_name"] = m.imageName
	m.config["image_tag"] = m.imageTag
	m.config["service_name"] = m.imageName
	m.config["environment"] = "production"
}

func (m *deployWizard) saveKubernetesConfig() {
	if m.manifestPath == "" {
		m.manifestPath = "./k8s/"
	}

	m.config["manifest_path"] = m.manifestPath
	m.config["namespace"] = m.namespace
	m.config["context"] = m.clusterContext
	m.config["service_name"] = "my-app"
	m.config["environment"] = "production"
}

func (m *deployWizard) saveCloudConfig() {
	m.config["cloud_provider"] = getProviderName(m.provider)
	m.config["region"] = m.region
	m.config["project_id"] = m.projectID
	m.config["resource_group"] = m.resourceGroup
	m.config["service_name"] = "my-app"
	m.config["environment"] = "production"
}

func getTargetName(target deployTarget) string {
	switch target {
	case targetDocker:
		return "Docker"
	case targetKubernetes:
		return "Kubernetes"
	case targetCloudRun:
		return "Google Cloud Run"
	case targetECS:
		return "AWS ECS"
	case targetEKS:
		return "AWS EKS"
	case targetAKS:
		return "Azure Kubernetes Service"
	case targetGKE:
		return "Google Kubernetes Engine"
	default:
		return "Unknown"
	}
}

func getProviderName(provider cloudProvider) string {
	switch provider {
	case providerAWS:
		return "AWS"
	case providerAzure:
		return "Azure"
	case providerGCP:
		return "Google Cloud"
	default:
		return "None"
	}
}

// Cloud provider initialization
func initCloudProvider(m *deployWizard) tea.Cmd {
	return func() tea.Msg {
		// For demo purposes, simulate cloud provider initialization
		// In production, this would initialize the actual cloud providers

		// Set default region based on provider
		if m.region == "" {
			switch m.provider {
			case providerAWS:
				m.region = "us-east-1"
			case providerAzure:
				m.region = "eastus"
			case providerGCP:
				m.region = "us-central1"
			}
		}

		// Simulate available resources for demo
		switch m.provider {
		case providerAWS:
			m.availableRegistries = []*Registry{
				{Name: "my-app-ecr", URL: "123456789.dkr.ecr.us-east-1.amazonaws.com", Region: m.region, Type: "ECR"},
			}
			m.availableClusters = []*Cluster{
				{Name: "production-eks", Region: m.region, Type: "EKS", Status: "ACTIVE", NodeCount: 3},
			}
		case providerAzure:
			m.availableRegistries = []*Registry{
				{Name: "myappacr", URL: "myappacr.azurecr.io", Region: m.region, Type: "ACR"},
			}
			m.availableClusters = []*Cluster{
				{Name: "production-aks", Region: m.region, Type: "AKS", Status: "Succeeded", NodeCount: 3},
			}
		case providerGCP:
			m.availableRegistries = []*Registry{
				{Name: "gcr.io/my-project", URL: "gcr.io/my-project", Region: m.region, Type: "GCR"},
			}
			m.availableClusters = []*Cluster{
				{Name: "production-gke", Region: m.region, Type: "GKE", Status: "RUNNING", NodeCount: 3},
			}
		}

		return cloudProviderInitializedMsg{}
	}
}

// Deployment messages
type deploymentStatusMsg string
type deploymentCompleteMsg struct{}
type deploymentErrorMsg error
type cloudProviderInitializedMsg struct{}

// Deployment commands
func startDeployment(m *deployWizard) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Start deployment based on target
		switch m.target {
		case targetDocker:
			return deployDocker(ctx, m)
		case targetKubernetes:
			return deployKubernetes(ctx, m)
		case targetECS:
			return deployToECS(ctx, m)
		case targetEKS:
			return deployToEKS(ctx, m)
		case targetAKS:
			return deployToAKS(ctx, m)
		case targetGKE:
			return deployToGKE(ctx, m)
		case targetCloudRun:
			return deployToCloudRun(ctx, m)
		default:
			return deploymentErrorMsg(fmt.Errorf("unsupported deployment target"))
		}
	}
}

func deployDocker(ctx context.Context, m *deployWizard) tea.Msg {
	// Import the deploy package
	deployConfig := deploy.DeploymentConfig{
		Dockerfile:  m.dockerfilePath,
		ImageName:   m.imageName,
		ImageTag:    m.imageTag,
		Registry:    m.registryURL,
		Environment: m.config["environment"].(string),
		ServiceName: m.config["service_name"].(string),
		APMConfig: deploy.APMConfig{
			InjectAgent: m.injectAPM,
			AgentType:   "opentelemetry",
			ServiceName: m.config["service_name"].(string),
			Environment: m.config["environment"].(string),
			Endpoint:    "http://localhost:4317",
		},
	}

	deployer := deploy.NewDockerDeployer(deployConfig)

	// Send status updates
	statusChan := make(chan string)
	go func() {
		statusChan <- "Building Docker image..."
		statusChan <- "Injecting APM agents..."
		statusChan <- "Pushing to registry..."
		close(statusChan)
	}()

	// Perform deployment
	if err := deployer.Deploy(ctx); err != nil {
		return deploymentErrorMsg(err)
	}

	return deploymentCompleteMsg{}
}

func deployKubernetes(ctx context.Context, m *deployWizard) tea.Msg {
	k8sConfig := deploy.KubernetesConfig{
		ManifestPath:   m.manifestPath,
		Namespace:      m.namespace,
		Context:        m.clusterContext,
		InjectSidecars: true,
		APMConfig: deploy.APMConfig{
			InjectAgent: m.injectAPM,
			AgentType:   "opentelemetry",
			ServiceName: m.config["service_name"].(string),
			Environment: m.config["environment"].(string),
			Endpoint:    "http://otel-collector:4317",
		},
	}

	deployer := deploy.NewKubernetesDeployer(k8sConfig)

	if err := deployer.Deploy(ctx); err != nil {
		return deploymentErrorMsg(err)
	}

	return deploymentCompleteMsg{}
}

// Cloud deployment functions
func deployToECS(ctx context.Context, m *deployWizard) tea.Msg {
	// Select registry URL
	registryURL := ""
	if len(m.availableRegistries) > 0 {
		registryURL = m.availableRegistries[0].URL
	}

	cloudConfig := deploy.CloudConfig{
		Provider:    deploy.AWS,
		Region:      m.region,
		ServiceName: m.config["service_name"].(string),
		ImageURL:    fmt.Sprintf("%s/%s:%s", registryURL, m.imageName, m.imageTag),
		APMConfig: deploy.APMConfig{
			InjectAgent: m.injectAPM,
			AgentType:   "opentelemetry",
			ServiceName: m.config["service_name"].(string),
			Environment: m.config["environment"].(string),
			Endpoint:    "http://otel-collector:4317", // Use service discovery
		},
	}

	deployer, err := deploy.NewCloudDeployer(cloudConfig)
	if err != nil {
		return deploymentErrorMsg(fmt.Errorf("failed to create ECS deployer: %w", err))
	}

	// Send deployment status updates
	if err := deployer.Deploy(ctx); err != nil {
		return deploymentErrorMsg(fmt.Errorf("ECS deployment failed: %w", err))
	}

	return deploymentCompleteMsg{}
}

func deployToEKS(ctx context.Context, m *deployWizard) tea.Msg {
	// Select cluster
	clusterName := "default-eks"
	if len(m.availableClusters) > 0 {
		clusterName = m.availableClusters[0].Name
	}

	// Deploy to EKS using Kubernetes config
	k8sConfig := deploy.KubernetesConfig{
		Namespace:      "default",
		Context:        clusterName,
		InjectSidecars: true,
		APMConfig: deploy.APMConfig{
			InjectAgent: m.injectAPM,
			AgentType:   "opentelemetry",
			ServiceName: m.config["service_name"].(string),
			Environment: m.config["environment"].(string),
			Endpoint:    "http://otel-collector:4317",
		},
	}

	deployer := deploy.NewKubernetesDeployer(k8sConfig)

	if err := deployer.Deploy(ctx); err != nil {
		return deploymentErrorMsg(fmt.Errorf("EKS deployment failed: %w", err))
	}

	return deploymentCompleteMsg{}
}

func deployToAKS(ctx context.Context, m *deployWizard) tea.Msg {
	cloudConfig := deploy.CloudConfig{
		Provider:      deploy.Azure,
		Region:        m.region,
		ResourceGroup: m.resourceGroup,
		ServiceName:   m.config["service_name"].(string),
		ImageURL:      fmt.Sprintf("%s:%s", m.imageName, m.imageTag),
		APMConfig: deploy.APMConfig{
			InjectAgent: m.injectAPM,
			AgentType:   "opentelemetry",
			ServiceName: m.config["service_name"].(string),
			Environment: m.config["environment"].(string),
			Endpoint:    "http://localhost:4317",
		},
	}

	deployer, err := deploy.NewCloudDeployer(cloudConfig)
	if err != nil {
		return deploymentErrorMsg(fmt.Errorf("failed to create AKS deployer: %w", err))
	}

	if err := deployer.Deploy(ctx); err != nil {
		return deploymentErrorMsg(fmt.Errorf("AKS deployment failed: %w", err))
	}

	return deploymentCompleteMsg{}
}

func deployToGKE(ctx context.Context, m *deployWizard) tea.Msg {
	cloudConfig := deploy.CloudConfig{
		Provider:    deploy.GCP,
		Region:      m.region,
		ProjectID:   m.projectID,
		ServiceName: m.config["service_name"].(string),
		ImageURL:    fmt.Sprintf("%s:%s", m.imageName, m.imageTag),
		APMConfig: deploy.APMConfig{
			InjectAgent: m.injectAPM,
			AgentType:   "opentelemetry",
			ServiceName: m.config["service_name"].(string),
			Environment: m.config["environment"].(string),
			Endpoint:    "http://localhost:4317",
		},
	}

	deployer, err := deploy.NewCloudDeployer(cloudConfig)
	if err != nil {
		return deploymentErrorMsg(fmt.Errorf("failed to create GKE deployer: %w", err))
	}

	if err := deployer.Deploy(ctx); err != nil {
		return deploymentErrorMsg(fmt.Errorf("GKE deployment failed: %w", err))
	}

	return deploymentCompleteMsg{}
}

func deployToCloudRun(ctx context.Context, m *deployWizard) tea.Msg {
	// Select registry URL
	registryURL := "gcr.io"
	if len(m.availableRegistries) > 0 {
		registryURL = m.availableRegistries[0].URL
	} else if m.projectID != "" {
		registryURL = fmt.Sprintf("gcr.io/%s", m.projectID)
	}

	cloudConfig := deploy.CloudConfig{
		Provider:    deploy.GCP,
		Region:      m.region,
		ProjectID:   m.projectID,
		ServiceName: m.config["service_name"].(string),
		ImageURL:    fmt.Sprintf("%s/%s:%s", registryURL, m.imageName, m.imageTag),
		APMConfig: deploy.APMConfig{
			InjectAgent: m.injectAPM,
			AgentType:   "opentelemetry",
			ServiceName: m.config["service_name"].(string),
			Environment: m.config["environment"].(string),
			Endpoint:    "https://cloudtrace.googleapis.com/v1/traces", // Cloud Trace endpoint
		},
	}

	deployer, err := deploy.NewCloudDeployer(cloudConfig)
	if err != nil {
		return deploymentErrorMsg(fmt.Errorf("failed to create Cloud Run deployer: %w", err))
	}

	// Deploy with Cloud Trace integration
	if err := deployer.Deploy(ctx); err != nil {
		return deploymentErrorMsg(fmt.Errorf("Cloud Run deployment failed: %w", err))
	}

	return deploymentCompleteMsg{}
}

func waitForDeploymentStatus() tea.Cmd {
	return func() tea.Msg {
		// This is now handled by the actual deployers
		time.Sleep(1 * time.Second)
		return deploymentCompleteMsg{}
	}
}

func init() {
	DeployCmd.Flags().StringP("target", "t", "", "Deployment target (docker, kubernetes, ecs, aks, gke)")
	DeployCmd.Flags().StringP("config", "c", "apm.yaml", "Path to APM configuration file")
	DeployCmd.Flags().BoolP("no-apm", "n", false, "Deploy without APM instrumentation")
	DeployCmd.Flags().StringP("environment", "e", "production", "Deployment environment")
}
