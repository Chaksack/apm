package instrumentation

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// TestCollector is a mock Prometheus collector for testing
type TestCollector struct {
	mu      sync.Mutex
	metrics map[string]float64
	labels  map[string]map[string]string
}

// NewTestCollector creates a new test collector
func NewTestCollector() *TestCollector {
	return &TestCollector{
		metrics: make(map[string]float64),
		labels:  make(map[string]map[string]string),
	}
}

// Inc increments a counter metric
func (tc *TestCollector) Inc(name string, labels map[string]string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	key := tc.makeKey(name, labels)
	tc.metrics[key]++
	tc.labels[key] = labels
}

// Add adds a value to a metric
func (tc *TestCollector) Add(name string, value float64, labels map[string]string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	key := tc.makeKey(name, labels)
	tc.metrics[key] += value
	tc.labels[key] = labels
}

// Set sets a gauge metric value
func (tc *TestCollector) Set(name string, value float64, labels map[string]string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	key := tc.makeKey(name, labels)
	tc.metrics[key] = value
	tc.labels[key] = labels
}

// Get returns the current value of a metric
func (tc *TestCollector) Get(name string, labels map[string]string) float64 {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	key := tc.makeKey(name, labels)
	return tc.metrics[key]
}

// Reset clears all metrics
func (tc *TestCollector) Reset() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	tc.metrics = make(map[string]float64)
	tc.labels = make(map[string]map[string]string)
}

// makeKey creates a unique key for a metric with labels
func (tc *TestCollector) makeKey(name string, labels map[string]string) string {
	if len(labels) == 0 {
		return name
	}

	var parts []string
	for k, v := range labels {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return fmt.Sprintf("%s{%s}", name, strings.Join(parts, ","))
}

// AssertMetric checks if a metric has the expected value
func (tc *TestCollector) AssertMetric(t *testing.T, name string, expected float64, labels map[string]string) {
	t.Helper()

	actual := tc.Get(name, labels)
	if actual != expected {
		t.Errorf("metric %s expected %f, got %f", name, expected, actual)
	}
}

// TestTrace represents a captured trace for testing
type TestTrace struct {
	TraceID    string
	SpanID     string
	ParentID   string
	Name       string
	StartTime  time.Time
	EndTime    time.Time
	Attributes map[string]interface{}
	Events     []TestEvent
	Status     string
}

// TestEvent represents an event within a trace
type TestEvent struct {
	Name       string
	Timestamp  time.Time
	Attributes map[string]interface{}
}

// TestTracer is a mock tracer for testing
type TestTracer struct {
	mu     sync.Mutex
	traces []TestTrace
}

// NewTestTracer creates a new test tracer
func NewTestTracer() *TestTracer {
	return &TestTracer{
		traces: make([]TestTrace, 0),
	}
}

// StartSpan starts a new test span
func (tt *TestTracer) StartSpan(ctx context.Context, name string) (context.Context, *TestSpan) {
	span := &TestSpan{
		tracer:     tt,
		Name:       name,
		TraceID:    generateTestID(),
		SpanID:     generateTestID(),
		StartTime:  time.Now(),
		Attributes: make(map[string]interface{}),
		Events:     make([]TestEvent, 0),
	}

	// Extract parent from context if available
	if parentSpan := ctx.Value("test_span"); parentSpan != nil {
		if ps, ok := parentSpan.(*TestSpan); ok {
			span.ParentID = ps.SpanID
			span.TraceID = ps.TraceID
		}
	}

	ctx = context.WithValue(ctx, "test_span", span)
	return ctx, span
}

// GetTraces returns all captured traces
func (tt *TestTracer) GetTraces() []TestTrace {
	tt.mu.Lock()
	defer tt.mu.Unlock()

	traces := make([]TestTrace, len(tt.traces))
	copy(traces, tt.traces)
	return traces
}

// Reset clears all traces
func (tt *TestTracer) Reset() {
	tt.mu.Lock()
	defer tt.mu.Unlock()
	tt.traces = make([]TestTrace, 0)
}

// AssertSpanExists checks if a span with the given name exists
func (tt *TestTracer) AssertSpanExists(t *testing.T, spanName string) {
	t.Helper()

	traces := tt.GetTraces()
	for _, trace := range traces {
		if trace.Name == spanName {
			return
		}
	}

	t.Errorf("span %s not found", spanName)
}

// AssertSpanAttribute checks if a span has an attribute with the expected value
func (tt *TestTracer) AssertSpanAttribute(t *testing.T, spanName, attrName string, expected interface{}) {
	t.Helper()

	traces := tt.GetTraces()
	for _, trace := range traces {
		if trace.Name == spanName {
			if actual, ok := trace.Attributes[attrName]; ok {
				if actual != expected {
					t.Errorf("span %s attribute %s expected %v, got %v", spanName, attrName, expected, actual)
				}
				return
			}
			t.Errorf("span %s does not have attribute %s", spanName, attrName)
			return
		}
	}

	t.Errorf("span %s not found", spanName)
}

// TestSpan represents a span in testing
type TestSpan struct {
	tracer     *TestTracer
	TraceID    string
	SpanID     string
	ParentID   string
	Name       string
	StartTime  time.Time
	EndTime    time.Time
	Attributes map[string]interface{}
	Events     []TestEvent
	Status     string
}

// End ends the span and records it
func (ts *TestSpan) End() {
	ts.EndTime = time.Now()

	ts.tracer.mu.Lock()
	defer ts.tracer.mu.Unlock()

	ts.tracer.traces = append(ts.tracer.traces, TestTrace{
		TraceID:    ts.TraceID,
		SpanID:     ts.SpanID,
		ParentID:   ts.ParentID,
		Name:       ts.Name,
		StartTime:  ts.StartTime,
		EndTime:    ts.EndTime,
		Attributes: ts.Attributes,
		Events:     ts.Events,
		Status:     ts.Status,
	})
}

// SetAttribute sets an attribute on the span
func (ts *TestSpan) SetAttribute(key string, value interface{}) {
	ts.Attributes[key] = value
}

// AddEvent adds an event to the span
func (ts *TestSpan) AddEvent(name string, attributes map[string]interface{}) {
	ts.Events = append(ts.Events, TestEvent{
		Name:       name,
		Timestamp:  time.Now(),
		Attributes: attributes,
	})
}

// SetStatus sets the span status
func (ts *TestSpan) SetStatus(status string) {
	ts.Status = status
}

// generateTestID generates a test ID
func generateTestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// TestLogger is a mock logger for testing
type TestLogger struct {
	mu      sync.Mutex
	entries []LogEntry
}

// LogEntry represents a log entry in tests
type LogEntry struct {
	Level     string
	Message   string
	Fields    map[string]interface{}
	Timestamp time.Time
}

// NewTestLogger creates a new test logger
func NewTestLogger() *TestLogger {
	return &TestLogger{
		entries: make([]LogEntry, 0),
	}
}

// Log logs a message
func (tl *TestLogger) Log(level, message string, fields map[string]interface{}) {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	tl.entries = append(tl.entries, LogEntry{
		Level:     level,
		Message:   message,
		Fields:    fields,
		Timestamp: time.Now(),
	})
}

// GetEntries returns all log entries
func (tl *TestLogger) GetEntries() []LogEntry {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	entries := make([]LogEntry, len(tl.entries))
	copy(entries, tl.entries)
	return entries
}

// Reset clears all log entries
func (tl *TestLogger) Reset() {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	tl.entries = make([]LogEntry, 0)
}

// AssertLogContains checks if a log entry contains the expected message
func (tl *TestLogger) AssertLogContains(t *testing.T, level, message string) {
	t.Helper()

	entries := tl.GetEntries()
	for _, entry := range entries {
		if entry.Level == level && strings.Contains(entry.Message, message) {
			return
		}
	}

	t.Errorf("log entry with level %s and message containing %s not found", level, message)
}

// TestHelpers provides utility functions for testing instrumented code
type TestHelpers struct{}

// NewTestApp creates a test Fiber app with instrumentation
func (th *TestHelpers) NewTestApp() *fiber.App {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	// Add basic middleware
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("request_id", generateTestID())
		return c.Next()
	})

	return app
}

