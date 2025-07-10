package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// MFADevice represents an MFA device configuration
type MFADevice struct {
	SerialNumber string    `json:"serialNumber"`
	Type         string    `json:"type"` // "virtual" or "hardware"
	EnableDate   time.Time `json:"enableDate"`
	UserName     string    `json:"userName"`
	ARN          string    `json:"arn"`
}

// MFAValidationResult represents the result of MFA device validation
type MFAValidationResult struct {
	IsValid        bool       `json:"isValid"`
	Device         *MFADevice `json:"device,omitempty"`
	Error          string     `json:"error,omitempty"`
	ValidationTime time.Time  `json:"validationTime"`
}

// MFASessionCache represents cached MFA session information
type MFASessionCache struct {
	RoleArn     string       `json:"roleArn"`
	Credentials *Credentials `json:"credentials"`
	ExpiresAt   time.Time    `json:"expiresAt"`
	MFADevice   string       `json:"mfaDevice"`
	SessionName string       `json:"sessionName"`
}

// AssumeRoleWithMFAEnhanced assumes a role using MFA with enhanced functionality
func (p *AWSProvider) AssumeRoleWithMFAEnhanced(ctx context.Context, roleArn, mfaDeviceArn, mfaToken string, options *AssumeRoleOptions) (*Credentials, error) {
	// Validate MFA device first
	validation, err := p.ValidateMFADevice(ctx, mfaDeviceArn)
	if err != nil {
		return nil, fmt.Errorf("failed to validate MFA device: %w", err)
	}

	if !validation.IsValid {
		return nil, fmt.Errorf("MFA device validation failed: %s", validation.Error)
	}

	// Validate MFA token format
	if err := p.validateMFAToken(mfaToken); err != nil {
		return nil, fmt.Errorf("invalid MFA token: %w", err)
	}

	// Use existing AssumeRoleWithMFA method
	return p.AssumeRoleWithMFA(ctx, roleArn, mfaDeviceArn, mfaToken, options)
}

// ValidateMFADevice validates an MFA device
func (p *AWSProvider) ValidateMFADevice(ctx context.Context, mfaDeviceArn string) (*MFAValidationResult, error) {
	result := &MFAValidationResult{
		ValidationTime: time.Now(),
	}

	// Extract user name from MFA device ARN
	// Format: arn:aws:iam::123456789012:mfa/username
	parts := strings.Split(mfaDeviceArn, "/")
	if len(parts) < 2 {
		result.Error = "invalid MFA device ARN format"
		return result, nil
	}
	userName := parts[len(parts)-1]

	// List MFA devices for the user
	cmd := exec.CommandContext(ctx, "aws", "iam", "list-mfa-devices", "--user-name", userName, "--output", "json")
	if p.config != nil && p.config.Region != "" {
		cmd.Env = append(cmd.Environ(), fmt.Sprintf("AWS_DEFAULT_REGION=%s", p.config.Region))
	}

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.Error = fmt.Sprintf("failed to list MFA devices: %s", string(exitErr.Stderr))
		} else {
			result.Error = fmt.Sprintf("failed to list MFA devices: %v", err)
		}
		return result, nil
	}

	var mfaListResponse struct {
		MFADevices []struct {
			SerialNumber string `json:"SerialNumber"`
			EnableDate   string `json:"EnableDate"`
			UserName     string `json:"UserName"`
		} `json:"MFADevices"`
	}

	if err := json.Unmarshal(output, &mfaListResponse); err != nil {
		result.Error = fmt.Sprintf("failed to parse MFA devices response: %v", err)
		return result, nil
	}

	// Find the specified MFA device
	for _, device := range mfaListResponse.MFADevices {
		if device.SerialNumber == mfaDeviceArn {
			enableDate, _ := time.Parse(time.RFC3339, device.EnableDate)
			result.IsValid = true
			result.Device = &MFADevice{
				SerialNumber: device.SerialNumber,
				Type:         p.determineMFADeviceType(device.SerialNumber),
				EnableDate:   enableDate,
				UserName:     device.UserName,
				ARN:          device.SerialNumber,
			}
			return result, nil
		}
	}

	result.Error = "MFA device not found or not enabled"
	return result, nil
}

