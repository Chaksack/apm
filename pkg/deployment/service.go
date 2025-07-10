package deployment

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gomodule/redigo/redis"
)

// Service provides deployment monitoring and management capabilities
type Service struct {
	monitors    map[DeploymentPlatform]DeploymentMonitor
	rollbacks   map[DeploymentPlatform]RollbackController
	history     HistoryManager
	hub         *WebSocketHub
	streamer    *StatusStreamer
	cache       *redis.Pool
	mu          sync.RWMutex
}

// ServiceConfig contains configuration for the deployment service
type ServiceConfig struct {
	KubeConfig       string
	KubeNamespace    string
	DatabaseURL      string
	RedisURL         string
	DockerHost       string
}

// NewService creates a new deployment service
func NewService(config ServiceConfig) (*Service, error) {
	// Create history manager
	history, err := NewPostgresHistoryManager(config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create history manager: %w", err)
	}

	// Create WebSocket hub
	hub := NewWebSocketHub()
	go hub.Run()

	// Create status streamer
	streamer := NewStatusStreamer(hub)

	// Create Redis pool for caching
	redisPool := &redis.Pool{
		MaxIdle:     3,
		MaxActive:   10,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.DialURL(config.RedisURL)
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}

	service := &Service{
		monitors:  make(map[DeploymentPlatform]DeploymentMonitor),
		rollbacks: make(map[DeploymentPlatform]RollbackController),
		history:   history,
		hub:       hub,
		streamer:  streamer,
		cache:     redisPool,
	}

	// Initialize Kubernetes monitor and rollback controller
	if config.KubeConfig != "" || config.KubeNamespace != "" {
		kubeMonitor, err := NewKubernetesMonitor(config.KubeConfig, config.KubeNamespace)
		if err != nil {
			return nil, fmt.Errorf("failed to create kubernetes monitor: %w", err)
		}
		service.monitors[PlatformKubernetes] = kubeMonitor

		// Create Kubernetes rollback controller
		kubeRollback := NewKubernetesRollbackController(kubeMonitor.client, config.KubeNamespace, kubeMonitor)
		service.rollbacks[PlatformKubernetes] = kubeRollback
	}

	// TODO: Initialize Docker monitor and rollback controller
	// TODO: Initialize cloud platform monitors and rollback controllers

	return service, nil
}

// StartDeployment starts monitoring a new deployment
func (s *Service) StartDeployment(request DeploymentRequest) (*Deployment, error) {
	// Create deployment
	deployment := &Deployment{
		ID:            uuid.New().String(),
		Name:          request.Name,
		Version:       request.Version,
		Platform:      request.Platform,
		Environment:   request.Environment,
		Status:        StatusPending,
		StartTime:     time.Now(),
		Components:    request.Components,
		Configuration: request.Configuration,
		Metadata:      request.Metadata,
	}

	// Record deployment in history
	if err := s.history.RecordDeployment(deployment); err != nil {
		return nil, fmt.Errorf("failed to record deployment: %w", err)
	}

	// Get appropriate monitor
	monitor, exists := s.monitors[request.Platform]
	if !exists {
		return nil, fmt.Errorf("unsupported platform: %s", request.Platform)
	}

	// Start monitoring
	deployment.Status = StatusPreparing
	if err := monitor.Start(deployment); err != nil {
		deployment.Status = StatusFailed
		deployment.Error = err.Error()
		s.history.RecordDeployment(deployment)
		return nil, fmt.Errorf("failed to start monitoring: %w", err)
	}

	// Update status
	deployment.Status = StatusDeploying
	s.history.RecordDeployment(deployment)

	// Stream initial status
	s.streamer.StreamDeploymentStatus(deployment)

	// Start background monitoring
	go s.monitorDeployment(deployment.ID)

	// Cache deployment
	s.cacheDeployment(deployment)

	return deployment, nil
}

// GetDeploymentStatus gets the current status of a deployment
func (s *Service) GetDeploymentStatus(deploymentID string) (*Deployment, error) {
	// Check cache first
	if deployment := s.getCachedDeployment(deploymentID); deployment != nil {
		return deployment, nil
	}

	// Get from history
	deployment, err := s.history.GetDeployment(deploymentID)
	if err != nil {
		return nil, err
	}

	// Get real-time status from monitor if active
	if monitor := s.getMonitorForDeployment(deployment); monitor != nil {
		if status, err := monitor.GetStatus(deploymentID); err == nil {
			deployment = status
		}
	}

	// Cache deployment
	s.cacheDeployment(deployment)

	return deployment, nil
}

// CheckDeploymentHealth performs health checks for a deployment
func (s *Service) CheckDeploymentHealth(deploymentID string) ([]HealthCheck, error) {
	deployment, err := s.GetDeploymentStatus(deploymentID)
	if err != nil {
		return nil, err
	}

	monitor := s.getMonitorForDeployment(deployment)
	if monitor == nil {
		return nil, fmt.Errorf("no monitor available for deployment")
	}

	healthChecks, err := monitor.CheckHealth(deploymentID)
	if err != nil {
		return nil, err
	}

	// Update deployment with health checks
	deployment.HealthChecks = healthChecks
	s.history.RecordDeployment(deployment)

	// Stream health check results
	s.streamer.StreamHealthCheck(deploymentID, healthChecks)

	return healthChecks, nil
}

