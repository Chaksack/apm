package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var StatusCmd = &cobra.Command{
	Use:   "status [deployment-id]",
	Short: "Check deployment status and health",
	Long: `Check the status of APM deployments and monitor their health.
If no deployment ID is specified, shows the status of the most recent deployment.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runStatus,
}

var (
	watchStatus    bool
	watchInterval  int
	statusJSON     bool
	statusVerbose  bool
	allDeployments bool
)

type statusModel struct {
	deployments []deploymentStatus
	current     int
	watching    bool
	interval    time.Duration
	err         error
	lastUpdate  time.Time
	width       int
	height      int
}

type deploymentStatus struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Status      string            `json:"status"`
	Health      string            `json:"health"`
	StartTime   time.Time         `json:"start_time"`
	Duration    time.Duration     `json:"duration,omitempty"`
	Progress    int               `json:"progress"`
	Components  []componentStatus `json:"components"`
	Resources   resourceInfo      `json:"resources"`
	Endpoints   map[string]string `json:"endpoints"`
	Metrics     deploymentMetrics `json:"metrics,omitempty"`
	LastChecked time.Time         `json:"last_checked"`
	Message     string            `json:"message,omitempty"`
	Error       string            `json:"error,omitempty"`
}

type componentStatus struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Status  string `json:"status"`
	Health  string `json:"health"`
	Version string `json:"version,omitempty"`
	Message string `json:"message,omitempty"`
}

type resourceInfo struct {
	CPU      string `json:"cpu,omitempty"`
	Memory   string `json:"memory,omitempty"`
	Storage  string `json:"storage,omitempty"`
	Replicas int    `json:"replicas,omitempty"`
	Nodes    int    `json:"nodes,omitempty"`
}

type deploymentMetrics struct {
	RequestRate    float64 `json:"request_rate,omitempty"`
	ErrorRate      float64 `json:"error_rate,omitempty"`
	ResponseTime   float64 `json:"response_time_ms,omitempty"`
	Availability   float64 `json:"availability,omitempty"`
	ActiveSessions int     `json:"active_sessions,omitempty"`
}

func init() {
	StatusCmd.Flags().BoolVarP(&watchStatus, "watch", "w", false, "Continuously watch deployment status")
	StatusCmd.Flags().IntVar(&watchInterval, "interval", 5, "Watch interval in seconds")
	StatusCmd.Flags().BoolVar(&statusJSON, "json", false, "Output status in JSON format")
	StatusCmd.Flags().BoolVarP(&statusVerbose, "verbose", "v", false, "Show detailed status information")
	StatusCmd.Flags().BoolVarP(&allDeployments, "all", "a", false, "Show all deployments")
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Load configuration
	config := viper.New()
	config.SetConfigName("apm")
	config.SetConfigType("yaml")
	config.AddConfigPath(".")

	if err := config.ReadInConfig(); err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	// Get deployment ID
	deploymentID := ""
	if len(args) > 0 {
		deploymentID = args[0]
	}

	// Get deployment statuses
	statuses, err := getDeploymentStatuses(deploymentID, config)
	if err != nil {
		return err
	}

	if len(statuses) == 0 {
		fmt.Println("No deployments found.")
		return nil
	}

	// Handle JSON output
	if statusJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if allDeployments {
			return encoder.Encode(statuses)
		}
		return encoder.Encode(statuses[0])
	}

	// Handle watch mode
	if watchStatus {
		model := statusModel{
			deployments: statuses,
			watching:    true,
			interval:    time.Duration(watchInterval) * time.Second,
			lastUpdate:  time.Now(),
		}

		p := tea.NewProgram(model)
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("error running status watcher: %w", err)
		}
		return nil
	}

	// Display status once
	if allDeployments {
		displayAllStatuses(statuses)
	} else {
		displayDetailedStatus(statuses[0])
	}

	return nil
}

func getDeploymentStatuses(deploymentID string, config *viper.Viper) ([]deploymentStatus, error) {
	// This would integrate with the deploy package to get real status
	// For now, we'll simulate based on configuration and deployment history

	// TODO: Integrate with actual deployment service when available
	// deployService := deploy.GetDeploymentService()
	// if deployService != nil {
	//     return getStatusFromService(deployService, deploymentID)
	// }

	// Fallback to mock data for demonstration
	return getMockStatuses(deploymentID, config)
}

// TODO: Implement these functions when deploy.Service is available
// func getStatusFromService(service *deploy.Service, deploymentID string) ([]deploymentStatus, error) {
// 	ctx := context.Background()
//
// 	if deploymentID != "" {
// 		// Get specific deployment
// 		deployment, err := service.GetDeployment(ctx, deploymentID)
// 		if err != nil {
// 			return nil, err
// 		}
// 		return []deploymentStatus{convertDeployment(deployment)}, nil
// 	}
//
// 	// Get all deployments
// 	deployments, err := service.ListDeployments(ctx, 10)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	statuses := make([]deploymentStatus, len(deployments))
// 	for i, d := range deployments {
// 		statuses[i] = convertDeployment(d)
// 	}
//
// 	return statuses, nil
// }
//
// func convertDeployment(d *deploy.Deployment) deploymentStatus {
// 	status := deploymentStatus{
// 		ID:        d.ID,
// 		Name:      d.Name,
// 		Type:      string(d.Type),
// 		Status:    string(d.Status),
// 		Health:    string(d.Health.Status),
// 		StartTime: d.StartedAt,
// 		Progress:  d.Progress,
// 		Message:   d.Health.Message,
// 	}
//
// 	if d.CompletedAt != nil {
// 		status.Duration = d.CompletedAt.Sub(d.StartedAt)
// 	}
//
// 	// Convert components
// 	for _, comp := range d.Components {
// 		status.Components = append(status.Components, componentStatus{
// 			Name:    comp.Name,
// 			Type:    comp.Type,
// 			Status:  string(comp.Status),
// 			Health:  string(comp.Health),
// 			Version: comp.Version,
// 			Message: comp.Message,
// 		})
// 	}
//
// 	return status
// }

func getMockStatuses(deploymentID string, config *viper.Viper) ([]deploymentStatus, error) {
	// Mock data for demonstration
	mockStatus := deploymentStatus{
		ID:        "dep-" + time.Now().Format("20060102-150405"),
		Name:      config.GetString("project.name"),
		Type:      "kubernetes",
		Status:    "running",
		Health:    "healthy",
		StartTime: time.Now().Add(-2 * time.Hour),
		Duration:  2 * time.Hour,
		Progress:  100,
		Components: []componentStatus{
			{Name: "app", Type: "deployment", Status: "running", Health: "healthy", Version: "1.0.0"},
			{Name: "prometheus", Type: "statefulset", Status: "running", Health: "healthy"},
			{Name: "grafana", Type: "deployment", Status: "running", Health: "healthy"},
		},
		Resources: resourceInfo{
			CPU:      "2 cores",
			Memory:   "4 GB",
			Storage:  "10 GB",
			Replicas: 3,
			Nodes:    2,
		},
		Endpoints: map[string]string{
			"app":        "https://app.example.com",
			"prometheus": "https://prometheus.example.com",
			"grafana":    "https://grafana.example.com",
		},
		Metrics: deploymentMetrics{
			RequestRate:    1250.5,
			ErrorRate:      0.01,
			ResponseTime:   45.2,
			Availability:   99.99,
			ActiveSessions: 342,
		},
		LastChecked: time.Now(),
	}

	if deploymentID != "" && deploymentID != mockStatus.ID {
		return nil, fmt.Errorf("deployment %s not found", deploymentID)
	}

	return []deploymentStatus{mockStatus}, nil
}

// Tea Model implementation for watch mode
func (m statusModel) Init() tea.Cmd {
	return tick()
}

func (m statusModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "r":
			return m, refreshStatus(&m)
		case "up", "k":
			if m.current > 0 {
				m.current--
			}
		case "down", "j":
			if m.current < len(m.deployments)-1 {
				m.current++
			}
		}

	case tickMsg:
		m.lastUpdate = time.Now()
		return m, tea.Batch(
			refreshStatus(&m),
			tick(),
		)

	case statusUpdatedMsg:
		m.deployments = msg.statuses
		return m, nil
	}

	return m, nil
}

func (m statusModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress 'q' to quit.", m.err)
	}

	if len(m.deployments) == 0 {
		return "No deployments found.\n\nPress 'q' to quit."
	}

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		MarginBottom(1)

	s := titleStyle.Render("📊 Deployment Status") + "\n\n"

	// Show current deployment details
	if m.current < len(m.deployments) {
		s += renderDeploymentStatus(m.deployments[m.current], true)
	}

	// Show navigation info
	if len(m.deployments) > 1 {
		navStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(2)
		s += navStyle.Render(fmt.Sprintf("\nShowing %d of %d deployments. Use ↑/↓ to navigate.", m.current+1, len(m.deployments)))
	}

	// Show last update time
	updateStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)
	s += updateStyle.Render(fmt.Sprintf("\nLast updated: %s (refreshing every %s)",
		m.lastUpdate.Format("15:04:05"), m.interval))

	// Instructions
	s += "\n\n[r] Refresh  [q] Quit"

	return s
}

// Tea commands
type tickMsg time.Time
type statusUpdatedMsg struct {
	statuses []deploymentStatus
}

func tick() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func refreshStatus(m *statusModel) tea.Cmd {
	return func() tea.Msg {
		// Refresh status
		config := viper.New()
		config.SetConfigName("apm")
		config.SetConfigType("yaml")
		config.AddConfigPath(".")
		config.ReadInConfig()

		statuses, err := getDeploymentStatuses("", config)
		if err != nil {
			m.err = err
			return nil
		}

		return statusUpdatedMsg{statuses: statuses}
	}
}

// Display functions
func displayAllStatuses(statuses []deploymentStatus) {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		MarginBottom(1)

	fmt.Println(titleStyle.Render("📊 All Deployments"))
	fmt.Println()

	// Table header
	headerStyle := lipgloss.NewStyle().Bold(true)
	fmt.Printf("%-20s %-15s %-10s %-10s %-20s %-15s\n",
		headerStyle.Render("ID"),
		headerStyle.Render("Name"),
		headerStyle.Render("Type"),
		headerStyle.Render("Status"),
		headerStyle.Render("Started"),
		headerStyle.Render("Duration"),
	)
	fmt.Println(strings.Repeat("-", 100))

	// Table rows
	for _, status := range statuses {
		statusColor := getStatusColor(status.Status)
		statusStyle := lipgloss.NewStyle().Foreground(statusColor)

		duration := "Running"
		if status.Duration > 0 {
			duration = formatDuration(status.Duration)
		}

		fmt.Printf("%-20s %-15s %-10s %-10s %-20s %-15s\n",
			status.ID,
			status.Name,
			status.Type,
			statusStyle.Render(status.Status),
			status.StartTime.Format("2006-01-02 15:04"),
			duration,
		)
	}
}

func displayDetailedStatus(status deploymentStatus) {
	fmt.Print(renderDeploymentStatus(status, statusVerbose))
}

func renderDeploymentStatus(status deploymentStatus, verbose bool) string {
	var b strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	b.WriteString(headerStyle.Render(fmt.Sprintf("Deployment: %s", status.Name)) + "\n")
	b.WriteString(strings.Repeat("─", 50) + "\n\n")

	// Basic info
	statusColor := getStatusColor(status.Status)
	statusStyle := lipgloss.NewStyle().Foreground(statusColor).Bold(true)
	healthColor := getHealthColor(status.Health)
	healthStyle := lipgloss.NewStyle().Foreground(healthColor).Bold(true)

	b.WriteString(fmt.Sprintf("ID:       %s\n", status.ID))
	b.WriteString(fmt.Sprintf("Type:     %s\n", status.Type))
	b.WriteString(fmt.Sprintf("Status:   %s\n", statusStyle.Render(status.Status)))
	b.WriteString(fmt.Sprintf("Health:   %s\n", healthStyle.Render(status.Health)))
	b.WriteString(fmt.Sprintf("Started:  %s\n", status.StartTime.Format("2006-01-02 15:04:05")))

	if status.Duration > 0 {
		b.WriteString(fmt.Sprintf("Duration: %s\n", formatDuration(status.Duration)))
	}

	if status.Progress < 100 && status.Progress > 0 {
		b.WriteString(fmt.Sprintf("Progress: %d%%\n", status.Progress))
	}

	if status.Message != "" {
		b.WriteString(fmt.Sprintf("Message:  %s\n", status.Message))
	}

	if status.Error != "" {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		b.WriteString(fmt.Sprintf("Error:    %s\n", errorStyle.Render(status.Error)))
	}

	// Components
	b.WriteString("\n" + headerStyle.Render("Components:") + "\n")
	for _, comp := range status.Components {
		compStatusColor := getStatusColor(comp.Status)
		compStatusStyle := lipgloss.NewStyle().Foreground(compStatusColor)

		b.WriteString(fmt.Sprintf("  %-20s %-15s %s",
			comp.Name,
			comp.Type,
			compStatusStyle.Render(comp.Status),
		))

		if comp.Version != "" {
			b.WriteString(fmt.Sprintf(" (v%s)", comp.Version))
		}

		if comp.Message != "" && verbose {
			b.WriteString(fmt.Sprintf("\n    %s", comp.Message))
		}

		b.WriteString("\n")
	}

	// Resources
	if verbose {
		b.WriteString("\n" + headerStyle.Render("Resources:") + "\n")
		if status.Resources.CPU != "" {
			b.WriteString(fmt.Sprintf("  CPU:      %s\n", status.Resources.CPU))
		}
		if status.Resources.Memory != "" {
			b.WriteString(fmt.Sprintf("  Memory:   %s\n", status.Resources.Memory))
		}
		if status.Resources.Storage != "" {
			b.WriteString(fmt.Sprintf("  Storage:  %s\n", status.Resources.Storage))
		}
		if status.Resources.Replicas > 0 {
			b.WriteString(fmt.Sprintf("  Replicas: %d\n", status.Resources.Replicas))
		}
		if status.Resources.Nodes > 0 {
			b.WriteString(fmt.Sprintf("  Nodes:    %d\n", status.Resources.Nodes))
		}
	}

	// Endpoints
	if len(status.Endpoints) > 0 {
		b.WriteString("\n" + headerStyle.Render("Endpoints:") + "\n")
		for name, url := range status.Endpoints {
			b.WriteString(fmt.Sprintf("  %-15s %s\n", name+":", url))
		}
	}

	// Metrics
	if verbose && status.Metrics.RequestRate > 0 {
		b.WriteString("\n" + headerStyle.Render("Metrics:") + "\n")
		b.WriteString(fmt.Sprintf("  Request Rate:    %.1f req/s\n", status.Metrics.RequestRate))
		b.WriteString(fmt.Sprintf("  Error Rate:      %.2f%%\n", status.Metrics.ErrorRate*100))
		b.WriteString(fmt.Sprintf("  Response Time:   %.1f ms\n", status.Metrics.ResponseTime))
		b.WriteString(fmt.Sprintf("  Availability:    %.2f%%\n", status.Metrics.Availability))
		b.WriteString(fmt.Sprintf("  Active Sessions: %d\n", status.Metrics.ActiveSessions))
	}

	return b.String()
}

func getStatusColor(status string) lipgloss.Color {
	switch strings.ToLower(status) {
	case "running", "deployed", "active", "healthy":
		return lipgloss.Color("42") // Green
	case "pending", "deploying", "starting", "updating":
		return lipgloss.Color("214") // Yellow
	case "failed", "error", "crashed", "unhealthy":
		return lipgloss.Color("196") // Red
	case "stopped", "terminated", "completed":
		return lipgloss.Color("241") // Gray
	default:
		return lipgloss.Color("86") // Blue
	}
}

func getHealthColor(health string) lipgloss.Color {
	switch strings.ToLower(health) {
	case "healthy":
		return lipgloss.Color("42") // Green
	case "degraded", "warning":
		return lipgloss.Color("214") // Yellow
	case "unhealthy", "critical":
		return lipgloss.Color("196") // Red
	default:
		return lipgloss.Color("241") // Gray
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	} else {
		days := d.Hours() / 24
		return fmt.Sprintf("%.1fd", days)
	}
}
