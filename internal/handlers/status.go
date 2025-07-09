package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// startTime is used to calculate uptime
var startTime = time.Now()

// SystemStatus represents the APM system status
type SystemStatus struct {
	Status     string                 `json:"status"`
	Version    string                 `json:"version"`
	Uptime     string                 `json:"uptime"`
	Components map[string]string      `json:"components"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// Status returns the APM system status
func Status(c *fiber.Ctx) error {
	uptime := time.Since(startTime).Round(time.Second).String()

	status := SystemStatus{
		Status:  "operational",
		Version: "1.0.0",
		Uptime:  uptime,
		Components: map[string]string{
			"prometheus":   "healthy",
			"grafana":      "healthy",
			"alertmanager": "healthy",
			"loki":         "healthy",
			"promtail":     "healthy",
		},
		Metadata: map[string]interface{}{
			"timestamp": time.Now().Unix(),
			"timezone":  time.Local.String(),
		},
	}

	return c.JSON(status)
}
