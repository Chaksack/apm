package cloud

import (
	"context"
	"io"
	"time"
)

// AuthManager interface for centralized authentication management
type AuthManager interface {
	// Authentication validation
	IsAuthenticated(ctx context.Context, provider Provider) (bool, error)
	Authenticate(ctx context.Context, provider Provider, options AuthOptions) error
	RefreshCredentials(ctx context.Context, provider Provider) error

	// Token management
	GetValidToken(ctx context.Context, provider Provider) (string, error)
	CacheToken(provider Provider, token string, expiry time.Time) error
	ClearCache(provider Provider) error

	// Session management
	CreateSession(ctx context.Context, provider Provider) (*AuthSession, error)
	ValidateSession(ctx context.Context, session *AuthSession) error
	RevokeSession(ctx context.Context, session *AuthSession) error
}

// ConfigManager interface for configuration storage and management
type ConfigManager interface {
	// Configuration persistence
	LoadConfig(provider Provider) (*ProviderConfig, error)
	SaveConfig(provider Provider, config *ProviderConfig) error
	DeleteConfig(provider Provider) error

	// Environment-specific configurations
	LoadEnvironmentConfig(provider Provider, environment string) (*ProviderConfig, error)
	SaveEnvironmentConfig(provider Provider, environment string, config *ProviderConfig) error
	ListEnvironments(provider Provider) ([]string, error)

	// Configuration validation
	ValidateConfig(config *ProviderConfig) (*ValidationResult, error)
	MergeConfigs(base, override *ProviderConfig) (*ProviderConfig, error)

	// Backup and restore
	BackupConfig(provider Provider) ([]byte, error)
	RestoreConfig(provider Provider, data []byte) error
}

// AuthOptions represents authentication configuration options
type AuthOptions struct {
	Method         AuthMethod        `json:"method"`
	Profile        string            `json:"profile,omitempty"`
	Region         string            `json:"region,omitempty"`
	AccessKey      string            `json:"access_key,omitempty"`
	SecretKey      string            `json:"secret_key,omitempty"`
	Token          string            `json:"token,omitempty"`
	ClientID       string            `json:"client_id,omitempty"`
	ClientSecret   string            `json:"client_secret,omitempty"`
	TenantID       string            `json:"tenant_id,omitempty"`
	ProjectID      string            `json:"project_id,omitempty"`
	ServiceAccount string            `json:"service_account,omitempty"`
	KeyFile        string            `json:"key_file,omitempty"`
	Interactive    bool              `json:"interactive"`
	Properties     map[string]string `json:"properties,omitempty"`
}

// AuthSession represents an authenticated session
type AuthSession struct {
	Provider   Provider          `json:"provider"`
	Method     AuthMethod        `json:"method"`
	Token      string            `json:"token"`
	Expiry     time.Time         `json:"expiry"`
	CreatedAt  time.Time         `json:"created_at"`
	LastUsed   time.Time         `json:"last_used"`
	Properties map[string]string `json:"properties,omitempty"`
}

// PushOptions represents options for pushing images
type PushOptions struct {
	Tags        []string          `json:"tags,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Platform    string            `json:"platform,omitempty"`
	Compress    bool              `json:"compress"`
	Progress    io.Writer         `json:"-"`
	Timeout     time.Duration     `json:"timeout,omitempty"`
}

// PullOptions represents options for pulling images
type PullOptions struct {
	Platform     string        `json:"platform,omitempty"`
	Progress     io.Writer     `json:"-"`
	Timeout      time.Duration `json:"timeout,omitempty"`
	VerifyDigest bool          `json:"verify_digest"`
}

// ImageInfo represents information about a container image
type ImageInfo struct {
	Repository  string            `json:"repository"`
	Tag         string            `json:"tag"`
	Digest      string            `json:"digest"`
	Size        int64             `json:"size"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Platform    string            `json:"platform,omitempty"`
}

