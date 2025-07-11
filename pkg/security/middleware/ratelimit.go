package middleware

import (
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	// Global rate limit
	RequestsPerMinute int `yaml:"requests_per_minute" json:"requests_per_minute"`
	BurstSize         int `yaml:"burst_size" json:"burst_size"`

	// Per-IP rate limiting
	PerIPRequestsPerMinute int `yaml:"per_ip_requests_per_minute" json:"per_ip_requests_per_minute"`
	PerIPBurstSize         int `yaml:"per_ip_burst_size" json:"per_ip_burst_size"`

	// Endpoint-specific limits
	EndpointLimits map[string]EndpointLimit `yaml:"endpoint_limits" json:"endpoint_limits"`

	// Whitelist IPs that bypass rate limiting
	WhitelistIPs []string `yaml:"whitelist_ips" json:"whitelist_ips"`

	// Response headers
	IncludeHeaders bool `yaml:"include_headers" json:"include_headers"`
}

// EndpointLimit represents rate limit for specific endpoint
type EndpointLimit struct {
	RequestsPerMinute int `yaml:"requests_per_minute" json:"requests_per_minute"`
	BurstSize         int `yaml:"burst_size" json:"burst_size"`
}

// DefaultRateLimitConfig provides default rate limits
var DefaultRateLimitConfig = RateLimitConfig{
	RequestsPerMinute:      1000,
	BurstSize:              50,
	PerIPRequestsPerMinute: 100,
	PerIPBurstSize:         10,
	IncludeHeaders:         true,
	EndpointLimits: map[string]EndpointLimit{
		"/api/auth/login": {
			RequestsPerMinute: 5,
			BurstSize:         2,
		},
		"/api/auth/register": {
			RequestsPerMinute: 3,
			BurstSize:         1,
		},
		"/api/deploy": {
			RequestsPerMinute: 10,
			BurstSize:         3,
		},
	},
}

// rateLimiter implements token bucket algorithm
type rateLimiter struct {
	tokens    float64
	capacity  float64
	rate      float64
	lastCheck time.Time
	mu        sync.Mutex
}

// newRateLimiter creates a new rate limiter
func newRateLimiter(rate float64, burst int) *rateLimiter {
	return &rateLimiter{
		tokens:    float64(burst),
		capacity:  float64(burst),
		rate:      rate,
		lastCheck: time.Now(),
	}
}

// allow checks if request is allowed
func (r *rateLimiter) allow() (bool, float64, float64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(r.lastCheck).Seconds()
	r.lastCheck = now

	// Add tokens based on elapsed time
	r.tokens += elapsed * r.rate
	if r.tokens > r.capacity {
		r.tokens = r.capacity
	}

	// Check if we have tokens available
	if r.tokens >= 1 {
		r.tokens--
		return true, r.tokens, r.capacity
	}

	return false, r.tokens, r.capacity
}

// RateLimitMiddleware provides rate limiting middleware
type RateLimitMiddleware struct {
	config           RateLimitConfig
	globalLimiter    *rateLimiter
	ipLimiters       map[string]*rateLimiter
	endpointLimiters map[string]*rateLimiter
	whitelistMap     map[string]bool
	logger           *zap.Logger
	mu               sync.RWMutex
}

// NewRateLimitMiddleware creates a new rate limit middleware
func NewRateLimitMiddleware(config RateLimitConfig, logger *zap.Logger) *RateLimitMiddleware {
	// Apply defaults
	if config.RequestsPerMinute == 0 {
		config.RequestsPerMinute = DefaultRateLimitConfig.RequestsPerMinute
	}
	if config.BurstSize == 0 {
		config.BurstSize = DefaultRateLimitConfig.BurstSize
	}
	if config.PerIPRequestsPerMinute == 0 {
		config.PerIPRequestsPerMinute = DefaultRateLimitConfig.PerIPRequestsPerMinute
	}
	if config.PerIPBurstSize == 0 {
		config.PerIPBurstSize = DefaultRateLimitConfig.PerIPBurstSize
	}

	// Create whitelist map
	whitelistMap := make(map[string]bool)
	for _, ip := range config.WhitelistIPs {
		whitelistMap[ip] = true
	}

	// Create endpoint limiters
	endpointLimiters := make(map[string]*rateLimiter)
	for endpoint, limit := range config.EndpointLimits {
		rate := float64(limit.RequestsPerMinute) / 60.0
		endpointLimiters[endpoint] = newRateLimiter(rate, limit.BurstSize)
	}

	return &RateLimitMiddleware{
		config:           config,
		globalLimiter:    newRateLimiter(float64(config.RequestsPerMinute)/60.0, config.BurstSize),
		ipLimiters:       make(map[string]*rateLimiter),
		endpointLimiters: endpointLimiters,
		whitelistMap:     whitelistMap,
		logger:           logger,
	}
}

