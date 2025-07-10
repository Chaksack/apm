package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RoleChainManager manages complex role assumption chains with proper credential handling
type RoleChainManager struct {
	provider    *AWSProvider
	sessions    map[string]*ChainedSession
	mu          sync.RWMutex
	stopRefresh chan struct{}
}

// ChainedSession represents a session created through role chaining
type ChainedSession struct {
	ChainID     string              `json:"chainId"`
	Steps       []*RoleChainStep    `json:"steps"`
	Credentials []*ChainCredentials `json:"credentials"`
	FinalCreds  *Credentials        `json:"finalCredentials"`
	CreatedAt   time.Time           `json:"createdAt"`
	LastUsed    time.Time           `json:"lastUsed"`
	mu          sync.RWMutex
}

// ChainCredentials represents credentials at each step of the chain
type ChainCredentials struct {
	StepIndex   int          `json:"stepIndex"`
	RoleArn     string       `json:"roleArn"`
	Credentials *Credentials `json:"credentials"`
	AssumedAt   time.Time    `json:"assumedAt"`
}

// RoleChainConfig configures the role chain behavior
type RoleChainConfig struct {
	MaxSteps              int           `json:"maxSteps"`
	DefaultDuration       int           `json:"defaultDuration"`
	RefreshBeforeExpiry   time.Duration `json:"refreshBeforeExpiry"`
	EnableAutoRefresh     bool          `json:"enableAutoRefresh"`
	RetryAttempts         int           `json:"retryAttempts"`
	RetryDelay            time.Duration `json:"retryDelay"`
	ConcurrentAssumptions bool          `json:"concurrentAssumptions"`
}

// DefaultRoleChainConfig returns the default configuration
func DefaultRoleChainConfig() *RoleChainConfig {
	return &RoleChainConfig{
		MaxSteps:              5,
		DefaultDuration:       3600, // 1 hour
		RefreshBeforeExpiry:   5 * time.Minute,
		EnableAutoRefresh:     true,
		RetryAttempts:         3,
		RetryDelay:            time.Second,
		ConcurrentAssumptions: false,
	}
}

// NewRoleChainManager creates a new role chain manager
func NewRoleChainManager(provider *AWSProvider) *RoleChainManager {
	manager := &RoleChainManager{
		provider:    provider,
		sessions:    make(map[string]*ChainedSession),
		stopRefresh: make(chan struct{}),
	}

	// Start refresh worker if auto-refresh is enabled
	go manager.refreshWorker()

	return manager
}

// Close stops the refresh worker and cleans up resources
func (m *RoleChainManager) Close() {
	close(m.stopRefresh)
}

// AssumeRoleChain performs a sequential role assumption chain with improved credential handling
func (m *RoleChainManager) AssumeRoleChain(ctx context.Context, roleChain []*RoleChainStep, config *RoleChainConfig) (*ChainedSession, error) {
	if config == nil {
		config = DefaultRoleChainConfig()
	}

	if len(roleChain) == 0 {
		return nil, fmt.Errorf("role chain cannot be empty")
	}

	if len(roleChain) > config.MaxSteps {
		return nil, fmt.Errorf("role chain exceeds maximum steps (%d > %d)", len(roleChain), config.MaxSteps)
	}

	// Validate all steps first
	if err := m.validateChain(roleChain); err != nil {
		return nil, fmt.Errorf("chain validation failed: %w", err)
	}

	// Create session
	session := &ChainedSession{
		ChainID:     generateChainID(),
		Steps:       roleChain,
		Credentials: make([]*ChainCredentials, 0, len(roleChain)),
		CreatedAt:   time.Now(),
		LastUsed:    time.Now(),
	}

	// Execute the chain
	var currentCreds *Credentials
	for i, step := range roleChain {
		// Prepare options
		options := step.Options
		if options == nil {
			options = DefaultAssumeRoleOptions()
		}

		if step.SessionName != "" {
			options.SessionName = step.SessionName
		} else {
			options.SessionName = fmt.Sprintf("apm-chain-%s-step-%d", session.ChainID[:8], i+1)
		}

		if step.ExternalID != "" {
			options.ExternalID = step.ExternalID
		}

		if options.DurationSeconds == 0 {
			options.DurationSeconds = config.DefaultDuration
		}

		// Assume role with retry logic
		var creds *Credentials
		var err error

		for attempt := 0; attempt <= config.RetryAttempts; attempt++ {
			if i == 0 {
				// First step: use default credentials
				creds, err = m.assumeRoleWithDefaultCreds(ctx, step.RoleArn, options)
			} else {
				// Subsequent steps: use previous credentials
				creds, err = m.assumeRoleWithCreds(ctx, step.RoleArn, options, currentCreds)
			}

			if err == nil {
				break
			}

			if attempt < config.RetryAttempts {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(config.RetryDelay * time.Duration(attempt+1)):
					// Exponential backoff
				}
			}
		}

		if err != nil {
			// Rollback on failure
			m.rollbackChain(session, i)
			return nil, fmt.Errorf("failed at step %d (role: %s): %w", i+1, step.RoleArn, err)
		}

		// Store credentials for this step
		chainCreds := &ChainCredentials{
			StepIndex:   i,
			RoleArn:     step.RoleArn,
			Credentials: creds,
			AssumedAt:   time.Now(),
		}
		session.Credentials = append(session.Credentials, chainCreds)
		currentCreds = creds
	}

	// Set final credentials
	session.FinalCreds = currentCreds

	// Store session for management
	m.mu.Lock()
	m.sessions[session.ChainID] = session
	m.mu.Unlock()

	return session, nil
}

