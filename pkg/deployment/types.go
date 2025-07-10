package deployment

import (
	"time"
)

// DeploymentStatus represents the current status of a deployment
type DeploymentStatus string

const (
	StatusPending    DeploymentStatus = "pending"
	StatusPreparing  DeploymentStatus = "preparing"
	StatusDeploying  DeploymentStatus = "deploying"
	StatusVerifying  DeploymentStatus = "verifying"
	StatusCompleted  DeploymentStatus = "completed"
	StatusFailed     DeploymentStatus = "failed"
	StatusRollingBack DeploymentStatus = "rolling_back"
	StatusRolledBack  DeploymentStatus = "rolled_back"
)

// DeploymentPlatform represents the target deployment platform
type DeploymentPlatform string

const (
	PlatformKubernetes    DeploymentPlatform = "kubernetes"
	PlatformDocker        DeploymentPlatform = "docker"
	PlatformDockerCompose DeploymentPlatform = "docker-compose"
	PlatformAWS           DeploymentPlatform = "aws"
	PlatformGCP           DeploymentPlatform = "gcp"
	PlatformAzure         DeploymentPlatform = "azure"
)

// Deployment represents a deployment operation
type Deployment struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	Version       string             `json:"version"`
	Platform      DeploymentPlatform `json:"platform"`
	Environment   string             `json:"environment"`
	Status        DeploymentStatus   `json:"status"`
	StartTime     time.Time          `json:"start_time"`
	EndTime       *time.Time         `json:"end_time,omitempty"`
	Components    []Component        `json:"components"`
	Configuration map[string]interface{} `json:"configuration"`
	Progress      *DeploymentProgress    `json:"progress,omitempty"`
	HealthChecks  []HealthCheck          `json:"health_checks,omitempty"`
	Error         string                 `json:"error,omitempty"`
	RollbackInfo  *RollbackInfo          `json:"rollback_info,omitempty"`
	Metadata      map[string]string      `json:"metadata,omitempty"`
}

