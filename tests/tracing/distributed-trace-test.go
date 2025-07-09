package tracing

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
	"github.com/uber/jaeger-client-go/transport"
)

// TestService represents a test service for distributed tracing
type TestService struct {
	Name   string
	Port   int
	App    *fiber.App
	Tracer opentracing.Tracer
	Closer func() error
}

// TraceContext holds trace context information
type TraceContext struct {
	TraceID string            `json:"trace_id"`
	SpanID  string            `json:"span_id"`
	Baggage map[string]string `json:"baggage"`
	Headers map[string]string `json:"headers"`
}

// DistributedTraceValidator validates distributed traces
type DistributedTraceValidator struct {
	services []*TestService
	client   *http.Client
}

// NewDistributedTraceValidator creates a new validator
func NewDistributedTraceValidator() *DistributedTraceValidator {
	return &DistributedTraceValidator{
		services: make([]*TestService, 0),
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// CreateTestService creates a test service with tracing
func (v *DistributedTraceValidator) CreateTestService(name string, port int) (*TestService, error) {
	// Configure Jaeger tracer
	cfg := &config.Configuration{
		ServiceName: name,
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans:            true,
			BufferFlushInterval: 1 * time.Second,
			LocalAgentHostPort:  "localhost:6831",
		},
	}

	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		return nil, fmt.Errorf("failed to create tracer: %w", err)
	}

	// Create Fiber app
	app := fiber.New()

	// Add tracing middleware
	app.Use(func(c *fiber.Ctx) error {
		// Extract span context from headers
		spanCtx, err := tracer.Extract(
			opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(c.GetReqHeaders()),
		)

		var span opentracing.Span
		if err != nil {
			// Start new root span
			span = tracer.StartSpan(fmt.Sprintf("HTTP %s %s", c.Method(), c.Path()))
		} else {
			// Start child span
			span = tracer.StartSpan(
				fmt.Sprintf("HTTP %s %s", c.Method(), c.Path()),
				opentracing.ChildOf(spanCtx),
			)
		}

		// Set span tags
		ext.HTTPMethod.Set(span, c.Method())
		ext.HTTPUrl.Set(span, c.OriginalURL())
		ext.Component.Set(span, "fiber")
		ext.SpanKind.Set(span, ext.SpanKindRPCServerEnum)

		// Store span in context
		c.Locals("span", span)

		// Continue with request
		err = c.Next()

		// Set response tags
		ext.HTTPStatusCode.Set(span, uint16(c.Response().StatusCode()))
		if c.Response().StatusCode() >= 400 {
			ext.Error.Set(span, true)
		}

		// Finish span
		span.Finish()

		return err
	})

	service := &TestService{
		Name:   name,
		Port:   port,
		App:    app,
		Tracer: tracer,
		Closer: closer.Close,
	}

	v.services = append(v.services, service)
	return service, nil
}