// ListMFADevices lists all MFA devices for a user
func (p *AWSProvider) ListMFADevices(ctx context.Context, userName string) ([]*MFADevice, error) {
	cmd := exec.CommandContext(ctx, "aws", "iam", "list-mfa-devices", "--user-name", userName, "--output", "json")
	if p.config != nil && p.config.Region != "" {
		cmd.Env = append(cmd.Environ(), fmt.Sprintf("AWS_DEFAULT_REGION=%s", p.config.Region))
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list MFA devices: %w", err)
	}

	var mfaListResponse struct {
		MFADevices []struct {
			SerialNumber string `json:"SerialNumber"`
			EnableDate   string `json:"EnableDate"`
			UserName     string `json:"UserName"`
		} `json:"MFADevices"`
	}

	if err := json.Unmarshal(output, &mfaListResponse); err != nil {
		return nil, fmt.Errorf("failed to parse MFA devices response: %w", err)
	}

	devices := make([]*MFADevice, 0, len(mfaListResponse.MFADevices))
	for _, device := range mfaListResponse.MFADevices {
		enableDate, _ := time.Parse(time.RFC3339, device.EnableDate)
		devices = append(devices, &MFADevice{
			SerialNumber: device.SerialNumber,
			Type:         p.determineMFADeviceType(device.SerialNumber),
			EnableDate:   enableDate,
			UserName:     device.UserName,
			ARN:          device.SerialNumber,
		})
	}

	return devices, nil
}

// GetCurrentUserMFADevices gets MFA devices for the current user
func (p *AWSProvider) GetCurrentUserMFADevices(ctx context.Context) ([]*MFADevice, error) {
	// Get current user name
	cmd := exec.CommandContext(ctx, "aws", "sts", "get-caller-identity", "--output", "json")
	if p.config != nil && p.config.Region != "" {
		cmd.Env = append(cmd.Environ(), fmt.Sprintf("AWS_DEFAULT_REGION=%s", p.config.Region))
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get caller identity: %w", err)
	}

	var identity struct {
		Arn string `json:"Arn"`
	}

	if err := json.Unmarshal(output, &identity); err != nil {
		return nil, fmt.Errorf("failed to parse caller identity: %w", err)
	}

	// Extract username from ARN
	// Format: arn:aws:iam::123456789012:user/username
	parts := strings.Split(identity.Arn, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("unable to extract username from ARN: %s", identity.Arn)
	}
	userName := parts[len(parts)-1]

	return p.ListMFADevices(ctx, userName)
}

// determineMFADeviceType determines if an MFA device is virtual or hardware based on its ARN
func (p *AWSProvider) determineMFADeviceType(serialNumber string) string {
	if strings.Contains(serialNumber, ":mfa/") {
		return "virtual"
	}
	return "hardware"
}

// validateMFAToken validates the format of an MFA token
func (p *AWSProvider) validateMFAToken(token string) error {
	// MFA tokens should be 6 digits
	if len(token) != 6 {
		return fmt.Errorf("MFA token must be 6 digits")
	}

	// Check if all characters are digits
	for _, c := range token {
		if c < '0' || c > '9' {
			return fmt.Errorf("MFA token must contain only digits")
		}
	}

	return nil
}

// AssumeRoleWithMFARetry assumes a role with MFA, retrying on invalid token
func (p *AWSProvider) AssumeRoleWithMFARetry(ctx context.Context, roleArn, mfaDeviceArn, mfaToken string, options *AssumeRoleOptions, maxRetries int) (*Credentials, error) {
	var lastErr error

	for i := 0; i <= maxRetries; i++ {
		creds, err := p.AssumeRoleWithMFAEnhanced(ctx, roleArn, mfaDeviceArn, mfaToken, options)
		if err == nil {
			return creds, nil
		}

		lastErr = err

		// Check if the error is due to invalid MFA token
		if !strings.Contains(err.Error(), "MultiFactorAuthentication failed") &&
			!strings.Contains(err.Error(), "invalid MFA") {
			// This is not an MFA-related error, don't retry
			return nil, err
		}

		if i < maxRetries {
			// For retries, we would need to prompt for a new token
			// This is just a placeholder - in practice, this would need to be handled by the caller
			return nil, fmt.Errorf("MFA token invalid, please provide a new token: %w", lastErr)
		}
	}

	return nil, fmt.Errorf("failed to assume role after %d retries: %w", maxRetries, lastErr)
}

