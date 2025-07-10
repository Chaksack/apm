package commands

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// generateDryRunReport generates a deployment plan for dry-run mode
func generateDryRunReport(m *deployWizard) string {
	var report strings.Builder

	// Styles
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		MarginBottom(1)

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("214"))

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255"))

	warningStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Italic(true)

	// Header
	report.WriteString(headerStyle.Render("🔍 Deployment Plan (Dry Run)") + "\n")
	report.WriteString(strings.Repeat("─", 60) + "\n\n")

	// Deployment Target
	report.WriteString(sectionStyle.Render("Deployment Target") + "\n")
	report.WriteString(fmt.Sprintf("  %s %s\n", keyStyle.Render("Type:"), valueStyle.Render(getTargetName(m.target))))

	if m.provider != providerNone {
		report.WriteString(fmt.Sprintf("  %s %s\n", keyStyle.Render("Provider:"), valueStyle.Render(getProviderName(m.provider))))
	}

	if m.region != "" {
		report.WriteString(fmt.Sprintf("  %s %s\n", keyStyle.Render("Region:"), valueStyle.Render(m.region)))
	}
	report.WriteString("\n")

	// Application Configuration
	report.WriteString(sectionStyle.Render("Application") + "\n")
	serviceName := "my-app"
	if name, ok := m.config["service_name"].(string); ok {
		serviceName = name
	}
	report.WriteString(fmt.Sprintf("  %s %s\n", keyStyle.Render("Service Name:"), valueStyle.Render(serviceName)))
	report.WriteString(fmt.Sprintf("  %s %s\n", keyStyle.Render("Environment:"), valueStyle.Render(m.config["environment"].(string))))

	switch m.target {
	case targetDocker:
		report.WriteString(fmt.Sprintf("  %s %s\n", keyStyle.Render("Dockerfile:"), valueStyle.Render(m.dockerfilePath)))
		report.WriteString(fmt.Sprintf("  %s %s:%s\n", keyStyle.Render("Image:"), valueStyle.Render(m.imageName), valueStyle.Render(m.imageTag)))
		if m.registryURL != "" {
			report.WriteString(fmt.Sprintf("  %s %s\n", keyStyle.Render("Registry:"), valueStyle.Render(m.registryURL)))
		}
	case targetKubernetes:
		report.WriteString(fmt.Sprintf("  %s %s\n", keyStyle.Render("Manifests:"), valueStyle.Render(m.manifestPath)))
		report.WriteString(fmt.Sprintf("  %s %s\n", keyStyle.Render("Namespace:"), valueStyle.Render(m.namespace)))
		if m.clusterContext != "" {
			report.WriteString(fmt.Sprintf("  %s %s\n", keyStyle.Render("Context:"), valueStyle.Render(m.clusterContext)))
		}
	}
	report.WriteString("\n")

	// APM Configuration
	report.WriteString(sectionStyle.Render("APM Instrumentation") + "\n")
	if m.injectAPM {
		report.WriteString(fmt.Sprintf("  %s %s\n", keyStyle.Render("Status:"), valueStyle.Render("Enabled")))
		report.WriteString(fmt.Sprintf("  %s %s\n", keyStyle.Render("Agent Type:"), valueStyle.Render("OpenTelemetry")))

		// Show enabled APM tools
		tools := []string{}
		if m.apmConfig["apm"].(map[string]interface{})["prometheus"].(map[string]interface{})["enabled"].(bool) {
			tools = append(tools, "Prometheus")
		}
		if m.apmConfig["apm"].(map[string]interface{})["grafana"].(map[string]interface{})["enabled"].(bool) {
			tools = append(tools, "Grafana")
		}
		if m.apmConfig["apm"].(map[string]interface{})["jaeger"].(map[string]interface{})["enabled"].(bool) {
			tools = append(tools, "Jaeger")
		}
		if m.apmConfig["apm"].(map[string]interface{})["loki"].(map[string]interface{})["enabled"].(bool) {
			tools = append(tools, "Loki")
		}

		if len(tools) > 0 {
			report.WriteString(fmt.Sprintf("  %s %s\n", keyStyle.Render("Tools:"), valueStyle.Render(strings.Join(tools, ", "))))
		}
	} else {
		report.WriteString(fmt.Sprintf("  %s %s\n", keyStyle.Render("Status:"), valueStyle.Render("Disabled")))
	}
	report.WriteString("\n")

	// Actions to be performed
	report.WriteString(sectionStyle.Render("Actions to be Performed") + "\n")
	actions := getDeploymentActions(m)
	for i, action := range actions {
		report.WriteString(fmt.Sprintf("  %d. %s\n", i+1, action))
	}
	report.WriteString("\n")

	// Resources to be created
	if m.provider != providerNone {
		report.WriteString(sectionStyle.Render("Cloud Resources") + "\n")
		resources := getCloudResources(m)
		for _, resource := range resources {
			report.WriteString(fmt.Sprintf("  • %s\n", resource))
		}
		report.WriteString("\n")
	}

	// Estimated costs (if applicable)
	if m.provider != providerNone {
		report.WriteString(sectionStyle.Render("Estimated Costs") + "\n")
		report.WriteString("  " + warningStyle.Render("Note: These are rough estimates. Actual costs may vary.") + "\n")
		costs := getEstimatedCosts(m)
		for service, cost := range costs {
			report.WriteString(fmt.Sprintf("  %s %s\n", keyStyle.Render(service+":"), valueStyle.Render(cost)))
		}
		report.WriteString("\n")
	}

	// Commands that would be executed
	report.WriteString(sectionStyle.Render("Commands to Execute") + "\n")
	commands := getDeploymentCommands(m)
	cmdStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Background(lipgloss.Color("235")).
		Padding(0, 1)

	for _, cmd := range commands {
		report.WriteString("  " + cmdStyle.Render(cmd) + "\n")
	}
	report.WriteString("\n")

	// Footer
	report.WriteString(strings.Repeat("─", 60) + "\n")
	report.WriteString(warningStyle.Render("This is a dry run. No changes will be made.") + "\n")
	report.WriteString("To execute this deployment, run without --dry-run flag.\n")

	return report.String()
}

