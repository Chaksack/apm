package instrumentation

import (
	"os"
	"strconv"
	"strings"
)

// Config holds the configuration for instrumentation
type Config struct {
	ServiceName string
	Environment string
	Version     string

	Metrics MetricsConfig
	Logging LoggingConfig
}

// MetricsConfig holds metrics-specific configuration
type MetricsConfig struct {
	Enabled   bool
	Namespace string
	Subsystem string
	Path      string // Prometheus metrics endpoint path
}

// LoggingConfig holds logging-specific configuration
type LoggingConfig struct {
	Level            string                 // debug, info, warn, error
	Encoding         string                 // json or console
	Development      bool                   // Development mode
	OutputPaths      []string               // Output paths (stdout, stderr, or file paths)
	ErrorOutputPaths []string               // Error output paths
	EnableCaller     bool                   // Enable caller information
	EnableStacktrace bool                   // Enable stack trace for errors
	InitialFields    map[string]interface{} // Initial fields to add to all logs
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		ServiceName: getEnv("SERVICE_NAME", "app"),
		Environment: getEnv("ENVIRONMENT", "development"),
		Version:     getEnv("VERSION", "unknown"),

		Metrics: MetricsConfig{
			Enabled:   getEnvBool("METRICS_ENABLED", true),
			Namespace: getEnv("METRICS_NAMESPACE", ""),
			Subsystem: getEnv("METRICS_SUBSYSTEM", ""),
			Path:      getEnv("METRICS_PATH", "/metrics"),
		},

		Logging: LoggingConfig{
			Level:            getEnv("LOG_LEVEL", "info"),
			Encoding:         getEnv("LOG_ENCODING", "json"),
			Development:      getEnvBool("LOG_DEVELOPMENT", false),
			OutputPaths:      getEnvSlice("LOG_OUTPUT_PATHS", []string{"stdout"}),
			ErrorOutputPaths: getEnvSlice("LOG_ERROR_OUTPUT_PATHS", []string{"stderr"}),
			EnableCaller:     getEnvBool("LOG_ENABLE_CALLER", false),
			EnableStacktrace: getEnvBool("LOG_ENABLE_STACKTRACE", false),
			InitialFields: map[string]interface{}{
				"service": getEnv("SERVICE_NAME", "app"),
				"env":     getEnv("ENVIRONMENT", "development"),
				"version": getEnv("VERSION", "unknown"),
			},
		},
	}
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() *Config {
	cfg := DefaultConfig()

	// Override with environment variables if set
	if svcName := os.Getenv("SERVICE_NAME"); svcName != "" {
		cfg.ServiceName = svcName
		cfg.Logging.InitialFields["service"] = svcName
	}

	if env := os.Getenv("ENVIRONMENT"); env != "" {
		cfg.Environment = env
		cfg.Logging.InitialFields["env"] = env
	}

	if version := os.Getenv("VERSION"); version != "" {
		cfg.Version = version
		cfg.Logging.InitialFields["version"] = version
	}

	// Load metrics config
	if enabled := os.Getenv("METRICS_ENABLED"); enabled != "" {
		cfg.Metrics.Enabled = parseBool(enabled)
	}

	if namespace := os.Getenv("METRICS_NAMESPACE"); namespace != "" {
		cfg.Metrics.Namespace = namespace
	}

	if subsystem := os.Getenv("METRICS_SUBSYSTEM"); subsystem != "" {
		cfg.Metrics.Subsystem = subsystem
	}

	if path := os.Getenv("METRICS_PATH"); path != "" {
		cfg.Metrics.Path = path
	}

	// Load logging config
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		cfg.Logging.Level = level
	}

	if encoding := os.Getenv("LOG_ENCODING"); encoding != "" {
		cfg.Logging.Encoding = encoding
	}

	if dev := os.Getenv("LOG_DEVELOPMENT"); dev != "" {
		cfg.Logging.Development = parseBool(dev)
	}

	if outputs := os.Getenv("LOG_OUTPUT_PATHS"); outputs != "" {
		cfg.Logging.OutputPaths = strings.Split(outputs, ",")
	}

	if errorOutputs := os.Getenv("LOG_ERROR_OUTPUT_PATHS"); errorOutputs != "" {
		cfg.Logging.ErrorOutputPaths = strings.Split(errorOutputs, ",")
	}

	if caller := os.Getenv("LOG_ENABLE_CALLER"); caller != "" {
		cfg.Logging.EnableCaller = parseBool(caller)
	}

	if stacktrace := os.Getenv("LOG_ENABLE_STACKTRACE"); stacktrace != "" {
		cfg.Logging.EnableStacktrace = parseBool(stacktrace)
	}

	return cfg
}

// getEnv returns the value of an environment variable or a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool returns the boolean value of an environment variable or a default value
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return parseBool(value)
	}
	return defaultValue
}

// getEnvSlice returns a slice from a comma-separated environment variable
func getEnvSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

// parseBool parses a string to boolean
func parseBool(s string) bool {
	b, err := strconv.ParseBool(s)
	if err != nil {
		// Handle common boolean representations
		s = strings.ToLower(strings.TrimSpace(s))
		return s == "true" || s == "yes" || s == "on" || s == "1"
	}
	return b
}
