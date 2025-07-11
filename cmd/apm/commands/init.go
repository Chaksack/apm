package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yourusername/apm/pkg/security"
)

var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize APM configuration with interactive setup",
	Long: `Initialize APM configuration through an interactive wizard that guides you through:
- Selecting APM tools to integrate (Prometheus, Grafana, Jaeger, Loki, etc.)
- Configuring essential parameters for each tool
- Specifying custom configuration paths
- Setting up environment-specific configurations`,
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	// Check if apm.yaml already exists
	configPath := "apm.yaml"
	if _, err := os.Stat(configPath); err == nil {
		fmt.Println("Found existing apm.yaml configuration.")
		fmt.Println("Running init will update your existing configuration.")
	}

	// Create and run the wizard
	wizard := newInitWizard()
	p := tea.NewProgram(wizard, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("error running wizard: %w", err)
	}

	// Get the final configuration
	if m, ok := finalModel.(initWizard); ok && m.completed {
		// Add wizard data to config for saving
		m.config["wizard"] = m
		if err := saveConfiguration(m.config); err != nil {
			return fmt.Errorf("error saving configuration: %w", err)
		}

		fmt.Println("\n✅ APM configuration saved to apm.yaml")
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Run 'apm test' to validate your configuration")
		fmt.Println("  2. Run 'apm run' to start your application with APM")
		fmt.Println("  3. Run 'apm dashboard' to access monitoring interfaces")

		if m.slackEnabled {
			fmt.Println("\n💬 Slack notifications configured for alerts!")
		}
	}

	return nil
}

// Wizard state management
type screen int

const (
	screenWelcome screen = iota
	screenProjectType
	screenComponents
	screenPrometheus
	screenGrafana
	screenJaeger
	screenLoki
	screenNotifications
	screenSlack
	screenEnvironment
	screenReview
	screenComplete
)

type initWizard struct {
	screen          screen
	config          map[string]interface{}
	selections      map[string]bool
	currentInput    string
	err             error
	completed       bool
	width           int
	height          int
	slackWebhook    string
	slackChannel    string
	slackEnabled    bool
	notifySelection int // 0: None, 1: Slack, 2: Email (future)
}

func newInitWizard() initWizard {
	return initWizard{
		screen: screenWelcome,
		config: make(map[string]interface{}),
		selections: map[string]bool{
			"prometheus": true,
			"grafana":    true,
			"jaeger":     false,
			"loki":       false,
		},
		slackChannel:    "#alerts",
		notifySelection: 0,
	}
}

// Tea Model interface implementation
func (m initWizard) Init() tea.Cmd {
	return tea.SetWindowTitle("APM Configuration Wizard")
}

func (m initWizard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "enter":
			return m.handleEnter()

		case "tab", "down", "j":
			if m.screen == screenNotifications && m.notifySelection < 2 {
				m.notifySelection++
				return m, nil
			}
			return m.handleNext()

		case "shift+tab", "up", "k":
			if m.screen == screenNotifications && m.notifySelection > 0 {
				m.notifySelection--
				return m, nil
			}
			return m.handlePrev()

		case " ":
			return m.handleSpace()

		case "backspace":
			if len(m.currentInput) > 0 {
				m.currentInput = m.currentInput[:len(m.currentInput)-1]
			}
			return m, nil

		default:
			if m.screen == screenProjectType || m.screen == screenSlack {
				m.currentInput += msg.String()
			}
			return m, nil
		}
	}

	return m, nil
}

func (m initWizard) View() string {
	if m.err != nil {
		return renderError(m.err)
	}

	switch m.screen {
	case screenWelcome:
		return renderWelcome()
	case screenProjectType:
		return renderProjectType(m)
	case screenComponents:
		return renderComponents(m)
	case screenPrometheus:
		return renderPrometheusConfig(m)
	case screenGrafana:
		return renderGrafanaConfig(m)
	case screenJaeger:
		return renderJaegerConfig(m)
	case screenLoki:
		return renderLokiConfig(m)
	case screenNotifications:
		return renderNotifications(m)
	case screenSlack:
		return renderSlackConfig(m)
	case screenEnvironment:
		return renderEnvironment(m)
	case screenReview:
		return renderReview(m)
	case screenComplete:
		return renderComplete()
	default:
		return "Unknown screen"
	}
}

// Screen renderers
func renderWelcome() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		MarginTop(2).
		MarginBottom(2)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginBottom(2)

	return titleStyle.Render("🚀 Welcome to APM Configuration Wizard") + "\n" +
		descStyle.Render("This wizard will help you set up Application Performance Monitoring\nfor your GoFiber application.") + "\n\n" +
		"Press [Enter] to continue..."
}

func renderProjectType(m initWizard) string {
	// Simplified for now - we'll enhance this with proper form handling
	return "📁 Project Configuration\n\n" +
		"Project Name: " + m.currentInput + "_\n\n" +
		"Press [Enter] to continue..."
}