func getDeploymentActions(m *deployWizard) []string {
	actions := []string{}

	switch m.target {
	case targetDocker:
		actions = append(actions, "Build Docker image from "+m.dockerfilePath)
		if m.injectAPM {
			actions = append(actions, "Inject APM instrumentation agents")
		}
		actions = append(actions, "Tag image as "+m.imageName+":"+m.imageTag)
		if m.registryURL != "" {
			actions = append(actions, "Push image to registry: "+m.registryURL)
		}

	case targetKubernetes:
		actions = append(actions, "Validate Kubernetes manifests")
		if m.injectAPM {
			actions = append(actions, "Inject APM sidecar containers")
		}
		actions = append(actions, "Create/update resources in namespace: "+m.namespace)
		actions = append(actions, "Wait for pods to become ready")

	case targetECS:
		actions = append(actions, "Build and push Docker image to ECR")
		actions = append(actions, "Create/update ECS task definition")
		actions = append(actions, "Create/update ECS service")
		actions = append(actions, "Configure load balancer")

	case targetEKS:
		actions = append(actions, "Build and push Docker image to ECR")
		actions = append(actions, "Generate kubeconfig for EKS cluster")
		actions = append(actions, "Deploy Kubernetes manifests")
		actions = append(actions, "Configure ingress/load balancer")
	}

	return actions
}

