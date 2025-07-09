package instrumentation

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"
)

// ExporterConfig holds configuration for exporters
type ExporterConfig struct {
	Type     string            // "otlp-grpc", "otlp-http", "jaeger", "stdout", "multi"
	Endpoint string            // Endpoint for the exporter
	Headers  map[string]string // Headers for OTLP exporters
	Insecure bool              // Use insecure connection
	// For stdout exporter
	Writer io.Writer
	// For multi-exporter
	Exporters []ExporterConfig
	// Batch processor configuration
	BatchTimeout   int // Milliseconds
	MaxExportBatch int
	MaxQueueSize   int
}

// DefaultBatchProcessorOptions returns default batch processor options
func DefaultBatchProcessorOptions() []trace.BatchSpanProcessorOption {
	return []trace.BatchSpanProcessorOption{
		trace.WithMaxExportBatchSize(512),
		trace.WithMaxQueueSize(2048),
		trace.WithBatchTimeout(5000), // 5 seconds
	}
}

// CreateExporter creates a span exporter based on the configuration
func CreateExporter(ctx context.Context, config ExporterConfig) (trace.SpanExporter, error) {
	switch config.Type {
	case "otlp-grpc":
		return createOTLPGRPCExporter(ctx, config)
	case "otlp-http":
		return createOTLPHTTPExporter(ctx, config)
	case "jaeger":
		return createJaegerExporterFromConfig(config)
	case "stdout":
		return createStdoutExporter(config)
	case "multi":
		return createMultiExporter(ctx, config)
	default:
		return nil, fmt.Errorf("unknown exporter type: %s", config.Type)
	}
}

// createOTLPGRPCExporter creates an OTLP gRPC exporter
func createOTLPGRPCExporter(ctx context.Context, config ExporterConfig) (trace.SpanExporter, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(config.Endpoint),
	}

	if config.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	if len(config.Headers) > 0 {
		opts = append(opts, otlptracegrpc.WithHeaders(config.Headers))
	}

	client := otlptracegrpc.NewClient(opts...)
	return otlptrace.New(ctx, client)
}

// createOTLPHTTPExporter creates an OTLP HTTP exporter
func createOTLPHTTPExporter(ctx context.Context, config ExporterConfig) (trace.SpanExporter, error) {
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(config.Endpoint),
	}

	if config.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	if len(config.Headers) > 0 {
		opts = append(opts, otlptracehttp.WithHeaders(config.Headers))
	}

	client := otlptracehttp.NewClient(opts...)
	return otlptrace.New(ctx, client)
}

// createJaegerExporterFromConfig creates a Jaeger exporter from config
func createJaegerExporterFromConfig(config ExporterConfig) (trace.SpanExporter, error) {
	return jaeger.New(jaeger.WithCollectorEndpoint(
		jaeger.WithEndpoint(config.Endpoint),
	))
}

// createStdoutExporter creates a stdout exporter
func createStdoutExporter(config ExporterConfig) (trace.SpanExporter, error) {
	writer := config.Writer
	if writer == nil {
		writer = os.Stdout
	}

	return stdouttrace.New(
		stdouttrace.WithWriter(writer),
		stdouttrace.WithPrettyPrint(),
	)
}

// createMultiExporter creates a multi-exporter that exports to multiple destinations
func createMultiExporter(ctx context.Context, config ExporterConfig) (trace.SpanExporter, error) {
	var exporters []trace.SpanExporter

	for _, expConfig := range config.Exporters {
		exp, err := CreateExporter(ctx, expConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create exporter %s: %w", expConfig.Type, err)
		}
		exporters = append(exporters, exp)
	}

	return &multiExporter{exporters: exporters}, nil
}

// multiExporter implements trace.SpanExporter for multiple exporters
type multiExporter struct {
	exporters []trace.SpanExporter
}

// ExportSpans exports spans to all configured exporters
func (m *multiExporter) ExportSpans(ctx context.Context, spans []trace.ReadOnlySpan) error {
	var errs []error
	for _, exp := range m.exporters {
		if err := exp.ExportSpans(ctx, spans); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("export errors: %v", errs)
	}
	return nil
}

// Shutdown shuts down all exporters
func (m *multiExporter) Shutdown(ctx context.Context) error {
	var errs []error
	for _, exp := range m.exporters {
		if err := exp.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}
	return nil
}

// CreateBatchProcessor creates a batch span processor with the given exporter
func CreateBatchProcessor(exporter trace.SpanExporter, config ExporterConfig) trace.SpanProcessor {
	opts := DefaultBatchProcessorOptions()

	// Override with custom values if provided
	if config.BatchTimeout > 0 {
		opts = append(opts, trace.WithBatchTimeout(time.Duration(config.BatchTimeout)*time.Millisecond))
	}
	if config.MaxExportBatch > 0 {
		opts = append(opts, trace.WithMaxExportBatchSize(config.MaxExportBatch))
	}
	if config.MaxQueueSize > 0 {
		opts = append(opts, trace.WithMaxQueueSize(config.MaxQueueSize))
	}

	return trace.NewBatchSpanProcessor(exporter, opts...)
}
