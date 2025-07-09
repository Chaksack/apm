package instrumentation

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/trace"
)

// contextKey is a type for context keys
type contextKey string

const (
	// CorrelationIDKey is the context key for correlation ID
	CorrelationIDKey contextKey = "correlation-id"
)

// WithCorrelationID adds a correlation ID to the context
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, CorrelationIDKey, correlationID)
}

// GetCorrelationID retrieves the correlation ID from context
func GetCorrelationID(ctx context.Context) string {
	if id, ok := ctx.Value(CorrelationIDKey).(string); ok {
		return id
	}
	return ""
}

// GenerateCorrelationID generates a new correlation ID
func GenerateCorrelationID() string {
	return uuid.New().String()
}

// InjectCorrelationID injects correlation ID into trace context and baggage
func InjectCorrelationID(ctx context.Context, correlationID string) context.Context {
	// Add to context
	ctx = WithCorrelationID(ctx, correlationID)

	// Add to current span
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.SetAttributes(
			attribute.String("correlation.id", correlationID),
		)
	}

	// Add to baggage for propagation
	member, _ := baggage.NewMember("correlation-id", correlationID)
	bag, _ := baggage.New(member)
	ctx = baggage.ContextWithBaggage(ctx, bag)

	return ctx
}

// ExtractCorrelationID extracts correlation ID from context or generates a new one
func ExtractCorrelationID(ctx context.Context) (string, context.Context) {
	// Try to get from baggage first
	bag := baggage.FromContext(ctx)
	if member := bag.Member("correlation-id"); member.Key() != "" {
		correlationID := member.Value()
		return correlationID, InjectCorrelationID(ctx, correlationID)
	}

	// Try to get from context
	if correlationID := GetCorrelationID(ctx); correlationID != "" {
		return correlationID, ctx
	}

	// Generate new one
	correlationID := GenerateCorrelationID()
	return correlationID, InjectCorrelationID(ctx, correlationID)
}

// StartSpanWithCorrelation starts a new span with correlation ID
func StartSpanWithCorrelation(ctx context.Context, tracer trace.Tracer, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	// Ensure correlation ID exists
	correlationID, ctx := ExtractCorrelationID(ctx)

	// Start span
	ctx, span := tracer.Start(ctx, spanName, opts...)

	// Add correlation ID to span
	span.SetAttributes(attribute.String("correlation.id", correlationID))

	return ctx, span
}

// FiberContextWithCorrelation creates a context with correlation ID for Fiber
func FiberContextWithCorrelation(c *fiber.Ctx) context.Context {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = c.Context()
	}

	// Check for correlation ID in headers
	correlationID := c.Get("X-Correlation-ID")
	if correlationID == "" {
		correlationID = c.Get("X-Request-ID")
	}
	if correlationID == "" {
		correlationID = GenerateCorrelationID()
	}

	// Set correlation ID in response header
	c.Set("X-Correlation-ID", correlationID)

	return InjectCorrelationID(ctx, correlationID)
}

// GetBaggageValue retrieves a baggage value from context
func GetBaggageValue(ctx context.Context, key string) string {
	bag := baggage.FromContext(ctx)
	if member := bag.Member(key); member.Key() != "" {
		return member.Value()
	}
	return ""
}

// SetBaggageValue sets a baggage value in context
func SetBaggageValue(ctx context.Context, key, value string) context.Context {
	member, err := baggage.NewMember(key, value)
	if err != nil {
		return ctx
	}

	bag := baggage.FromContext(ctx)
	bag, _ = bag.SetMember(member)

	return baggage.ContextWithBaggage(ctx, bag)
}