// RepositoryOptions represents options for creating repositories
type RepositoryOptions struct {
	Description string            `json:"description,omitempty"`
	Private     bool              `json:"private"`
	ScanOnPush  bool              `json:"scan_on_push"`
	MutableTags bool              `json:"mutable_tags"`
	Labels      map[string]string `json:"labels,omitempty"`
	Lifecycle   *LifecyclePolicy  `json:"lifecycle,omitempty"`
}

// RepositoryInfo represents information about a repository
type RepositoryInfo struct {
	Name        string            `json:"name"`
	URI         string            `json:"uri"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	ImageCount  int               `json:"image_count"`
	Size        int64             `json:"size"`
	Private     bool              `json:"private"`
	Labels      map[string]string `json:"labels,omitempty"`
	Description string            `json:"description,omitempty"`
}

// LifecyclePolicy represents image lifecycle management policy
type LifecyclePolicy struct {
	Rules []LifecycleRule `json:"rules"`
}

// LifecycleRule represents a single lifecycle rule
type LifecycleRule struct {
	Selection   ImageSelection `json:"selection"`
	Action      string         `json:"action"` // delete, expire
	Description string         `json:"description,omitempty"`
}

// ImageSelection represents criteria for selecting images
type ImageSelection struct {
	TagStatus     string   `json:"tag_status"` // tagged, untagged, any
	CountType     string   `json:"count_type"` // imageCountMoreThan, sinceImagePushed
	CountNumber   int      `json:"count_number"`
	CountUnit     string   `json:"count_unit,omitempty"` // days
	TagPrefixList []string `json:"tag_prefix_list,omitempty"`
}

// ScanResult represents image vulnerability scan results
type ScanResult struct {
	ScanStatus  string               `json:"scan_status"`
	CompletedAt time.Time            `json:"completed_at,omitempty"`
	Findings    []SecurityFinding    `json:"findings,omitempty"`
	Summary     VulnerabilitySummary `json:"summary"`
}

// SecurityFinding represents a security vulnerability
type SecurityFinding struct {
	Severity    string  `json:"severity"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	URI         string  `json:"uri,omitempty"`
	CVEID       string  `json:"cve_id,omitempty"`
	Score       float64 `json:"score,omitempty"`
}

// VulnerabilitySummary represents a summary of vulnerabilities
type VulnerabilitySummary struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
	Info     int `json:"info"`
	Total    int `json:"total"`
}