// InitiateRollback initiates a rollback for a deployment
func (s *Service) InitiateRollback(deploymentID string, reason string, targetVersion string) (*RollbackInfo, error) {
	deployment, err := s.GetDeploymentStatus(deploymentID)
	if err != nil {
		return nil, err
	}

	controller := s.getRollbackControllerForDeployment(deployment)
	if controller == nil {
		return nil, fmt.Errorf("no rollback controller available for platform %s", deployment.Platform)
	}

	// Check if rollback is possible
	canRollback, message, err := controller.CanRollback(deploymentID)
	if err != nil {
		return nil, err
	}
	if !canRollback {
		return nil, fmt.Errorf("cannot rollback: %s", message)
	}

	// Initiate rollback
	rollbackInfo, err := controller.InitiateRollback(deploymentID, reason, targetVersion)
	if err != nil {
		return nil, err
	}

	// Record rollback event
	s.history.RecordEvent(deploymentID, "rollback_initiated", map[string]interface{}{
		"reason":         reason,
		"target_version": targetVersion,
		"initiated_by":   rollbackInfo.InitiatedBy,
	})

	// Stream rollback status
	s.streamer.StreamDeploymentStatus(deployment)

	return rollbackInfo, nil
}

// GetRollbackCommands generates rollback commands for a deployment
func (s *Service) GetRollbackCommands(deploymentID string, targetVersion string) ([]RollbackCommand, error) {
	deployment, err := s.GetDeploymentStatus(deploymentID)
	if err != nil {
		return nil, err
	}

	controller := s.getRollbackControllerForDeployment(deployment)
	if controller == nil {
		return nil, fmt.Errorf("no rollback controller available for platform %s", deployment.Platform)
	}

	return controller.GenerateRollbackCommands(deployment, targetVersion)
}

// GetDeploymentHistory gets deployment history
func (s *Service) GetDeploymentHistory(filters HistoryFilters) ([]DeploymentHistory, error) {
	return s.history.GetHistory(filters)
}

// GetDeployments gets multiple deployments
func (s *Service) GetDeployments(filters DeploymentFilters) ([]Deployment, error) {
	return s.history.GetDeployments(filters)
}

// GetWebSocketHub returns the WebSocket hub for handling connections
func (s *Service) GetWebSocketHub() *WebSocketHub {
	return s.hub
}

// monitorDeployment monitors a deployment in the background
func (s *Service) monitorDeployment(deploymentID string) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	timeout := time.After(30 * time.Minute)

	for {
		select {
		case <-ticker.C:
			deployment, err := s.GetDeploymentStatus(deploymentID)
			if err != nil {
				continue
			}

			// Check if deployment is complete
			if deployment.Status == StatusCompleted || 
			   deployment.Status == StatusFailed || 
			   deployment.Status == StatusRolledBack {
				// Final status reached
				endTime := time.Now()
				deployment.EndTime = &endTime
				s.history.RecordDeployment(deployment)
				
				// Record completion event
				s.history.RecordEvent(deploymentID, "deployment_completed", map[string]interface{}{
					"status":   deployment.Status,
					"duration": endTime.Sub(deployment.StartTime).String(),
				})

				// Stream final status
				s.streamer.StreamDeploymentStatus(deployment)
				return
			}

			// Perform health checks
			if deployment.Status == StatusVerifying || deployment.Status == StatusCompleted {
				healthChecks, err := s.CheckDeploymentHealth(deploymentID)
				if err == nil {
					// Determine overall health
					allHealthy := true
					for _, check := range healthChecks {
						if check.Status != HealthStatusHealthy {
							allHealthy = false
							break
						}
					}

					if allHealthy && deployment.Status == StatusVerifying {
						deployment.Status = StatusCompleted
						s.history.RecordDeployment(deployment)
						s.streamer.StreamDeploymentStatus(deployment)
					}
				}
			}

		case <-timeout:
			// Deployment timeout
			deployment, _ := s.GetDeploymentStatus(deploymentID)
			if deployment != nil && 
			   deployment.Status != StatusCompleted && 
			   deployment.Status != StatusFailed {
				deployment.Status = StatusFailed
				deployment.Error = "deployment timed out after 30 minutes"
				endTime := time.Now()
				deployment.EndTime = &endTime
				s.history.RecordDeployment(deployment)
				s.streamer.StreamError(deploymentID, deployment.Error)
			}
			return
		}
	}
}

// getMonitorForDeployment gets the appropriate monitor for a deployment
func (s *Service) getMonitorForDeployment(deployment *Deployment) DeploymentMonitor {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.monitors[deployment.Platform]
}

// getRollbackControllerForDeployment gets the appropriate rollback controller
func (s *Service) getRollbackControllerForDeployment(deployment *Deployment) RollbackController {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.rollbacks[deployment.Platform]
}

// cacheDeployment caches a deployment in Redis
func (s *Service) cacheDeployment(deployment *Deployment) {
	conn := s.cache.Get()
	defer conn.Close()

	key := fmt.Sprintf("deployment:%s", deployment.ID)
	data, _ := json.Marshal(deployment)
	conn.Do("SETEX", key, 300, data) // 5 minute cache
}

// getCachedDeployment gets a cached deployment from Redis
func (s *Service) getCachedDeployment(deploymentID string) *Deployment {
	conn := s.cache.Get()
	defer conn.Close()

	key := fmt.Sprintf("deployment:%s", deploymentID)
	data, err := redis.Bytes(conn.Do("GET", key))
	if err != nil {
		return nil
	}

	var deployment Deployment
	if err := json.Unmarshal(data, &deployment); err != nil {
		return nil
	}

	return &deployment
}

// DeploymentRequest represents a request to start a new deployment
type DeploymentRequest struct {
	Name          string                 `json:"name" validate:"required"`
	Version       string                 `json:"version" validate:"required"`
	Platform      DeploymentPlatform     `json:"platform" validate:"required"`
	Environment   string                 `json:"environment" validate:"required"`
	Components    []Component            `json:"components"`
	Configuration map[string]interface{} `json:"configuration"`
	Metadata      map[string]string      `json:"metadata"`
}