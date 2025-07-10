// Copyright (c) 2024 APM Solution Contributors
// Authors: Andrew Chakdahah (chakdahah@gmail.com) and Yaw Boateng Kessie (ybkess@gmail.com)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package routes

import (
	"github.com/chaksack/apm/internal/handlers"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// SetupRoutes configures all application routes
func SetupRoutes(app *fiber.App) error {
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

	// Create tool handlers
	toolHandlers, err := handlers.NewToolHandlers()
	if err != nil {
		return err
	}

	// Tools routes
	tools := app.Group("/tools")
	tools.Get("/", toolHandlers.ListTools)
	tools.Get("/detect", toolHandlers.DetectTools)
	tools.Get("/ports", toolHandlers.GetAllocatedPorts)
	tools.Get("/port-registry", toolHandlers.GetPortRegistry)
	tools.Post("/allocate-port", toolHandlers.AllocatePort)
	tools.Get("/:tool", toolHandlers.RedirectToTool)
	tools.Get("/:tool/health", toolHandlers.GetToolHealth)
	tools.Get("/:tool/config", toolHandlers.GetToolConfig)
	tools.Post("/:tool/config", toolHandlers.GetToolConfig)

	return nil
}
