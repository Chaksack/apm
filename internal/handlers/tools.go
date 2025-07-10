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

package handlers

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
)

// ToolRedirectConfig holds the configuration for tool redirects
type ToolRedirectConfig struct {
	Host string
	Port int
	Path string
}

// Default tool configurations
var toolConfigs = map[string]ToolRedirectConfig{
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

// RedirectToTool redirects to the specified monitoring tool
func RedirectToTool(c *fiber.Ctx) error {
	toolName := c.Params("tool")
	
	config, exists := toolConfigs[toolName]
	if !exists {
		return c.Status(404).JSON(fiber.Map{
			"error": fmt.Sprintf("Tool '%s' not found", toolName),
		})
	}
	
	redirectURL := fmt.Sprintf("http://%s:%d%s", config.Host, config.Port, config.Path)
	return c.Redirect(redirectURL, fiber.StatusTemporaryRedirect)
}

// ListTools returns a list of available monitoring tools
func ListTools(c *fiber.Ctx) error {
	tools := make([]map[string]interface{}, 0, len(toolConfigs))
	
	for name, config := range toolConfigs {
		tools = append(tools, map[string]interface{}{
			"name": name,
			"url":  fmt.Sprintf("http://%s:%d%s", config.Host, config.Port, config.Path),
		})
	}
	
	return c.JSON(fiber.Map{
		"tools": tools,
	})
}