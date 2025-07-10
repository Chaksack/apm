package cloud

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// DefaultAuthManager implements the AuthManager interface
type DefaultAuthManager struct {
	credentialMgr CredentialManager
	sessions      map[string]*AuthSession
	tokens        map[Provider]string
	tokenExpiry   map[Provider]time.Time
	mu            sync.RWMutex
	sessionTTL    time.Duration
	tokenTTL      time.Duration
}

// NewDefaultAuthManager creates a new default auth manager
func NewDefaultAuthManager(credentialMgr CredentialManager) *DefaultAuthManager {
	return &DefaultAuthManager{
		credentialMgr: credentialMgr,
		sessions:      make(map[string]*AuthSession),
		tokens:        make(map[Provider]string),
		tokenExpiry:   make(map[Provider]time.Time),
		sessionTTL:    2 * time.Hour,
		tokenTTL:      1 * time.Hour,
	}
}

// IsAuthenticated checks if a provider is authenticated
func (dam *DefaultAuthManager) IsAuthenticated(ctx context.Context, provider Provider) (bool, error) {
	dam.mu.RLock()
	defer dam.mu.RUnlock()

	// Check if we have a valid token
	if expiry, exists := dam.tokenExpiry[provider]; exists {
		if time.Now().Before(expiry) {
			return true, nil
		}
		// Token expired, remove it
		delete(dam.tokens, provider)
		delete(dam.tokenExpiry, provider)
	}

	// Check if we have valid credentials stored
	if dam.credentialMgr != nil {
		creds, err := dam.credentialMgr.Retrieve(provider, "")
		if err == nil && creds != nil {
			// Check if credentials haven't expired
			if creds.Expiry == nil || time.Now().Before(*creds.Expiry) {
				return true, nil
			}
		}
	}

	return false, nil
}

// Authenticate authenticates with a provider using the given options
func (dam *DefaultAuthManager) Authenticate(ctx context.Context, provider Provider, options AuthOptions) error {
	errorBuilder := NewErrorBuilder(provider, "authenticate")

	// Validate options
	if err := dam.validateAuthOptions(options); err != nil {
		return errorBuilder.BuildWithCause(ErrCodeInvalidInput, "Invalid authentication options", err)
	}

	// Create credentials from options
	creds, err := dam.createCredentialsFromOptions(provider, options)
	if err != nil {
		return errorBuilder.BuildWithCause(ErrCodeAuthenticationFailed, "Failed to create credentials", err)
	}

	// Validate credentials by testing authentication
	if err := dam.validateCredentials(ctx, provider, creds); err != nil {
		return errorBuilder.BuildWithCause(ErrCodeAuthenticationFailed, "Credential validation failed", err)
	}

	// Store credentials if validation successful
	if dam.credentialMgr != nil {
		if err := dam.credentialMgr.Store(creds); err != nil {
			return errorBuilder.BuildWithCause(ErrCodeInternalError, "Failed to store credentials", err)
		}
	}

	// Generate and cache token
	token, err := dam.generateToken(provider, creds)
	if err != nil {
		return errorBuilder.BuildWithCause(ErrCodeInternalError, "Failed to generate token", err)
	}

	dam.mu.Lock()
	dam.tokens[provider] = token
	dam.tokenExpiry[provider] = time.Now().Add(dam.tokenTTL)
	dam.mu.Unlock()

	return nil
}

