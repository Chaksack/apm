package main

import (
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	configFile string
	verbose    bool
	debug      bool
	jsonOutput bool
	noColor    bool
)

// Version information
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// C-style printers
var (
	Success = color.New(color.FgGreen).FprintfFunc()
	Error   = color.New(color.FgRed).FprintfFunc()
	Info    = color.New(color.FgBlue).FprintfFunc()
	Warn    = color.New(color.FgYellow).FprintfFunc()
)

const (
	geminiAsciiArt = `   __  ___  __  ___  __   __
  /   / __ /  / /__ /  \ |__)
 /__ /___ /__/ /___ \__/ |
`
	shortDescription = "APM CLI - Application Performance Monitoring tool"
	longDescription  = `APM CLI provides a unified interface for managing and interacting 
with the Application Performance Monitoring stack. It simplifies setup, 
development, testing, and monitoring access.`
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "apm",
		Short: shortDescription,
		Long:  longDescription,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if noColor {
				color.NoColor = true
			}
		},
	}

	// Customizing the help and usage templates
	rootCmd.SetHelpTemplate(getHelpTemplate())
	rootCmd.SetUsageTemplate(getUsageTemplate())

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "./apm.yaml", "Path to config file")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")

	// Add commands
	rootCmd.AddCommand(
		newInitCommand(),
		newRunCommand(),
		newTestCommand(),
		newDashboardCommand(),
	)

	if err := rootCmd.Execute(); err != nil {
		Error(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}


// newInitCommand creates the init command
func newInitCommand() *cobra.Command {
	var (
		projectName    string
		projectType    string
		environment    string
		force          bool
		template       string
		skipValidation bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new APM configuration",
		Long:  `Initialize a new APM configuration for your project with an interactive setup wizard.`, 
		RunE: func(cmd *cobra.Command, args []string) error {
			s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
			s.Suffix = " Initializing APM configuration..."
			s.Color("blue")
			s.Start()
			time.Sleep(3 * time.Second) // Simulate work
			s.Stop()
			Success(os.Stdout, "✅ APM configuration initialized successfully!\n")
			return nil
		},
	}

	cmd.Flags().StringVarP(&projectName, "name", "n", "", "Project name")
	cmd.Flags().StringVarP(&projectType, "type", "t", "", "Project type (gofiber|generic)")
	cmd.Flags().StringVarP(&environment, "env", "e", "local", "Environment (local|docker|kubernetes)")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing configuration")
	cmd.Flags().StringVar(&template, "template", "", "Use configuration template (minimal|standard|full)")
	cmd.Flags().BoolVar(&skipValidation, "skip-validation", false, "Skip configuration validation")

	return cmd
}

// newRunCommand creates the run command
func newRunCommand() *cobra.Command {
	var (
		stackOnly  bool
		appOnly    bool
		hotReload  bool
		port       int
		envFile    string
		detach     bool
		follow     bool
		tailLines  int
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run application with hot reload",
		Long:  `Start the application and monitoring stack with development features.`, 
		RunE: func(cmd *cobra.Command, args []string) error {
			Info(os.Stdout, "🚀 Starting APM stack...\n")
			return nil
		},
	}

	cmd.Flags().BoolVar(&stackOnly, "stack-only", false, "Run only the monitoring stack")
	cmd.Flags().BoolVar(&appOnly, "app-only", false, "Run only the application")
	cmd.Flags().BoolVarP(&hotReload, "hot-reload", "r", true, "Enable hot reload")
	cmd.Flags().IntVarP(&port, "port", "p", 0, "Application port")
	cmd.Flags().StringVar(&envFile, "env-file", "", "Environment file to load")
	cmd.Flags().BoolVarP(&detach, "detach", "d", false, "Run in background")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")
	cmd.Flags().IntVar(&tailLines, "tail", 100, "Number of lines to tail")

	return cmd
}

// newTestCommand creates the test command
func newTestCommand() *cobra.Command {
	var (
		connectivity bool
		configOnly   bool
		timeout      string
		retry        int
		fix          bool
	)

	cmd := &cobra.Command{
		Use:   "test [component...]",
		Short: "Validate configuration and health",
		Long:  `Test and validate the APM setup and component health.`, 
		RunE: func(cmd *cobra.Command, args []string) error {
			Info(os.Stdout, "🔍 Validating APM configuration...\n")
			return nil
		},
	}

	cmd.Flags().BoolVar(&connectivity, "connectivity", false, "Test network connectivity only")
	cmd.Flags().BoolVar(&configOnly, "config-only", false, "Validate configuration without testing")
	cmd.Flags().StringVar(&timeout, "timeout", "30s", "Test timeout")
	cmd.Flags().IntVar(&retry, "retry", 3, "Number of retries")
	cmd.Flags().BoolVar(&fix, "fix", false, "Attempt to fix common issues")

	return cmd
}

// newDashboardCommand creates the dashboard command
func newDashboardCommand() *cobra.Command {
	var (
		browser      string
		noBrowser    bool
		portForward  bool
		namespace    string
		list         bool
	)

	cmd := &cobra.Command{
		Use:   "dashboard [component]",
		Short: "Access monitoring interfaces",
		Long:  `Quick access to all monitoring dashboards and tools.`, 
		RunE: func(cmd *cobra.Command, args []string) error {
			Info(os.Stdout, "🎯 APM Dashboards\n")
			return nil
		},
	}

	cmd.Flags().StringVarP(&browser, "browser", "b", "", "Browser to use")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Show URLs without opening browser")
	cmd.Flags().BoolVar(&portForward, "port-forward", false, "Enable Kubernetes port forwarding")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Kubernetes namespace")
	cmd.Flags().BoolVarP(&list, "list", "l", false, "List all available dashboards")

	return cmd
}

func getHelpTemplate() string {
	return fmt.Sprintf(`%s
%s

%s
  {{.UseLine}}{{if .HasAvailableSubCommands}}
%s
  {{.CommandPath}} [command]}{{end}}{{if .HasExample}}

%s
  {{.Example}}{{end}}{{if .HasAvailableSubCommands}}

%s
{{range .Commands}}{{.Name | printf "%%-11s"}}{{.Short}}
{{end}}{{end}}{{if .HasAvailableLocalFlags}}

%s
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

%s
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

%s
  {{.CommandPath}} [command] --help{{end}}
`,
		color.HiBlueString(geminiAsciiArt),
		color.HiGreenString(shortDescription),
		color.HiWhiteString("Usage:"),
		color.HiYellowString("Examples:"),
		color.HiWhiteString("Available Commands:"),
		color.HiWhiteString("Flags:"),
		color.HiWhiteString("Global Flags:"),
		color.HiWhiteString("Additional help topics:"))
}

func getUsageTemplate() string {
	return fmt.Sprintf(`%s
%s
  {{.CommandPath}} [command]

%s
{{range .Commands}}{{.Name | printf "%%-11s"}}{{.Short}}
{{end}}
`,
		color.HiBlueString(geminiAsciiArt),
		color.HiWhiteString("Usage:"),
		color.HiWhiteString("Available Commands:"))
}