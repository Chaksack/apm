package sidecar

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Type represents the type of sidecar
type Type string

const (
	TypeMetrics Type = "metrics"
	TypeLogging Type = "logging"
	TypeTracing Type = "tracing"
	TypeMesh    Type = "mesh"
)

// Config represents sidecar configuration
type Config struct {
	Type            Type
	Name            string
	Image           string
	Version         string
	Ports           []corev1.ContainerPort
	Environment     []corev1.EnvVar
	VolumeMounts    []corev1.VolumeMount
	Resources       corev1.ResourceRequirements
	LivenessProbe   *corev1.Probe
	ReadinessProbe  *corev1.Probe
	SecurityContext *corev1.SecurityContext
	Args            []string
	Command         []string
}

// Injector handles sidecar injection
type Injector struct {
	configs  map[Type]*Config
	defaults InjectorDefaults
}

// InjectorDefaults contains default settings
type InjectorDefaults struct {
	DefaultCPURequest    string
	DefaultMemoryRequest string
	DefaultCPULimit      string
	DefaultMemoryLimit   string
	ImagePullPolicy      corev1.PullPolicy
	RunAsNonRoot         bool
	ReadOnlyRootFS       bool
}

// NewInjector creates a new sidecar injector
func NewInjector() *Injector {
	return &Injector{
		configs: make(map[Type]*Config),
		defaults: InjectorDefaults{
			DefaultCPURequest:    "10m",
			DefaultMemoryRequest: "32Mi",
			DefaultCPULimit:      "100m",
			DefaultMemoryLimit:   "128Mi",
			ImagePullPolicy:      corev1.PullIfNotPresent,
			RunAsNonRoot:         true,
			ReadOnlyRootFS:       true,
		},
	}
}

// RegisterSidecar registers a sidecar configuration
func (i *Injector) RegisterSidecar(config *Config) {
	i.configs[config.Type] = config
}

// InjectSidecar injects a sidecar into a pod spec
func (i *Injector) InjectSidecar(pod *corev1.PodSpec, sidecarType Type, overrides map[string]string) error {
	config, exists := i.configs[sidecarType]
	if !exists {
		// Use default configuration
		config = i.getDefaultConfig(sidecarType)
	}

	// Check if sidecar already exists
	for _, container := range pod.Containers {
		if container.Name == config.Name {
			return fmt.Errorf("sidecar %s already exists", config.Name)
		}
	}

	// Create sidecar container
	sidecar := i.createContainer(config, overrides)

	// Add volumes if needed
	i.addVolumes(pod, config)

	// Inject sidecar
	pod.Containers = append(pod.Containers, sidecar)

	return nil
}