// RefreshCredentials refreshes credentials for a provider
func (dam *DefaultAuthManager) RefreshCredentials(ctx context.Context, provider Provider) error {
	errorBuilder := NewErrorBuilder(provider, "refresh_credentials")

	dam.mu.Lock()
	defer dam.mu.Unlock()

	// Get existing credentials
	if dam.credentialMgr == nil {
		return errorBuilder.Build(ErrCodeMissingConfiguration, "No credential manager configured")
	}

	creds, err := dam.credentialMgr.Retrieve(provider, "")
	if err != nil {
		return errorBuilder.BuildWithCause(ErrCodeAuthenticationFailed, "Failed to retrieve credentials", err)
	}

	// Refresh based on provider and auth method
	newCreds, err := dam.refreshCredentialsByMethod(ctx, provider, creds)
	if err != nil {
		return errorBuilder.BuildWithCause(ErrCodeAuthenticationFailed, "Failed to refresh credentials", err)
	}

	// Store updated credentials
	if err := dam.credentialMgr.Store(newCreds); err != nil {
		return errorBuilder.BuildWithCause(ErrCodeInternalError, "Failed to store refreshed credentials", err)
	}

	// Update token
	token, err := dam.generateToken(provider, newCreds)
	if err != nil {
		return errorBuilder.BuildWithCause(ErrCodeInternalError, "Failed to generate token after refresh", err)
	}

	dam.tokens[provider] = token
	dam.tokenExpiry[provider] = time.Now().Add(dam.tokenTTL)

	return nil
}

// GetValidToken gets a valid token for a provider
func (dam *DefaultAuthManager) GetValidToken(ctx context.Context, provider Provider) (string, error) {
	errorBuilder := NewErrorBuilder(provider, "get_valid_token")

	dam.mu.RLock()
	token, hasToken := dam.tokens[provider]
	expiry, hasExpiry := dam.tokenExpiry[provider]
	dam.mu.RUnlock()

	// Check if we have a valid cached token
	if hasToken && hasExpiry && time.Now().Before(expiry) {
		return token, nil
	}

	// Try to refresh credentials and get new token
	if err := dam.RefreshCredentials(ctx, provider); err != nil {
		return "", errorBuilder.BuildWithCause(ErrCodeTokenExpired, "Failed to refresh token", err)
	}

	dam.mu.RLock()
	token = dam.tokens[provider]
	dam.mu.RUnlock()

	return token, nil
}

// CacheToken caches a token for a provider
func (dam *DefaultAuthManager) CacheToken(provider Provider, token string, expiry time.Time) error {
	dam.mu.Lock()
	defer dam.mu.Unlock()

	dam.tokens[provider] = token
	dam.tokenExpiry[provider] = expiry

	return nil
}

// ClearCache clears the token cache for a provider
func (dam *DefaultAuthManager) ClearCache(provider Provider) error {
	dam.mu.Lock()
	defer dam.mu.Unlock()

	delete(dam.tokens, provider)
	delete(dam.tokenExpiry, provider)

	return nil
}

// CreateSession creates a new authentication session
func (dam *DefaultAuthManager) CreateSession(ctx context.Context, provider Provider) (*AuthSession, error) {
	errorBuilder := NewErrorBuilder(provider, "create_session")

	// Check if provider is authenticated
	authenticated, err := dam.IsAuthenticated(ctx, provider)
	if err != nil {
		return nil, errorBuilder.BuildWithCause(ErrCodeAuthenticationFailed, "Failed to check authentication", err)
	}

	if !authenticated {
		return nil, errorBuilder.Build(ErrCodeNotAuthenticated, "Provider is not authenticated")
	}

	// Get valid token
	token, err := dam.GetValidToken(ctx, provider)
	if err != nil {
		return nil, errorBuilder.BuildWithCause(ErrCodeTokenExpired, "Failed to get valid token", err)
	}

	// Generate session ID
	sessionID, err := dam.generateSessionID()
	if err != nil {
		return nil, errorBuilder.BuildWithCause(ErrCodeInternalError, "Failed to generate session ID", err)
	}

	// Create session
	session := &AuthSession{
		Provider:  provider,
		Method:    AuthMethodCLI, // Default method, could be determined from stored credentials
		Token:     token,
		Expiry:    time.Now().Add(dam.sessionTTL),
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
		Properties: map[string]string{
			"session_id": sessionID,
		},
	}

	// Store session
	dam.mu.Lock()
	dam.sessions[sessionID] = session
	dam.mu.Unlock()

	return session, nil
}

