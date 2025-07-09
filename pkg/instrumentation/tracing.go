package instrumentation

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// TracerConfig holds configuration for the tracer
type TracerConfig struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	ExporterType   string // "otlp", "jaeger", or "stdout"
	Endpoint       string
	SampleRate     float64
}

// InitTracer initializes the OpenTelemetry tracer with the specified configuration
func InitTracer(ctx context.Context, config TracerConfig) (trace.TracerProvider, func(), error) {
	// Create resource
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(config.ServiceVersion),
			semconv.DeploymentEnvironmentKey.String(config.Environment),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create exporter based on configuration
	var exporter sdktrace.SpanExporter
	switch config.ExporterType {
	case "otlp":
		exporter, err = createOTLPExporter(ctx, config.Endpoint)
	case "jaeger":
		exporter, err = createJaegerExporter(config.Endpoint)
	default:
		return nil, nil, fmt.Errorf("unsupported exporter type: %s", config.ExporterType)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// Create sampler
	sampler := sdktrace.TraceIDRatioBased(config.SampleRate)

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Return cleanup function
	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			otel.Handle(err)
		}
	}

	return tp, cleanup, nil
}

// createOTLPExporter creates an OTLP exporter
func createOTLPExporter(ctx context.Context, endpoint string) (sdktrace.SpanExporter, error) {
	client := otlptracegrpc.NewClient(
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	return otlptrace.New(ctx, client)
}

// createJaegerExporter creates a Jaeger exporter
func createJaegerExporter(endpoint string) (sdktrace.SpanExporter, error) {
	return jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(endpoint)))
}

// GetTracer returns a tracer with the specified name
func GetTracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