// assumeRoleWithDefaultCreds assumes a role using default AWS credentials
func (m *RoleChainManager) assumeRoleWithDefaultCreds(ctx context.Context, roleArn string, options *AssumeRoleOptions) (*Credentials, error) {
	return m.provider.AssumeRoleWithOptions(ctx, roleArn, options)
}

// assumeRoleWithCreds assumes a role using specific credentials
func (m *RoleChainManager) assumeRoleWithCreds(ctx context.Context, roleArn string, options *AssumeRoleOptions, creds *Credentials) (*Credentials, error) {
	// Build AWS CLI command with explicit credentials
	args := []string{
		"sts", "assume-role",
		"--role-arn", roleArn,
		"--role-session-name", options.SessionName,
	}

	if options.DurationSeconds > 0 {
		args = append(args, "--duration-seconds", strconv.Itoa(options.DurationSeconds))
	}

	if options.ExternalID != "" {
		args = append(args, "--external-id", options.ExternalID)
	}

	if options.MFASerialNumber != "" && options.MFATokenCode != "" {
		args = append(args, "--serial-number", options.MFASerialNumber, "--token-code", options.MFATokenCode)
	}

	if options.Policy != "" {
		args = append(args, "--policy", options.Policy)
	}

	for _, policyArn := range options.PolicyArns {
		args = append(args, "--policy-arns", policyArn)
	}

	if options.Region != "" {
		args = append(args, "--region", options.Region)
	}

	// Create command with credentials as environment variables
	cmd := exec.CommandContext(ctx, "aws", args...)
	cmd.Env = append(cmd.Env,
		fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", creds.AccessKey),
		fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", creds.SecretKey),
	)
	if creds.Token != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("AWS_SESSION_TOKEN=%s", creds.Token))
	}

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("failed to assume role %s: %s", roleArn, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to assume role %s: %w", roleArn, err)
	}

	var result struct {
		Credentials struct {
			AccessKeyId     string    `json:"AccessKeyId"`
			SecretAccessKey string    `json:"SecretAccessKey"`
			SessionToken    string    `json:"SessionToken"`
			Expiration      time.Time `json:"Expiration"`
		} `json:"Credentials"`
		AssumedRoleUser struct {
			AssumedRoleId string `json:"AssumedRoleId"`
			Arn           string `json:"Arn"`
		} `json:"AssumedRoleUser"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse assume role response: %w", err)
	}

	return &Credentials{
		Provider:   ProviderAWS,
		AuthMethod: AuthMethodIAMRole,
		AccessKey:  result.Credentials.AccessKeyId,
		SecretKey:  result.Credentials.SecretAccessKey,
		Token:      result.Credentials.SessionToken,
		Region:     options.Region,
		Account:    extractAccountFromArn(result.AssumedRoleUser.Arn),
		Expiry:     &result.Credentials.Expiration,
		Properties: map[string]string{
			"role_arn":         roleArn,
			"session_name":     options.SessionName,
			"assumed_role_id":  result.AssumedRoleUser.AssumedRoleId,
			"assumed_role_arn": result.AssumedRoleUser.Arn,
		},
	}, nil
}

// validateChain validates the role chain configuration
func (m *RoleChainManager) validateChain(chain []*RoleChainStep) error {
	seenRoles := make(map[string]bool)

	for i, step := range chain {
		if step.RoleArn == "" {
			return fmt.Errorf("step %d: role ARN cannot be empty", i+1)
		}

		// Validate ARN format
		if !strings.HasPrefix(step.RoleArn, "arn:aws:iam::") {
			return fmt.Errorf("step %d: invalid role ARN format: %s", i+1, step.RoleArn)
		}

		// Check for circular dependencies
		if seenRoles[step.RoleArn] {
			return fmt.Errorf("step %d: circular dependency detected for role %s", i+1, step.RoleArn)
		}
		seenRoles[step.RoleArn] = true

		// Validate external ID if specified
		if step.ExternalID != "" && len(step.ExternalID) < 2 {
			return fmt.Errorf("step %d: external ID too short", i+1)
		}
	}

	return nil
}

// rollbackChain performs cleanup for failed chain assumptions
func (m *RoleChainManager) rollbackChain(session *ChainedSession, failedStep int) {
	// Log the rollback
	if m.provider.config != nil && m.provider.config.DebugMode {
		fmt.Printf("Rolling back chain %s due to failure at step %d\n", session.ChainID, failedStep+1)
	}

	// Clear any stored credentials for this chain
	session.mu.Lock()
	session.Credentials = session.Credentials[:failedStep]
	session.mu.Unlock()
}

// GetSession retrieves a chained session by ID
func (m *RoleChainManager) GetSession(chainID string) (*ChainedSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[chainID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", chainID)
	}

	// Update last used time
	session.mu.Lock()
	session.LastUsed = time.Now()
	session.mu.Unlock()

	return session, nil
}

// RefreshChain refreshes all credentials in a chain
func (m *RoleChainManager) RefreshChain(ctx context.Context, chainID string) (*ChainedSession, error) {
	session, err := m.GetSession(chainID)
	if err != nil {
		return nil, err
	}

	// Re-execute the entire chain
	config := &RoleChainConfig{
		DefaultDuration:   3600,
		EnableAutoRefresh: false, // Disable auto-refresh for manual refresh
		RetryAttempts:     3,
		RetryDelay:        time.Second,
	}

	return m.AssumeRoleChain(ctx, session.Steps, config)
}

// refreshWorker periodically refreshes expiring sessions
func (m *RoleChainManager) refreshWorker() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopRefresh:
			return
		case <-ticker.C:
			m.refreshExpiringSessions()
		}
	}
}

// refreshExpiringSessions refreshes sessions that are near expiry
func (m *RoleChainManager) refreshExpiringSessions() {
	m.mu.RLock()
	sessions := make([]*ChainedSession, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}
	m.mu.RUnlock()

	ctx := context.Background()
	for _, session := range sessions {
		if session.FinalCreds != nil && session.FinalCreds.Expiry != nil {
			timeUntilExpiry := time.Until(*session.FinalCreds.Expiry)
			if timeUntilExpiry < 5*time.Minute && timeUntilExpiry > 0 {
				// Refresh the chain
				if _, err := m.RefreshChain(ctx, session.ChainID); err != nil {
					// Log error but continue with other sessions
					if m.provider.config != nil && m.provider.config.DebugMode {
						fmt.Printf("Failed to refresh chain %s: %v\n", session.ChainID, err)
					}
				}
			}
		}
	}
}

// RemoveSession removes a chained session
func (m *RoleChainManager) RemoveSession(chainID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, chainID)
}

// ListSessions returns all active chained sessions
func (m *RoleChainManager) ListSessions() []*ChainedSession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*ChainedSession, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

// ValidateChainStep validates that a specific step in the chain can be assumed
func (m *RoleChainManager) ValidateChainStep(ctx context.Context, step *RoleChainStep, previousCreds *Credentials) error {
	options := step.Options
	if options == nil {
		options = DefaultAssumeRoleOptions()
	}

	// Set a short duration for validation
	options.DurationSeconds = 900 // 15 minutes

	var err error
	if previousCreds == nil {
		_, err = m.assumeRoleWithDefaultCreds(ctx, step.RoleArn, options)
	} else {
		_, err = m.assumeRoleWithCreds(ctx, step.RoleArn, options, previousCreds)
	}

	return err
}

// generateChainID generates a unique chain ID
func generateChainID() string {
	return fmt.Sprintf("chain-%d-%d", time.Now().Unix(), time.Now().Nanosecond())
}

// extractAccountFromArn extracts the account ID from an ARN
func extractAccountFromArn(arn string) string {
	parts := strings.Split(arn, ":")
	if len(parts) >= 5 {
		return parts[4]
	}
	return ""
}

// Enhanced AssumeRoleChain method for AWSProvider that uses the RoleChainManager
func (p *AWSProvider) AssumeRoleChainEnhanced(ctx context.Context, roleChain []*RoleChainStep, config *RoleChainConfig) (*Credentials, error) {
	if p.crossAccountManager == nil {
		p.crossAccountManager = NewCrossAccountRoleManager(p)
	}

	chainManager := NewRoleChainManager(p)
	defer chainManager.Close()

	session, err := chainManager.AssumeRoleChain(ctx, roleChain, config)
	if err != nil {
		return nil, err
	}

	return session.FinalCreds, nil
}
