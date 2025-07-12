package main

import (
	"testing"
	"time"

	"github.com/chaksack/apm/pkg/cloud"
)

func TestAssumeRoleOptions(t *testing.T) {
	options := cloud.DefaultAssumeRoleOptions()

	if options.SessionName == "" {
		t.Error("Default session name should not be empty")
	}

	if options.DurationSeconds != 3600 {
		t.Errorf("Expected duration 3600, got %d", options.DurationSeconds)
	}

	if !options.EnableCredentialCache {
		t.Error("Expected credential cache to be enabled by default")
	}

	if !options.EnableAutoRefresh {
		t.Error("Expected auto refresh to be enabled by default")
	}
}

func TestCrossAccountSession(t *testing.T) {
	credentials := &cloud.Credentials{
		Provider:  cloud.ProviderAWS,
		AccessKey: "test-access-key",
		SecretKey: "test-secret-key",
		Token:     "test-token",
		Expiry:    timePtr(time.Now().Add(1 * time.Hour)),
	}

	session := &cloud.CrossAccountSession{
		Credentials:      credentials,
		RoleArn:          "arn:aws:iam::123456789012:role/TestRole",
		SessionName:      "test-session",
		CreatedAt:        time.Now(),
		ExpiresAt:        *credentials.Expiry,
		RefreshThreshold: 5 * time.Minute,
	}

	// Test session is not expired
	if session.IsExpired() {
		t.Error("Session should not be expired")
	}

	// Test time until expiry
	timeUntilExpiry := session.TimeUntilExpiry()
	if timeUntilExpiry <= 0 {
		t.Error("Time until expiry should be positive")
	}

	// Test time until refresh
	timeUntilRefresh := session.TimeUntilRefresh()
	if timeUntilRefresh <= 0 {
		t.Error("Time until refresh should be positive")
	}

	// Test update credentials
	newCredentials := &cloud.Credentials{
		Provider:  cloud.ProviderAWS,
		AccessKey: "new-access-key",
		SecretKey: "new-secret-key",
		Token:     "new-token",
		Expiry:    timePtr(time.Now().Add(2 * time.Hour)),
	}

	session.UpdateCredentials(newCredentials)

	if session.Credentials.AccessKey != "new-access-key" {
		t.Error("Session credentials were not updated")
	}
}

func TestAccountConfig(t *testing.T) {
	account := &cloud.AccountConfig{
		AccountID:       "123456789012",
		AccountName:     "Test Account",
		Environment:     "test",
		DefaultRegion:   "us-east-1",
		SessionDuration: 3600,
		Roles: []*cloud.RoleConfig{
			{
				RoleName:        "TestRole",
				Description:     "Test role description",
				SessionDuration: 1800,
			},
		},
	}

	if account.AccountID == "" {
		t.Error("Account ID should not be empty")
	}

	if len(account.Roles) != 1 {
		t.Errorf("Expected 1 role, got %d", len(account.Roles))
	}

	if account.Roles[0].RoleName != "TestRole" {
		t.Errorf("Expected role name 'TestRole', got '%s'", account.Roles[0].RoleName)
	}
}

