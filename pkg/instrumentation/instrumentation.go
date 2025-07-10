// Copyright (c) 2024 APM Solution Contributors
// Authors: Andrew Chakdahah (chakdahah@gmail.com) and Yaw Boateng Kessie (ybkess@gmail.com)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package instrumentation

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// Instrumentation provides a unified interface for metrics, logging, and tracing
type Instrumentation struct {
	Logger  *zap.Logger
	Metrics *MetricsCollector
	config  *Config

	shutdownFuncs []func() error
	mu            sync.Mutex
}

// New creates a new instrumentation instance
func New(cfg *Config) (*Instrumentation, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Initialize logger
	logger, err := initLogger(cfg.Logging)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Initialize metrics
	metrics, err := initMetrics(cfg.Metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize metrics: %w", err)
	}

	inst := &Instrumentation{
		Logger:        logger,
		Metrics:       metrics,
		config:        cfg,
		shutdownFuncs: make([]func() error, 0),
	}

	// Register Prometheus metrics
	if err := inst.registerMetrics(); err != nil {
		return nil, fmt.Errorf("failed to register metrics: %w", err)
	}

	return inst, nil
}

// RegisterShutdownFunc registers a function to be called during shutdown
func (i *Instrumentation) RegisterShutdownFunc(fn func() error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.shutdownFuncs = append(i.shutdownFuncs, fn)
}

// Shutdown gracefully shuts down all instrumentation components
func (i *Instrumentation) Shutdown(ctx context.Context) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	var errs []error

	// Execute all shutdown functions
	for _, fn := range i.shutdownFuncs {
		if err := fn(); err != nil {
			errs = append(errs, err)
		}
	}

	// Sync logger
	if err := i.Logger.Sync(); err != nil {
		errs = append(errs, fmt.Errorf("failed to sync logger: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}

	return nil
}

// WaitForShutdown blocks until a shutdown signal is received
func (i *Instrumentation) WaitForShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan

	i.Logger.Info("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := i.Shutdown(ctx); err != nil {
		i.Logger.Error("failed to shutdown gracefully", zap.Error(err))
	}
}

// FiberMiddleware returns a Fiber middleware that instruments HTTP requests
func (i *Instrumentation) FiberMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Get request details
		method := c.Method()
		path := c.Route().Path
		if path == "" {
			path = c.Path()
		}

		// Process request
		err := c.Next()

		// Record metrics
		duration := time.Since(start)
		status := c.Response().StatusCode()

		i.Metrics.RecordHTTPRequest(method, path, status, duration)
		i.Metrics.RecordHTTPRequestSize(method, path, float64(len(c.Body())))
		i.Metrics.RecordHTTPResponseSize(method, path, float64(len(c.Response().Body())))

		// Log request
		fields := []zap.Field{
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", status),
			zap.Duration("duration", duration),
			zap.String("ip", c.IP()),
			zap.String("user_agent", c.Get("User-Agent")),
		}

		if err != nil {
			fields = append(fields, zap.Error(err))
			i.Logger.Error("request failed", fields...)
		} else if status >= 400 {
			i.Logger.Warn("request completed with error status", fields...)
		} else {
			i.Logger.Info("request completed", fields...)
		}

		return err
	}
}

// registerMetrics registers all Prometheus metrics
func (i *Instrumentation) registerMetrics() error {
	// Register HTTP metrics
	prometheus.MustRegister(
		i.Metrics.httpRequestsTotal,
		i.Metrics.httpRequestDuration,
		i.Metrics.httpRequestSize,
		i.Metrics.httpResponseSize,
	)

	// Register custom metrics if any
	for _, collector := range i.Metrics.customCollectors {
		prometheus.MustRegister(collector)
	}

	return nil
}

// initLogger initializes the zap logger
func initLogger(cfg LoggingConfig) (*zap.Logger, error) {
	var zapCfg zap.Config

	if cfg.Development {
		zapCfg = zap.NewDevelopmentConfig()
	} else {
		zapCfg = zap.NewProductionConfig()
	}

	// Set log level
	if err := zapCfg.Level.UnmarshalText([]byte(cfg.Level)); err != nil {
		return nil, fmt.Errorf("invalid log level %s: %w", cfg.Level, err)
	}

	// Set output paths
	zapCfg.OutputPaths = cfg.OutputPaths
	zapCfg.ErrorOutputPaths = cfg.ErrorOutputPaths

	// Set encoding
	zapCfg.Encoding = cfg.Encoding

	return zapCfg.Build()
}

// initMetrics initializes the metrics collector
func initMetrics(cfg MetricsConfig) (*MetricsCollector, error) {
	return NewMetricsCollector(cfg.Namespace, cfg.Subsystem), nil
}
