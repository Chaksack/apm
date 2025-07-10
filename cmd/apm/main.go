package main

import (
	"fmt"
	"os"

	"github.com/chaksack/apm/cmd/apm/commands"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "apm",
	Short: "APM CLI - Application Performance Monitoring tool for GoFiber applications",
	Long: `APM CLI is a comprehensive tool to streamline the setup, execution, and monitoring 
of GoFiber applications with integrated APM tools including Prometheus, Grafana, Jaeger, and Loki.

Commands:
  init       Initialize APM configuration with interactive setup
  run        Run application with APM instrumentation and hot reload
  test       Validate configuration and perform health checks
  dashboard  Access monitoring interfaces
  deploy     Deploy APM-instrumented application to cloud

Examples:
  apm init                    # Interactive setup wizard
  apm run                     # Run with configuration from apm.yaml
  apm run "go run main.go"    # Run specific command
  apm test                    # Validate configuration
  apm dashboard               # Access monitoring tools
  apm deploy                  # Deploy to cloud with APM`,
	Version: "1.0.0",
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Add commands
	rootCmd.AddCommand(commands.InitCmd)
	rootCmd.AddCommand(commands.RunCmd)
	rootCmd.AddCommand(commands.TestCmd)
	rootCmd.AddCommand(commands.DashboardCmd)
	rootCmd.AddCommand(commands.DeployCmd)
	rootCmd.AddCommand(commands.LogsCmd)
	rootCmd.AddCommand(commands.StatusCmd)

	// Configure root command
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.SetVersionTemplate(`{{.Name}} {{.Version}}
`)

	// Add global flags
	rootCmd.PersistentFlags().String("config", "apm.yaml", "Path to configuration file")
	rootCmd.PersistentFlags().Bool("json", false, "Output in JSON format")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().Bool("no-color", false, "Disable colored output")
}