func renderComponents(m initWizard) string {
	s := "🔧 Select APM Components\n\n"
	s += "Use [Space] to toggle, [Enter] to continue\n\n"

	components := []string{"prometheus", "grafana", "jaeger", "loki"}
	for _, comp := range components {
		if m.selections[comp] {
			s += fmt.Sprintf("[✓] %s\n", comp)
		} else {
			s += fmt.Sprintf("[ ] %s\n", comp)
		}
	}

	return s
}

func renderPrometheusConfig(m initWizard) string {
	return "📊 Prometheus Configuration\n\n" +
		"Port: 9090\n" +
		"Scrape Interval: 15s\n\n" +
		"Press [Enter] to continue..."
}

func renderGrafanaConfig(m initWizard) string {
	return "📈 Grafana Configuration\n\n" +
		"Port: 3000\n" +
		"Admin Password: (will be generated)\n\n" +
		"Press [Enter] to continue..."
}

func renderJaegerConfig(m initWizard) string {
	return "🔍 Jaeger Configuration\n\n" +
		"Agent Port: 6831\n" +
		"UI Port: 16686\n\n" +
		"Press [Enter] to continue..."
}

func renderLokiConfig(m initWizard) string {
	return "📝 Loki Configuration\n\n" +
		"Port: 3100\n" +
		"Retention: 7d\n\n" +
		"Press [Enter] to continue..."
}

func renderNotifications(m initWizard) string {
	style := lipgloss.NewStyle().MarginBottom(1)
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)

	s := "🔔 Notification Configuration\n\n"
	s += "Select notification method for alerts:\n\n"

	options := []string{"None", "Slack", "Email (Coming Soon)"}
	for i, opt := range options {
		prefix := "  "
		if i == m.notifySelection {
			prefix = "▸ "
			s += selectedStyle.Render(prefix+opt) + "\n"
		} else {
			s += style.Render(prefix+opt) + "\n"
		}
	}

	s += "\nUse [↑/↓] to select, [Enter] to continue..."
	return s
}

func renderSlackConfig(m initWizard) string {
	inputStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	s := "💬 Slack Configuration\n\n"

	// Show webhook URL input
	s += labelStyle.Render("Webhook URL:") + "\n"
	if m.currentInput != "" || m.slackWebhook != "" {
		webhookDisplay := m.slackWebhook
		if m.screen == screenSlack && m.currentInput != "" {
			webhookDisplay = m.currentInput
		}
		s += inputStyle.Render(webhookDisplay) + "_\n\n"
	} else {
		s += inputStyle.Render("https://hooks.slack.com/services/...") + "_\n\n"
	}

	// Show channel
	s += labelStyle.Render("Channel:") + " " + inputStyle.Render(m.slackChannel) + "\n\n"

	s += "Enter your Slack webhook URL and press [Enter] to continue...\n"
	s += "Get webhook URL from: https://api.slack.com/messaging/webhooks"

	return s
}

func renderEnvironment(m initWizard) string {
	return "🌍 Environment Configuration\n\n" +
		"Environment: development\n" +
		"Config Path: ./apm.yaml\n\n" +
		"Press [Enter] to continue..."
}

func renderReview(m initWizard) string {
	s := "📋 Configuration Review\n\n"

	// Project info
	projectName := "my-app"
	if name, ok := m.config["project_name"].(string); ok && name != "" {
		projectName = name
	}
	s += fmt.Sprintf("Project: %s\n", projectName)

	// Components
	components := []string{}
	for comp, enabled := range m.selections {
		if enabled {
			components = append(components, comp)
		}
	}
	s += fmt.Sprintf("Components: %s\n", strings.Join(components, ", "))

	// Notifications
	if m.slackEnabled && m.slackWebhook != "" {
		s += fmt.Sprintf("Notifications: Slack (%s)\n", m.slackChannel)
	} else {
		s += "Notifications: None\n"
	}

	s += "Environment: development\n\n"
	s += "Press [Enter] to save configuration..."

	return s
}

func renderComplete() string {
	return "✅ Configuration Complete!\n\n" +
		"Your APM configuration has been saved to apm.yaml\n\n" +
		"Press [q] to exit..."
}

func renderError(err error) string {
	return fmt.Sprintf("❌ Error: %v\n\nPress [q] to exit...", err)
}

