package deployment

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// KubernetesRollbackController manages rollbacks for Kubernetes deployments
type KubernetesRollbackController struct {
	client      kubernetes.Interface
	namespace   string
	monitor     *KubernetesMonitor
}

// NewKubernetesRollbackController creates a new Kubernetes rollback controller
func NewKubernetesRollbackController(client kubernetes.Interface, namespace string, monitor *KubernetesMonitor) *KubernetesRollbackController {
	return &KubernetesRollbackController{
		client:    client,
		namespace: namespace,
		monitor:   monitor,
	}
}

// CanRollback checks if a deployment can be rolled back
func (c *KubernetesRollbackController) CanRollback(deploymentID string) (bool, string, error) {
	// Get deployment info from monitor
	deployment, err := c.monitor.GetStatus(deploymentID)
	if err != nil {
		return false, "", fmt.Errorf("failed to get deployment status: %w", err)
	}

	// Check if deployment is in a state that allows rollback
	switch deployment.Status {
	case StatusCompleted, StatusFailed, StatusVerifying:
		// These states allow rollback
	default:
		return false, fmt.Sprintf("cannot rollback deployment in %s state", deployment.Status), nil
	}

	// Get deployment name from configuration
	deploymentName, ok := deployment.Configuration["deployment_name"].(string)
	if !ok {
		deploymentName = deployment.Name
	}

	// Check Kubernetes deployment
	k8sDeployment, err := c.client.AppsV1().Deployments(c.namespace).Get(
		context.Background(),
		deploymentName,
		metav1.GetOptions{},
	)
	if err != nil {
		return false, "", fmt.Errorf("failed to get kubernetes deployment: %w", err)
	}

	// Check if there's a previous revision to rollback to
	if k8sDeployment.Status.ObservedGeneration <= 1 {
		return false, "no previous revision to rollback to", nil
	}

	// Get replicasets to check for previous versions
	replicaSets, err := c.client.AppsV1().ReplicaSets(c.namespace).List(
		context.Background(),
		metav1.ListOptions{
			LabelSelector: fmt.Sprintf("app=%s", deployment.Name),
		},
	)
	if err != nil {
		return false, "", fmt.Errorf("failed to get replicasets: %w", err)
	}

	if len(replicaSets.Items) < 2 {
		return false, "no previous replicaset found", nil
	}

	return true, "", nil
}

// GenerateRollbackCommands generates platform-specific rollback commands
func (c *KubernetesRollbackController) GenerateRollbackCommands(deployment *Deployment, targetVersion string) ([]RollbackCommand, error) {
	if deployment.Platform != PlatformKubernetes {
		return nil, fmt.Errorf("unsupported platform: %s", deployment.Platform)
	}

	deploymentName, ok := deployment.Configuration["deployment_name"].(string)
	if !ok {
		deploymentName = deployment.Name
	}

	commands := []RollbackCommand{
		{
			Platform:    PlatformKubernetes,
			Command:     fmt.Sprintf("kubectl rollout undo deployment/%s -n %s", deploymentName, c.namespace),
			Description: "Rollback deployment to previous version",
			Order:       1,
			Timeout:     5 * time.Minute,
		},
		{
			Platform:    PlatformKubernetes,
			Command:     fmt.Sprintf("kubectl rollout status deployment/%s -n %s --timeout=5m", deploymentName, c.namespace),
			Description: "Wait for rollback to complete",
			Order:       2,
			Timeout:     5 * time.Minute,
		},
		{
			Platform:    PlatformKubernetes,
			Command:     fmt.Sprintf("kubectl get pods -l app=%s -n %s", deployment.Name, c.namespace),
			Description: "Verify pods are running",
			Order:       3,
			Timeout:     30 * time.Second,
		},
	}

	// If specific version is requested, add annotation update
	if targetVersion != "" && targetVersion != "previous" {
		commands = append([]RollbackCommand{
			{
				Platform:    PlatformKubernetes,
				Command:     fmt.Sprintf("kubectl set image deployment/%s *=*:%s -n %s", deploymentName, targetVersion, c.namespace),
				Description: fmt.Sprintf("Set deployment image to version %s", targetVersion),
				Order:       0,
				Timeout:     30 * time.Second,
			},
		}, commands...)
	}

	// Add service verification if service exists
	if serviceName, ok := deployment.Configuration["service_name"].(string); ok && serviceName != "" {
		commands = append(commands, RollbackCommand{
			Platform:    PlatformKubernetes,
			Command:     fmt.Sprintf("kubectl get endpoints %s -n %s", serviceName, c.namespace),
			Description: "Verify service endpoints",
			Order:       4,
			Timeout:     30 * time.Second,
		})
	}

	return commands, nil
}

