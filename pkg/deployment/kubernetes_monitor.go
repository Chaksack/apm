package deployment

import (
	"context"
	"fmt"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// KubernetesMonitor monitors Kubernetes deployments
type KubernetesMonitor struct {
	client      kubernetes.Interface
	namespace   string
	deployments map[string]*kubernetesDeploymentInfo
	mu          sync.RWMutex
	stopCh      map[string]chan struct{}
}

type kubernetesDeploymentInfo struct {
	deployment *Deployment
	resources  kubernetesResources
	watcher    watch.Interface
}

type kubernetesResources struct {
	deploymentName string
	serviceName    string
	configMapName  string
	labels         map[string]string
}

// NewKubernetesMonitor creates a new Kubernetes deployment monitor
func NewKubernetesMonitor(kubeconfig string, namespace string) (*KubernetesMonitor, error) {
	var config *rest.Config
	var err error

	if kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &KubernetesMonitor{
		client:      clientset,
		namespace:   namespace,
		deployments: make(map[string]*kubernetesDeploymentInfo),
		stopCh:      make(map[string]chan struct{}),
	}, nil
}

// Start begins monitoring a Kubernetes deployment
func (m *KubernetesMonitor) Start(deployment *Deployment) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if deployment.Platform != PlatformKubernetes {
		return fmt.Errorf("unsupported platform: %s", deployment.Platform)
	}

	// Extract Kubernetes resources from deployment configuration
	resources, err := m.extractResources(deployment)
	if err != nil {
		return fmt.Errorf("failed to extract resources: %w", err)
	}

	// Create deployment info
	info := &kubernetesDeploymentInfo{
		deployment: deployment,
		resources:  resources,
	}

	// Start watching the deployment
	stopCh := make(chan struct{})
	m.stopCh[deployment.ID] = stopCh

	go m.watchDeployment(deployment.ID, info, stopCh)

	m.deployments[deployment.ID] = info

	return nil
}

// GetStatus returns the current status of a deployment
func (m *KubernetesMonitor) GetStatus(deploymentID string) (*Deployment, error) {
	m.mu.RLock()
	info, exists := m.deployments[deploymentID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("deployment not found: %s", deploymentID)
	}

	// Get current deployment status from Kubernetes
	deployment, err := m.client.AppsV1().Deployments(m.namespace).Get(
		context.Background(),
		info.resources.deploymentName,
		metav1.GetOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	// Update deployment status
	status := m.calculateDeploymentStatus(deployment)
	info.deployment.Status = status

	// Get pod status
	pods, err := m.getPods(info.resources.labels)
	if err == nil {
		info.deployment.Progress = m.calculateProgress(deployment, pods)
	}

	return info.deployment, nil
}

// UpdateProgress updates the deployment progress
func (m *KubernetesMonitor) UpdateProgress(deploymentID string, progress *DeploymentProgress) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, exists := m.deployments[deploymentID]
	if !exists {
		return fmt.Errorf("deployment not found: %s", deploymentID)
	}

	info.deployment.Progress = progress
	return nil
}

