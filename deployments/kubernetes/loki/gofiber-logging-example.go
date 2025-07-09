package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

// CustomLogger creates a structured JSON logger for GoFiber that works with Promtail
func CustomLogger() fiber.Handler {
	return logger.New(logger.Config{
		Format:     "${custom}",
		TimeFormat: time.RFC3339,
		CustomTags: map[string]logger.LogFunc{
			"custom": func(output logger.Buffer, c *fiber.Ctx, data *logger.Data, extraParam string) (int, error) {
				logEntry := map[string]interface{}{
					"time":       time.Now().Format(time.RFC3339),
					"level":      "info",
					"method":     c.Method(),
					"path":       c.Path(),
					"status":     c.Response().StatusCode(),
					"latency":    data.Stop.Sub(data.Start).String(),
					"ip":         c.IP(),
					"user_agent": c.Get("User-Agent"),
					"request_id": c.Locals("requestid"),
				}

				// Add error if status >= 400
				if c.Response().StatusCode() >= 400 {
					logEntry["level"] = "error"
					if err := c.Locals("error"); err != nil {
						logEntry["error"] = fmt.Sprintf("%v", err)
					}
				}

				// Marshal to JSON
				jsonData, err := json.Marshal(logEntry)
				if err != nil {
					return 0, err
				}

				return output.Write(append(jsonData, '\n'))
			},
		},
		Output: os.Stdout,
	})
}

// ErrorLogger logs errors in structured format
func ErrorLogger(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	logEntry := map[string]interface{}{
		"time":       time.Now().Format(time.RFC3339),
		"level":      "error",
		"method":     c.Method(),
		"path":       c.Path(),
		"status":     code,
		"ip":         c.IP(),
		"user_agent": c.Get("User-Agent"),
		"request_id": c.Locals("requestid"),
		"error":      err.Error(),
		"msg":        "Request failed",
	}

	jsonData, _ := json.Marshal(logEntry)
	fmt.Println(string(jsonData))

	c.Locals("error", err)
	return c.Status(code).JSON(fiber.Map{
		"error": err.Error(),
	})
}

// Example usage
func main() {
	app := fiber.New(fiber.Config{
		ErrorHandler: ErrorLogger,
	})

	// Add request ID middleware
	app.Use(requestid.New())

	// Add custom logger
	app.Use(CustomLogger())

	// Example routes
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Hello, World!",
		})
	})

	app.Get("/error", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusBadRequest, "This is a test error")
	})

	// Start server
	app.Listen(":3000")
}

// Additional structured logging helper
type StructuredLogger struct {
	Level string                 `json:"level"`
	Time  string                 `json:"time"`
	Msg   string                 `json:"msg"`
	Data  map[string]interface{} `json:",inline"`
}

func LogStructured(level, msg string, data map[string]interface{}) {
	entry := StructuredLogger{
		Level: level,
		Time:  time.Now().Format(time.RFC3339),
		Msg:   msg,
		Data:  data,
	}
	jsonData, _ := json.Marshal(entry)
	fmt.Println(string(jsonData))
}

// Example usage:
// LogStructured("info", "User logged in", map[string]interface{}{
//     "user_id": "123",
//     "session": "abc-def-ghi",
// })