// RemoveSidecar removes a sidecar from a pod spec
func (i *Injector) RemoveSidecar(pod *corev1.PodSpec, sidecarName string) error {
	found := false
	containers := []corev1.Container{}

	for _, container := range pod.Containers {
		if container.Name != sidecarName {
			containers = append(containers, container)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("sidecar %s not found", sidecarName)
	}

	pod.Containers = containers
	return nil
}

// UpdateSidecar updates an existing sidecar
func (i *Injector) UpdateSidecar(pod *corev1.PodSpec, config *Config) error {
	for idx, container := range pod.Containers {
		if container.Name == config.Name {
			pod.Containers[idx] = i.createContainer(config, nil)
			return nil
		}
	}
	return fmt.Errorf("sidecar %s not found", config.Name)
}

// createContainer creates a container from config
func (i *Injector) createContainer(config *Config, overrides map[string]string) corev1.Container {
	container := corev1.Container{
		Name:            config.Name,
		Image:           i.getImage(config, overrides),
		ImagePullPolicy: i.defaults.ImagePullPolicy,
		Ports:           config.Ports,
		Env:             i.mergeEnv(config.Environment, overrides),
		VolumeMounts:    config.VolumeMounts,
		Resources:       i.getResources(config, overrides),
	}

	if len(config.Args) > 0 {
		container.Args = config.Args
	}
	if len(config.Command) > 0 {
		container.Command = config.Command
	}

	// Set probes
	if config.LivenessProbe != nil {
		container.LivenessProbe = config.LivenessProbe
	}
	if config.ReadinessProbe != nil {
		container.ReadinessProbe = config.ReadinessProbe
	}

	// Set security context
	container.SecurityContext = i.getSecurityContext(config)

	return container
}

// getImage returns the image with overrides applied
func (i *Injector) getImage(config *Config, overrides map[string]string) string {
	if override, exists := overrides["image"]; exists {
		return override
	}
	if config.Version != "" {
		return fmt.Sprintf("%s:%s", config.Image, config.Version)
	}
	return config.Image
}

// mergeEnv merges environment variables with overrides
func (i *Injector) mergeEnv(env []corev1.EnvVar, overrides map[string]string) []corev1.EnvVar {
	result := make([]corev1.EnvVar, len(env))
	copy(result, env)

	for key, value := range overrides {
		if strings.HasPrefix(key, "env.") {
			envKey := strings.TrimPrefix(key, "env.")
			found := false
			for idx, e := range result {
				if e.Name == envKey {
					result[idx].Value = value
					found = true
					break
				}
			}
			if !found {
				result = append(result, corev1.EnvVar{
					Name:  envKey,
					Value: value,
				})
			}
		}
	}

	return result
}

// getResources returns resource requirements with overrides
func (i *Injector) getResources(config *Config, overrides map[string]string) corev1.ResourceRequirements {
	resources := config.Resources
	if resources.Requests == nil {
		resources.Requests = corev1.ResourceList{}
	}
	if resources.Limits == nil {
		resources.Limits = corev1.ResourceList{}
	}

	// Apply defaults if not set
	if _, exists := resources.Requests[corev1.ResourceCPU]; !exists {
		resources.Requests[corev1.ResourceCPU] = resource.MustParse(i.defaults.DefaultCPURequest)
	}
	if _, exists := resources.Requests[corev1.ResourceMemory]; !exists {
		resources.Requests[corev1.ResourceMemory] = resource.MustParse(i.defaults.DefaultMemoryRequest)
	}
	if _, exists := resources.Limits[corev1.ResourceCPU]; !exists {
		resources.Limits[corev1.ResourceCPU] = resource.MustParse(i.defaults.DefaultCPULimit)
	}
	if _, exists := resources.Limits[corev1.ResourceMemory]; !exists {
		resources.Limits[corev1.ResourceMemory] = resource.MustParse(i.defaults.DefaultMemoryLimit)
	}

	// Apply overrides
	if cpu, exists := overrides["resources.requests.cpu"]; exists {
		resources.Requests[corev1.ResourceCPU] = resource.MustParse(cpu)
	}
	if memory, exists := overrides["resources.requests.memory"]; exists {
		resources.Requests[corev1.ResourceMemory] = resource.MustParse(memory)
	}
	if cpu, exists := overrides["resources.limits.cpu"]; exists {
		resources.Limits[corev1.ResourceCPU] = resource.MustParse(cpu)
	}
	if memory, exists := overrides["resources.limits.memory"]; exists {
		resources.Limits[corev1.ResourceMemory] = resource.MustParse(memory)
	}

	return resources
}

// getSecurityContext returns security context with defaults
func (i *Injector) getSecurityContext(config *Config) *corev1.SecurityContext {
	if config.SecurityContext != nil {
		return config.SecurityContext
	}

	runAsNonRoot := i.defaults.RunAsNonRoot
	readOnlyRootFS := i.defaults.ReadOnlyRootFS
	
	return &corev1.SecurityContext{
		RunAsNonRoot:             &runAsNonRoot,
		ReadOnlyRootFilesystem:   &readOnlyRootFS,
		AllowPrivilegeEscalation: &[]bool{false}[0],
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
	}
}

// addVolumes adds required volumes to pod spec
func (i *Injector) addVolumes(pod *corev1.PodSpec, config *Config) {
	requiredVolumes := i.getRequiredVolumes(config.Type)
	
	for _, reqVol := range requiredVolumes {
		found := false
		for _, vol := range pod.Volumes {
			if vol.Name == reqVol.Name {
				found = true
				break
			}
		}
		if !found {
			pod.Volumes = append(pod.Volumes, reqVol)
		}
	}
}

// getRequiredVolumes returns volumes required by sidecar type
func (i *Injector) getRequiredVolumes(sidecarType Type) []corev1.Volume {
	switch sidecarType {
	case TypeMetrics:
		return []corev1.Volume{
			{
				Name: "proc",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/proc",
					},
				},
			},
			{
				Name: "sys",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/sys",
					},
				},
			},
		}
	case TypeLogging:
		return []corev1.Volume{
			{
				Name: "varlog",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/var/log",
					},
				},
			},
			{
				Name: "varlibdockercontainers",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/var/lib/docker/containers",
					},
				},
			},
		}
	default:
		return []corev1.Volume{}
	}
}