// ValidateSession validates an authentication session
func (dam *DefaultAuthManager) ValidateSession(ctx context.Context, session *AuthSession) error {
	if session == nil {
		return NewErrorBuilder(ProviderAWS, "validate_session").Build(ErrCodeInvalidInput, "Session is nil")
	}

	errorBuilder := NewErrorBuilder(session.Provider, "validate_session")

	// Check if session exists
	sessionID := session.Properties["session_id"]
	if sessionID == "" {
		return errorBuilder.Build(ErrCodeInvalidInput, "Session ID is missing")
	}

	dam.mu.RLock()
	storedSession, exists := dam.sessions[sessionID]
	dam.mu.RUnlock()

	if !exists {
		return errorBuilder.Build(ErrCodeSessionExpired, "Session not found")
	}

	// Check if session is expired
	if time.Now().After(storedSession.Expiry) {
		dam.mu.Lock()
		delete(dam.sessions, sessionID)
		dam.mu.Unlock()
		return errorBuilder.Build(ErrCodeSessionExpired, "Session has expired")
	}

	// Update last used time
	dam.mu.Lock()
	storedSession.LastUsed = time.Now()
	dam.mu.Unlock()

	return nil
}

// RevokeSession revokes an authentication session
func (dam *DefaultAuthManager) RevokeSession(ctx context.Context, session *AuthSession) error {
	if session == nil {
		return NewErrorBuilder(ProviderAWS, "revoke_session").Build(ErrCodeInvalidInput, "Session is nil")
	}

	sessionID := session.Properties["session_id"]
	if sessionID == "" {
		return NewErrorBuilder(session.Provider, "revoke_session").Build(ErrCodeInvalidInput, "Session ID is missing")
	}

	dam.mu.Lock()
	delete(dam.sessions, sessionID)
	dam.mu.Unlock()

	return nil
}

// validateAuthOptions validates authentication options
func (dam *DefaultAuthManager) validateAuthOptions(options AuthOptions) error {
	if options.Method == "" {
		return fmt.Errorf("authentication method is required")
	}

	switch options.Method {
	case AuthMethodCLI:
		// CLI method doesn't require additional parameters
		return nil
	case AuthMethodAccessKey:
		if options.AccessKey == "" || options.SecretKey == "" {
			return fmt.Errorf("access key and secret key are required for access key authentication")
		}
		return nil
	case AuthMethodServiceKey:
		if options.KeyFile == "" {
			return fmt.Errorf("key file is required for service key authentication")
		}
		return nil
	case AuthMethodIAMRole:
		// IAM role method may not require additional parameters
		return nil
	default:
		return fmt.Errorf("unsupported authentication method: %s", options.Method)
	}
}

// createCredentialsFromOptions creates credentials from auth options
func (dam *DefaultAuthManager) createCredentialsFromOptions(provider Provider, options AuthOptions) (*Credentials, error) {
	creds := &Credentials{
		Provider:   provider,
		AuthMethod: options.Method,
		Profile:    options.Profile,
		Region:     options.Region,
		Properties: make(map[string]string),
	}

	// Copy properties
	for k, v := range options.Properties {
		creds.Properties[k] = v
	}

	// Set method-specific fields
	switch options.Method {
	case AuthMethodAccessKey:
		creds.AccessKey = options.AccessKey
		creds.SecretKey = options.SecretKey
		creds.Token = options.Token
	case AuthMethodServiceKey:
		creds.Properties["key_file"] = options.KeyFile
		creds.Properties["client_id"] = options.ClientID
		creds.Properties["client_secret"] = options.ClientSecret
		creds.Properties["tenant_id"] = options.TenantID
		creds.Properties["project_id"] = options.ProjectID
	case AuthMethodCLI:
		// CLI credentials don't need specific fields
	case AuthMethodIAMRole:
		// IAM role may have specific properties
		if options.Token != "" {
			creds.Token = options.Token
		}
	}

	return creds, nil
}

