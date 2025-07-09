module github.com/apm/sample-app

go 1.21

require (
	github.com/gofiber/fiber/v2 v2.52.0
	github.com/gofiber/contrib/otelfiber v1.0.10
	github.com/prometheus/client_golang v1.17.0
	go.opentelemetry.io/otel v1.21.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.21.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.21.0
	go.opentelemetry.io/otel/sdk v1.21.0
	go.opentelemetry.io/otel/trace v1.21.0
	go.uber.org/zap v1.26.0
	google.golang.org/grpc v1.60.1
)