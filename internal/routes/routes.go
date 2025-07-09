package routes

import (
	"github.com/chaksack/apm/internal/handlers"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// SetupRoutes configures all application routes
func SetupRoutes(app *fiber.App) {
	// Middleware
	app.Use(logger.New())
	app.Use(recover.New())

	// Health check endpoint
	app.Get("/health", handlers.Health)

	// Prometheus metrics endpoint
	app.Get("/metrics", handlers.Metrics)

	// API v1 routes
	api := app.Group("/api/v1")
	api.Get("/status", handlers.Status)
}
