package cloud

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DegradationLevel represents the level of service degradation
type DegradationLevel int

const (
	DegradationNone DegradationLevel = iota
	DegradationMinor
	DegradationMajor
	DegradationSevere
	DegradationComplete
)

// String returns the string representation of degradation level
func (dl DegradationLevel) String() string {
	switch dl {
	case DegradationNone:
		return "none"
	case DegradationMinor:
		return "minor"
	case DegradationMajor:
		return "major"
	case DegradationSevere:
		return "severe"
	case DegradationComplete:
		return "complete"
	default:
		return "unknown"
	}
}

// ServiceHealth represents the health status of a service
type ServiceHealth struct {
	Provider         Provider         `json:"provider"`
	Service          string           `json:"service"`
	Healthy          bool             `json:"healthy"`
	DegradationLevel DegradationLevel `json:"degradation_level"`
	LastCheck        time.Time        `json:"last_check"`
	ErrorCount       int              `json:"error_count"`
	ErrorRate        float64          `json:"error_rate"`
	AvgResponseTime  time.Duration    `json:"avg_response_time"`
	Description      string           `json:"description,omitempty"`
}

// DegradationStrategy defines how to handle service degradation
type DegradationStrategy struct {
	FallbackEnabled      bool                          `json:"fallback_enabled"`
	CacheEnabled         bool                          `json:"cache_enabled"`
	CacheTTL             time.Duration                 `json:"cache_ttl"`
	ReducedFunctionality bool                          `json:"reduced_functionality"`
	PartialResults       bool                          `json:"partial_results"`
	FallbackProviders    []Provider                    `json:"fallback_providers,omitempty"`
	DisabledFeatures     []string                      `json:"disabled_features,omitempty"`
	CustomHandlers       map[string]DegradationHandler `json:"-"`
}

// DegradationHandler defines how to handle specific types of degradation
type DegradationHandler func(ctx context.Context, error error) (interface{}, error)

// GracefulDegradationManager manages service degradation across cloud providers
type GracefulDegradationManager struct {
	healthStatus   map[string]*ServiceHealth
	strategies     map[string]*DegradationStrategy
	cache          map[string]*CachedResult
	fallbackChains map[Provider][]Provider
	mu             sync.RWMutex
	healthChecker  *HealthChecker
	retryManager   *AdvancedRetryManager
}

// CachedResult represents a cached operation result
type CachedResult struct {
	Data      interface{}   `json:"data"`
	Timestamp time.Time     `json:"timestamp"`
	TTL       time.Duration `json:"ttl"`
	Provider  Provider      `json:"provider"`
	Operation string        `json:"operation"`
}

// NewGracefulDegradationManager creates a new graceful degradation manager
func NewGracefulDegradationManager() *GracefulDegradationManager {
	gdm := &GracefulDegradationManager{
		healthStatus:   make(map[string]*ServiceHealth),
		strategies:     make(map[string]*DegradationStrategy),
		cache:          make(map[string]*CachedResult),
		fallbackChains: make(map[Provider][]Provider),
		healthChecker:  NewHealthChecker(),
		retryManager:   NewAdvancedRetryManager(),
	}

	gdm.initializeDefaultStrategies()
	gdm.initializeFallbackChains()

	return gdm
}