// CheckHealth performs health checks for a deployment
func (m *KubernetesMonitor) CheckHealth(deploymentID string) ([]HealthCheck, error) {
	m.mu.RLock()
	info, exists := m.deployments[deploymentID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("deployment not found: %s", deploymentID)
	}

	var healthChecks []HealthCheck

	// Get pods
	pods, err := m.getPods(info.resources.labels)
	if err != nil {
		return nil, fmt.Errorf("failed to get pods: %w", err)
	}

	// Check pod health
	for _, pod := range pods.Items {
		// Readiness check
		readiness := HealthCheck{
			Name:        fmt.Sprintf("%s-readiness", pod.Name),
			Type:        HealthCheckReadiness,
			LastChecked: time.Now(),
			Metadata: map[string]string{
				"pod": pod.Name,
			},
		}

		if isPodReady(&pod) {
			readiness.Status = HealthStatusHealthy
			readiness.Message = "Pod is ready"
		} else {
			readiness.Status = HealthStatusUnhealthy
			readiness.Message = getPodConditionMessage(&pod)
		}

		healthChecks = append(healthChecks, readiness)

		// Liveness check (based on container status)
		for _, container := range pod.Status.ContainerStatuses {
			liveness := HealthCheck{
				Name:        fmt.Sprintf("%s-%s-liveness", pod.Name, container.Name),
				Type:        HealthCheckLiveness,
				LastChecked: time.Now(),
				Metadata: map[string]string{
					"pod":       pod.Name,
					"container": container.Name,
				},
			}

			if container.Ready && container.State.Running != nil {
				liveness.Status = HealthStatusHealthy
				liveness.Message = "Container is running"
			} else {
				liveness.Status = HealthStatusUnhealthy
				liveness.Message = getContainerStateMessage(&container)
			}

			healthChecks = append(healthChecks, liveness)
		}
	}

	// Check service endpoints
	if info.resources.serviceName != "" {
		endpoints, err := m.client.CoreV1().Endpoints(m.namespace).Get(
			context.Background(),
			info.resources.serviceName,
			metav1.GetOptions{},
		)
		if err == nil {
			serviceHealth := HealthCheck{
				Name:        fmt.Sprintf("%s-service", info.resources.serviceName),
				Type:        HealthCheckCustom,
				LastChecked: time.Now(),
				Metadata: map[string]string{
					"service": info.resources.serviceName,
				},
			}

			if len(endpoints.Subsets) > 0 && len(endpoints.Subsets[0].Addresses) > 0 {
				serviceHealth.Status = HealthStatusHealthy
				serviceHealth.Message = fmt.Sprintf("Service has %d endpoints", len(endpoints.Subsets[0].Addresses))
			} else {
				serviceHealth.Status = HealthStatusUnhealthy
				serviceHealth.Message = "Service has no endpoints"
			}

			healthChecks = append(healthChecks, serviceHealth)
		}
	}

	return healthChecks, nil
}

// Stop stops monitoring a deployment
func (m *KubernetesMonitor) Stop(deploymentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, exists := m.deployments[deploymentID]
	if !exists {
		return fmt.Errorf("deployment not found: %s", deploymentID)
	}

	// Stop watching
	if stopCh, ok := m.stopCh[deploymentID]; ok {
		close(stopCh)
		delete(m.stopCh, deploymentID)
	}

	// Stop watcher
	if info.watcher != nil {
		info.watcher.Stop()
	}

	delete(m.deployments, deploymentID)

	return nil
}

// watchDeployment watches a Kubernetes deployment for changes
func (m *KubernetesMonitor) watchDeployment(deploymentID string, info *kubernetesDeploymentInfo, stopCh <-chan struct{}) {
	// Watch deployment
	watcher, err := m.client.AppsV1().Deployments(m.namespace).Watch(
		context.Background(),
		metav1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", info.resources.deploymentName),
		},
	)
	if err != nil {
		return
	}
	info.watcher = watcher

	for {
		select {
		case event := <-watcher.ResultChan():
			if event.Type == watch.Error {
				continue
			}

			deployment, ok := event.Object.(*appsv1.Deployment)
			if !ok {
				continue
			}

			// Update deployment status
			m.mu.Lock()
			if currentInfo, exists := m.deployments[deploymentID]; exists {
				currentInfo.deployment.Status = m.calculateDeploymentStatus(deployment)
				
				// Get pods for progress calculation
				pods, err := m.getPods(info.resources.labels)
				if err == nil {
					currentInfo.deployment.Progress = m.calculateProgress(deployment, pods)
				}
			}
			m.mu.Unlock()

		case <-stopCh:
			watcher.Stop()
			return
		}
	}
}