// getDefaultConfig returns default configuration for sidecar type
func (i *Injector) getDefaultConfig(sidecarType Type) *Config {
	switch sidecarType {
	case TypeMetrics:
		return i.getPrometheusExporterConfig()
	case TypeLogging:
		return i.getFluentBitConfig()
	case TypeTracing:
		return i.getOTelCollectorConfig()
	default:
		return &Config{
			Type: sidecarType,
			Name: string(sidecarType) + "-sidecar",
		}
	}
}

// getPrometheusExporterConfig returns Prometheus exporter configuration
func (i *Injector) getPrometheusExporterConfig() *Config {
	return &Config{
		Type:    TypeMetrics,
		Name:    "prometheus-exporter",
		Image:   "prom/node-exporter",
		Version: "v1.5.0",
		Ports: []corev1.ContainerPort{
			{
				Name:          "metrics",
				ContainerPort: 9100,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Args: []string{
			"--path.procfs=/host/proc",
			"--path.sysfs=/host/sys",
			"--path.rootfs=/host/root",
			"--collector.filesystem.mount-points-exclude=^/(dev|proc|sys|var/lib/docker/.+)($|/)",
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "proc",
				MountPath: "/host/proc",
				ReadOnly:  true,
			},
			{
				Name:      "sys",
				MountPath: "/host/sys",
				ReadOnly:  true,
			},
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/",
					Port: intstr.FromInt(9100),
				},
			},
			InitialDelaySeconds: 30,
			PeriodSeconds:       30,
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/",
					Port: intstr.FromInt(9100),
				},
			},
			InitialDelaySeconds: 5,
			PeriodSeconds:       10,
		},
	}
}

// getFluentBitConfig returns Fluent Bit configuration
func (i *Injector) getFluentBitConfig() *Config {
	return &Config{
		Type:    TypeLogging,
		Name:    "fluent-bit",
		Image:   "fluent/fluent-bit",
		Version: "2.1.0",
		Environment: []corev1.EnvVar{
			{
				Name:  "FLUENT_LOKI_URL",
				Value: "http://loki:3100/loki/api/v1/push",
			},
			{
				Name: "K8S_NAMESPACE",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.namespace",
					},
				},
			},
			{
				Name: "K8S_POD_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "varlog",
				MountPath: "/var/log",
				ReadOnly:  true,
			},
			{
				Name:      "varlibdockercontainers",
				MountPath: "/var/lib/docker/containers",
				ReadOnly:  true,
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("20m"),
				corev1.ResourceMemory: resource.MustParse("64Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("256Mi"),
			},
		},
	}
}

// getOTelCollectorConfig returns OpenTelemetry Collector configuration
func (i *Injector) getOTelCollectorConfig() *Config {
	return &Config{
		Type:    TypeTracing,
		Name:    "otel-agent",
		Image:   "otel/opentelemetry-collector",
		Version: "0.88.0",
		Args: []string{
			"--config=/conf/otel-agent-config.yaml",
		},
		Environment: []corev1.EnvVar{
			{
				Name:  "OTEL_RESOURCE_ATTRIBUTES",
				Value: "service.name=$(K8S_POD_NAME),k8s.namespace.name=$(K8S_NAMESPACE)",
			},
			{
				Name: "K8S_NAMESPACE",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.namespace",
					},
				},
			},
			{
				Name: "K8S_POD_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          "otlp-grpc",
				ContainerPort: 4317,
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "otlp-http",
				ContainerPort: 4318,
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "jaeger-compact",
				ContainerPort: 6831,
				Protocol:      corev1.ProtocolUDP,
			},
			{
				Name:          "jaeger-grpc",
				ContainerPort: 14250,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("50m"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("200m"),
				corev1.ResourceMemory: resource.MustParse("512Mi"),
			},
		},
	}
}