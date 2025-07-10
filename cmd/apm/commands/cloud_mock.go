package commands

// Mock cloud types for demonstration purposes
// In production, these would be imported from pkg/cloud

type Registry struct {
	Name   string
	URL    string
	Region string
	Type   string
}

type Cluster struct {
	Name      string
	Region    string
	Type      string
	Status    string
	NodeCount int
}

type Manager struct{}

func NewManager() (*Manager, error) {
	return &Manager{}, nil
}