// validateCredentials validates credentials by testing authentication
func (dam *DefaultAuthManager) validateCredentials(ctx context.Context, provider Provider, creds *Credentials) error {
	// This is a simplified validation - in a real implementation,
	// you would make actual API calls to validate credentials

	switch provider {
	case ProviderAWS:
		return dam.validateAWSCredentials(ctx, creds)
	case ProviderAzure:
		return dam.validateAzureCredentials(ctx, creds)
	case ProviderGCP:
		return dam.validateGCPCredentials(ctx, creds)
	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}
}

// validateAWSCredentials validates AWS credentials
func (dam *DefaultAuthManager) validateAWSCredentials(ctx context.Context, creds *Credentials) error {
	// In a real implementation, you would use AWS SDK to call sts:GetCallerIdentity
	// For now, just validate that required fields are present

	switch creds.AuthMethod {
	case AuthMethodAccessKey:
		if creds.AccessKey == "" || creds.SecretKey == "" {
			return fmt.Errorf("AWS access key and secret key are required")
		}
	case AuthMethodCLI:
		// CLI credentials are validated by the CLI itself
	case AuthMethodIAMRole:
		// IAM role validation would require checking role ARN
	default:
		return fmt.Errorf("unsupported AWS auth method: %s", creds.AuthMethod)
	}

	return nil
}

// validateAzureCredentials validates Azure credentials
func (dam *DefaultAuthManager) validateAzureCredentials(ctx context.Context, creds *Credentials) error {
	// In a real implementation, you would use Azure SDK to validate credentials

	switch creds.AuthMethod {
	case AuthMethodServiceKey:
		if creds.Properties["client_id"] == "" || creds.Properties["client_secret"] == "" || creds.Properties["tenant_id"] == "" {
			return fmt.Errorf("Azure client ID, client secret, and tenant ID are required")
		}
	case AuthMethodCLI:
		// CLI credentials are validated by the CLI itself
	default:
		return fmt.Errorf("unsupported Azure auth method: %s", creds.AuthMethod)
	}

	return nil
}

// validateGCPCredentials validates GCP credentials
func (dam *DefaultAuthManager) validateGCPCredentials(ctx context.Context, creds *Credentials) error {
	// In a real implementation, you would use GCP SDK to validate credentials

	switch creds.AuthMethod {
	case AuthMethodServiceKey:
		if creds.Properties["key_file"] == "" {
			return fmt.Errorf("GCP service account key file is required")
		}
	case AuthMethodCLI:
		// CLI credentials are validated by the CLI itself
	default:
		return fmt.Errorf("unsupported GCP auth method: %s", creds.AuthMethod)
	}

	return nil
}

// refreshCredentialsByMethod refreshes credentials based on the authentication method
func (dam *DefaultAuthManager) refreshCredentialsByMethod(ctx context.Context, provider Provider, creds *Credentials) (*Credentials, error) {
	switch creds.AuthMethod {
	case AuthMethodAccessKey:
		// Access key credentials typically don't need refreshing unless using STS
		return creds, nil
	case AuthMethodCLI:
		// CLI credentials are refreshed by the CLI itself
		return creds, nil
	case AuthMethodIAMRole:
		// IAM role credentials may need token refresh
		return dam.refreshIAMRoleCredentials(ctx, provider, creds)
	case AuthMethodServiceKey:
		// Service key credentials may need token refresh
		return dam.refreshServiceKeyCredentials(ctx, provider, creds)
	default:
		return nil, fmt.Errorf("unsupported auth method for refresh: %s", creds.AuthMethod)
	}
}

// refreshIAMRoleCredentials refreshes IAM role credentials
func (dam *DefaultAuthManager) refreshIAMRoleCredentials(ctx context.Context, provider Provider, creds *Credentials) (*Credentials, error) {
	// In a real implementation, you would use the appropriate SDK to refresh the role credentials
	newCreds := *creds
	newCreds.Expiry = &time.Time{}
	*newCreds.Expiry = time.Now().Add(1 * time.Hour)
	return &newCreds, nil
}

