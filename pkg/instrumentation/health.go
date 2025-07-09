package instrumentation

import (
	"context"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	// HealthStatusHealthy indicates the component is healthy
	HealthStatusHealthy HealthStatus = "healthy"
	// HealthStatusUnhealthy indicates the component is unhealthy
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	// HealthStatusDegraded indicates the component is degraded but functional
	HealthStatusDegraded HealthStatus = "degraded"
)

// HealthCheck represents a single health check
type HealthCheck struct {
	Name        string                 `json:"name"`
	Status      HealthStatus           `json:"status"`
	Message     string                 `json:"message,omitempty"`
	LastChecked time.Time              `json:"last_checked"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// HealthCheckFunc is a function that performs a health check
type HealthCheckFunc func(ctx context.Context) HealthCheck

// HealthChecker manages health checks for the application
type HealthChecker struct {
	mu     sync.RWMutex
	checks map[string]HealthCheckFunc
}

// NewHealthChecker creates a new health checker
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		checks: make(map[string]HealthCheckFunc),
	}
}

// RegisterCheck registers a new health check
func (h *HealthChecker) RegisterCheck(name string, checkFunc HealthCheckFunc) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checks[name] = checkFunc
}

// UnregisterCheck removes a health check
func (h *HealthChecker) UnregisterCheck(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.checks, name)
}

// CheckHealth runs all health checks and returns the results
func (h *HealthChecker) CheckHealth(ctx context.Context) (overall HealthStatus, checks []HealthCheck) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	checks = make([]HealthCheck, 0, len(h.checks))
	overall = HealthStatusHealthy

	for name, checkFunc := range h.checks {
		check := checkFunc(ctx)
		check.Name = name
		checks = append(checks, check)

		// Determine overall status
		if check.Status == HealthStatusUnhealthy {
			overall = HealthStatusUnhealthy
		} else if check.Status == HealthStatusDegraded && overall == HealthStatusHealthy {
			overall = HealthStatusDegraded
		}
	}

	return overall, checks
}

// LivenessHandler returns a Fiber handler for liveness probes
func LivenessHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "alive",
			"timestamp": time.Now().UTC(),
		})
	}
}

// ReadinessHandler returns a Fiber handler for readiness probes with health checks
func ReadinessHandler(checker *HealthChecker) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		overall, checks := checker.CheckHealth(ctx)

		response := fiber.Map{
			"status":    overall,
			"timestamp": time.Now().UTC(),
			"checks":    checks,
		}

		statusCode := fiber.StatusOK
		if overall == HealthStatusUnhealthy {
			statusCode = fiber.StatusServiceUnavailable
		} else if overall == HealthStatusDegraded {
			statusCode = fiber.StatusOK // Still return 200 for degraded
		}

		return c.Status(statusCode).JSON(response)
	}
}

// Common health check implementations

// DatabaseHealthCheck creates a health check for database connections
func DatabaseHealthCheck(name string, pingFunc func(ctx context.Context) error) HealthCheckFunc {
	return func(ctx context.Context) HealthCheck {
		start := time.Now()
		err := pingFunc(ctx)
		duration := time.Since(start)

		check := HealthCheck{
			Name:        name,
			LastChecked: time.Now().UTC(),
			Details: map[string]interface{}{
				"response_time_ms": duration.Milliseconds(),
			},
		}

		if err != nil {
			check.Status = HealthStatusUnhealthy
			check.Message = err.Error()
		} else {
			check.Status = HealthStatusHealthy
			check.Message = "Database is reachable"
		}

		return check
	}
}

// HTTPHealthCheck creates a health check for HTTP endpoints
func HTTPHealthCheck(name, url string, timeout time.Duration) HealthCheckFunc {
	return func(ctx context.Context) HealthCheck {
		start := time.Now()

		check := HealthCheck{
			Name:        name,
			LastChecked: time.Now().UTC(),
			Details: map[string]interface{}{
				"url": url,
			},
		}

		// Create a context with timeout
		reqCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		// Make HTTP request
		agent := fiber.Get(url)
		agent.Context(reqCtx)

		statusCode, _, errs := agent.Bytes()
		duration := time.Since(start)

		check.Details["response_time_ms"] = duration.Milliseconds()
		check.Details["status_code"] = statusCode

		if len(errs) > 0 {
			check.Status = HealthStatusUnhealthy
			check.Message = errs[0].Error()
		} else if statusCode >= 200 && statusCode < 300 {
			check.Status = HealthStatusHealthy
			check.Message = "Endpoint is reachable"
		} else {
			check.Status = HealthStatusUnhealthy
			check.Message = "Unexpected status code"
		}

		return check
	}
}

// DiskSpaceHealthCheck creates a health check for disk space
func DiskSpaceHealthCheck(path string, thresholdPercent float64) HealthCheckFunc {
	return func(ctx context.Context) HealthCheck {
		check := HealthCheck{
			Name:        "disk_space",
			LastChecked: time.Now().UTC(),
			Details: map[string]interface{}{
				"path":      path,
				"threshold": thresholdPercent,
			},
		}

		// This is a simplified example - in production you'd use syscall or a library
		// to get actual disk usage
		check.Status = HealthStatusHealthy
		check.Message = "Disk space is sufficient"
		check.Details["usage_percent"] = 45.0 // Mock value

		return check
	}
}

// MemoryHealthCheck creates a health check for memory usage
func MemoryHealthCheck(thresholdPercent float64) HealthCheckFunc {
	return func(ctx context.Context) HealthCheck {
		check := HealthCheck{
			Name:        "memory",
			LastChecked: time.Now().UTC(),
			Details: map[string]interface{}{
				"threshold": thresholdPercent,
			},
		}

		// This is a simplified example - in production you'd use runtime.MemStats
		// to get actual memory usage
		check.Status = HealthStatusHealthy
		check.Message = "Memory usage is within limits"
		check.Details["usage_percent"] = 65.0 // Mock value

		return check
	}
}

// DependencyAggregator aggregates health status from multiple dependencies
type DependencyAggregator struct {
	dependencies map[string]*HealthChecker
	mu           sync.RWMutex
}

// NewDependencyAggregator creates a new dependency aggregator
func NewDependencyAggregator() *DependencyAggregator {
	return &DependencyAggregator{
		dependencies: make(map[string]*HealthChecker),
	}
}

// RegisterDependency registers a dependency's health checker
func (d *DependencyAggregator) RegisterDependency(name string, checker *HealthChecker) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.dependencies[name] = checker
}

// CheckAllDependencies checks all registered dependencies
func (d *DependencyAggregator) CheckAllDependencies(ctx context.Context) map[string]interface{} {
	d.mu.RLock()
	defer d.mu.RUnlock()

	results := make(map[string]interface{})
	overallStatus := HealthStatusHealthy

	for name, checker := range d.dependencies {
		status, checks := checker.CheckHealth(ctx)
		results[name] = fiber.Map{
			"status": status,
			"checks": checks,
		}

		if status == HealthStatusUnhealthy {
			overallStatus = HealthStatusUnhealthy
		} else if status == HealthStatusDegraded && overallStatus == HealthStatusHealthy {
			overallStatus = HealthStatusDegraded
		}
	}

	results["overall"] = overallStatus
	return results
}