// NodeInfo represents information about a cluster node
type NodeInfo struct {
	Name        string            `json:"name"`
	Status      string            `json:"status"`
	Roles       []string          `json:"roles"`
	Version     string            `json:"version"`
	OS          string            `json:"os"`
	Arch        string            `json:"arch"`
	CreatedAt   time.Time         `json:"created_at"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Capacity    ResourceList      `json:"capacity"`
	Allocatable ResourceList      `json:"allocatable"`
}

// ResourceList represents compute resources
type ResourceList struct {
	CPU     string `json:"cpu"`
	Memory  string `json:"memory"`
	Pods    string `json:"pods"`
	Storage string `json:"storage,omitempty"`
}

// WorkloadInfo represents information about a workload
type WorkloadInfo struct {
	Name       string            `json:"name"`
	Kind       string            `json:"kind"`
	Namespace  string            `json:"namespace"`
	Status     string            `json:"status"`
	Replicas   int32             `json:"replicas"`
	Ready      int32             `json:"ready"`
	CreatedAt  time.Time         `json:"created_at"`
	Labels     map[string]string `json:"labels,omitempty"`
	Conditions []Condition       `json:"conditions,omitempty"`
}

// ServiceInfo represents information about a service
type ServiceInfo struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Type        string            `json:"type"`
	ClusterIP   string            `json:"cluster_ip"`
	ExternalIP  string            `json:"external_ip,omitempty"`
	Ports       []ServicePort     `json:"ports"`
	CreatedAt   time.Time         `json:"created_at"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ServicePort represents a service port
type ServicePort struct {
	Name       string `json:"name,omitempty"`
	Protocol   string `json:"protocol"`
	Port       int32  `json:"port"`
	TargetPort string `json:"target_port"`
	NodePort   int32  `json:"node_port,omitempty"`
}

// ExposeOptions represents options for exposing services
type ExposeOptions struct {
	Type         string               `json:"type"` // ClusterIP, NodePort, LoadBalancer
	Port         int32                `json:"port"`
	TargetPort   string               `json:"target_port"`
	Protocol     string               `json:"protocol"` // TCP, UDP
	Labels       map[string]string    `json:"labels,omitempty"`
	Annotations  map[string]string    `json:"annotations,omitempty"`
	LoadBalancer *LoadBalancerOptions `json:"load_balancer,omitempty"`
}

// LoadBalancerOptions represents load balancer specific options
type LoadBalancerOptions struct {
	SourceRanges []string          `json:"source_ranges,omitempty"`
	Annotations  map[string]string `json:"annotations,omitempty"`
}

// HPAInfo represents information about Horizontal Pod Autoscaler
type HPAInfo struct {
	Name        string    `json:"name"`
	Namespace   string    `json:"namespace"`
	TargetRef   string    `json:"target_ref"`
	MinReplicas int32     `json:"min_replicas"`
	MaxReplicas int32     `json:"max_replicas"`
	Current     int32     `json:"current_replicas"`
	Desired     int32     `json:"desired_replicas"`
	CreatedAt   time.Time `json:"created_at"`
}

// ClusterMetrics represents cluster-level metrics
type ClusterMetrics struct {
	Nodes     NodeMetrics     `json:"nodes"`
	Pods      PodMetrics      `json:"pods"`
	Resources ResourceMetrics `json:"resources"`
	Network   NetworkMetrics  `json:"network"`
	Timestamp time.Time       `json:"timestamp"`
}

// NodeMetrics represents node-level metrics
type NodeMetrics struct {
	Total    int `json:"total"`
	Ready    int `json:"ready"`
	NotReady int `json:"not_ready"`
	Unknown  int `json:"unknown"`
}

// PodMetrics represents pod-level metrics
type PodMetrics struct {
	Total     int `json:"total"`
	Running   int `json:"running"`
	Pending   int `json:"pending"`
	Failed    int `json:"failed"`
	Succeeded int `json:"succeeded"`
}

// ResourceMetrics represents resource usage metrics
type ResourceMetrics struct {
	CPU     ResourceUsage `json:"cpu"`
	Memory  ResourceUsage `json:"memory"`
	Storage ResourceUsage `json:"storage"`
}

// ResourceUsage represents usage statistics for a resource
type ResourceUsage struct {
	Used    string  `json:"used"`
	Total   string  `json:"total"`
	Percent float64 `json:"percent"`
}

// NetworkMetrics represents network-level metrics
type NetworkMetrics struct {
	ServicesTotal   int `json:"services_total"`
	IngressTotal    int `json:"ingress_total"`
	NetworkPolicies int `json:"network_policies"`
}

// LogOptions represents options for retrieving logs
type LogOptions struct {
	Follow     bool          `json:"follow"`
	Previous   bool          `json:"previous"`
	Since      time.Time     `json:"since,omitempty"`
	SinceTime  *time.Time    `json:"since_time,omitempty"`
	Timestamps bool          `json:"timestamps"`
	TailLines  *int64        `json:"tail_lines,omitempty"`
	LimitBytes *int64        `json:"limit_bytes,omitempty"`
	Timeout    time.Duration `json:"timeout,omitempty"`
}

// Condition represents a condition of a Kubernetes resource
type Condition struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	LastTransitionTime time.Time `json:"last_transition_time"`
	Reason             string    `json:"reason,omitempty"`
	Message            string    `json:"message,omitempty"`
}
