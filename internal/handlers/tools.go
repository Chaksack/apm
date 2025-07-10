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
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/chaksack/apm/pkg/tools"
)

// ToolHandlers provides HTTP handlers for tool management
type ToolHandlers struct {
	detector          *tools.DetectorFactory
	healthChecker     *tools.HealthCheckerFactory
	portManager       *tools.PortManager
	templateRenderer  *tools.ConfigTemplateRenderer
}

// NewToolHandlers creates new tool handlers
func NewToolHandlers() (*ToolHandlers, error) {
	renderer, err := tools.NewConfigTemplateRenderer()
	if err != nil {
		return nil, err
	}

	return &ToolHandlers{
		detector:          tools.NewDetectorFactory(),
		healthChecker:     tools.NewHealthCheckerFactory(),
		portManager:       tools.NewPortManager(),
		templateRenderer:  renderer,
	}, nil
}

// DetectTools detects all installed APM tools
func (th *ToolHandlers) DetectTools(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	detectedTools, err := tools.DetectAllTools(ctx)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to detect tools: %v", err),
		})
	}

	// Check health for each detected tool
	for _, tool := range detectedTools {
		checker, err := th.healthChecker.CreateHealthChecker(tool)
		if err != nil {
			tool.Status = tools.ToolStatusUnknown
			continue
		}

		health, err := checker.Check(ctx)
		if err != nil {
			tool.Status = tools.ToolStatusUnhealthy
		} else {
			tool.Status = health.Status
			tool.Version = health.Version
		}
		tool.LastHealthCheck = time.Now()
	}

	return c.JSON(fiber.Map{
		"tools": detectedTools,
		"count": len(detectedTools),
	})
}

// GetToolHealth checks the health of a specific tool
func (th *ToolHandlers) GetToolHealth(c *fiber.Ctx) error {
	toolName := c.Params("tool")
	
	// Map tool name to type
	toolType := tools.ToolType(toolName)
	
	// Create detector for the specific tool
	detector, err := th.detector.CreateDetector(toolType)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": fmt.Sprintf("Unsupported tool: %s", toolName),
		})
	}

	// Detect the tool
	tool, err := detector.Detect()
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fmt.Sprintf("Tool '%s' not found", toolName),
		})
	}

	// Create health checker
	checker, err := th.healthChecker.CreateHealthChecker(tool)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to create health checker: %v", err),
		})
	}

	// Check health
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	health, err := checker.Check(ctx)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Health check failed: %v", err),
		})
	}

	// Get metrics
	metrics, _ := checker.GetMetrics()

	return c.JSON(fiber.Map{
		"tool":    tool,
		"health":  health,
		"metrics": metrics,
	})
}

// GetToolConfig generates configuration for a tool
func (th *ToolHandlers) GetToolConfig(c *fiber.Ctx) error {
	toolName := c.Params("tool")
	toolType := tools.ToolType(toolName)

	// Parse configuration data from request body
	var configData map[string]interface{}
	if err := c.BodyParser(&configData); err != nil {
		// Use default configuration
		configData = make(map[string]interface{})
	}

	// Render configuration
	config, err := th.templateRenderer.Render(toolType, configData)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to generate configuration: %v", err),
		})
	}

	// Return as plain text for direct use
	c.Set("Content-Type", "text/plain")
	return c.SendString(config)
}

// GetAllocatedPorts returns all allocated ports
func (th *ToolHandlers) GetAllocatedPorts(c *fiber.Ctx) error {
	ports := th.portManager.GetAllocatedPorts()
	
	return c.JSON(fiber.Map{
		"allocated_ports": ports,
		"count":          len(ports),
	})
}

// AllocatePort allocates a port for a tool
func (th *ToolHandlers) AllocatePort(c *fiber.Ctx) error {
	var request struct {
		ToolType string `json:"tool_type"`
		PortName string `json:"port_name,omitempty"`
	}

	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	toolType := tools.ToolType(request.ToolType)
	
	var port int
	var err error
	
	if request.PortName != "" {
		// Allocate additional port
		port, err = th.portManager.AllocateAdditionalPort(toolType, request.PortName)
	} else {
		// Allocate main port
		port, err = th.portManager.AllocatePort(toolType)
	}

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to allocate port: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"tool_type": request.ToolType,
		"port_name": request.PortName,
		"port":      port,
	})
}

// GetPortRegistry returns the port registry information
func (th *ToolHandlers) GetPortRegistry(c *fiber.Ctx) error {
	registry := make(map[string]interface{})
	
	// Main ports
	for toolType, config := range tools.PortRegistry {
		registry[string(toolType)] = map[string]interface{}{
			"default":      config.Default,
			"alternatives": config.Alternatives,
			"description":  config.Description,
		}
	}

	// Additional ports
	for toolType, additionalPorts := range tools.AdditionalPorts {
		if _, exists := registry[string(toolType)]; exists {
			toolRegistry := registry[string(toolType)].(map[string]interface{})
			additionalMap := make(map[string]interface{})
			
			for name, config := range additionalPorts {
				additionalMap[name] = map[string]interface{}{
					"default":     config.Default,
					"protocol":    config.Protocol,
					"description": config.Description,
				}
			}
			
			toolRegistry["additional_ports"] = additionalMap
		}
	}

	return c.JSON(fiber.Map{
		"port_registry": registry,
	})
}

// RedirectToTool redirects to the specified monitoring tool
func (th *ToolHandlers) RedirectToTool(c *fiber.Ctx) error {
	toolName := c.Params("tool")
	toolType := tools.ToolType(toolName)
	
	// Create detector for the specific tool
	detector, err := th.detector.CreateDetector(toolType)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": fmt.Sprintf("Unsupported tool: %s", toolName),
		})
	}

	// Detect the tool
	tool, err := detector.Detect()
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fmt.Sprintf("Tool '%s' not found or not running", toolName),
		})
	}
	
	return c.Redirect(tool.Endpoint, fiber.StatusTemporaryRedirect)
}

// ListTools returns a list of available monitoring tools with their status
func (th *ToolHandlers) ListTools(c *fiber.Ctx) error {
	supportedTools := []tools.ToolType{
		tools.ToolTypePrometheus,
		tools.ToolTypeGrafana,
		tools.ToolTypeJaeger,
		tools.ToolTypeLoki,
		tools.ToolTypeAlertManager,
	}

	toolList := make([]map[string]interface{}, 0, len(supportedTools))
	
	for _, toolType := range supportedTools {
		detector, err := th.detector.CreateDetector(toolType)
		if err != nil {
			continue
		}

		toolInfo := map[string]interface{}{
			"name":   string(toolType),
			"status": "not_installed",
		}

		// Try to detect the tool
		if tool, err := detector.Detect(); err == nil {
			toolInfo["status"] = "installed"
			toolInfo["endpoint"] = tool.Endpoint
			toolInfo["port"] = tool.Port
			
			// Check health
			if checker, err := th.healthChecker.CreateHealthChecker(tool); err == nil {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				
				if health, err := checker.Check(ctx); err == nil {
					toolInfo["health"] = string(health.Status)
					toolInfo["version"] = health.Version
				}
			}
		}
		
		// Add port information
		if portConfig, exists := tools.PortRegistry[toolType]; exists {
			toolInfo["default_port"] = portConfig.Default
		}
		
		toolList = append(toolList, toolInfo)
	}
	
	return c.JSON(fiber.Map{
		"tools": toolList,
		"count": len(toolList),
	})
}