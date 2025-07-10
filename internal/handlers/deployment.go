package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/chaksack/apm/pkg/deployment"
)

// DeploymentHandler handles deployment-related requests
type DeploymentHandler struct {
	service *deployment.Service
}

// NewDeploymentHandler creates a new deployment handler
func NewDeploymentHandler(service *deployment.Service) *DeploymentHandler {
	return &DeploymentHandler{
		service: service,
	}
}

// StartDeployment starts a new deployment
func (h *DeploymentHandler) StartDeployment(c *fiber.Ctx) error {
	var request deployment.DeploymentRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if request.Name == "" || request.Version == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Name and version are required",
		})
	}

	// Start deployment
	deployment, err := h.service.StartDeployment(request)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(deployment)
}

// GetDeploymentStatus gets the status of a deployment
func (h *DeploymentHandler) GetDeploymentStatus(c *fiber.Ctx) error {
	deploymentID := c.Params("id")
	if deploymentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Deployment ID is required",
		})
	}

	deployment, err := h.service.GetDeploymentStatus(deploymentID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(deployment)
}

// GetDeployments gets multiple deployments with filters
func (h *DeploymentHandler) GetDeployments(c *fiber.Ctx) error {
	filters := deployment.DeploymentFilters{
		Platform:    deployment.DeploymentPlatform(c.Query("platform")),
		Environment: c.Query("environment"),
		Status:      deployment.DeploymentStatus(c.Query("status")),
		Limit:       c.QueryInt("limit", 20),
		Offset:      c.QueryInt("offset", 0),
	}

	// Parse time filters
	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filters.StartTime = &t
		}
	}
	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			filters.EndTime = &t
		}
	}

	deployments, err := h.service.GetDeployments(filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"deployments": deployments,
		"total":       len(deployments),
		"limit":       filters.Limit,
		"offset":      filters.Offset,
	})
}

// CheckDeploymentHealth performs health checks for a deployment
func (h *DeploymentHandler) CheckDeploymentHealth(c *fiber.Ctx) error {
	deploymentID := c.Params("id")
	if deploymentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Deployment ID is required",
		})
	}

	healthChecks, err := h.service.CheckDeploymentHealth(deploymentID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Calculate overall health status
	overallStatus := deployment.HealthStatusHealthy
	unhealthyCount := 0
	for _, check := range healthChecks {
		if check.Status == deployment.HealthStatusUnhealthy {
			unhealthyCount++
			overallStatus = deployment.HealthStatusUnhealthy
		} else if check.Status == deployment.HealthStatusDegraded && overallStatus == deployment.HealthStatusHealthy {
			overallStatus = deployment.HealthStatusDegraded
		}
	}

	return c.JSON(fiber.Map{
		"deployment_id":   deploymentID,
		"overall_status":  overallStatus,
		"health_checks":   healthChecks,
		"total_checks":    len(healthChecks),
		"unhealthy_count": unhealthyCount,
		"timestamp":       time.Now(),
	})
}

// InitiateRollback initiates a rollback for a deployment
func (h *DeploymentHandler) InitiateRollback(c *fiber.Ctx) error {
	deploymentID := c.Params("id")
	if deploymentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Deployment ID is required",
		})
	}

	var request struct {
		Reason        string `json:"reason" validate:"required"`
		TargetVersion string `json:"target_version"`
	}
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if request.Reason == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Rollback reason is required",
		})
	}

	rollbackInfo, err := h.service.InitiateRollback(deploymentID, request.Reason, request.TargetVersion)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(rollbackInfo)
}

// GetRollbackCommands gets rollback commands for a deployment
func (h *DeploymentHandler) GetRollbackCommands(c *fiber.Ctx) error {
	deploymentID := c.Params("id")
	if deploymentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Deployment ID is required",
		})
	}

	targetVersion := c.Query("target_version", "previous")

	commands, err := h.service.GetRollbackCommands(deploymentID, targetVersion)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"deployment_id":  deploymentID,
		"target_version": targetVersion,
		"commands":       commands,
		"total":          len(commands),
	})
}

// GetDeploymentHistory gets deployment history
func (h *DeploymentHandler) GetDeploymentHistory(c *fiber.Ctx) error {
	deploymentID := c.Params("id")

	filters := deployment.HistoryFilters{
		DeploymentID: deploymentID,
		Event:        c.Query("event"),
		Limit:        c.QueryInt("limit", 50),
		Offset:       c.QueryInt("offset", 0),
	}

	// Parse time filters
	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filters.StartTime = &t
		}
	}
	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			filters.EndTime = &t
		}
	}

	history, err := h.service.GetDeploymentHistory(filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"deployment_id": deploymentID,
		"history":       history,
		"total":         len(history),
		"limit":         filters.Limit,
		"offset":        filters.Offset,
	})
}

// WebSocketHandler handles WebSocket connections for real-time updates
func (h *DeploymentHandler) WebSocketHandler() fiber.Handler {
	return websocket.New(deployment.HandleWebSocket(h.service.GetWebSocketHub()))
}

// RegisterRoutes registers deployment routes
func (h *DeploymentHandler) RegisterRoutes(app *fiber.App) {
	// API routes
	api := app.Group("/api/v1/deployments")
	
	// Deployment management
	api.Post("/", h.StartDeployment)
	api.Get("/", h.GetDeployments)
	api.Get("/:id", h.GetDeploymentStatus)
	api.Get("/:id/health", h.CheckDeploymentHealth)
	api.Get("/:id/history", h.GetDeploymentHistory)
	
	// Rollback operations
	api.Post("/:id/rollback", h.InitiateRollback)
	api.Get("/:id/rollback/commands", h.GetRollbackCommands)
	
	// WebSocket endpoint for real-time updates
	app.Get("/ws/deployments", websocket.New(deployment.HandleWebSocket(h.service.GetWebSocketHub())))
}