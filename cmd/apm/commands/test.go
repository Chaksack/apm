package commands

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var TestCmd = &cobra.Command{
	Use:   "test",
	Short: "Validate APM configuration and perform health checks",
	Long: `Validate the APM configuration file and perform connectivity tests for all configured tools.
This includes checking syntax, required parameters, and testing connections to Prometheus, Grafana, Jaeger, and Loki.`,
	RunE: runTest,
}

type testResult struct {
	name    string
	status  string
	message string
	passed  bool
}

func runTest(cmd *cobra.Command, args []string) error {
	// Style definitions
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		MarginBottom(1)

	passStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true)

	failStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)

	fmt.Println(titleStyle.Render("🧪 APM Configuration Test"))
	fmt.Println()

	// Load configuration
	config := viper.New()
	config.SetConfigName("apm")
	config.SetConfigType("yaml")
	config.AddConfigPath(".")

	results := []testResult{}

	// Test 1: Configuration file exists and is valid
	configTest := testConfigFile(config)
	results = append(results, configTest)
	renderTestResult(configTest, passStyle, failStyle)

	if !configTest.passed {
		fmt.Println("\n❌ Configuration file test failed. Please run 'apm init' first.")
		return nil
	}

	// Test 2: Required fields validation
	validationTest := testRequiredFields(config)
	results = append(results, validationTest)
	renderTestResult(validationTest, passStyle, failStyle)

	// Test 3: Prometheus connectivity
	if config.GetBool("apm.prometheus.enabled") {
		promTest := testPrometheus(config)
		results = append(results, promTest)
		renderTestResult(promTest, passStyle, failStyle)
	}

	// Test 4: Grafana connectivity
	if config.GetBool("apm.grafana.enabled") {
		grafanaTest := testGrafana(config)
		results = append(results, grafanaTest)
		renderTestResult(grafanaTest, passStyle, failStyle)
	}

	// Test 5: Jaeger connectivity
	if config.GetBool("apm.jaeger.enabled") {
		jaegerTest := testJaeger(config)
		results = append(results, jaegerTest)
		renderTestResult(jaegerTest, passStyle, failStyle)
	}

	// Test 6: Loki connectivity
	if config.GetBool("apm.loki.enabled") {
		lokiTest := testLoki(config)
		results = append(results, lokiTest)
		renderTestResult(lokiTest, passStyle, failStyle)
	}

	// Test 7: Slack webhook validation
	if config.GetBool("notifications.slack.enabled") {
		slackTest := testSlackWebhook(config)
		results = append(results, slackTest)
		renderTestResult(slackTest, passStyle, failStyle)
	}

	// Test 8: Application entry point
	appTest := testApplicationEntry(config)
	results = append(results, appTest)
	renderTestResult(appTest, passStyle, failStyle)

	// Summary
	passed := 0
	failed := 0
	for _, r := range results {
		if r.passed {
			passed++
		} else {
			failed++
		}
	}

	fmt.Println("\n" + strings.Repeat("─", 50))
	summaryStyle := lipgloss.NewStyle().Bold(true)
	if failed == 0 {
		fmt.Println(summaryStyle.Foreground(lipgloss.Color("42")).Render(
			fmt.Sprintf("✅ All tests passed! (%d/%d)", passed, len(results))))
		fmt.Println("\nYour APM configuration is valid and all services are reachable.")
		fmt.Println("You can now run 'apm run' to start your application with APM.")
	} else {
		fmt.Println(summaryStyle.Foreground(lipgloss.Color("196")).Render(
			fmt.Sprintf("❌ Some tests failed: %d passed, %d failed", passed, failed)))
		fmt.Println("\nPlease fix the issues above before running your application.")
	}

	return nil
}

func renderTestResult(result testResult, passStyle, failStyle lipgloss.Style) {
	status := failStyle.Render("✗ FAIL")
	if result.passed {
		status = passStyle.Render("✓ PASS")
	}

	fmt.Printf("%-40s %s\n", result.name, status)
	if result.message != "" {
		fmt.Printf("  └─ %s\n", result.message)
	}
}

func testConfigFile(config *viper.Viper) testResult {
	err := config.ReadInConfig()
	if err != nil {
		if os.IsNotExist(err) {
			return testResult{
				name:    "Configuration file (apm.yaml)",
				status:  "FAIL",
				message: "File not found. Run 'apm init' to create it.",
				passed:  false,
			}
		}
		return testResult{
			name:    "Configuration file (apm.yaml)",
			status:  "FAIL",
			message: fmt.Sprintf("Parse error: %v", err),
			passed:  false,
		}
	}

	return testResult{
		name:   "Configuration file (apm.yaml)",
		status: "PASS",
		passed: true,
	}
}

func testRequiredFields(config *viper.Viper) testResult {
	required := []string{
		"version",
		"project.name",
		"project.environment",
		"application.entry_point",
	}

	missing := []string{}
	for _, field := range required {
		if !config.IsSet(field) || config.GetString(field) == "" {
			missing = append(missing, field)
		}
	}

	if len(missing) > 0 {
		return testResult{
			name:    "Required configuration fields",
			status:  "FAIL",
			message: fmt.Sprintf("Missing fields: %v", missing),
			passed:  false,
		}
	}

	return testResult{
		name:   "Required configuration fields",
		status: "PASS",
		passed: true,
	}
}