// Component represents a component being deployed
type Component struct {
	Name         string           `json:"name"`
	Type         string           `json:"type"`
	Version      string           `json:"version"`
	Status       DeploymentStatus `json:"status"`
	StartTime    time.Time        `json:"start_time"`
	EndTime      *time.Time       `json:"end_time,omitempty"`
	HealthChecks []HealthCheck    `json:"health_checks,omitempty"`
	Error        string           `json:"error,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// DeploymentProgress tracks the progress of a deployment
type DeploymentProgress struct {
	TotalSteps     int               `json:"total_steps"`
	CurrentStep    int               `json:"current_step"`
	CurrentStage   string            `json:"current_stage"`
	Percentage     float64           `json:"percentage"`
	EstimatedTime  *time.Duration    `json:"estimated_time,omitempty"`
	Messages       []ProgressMessage `json:"messages"`
}

// ProgressMessage represents a progress update message
type ProgressMessage struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Component string    `json:"component,omitempty"`
}

// HealthCheck represents a health check result
type HealthCheck struct {
	Name        string            `json:"name"`
	Type        HealthCheckType   `json:"type"`
	Status      HealthStatus      `json:"status"`
	Message     string            `json:"message,omitempty"`
	LastChecked time.Time         `json:"last_checked"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// HealthCheckType represents the type of health check
type HealthCheckType string

const (
	HealthCheckReadiness HealthCheckType = "readiness"
	HealthCheckLiveness  HealthCheckType = "liveness"
	HealthCheckStartup   HealthCheckType = "startup"
	HealthCheckCustom    HealthCheckType = "custom"
)

// HealthStatus represents the health status
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// RollbackInfo contains information about rollback operations
type RollbackInfo struct {
	TargetVersion   string            `json:"target_version"`
	TargetDeploymentID string         `json:"target_deployment_id"`
	Reason          string            `json:"reason"`
	InitiatedBy     string            `json:"initiated_by"`
	InitiatedAt     time.Time         `json:"initiated_at"`
	CompletedAt     *time.Time        `json:"completed_at,omitempty"`
	Status          DeploymentStatus  `json:"status"`
	Commands        []RollbackCommand `json:"commands,omitempty"`
}

// RollbackCommand represents a command to execute for rollback
type RollbackCommand struct {
	Platform    DeploymentPlatform `json:"platform"`
	Command     string             `json:"command"`
	Description string             `json:"description"`
	Order       int                `json:"order"`
	Timeout     time.Duration      `json:"timeout"`
}

// DeploymentHistory represents historical deployment data
type DeploymentHistory struct {
	ID           string    `json:"id"`
	DeploymentID string    `json:"deployment_id"`
	Timestamp    time.Time `json:"timestamp"`
	Event        string    `json:"event"`
	Details      map[string]interface{} `json:"details"`
	Actor        string    `json:"actor,omitempty"`
}

// DeploymentMonitor interface for monitoring deployments
type DeploymentMonitor interface {
	// Start begins monitoring a deployment
	Start(deployment *Deployment) error
	
	// GetStatus returns the current status of a deployment
	GetStatus(deploymentID string) (*Deployment, error)
	
	// UpdateProgress updates the deployment progress
	UpdateProgress(deploymentID string, progress *DeploymentProgress) error
	
	// CheckHealth performs health checks for a deployment
	CheckHealth(deploymentID string) ([]HealthCheck, error)
	
	// Stop stops monitoring a deployment
	Stop(deploymentID string) error
}

// RollbackController interface for managing rollbacks
type RollbackController interface {
	// CanRollback checks if a deployment can be rolled back
	CanRollback(deploymentID string) (bool, string, error)
	
	// GenerateRollbackCommands generates platform-specific rollback commands
	GenerateRollbackCommands(deployment *Deployment, targetVersion string) ([]RollbackCommand, error)
	
	// InitiateRollback starts a rollback operation
	InitiateRollback(deploymentID string, reason string, targetVersion string) (*RollbackInfo, error)
	
	// GetRollbackStatus returns the status of a rollback operation
	GetRollbackStatus(deploymentID string) (*RollbackInfo, error)
}

// HistoryManager interface for managing deployment history
type HistoryManager interface {
	// RecordDeployment records a new deployment
	RecordDeployment(deployment *Deployment) error
	
	// RecordEvent records a deployment event
	RecordEvent(deploymentID string, event string, details map[string]interface{}) error
	
	// GetHistory retrieves deployment history
	GetHistory(filters HistoryFilters) ([]DeploymentHistory, error)
	
	// GetDeployment retrieves a specific deployment
	GetDeployment(deploymentID string) (*Deployment, error)
	
	// GetDeployments retrieves multiple deployments
	GetDeployments(filters DeploymentFilters) ([]Deployment, error)
}

// HistoryFilters contains filters for querying deployment history
type HistoryFilters struct {
	DeploymentID string
	StartTime    *time.Time
	EndTime      *time.Time
	Event        string
	Limit        int
	Offset       int
}

// DeploymentFilters contains filters for querying deployments
type DeploymentFilters struct {
	Platform    DeploymentPlatform
	Environment string
	Status      DeploymentStatus
	StartTime   *time.Time
	EndTime     *time.Time
	Limit       int
	Offset      int
}

// StatusUpdate represents a real-time status update
type StatusUpdate struct {
	DeploymentID string             `json:"deployment_id"`
	Timestamp    time.Time          `json:"timestamp"`
	Type         StatusUpdateType   `json:"type"`
	Data         interface{}        `json:"data"`
}

// StatusUpdateType represents the type of status update
type StatusUpdateType string

const (
	UpdateTypeStatus   StatusUpdateType = "status"
	UpdateTypeProgress StatusUpdateType = "progress"
	UpdateTypeHealth   StatusUpdateType = "health"
	UpdateTypeLog      StatusUpdateType = "log"
	UpdateTypeError    StatusUpdateType = "error"
)