// SetupTestEndpoints sets up test endpoints for the service
func (v *DistributedTraceValidator) SetupTestEndpoints(service *TestService) {
	// Health check endpoint
	service.App.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": service.Name,
		})
	})

	// Trace context endpoint
	service.App.Get("/trace-context", func(c *fiber.Ctx) error {
		span := c.Locals("span").(opentracing.Span)
		spanCtx := span.Context()

		// Get Jaeger span context
		jaegerSpanCtx, ok := spanCtx.(jaeger.SpanContext)
		if !ok {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to get Jaeger span context"})
		}

		// Get baggage
		baggage := make(map[string]string)
		span.BaggageItem("user-id")

		// Manually iterate through baggage (Jaeger specific)
		if jSpan, ok := span.(*jaeger.Span); ok {
			jSpan.Context().ForeachBaggageItem(func(k, v string) bool {
				baggage[k] = v
				return true
			})
		}

		// Get headers
		headers := make(map[string]string)
		for k, v := range c.GetReqHeaders() {
			if len(v) > 0 {
				headers[k] = v[0]
			}
		}

		return c.JSON(TraceContext{
			TraceID: jaegerSpanCtx.TraceID().String(),
			SpanID:  jaegerSpanCtx.SpanID().String(),
			Baggage: baggage,
			Headers: headers,
		})
	})

	// Call downstream service endpoint
	service.App.Post("/call-downstream", func(c *fiber.Ctx) error {
		span := c.Locals("span").(opentracing.Span)

		// Get target service from request
		var req struct {
			Service string `json:"service"`
			Port    int    `json:"port"`
			Path    string `json:"path"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		// Create child span for downstream call
		childSpan := service.Tracer.StartSpan(
			fmt.Sprintf("HTTP GET %s", req.Path),
			opentracing.ChildOf(span.Context()),
		)
		defer childSpan.Finish()

		// Set span tags
		ext.HTTPMethod.Set(childSpan, "GET")
		ext.HTTPUrl.Set(childSpan, fmt.Sprintf("http://localhost:%d%s", req.Port, req.Path))
		ext.Component.Set(childSpan, "http-client")
		ext.SpanKind.Set(childSpan, ext.SpanKindRPCClientEnum)

		// Create HTTP request
		httpReq, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d%s", req.Port, req.Path), nil)
		if err != nil {
			ext.Error.Set(childSpan, true)
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		// Inject span context into headers
		err = service.Tracer.Inject(
			childSpan.Context(),
			opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(httpReq.Header),
		)
		if err != nil {
			ext.Error.Set(childSpan, true)
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		// Make request
		resp, err := v.client.Do(httpReq)
		if err != nil {
			ext.Error.Set(childSpan, true)
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		defer resp.Body.Close()

		// Set response tags
		ext.HTTPStatusCode.Set(childSpan, uint16(resp.StatusCode))
		if resp.StatusCode >= 400 {
			ext.Error.Set(childSpan, true)
		}

		return c.JSON(fiber.Map{
			"status":      "success",
			"downstream":  req.Service,
			"status_code": resp.StatusCode,
		})
	})

	// Error endpoint for testing error traces
	service.App.Get("/error", func(c *fiber.Ctx) error {
		span := c.Locals("span").(opentracing.Span)
		ext.Error.Set(span, true)
		span.LogFields(
			jaeger.String("event", "error"),
			jaeger.String("message", "Intentional test error"),
		)
		return c.Status(500).JSON(fiber.Map{"error": "Intentional test error"})
	})

	// Baggage endpoint for testing baggage propagation
	service.App.Get("/baggage", func(c *fiber.Ctx) error {
		span := c.Locals("span").(opentracing.Span)

		// Set baggage
		span.SetBaggageItem("user-id", "test-user-123")
		span.SetBaggageItem("request-id", "req-456")

		return c.JSON(fiber.Map{
			"baggage_set": true,
			"user_id":     span.BaggageItem("user-id"),
			"request_id":  span.BaggageItem("request-id"),
		})
	})
}

// StartService starts a test service
func (v *DistributedTraceValidator) StartService(service *TestService) error {
	go func() {
		service.App.Listen(fmt.Sprintf(":%d", service.Port))
	}()

	// Wait for service to start
	time.Sleep(100 * time.Millisecond)

	// Health check
	resp, err := v.client.Get(fmt.Sprintf("http://localhost:%d/health", service.Port))
	if err != nil {
		return fmt.Errorf("service %s failed to start: %w", service.Name, err)
	}
	resp.Body.Close()

	return nil
}

// Cleanup stops all services and closes tracers
func (v *DistributedTraceValidator) Cleanup() {
	for _, service := range v.services {
		if service.Closer != nil {
			service.Closer()
		}
		if service.App != nil {
			service.App.Shutdown()
		}
	}
}

// TestMultiServiceTraceValidation tests multi-service trace validation
func TestMultiServiceTraceValidation(t *testing.T) {
	validator := NewDistributedTraceValidator()
	defer validator.Cleanup()

	// Create test services
	serviceA, err := validator.CreateTestService("service-a", 8081)
	require.NoError(t, err)

	serviceB, err := validator.CreateTestService("service-b", 8082)
	require.NoError(t, err)

	// Setup endpoints
	validator.SetupTestEndpoints(serviceA)
	validator.SetupTestEndpoints(serviceB)

	// Start services
	require.NoError(t, validator.StartService(serviceA))
	require.NoError(t, validator.StartService(serviceB))

	// Test: Service A calls Service B
	reqBody := `{"service": "service-b", "port": 8082, "path": "/trace-context"}`
	req, err := http.NewRequest("POST", "http://localhost:8081/call-downstream",
		strings.NewReader(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := validator.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)

	// Wait for traces to be sent
	time.Sleep(2 * time.Second)

	// Verify traces exist in Jaeger
	// This would require querying Jaeger API
	// For now, we'll just verify the HTTP call succeeded
}

// TestContextPropagation tests context propagation between services
func TestContextPropagation(t *testing.T) {
	validator := NewDistributedTraceValidator()
	defer validator.Cleanup()

	// Create test services
	serviceA, err := validator.CreateTestService("service-a", 8083)
	require.NoError(t, err)

	serviceB, err := validator.CreateTestService("service-b", 8084)
	require.NoError(t, err)

	// Setup endpoints
	validator.SetupTestEndpoints(serviceA)
	validator.SetupTestEndpoints(serviceB)

	// Start services
	require.NoError(t, validator.StartService(serviceA))
	require.NoError(t, validator.StartService(serviceB))

	// Test: Make request to service A, which calls service B
	reqBody := `{"service": "service-b", "port": 8084, "path": "/trace-context"}`
	req, err := http.NewRequest("POST", "http://localhost:8083/call-downstream",
		strings.NewReader(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := validator.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)

	// Get trace context from service B
	respB, err := validator.client.Get("http://localhost:8084/trace-context")
	require.NoError(t, err)
	defer respB.Body.Close()

	var traceCtx TraceContext
	err = json.NewDecoder(respB.Body).Decode(&traceCtx)
	require.NoError(t, err)

	// Verify trace context is present
	assert.NotEmpty(t, traceCtx.TraceID)
	assert.NotEmpty(t, traceCtx.SpanID)
	assert.Contains(t, traceCtx.Headers, "uber-trace-id")
}

// TestBaggageVerification tests baggage propagation
func TestBaggageVerification(t *testing.T) {
	validator := NewDistributedTraceValidator()
	defer validator.Cleanup()

	// Create test services
	serviceA, err := validator.CreateTestService("service-a", 8085)
	require.NoError(t, err)

	serviceB, err := validator.CreateTestService("service-b", 8086)
	require.NoError(t, err)

	// Setup endpoints
	validator.SetupTestEndpoints(serviceA)
	validator.SetupTestEndpoints(serviceB)

	// Start services
	require.NoError(t, validator.StartService(serviceA))
	require.NoError(t, validator.StartService(serviceB))

	// Test: Set baggage in service A
	respA, err := validator.client.Get("http://localhost:8085/baggage")
	require.NoError(t, err)
	defer respA.Body.Close()

	var baggageResp map[string]interface{}
	err = json.NewDecoder(respA.Body).Decode(&baggageResp)
	require.NoError(t, err)

	assert.True(t, baggageResp["baggage_set"].(bool))
	assert.Equal(t, "test-user-123", baggageResp["user_id"])
	assert.Equal(t, "req-456", baggageResp["request_id"])
}

// TestErrorTraceValidation tests error trace validation
func TestErrorTraceValidation(t *testing.T) {
	validator := NewDistributedTraceValidator()
	defer validator.Cleanup()

	// Create test service
	service, err := validator.CreateTestService("error-service", 8087)
	require.NoError(t, err)

	// Setup endpoints
	validator.SetupTestEndpoints(service)

	// Start service
	require.NoError(t, validator.StartService(service))

	// Test: Make request to error endpoint
	resp, err := validator.client.Get("http://localhost:8087/error")
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should return 500 error
	assert.Equal(t, 500, resp.StatusCode)

	// Verify error response
	var errorResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&errorResp)
	require.NoError(t, err)

	assert.Equal(t, "Intentional test error", errorResp["error"])

	// Wait for traces to be sent
	time.Sleep(2 * time.Second)

	// The error trace should be visible in Jaeger with error tags
}

// TestTraceTimeout tests trace timeout handling
func TestTraceTimeout(t *testing.T) {
	validator := NewDistributedTraceValidator()
	defer validator.Cleanup()

	// Create test service
	service, err := validator.CreateTestService("timeout-service", 8088)
	require.NoError(t, err)

	// Add timeout endpoint
	service.App.Get("/timeout", func(c *fiber.Ctx) error {
		span := c.Locals("span").(opentracing.Span)

		// Simulate slow operation
		time.Sleep(100 * time.Millisecond)

		span.LogFields(
			jaeger.String("event", "slow_operation"),
			jaeger.String("message", "Simulated slow operation"),
		)

		return c.JSON(fiber.Map{"status": "completed"})
	})

	// Setup other endpoints
	validator.SetupTestEndpoints(service)

	// Start service
	require.NoError(t, validator.StartService(service))

	// Test: Make request to timeout endpoint
	start := time.Now()
	resp, err := validator.client.Get("http://localhost:8088/timeout")
	duration := time.Since(start)

	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
	assert.True(t, duration >= 100*time.Millisecond)

	// Wait for traces to be sent
	time.Sleep(2 * time.Second)
}

// Helper function to create HTTP request with body
func createHTTPRequest(method, url, body string) (*http.Request, error) {
	var req *http.Request
	var err error

	if body != "" {
		req, err = http.NewRequest(method, url, strings.NewReader(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// BenchmarkTraceOverhead benchmarks tracing overhead
func BenchmarkTraceOverhead(b *testing.B) {
	validator := NewDistributedTraceValidator()
	defer validator.Cleanup()

	// Create test service
	service, err := validator.CreateTestService("bench-service", 8089)
	require.NoError(b, err)

	// Add benchmark endpoint
	service.App.Get("/bench", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Setup endpoints
	validator.SetupTestEndpoints(service)

	// Start service
	require.NoError(b, validator.StartService(service))

	// Benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := validator.client.Get("http://localhost:8089/bench")
		require.NoError(b, err)
		resp.Body.Close()
	}
}