// extractResources extracts Kubernetes resource information from deployment config
func (m *KubernetesMonitor) extractResources(deployment *Deployment) (kubernetesResources, error) {
	resources := kubernetesResources{
		labels: make(map[string]string),
	}

	// Extract deployment name
	if name, ok := deployment.Configuration["deployment_name"].(string); ok {
		resources.deploymentName = name
	} else {
		resources.deploymentName = deployment.Name
	}

	// Extract service name
	if name, ok := deployment.Configuration["service_name"].(string); ok {
		resources.serviceName = name
	}

	// Extract labels
	if labels, ok := deployment.Configuration["labels"].(map[string]interface{}); ok {
		for k, v := range labels {
			if str, ok := v.(string); ok {
				resources.labels[k] = str
			}
		}
	}

	// Default labels
	if len(resources.labels) == 0 {
		resources.labels["app"] = deployment.Name
		resources.labels["version"] = deployment.Version
	}

	return resources, nil
}

// calculateDeploymentStatus calculates the deployment status from Kubernetes deployment
func (m *KubernetesMonitor) calculateDeploymentStatus(deployment *appsv1.Deployment) DeploymentStatus {
	// Check deployment conditions
	for _, condition := range deployment.Status.Conditions {
		if condition.Type == appsv1.DeploymentProgressing {
			if condition.Status == corev1.ConditionTrue {
				if deployment.Status.UpdatedReplicas < *deployment.Spec.Replicas {
					return StatusDeploying
				}
				if deployment.Status.ReadyReplicas < *deployment.Spec.Replicas {
					return StatusVerifying
				}
			} else if condition.Reason == "ProgressDeadlineExceeded" {
				return StatusFailed
			}
		}
	}

	// Check if deployment is complete
	if deployment.Status.ObservedGeneration >= deployment.Generation &&
		deployment.Status.UpdatedReplicas == *deployment.Spec.Replicas &&
		deployment.Status.ReadyReplicas == *deployment.Spec.Replicas {
		return StatusCompleted
	}

	return StatusDeploying
}

// calculateProgress calculates deployment progress
func (m *KubernetesMonitor) calculateProgress(deployment *appsv1.Deployment, pods *corev1.PodList) *DeploymentProgress {
	totalReplicas := int32(0)
	if deployment.Spec.Replicas != nil {
		totalReplicas = *deployment.Spec.Replicas
	}

	progress := &DeploymentProgress{
		TotalSteps:  int(totalReplicas),
		CurrentStep: int(deployment.Status.ReadyReplicas),
		Messages:    []ProgressMessage{},
	}

	if totalReplicas > 0 {
		progress.Percentage = float64(deployment.Status.ReadyReplicas) / float64(totalReplicas) * 100
	}

	// Determine current stage
	if deployment.Status.UpdatedReplicas < totalReplicas {
		progress.CurrentStage = "Rolling out new version"
	} else if deployment.Status.ReadyReplicas < totalReplicas {
		progress.CurrentStage = "Waiting for pods to be ready"
	} else {
		progress.CurrentStage = "Deployment complete"
	}

	// Add pod status messages
	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodRunning {
			progress.Messages = append(progress.Messages, ProgressMessage{
				Timestamp: time.Now(),
				Level:     "info",
				Message:   fmt.Sprintf("Pod %s is %s", pod.Name, pod.Status.Phase),
				Component: pod.Name,
			})
		}
	}

	return progress
}

// getPods gets pods for the deployment
func (m *KubernetesMonitor) getPods(labels map[string]string) (*corev1.PodList, error) {
	labelSelector := ""
	for k, v := range labels {
		if labelSelector != "" {
			labelSelector += ","
		}
		labelSelector += fmt.Sprintf("%s=%s", k, v)
	}

	return m.client.CoreV1().Pods(m.namespace).List(
		context.Background(),
		metav1.ListOptions{
			LabelSelector: labelSelector,
		},
	)
}

// Helper functions

func isPodReady(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func getPodConditionMessage(pod *corev1.Pod) string {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status != corev1.ConditionTrue {
			return condition.Message
		}
	}
	return "Unknown"
}

func getContainerStateMessage(status *corev1.ContainerStatus) string {
	if status.State.Waiting != nil {
		return fmt.Sprintf("Waiting: %s", status.State.Waiting.Reason)
	}
	if status.State.Terminated != nil {
		return fmt.Sprintf("Terminated: %s", status.State.Terminated.Reason)
	}
	return "Unknown state"
}