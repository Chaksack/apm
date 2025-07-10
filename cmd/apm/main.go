package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// startTime is used to calculate uptime
var startTime = time.Now()

func main() {
	// Create structured logger
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.LUTC)
	log.SetPrefix("[APM] ")

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:               "APM Service",
		DisableStartupMessage: false,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}

			log.Printf("ERROR: status=%d method=%s path=%s error=%v",
				code, c.Method(), c.Path(), err)

			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Middleware
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))

	// Structured logging middleware
	app.Use(logger.New(logger.Config{
		Format:     "[${time}] ${status} - ${latency} ${method} ${path}\n",
		TimeFormat: "2006-01-02 15:04:05",
		TimeZone:   "UTC",
		Output:     os.Stdout,
	}))

	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "healthy",
			"time":   time.Now().UTC(),
		})
	})

	// Prometheus metrics endpoint
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	// Root endpoint
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"service": "APM",
			"version": "1.0.0",
		})
	})

	// API v1 routes
	api := app.Group("/api/v1")

	// Status endpoint
	api.Get("/status", func(c *fiber.Ctx) error {
		uptime := time.Since(startTime).Round(time.Second).String()

		status := fiber.Map{
			"status":  "operational",
			"version": "1.0.0",
			"uptime":  uptime,
			"components": fiber.Map{
				"prometheus":   "healthy",
				"grafana":      "healthy",
				"alertmanager": "healthy",
				"loki":         "healthy",
				"promtail":     "healthy",
			},
			"metadata": fiber.Map{
				"timestamp": time.Now().Unix(),
				"timezone":  time.Local.String(),
			},
		}

		return c.JSON(status)
	})

	// Tool configurations
	toolConfigs := map[string]struct {
		Host string
		Port int
		Path string
	}{
		"prometheus": {
			Host: "localhost",
			Port: 9090,
			Path: "/",
		},
		"grafana": {
			Host: "localhost",
			Port: 3000,
			Path: "/",
		},
		"jaeger": {
			Host: "localhost",
			Port: 16686,
			Path: "/",
		},
		"loki": {
			Host: "localhost",
			Port: 3100,
			Path: "/",
		},
		"alertmanager": {
			Host: "localhost",
			Port: 9093,
			Path: "/",
		},
		"cadvisor": {
			Host: "localhost",
			Port: 8090,
			Path: "/",
		},
		"node-exporter": {
			Host: "localhost",
			Port: 9100,
			Path: "/metrics",
		},
	}

	// Tools routes
	tools := app.Group("/tools")

	// List all tools
	tools.Get("/", func(c *fiber.Ctx) error {
		toolsList := make([]map[string]interface{}, 0, len(toolConfigs))

		for name, config := range toolConfigs {
			toolsList = append(toolsList, map[string]interface{}{
				"name": name,
				"url":  fmt.Sprintf("http://%s:%d%s", config.Host, config.Port, config.Path),
			})
		}

		return c.JSON(fiber.Map{
			"tools": toolsList,
		})
	})

	// Redirect to specific tool
	tools.Get("/:tool", func(c *fiber.Ctx) error {
		toolName := c.Params("tool")

		config, exists := toolConfigs[toolName]
		if !exists {
			return c.Status(404).JSON(fiber.Map{
				"error": fmt.Sprintf("Tool '%s' not found", toolName),
			})
		}

		redirectURL := fmt.Sprintf("http://%s:%d%s", config.Host, config.Port, config.Path)
		return c.Redirect(redirectURL, fiber.StatusTemporaryRedirect)
	})

	// Start server in a goroutine
	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}

		log.Printf("Starting server on port %s", port)
		if err := app.Listen(":" + port); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