// refreshServiceKeyCredentials refreshes service key credentials
func (dam *DefaultAuthManager) refreshServiceKeyCredentials(ctx context.Context, provider Provider, creds *Credentials) (*Credentials, error) {
	// In a real implementation, you would use the appropriate SDK to refresh the service key credentials
	newCreds := *creds
	newCreds.Expiry = &time.Time{}
	*newCreds.Expiry = time.Now().Add(1 * time.Hour)
	return &newCreds, nil
}

// generateToken generates a token for a provider and credentials
func (dam *DefaultAuthManager) generateToken(provider Provider, creds *Credentials) (string, error) {
	// In a real implementation, this would generate or retrieve an actual token
	// For now, generate a simple token based on provider and timestamp

	tokenData := fmt.Sprintf("%s:%s:%d", provider, creds.AuthMethod, time.Now().Unix())
	return fmt.Sprintf("apm_token_%x", []byte(tokenData)), nil
}

// generateSessionID generates a unique session ID
func (dam *DefaultAuthManager) generateSessionID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CleanupExpiredSessions removes expired sessions
func (dam *DefaultAuthManager) CleanupExpiredSessions() {
	dam.mu.Lock()
	defer dam.mu.Unlock()

	now := time.Now()
	for sessionID, session := range dam.sessions {
		if now.After(session.Expiry) {
			delete(dam.sessions, sessionID)
		}
	}
}

// GetSessionStats returns session statistics
func (dam *DefaultAuthManager) GetSessionStats() AuthSessionStats {
	dam.mu.RLock()
	defer dam.mu.RUnlock()

	stats := AuthSessionStats{
		TotalSessions:      len(dam.sessions),
		SessionsByProvider: make(map[Provider]int),
	}

	now := time.Now()
	for _, session := range dam.sessions {
		stats.SessionsByProvider[session.Provider]++
		if now.After(session.Expiry) {
			stats.ExpiredSessions++
		}
	}

	return stats
}

// SetSessionTTL sets the session time-to-live
func (dam *DefaultAuthManager) SetSessionTTL(ttl time.Duration) {
	dam.mu.Lock()
	defer dam.mu.Unlock()
	dam.sessionTTL = ttl
}

// SetTokenTTL sets the token time-to-live
func (dam *DefaultAuthManager) SetTokenTTL(ttl time.Duration) {
	dam.mu.Lock()
	defer dam.mu.Unlock()
	dam.tokenTTL = ttl
}

// AuthSessionStats represents authentication session statistics
type AuthSessionStats struct {
	TotalSessions      int              `json:"total_sessions"`
	ExpiredSessions    int              `json:"expired_sessions"`
	SessionsByProvider map[Provider]int `json:"sessions_by_provider"`
}

// AuthenticationHelper provides helper methods for authentication
type AuthenticationHelper struct {
	authMgr AuthManager
}

// NewAuthenticationHelper creates a new authentication helper
func NewAuthenticationHelper(authMgr AuthManager) *AuthenticationHelper {
	return &AuthenticationHelper{
		authMgr: authMgr,
	}
}

// AuthenticateWithCLI authenticates using CLI method
func (ah *AuthenticationHelper) AuthenticateWithCLI(ctx context.Context, provider Provider, profile string) error {
	options := AuthOptions{
		Method:  AuthMethodCLI,
		Profile: profile,
	}

	return ah.authMgr.Authenticate(ctx, provider, options)
}

// AuthenticateWithAccessKey authenticates using access key method
func (ah *AuthenticationHelper) AuthenticateWithAccessKey(ctx context.Context, provider Provider, accessKey, secretKey, region string) error {
	options := AuthOptions{
		Method:    AuthMethodAccessKey,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Region:    region,
	}

	return ah.authMgr.Authenticate(ctx, provider, options)
}

// AuthenticateWithServiceKey authenticates using service key method
func (ah *AuthenticationHelper) AuthenticateWithServiceKey(ctx context.Context, provider Provider, keyFile string, properties map[string]string) error {
	options := AuthOptions{
		Method:     AuthMethodServiceKey,
		KeyFile:    keyFile,
		Properties: properties,
	}

	return ah.authMgr.Authenticate(ctx, provider, options)
}

