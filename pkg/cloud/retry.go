package cloud

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

// RetryConfig represents configuration for retry logic
type RetryConfig struct {
	MaxAttempts     int           `json:"max_attempts"`
	InitialDelay    time.Duration `json:"initial_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	BackoffFactor   float64       `json:"backoff_factor"`
	JitterPercent   float64       `json:"jitter_percent"`
	Strategy        RetryStrategy `json:"strategy"`
	RetryableErrors []string      `json:"retryable_errors,omitempty"`
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		JitterPercent: 0.1,
		Strategy:      RetryStrategyExponential,
	}
}

// ExponentialRetryConfig returns a configuration for exponential backoff
func ExponentialRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:   5,
		InitialDelay:  500 * time.Millisecond,
		MaxDelay:      60 * time.Second,
		BackoffFactor: 2.0,
		JitterPercent: 0.2,
		Strategy:      RetryStrategyExponential,
	}
}

// LinearRetryConfig returns a configuration for linear backoff
func LinearRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  2 * time.Second,
		MaxDelay:      10 * time.Second,
		BackoffFactor: 1.0,
		JitterPercent: 0.1,
		Strategy:      RetryStrategyLinear,
	}
}

// ImmediateRetryConfig returns a configuration for immediate retry
func ImmediateRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:   2,
		InitialDelay:  0,
		MaxDelay:      0,
		BackoffFactor: 1.0,
		JitterPercent: 0,
		Strategy:      RetryStrategyImmediate,
	}
}

// RetryableOperation represents an operation that can be retried
type RetryableOperation func(ctx context.Context, attempt int) error

// RetryResult represents the result of a retry operation
type RetryResult struct {
	Success      bool          `json:"success"`
	Attempts     int           `json:"attempts"`
	TotalDelay   time.Duration `json:"total_delay"`
	LastError    error         `json:"last_error,omitempty"`
	ErrorHistory []error       `json:"error_history,omitempty"`
}

// Retrier handles retry logic for operations
type Retrier struct {
	config     RetryConfig
	classifier *ErrorClassifier
	rand       *rand.Rand
}

// NewRetrier creates a new retrier with the given configuration
func NewRetrier(config RetryConfig) *Retrier {
	return &Retrier{
		config:     config,
		classifier: NewErrorClassifier(),
		rand:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// NewDefaultRetrier creates a new retrier with default configuration
func NewDefaultRetrier() *Retrier {
	return NewRetrier(DefaultRetryConfig())
}

// Retry executes an operation with retry logic
func (r *Retrier) Retry(ctx context.Context, operation RetryableOperation) *RetryResult {
	result := &RetryResult{
		ErrorHistory: make([]error, 0, r.config.MaxAttempts),
	}

	var totalDelay time.Duration

	for attempt := 1; attempt <= r.config.MaxAttempts; attempt++ {
		err := operation(ctx, attempt)

		if err == nil {
			result.Success = true
			result.Attempts = attempt
			result.TotalDelay = totalDelay
			return result
		}

		result.LastError = err
		result.ErrorHistory = append(result.ErrorHistory, err)

		// Check if error is retryable
		if !r.isRetryable(err) {
			result.Attempts = attempt
			result.TotalDelay = totalDelay
			return result
		}

		// Don't wait after the last attempt
		if attempt == r.config.MaxAttempts {
			result.Attempts = attempt
			result.TotalDelay = totalDelay
			return result
		}

		// Calculate delay for next attempt
		delay := r.calculateDelay(attempt)
		totalDelay += delay

		// Wait before next attempt
		select {
		case <-ctx.Done():
			result.LastError = ctx.Err()
			result.Attempts = attempt
			result.TotalDelay = totalDelay
			return result
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	result.Attempts = r.config.MaxAttempts
	result.TotalDelay = totalDelay
	return result
}

// RetryWithBackoff executes an operation with exponential backoff
func (r *Retrier) RetryWithBackoff(ctx context.Context, operation RetryableOperation) error {
	result := r.Retry(ctx, operation)

	if result.Success {
		return nil
	}

	return result.LastError
}

// isRetryable determines if an error is retryable
func (r *Retrier) isRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Use error classifier to determine if error is retryable
	classification := r.classifier.Classify(err)
	return classification.Retryable
}

// calculateDelay calculates the delay before the next attempt
func (r *Retrier) calculateDelay(attempt int) time.Duration {
	var delay time.Duration

	switch r.config.Strategy {
	case RetryStrategyImmediate:
		return 0
	case RetryStrategyLinear:
		delay = time.Duration(attempt) * r.config.InitialDelay
	case RetryStrategyExponential:
		delay = time.Duration(float64(r.config.InitialDelay) * math.Pow(r.config.BackoffFactor, float64(attempt-1)))
	default:
		delay = r.config.InitialDelay
	}

	// Apply maximum delay limit
	if delay > r.config.MaxDelay {
		delay = r.config.MaxDelay
	}

	// Add jitter to prevent thundering herd
	if r.config.JitterPercent > 0 {
		jitter := time.Duration(float64(delay) * r.config.JitterPercent * (r.rand.Float64()*2 - 1))
		delay += jitter
	}

	// Ensure delay is not negative
	if delay < 0 {
		delay = 0
	}

	return delay
}

// CloudRetryManager manages retry logic for cloud operations
type CloudRetryManager struct {
	defaultRetrier    *Retrier
	providerRetrieers map[Provider]*Retrier
	operationConfigs  map[string]RetryConfig
}

// NewCloudRetryManager creates a new cloud retry manager
func NewCloudRetryManager() *CloudRetryManager {
	return &CloudRetryManager{
		defaultRetrier:    NewDefaultRetrier(),
		providerRetrieers: make(map[Provider]*Retrier),
		operationConfigs:  make(map[string]RetryConfig),
	}
}

// SetProviderRetryConfig sets retry configuration for a specific provider
func (crm *CloudRetryManager) SetProviderRetryConfig(provider Provider, config RetryConfig) {
	crm.providerRetrieers[provider] = NewRetrier(config)
}

// SetOperationRetryConfig sets retry configuration for a specific operation
func (crm *CloudRetryManager) SetOperationRetryConfig(operation string, config RetryConfig) {
	crm.operationConfigs[operation] = config
}

// GetRetrier gets the appropriate retrier for a provider and operation
func (crm *CloudRetryManager) GetRetrier(provider Provider, operation string) *Retrier {
	// Check for operation-specific config first
	if config, exists := crm.operationConfigs[operation]; exists {
		return NewRetrier(config)
	}

	// Check for provider-specific retrier
	if retrier, exists := crm.providerRetrieers[provider]; exists {
		return retrier
	}

	// Return default retrier
	return crm.defaultRetrier
}

// ExecuteWithRetry executes an operation with appropriate retry logic
func (crm *CloudRetryManager) ExecuteWithRetry(ctx context.Context, provider Provider, operation string, op RetryableOperation) error {
	retrier := crm.GetRetrier(provider, operation)
	return retrier.RetryWithBackoff(ctx, op)
}

// InitializeDefaultConfigs initializes default retry configurations for common operations
func (crm *CloudRetryManager) InitializeDefaultConfigs() {
	// Authentication operations - quick retries
	crm.SetOperationRetryConfig("authenticate", RetryConfig{
		MaxAttempts:   2,
		InitialDelay:  500 * time.Millisecond,
		MaxDelay:      2 * time.Second,
		BackoffFactor: 2.0,
		JitterPercent: 0.1,
		Strategy:      RetryStrategyExponential,
	})

	// List operations - moderate retries
	crm.SetOperationRetryConfig("list_clusters", RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      10 * time.Second,
		BackoffFactor: 2.0,
		JitterPercent: 0.2,
		Strategy:      RetryStrategyExponential,
	})

	crm.SetOperationRetryConfig("list_registries", RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      10 * time.Second,
		BackoffFactor: 2.0,
		JitterPercent: 0.2,
		Strategy:      RetryStrategyExponential,
	})

	// Resource operations - aggressive retries
	crm.SetOperationRetryConfig("get_cluster", RetryConfig{
		MaxAttempts:   5,
		InitialDelay:  2 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		JitterPercent: 0.3,
		Strategy:      RetryStrategyExponential,
	})

	crm.SetOperationRetryConfig("get_registry", RetryConfig{
		MaxAttempts:   5,
		InitialDelay:  2 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		JitterPercent: 0.3,
		Strategy:      RetryStrategyExponential,
	})

	// Network operations - very aggressive retries
	crm.SetOperationRetryConfig("network", RetryConfig{
		MaxAttempts:   10,
		InitialDelay:  1 * time.Second,
		MaxDelay:      60 * time.Second,
		BackoffFactor: 1.5,
		JitterPercent: 0.4,
		Strategy:      RetryStrategyExponential,
	})

	// Registry authentication - special handling
	crm.SetOperationRetryConfig("authenticate_registry", RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  2 * time.Second,
		MaxDelay:      15 * time.Second,
		BackoffFactor: 2.0,
		JitterPercent: 0.15,
		Strategy:      RetryStrategyExponential,
	})

	// Provider-specific configurations
	crm.SetProviderRetryConfig(ProviderAWS, RetryConfig{
		MaxAttempts:   4,
		InitialDelay:  1 * time.Second,
		MaxDelay:      20 * time.Second,
		BackoffFactor: 2.0,
		JitterPercent: 0.2,
		Strategy:      RetryStrategyExponential,
	})

	crm.SetProviderRetryConfig(ProviderAzure, RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      15 * time.Second,
		BackoffFactor: 2.0,
		JitterPercent: 0.15,
		Strategy:      RetryStrategyExponential,
	})

	crm.SetProviderRetryConfig(ProviderGCP, RetryConfig{
		MaxAttempts:   5,
		InitialDelay:  800 * time.Millisecond,
		MaxDelay:      25 * time.Second,
		BackoffFactor: 1.8,
		JitterPercent: 0.25,
		Strategy:      RetryStrategyExponential,
	})
}

// CircuitBreaker implements circuit breaker pattern for cloud operations
type CircuitBreaker struct {
	name            string
	maxFailures     int
	resetTimeout    time.Duration
	state           CircuitBreakerState
	failures        int
	lastFailureTime time.Time
	mu              sync.RWMutex
}

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	CircuitBreakerClosed CircuitBreakerState = iota
	CircuitBreakerOpen
	CircuitBreakerHalfOpen
)

// String returns the string representation of circuit breaker state
func (s CircuitBreakerState) String() string {
	switch s {
	case CircuitBreakerClosed:
		return "closed"
	case CircuitBreakerOpen:
		return "open"
	case CircuitBreakerHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		name:         name,
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        CircuitBreakerClosed,
	}
}

// Execute executes an operation through the circuit breaker
func (cb *CircuitBreaker) Execute(ctx context.Context, operation func(ctx context.Context) error) error {
	if !cb.canExecute() {
		return NewErrorBuilder(ProviderAWS, "circuit_breaker").
			BuildUserFriendly(ErrCodeServiceUnavailable,
				fmt.Sprintf("Circuit breaker %s is open", cb.name),
				fmt.Sprintf("Service %s is temporarily unavailable due to repeated failures", cb.name))
	}

	err := operation(ctx)
	cb.onResult(err)
	return err
}

// canExecute checks if the operation can be executed
func (cb *CircuitBreaker) canExecute() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case CircuitBreakerClosed:
		return true
	case CircuitBreakerOpen:
		// Check if reset timeout has passed
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			// Double-check after acquiring write lock
			if cb.state == CircuitBreakerOpen && time.Since(cb.lastFailureTime) > cb.resetTimeout {
				cb.state = CircuitBreakerHalfOpen
			}
			cb.mu.Unlock()
			cb.mu.RLock()
			return cb.state == CircuitBreakerHalfOpen
		}
		return false
	case CircuitBreakerHalfOpen:
		return true
	default:
		return false
	}
}

// onResult handles the result of an operation
func (cb *CircuitBreaker) onResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err == nil {
		// Success - reset to closed state
		cb.failures = 0
		cb.state = CircuitBreakerClosed
		return
	}

	// Failure
	cb.failures++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case CircuitBreakerClosed:
		if cb.failures >= cb.maxFailures {
			cb.state = CircuitBreakerOpen
		}
	case CircuitBreakerHalfOpen:
		// Any failure in half-open state returns to open
		cb.state = CircuitBreakerOpen
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetStats returns statistics about the circuit breaker
func (cb *CircuitBreaker) GetStats() CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return CircuitBreakerStats{
		Name:            cb.name,
		State:           cb.state,
		Failures:        cb.failures,
		MaxFailures:     cb.maxFailures,
		LastFailureTime: cb.lastFailureTime,
		ResetTimeout:    cb.resetTimeout,
	}
}

// CircuitBreakerStats represents circuit breaker statistics
type CircuitBreakerStats struct {
	Name            string              `json:"name"`
	State           CircuitBreakerState `json:"state"`
	Failures        int                 `json:"failures"`
	MaxFailures     int                 `json:"max_failures"`
	LastFailureTime time.Time           `json:"last_failure_time"`
	ResetTimeout    time.Duration       `json:"reset_timeout"`
}

// AdvancedRetryManager combines retry logic with circuit breaker pattern
type AdvancedRetryManager struct {
	retryManager    *CloudRetryManager
	circuitBreakers map[string]*CircuitBreaker
	mu              sync.RWMutex
}

// NewAdvancedRetryManager creates a new advanced retry manager
func NewAdvancedRetryManager() *AdvancedRetryManager {
	arm := &AdvancedRetryManager{
		retryManager:    NewCloudRetryManager(),
		circuitBreakers: make(map[string]*CircuitBreaker),
	}

	arm.retryManager.InitializeDefaultConfigs()
	arm.initializeDefaultCircuitBreakers()

	return arm
}

// initializeDefaultCircuitBreakers initializes default circuit breakers
func (arm *AdvancedRetryManager) initializeDefaultCircuitBreakers() {
	// Provider-level circuit breakers
	arm.circuitBreakers["aws"] = NewCircuitBreaker("aws", 5, 2*time.Minute)
	arm.circuitBreakers["azure"] = NewCircuitBreaker("azure", 5, 2*time.Minute)
	arm.circuitBreakers["gcp"] = NewCircuitBreaker("gcp", 5, 2*time.Minute)

	// Operation-level circuit breakers
	arm.circuitBreakers["list_clusters"] = NewCircuitBreaker("list_clusters", 3, 1*time.Minute)
	arm.circuitBreakers["list_registries"] = NewCircuitBreaker("list_registries", 3, 1*time.Minute)
	arm.circuitBreakers["authenticate"] = NewCircuitBreaker("authenticate", 2, 30*time.Second)
}

// ExecuteWithRetryAndCircuitBreaker executes an operation with both retry and circuit breaker
func (arm *AdvancedRetryManager) ExecuteWithRetryAndCircuitBreaker(
	ctx context.Context,
	provider Provider,
	operation string,
	op RetryableOperation,
) error {
	circuitBreakerKey := fmt.Sprintf("%s_%s", provider, operation)

	// Get or create circuit breaker
	circuitBreaker := arm.getOrCreateCircuitBreaker(circuitBreakerKey, operation)

	// Execute through circuit breaker
	return circuitBreaker.Execute(ctx, func(ctx context.Context) error {
		return arm.retryManager.ExecuteWithRetry(ctx, provider, operation, op)
	})
}

// getOrCreateCircuitBreaker gets or creates a circuit breaker for the given key
func (arm *AdvancedRetryManager) getOrCreateCircuitBreaker(key, operation string) *CircuitBreaker {
	arm.mu.RLock()
	if cb, exists := arm.circuitBreakers[key]; exists {
		arm.mu.RUnlock()
		return cb
	}
	arm.mu.RUnlock()

	arm.mu.Lock()
	defer arm.mu.Unlock()

	// Double-check after acquiring write lock
	if cb, exists := arm.circuitBreakers[key]; exists {
		return cb
	}

	// Create new circuit breaker based on operation type
	var cb *CircuitBreaker
	switch operation {
	case "authenticate":
		cb = NewCircuitBreaker(key, 2, 30*time.Second)
	case "list_clusters", "list_registries":
		cb = NewCircuitBreaker(key, 3, 1*time.Minute)
	default:
		cb = NewCircuitBreaker(key, 5, 2*time.Minute)
	}

	arm.circuitBreakers[key] = cb
	return cb
}

// GetCircuitBreakerStats returns statistics for all circuit breakers
func (arm *AdvancedRetryManager) GetCircuitBreakerStats() map[string]CircuitBreakerStats {
	arm.mu.RLock()
	defer arm.mu.RUnlock()

	stats := make(map[string]CircuitBreakerStats)
	for key, cb := range arm.circuitBreakers {
		stats[key] = cb.GetStats()
	}

	return stats
}

// ResetCircuitBreaker manually resets a circuit breaker
func (arm *AdvancedRetryManager) ResetCircuitBreaker(key string) error {
	arm.mu.Lock()
	defer arm.mu.Unlock()

	if cb, exists := arm.circuitBreakers[key]; exists {
		cb.mu.Lock()
		cb.state = CircuitBreakerClosed
		cb.failures = 0
		cb.mu.Unlock()
		return nil
	}

	return fmt.Errorf("circuit breaker %s not found", key)
}

// SetRetryConfig sets retry configuration for a provider or operation
func (arm *AdvancedRetryManager) SetRetryConfig(provider Provider, operation string, config RetryConfig) {
	if operation != "" {
		arm.retryManager.SetOperationRetryConfig(operation, config)
	} else {
		arm.retryManager.SetProviderRetryConfig(provider, config)
	}
}

// RateLimitedRetrier implements rate-limited retry logic
type RateLimitedRetrier struct {
	retrier     *Retrier
	rateLimiter *time.Ticker
	semaphore   chan struct{}
}

// NewRateLimitedRetrier creates a new rate-limited retrier
func NewRateLimitedRetrier(config RetryConfig, requestsPerSecond float64, maxConcurrent int) *RateLimitedRetrier {
	interval := time.Duration(float64(time.Second) / requestsPerSecond)

	return &RateLimitedRetrier{
		retrier:     NewRetrier(config),
		rateLimiter: time.NewTicker(interval),
		semaphore:   make(chan struct{}, maxConcurrent),
	}
}

// Execute executes an operation with rate limiting and retry logic
func (rlr *RateLimitedRetrier) Execute(ctx context.Context, operation RetryableOperation) error {
	// Acquire semaphore slot
	select {
	case rlr.semaphore <- struct{}{}:
		defer func() { <-rlr.semaphore }()
	case <-ctx.Done():
		return ctx.Err()
	}

	// Wait for rate limiter
	select {
	case <-rlr.rateLimiter.C:
		// Continue
	case <-ctx.Done():
		return ctx.Err()
	}

	// Execute with retry
	return rlr.retrier.RetryWithBackoff(ctx, operation)
}

// Close stops the rate limiter
func (rlr *RateLimitedRetrier) Close() {
	rlr.rateLimiter.Stop()
}