// initializeDefaultStrategies sets up default degradation strategies
func (gdm *GracefulDegradationManager) initializeDefaultStrategies() {
	// Authentication strategy - fallback to other providers
	gdm.strategies["authenticate"] = &DegradationStrategy{
		FallbackEnabled: true,
		CacheEnabled:    true,
		CacheTTL:        5 * time.Minute,
		PartialResults:  false,
		CustomHandlers:  make(map[string]DegradationHandler),
	}

	// List operations strategy - use cache and partial results
	gdm.strategies["list_clusters"] = &DegradationStrategy{
		FallbackEnabled: true,
		CacheEnabled:    true,
		CacheTTL:        10 * time.Minute,
		PartialResults:  true,
		CustomHandlers:  make(map[string]DegradationHandler),
	}

	gdm.strategies["list_registries"] = &DegradationStrategy{
		FallbackEnabled: true,
		CacheEnabled:    true,
		CacheTTL:        10 * time.Minute,
		PartialResults:  true,
		CustomHandlers:  make(map[string]DegradationHandler),
	}

	// Resource operations strategy - aggressive caching and fallbacks
	gdm.strategies["get_cluster"] = &DegradationStrategy{
		FallbackEnabled: true,
		CacheEnabled:    true,
		CacheTTL:        15 * time.Minute,
		PartialResults:  false,
		CustomHandlers:  make(map[string]DegradationHandler),
	}

	gdm.strategies["get_registry"] = &DegradationStrategy{
		FallbackEnabled: true,
		CacheEnabled:    true,
		CacheTTL:        15 * time.Minute,
		PartialResults:  false,
		CustomHandlers:  make(map[string]DegradationHandler),
	}

	// Registry authentication - fallback to other registries
	gdm.strategies["authenticate_registry"] = &DegradationStrategy{
		FallbackEnabled: true,
		CacheEnabled:    true,
		CacheTTL:        30 * time.Minute,
		PartialResults:  false,
		CustomHandlers:  make(map[string]DegradationHandler),
	}
}

// initializeFallbackChains sets up fallback chains for providers
func (gdm *GracefulDegradationManager) initializeFallbackChains() {
	// AWS fallback chain: AWS -> GCP -> Azure
	gdm.fallbackChains[ProviderAWS] = []Provider{ProviderGCP, ProviderAzure}

	// Azure fallback chain: Azure -> AWS -> GCP
	gdm.fallbackChains[ProviderAzure] = []Provider{ProviderAWS, ProviderGCP}

	// GCP fallback chain: GCP -> Azure -> AWS
	gdm.fallbackChains[ProviderGCP] = []Provider{ProviderAzure, ProviderAWS}
}

// ExecuteWithDegradation executes an operation with graceful degradation
func (gdm *GracefulDegradationManager) ExecuteWithDegradation(
	ctx context.Context,
	provider Provider,
	operation string,
	primaryOp func(ctx context.Context, provider Provider) (interface{}, error),
) (interface{}, error) {

	serviceKey := fmt.Sprintf("%s_%s", provider, operation)
	strategy := gdm.getStrategy(operation)

	// Check if we should try cache first
	if strategy.CacheEnabled {
		if cached := gdm.getFromCache(serviceKey); cached != nil {
			return cached.Data, nil
		}
	}

	// Try primary operation
	result, err := gdm.tryOperation(ctx, provider, operation, primaryOp)
	if err == nil {
		// Success - cache result and update health
		if strategy.CacheEnabled {
			gdm.updateCache(serviceKey, result, strategy.CacheTTL, provider, operation)
		}
		gdm.updateServiceHealth(serviceKey, true, nil)
		return result, nil
	}

	// Primary operation failed - update health
	gdm.updateServiceHealth(serviceKey, false, err)

	// Check degradation level and decide on fallback strategy
	health := gdm.getServiceHealth(serviceKey)
	return gdm.handleDegradation(ctx, provider, operation, health, strategy, primaryOp, err)
}

// handleDegradation handles service degradation based on strategy
func (gdm *GracefulDegradationManager) handleDegradation(
	ctx context.Context,
	provider Provider,
	operation string,
	health *ServiceHealth,
	strategy *DegradationStrategy,
	primaryOp func(ctx context.Context, provider Provider) (interface{}, error),
	originalError error,
) (interface{}, error) {

	switch health.DegradationLevel {
	case DegradationNone, DegradationMinor:
		// Light degradation - try cache or retry
		return gdm.handleLightDegradation(ctx, provider, operation, strategy, primaryOp, originalError)

	case DegradationMajor:
		// Major degradation - try fallback providers
		return gdm.handleMajorDegradation(ctx, provider, operation, strategy, primaryOp, originalError)

	case DegradationSevere:
		// Severe degradation - use cache or reduced functionality
		return gdm.handleSevereDegradation(ctx, provider, operation, strategy, originalError)

	case DegradationComplete:
		// Complete degradation - return cached data or error
		return gdm.handleCompleteDegradation(ctx, provider, operation, strategy, originalError)

	default:
		return nil, originalError
	}
}