// ValidateMFARequirement checks if a role requires MFA
func (p *AWSProvider) ValidateMFARequirement(ctx context.Context, roleArn string) (bool, error) {
	// Get role information
	roleName := p.extractRoleNameFromArn(roleArn)
	accountID := p.extractAccountIDFromArn(roleArn)

	cmd := exec.CommandContext(ctx, "aws", "iam", "get-role", "--role-name", roleName, "--output", "json")

	// If cross-account, we might not have permissions to check directly
	if p.extractAccountIDFromArn(roleArn) != accountID {
		// For cross-account roles, we can't directly check the trust policy
		// Return false as we can't determine the requirement
		return false, nil
	}

	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to get role information: %w", err)
	}

	var roleResponse struct {
		Role struct {
			AssumeRolePolicyDocument string `json:"AssumeRolePolicyDocument"`
		} `json:"Role"`
	}

	if err := json.Unmarshal(output, &roleResponse); err != nil {
		return false, fmt.Errorf("failed to parse role response: %w", err)
	}

	// Parse trust policy to check for MFA requirement
	var trustPolicy struct {
		Statement []struct {
			Effect    string                 `json:"Effect"`
			Principal map[string]interface{} `json:"Principal"`
			Action    interface{}            `json:"Action"`
			Condition map[string]interface{} `json:"Condition,omitempty"`
		} `json:"Statement"`
	}

	if err := json.Unmarshal([]byte(roleResponse.Role.AssumeRolePolicyDocument), &trustPolicy); err != nil {
		return false, fmt.Errorf("failed to parse trust policy: %w", err)
	}

	// Check for MFA condition
	for _, statement := range trustPolicy.Statement {
		if statement.Effect == "Allow" && statement.Condition != nil {
			if boolConditions, ok := statement.Condition["Bool"].(map[string]interface{}); ok {
				if mfaValue, exists := boolConditions["aws:MultiFactorAuthPresent"]; exists {
					if mfaStr, ok := mfaValue.(string); ok && mfaStr == "true" {
						return true, nil
					}
				}
			}
		}
	}

	return false, nil
}

// extractRoleNameFromArn extracts the role name from an ARN
func (p *AWSProvider) extractRoleNameFromArn(arn string) string {
	// Format: arn:aws:iam::123456789012:role/role-name
	parts := strings.Split(arn, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// extractAccountIDFromArn extracts the account ID from an ARN
func (p *AWSProvider) extractAccountIDFromArn(arn string) string {
	// Format: arn:aws:iam::123456789012:role/role-name
	parts := strings.Split(arn, ":")
	if len(parts) >= 5 {
		return parts[4]
	}
	return ""
}

// MFASessionManager manages MFA sessions with caching
type MFASessionManager struct {
	provider *AWSProvider
	cache    map[string]*MFASessionCache
}

// NewMFASessionManager creates a new MFA session manager
func (p *AWSProvider) NewMFASessionManager() *MFASessionManager {
	return &MFASessionManager{
		provider: p,
		cache:    make(map[string]*MFASessionCache),
	}
}

// GetOrCreateSession gets an existing session or creates a new one
func (m *MFASessionManager) GetOrCreateSession(ctx context.Context, roleArn, mfaDeviceArn, mfaToken string, options *AssumeRoleOptions) (*Credentials, error) {
	cacheKey := fmt.Sprintf("%s:%s", roleArn, mfaDeviceArn)

	// Check cache
	if cached, exists := m.cache[cacheKey]; exists {
		if time.Now().Before(cached.ExpiresAt.Add(-5 * time.Minute)) {
			// Still valid with 5-minute buffer
			return cached.Credentials, nil
		}
	}

	// Create new session
	creds, err := m.provider.AssumeRoleWithMFAEnhanced(ctx, roleArn, mfaDeviceArn, mfaToken, options)
	if err != nil {
		return nil, err
	}

	// Cache the session
	m.cache[cacheKey] = &MFASessionCache{
		RoleArn:     roleArn,
		Credentials: creds,
		ExpiresAt:   creds.Expiration,
		MFADevice:   mfaDeviceArn,
		SessionName: options.SessionName,
	}

	return creds, nil
}

// ClearExpiredSessions removes expired sessions from cache
func (m *MFASessionManager) ClearExpiredSessions() {
	now := time.Now()
	for key, session := range m.cache {
		if now.After(session.ExpiresAt) {
			delete(m.cache, key)
		}
	}
}
