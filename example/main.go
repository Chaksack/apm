package main

import (
	"log"
	"time"

	"github.com/chaksack/apm/pkg/instrumentation"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"go.uber.org/zap"
)

func main() {
	// Load configuration from environment
	cfg := instrumentation.LoadFromEnv()

	// Initialize instrumentation
	inst, err := instrumentation.New(cfg)
	if err != nil {
		log.Fatalf("failed to initialize instrumentation: %v", err)
	}

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:               cfg.ServiceName,
		DisableStartupMessage: true,
	})

	// Add request ID middleware
	app.Use(requestid.New())

	// Add instrumentation middleware
	app.Use(inst.FiberMiddleware())

	// Add logging middleware
	app.Use(instrumentation.LoggerMiddleware(inst.Logger))

	// Prometheus metrics endpoint
	if cfg.Metrics.Enabled {
		app.Get(cfg.Metrics.Path, func(c *fiber.Ctx) error {
			fasthttpadaptor.NewFastHTTPHandler(promhttp.Handler())(c.Context())
			return nil
		})
	}

	// Example endpoints
	app.Get("/", func(c *fiber.Ctx) error {
		logger := instrumentation.GetLogger(c)
		logger.Info("handling root request")

		return c.JSON(fiber.Map{
			"service":     cfg.ServiceName,
			"version":     cfg.Version,
			"environment": cfg.Environment,
		})
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "healthy",
			"time":   time.Now().Unix(),
		})
	})

	app.Post("/api/users", func(c *fiber.Ctx) error {
		logger := instrumentation.GetLogger(c)

		// Simulate some work
		time.Sleep(50 * time.Millisecond)

		logger.Info("created new user", zap.String("user_id", "123"))

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"id":      "123",
			"message": "user created",
		})
	})

	app.Get("/api/users/:id", func(c *fiber.Ctx) error {
		logger := instrumentation.GetLogger(c)
		userID := c.Params("id")

		if userID == "404" {
			logger.Warn("user not found", zap.String("user_id", userID))
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "user not found",
			})
		}

		logger.Info("fetched user", zap.String("user_id", userID))

		return c.JSON(fiber.Map{
			"id":   userID,
			"name": "John Doe",
		})
	})

	app.Get("/error", func(c *fiber.Ctx) error {
		logger := instrumentation.GetLogger(c)
		logger.Error("intentional error endpoint hit")

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "something went wrong",
		})
	})

	// Custom metrics example
	requestCounter := inst.Metrics.NewCounter("custom_requests_total", "Total custom requests", []string{"type"})
	processingTime := inst.Metrics.NewHistogram("processing_duration_seconds", "Processing duration", []string{"operation"}, nil)
	activeConnections := inst.Metrics.NewGauge("active_connections", "Number of active connections", []string{"type"})

	app.Post("/process", func(c *fiber.Ctx) error {
		logger := instrumentation.GetLogger(c)
		start := time.Now()

		// Track custom metrics
		requestCounter.WithLabelValues("process").Inc()
		activeConnections.WithLabelValues("websocket").Inc()
		defer activeConnections.WithLabelValues("websocket").Dec()

		// Simulate processing
		time.Sleep(100 * time.Millisecond)

		processingTime.WithLabelValues("data_processing").Observe(time.Since(start).Seconds())

		logger.Info("processed request", zap.Duration("duration", time.Since(start)))

		return c.JSON(fiber.Map{
			"status": "processed",
		})
	})

	// Start server
	go func() {
		inst.Logger.Info("starting server",
			zap.String("address", ":8080"),
			zap.String("service", cfg.ServiceName),
			zap.String("version", cfg.Version),
		)

		if err := app.Listen(":8080"); err != nil {
			inst.Logger.Fatal("failed to start server", zap.Error(err))
		}
	}()

	// Wait for shutdown signal
	inst.WaitForShutdown()
}