// handleLightDegradation handles light service degradation
func (gdm *GracefulDegradationManager) handleLightDegradation(
	ctx context.Context,
	provider Provider,
	operation string,
	strategy *DegradationStrategy,
	primaryOp func(ctx context.Context, provider Provider) (interface{}, error),
	originalError error,
) (interface{}, error) {

	// Try cache first
	if strategy.CacheEnabled {
		serviceKey := fmt.Sprintf("%s_%s", provider, operation)
		if cached := gdm.getFromCache(serviceKey); cached != nil {
			return cached.Data, nil
		}
	}

	// Retry the operation with exponential backoff
	var result interface{}
	err := gdm.retryManager.ExecuteWithRetryAndCircuitBreaker(ctx, provider, operation,
		func(ctx context.Context, attempt int) error {
			var opErr error
			result, opErr = primaryOp(ctx, provider)
			return opErr
		})

	if err == nil {
		return result, nil
	}

	// If retry failed and we have a custom handler, try it
	if handler, exists := strategy.CustomHandlers[operation]; exists {
		return handler(ctx, originalError)
	}

	return nil, originalError
}

// handleMajorDegradation handles major service degradation
func (gdm *GracefulDegradationManager) handleMajorDegradation(
	ctx context.Context,
	provider Provider,
	operation string,
	strategy *DegradationStrategy,
	primaryOp func(ctx context.Context, provider Provider) (interface{}, error),
	originalError error,
) (interface{}, error) {

	// Try fallback providers if enabled
	if strategy.FallbackEnabled {
		fallbackProviders := gdm.fallbackChains[provider]
		for _, fallbackProvider := range fallbackProviders {
			// Check if fallback provider is healthy
			fallbackKey := fmt.Sprintf("%s_%s", fallbackProvider, operation)
			if fallbackHealth := gdm.getServiceHealth(fallbackKey); fallbackHealth != nil {
				if fallbackHealth.DegradationLevel <= DegradationMinor {
					result, err := primaryOp(ctx, fallbackProvider)
					if err == nil {
						return result, nil
					}
				}
			} else {
				// Try fallback provider if we don't have health info
				result, err := primaryOp(ctx, fallbackProvider)
				if err == nil {
					return result, nil
				}
			}
		}
	}

	// Fallback to cache if available
	if strategy.CacheEnabled {
		serviceKey := fmt.Sprintf("%s_%s", provider, operation)
		if cached := gdm.getFromCache(serviceKey); cached != nil {
			return cached.Data, nil
		}
	}

	// Try custom handler
	if handler, exists := strategy.CustomHandlers[operation]; exists {
		return handler(ctx, originalError)
	}

	return nil, originalError
}

// handleSevereDegradation handles severe service degradation
func (gdm *GracefulDegradationManager) handleSevereDegradation(
	ctx context.Context,
	provider Provider,
	operation string,
	strategy *DegradationStrategy,
	originalError error,
) (interface{}, error) {

	// Try cache with extended TTL
	if strategy.CacheEnabled {
		serviceKey := fmt.Sprintf("%s_%s", provider, operation)
		if cached := gdm.getFromCache(serviceKey); cached != nil {
			// For severe degradation, accept older cache entries
			if time.Since(cached.Timestamp) < strategy.CacheTTL*3 {
				return cached.Data, nil
			}
		}
	}

	// Return partial results if supported
	if strategy.PartialResults {
		return gdm.getPartialResults(provider, operation), nil
	}

	// Try custom handler
	if handler, exists := strategy.CustomHandlers[operation]; exists {
		return handler(ctx, originalError)
	}

	return nil, NewErrorBuilder(provider, operation).
		BuildUserFriendly(ErrCodeServiceUnavailable,
			"Service is severely degraded",
			"The cloud service is experiencing severe issues. Some features may be unavailable.")
}

// handleCompleteDegradation handles complete service degradation
func (gdm *GracefulDegradationManager) handleCompleteDegradation(
	ctx context.Context,
	provider Provider,
	operation string,
	strategy *DegradationStrategy,
	originalError error,
) (interface{}, error) {

	// Only try cache
	if strategy.CacheEnabled {
		serviceKey := fmt.Sprintf("%s_%s", provider, operation)
		if cached := gdm.getFromCache(serviceKey); cached != nil {
			// For complete degradation, accept any cached data
			return cached.Data, nil
		}
	}

	// Return empty result for operations that support it
	if strategy.PartialResults {
		return gdm.getEmptyResult(operation), nil
	}

	return nil, NewErrorBuilder(provider, operation).
		BuildUserFriendly(ErrCodeServiceUnavailable,
			"Service is completely unavailable",
			"The cloud service is completely unavailable. Please try again later.")
}

