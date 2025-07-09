package instrumentation

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// FiberOtelMiddleware creates a Fiber middleware for OpenTelemetry tracing
func FiberOtelMiddleware(serviceName string) fiber.Handler {
	tracer := otel.Tracer(serviceName)
	propagator := otel.GetTextMapPropagator()

	return func(c *fiber.Ctx) error {
		// Extract trace context from incoming request
		ctx := propagator.Extract(c.Context(), propagation.HeaderCarrier(c.GetReqHeaders()))

		// Start span
		spanName := fmt.Sprintf("%s %s", c.Method(), c.Path())
		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(extractSpanAttributes(c)...),
		)
		defer span.End()

		// Store context in Fiber locals
		c.SetUserContext(ctx)

		// Process request
		err := c.Next()

		// Set span status based on response
		statusCode := c.Response().StatusCode()
		span.SetAttributes(semconv.HTTPStatusCode(statusCode))

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else if statusCode >= 400 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", statusCode))
		} else {
			span.SetStatus(codes.Ok, "")
		}

		return err
	}
}

// extractSpanAttributes extracts relevant attributes from the Fiber context
func extractSpanAttributes(c *fiber.Ctx) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		semconv.HTTPMethodKey.String(c.Method()),
		semconv.HTTPTargetKey.String(c.OriginalURL()),
		semconv.HTTPRouteKey.String(c.Path()),
		semconv.HTTPSchemeKey.String(c.Protocol()),
		semconv.NetHostNameKey.String(c.Hostname()),
		semconv.HTTPUserAgentKey.String(c.Get("User-Agent")),
		semconv.HTTPRequestContentLengthKey.Int(len(c.Body())),
	}

	// Add client IP if available
	if clientIP := c.IP(); clientIP != "" {
		attrs = append(attrs, semconv.NetSockPeerAddrKey.String(clientIP))
	}

	// Add request ID if available
	if requestID := c.Get("X-Request-ID"); requestID != "" {
		attrs = append(attrs, attribute.String("http.request_id", requestID))
	}

	return attrs
}

// PropagateContext is a helper to propagate trace context in outgoing requests
func PropagateContext(c *fiber.Ctx, headers map[string]string) {
	ctx := c.UserContext()
	if ctx == nil {
		return
	}

	propagator := otel.GetTextMapPropagator()
	propagator.Inject(ctx, propagation.MapCarrier(headers))
}

// GetSpanFromContext retrieves the current span from Fiber context
func GetSpanFromContext(c *fiber.Ctx) trace.Span {
	ctx := c.UserContext()
	if ctx == nil {
		return trace.SpanFromContext(c.Context())
	}
	return trace.SpanFromContext(ctx)
}