// Apply returns the rate limiting middleware handler
func (m *RateLimitMiddleware) Apply() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ip := c.IP()

		// Check whitelist
		if m.whitelistMap[ip] {
			return c.Next()
		}

		// Check global rate limit
		allowed, remaining, limit := m.globalLimiter.allow()
		if !allowed {
			m.logger.Warn("global rate limit exceeded",
				zap.String("ip", ip),
				zap.String("path", c.Path()))

			return m.rateLimitExceeded(c, remaining, limit)
		}

		// Check per-IP rate limit
		ipLimiter := m.getIPLimiter(ip)
		allowed, remaining, limit = ipLimiter.allow()
		if !allowed {
			m.logger.Warn("IP rate limit exceeded",
				zap.String("ip", ip),
				zap.String("path", c.Path()))

			return m.rateLimitExceeded(c, remaining, limit)
		}

		// Check endpoint-specific rate limit
		if endpointLimiter, exists := m.endpointLimiters[c.Path()]; exists {
			allowed, remaining, limit = endpointLimiter.allow()
			if !allowed {
				m.logger.Warn("endpoint rate limit exceeded",
					zap.String("ip", ip),
					zap.String("path", c.Path()))

				return m.rateLimitExceeded(c, remaining, limit)
			}
		}

		// Add rate limit headers if configured
		if m.config.IncludeHeaders {
			c.Set("X-RateLimit-Limit", fmt.Sprintf("%.0f", limit))
			c.Set("X-RateLimit-Remaining", fmt.Sprintf("%.0f", remaining))
			c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Minute).Unix()))
		}

		return c.Next()
	}
}

// DDoSProtection provides additional DDoS protection
func (m *RateLimitMiddleware) DDoSProtection() fiber.Handler {
	// Track request patterns for DDoS detection
	type requestPattern struct {
		count      int
		lastSeen   time.Time
		blocked    bool
		blockUntil time.Time
	}

	patterns := make(map[string]*requestPattern)
	var patternsMu sync.RWMutex

	// Cleanup old patterns periodically
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			patternsMu.Lock()
			now := time.Now()
			for key, pattern := range patterns {
				if now.Sub(pattern.lastSeen) > 10*time.Minute {
					delete(patterns, key)
				}
			}
			patternsMu.Unlock()
		}
	}()

	return func(c *fiber.Ctx) error {
		ip := c.IP()
		userAgent := c.Get("User-Agent")
		key := fmt.Sprintf("%s:%s", ip, userAgent)

		patternsMu.Lock()
		pattern, exists := patterns[key]
		if !exists {
			pattern = &requestPattern{
				count:    0,
				lastSeen: time.Now(),
			}
			patterns[key] = pattern
		}

		now := time.Now()

		// Check if currently blocked
		if pattern.blocked && now.Before(pattern.blockUntil) {
			patternsMu.Unlock()

			m.logger.Warn("DDoS protection: blocked request",
				zap.String("ip", ip),
				zap.String("user_agent", userAgent),
				zap.Time("blocked_until", pattern.blockUntil))

			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   "too_many_requests",
				"message": "temporarily blocked due to suspicious activity",
			})
		}

		// Update pattern
		if now.Sub(pattern.lastSeen) < time.Second {
			pattern.count++
		} else {
			pattern.count = 1
		}
		pattern.lastSeen = now

		// Check for suspicious patterns
		if pattern.count > 10 { // More than 10 requests per second
			pattern.blocked = true
			pattern.blockUntil = now.Add(5 * time.Minute)
			patternsMu.Unlock()

			m.logger.Warn("DDoS protection: blocking suspicious pattern",
				zap.String("ip", ip),
				zap.String("user_agent", userAgent),
				zap.Int("request_count", pattern.count))

			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   "too_many_requests",
				"message": "temporarily blocked due to suspicious activity",
			})
		}

		patternsMu.Unlock()

		return c.Next()
	}
}

// getIPLimiter gets or creates rate limiter for IP
func (m *RateLimitMiddleware) getIPLimiter(ip string) *rateLimiter {
	m.mu.RLock()
	limiter, exists := m.ipLimiters[ip]
	m.mu.RUnlock()

	if exists {
		return limiter
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double check after acquiring write lock
	if limiter, exists := m.ipLimiters[ip]; exists {
		return limiter
	}

	// Create new limiter
	rate := float64(m.config.PerIPRequestsPerMinute) / 60.0
	limiter = newRateLimiter(rate, m.config.PerIPBurstSize)
	m.ipLimiters[ip] = limiter

	// Clean up old limiters if too many
	if len(m.ipLimiters) > 10000 {
		m.cleanupIPLimiters()
	}

	return limiter
}

// cleanupIPLimiters removes old IP limiters
func (m *RateLimitMiddleware) cleanupIPLimiters() {
	// This should be called with write lock held
	now := time.Now()
	for ip, limiter := range m.ipLimiters {
		limiter.mu.Lock()
		if now.Sub(limiter.lastCheck) > 10*time.Minute {
			delete(m.ipLimiters, ip)
		}
		limiter.mu.Unlock()
	}
}

// rateLimitExceeded handles rate limit exceeded response
func (m *RateLimitMiddleware) rateLimitExceeded(c *fiber.Ctx, remaining, limit float64) error {
	retryAfter := int(60 - (remaining * 60 / limit))
	if retryAfter < 1 {
		retryAfter = 1
	}

	c.Set("Retry-After", fmt.Sprintf("%d", retryAfter))

	if m.config.IncludeHeaders {
		c.Set("X-RateLimit-Limit", fmt.Sprintf("%.0f", limit))
		c.Set("X-RateLimit-Remaining", "0")
		c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Duration(retryAfter)*time.Second).Unix()))
	}

	return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
		"error":       "rate_limit_exceeded",
		"message":     "too many requests",
		"retry_after": retryAfter,
	})
}