// Navigation handlers
func (m initWizard) handleEnter() (initWizard, tea.Cmd) {
	switch m.screen {
	case screenWelcome:
		m.screen = screenProjectType
	case screenProjectType:
		m.config["project_name"] = m.currentInput
		m.currentInput = ""
		m.screen = screenComponents
	case screenComponents:
		m.screen = screenPrometheus
	case screenPrometheus:
		if m.selections["grafana"] {
			m.screen = screenGrafana
		} else if m.selections["jaeger"] {
			m.screen = screenJaeger
		} else if m.selections["loki"] {
			m.screen = screenLoki
		} else {
			m.screen = screenNotifications
		}
	case screenGrafana:
		if m.selections["jaeger"] {
			m.screen = screenJaeger
		} else if m.selections["loki"] {
			m.screen = screenLoki
		} else {
			m.screen = screenNotifications
		}
	case screenJaeger:
		if m.selections["loki"] {
			m.screen = screenLoki
		} else {
			m.screen = screenNotifications
		}
	case screenLoki:
		m.screen = screenNotifications
	case screenNotifications:
		if m.notifySelection == 1 { // Slack selected
			m.screen = screenSlack
		} else {
			m.screen = screenEnvironment
		}
	case screenSlack:
		m.slackWebhook = m.currentInput
		m.currentInput = ""
		m.slackEnabled = true
		m.screen = screenEnvironment
	case screenEnvironment:
		m.screen = screenReview
	case screenReview:
		m.completed = true
		m.screen = screenComplete
		return m, tea.Quit
	case screenComplete:
		return m, tea.Quit
	}
	return m, nil
}

func (m initWizard) handleNext() (initWizard, tea.Cmd) {
	// Handle navigation within screens
	return m, nil
}

func (m initWizard) handlePrev() (initWizard, tea.Cmd) {
	// Handle navigation within screens
	return m, nil
}

func (m initWizard) handleSpace() (initWizard, tea.Cmd) {
	// Toggle selections in component screen
	if m.screen == screenComponents {
		// This would toggle the current selection
	}
	return m, nil
}

// Configuration saving
func saveConfiguration(config map[string]interface{}) error {
	// Extract wizard data
	m := config["wizard"].(initWizard)

	// Create default configuration structure
	fullConfig := map[string]interface{}{
		"version": "1.0",
		"project": map[string]interface{}{
			"name":        config["project_name"],
			"environment": "development",
		},
		"apm": map[string]interface{}{
			"prometheus": map[string]interface{}{
				"enabled": m.selections["prometheus"],
				"port":    9090,
				"config": map[string]interface{}{
					"scrape_interval": "15s",
					"scrape_configs": []interface{}{
						map[string]interface{}{
							"job_name": "app",
							"static_configs": []interface{}{
								map[string]interface{}{
									"targets": []string{"localhost:8080"},
								},
							},
						},
					},
				},
			},
			"grafana": map[string]interface{}{
				"enabled": m.selections["grafana"],
				"port":    3000,
				"config": map[string]interface{}{
					"security": map[string]interface{}{
						"admin_password": generateDefaultPassword(),
					},
					"datasources": []interface{}{
						map[string]interface{}{
							"name": "Prometheus",
							"type": "prometheus",
							"url":  "http://localhost:9090",
						},
					},
				},
			},
			"jaeger": map[string]interface{}{
				"enabled":    m.selections["jaeger"],
				"agent_port": 6831,
				"ui_port":    16686,
			},
			"loki": map[string]interface{}{
				"enabled":   m.selections["loki"],
				"port":      3100,
				"retention": "7d",
			},
			"alertmanager": map[string]interface{}{
				"enabled": m.selections["prometheus"] && m.slackEnabled,
				"port":    9093,
				"config": map[string]interface{}{
					"receivers": []interface{}{
						map[string]interface{}{
							"name": "default",
							"slack_configs": []interface{}{
								map[string]interface{}{
									"api_url": m.slackWebhook,
									"channel": m.slackChannel,
									"title":   "APM Alert",
									"text":    "{{ range .Alerts }}{{ .Annotations.summary }}\n{{ end }}",
								},
							},
						},
					},
					"route": map[string]interface{}{
						"receiver": "default",
					},
				},
			},
		},
		"notifications": map[string]interface{}{
			"slack": map[string]interface{}{
				"enabled":     m.slackEnabled,
				"webhook_url": m.slackWebhook,
				"channel":     m.slackChannel,
			},
		},
		"application": map[string]interface{}{
			"entry_point":   "./cmd/app/main.go",
			"build_command": "go build",
			"run_command":   "./app",
			"hot_reload": map[string]interface{}{
				"enabled":    true,
				"paths":      []string{"."},
				"exclude":    []string{"vendor", "node_modules", ".git"},
				"extensions": []string{".go", ".mod"},
			},
		},
	}

	// Save using viper
	v := viper.New()
	v.SetConfigType("yaml")

	for k, val := range fullConfig {
		v.Set(k, val)
	}

	configPath := filepath.Join(".", "apm.yaml")
	return v.WriteConfigAs(configPath)
}

// generateDefaultPassword generates a secure default password
func generateDefaultPassword() string {
	password, err := security.GenerateSecurePassword(16)
	if err != nil {
		// Fallback to a static but more secure default
		return "APM-Secure-2024!"
	}

	// Display the generated password to the user
	fmt.Printf("\n🔐 Generated secure Grafana admin password: %s\n", password)
	fmt.Println("⚠️  Please save this password in a secure location!")

	return password
}
