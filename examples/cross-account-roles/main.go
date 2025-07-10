package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ybke/apm/pkg/cloud"
)

func main() {
	fmt.Println("Cross-Account Role Assumption Example for APM Tool")
	fmt.Println("================================================")

	// Create AWS provider
	config := &cloud.ProviderConfig{
		Provider:      cloud.ProviderAWS,
		DefaultRegion: "us-east-1",
		EnableCache:   true,
		CacheDuration: 30 * time.Minute,
	}

	provider, err := cloud.NewAWSProvider(config)
	if err != nil {
		log.Fatalf("Failed to create AWS provider: %v", err)
	}

	ctx := context.Background()

	// Example 1: Basic Cross-Account Role Assumption
	fmt.Println("\n1. Basic Cross-Account Role Assumption")
	fmt.Println("--------------------------------------")
	
	sourceAccount := "123456789012"
	targetAccount := "987654321098"
	roleName := "APMCrossAccountRole"
	
	options := &cloud.AssumeRoleOptions{
		SessionName:     "apm-cross-account-demo",
		DurationSeconds: 3600, // 1 hour
		Region:          "us-east-1",
	}
	
	credentials, err := provider.AssumeRoleAcrossAccount(ctx, sourceAccount, targetAccount, roleName, options)
	if err != nil {
		fmt.Printf("Error assuming role across accounts: %v\n", err)
	} else {
		fmt.Printf("Successfully assumed role: %s\n", credentials.Properties["assumed_role_arn"])
		fmt.Printf("Session expires at: %v\n", credentials.Expiry)
	}

	// Example 2: MFA-Based Role Assumption
	fmt.Println("\n2. MFA-Based Role Assumption")
	fmt.Println("-----------------------------")
	
	roleArn := "arn:aws:iam::987654321098:role/APMSecureRole"
	mfaDeviceArn := "arn:aws:iam::123456789012:mfa/user@example.com"
	mfaToken := "123456" // In practice, this would be from an MFA device
	
	mfaCredentials, err := provider.AssumeRoleWithMFA(ctx, roleArn, mfaDeviceArn, mfaToken, options)
	if err != nil {
		fmt.Printf("Error assuming role with MFA: %v\n", err)
	} else {
		fmt.Printf("Successfully assumed role with MFA: %s\n", mfaCredentials.Properties["assumed_role_arn"])
	}

	// Example 3: Role Chaining for Complex Multi-Account Scenarios
	fmt.Println("\n3. Role Chaining for Multi-Account Access")
	fmt.Println("----------------------------------------")
	
	roleChain := []*cloud.RoleChainStep{
		{
			RoleArn:     "arn:aws:iam::111111111111:role/OrganizationRole",
			SessionName: "apm-org-step",
		},
		{
			RoleArn:     "arn:aws:iam::222222222222:role/DevelopmentRole",
			SessionName: "apm-dev-step",
		},
		{
			RoleArn:     "arn:aws:iam::333333333333:role/ProductionRole",
			SessionName: "apm-prod-step",
			ExternalID:  "unique-external-id-123",
		},
	}
	
	chainCredentials, err := provider.AssumeRoleChain(ctx, roleChain)
	if err != nil {
		fmt.Printf("Error assuming role chain: %v\n", err)
	} else {
		fmt.Printf("Successfully completed role chain to: %s\n", chainCredentials.Properties["assumed_role_arn"])
	}

	// Example 4: External ID for Partner Access
	fmt.Println("\n4. External ID for Partner Access")
	fmt.Println("---------------------------------")
	
	partnerRoleArn := "arn:aws:iam::444444444444:role/PartnerAccessRole"
	externalID := "unique-partner-id-456"
	
	partnerCredentials, err := provider.AssumeRoleWithExternalID(ctx, partnerRoleArn, externalID, options)
	if err != nil {
		fmt.Printf("Error assuming partner role: %v\n", err)
	} else {
		fmt.Printf("Successfully assumed partner role: %s\n", partnerCredentials.Properties["assumed_role_arn"])
	}

	// Example 5: Cross-Account Role Manager with Automatic Refresh
	fmt.Println("\n5. Cross-Account Role Manager with Auto-Refresh")
	fmt.Println("-----------------------------------------------")
	
	roleManager := provider.GetCrossAccountRoleManager()
	defer roleManager.Close() // Important: Clean up background workers
	
	autoRefreshOptions := &cloud.AssumeRoleOptions{
		SessionName:          "apm-auto-refresh",
		DurationSeconds:      900, // 15 minutes for demo
		EnableAutoRefresh:    true,
		AutoRefreshThreshold: 2 * time.Minute, // Refresh when 2 minutes left
	}
	
	session, err := roleManager.GetSession(ctx, roleArn, autoRefreshOptions)
	if err != nil {
		fmt.Printf("Error creating managed session: %v\n", err)
	} else {
		fmt.Printf("Created managed session for: %s\n", session.RoleArn)
		fmt.Printf("Session expires at: %v\n", session.ExpiresAt)
		fmt.Printf("Will auto-refresh at: %v\n", session.ExpiresAt.Add(-session.RefreshThreshold))
		
		// List all active sessions
		sessions := roleManager.ListSessions()
		fmt.Printf("Active sessions: %d\n", len(sessions))
	}

	// Example 6: Role Validation
	fmt.Println("\n6. Role Validation")
	fmt.Println("------------------")
	
	validation, err := provider.ValidateRoleAssumption(ctx, roleArn, options)
	if err != nil {
		fmt.Printf("Error validating role: %v\n", err)
	} else {
		fmt.Printf("Role ARN: %s\n", validation.RoleArn)
		fmt.Printf("Can assume: %v\n", validation.CanAssume)
		fmt.Printf("Trust policy valid: %v\n", validation.TrustPolicyValid)
		fmt.Printf("External ID required: %v\n", validation.ExternalIDRequired)
		fmt.Printf("MFA required: %v\n", validation.MFARequired)
		fmt.Printf("Max session duration: %d seconds\n", validation.MaxSessionDuration)
		if validation.Error != "" {
			fmt.Printf("Validation error: %s\n", validation.Error)
		}
	}

	// Example 7: Multi-Account Configuration Management
	fmt.Println("\n7. Multi-Account Configuration Management")
	fmt.Println("----------------------------------------")
	
	accountManager := provider.GetAccountManager()
	
	// Add development account
	devAccount := &cloud.AccountConfig{
		AccountID:       "111111111111",
		AccountName:     "APM Development",
		Environment:     "dev",
		DefaultRegion:   "us-west-2",
		SessionDuration: 3600,
		Roles: []*cloud.RoleConfig{
			{
				RoleName:        "APMDeveloperRole",
				Description:     "Role for APM development access",
				SessionDuration: 3600,
			},
			{
				RoleName:        "APMReadOnlyRole",
				Description:     "Read-only role for APM monitoring",
				SessionDuration: 1800,
			},
		},
	}
	
	err = accountManager.AddAccount(devAccount)
	if err != nil {
		fmt.Printf("Error adding dev account: %v\n", err)
	} else {
		fmt.Printf("Added development account: %s\n", devAccount.AccountName)
	}
	
	// Add production account
	prodAccount := &cloud.AccountConfig{
		AccountID:       "333333333333",
		AccountName:     "APM Production",
		Environment:     "prod",
		DefaultRegion:   "us-east-1",
		MFARequired:     true,
		SessionDuration: 1800, // 30 minutes for production
		ExternalID:      "prod-external-id-789",
		Roles: []*cloud.RoleConfig{
			{
				RoleName:        "APMProductionRole",
				Description:     "Production access for APM operations",
				MFARequired:     true,
				SessionDuration: 1800,
				ExternalID:      "prod-role-external-id",
			},
		},
	}
	
	err = accountManager.AddAccount(prodAccount)
	if err != nil {
		fmt.Printf("Error adding prod account: %v\n", err)
	} else {
		fmt.Printf("Added production account: %s\n", prodAccount.AccountName)
	}
	
	// List all accounts
	accounts := accountManager.ListAccounts()
	fmt.Printf("Total configured accounts: %d\n", len(accounts))
	
	// Get accounts by environment
	devAccounts := accountManager.GetAccountsByEnvironment("dev")
	prodAccounts := accountManager.GetAccountsByEnvironment("prod")
	fmt.Printf("Development accounts: %d\n", len(devAccounts))
	fmt.Printf("Production accounts: %d\n", len(prodAccounts))

	// Example 8: Configuration Validation
	fmt.Println("\n8. Configuration Validation")
	fmt.Println("---------------------------")
	
	validationResult, err := accountManager.ValidateConfiguration(ctx)
	if err != nil {
		fmt.Printf("Error validating configuration: %v\n", err)
	} else {
		fmt.Printf("Configuration is valid: %v\n", validationResult.IsValid)
		fmt.Printf("Validated accounts: %d\n", len(validationResult.Accounts))
		
		for _, accountVal := range validationResult.Accounts {
			fmt.Printf("  Account %s: valid=%v, roles=%d\n", 
				accountVal.AccountID, accountVal.IsValid, len(accountVal.Roles))
			if len(accountVal.ValidationErrors) > 0 {
				fmt.Printf("    Errors: %v\n", accountVal.ValidationErrors)
			}
		}
	}

	// Example 9: Save Configuration
	fmt.Println("\n9. Save Configuration")
	fmt.Println("--------------------")
	
	// Save to local file
	configPath := "/tmp/apm-accounts.json"
	err = accountManager.SaveConfig(ctx, configPath)
	if err != nil {
		fmt.Printf("Error saving config to file: %v\n", err)
	} else {
		fmt.Printf("Configuration saved to: %s\n", configPath)
	}
	
	// Example of saving to S3 (commented out as it requires actual S3 access)
	// s3ConfigPath := "s3://apm-config-bucket/accounts/multi-account-config.json"
	// err = accountManager.SaveConfig(ctx, s3ConfigPath)
	// if err != nil {
	// 	fmt.Printf("Error saving config to S3: %v\n", err)
	// } else {
	// 	fmt.Printf("Configuration saved to S3: %s\n", s3ConfigPath)
	// }

	fmt.Println("\n✅ Cross-Account Role Assumption Demo Completed!")
	fmt.Println("\nKey Features Demonstrated:")
	fmt.Println("- Basic cross-account role assumption")
	fmt.Println("- MFA-based role assumption for enhanced security")
	fmt.Println("- Role chaining for complex multi-account scenarios")
	fmt.Println("- External ID support for partner integrations")
	fmt.Println("- Automatic session refresh and management")
	fmt.Println("- Role validation and trust policy analysis")
	fmt.Println("- Multi-account configuration management")
	fmt.Println("- Configuration validation and error handling")
	fmt.Println("- Configuration persistence (file and S3)")
}