func getCloudResources(m *deployWizard) []string {
	resources := []string{}

	switch m.target {
	case targetECS:
		resources = append(resources,
			"ECS Task Definition",
			"ECS Service",
			"Application Load Balancer",
			"Target Group",
			"Security Groups",
			"CloudWatch Log Group",
		)
		if m.injectAPM {
			resources = append(resources, "X-Ray Service Map")
		}

	case targetEKS:
		resources = append(resources,
			"EKS Node Group (if not exists)",
			"Kubernetes Deployment",
			"Kubernetes Service",
			"Ingress/Load Balancer",
		)

	case targetAKS:
		resources = append(resources,
			"AKS Node Pool (if not exists)",
			"Kubernetes Deployment",
			"Kubernetes Service",
			"Azure Load Balancer",
		)
		if m.injectAPM {
			resources = append(resources, "Azure Monitor workspace")
		}

	case targetGKE:
		resources = append(resources,
			"GKE Node Pool (if not exists)",
			"Kubernetes Deployment",
			"Kubernetes Service",
			"Google Load Balancer",
		)
		if m.injectAPM {
			resources = append(resources, "Cloud Trace", "Cloud Monitoring workspace")
		}

	case targetCloudRun:
		resources = append(resources,
			"Cloud Run Service",
			"Container Registry Image",
			"Cloud Load Balancer",
		)
	}

	return resources
}

func getEstimatedCosts(m *deployWizard) map[string]string {
	costs := make(map[string]string)

	switch m.provider {
	case providerAWS:
		if m.target == targetECS {
			costs["ECS Fargate"] = "$0.04/hour per vCPU + $0.004/hour per GB"
			costs["Load Balancer"] = "$0.025/hour + $0.008/GB processed"
			costs["CloudWatch Logs"] = "$0.50/GB ingested"
		} else if m.target == targetEKS {
			costs["EKS Cluster"] = "$0.10/hour"
			costs["EC2 Nodes"] = "Varies by instance type"
			costs["Load Balancer"] = "$0.025/hour + $0.008/GB processed"
		}

	case providerAzure:
		costs["AKS Control Plane"] = "Free"
		costs["Virtual Machines"] = "Varies by size"
		costs["Load Balancer"] = "$0.025/hour"
		costs["Azure Monitor"] = "$2.30/GB ingested"

	case providerGCP:
		if m.target == targetCloudRun {
			costs["Cloud Run"] = "$0.00002400/vCPU-second + $0.00000250/GB-second"
			costs["Load Balancer"] = "$0.025/hour"
		} else if m.target == targetGKE {
			costs["GKE Cluster"] = "$0.10/hour"
			costs["Compute Engine"] = "Varies by machine type"
			costs["Load Balancer"] = "$0.025/hour"
		}
	}

	costs["Total Estimate"] = "~$50-200/month for small workloads"
	return costs
}

func getDeploymentCommands(m *deployWizard) []string {
	commands := []string{}

	switch m.target {
	case targetDocker:
		commands = append(commands, fmt.Sprintf("docker build -t %s:%s -f %s .", m.imageName, m.imageTag, m.dockerfilePath))
		if m.registryURL != "" {
			commands = append(commands, fmt.Sprintf("docker tag %s:%s %s/%s:%s", m.imageName, m.imageTag, m.registryURL, m.imageName, m.imageTag))
			commands = append(commands, fmt.Sprintf("docker push %s/%s:%s", m.registryURL, m.imageName, m.imageTag))
		}

	case targetKubernetes:
		if m.clusterContext != "" {
			commands = append(commands, fmt.Sprintf("kubectl config use-context %s", m.clusterContext))
		}
		commands = append(commands, fmt.Sprintf("kubectl apply -f %s -n %s", m.manifestPath, m.namespace))
		commands = append(commands, fmt.Sprintf("kubectl rollout status deployment -n %s", m.namespace))

	case targetECS:
		commands = append(commands, "aws ecr get-login-password | docker login --username AWS --password-stdin <registry>")
		commands = append(commands, "docker build -t <image> .")
		commands = append(commands, "docker push <registry>/<image>")
		commands = append(commands, "aws ecs register-task-definition --family <app> --cli-input-json file://task-definition.json")
		commands = append(commands, "aws ecs update-service --cluster <cluster> --service <service> --task-definition <task-def>")

	case targetEKS:
		commands = append(commands, fmt.Sprintf("aws eks update-kubeconfig --name <cluster> --region %s", m.region))
		commands = append(commands, "kubectl apply -f manifests/")
		commands = append(commands, "kubectl get pods -w")
	}

	return commands
}
