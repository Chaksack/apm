package commands

import (
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var DashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Access APM monitoring interfaces",
	Long: `Display a list of all configured APM tool web interfaces and provide quick access to them.
Select a tool to automatically open its web interface in your default browser.`,
	RunE: runDashboard,
}

type tool struct {
	name      string
	url       string
	port      int
	available bool
	status    string
}

type dashboardModel struct {
	tools    []tool
	cursor   int
	checking bool
	err      error
	width    int
	height   int
}

func runDashboard(cmd *cobra.Command, args []string) error {
	// Load configuration
	config := viper.New()
	config.SetConfigName("apm")
	config.SetConfigType("yaml")
	config.AddConfigPath(".")

	if err := config.ReadInConfig(); err != nil {
		return fmt.Errorf("error reading config file: %w. Run 'apm init' first", err)
	}

	// Create list of tools
	tools := []tool{}

	if config.GetBool("apm.prometheus.enabled") {
		port := config.GetInt("apm.prometheus.port")
		if port == 0 {
			port = 9090
		}
		tools = append(tools, tool{
			name: "Prometheus",
			url:  fmt.Sprintf("http://localhost:%d", port),
			port: port,
		})
	}

	if config.GetBool("apm.grafana.enabled") {
		port := config.GetInt("apm.grafana.port")
		if port == 0 {
			port = 3000
		}
		tools = append(tools, tool{
			name: "Grafana",
			url:  fmt.Sprintf("http://localhost:%d", port),
			port: port,
		})
	}

	if config.GetBool("apm.jaeger.enabled") {
		port := config.GetInt("apm.jaeger.ui_port")
		if port == 0 {
			port = 16686
		}
		tools = append(tools, tool{
			name: "Jaeger",
			url:  fmt.Sprintf("http://localhost:%d", port),
			port: port,
		})
	}

	if config.GetBool("apm.loki.enabled") {
		port := config.GetInt("apm.loki.port")
		if port == 0 {
			port = 3100
		}
		tools = append(tools, tool{
			name: "Loki",
			url:  fmt.Sprintf("http://localhost:%d", port),
			port: port,
		})
	}

	// Add AlertManager if configured
	if config.IsSet("apm.alertmanager.enabled") && config.GetBool("apm.alertmanager.enabled") {
		port := config.GetInt("apm.alertmanager.port")
		if port == 0 {
			port = 9093
		}
		tools = append(tools, tool{
			name: "AlertManager",
			url:  fmt.Sprintf("http://localhost:%d", port),
			port: port,
		})
	}

	if len(tools) == 0 {
		fmt.Println("No APM tools are enabled in your configuration.")
		fmt.Println("Run 'apm init' to configure APM tools.")
		return nil
	}

	// Create and run the interactive dashboard
	m := dashboardModel{
		tools:    tools,
		checking: true,
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running dashboard: %w", err)
	}

	return nil
}

func (m dashboardModel) Init() tea.Cmd {
	return checkToolsCmd(m.tools)
}

func (m dashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case toolsCheckedMsg:
		m.tools = msg
		m.checking = false

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.tools)-1 {
				m.cursor++
			}

		case "enter", " ":
			if !m.checking && m.cursor < len(m.tools) {
				tool := m.tools[m.cursor]
				if tool.available {
					openBrowser(tool.url)
				}
			}

		case "r":
			m.checking = true
			return m, checkToolsCmd(m.tools)
		}
	}

	return m, nil
}

func (m dashboardModel) View() string {
	if m.checking {
		return renderChecking()
	}

	return renderDashboard(m)
}

func renderChecking() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(2).
		MarginLeft(2)

	return style.Render("🔍 Checking APM tools availability...")
}

func renderDashboard(m dashboardModel) string {
	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		MarginBottom(2)

	// Status styles
	availableStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true)

	unavailableStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)

	s := titleStyle.Render("🎯 APM Dashboard") + "\n\n"

	// Tool list
	for i, tool := range m.tools {
		cursor := "  "
		if i == m.cursor {
			cursor = "▸ "
		}

		status := unavailableStyle.Render("✗ Offline")
		if tool.available {
			status = availableStyle.Render("✓ Online")
		}

		line := fmt.Sprintf("%s%-15s %-15s Port: %-5s",
			cursor,
			tool.name,
			status,
			strconv.Itoa(tool.port),
		)

		if i == m.cursor {
			s += selectedStyle.Render(line) + "\n"
		} else {
			s += line + "\n"
		}
	}

	// Instructions
	instructionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(2)

	instructions := []string{
		"",
		"[↑/↓] Navigate  [Enter] Open in browser  [r] Refresh  [q] Quit",
	}

	for _, inst := range instructions {
		s += instructionStyle.Render(inst) + "\n"
	}

	return s
}

// Tool checking
type toolsCheckedMsg []tool

func checkToolsCmd(tools []tool) tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{Timeout: 1 * time.Second}

		for i := range tools {
			// Check if tool is available
			resp, err := client.Get(tools[i].url)
			if err == nil {
				tools[i].available = true
				resp.Body.Close()
			} else {
				tools[i].available = false
			}
		}

		return toolsCheckedMsg(tools)
	}
}

// Browser opening
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
		args = []string{url}
	}

	return exec.Command(cmd, args...).Start()
}

func init() {
	DashboardCmd.Flags().StringP("config", "c", "apm.yaml", "Path to configuration file")
}