func TestMultiAccountConfig(t *testing.T) {
	config := &cloud.MultiAccountConfig{
		Organization:  "Test Organization",
		MasterAccount: "123456789012",
		Accounts:      make([]*cloud.AccountConfig, 0),
		DefaultRegion: "us-east-1",
		GlobalTags:    map[string]string{"Environment": "test"},
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if config.Organization == "" {
		t.Error("Organization should not be empty")
	}

	if config.DefaultRegion != "us-east-1" {
		t.Errorf("Expected default region 'us-east-1', got '%s'", config.DefaultRegion)
	}

	if len(config.GlobalTags) != 1 {
		t.Errorf("Expected 1 global tag, got %d", len(config.GlobalTags))
	}
}

func TestAccountManager(t *testing.T) {
	// Create a mock AWS provider
	providerConfig := &cloud.ProviderConfig{
		Provider:      cloud.ProviderAWS,
		DefaultRegion: "us-east-1",
		EnableCache:   true,
		CacheDuration: 30 * time.Minute,
	}

	provider, err := cloud.NewAWSProvider(providerConfig)
	if err != nil {
		t.Fatalf("Failed to create AWS provider: %v", err)
	}

	accountManager := provider.GetAccountManager()

	// Test adding an account
	account := &cloud.AccountConfig{
		AccountID:       "123456789012",
		AccountName:     "Test Account",
		Environment:     "test",
		DefaultRegion:   "us-east-1",
		SessionDuration: 3600,
	}

	err = accountManager.AddAccount(account)
	if err != nil {
		t.Errorf("Failed to add account: %v", err)
	}

	// Test getting the account
	retrievedAccount, err := accountManager.GetAccount("123456789012")
	if err != nil {
		t.Errorf("Failed to get account: %v", err)
	}

	if retrievedAccount.AccountName != "Test Account" {
		t.Errorf("Expected account name 'Test Account', got '%s'", retrievedAccount.AccountName)
	}

	// Test listing accounts
	accounts := accountManager.ListAccounts()
	if len(accounts) != 1 {
		t.Errorf("Expected 1 account, got %d", len(accounts))
	}

	// Test adding a duplicate account (should fail)
	err = accountManager.AddAccount(account)
	if err == nil {
		t.Error("Adding duplicate account should fail")
	}

	// Test updating account
	updatedAccount := &cloud.AccountConfig{
		AccountID:       "123456789012",
		AccountName:     "Updated Test Account",
		Environment:     "test",
		DefaultRegion:   "us-west-2",
		SessionDuration: 1800,
	}

	err = accountManager.UpdateAccount("123456789012", updatedAccount)
	if err != nil {
		t.Errorf("Failed to update account: %v", err)
	}

	// Verify update
	retrievedAccount, err = accountManager.GetAccount("123456789012")
	if err != nil {
		t.Errorf("Failed to get updated account: %v", err)
	}

	if retrievedAccount.AccountName != "Updated Test Account" {
		t.Errorf("Expected updated account name 'Updated Test Account', got '%s'", retrievedAccount.AccountName)
	}

	// Test removing account
	err = accountManager.RemoveAccount("123456789012")
	if err != nil {
		t.Errorf("Failed to remove account: %v", err)
	}

	// Verify removal
	_, err = accountManager.GetAccount("123456789012")
	if err == nil {
		t.Error("Getting removed account should fail")
	}
}

func TestRoleConfigValidation(t *testing.T) {
	providerConfig := &cloud.ProviderConfig{
		Provider:      cloud.ProviderAWS,
		DefaultRegion: "us-east-1",
		EnableCache:   true,
		CacheDuration: 30 * time.Minute,
	}

	provider, err := cloud.NewAWSProvider(providerConfig)
	if err != nil {
		t.Fatalf("Failed to create AWS provider: %v", err)
	}

	accountManager := provider.GetAccountManager()

	// Add account first
	account := &cloud.AccountConfig{
		AccountID:       "123456789012",
		AccountName:     "Test Account",
		Environment:     "test",
		DefaultRegion:   "us-east-1",
		SessionDuration: 3600,
	}

	err = accountManager.AddAccount(account)
	if err != nil {
		t.Fatalf("Failed to add account: %v", err)
	}

	// Test adding role to account
	role := &cloud.RoleConfig{
		RoleName:        "TestRole",
		Description:     "Test role for validation",
		SessionDuration: 1800,
	}

	err = accountManager.AddRoleToAccount("123456789012", role)
	if err != nil {
		t.Errorf("Failed to add role to account: %v", err)
	}

	// Test getting role from account
	retrievedRole, err := accountManager.GetRoleFromAccount("123456789012", "TestRole")
	if err != nil {
		t.Errorf("Failed to get role from account: %v", err)
	}

	if retrievedRole.RoleName != "TestRole" {
		t.Errorf("Expected role name 'TestRole', got '%s'", retrievedRole.RoleName)
	}

	// Verify role ARN was generated
	expectedArn := "arn:aws:iam::123456789012:role/TestRole"
	if retrievedRole.RoleArn != expectedArn {
		t.Errorf("Expected role ARN '%s', got '%s'", expectedArn, retrievedRole.RoleArn)
	}
}

func TestRoleValidation(t *testing.T) {
	validation := &cloud.RoleValidation{
		RoleArn:            "arn:aws:iam::123456789012:role/TestRole",
		CanAssume:          true,
		TrustPolicyValid:   true,
		ExternalIDRequired: false,
		MFARequired:        false,
		MaxSessionDuration: 3600,
		ValidatedAt:        time.Now(),
	}

	if !validation.CanAssume {
		t.Error("Expected role to be assumable")
	}

	if !validation.TrustPolicyValid {
		t.Error("Expected trust policy to be valid")
	}

	if validation.MaxSessionDuration != 3600 {
		t.Errorf("Expected max session duration 3600, got %d", validation.MaxSessionDuration)
	}
}

func TestConfigValidationResult(t *testing.T) {
	result := &cloud.ConfigValidationResult{
		IsValid:          true,
		Accounts:         make([]*cloud.AccountValidation, 0),
		ValidationErrors: make([]string, 0),
		ValidatedAt:      time.Now(),
	}

	accountValidation := &cloud.AccountValidation{
		AccountID:        "123456789012",
		IsValid:          true,
		Roles:            make([]*cloud.RoleValidation, 0),
		ValidationErrors: make([]string, 0),
	}

	result.Accounts = append(result.Accounts, accountValidation)

	if !result.IsValid {
		t.Error("Expected validation result to be valid")
	}

	if len(result.Accounts) != 1 {
		t.Errorf("Expected 1 account validation, got %d", len(result.Accounts))
	}

	if result.Accounts[0].AccountID != "123456789012" {
		t.Errorf("Expected account ID '123456789012', got '%s'", result.Accounts[0].AccountID)
	}
}

func TestGetAccountsByEnvironment(t *testing.T) {
	providerConfig := &cloud.ProviderConfig{
		Provider:      cloud.ProviderAWS,
		DefaultRegion: "us-east-1",
		EnableCache:   true,
		CacheDuration: 30 * time.Minute,
	}

	provider, err := cloud.NewAWSProvider(providerConfig)
	if err != nil {
		t.Fatalf("Failed to create AWS provider: %v", err)
	}

	accountManager := provider.GetAccountManager()

	// Add development account
	devAccount := &cloud.AccountConfig{
		AccountID:   "111111111111",
		AccountName: "Dev Account",
		Environment: "dev",
	}

	// Add production account
	prodAccount := &cloud.AccountConfig{
		AccountID:   "222222222222",
		AccountName: "Prod Account",
		Environment: "prod",
	}

	// Add another dev account
	devAccount2 := &cloud.AccountConfig{
		AccountID:   "333333333333",
		AccountName: "Dev Account 2",
		Environment: "dev",
	}

	err = accountManager.AddAccount(devAccount)
	if err != nil {
		t.Fatalf("Failed to add dev account: %v", err)
	}

	err = accountManager.AddAccount(prodAccount)
	if err != nil {
		t.Fatalf("Failed to add prod account: %v", err)
	}

	err = accountManager.AddAccount(devAccount2)
	if err != nil {
		t.Fatalf("Failed to add dev account 2: %v", err)
	}

	// Test filtering by environment
	devAccounts := accountManager.GetAccountsByEnvironment("dev")
	prodAccounts := accountManager.GetAccountsByEnvironment("prod")

	if len(devAccounts) != 2 {
		t.Errorf("Expected 2 dev accounts, got %d", len(devAccounts))
	}

	if len(prodAccounts) != 1 {
		t.Errorf("Expected 1 prod account, got %d", len(prodAccounts))
	}

	// Test non-existent environment
	stagingAccounts := accountManager.GetAccountsByEnvironment("staging")
	if len(stagingAccounts) != 0 {
		t.Errorf("Expected 0 staging accounts, got %d", len(stagingAccounts))
	}
}

// Helper function to create time pointer
func timePtr(t time.Time) *time.Time {
	return &t
}

// Benchmark tests
func BenchmarkAssumeRoleOptions(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = cloud.DefaultAssumeRoleOptions()
	}
}

func BenchmarkAccountManagerAddAccount(b *testing.B) {
	providerConfig := &cloud.ProviderConfig{
		Provider:      cloud.ProviderAWS,
		DefaultRegion: "us-east-1",
		EnableCache:   true,
		CacheDuration: 30 * time.Minute,
	}

	provider, err := cloud.NewAWSProvider(providerConfig)
	if err != nil {
		b.Fatalf("Failed to create AWS provider: %v", err)
	}

	accountManager := provider.GetAccountManager()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		account := &cloud.AccountConfig{
			AccountID:   fmt.Sprintf("account-%d", i),
			AccountName: fmt.Sprintf("Account %d", i),
			Environment: "test",
		}

		err := accountManager.AddAccount(account)
		if err != nil {
			b.Errorf("Failed to add account: %v", err)
		}
	}
}

func BenchmarkSessionIsExpired(b *testing.B) {
	session := &cloud.CrossAccountSession{
		ExpiresAt:        time.Now().Add(1 * time.Hour),
		RefreshThreshold: 5 * time.Minute,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = session.IsExpired()
	}
}
