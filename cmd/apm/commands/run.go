package commands

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var RunCmd = &cobra.Command{
	Use:   "run [command]",
	Short: "Run application with APM instrumentation and hot reload",
	Long: `Run your application with automatic APM agent injection and hot reload capabilities.
If no command is specified, it will use the command from apm.yaml configuration.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runApp,
}

type runner struct {
	config      *viper.Viper
	cmd         *exec.Cmd
	watcher     *fsnotify.Watcher
	mu          sync.Mutex
	restartChan chan bool
	ctx         context.Context
	cancel      context.CancelFunc
}

func runApp(cmd *cobra.Command, args []string) error {
	// Load configuration
	config := viper.New()
	config.SetConfigName("apm")
	config.SetConfigType("yaml")
	config.AddConfigPath(".")

	if err := config.ReadInConfig(); err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	// Create runner
	ctx, cancel := context.WithCancel(context.Background())
	r := &runner{
		config:      config,
		restartChan: make(chan bool, 1),
		ctx:         ctx,
		cancel:      cancel,
	}

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Determine command to run
	var runCommand string
	if len(args) > 0 {
		runCommand = args[0]
	} else {
		runCommand = config.GetString("application.run_command")
		if runCommand == "" {
			runCommand = "go run " + config.GetString("application.entry_point")
		}
	}

	fmt.Printf("🚀 Starting application: %s\n", runCommand)

	// Setup file watcher if hot reload is enabled
	if config.GetBool("application.hot_reload.enabled") {
		if err := r.setupWatcher(); err != nil {
			return fmt.Errorf("error setting up file watcher: %w", err)
		}
		defer r.watcher.Close()

		fmt.Println("👀 Hot reload enabled. Watching for file changes...")
	}

	// Start the application
	if err := r.startApp(runCommand); err != nil {
		return fmt.Errorf("error starting application: %w", err)
	}

	// Main event loop
	for {
		select {
		case <-sigChan:
			fmt.Println("\n🛑 Shutting down...")
			r.stopApp()
			return nil

		case <-r.restartChan:
			fmt.Println("\n🔄 Restarting application...")
			r.stopApp()
			time.Sleep(100 * time.Millisecond) // Brief pause
			if err := r.startApp(runCommand); err != nil {
				log.Printf("Error restarting application: %v", err)
			}

		case <-r.ctx.Done():
			return nil
		}
	}
}

func (r *runner) setupWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	r.watcher = watcher

	// Get paths to watch
	paths := r.config.GetStringSlice("application.hot_reload.paths")
	if len(paths) == 0 {
		paths = []string{"."}
	}

	excludePaths := r.config.GetStringSlice("application.hot_reload.exclude")
	extensions := r.config.GetStringSlice("application.hot_reload.extensions")

	// Add paths to watcher
	for _, path := range paths {
		if err := r.addPathToWatcher(path, excludePaths); err != nil {
			return err
		}
	}

	// Start watching
	go func() {
		debounce := time.NewTimer(0)
		<-debounce.C // Drain the initial timer

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Check if file should trigger reload
				if r.shouldReload(event, extensions, excludePaths) {
					debounce.Reset(time.Duration(r.config.GetInt("application.hot_reload.delay")) * time.Millisecond)
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("Watcher error: %v", err)

			case <-debounce.C:
				select {
				case r.restartChan <- true:
				default:
					// Channel full, restart already pending
				}
			}
		}
	}()

	return nil
}

func (r *runner) addPathToWatcher(path string, excludePaths []string) error {
	return filepath.Walk(path, func(walkPath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Check if path should be excluded
		for _, exclude := range excludePaths {
			if strings.Contains(walkPath, exclude) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Watch directories
		if info.IsDir() {
			return r.watcher.Add(walkPath)
		}

		return nil
	})
}

func (r *runner) shouldReload(event fsnotify.Event, extensions []string, excludePaths []string) bool {
	// Skip chmod events
	if event.Op&fsnotify.Chmod == fsnotify.Chmod {
		return false
	}

	// Check excluded paths
	for _, exclude := range excludePaths {
		if strings.Contains(event.Name, exclude) {
			return false
		}
	}

	// Check extensions
	if len(extensions) > 0 {
		hasValidExt := false
		for _, ext := range extensions {
			if strings.HasSuffix(event.Name, ext) {
				hasValidExt = true
				break
			}
		}
		if !hasValidExt {
			return false
		}
	}

	fmt.Printf("📝 File changed: %s\n", event.Name)
	return true
}

func (r *runner) startApp(command string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Setup environment variables for APM
	env := os.Environ()
	env = r.setupAPMEnvironment(env)

	// Parse command
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	// Create command
	cmd := exec.CommandContext(r.ctx, parts[0], parts[1:]...)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Set process group ID so we can kill all child processes
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Start the command
	if err := cmd.Start(); err != nil {
		return err
	}

	r.cmd = cmd

	// Monitor process
	go func() {
		if err := cmd.Wait(); err != nil {
			if !strings.Contains(err.Error(), "signal: killed") {
				log.Printf("Application exited with error: %v", err)
			}
		}
	}()

	return nil
}

func (r *runner) stopApp() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cmd == nil || r.cmd.Process == nil {
		return
	}

	// Kill the process group
	pgid, err := syscall.Getpgid(r.cmd.Process.Pid)
	if err == nil {
		syscall.Kill(-pgid, syscall.SIGTERM)
	}

	// Give it time to shutdown gracefully
	done := make(chan bool)
	go func() {
		r.cmd.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Process exited gracefully
	case <-time.After(5 * time.Second):
		// Force kill if not exited
		if err == nil {
			syscall.Kill(-pgid, syscall.SIGKILL)
		}
	}

	r.cmd = nil
}

func (r *runner) setupAPMEnvironment(env []string) []string {
	// Add OpenTelemetry environment variables
	if r.config.GetBool("apm.opentelemetry.enabled") {
		env = append(env,
			fmt.Sprintf("OTEL_SERVICE_NAME=%s", r.config.GetString("project.name")),
			fmt.Sprintf("OTEL_EXPORTER_OTLP_ENDPOINT=%s", r.config.GetString("apm.opentelemetry.endpoint")),
			"OTEL_TRACES_EXPORTER=otlp",
			"OTEL_METRICS_EXPORTER=otlp",
			"OTEL_LOGS_EXPORTER=otlp",
		)
	}

	// Add Jaeger environment variables
	if r.config.GetBool("apm.jaeger.enabled") {
		env = append(env,
			fmt.Sprintf("JAEGER_SERVICE_NAME=%s", r.config.GetString("project.name")),
			fmt.Sprintf("JAEGER_AGENT_HOST=%s", r.config.GetString("apm.jaeger.agent_host")),
			fmt.Sprintf("JAEGER_AGENT_PORT=%d", r.config.GetInt("apm.jaeger.agent_port")),
		)
	}

	// Add service configuration
	env = append(env,
		fmt.Sprintf("SERVICE_NAME=%s", r.config.GetString("project.name")),
		fmt.Sprintf("ENVIRONMENT=%s", r.config.GetString("project.environment")),
		fmt.Sprintf("LOG_LEVEL=%s", r.config.GetString("application.log_level")),
	)

	return env
}

func init() {
	RunCmd.Flags().BoolP("no-reload", "n", false, "Disable hot reload")
	RunCmd.Flags().StringP("config", "c", "apm.yaml", "Path to configuration file")
}