// tryOperation executes an operation with timeout and monitoring
func (gdm *GracefulDegradationManager) tryOperation(
	ctx context.Context,
	provider Provider,
	operation string,
	op func(ctx context.Context, provider Provider) (interface{}, error),
) (interface{}, error) {

	start := time.Now()
	result, err := op(ctx, provider)
	duration := time.Since(start)

	// Update performance metrics
	serviceKey := fmt.Sprintf("%s_%s", provider, operation)
	gdm.updatePerformanceMetrics(serviceKey, duration, err)

	return result, err
}

// getStrategy returns the degradation strategy for an operation
func (gdm *GracefulDegradationManager) getStrategy(operation string) *DegradationStrategy {
	gdm.mu.RLock()
	defer gdm.mu.RUnlock()

	if strategy, exists := gdm.strategies[operation]; exists {
		return strategy
	}

	// Return default strategy
	return &DegradationStrategy{
		FallbackEnabled: true,
		CacheEnabled:    true,
		CacheTTL:        5 * time.Minute,
		PartialResults:  false,
		CustomHandlers:  make(map[string]DegradationHandler),
	}
}

// updateServiceHealth updates the health status of a service
func (gdm *GracefulDegradationManager) updateServiceHealth(serviceKey string, success bool, err error) {
	gdm.mu.Lock()
	defer gdm.mu.Unlock()

	health, exists := gdm.healthStatus[serviceKey]
	if !exists {
		health = &ServiceHealth{
			Service:   serviceKey,
			LastCheck: time.Now(),
		}
		gdm.healthStatus[serviceKey] = health
	}

	health.LastCheck = time.Now()
	health.Healthy = success

	if success {
		// Reset error count on success
		if health.ErrorCount > 0 {
			health.ErrorCount = maxInt(0, health.ErrorCount-1)
		}
	} else {
		health.ErrorCount++
		if err != nil {
			health.Description = err.Error()
		}
	}

	// Update degradation level based on error count and rate
	health.DegradationLevel = gdm.calculateDegradationLevel(health)
}

// calculateDegradationLevel calculates degradation level based on health metrics
func (gdm *GracefulDegradationManager) calculateDegradationLevel(health *ServiceHealth) DegradationLevel {
	// Simple degradation calculation based on error count
	switch {
	case health.ErrorCount == 0:
		return DegradationNone
	case health.ErrorCount <= 2:
		return DegradationMinor
	case health.ErrorCount <= 5:
		return DegradationMajor
	case health.ErrorCount <= 10:
		return DegradationSevere
	default:
		return DegradationComplete
	}
}

// getServiceHealth returns the health status of a service
func (gdm *GracefulDegradationManager) getServiceHealth(serviceKey string) *ServiceHealth {
	gdm.mu.RLock()
	defer gdm.mu.RUnlock()

	return gdm.healthStatus[serviceKey]
}

// updatePerformanceMetrics updates performance metrics for a service
func (gdm *GracefulDegradationManager) updatePerformanceMetrics(serviceKey string, duration time.Duration, err error) {
	gdm.mu.Lock()
	defer gdm.mu.Unlock()

	health, exists := gdm.healthStatus[serviceKey]
	if !exists {
		return
	}

	// Simple moving average for response time
	if health.AvgResponseTime == 0 {
		health.AvgResponseTime = duration
	} else {
		health.AvgResponseTime = (health.AvgResponseTime + duration) / 2
	}

	// Update error rate (simplified calculation)
	if err != nil {
		health.ErrorRate = minFloat(1.0, health.ErrorRate+0.1)
	} else {
		health.ErrorRate = maxFloat(0.0, health.ErrorRate-0.05)
	}
}

// Cache operations
func (gdm *GracefulDegradationManager) updateCache(key string, data interface{}, ttl time.Duration, provider Provider, operation string) {
	gdm.mu.Lock()
	defer gdm.mu.Unlock()

	gdm.cache[key] = &CachedResult{
		Data:      data,
		Timestamp: time.Now(),
		TTL:       ttl,
		Provider:  provider,
		Operation: operation,
	}
}