// AssertMetricsEqual compares expected and actual Prometheus metrics
func (th *TestHelpers) AssertMetricsEqual(t *testing.T, expected, actual string) {
	t.Helper()

	expectedLines := strings.Split(strings.TrimSpace(expected), "\n")
	actualLines := strings.Split(strings.TrimSpace(actual), "\n")

	if len(expectedLines) != len(actualLines) {
		t.Errorf("expected %d lines, got %d", len(expectedLines), len(actualLines))
		return
	}

	for i, expectedLine := range expectedLines {
		if i >= len(actualLines) {
			t.Errorf("missing line %d: %s", i, expectedLine)
			continue
		}

		if expectedLine != actualLines[i] {
			t.Errorf("line %d mismatch:\nexpected: %s\nactual:   %s", i, expectedLine, actualLines[i])
		}
	}
}

// CollectMetrics collects metrics from a Prometheus collector
func (th *TestHelpers) CollectMetrics(collector prometheus.Collector) (string, error) {
	var buf bytes.Buffer
	err := testutil.CollectAndCompare(collector, &buf)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// MockHTTPRequest creates a mock HTTP request for testing
// Note: This is a simplified version that doesn't use actual fiber.Ctx
// It returns a mock context that can be used for basic testing
func (th *TestHelpers) MockHTTPRequest(method, path string, body []byte, headers map[string]string) *fiber.Ctx {
	// Create a new app for testing
	app := fiber.New()

	// Add a test route
	app.Add(method, path, func(c *fiber.Ctx) error {
		// This is just to register the route
		return nil
	})

	// Import net/http and httptest if needed for more complex testing

	// For now, return nil as this is just a stub
	// In real usage, you would need to modify the tests that use this function
	// to handle the nil case or implement a proper mock
	return nil
}

// WaitForCondition waits for a condition to be true
func (th *TestHelpers) WaitForCondition(timeout time.Duration, checkFunc func() bool) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		if checkFunc() {
			return nil
		}

		select {
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("condition not met within timeout")
			}
		}
	}
}
