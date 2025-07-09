package handlers

import (
	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics returns Prometheus metrics
func Metrics(c *fiber.Ctx) error {
	// Use the Prometheus HTTP handler with Fiber adaptor
	handler := adaptor.HTTPHandler(promhttp.Handler())
	return handler(c)
}