// EnsureAuthenticated ensures a provider is authenticated, attempting authentication if needed
func (ah *AuthenticationHelper) EnsureAuthenticated(ctx context.Context, provider Provider) error {
	authenticated, err := ah.authMgr.IsAuthenticated(ctx, provider)
	if err != nil {
		return err
	}

	if authenticated {
		return nil
	}

	// Try CLI authentication as default
	return ah.AuthenticateWithCLI(ctx, provider, "")
}

// GetAuthenticationStatus gets detailed authentication status for a provider
func (ah *AuthenticationHelper) GetAuthenticationStatus(ctx context.Context, provider Provider) (*AuthenticationStatus, error) {
	status := &AuthenticationStatus{
		Provider:      provider,
		Authenticated: false,
		LastChecked:   time.Now(),
	}

	authenticated, err := ah.authMgr.IsAuthenticated(ctx, provider)
	if err != nil {
		status.Error = err.Error()
		return status, nil
	}

	status.Authenticated = authenticated

	if authenticated {
		// Try to get token to check expiry
		token, err := ah.authMgr.GetValidToken(ctx, provider)
		if err == nil && token != "" {
			status.HasValidToken = true
		}
	}

	return status, nil
}

// AuthenticationStatus represents the authentication status of a provider
type AuthenticationStatus struct {
	Provider      Provider  `json:"provider"`
	Authenticated bool      `json:"authenticated"`
	HasValidToken bool      `json:"has_valid_token"`
	LastChecked   time.Time `json:"last_checked"`
	Error         string    `json:"error,omitempty"`
}

// MultiProviderAuthenticator handles authentication across multiple providers
type MultiProviderAuthenticator struct {
	authMgr AuthManager
	helper  *AuthenticationHelper
}

// NewMultiProviderAuthenticator creates a new multi-provider authenticator
func NewMultiProviderAuthenticator(authMgr AuthManager) *MultiProviderAuthenticator {
	return &MultiProviderAuthenticator{
		authMgr: authMgr,
		helper:  NewAuthenticationHelper(authMgr),
	}
}

// AuthenticateAll attempts to authenticate with all available providers
func (mpa *MultiProviderAuthenticator) AuthenticateAll(ctx context.Context) map[Provider]error {
	providers := []Provider{ProviderAWS, ProviderAzure, ProviderGCP}
	results := make(map[Provider]error)

	for _, provider := range providers {
		err := mpa.helper.EnsureAuthenticated(ctx, provider)
		if err != nil {
			results[provider] = err
		}
	}

	return results
}

// GetAllAuthenticationStatus gets authentication status for all providers
func (mpa *MultiProviderAuthenticator) GetAllAuthenticationStatus(ctx context.Context) map[Provider]*AuthenticationStatus {
	providers := []Provider{ProviderAWS, ProviderAzure, ProviderGCP}
	results := make(map[Provider]*AuthenticationStatus)

	for _, provider := range providers {
		status, err := mpa.helper.GetAuthenticationStatus(ctx, provider)
		if err != nil {
			status = &AuthenticationStatus{
				Provider:    provider,
				Error:       err.Error(),
				LastChecked: time.Now(),
			}
		}
		results[provider] = status
	}

	return results
}

// RefreshAll refreshes credentials for all authenticated providers
func (mpa *MultiProviderAuthenticator) RefreshAll(ctx context.Context) map[Provider]error {
	providers := []Provider{ProviderAWS, ProviderAzure, ProviderGCP}
	results := make(map[Provider]error)

	for _, provider := range providers {
		authenticated, err := mpa.authMgr.IsAuthenticated(ctx, provider)
		if err != nil {
			results[provider] = err
			continue
		}

		if authenticated {
			if err := mpa.authMgr.RefreshCredentials(ctx, provider); err != nil {
				results[provider] = err
			}
		}
	}

	return results
}
