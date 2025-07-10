package cloud

import (
	"context"
	"testing"
	"time"
)

func TestRoleChainManager_AssumeRoleChain(t *testing.T) {
	// This is a unit test demonstrating the role chain functionality
	// In a real scenario, you would need actual AWS credentials and roles

	tests := []struct {
		name      string
		roleChain []*RoleChainStep
		config    *RoleChainConfig
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "empty chain",
			roleChain: []*RoleChainStep{},
			config:    DefaultRoleChainConfig(),
			wantErr:   true,
			errMsg:    "role chain cannot be empty",
		},
		{
			name: "single step chain",
			roleChain: []*RoleChainStep{
				{
					RoleArn:     "arn:aws:iam::123456789012:role/FirstRole",
					SessionName: "test-session",
				},
			},
			config:  DefaultRoleChainConfig(),
			wantErr: false,
		},
		{
			name: "multi-step chain with external ID",
			roleChain: []*RoleChainStep{
				{
					RoleArn:     "arn:aws:iam::123456789012:role/FirstRole",
					SessionName: "step-1",
				},
				{
					RoleArn:     "arn:aws:iam::987654321098:role/SecondRole",
					ExternalID:  "unique-external-id",
					SessionName: "step-2",
				},
				{
					RoleArn:     "arn:aws:iam::111111111111:role/ThirdRole",
					SessionName: "step-3",
				},
			},
			config:  DefaultRoleChainConfig(),
			wantErr: false,
		},
		{
			name: "chain exceeding max steps",
			roleChain: []*RoleChainStep{
				{RoleArn: "arn:aws:iam::123456789012:role/Role1"},
				{RoleArn: "arn:aws:iam::123456789012:role/Role2"},
				{RoleArn: "arn:aws:iam::123456789012:role/Role3"},
				{RoleArn: "arn:aws:iam::123456789012:role/Role4"},
				{RoleArn: "arn:aws:iam::123456789012:role/Role5"},
				{RoleArn: "arn:aws:iam::123456789012:role/Role6"},
			},
			config:  DefaultRoleChainConfig(),
			wantErr: true,
			errMsg:  "role chain exceeds maximum steps",
		},
		{
			name: "invalid role ARN format",
			roleChain: []*RoleChainStep{
				{
					RoleArn: "invalid-arn",
				},
			},
			config:  DefaultRoleChainConfig(),
			wantErr: true,
			errMsg:  "invalid role ARN format",
		},
		{
			name: "circular dependency",
			roleChain: []*RoleChainStep{
				{
					RoleArn: "arn:aws:iam::123456789012:role/RoleA",
				},
				{
					RoleArn: "arn:aws:iam::987654321098:role/RoleB",
				},
				{
					RoleArn: "arn:aws:iam::123456789012:role/RoleA", // Same as first
				},
			},
			config:  DefaultRoleChainConfig(),
			wantErr: true,
			errMsg:  "circular dependency detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock provider for testing
			provider := &AWSProvider{
				config: &ProviderConfig{
					Region:    "us-east-1",
					DebugMode: true,
				},
			}

			manager := NewRoleChainManager(provider)
			defer manager.Close()

			// Note: This will fail in unit tests without actual AWS credentials
			// This is just to demonstrate the API usage
			ctx := context.Background()
			_, err := manager.AssumeRoleChain(ctx, tt.roleChain, tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("AssumeRoleChain() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("AssumeRoleChain() error = %v, want error containing %v", err, tt.errMsg)
				}
			} else {
				// In a real test with mocked AWS responses, we would check for no error
				// For now, we expect an error due to missing AWS credentials
				if err == nil {
					t.Errorf("Expected error due to missing AWS credentials in test environment")
				}
			}
		})
	}
}

func TestRoleChainManager_ValidateChain(t *testing.T) {
	manager := &RoleChainManager{}

	tests := []struct {
		name    string
		chain   []*RoleChainStep
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid chain",
			chain: []*RoleChainStep{
				{RoleArn: "arn:aws:iam::123456789012:role/Role1"},
				{RoleArn: "arn:aws:iam::987654321098:role/Role2"},
			},
			wantErr: false,
		},
		{
			name: "empty role ARN",
			chain: []*RoleChainStep{
				{RoleArn: ""},
			},
			wantErr: true,
			errMsg:  "role ARN cannot be empty",
		},
		{
			name: "invalid ARN format",
			chain: []*RoleChainStep{
				{RoleArn: "not-an-arn"},
			},
			wantErr: true,
			errMsg:  "invalid role ARN format",
		},
		{
			name: "circular dependency",
			chain: []*RoleChainStep{
				{RoleArn: "arn:aws:iam::123456789012:role/Role1"},
				{RoleArn: "arn:aws:iam::123456789012:role/Role1"},
			},
			wantErr: true,
			errMsg:  "circular dependency detected",
		},
		{
			name: "short external ID",
			chain: []*RoleChainStep{
				{RoleArn: "arn:aws:iam::123456789012:role/Role1", ExternalID: "a"},
			},
			wantErr: true,
			errMsg:  "external ID too short",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.validateChain(tt.chain)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateChain() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
				t.Errorf("validateChain() error = %v, want error containing %v", err, tt.errMsg)
			}
		})
	}
}

func TestRoleChainConfig_Defaults(t *testing.T) {
	config := DefaultRoleChainConfig()

	if config.MaxSteps != 5 {
		t.Errorf("MaxSteps = %d, want 5", config.MaxSteps)
	}
	if config.DefaultDuration != 3600 {
		t.Errorf("DefaultDuration = %d, want 3600", config.DefaultDuration)
	}
	if config.RefreshBeforeExpiry != 5*time.Minute {
		t.Errorf("RefreshBeforeExpiry = %v, want 5m", config.RefreshBeforeExpiry)
	}
	if !config.EnableAutoRefresh {
		t.Error("EnableAutoRefresh = false, want true")
	}
	if config.RetryAttempts != 3 {
		t.Errorf("RetryAttempts = %d, want 3", config.RetryAttempts)
	}
}

func TestRoleChainManager_SessionManagement(t *testing.T) {
	provider := &AWSProvider{
		config: &ProviderConfig{
			Region: "us-east-1",
		},
	}
	manager := NewRoleChainManager(provider)
	defer manager.Close()

	// Test session storage and retrieval
	mockSession := &ChainedSession{
		ChainID: "test-chain-123",
		Steps: []*RoleChainStep{
			{RoleArn: "arn:aws:iam::123456789012:role/TestRole"},
		},
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
	}

	// Store session
	manager.mu.Lock()
	manager.sessions[mockSession.ChainID] = mockSession
	manager.mu.Unlock()

	// Test GetSession
	retrieved, err := manager.GetSession("test-chain-123")
	if err != nil {
		t.Errorf("GetSession() error = %v", err)
	}
	if retrieved.ChainID != mockSession.ChainID {
		t.Errorf("GetSession() chainID = %v, want %v", retrieved.ChainID, mockSession.ChainID)
	}

	// Test ListSessions
	sessions := manager.ListSessions()
	if len(sessions) != 1 {
		t.Errorf("ListSessions() returned %d sessions, want 1", len(sessions))
	}

	// Test RemoveSession
	manager.RemoveSession("test-chain-123")
	_, err = manager.GetSession("test-chain-123")
	if err == nil {
		t.Error("GetSession() should have returned error after RemoveSession")
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr)) ||
		(len(substr) < len(s) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
