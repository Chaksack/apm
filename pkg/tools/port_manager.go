package tools

import (
	"fmt"
	"net"
	"sync"
)

// PortRegistry defines default ports for each tool
var PortRegistry = map[ToolType]PortConfig{
	ToolTypePrometheus: {
		Default:      9090,
		Alternatives: []int{9091, 9092},
		Protocol:     "tcp",
		Description:  "Prometheus metrics server",
	},
	ToolTypeGrafana: {
		Default:      3000,
		Alternatives: []int{3001, 3002, 3003},
		Protocol:     "tcp",
		Description:  "Grafana web UI",
	},
	ToolTypeJaeger: {
		Default:      16686,
		Alternatives: []int{16687, 16688},
		Protocol:     "tcp",
		Description:  "Jaeger UI",
	},
	ToolTypeLoki: {
		Default:      3100,
		Alternatives: []int{3101, 3102},
		Protocol:     "tcp",
		Description:  "Loki HTTP API",
	},
	ToolTypeAlertManager: {
		Default:      9093,
		Alternatives: []int{9094, 9095},
		Protocol:     "tcp",
		Description:  "AlertManager web UI",
	},
}

// AdditionalPorts defines additional ports used by tools
var AdditionalPorts = map[ToolType]map[string]PortConfig{
	ToolTypeJaeger: {
		"collector-grpc": {
			Default:     14250,
			Protocol:    "tcp",
			Description: "Jaeger collector gRPC",
		},
		"collector-http": {
			Default:     14268,
			Protocol:    "tcp",
			Description: "Jaeger collector HTTP",
		},
		"agent-compact": {
			Default:     6831,
			Protocol:    "udp",
			Description: "Jaeger agent compact thrift",
		},
		"agent-binary": {
			Default:     6832,
			Protocol:    "udp",
			Description: "Jaeger agent binary thrift",
		},
		"otlp-grpc": {
			Default:     4317,
			Protocol:    "tcp",
			Description: "OTLP gRPC receiver",
		},
		"otlp-http": {
			Default:     4318,
			Protocol:    "tcp",
			Description: "OTLP HTTP receiver",
		},
	},
	ToolTypeLoki: {
		"grpc": {
			Default:     9096,
			Protocol:    "tcp",
			Description: "Loki gRPC API",
		},
	},
	ToolTypeAlertManager: {
		"cluster": {
			Default:     9094,
			Protocol:    "tcp",
			Description: "AlertManager cluster",
		},
	},
}

// PortManager handles port allocation and conflict resolution
type PortManager struct {
	registry  map[ToolType]PortConfig
	allocated map[int]string
	mu        sync.RWMutex
}

// NewPortManager creates a new port manager
func NewPortManager() *PortManager {
	return &PortManager{
		registry:  PortRegistry,
		allocated: make(map[int]string),
	}
}

// AllocatePort finds an available port for a tool
func (pm *PortManager) AllocatePort(toolType ToolType) (int, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	config, exists := pm.registry[toolType]
	if !exists {
		return 0, fmt.Errorf("unknown tool type: %s", toolType)
	}

	// Try default port first
	if pm.isPortAvailable(config.Default) {
		pm.allocated[config.Default] = string(toolType)
		return config.Default, nil
	}

	// Try alternative ports
	for _, port := range config.Alternatives {
		if pm.isPortAvailable(port) {
			pm.allocated[port] = string(toolType)
			return port, nil
		}
	}

	// Find next available port in range
	return pm.findNextAvailablePort(config.Default)
}

// AllocateAdditionalPort allocates additional ports for tools that need multiple ports
func (pm *PortManager) AllocateAdditionalPort(toolType ToolType, portName string) (int, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	additionalPorts, exists := AdditionalPorts[toolType]
	if !exists {
		return 0, fmt.Errorf("no additional ports defined for tool type: %s", toolType)
	}

	config, exists := additionalPorts[portName]
	if !exists {
		return 0, fmt.Errorf("unknown port name %s for tool type: %s", portName, toolType)
	}

	if pm.isPortAvailable(config.Default) {
		pm.allocated[config.Default] = fmt.Sprintf("%s-%s", toolType, portName)
		return config.Default, nil
	}

	// Find alternative port
	return pm.findNextAvailablePort(config.Default)
}

// ReleasePort releases a previously allocated port
func (pm *PortManager) ReleasePort(port int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delete(pm.allocated, port)
}

// GetAllocatedPorts returns all currently allocated ports
func (pm *PortManager) GetAllocatedPorts() map[int]string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make(map[int]string)
	for port, tool := range pm.allocated {
		result[port] = tool
	}
	return result
}

// IsPortAllocated checks if a port is already allocated
func (pm *PortManager) IsPortAllocated(port int) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	_, allocated := pm.allocated[port]
	return allocated
}

// isPortAvailable checks if a port is available for use
func (pm *PortManager) isPortAvailable(port int) bool {
	// Check internal registry
	if _, allocated := pm.allocated[port]; allocated {
		return false
	}

	// Check if port is actually in use
	return isPortFree(port)
}

// findNextAvailablePort finds the next available port starting from basePort
func (pm *PortManager) findNextAvailablePort(basePort int) (int, error) {
	// Search in increments of 10 to avoid conflicting with related services
	for i := 1; i <= 100; i++ {
		port := basePort + (i * 10)
		if port > 65535 {
			break
		}
		if pm.isPortAvailable(port) {
			pm.allocated[port] = "dynamic"
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available ports found")
}

// isPortFree checks if a port is free on the system
func isPortFree(port int) bool {
	// Try TCP
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	listener.Close()

	// Try UDP for completeness
	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	udpListener, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return false
	}
	udpListener.Close()

	return true
}

// PortConflictResolver resolves port conflicts between tools
type PortConflictResolver struct {
	manager *PortManager
}

// NewPortConflictResolver creates a new port conflict resolver
func NewPortConflictResolver(manager *PortManager) *PortConflictResolver {
	return &PortConflictResolver{
		manager: manager,
	}
}

// ResolveConflicts checks for and resolves port conflicts
func (pcr *PortConflictResolver) ResolveConflicts(tools []*Tool) error {
	// Group tools by port
	portToTools := make(map[int][]*Tool)
	for _, tool := range tools {
		portToTools[tool.Port] = append(portToTools[tool.Port], tool)
	}

	// Resolve conflicts
	for port, conflictingTools := range portToTools {
		if len(conflictingTools) <= 1 {
			continue
		}

		// Keep the first tool on the original port
		for i := 1; i < len(conflictingTools); i++ {
			tool := conflictingTools[i]
			newPort, err := pcr.manager.AllocatePort(tool.Type)
			if err != nil {
				return fmt.Errorf("failed to resolve port conflict for %s: %w", tool.Name, err)
			}
			tool.Port = newPort
			tool.Endpoint = fmt.Sprintf("http://%s:%d", getHost(tool.Endpoint), newPort)
		}
	}

	return nil
}

// getHost extracts host from an endpoint URL
func getHost(endpoint string) string {
	// Simple extraction, assumes http://host:port format
	if len(endpoint) > 7 {
		start := 7 // len("http://")
		if endpoint[:8] == "https://" {
			start = 8
		}
		hostPort := endpoint[start:]
		if colonIdx := findLastIndex(hostPort, ":"); colonIdx != -1 {
			return hostPort[:colonIdx]
		}
		return hostPort
	}
	return "localhost"
}

// findLastIndex finds the last occurrence of a substring
func findLastIndex(s, substr string) int {
	for i := len(s) - len(substr); i >= 0; i-- {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}