func (gdm *GracefulDegradationManager) getFromCache(key string) *CachedResult {
	gdm.mu.RLock()
	defer gdm.mu.RUnlock()

	cached, exists := gdm.cache[key]
	if !exists {
		return nil
	}

	// Check if cache entry is still valid
	if time.Since(cached.Timestamp) > cached.TTL {
		return nil
	}

	return cached
}

// getPartialResults returns partial results for an operation
func (gdm *GracefulDegradationManager) getPartialResults(provider Provider, operation string) interface{} {
	switch operation {
	case "list_clusters":
		return []*Cluster{} // Empty list instead of error
	case "list_registries":
		return []*Registry{} // Empty list instead of error
	default:
		return nil
	}
}

// getEmptyResult returns an empty result for an operation
func (gdm *GracefulDegradationManager) getEmptyResult(operation string) interface{} {
	switch operation {
	case "list_clusters":
		return []*Cluster{}
	case "list_registries":
		return []*Registry{}
	default:
		return struct{}{}
	}
}

// SetStrategy sets a degradation strategy for an operation
func (gdm *GracefulDegradationManager) SetStrategy(operation string, strategy *DegradationStrategy) {
	gdm.mu.Lock()
	defer gdm.mu.Unlock()

	gdm.strategies[operation] = strategy
}

// SetCustomHandler sets a custom degradation handler for an operation
func (gdm *GracefulDegradationManager) SetCustomHandler(operation string, handler DegradationHandler) {
	gdm.mu.Lock()
	defer gdm.mu.Unlock()

	if strategy, exists := gdm.strategies[operation]; exists {
		strategy.CustomHandlers[operation] = handler
	} else {
		// Create new strategy with custom handler
		gdm.strategies[operation] = &DegradationStrategy{
			FallbackEnabled: true,
			CacheEnabled:    true,
			CacheTTL:        5 * time.Minute,
			CustomHandlers:  map[string]DegradationHandler{operation: handler},
		}
	}
}

// GetHealthStatus returns the health status of all services
func (gdm *GracefulDegradationManager) GetHealthStatus() map[string]*ServiceHealth {
	gdm.mu.RLock()
	defer gdm.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]*ServiceHealth)
	for key, health := range gdm.healthStatus {
		healthCopy := *health
		result[key] = &healthCopy
	}

	return result
}

// ClearCache clears the degradation cache
func (gdm *GracefulDegradationManager) ClearCache() {
	gdm.mu.Lock()
	defer gdm.mu.Unlock()

	gdm.cache = make(map[string]*CachedResult)
}

// ResetHealthStatus resets the health status for a service
func (gdm *GracefulDegradationManager) ResetHealthStatus(serviceKey string) {
	gdm.mu.Lock()
	defer gdm.mu.Unlock()

	if health, exists := gdm.healthStatus[serviceKey]; exists {
		health.ErrorCount = 0
		health.ErrorRate = 0
		health.Healthy = true
		health.DegradationLevel = DegradationNone
		health.Description = ""
		health.LastCheck = time.Now()
	}
}

// HealthChecker provides health checking capabilities
type HealthChecker struct {
	checks   map[string]HealthCheck
	interval time.Duration
	mu       sync.RWMutex
}

// HealthCheck represents a health check function
type HealthCheck func(ctx context.Context) error

// NewHealthChecker creates a new health checker
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		checks:   make(map[string]HealthCheck),
		interval: 30 * time.Second,
	}
}

// RegisterCheck registers a health check
func (hc *HealthChecker) RegisterCheck(name string, check HealthCheck) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.checks[name] = check
}

// RunChecks runs all registered health checks
func (hc *HealthChecker) RunChecks(ctx context.Context) map[string]error {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	results := make(map[string]error)
	for name, check := range hc.checks {
		results[name] = check(ctx)
	}

	return results
}

// StartPeriodicChecks starts periodic health checks
func (hc *HealthChecker) StartPeriodicChecks(ctx context.Context, callback func(map[string]error)) {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			results := hc.RunChecks(ctx)
			if callback != nil {
				callback(results)
			}
		}
	}
}

// Utility functions
func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
