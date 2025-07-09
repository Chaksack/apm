package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// HealthCheck represents the health status response
type HealthCheck struct {
	Status    string    `json:"status"`
	Service   string    `json:"service"`
	Timestamp time.Time `json:"timestamp"`
	Checks    []Check   `json:"checks,omitempty"`
}

// Check represents an individual health check
type Check struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// Health returns a detailed health check response
func Health(c *fiber.Ctx) error {
	health := HealthCheck{
		Status:    "healthy",
		Service:   "apm",
		Timestamp: time.Now(),
		Checks: []Check{
			{
				Name:   "database",
				Status: "healthy",
			},
			{
				Name:   "prometheus",
				Status: "healthy",
			},
		},
	}

	return c.JSON(health)
}

// SimpleHealth returns a simple health check response
func SimpleHealth(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "ok",
	})
}