func testPrometheus(config *viper.Viper) testResult {
	port := config.GetInt("apm.prometheus.port")
	if port == 0 {
		port = 9090
	}

	url := fmt.Sprintf("http://localhost:%d/-/ready", port)
	client := &http.Client{Timeout: 2 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return testResult{
			name:    "Prometheus connectivity",
			status:  "FAIL",
			message: fmt.Sprintf("Cannot connect to Prometheus on port %d", port),
			passed:  false,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return testResult{
			name:    "Prometheus connectivity",
			status:  "FAIL",
			message: fmt.Sprintf("Prometheus returned status %d", resp.StatusCode),
			passed:  false,
		}
	}

	return testResult{
		name:    "Prometheus connectivity",
		status:  "PASS",
		message: fmt.Sprintf("Connected successfully on port %d", port),
		passed:  true,
	}
}

func testGrafana(config *viper.Viper) testResult {
	port := config.GetInt("apm.grafana.port")
	if port == 0 {
		port = 3000
	}

	url := fmt.Sprintf("http://localhost:%d/api/health", port)
	client := &http.Client{Timeout: 2 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return testResult{
			name:    "Grafana connectivity",
			status:  "FAIL",
			message: fmt.Sprintf("Cannot connect to Grafana on port %d", port),
			passed:  false,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return testResult{
			name:    "Grafana connectivity",
			status:  "FAIL",
			message: fmt.Sprintf("Grafana returned status %d", resp.StatusCode),
			passed:  false,
		}
	}

	return testResult{
		name:    "Grafana connectivity",
		status:  "PASS",
		message: fmt.Sprintf("Connected successfully on port %d", port),
		passed:  true,
	}
}

func testJaeger(config *viper.Viper) testResult {
	port := config.GetInt("apm.jaeger.ui_port")
	if port == 0 {
		port = 16686
	}

	url := fmt.Sprintf("http://localhost:%d/", port)
	client := &http.Client{Timeout: 2 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return testResult{
			name:    "Jaeger connectivity",
			status:  "FAIL",
			message: fmt.Sprintf("Cannot connect to Jaeger UI on port %d", port),
			passed:  false,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return testResult{
			name:    "Jaeger connectivity",
			status:  "FAIL",
			message: fmt.Sprintf("Jaeger returned status %d", resp.StatusCode),
			passed:  false,
		}
	}

	return testResult{
		name:    "Jaeger connectivity",
		status:  "PASS",
		message: fmt.Sprintf("Connected successfully on port %d", port),
		passed:  true,
	}
}

func testLoki(config *viper.Viper) testResult {
	port := config.GetInt("apm.loki.port")
	if port == 0 {
		port = 3100
	}

	url := fmt.Sprintf("http://localhost:%d/ready", port)
	client := &http.Client{Timeout: 2 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return testResult{
			name:    "Loki connectivity",
			status:  "FAIL",
			message: fmt.Sprintf("Cannot connect to Loki on port %d", port),
			passed:  false,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return testResult{
			name:    "Loki connectivity",
			status:  "FAIL",
			message: fmt.Sprintf("Loki returned status %d", resp.StatusCode),
			passed:  false,
		}
	}

	return testResult{
		name:    "Loki connectivity",
		status:  "PASS",
		message: fmt.Sprintf("Connected successfully on port %d", port),
		passed:  true,
	}
}

func testSlackWebhook(config *viper.Viper) testResult {
	webhook := config.GetString("notifications.slack.webhook_url")
	if webhook == "" {
		return testResult{
			name:    "Slack webhook configuration",
			status:  "FAIL",
			message: "No webhook URL configured",
			passed:  false,
		}
	}

	// Validate webhook URL format
	if !strings.HasPrefix(webhook, "https://hooks.slack.com/services/") {
		return testResult{
			name:    "Slack webhook configuration",
			status:  "FAIL",
			message: "Invalid webhook URL format",
			passed:  false,
		}
	}

	// Note: We don't actually test the webhook by sending a message
	// as that would send unnecessary test notifications
	return testResult{
		name:    "Slack webhook configuration",
		status:  "PASS",
		message: fmt.Sprintf("Webhook configured for %s", config.GetString("notifications.slack.channel")),
		passed:  true,
	}
}

func testApplicationEntry(config *viper.Viper) testResult {
	entryPoint := config.GetString("application.entry_point")
	if entryPoint == "" {
		return testResult{
			name:    "Application entry point",
			status:  "FAIL",
			message: "No entry point specified",
			passed:  false,
		}
	}

	if _, err := os.Stat(entryPoint); os.IsNotExist(err) {
		return testResult{
			name:    "Application entry point",
			status:  "FAIL",
			message: fmt.Sprintf("File not found: %s", entryPoint),
			passed:  false,
		}
	}

	return testResult{
		name:    "Application entry point",
		status:  "PASS",
		message: fmt.Sprintf("Found: %s", entryPoint),
		passed:  true,
	}
}

func init() {
	TestCmd.Flags().StringP("config", "c", "apm.yaml", "Path to configuration file")
}
