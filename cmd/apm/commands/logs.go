package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yourusername/apm/pkg/security"
)

var LogsCmd = &cobra.Command{
	Use:   "logs [component]",
	Short: "View application and APM component logs",
	Long: `View logs from your application or APM components (prometheus, grafana, jaeger, loki).
If no component is specified, application logs are shown.`,
	Args:      cobra.MaximumNArgs(1),
	ValidArgs: []string{"app", "application", "prometheus", "grafana", "jaeger", "loki", "alertmanager"},
	RunE:      runLogs,
}

var (
	follow      bool
	tail        int
	since       string
	filter      string
	jsonOutput  bool
	logsVerbose bool
)

type logEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level,omitempty"`
	Message   string                 `json:"message"`
	Component string                 `json:"component"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

func init() {
	LogsCmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")
	LogsCmd.Flags().IntVarP(&tail, "tail", "n", 100, "Number of lines to show")
	LogsCmd.Flags().StringVar(&since, "since", "", "Show logs since duration (e.g., 1h, 30m)")
	LogsCmd.Flags().StringVar(&filter, "filter", "", "Filter log entries by pattern")
	LogsCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output logs in JSON format")
	LogsCmd.Flags().BoolVarP(&logsVerbose, "verbose", "v", false, "Show verbose log information")
}

func runLogs(cmd *cobra.Command, args []string) error {
	// Load configuration
	config := viper.New()
	config.SetConfigName("apm")
	config.SetConfigType("yaml")
	config.AddConfigPath(".")

	if err := config.ReadInConfig(); err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	// Validate tail parameter
	if err := security.ValidateTailLines(tail); err != nil {
		return fmt.Errorf("invalid tail parameter: %w", err)
	}

	// Validate and sanitize filter
	if filter != "" {
		filter = security.SanitizeLogFilter(filter)
	}

	// Validate since duration
	if since != "" {
		if err := security.ValidateDuration(since); err != nil {
			return fmt.Errorf("invalid since parameter: %w", err)
		}
	}

	// Determine which component to show logs for
	component := "app"
	if len(args) > 0 {
		component = normalizeComponent(args[0])
		// Validate component name
		if err := security.ValidateServiceName(component); err != nil {
			return fmt.Errorf("invalid component name: %w", err)
		}
	}

	// Get log source based on component
	logSource, err := getLogSource(component, config)
	if err != nil {
		return err
	}

	// Handle JSON output flag globally
	if jsonOutput {
		return streamLogsJSON(logSource, component)
	}

	// Display logs with formatting
	return streamLogs(logSource, component)
}

func normalizeComponent(name string) string {
	switch strings.ToLower(name) {
	case "app", "application":
		return "app"
	default:
		return strings.ToLower(name)
	}
}

func getLogSource(component string, config *viper.Viper) (io.ReadCloser, error) {
	ctx := context.Background()

	switch component {
	case "app":
		return getApplicationLogs(ctx, config)
	case "prometheus", "grafana", "jaeger", "loki", "alertmanager":
		return getComponentLogs(ctx, component, config)
	default:
		return nil, fmt.Errorf("unknown component: %s", component)
	}
}

func getApplicationLogs(ctx context.Context, config *viper.Viper) (io.ReadCloser, error) {
	// Check if running in Docker
	if isDockerized() {
		return getDockerLogs(ctx, config.GetString("project.name"))
	}

	// Check if running in Kubernetes
	if isKubernetes() {
		namespace := config.GetString("deployment.kubernetes.namespace")
		if namespace == "" {
			namespace = "default"
		}
		return getKubernetesLogs(ctx, config.GetString("project.name"), namespace)
	}

	// Default to local log file
	logPath := config.GetString("application.log_path")
	if logPath == "" {
		logPath = "./app.log"
	}

	// Validate log path to prevent path traversal
	workDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	// Allow logs only from current directory or configured directories
	allowedDirs := []string{workDir, "/var/log", "/tmp"}
	if err := security.ValidateFilePath(logPath, allowedDirs); err != nil {
		return nil, fmt.Errorf("invalid log path: %w", err)
	}

	return tailFile(logPath)
}

func getComponentLogs(ctx context.Context, component string, config *viper.Viper) (io.ReadCloser, error) {
	// Check if component is enabled
	if !config.GetBool(fmt.Sprintf("apm.%s.enabled", component)) {
		return nil, fmt.Errorf("%s is not enabled in configuration", component)
	}

	// Try Docker first
	containerName := fmt.Sprintf("apm-%s", component)
	if isDockerized() {
		return getDockerLogs(ctx, containerName)
	}

	// Try Kubernetes
	if isKubernetes() {
		namespace := "apm-system"
		return getKubernetesLogs(ctx, component, namespace)
	}

	// Default to systemd logs if available
	return getSystemdLogs(ctx, component)
}

func getDockerLogs(ctx context.Context, containerName string) (io.ReadCloser, error) {
	// Validate container name to prevent command injection
	if err := security.ValidateContainerName(containerName); err != nil {
		return nil, fmt.Errorf("invalid container name: %w", err)
	}

	args := []string{"logs"}

	if follow {
		args = append(args, "-f")
	}

	if tail > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", tail))
	}

	if since != "" {
		// Since was already validated in runLogs
		args = append(args, "--since", since)
	}

	args = append(args, containerName)

	cmd := exec.CommandContext(ctx, "docker", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get docker logs: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start docker logs: %w", err)
	}

	return stdout, nil
}

func getKubernetesLogs(ctx context.Context, appName, namespace string) (io.ReadCloser, error) {
	// Validate app name and namespace to prevent command injection
	if err := security.ValidateServiceName(appName); err != nil {
		return nil, fmt.Errorf("invalid app name: %w", err)
	}

	if err := security.ValidateNamespace(namespace); err != nil {
		return nil, fmt.Errorf("invalid namespace: %w", err)
	}

	args := []string{"logs", "-n", namespace}

	if follow {
		args = append(args, "-f")
	}

	if tail > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", tail))
	}

	if since != "" {
		// Since was already validated in runLogs
		args = append(args, "--since", since)
	}

	// Find pod by label
	args = append(args, "-l", fmt.Sprintf("app=%s", appName))

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes logs: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start kubectl logs: %w", err)
	}

	return stdout, nil
}

func getSystemdLogs(ctx context.Context, service string) (io.ReadCloser, error) {
	// Validate service name to prevent command injection
	if err := security.ValidateServiceName(service); err != nil {
		return nil, fmt.Errorf("invalid service name: %w", err)
	}

	args := []string{"-u", fmt.Sprintf("%s.service", service)}

	if follow {
		args = append(args, "-f")
	}

	if tail > 0 {
		args = append(args, "-n", fmt.Sprintf("%d", tail))
	}

	if since != "" {
		// Since was already validated in runLogs
		args = append(args, "--since", since)
	}

	cmd := exec.CommandContext(ctx, "journalctl", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get systemd logs: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start journalctl: %w", err)
	}

	return stdout, nil
}

func tailFile(path string) (io.ReadCloser, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// If tail is specified and not following, we need to seek to show last N lines
	if tail > 0 && !follow {
		// Read all lines first to get the last N
		scanner := bufio.NewScanner(file)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		file.Close()

		// Get last N lines
		start := 0
		if len(lines) > tail {
			start = len(lines) - tail
		}
		lastLines := lines[start:]

		// Create a reader from the last lines
		content := strings.Join(lastLines, "\n")
		if content != "" {
			content += "\n"
		}
		return io.NopCloser(strings.NewReader(content)), nil
	}

	// If not tailing or following, just return the file
	return file, nil
}

func streamLogs(source io.ReadCloser, component string) error {
	defer source.Close()

	// Style definitions
	timestampStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	componentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true)

	levelStyles := map[string]lipgloss.Style{
		"debug": lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		"info":  lipgloss.NewStyle().Foreground(lipgloss.Color("86")),
		"warn":  lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
		"error": lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
		"fatal": lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true).Underline(true),
	}

	scanner := bufio.NewScanner(source)
	// Set max buffer size to prevent memory exhaustion (1MB per line)
	const maxScanTokenSize = 1024 * 1024
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, maxScanTokenSize)

	lineCount := 0

	for scanner.Scan() {
		line := scanner.Text()

		// Apply filter if specified
		if filter != "" && !strings.Contains(strings.ToLower(line), strings.ToLower(filter)) {
			continue
		}

		// Parse and format the log line
		entry := parseLogLine(line, component)

		// Format output
		output := formatLogEntry(entry, timestampStyle, componentStyle, levelStyles)
		fmt.Println(output)

		lineCount++
		if !follow && tail > 0 && lineCount >= tail {
			break
		}
	}

	return scanner.Err()
}

func streamLogsJSON(source io.ReadCloser, component string) error {
	defer source.Close()

	scanner := bufio.NewScanner(source)
	// Set max buffer size to prevent memory exhaustion (1MB per line)
	const maxScanTokenSize = 1024 * 1024
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, maxScanTokenSize)

	encoder := json.NewEncoder(os.Stdout)
	lineCount := 0

	for scanner.Scan() {
		line := scanner.Text()

		// Apply filter if specified
		if filter != "" && !strings.Contains(strings.ToLower(line), strings.ToLower(filter)) {
			continue
		}

		// Parse log line
		entry := parseLogLine(line, component)

		// Output as JSON
		if err := encoder.Encode(entry); err != nil {
			return fmt.Errorf("failed to encode log entry: %w", err)
		}

		lineCount++
		if !follow && tail > 0 && lineCount >= tail {
			break
		}
	}

	return scanner.Err()
}

func parseLogLine(line, component string) logEntry {
	entry := logEntry{
		Timestamp: time.Now(),
		Component: component,
		Message:   line,
		Fields:    make(map[string]interface{}),
	}

	// Try to parse structured logs (JSON)
	var jsonLog map[string]interface{}
	if err := json.Unmarshal([]byte(line), &jsonLog); err == nil {
		// Successfully parsed JSON
		if ts, ok := jsonLog["timestamp"].(string); ok {
			if t, err := time.Parse(time.RFC3339, ts); err == nil {
				entry.Timestamp = t
			}
		}
		if level, ok := jsonLog["level"].(string); ok {
			entry.Level = level
		}
		if msg, ok := jsonLog["message"].(string); ok {
			entry.Message = msg
		}

		// Store other fields
		for k, v := range jsonLog {
			if k != "timestamp" && k != "level" && k != "message" {
				entry.Fields[k] = v
			}
		}
	} else {
		// Try to parse common log formats
		entry.Level = detectLogLevel(line)
	}

	return entry
}

func detectLogLevel(line string) string {
	lower := strings.ToLower(line)
	switch {
	case strings.Contains(lower, "fatal"):
		return "fatal"
	case strings.Contains(lower, "error"):
		return "error"
	case strings.Contains(lower, "warn"):
		return "warn"
	case strings.Contains(lower, "debug"):
		return "debug"
	default:
		return "info"
	}
}

func formatLogEntry(entry logEntry, timestampStyle, componentStyle lipgloss.Style, levelStyles map[string]lipgloss.Style) string {
	// Format timestamp
	ts := timestampStyle.Render(entry.Timestamp.Format("15:04:05.000"))

	// Format component
	comp := componentStyle.Render(fmt.Sprintf("[%s]", entry.Component))

	// Format level
	level := ""
	if entry.Level != "" {
		style, ok := levelStyles[entry.Level]
		if !ok {
			style = levelStyles["info"]
		}
		level = style.Render(fmt.Sprintf("[%s]", strings.ToUpper(entry.Level))) + " "
	}

	// Build output
	output := fmt.Sprintf("%s %s %s%s", ts, comp, level, entry.Message)

	// Add fields if verbose
	if logsVerbose && len(entry.Fields) > 0 {
		fields := []string{}
		for k, v := range entry.Fields {
			fields = append(fields, fmt.Sprintf("%s=%v", k, v))
		}
		output += " " + lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(strings.Join(fields, " "))
	}

	return output
}

func isDockerized() bool {
	// Only return true if we're actually running INSIDE a Docker container
	// by checking for .dockerenv file
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Don't check if docker daemon is accessible - that just means Docker
	// is installed, not that we're running in Docker
	return false
}

func isKubernetes() bool {
	// Check if running in Kubernetes by looking for service account
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount"); err == nil {
		return true
	}

	// Check if kubectl is configured (without executing it)
	// Just check if kubectl binary exists in PATH
	_, err := exec.LookPath("kubectl")
	return err == nil
}
