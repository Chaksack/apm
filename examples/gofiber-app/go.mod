module github.com/ybke/apm/examples/gofiber-app

go 1.23.0

toolchain go1.24.4

require (
	github.com/gofiber/contrib/otelfiber v1.0.10
	github.com/gofiber/fiber/v2 v2.52.0
	github.com/prometheus/client_golang v1.18.0
	github.com/sony/gobreaker v0.5.0
	github.com/ybke/apm v0.0.0-replace
	go.opentelemetry.io/otel v1.37.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.37.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.37.0
	go.opentelemetry.io/otel/sdk v1.37.0
	go.opentelemetry.io/otel/trace v1.37.0
	go.uber.org/zap v1.26.0
	google.golang.org/grpc v1.73.0
)

// Use local APM module
replace github.com/ybke/apm => ../..