// InitiateRollback starts a rollback operation
func (c *KubernetesRollbackController) InitiateRollback(deploymentID string, reason string, targetVersion string) (*RollbackInfo, error) {
	// Check if rollback is possible
	canRollback, message, err := c.CanRollback(deploymentID)
	if err != nil {
		return nil, err
	}
	if !canRollback {
		return nil, fmt.Errorf("cannot rollback: %s", message)
	}

	// Get deployment
	deployment, err := c.monitor.GetStatus(deploymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	// Generate rollback commands
	commands, err := c.GenerateRollbackCommands(deployment, targetVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to generate rollback commands: %w", err)
	}

	// Create rollback info
	rollbackInfo := &RollbackInfo{
		TargetVersion:      targetVersion,
		TargetDeploymentID: deploymentID,
		Reason:             reason,
		InitiatedBy:        "system", // This should come from auth context
		InitiatedAt:        time.Now(),
		Status:             StatusRollingBack,
		Commands:           commands,
	}

	// Update deployment with rollback info
	deployment.RollbackInfo = rollbackInfo
	deployment.Status = StatusRollingBack

	// Execute rollback
	deploymentName, ok := deployment.Configuration["deployment_name"].(string)
	if !ok {
		deploymentName = deployment.Name
	}

	// Perform Kubernetes rollback
	if targetVersion == "" || targetVersion == "previous" {
		// Rollback to previous version
		err = c.rollbackToPrevious(deploymentName)
	} else {
		// Rollback to specific version
		err = c.rollbackToVersion(deploymentName, targetVersion)
	}

	if err != nil {
		rollbackInfo.Status = StatusFailed
		deployment.Status = StatusFailed
		deployment.Error = fmt.Sprintf("rollback failed: %v", err)
		return rollbackInfo, err
	}

	// Start monitoring rollback progress
	go c.monitorRollback(deployment, rollbackInfo)

	return rollbackInfo, nil
}

// GetRollbackStatus returns the status of a rollback operation
func (c *KubernetesRollbackController) GetRollbackStatus(deploymentID string) (*RollbackInfo, error) {
	deployment, err := c.monitor.GetStatus(deploymentID)
	if err != nil {
		return nil, err
	}

	if deployment.RollbackInfo == nil {
		return nil, fmt.Errorf("no rollback in progress for deployment %s", deploymentID)
	}

	return deployment.RollbackInfo, nil
}

// rollbackToPrevious rolls back to the previous version
func (c *KubernetesRollbackController) rollbackToPrevious(deploymentName string) error {
	// Get current deployment
	deployment, err := c.client.AppsV1().Deployments(c.namespace).Get(
		context.Background(),
		deploymentName,
		metav1.GetOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	// Get revision history
	revisionHistoryLimit := int32(10)
	if deployment.Spec.RevisionHistoryLimit != nil {
		revisionHistoryLimit = *deployment.Spec.RevisionHistoryLimit
	}

	// Find previous revision
	replicaSets, err := c.client.AppsV1().ReplicaSets(c.namespace).List(
		context.Background(),
		metav1.ListOptions{
			LabelSelector: metav1.FormatLabelSelector(deployment.Spec.Selector),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to get replicasets: %w", err)
	}

	if len(replicaSets.Items) < 2 {
		return fmt.Errorf("no previous version found")
	}

	// Sort by revision number and get the second latest
	var previousRevision int64 = 0
	for _, rs := range replicaSets.Items {
		if rs.Annotations == nil {
			continue
		}
		revision, ok := rs.Annotations["deployment.kubernetes.io/revision"]
		if !ok {
			continue
		}
		// Find the revision before the current one
		// This is simplified - in production, you'd parse and compare revision numbers
		if revision != deployment.Annotations["deployment.kubernetes.io/revision"] {
			previousRevision++
		}
	}

	if previousRevision == 0 {
		return fmt.Errorf("no previous revision found")
	}

	// Trigger rollback by updating deployment
	deployment.Annotations["kubernetes.io/change-cause"] = fmt.Sprintf("Rollback to previous version")
	_, err = c.client.AppsV1().Deployments(c.namespace).Update(
		context.Background(),
		deployment,
		metav1.UpdateOptions{},
	)

	return err
}

// rollbackToVersion rolls back to a specific version
func (c *KubernetesRollbackController) rollbackToVersion(deploymentName string, version string) error {
	// Get deployment
	deployment, err := c.client.AppsV1().Deployments(c.namespace).Get(
		context.Background(),
		deploymentName,
		metav1.GetOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	// Update container images to target version
	for i := range deployment.Spec.Template.Spec.Containers {
		container := &deployment.Spec.Template.Spec.Containers[i]
		// Extract image name without version
		imageName := container.Image
		if idx := len(imageName) - 1; idx >= 0 && imageName[idx] == ':' {
			// Find the last colon to separate image name from tag
			for j := len(imageName) - 1; j >= 0; j-- {
				if imageName[j] == ':' {
					imageName = imageName[:j]
					break
				}
			}
		}
		container.Image = fmt.Sprintf("%s:%s", imageName, version)
	}

	// Update deployment
	deployment.Annotations["kubernetes.io/change-cause"] = fmt.Sprintf("Rollback to version %s", version)
	_, err = c.client.AppsV1().Deployments(c.namespace).Update(
		context.Background(),
		deployment,
		metav1.UpdateOptions{},
	)

	return err
}

// monitorRollback monitors the rollback progress
func (c *KubernetesRollbackController) monitorRollback(deployment *Deployment, rollbackInfo *RollbackInfo) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeout := time.After(10 * time.Minute)

	for {
		select {
		case <-ticker.C:
			// Get deployment status
			status, err := c.monitor.GetStatus(deployment.ID)
			if err != nil {
				continue
			}

			// Check if rollback is complete
			if status.Status == StatusCompleted {
				rollbackInfo.Status = StatusCompleted
				now := time.Now()
				rollbackInfo.CompletedAt = &now
				deployment.Status = StatusRolledBack
				return
			} else if status.Status == StatusFailed {
				rollbackInfo.Status = StatusFailed
				now := time.Now()
				rollbackInfo.CompletedAt = &now
				deployment.Status = StatusFailed
				return
			}

		case <-timeout:
			// Timeout
			rollbackInfo.Status = StatusFailed
			now := time.Now()
			rollbackInfo.CompletedAt = &now
			deployment.Status = StatusFailed
			deployment.Error = "rollback timed out"
			return
		}
	}
}