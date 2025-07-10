package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/apm/pkg/cloud"
)

func main() {
	// Example demonstrating AWS role chaining for complex multi-account scenarios
	fmt.Println("AWS Role Chain Example")
	fmt.Println("======================")

	// Create AWS provider
	provider, err := cloud.NewProvider(cloud.ProviderAWS, &cloud.ProviderConfig{
		Region:    "us-east-1",
		DebugMode: true,
	})
	if err != nil {
		log.Fatalf("Failed to create AWS provider: %v", err)
	}

	awsProvider, ok := provider.(*cloud.AWSProvider)
	if !ok {
		log.Fatal("Provider is not an AWS provider")
	}

	ctx := context.Background()

	// Example 1: Simple two-hop role chain
	fmt.Println("\nExample 1: Simple Two-Hop Role Chain")
	fmt.Println("-------------------------------------")
	demonstrateSimpleChain(ctx, awsProvider)

	// Example 2: Complex multi-account chain with external IDs
	fmt.Println("\nExample 2: Complex Multi-Account Chain")
	fmt.Println("--------------------------------------")
	demonstrateComplexChain(ctx, awsProvider)

	// Example 3: Role chain with MFA
	fmt.Println("\nExample 3: Role Chain with MFA")
	fmt.Println("-------------------------------")
	demonstrateChainWithMFA(ctx, awsProvider)

	// Example 4: Using the enhanced chain manager
	fmt.Println("\nExample 4: Enhanced Chain Manager")
	fmt.Println("---------------------------------")
	demonstrateEnhancedChainManager(ctx, awsProvider)
}

func demonstrateSimpleChain(ctx context.Context, provider *cloud.AWSProvider) {
	// Define a simple two-hop chain:
	// Current Account -> Development Account -> Production Account
	roleChain := []*cloud.RoleChainStep{
		{
			RoleArn:     "arn:aws:iam::123456789012:role/DevelopmentAccessRole",
			SessionName: "dev-access",
		},
		{
			RoleArn:     "arn:aws:iam::987654321098:role/ProductionReadOnlyRole",
			SessionName: "prod-readonly",
		},
	}

	fmt.Println("Attempting to assume role chain:")
	for i, step := range roleChain {
		fmt.Printf("  Step %d: %s\n", i+1, step.RoleArn)
	}

	// Execute the chain
	credentials, err := provider.AssumeRoleChain(ctx, roleChain)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("Note: This example requires actual AWS roles to be set up")
		return
	}

	fmt.Printf("Successfully assumed role chain!\n")
	fmt.Printf("Final Account: %s\n", credentials.Account)
	fmt.Printf("Expires at: %v\n", credentials.Expiry)
}

func demonstrateComplexChain(ctx context.Context, provider *cloud.AWSProvider) {
	// Define a complex chain with external IDs and custom options
	roleChain := []*cloud.RoleChainStep{
		{
			RoleArn:     "arn:aws:iam::111111111111:role/OrganizationAccessRole",
			SessionName: "org-access",
			Options: &cloud.AssumeRoleOptions{
				DurationSeconds: 3600, // 1 hour
				Tags: map[string]string{
					"Purpose": "CrossAccountAccess",
					"Team":    "Platform",
				},
			},
		},
		{
			RoleArn:     "arn:aws:iam::222222222222:role/PartnerIntegrationRole",
			ExternalID:  "unique-partner-identifier-12345",
			SessionName: "partner-access",
			Options: &cloud.AssumeRoleOptions{
				DurationSeconds: 1800, // 30 minutes
				Policy: `{
					"Version": "2012-10-17",
					"Statement": [{
						"Effect": "Allow",
						"Action": ["s3:GetObject"],
						"Resource": ["arn:aws:s3:::partner-bucket/*"]
					}]
				}`,
			},
		},
		{
			RoleArn:     "arn:aws:iam::333333333333:role/AuditRole",
			SessionName: "audit-access",
			Options: &cloud.AssumeRoleOptions{
				DurationSeconds: 900, // 15 minutes
				Region:         "eu-west-1", // Different region
			},
		},
	}

	fmt.Println("Complex chain configuration:")
	for i, step := range roleChain {
		fmt.Printf("  Step %d: %s\n", i+1, step.RoleArn)
		if step.ExternalID != "" {
			fmt.Printf("    - External ID: %s\n", step.ExternalID)
		}
		if step.Options != nil && step.Options.DurationSeconds > 0 {
			fmt.Printf("    - Duration: %d seconds\n", step.Options.DurationSeconds)
		}
	}

	credentials, err := provider.AssumeRoleChain(ctx, roleChain)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Successfully completed complex chain!\n")
	fmt.Printf("Final credentials in region: %s\n", credentials.Region)
}

func demonstrateChainWithMFA(ctx context.Context, provider *cloud.AWSProvider) {
	// Get MFA device ARN and token from environment or prompt
	mfaDevice := os.Getenv("AWS_MFA_DEVICE_ARN")
	if mfaDevice == "" {
		fmt.Println("Skipping MFA example: AWS_MFA_DEVICE_ARN not set")
		return
	}

	// In a real application, you would prompt for the MFA token
	fmt.Print("Enter MFA token code: ")
	var mfaToken string
	fmt.Scanln(&mfaToken)

	roleChain := []*cloud.RoleChainStep{
		{
			RoleArn:     "arn:aws:iam::123456789012:role/MFAProtectedRole",
			SessionName: "mfa-session",
			Options: &cloud.AssumeRoleOptions{
				MFASerialNumber: mfaDevice,
				MFATokenCode:    mfaToken,
				DurationSeconds: 3600,
			},
		},
		{
			RoleArn:     "arn:aws:iam::987654321098:role/HighlySecureRole",
			SessionName: "secure-session",
		},
	}

	credentials, err := provider.AssumeRoleChain(ctx, roleChain)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Successfully assumed MFA-protected chain!\n")
	fmt.Printf("Session valid until: %v\n", credentials.Expiry)
}

func demonstrateEnhancedChainManager(ctx context.Context, provider *cloud.AWSProvider) {
	// Create a chain manager for advanced features
	chainManager := cloud.NewRoleChainManager(provider)
	defer chainManager.Close()

	// Configure the chain behavior
	config := &cloud.RoleChainConfig{
		MaxSteps:              4,
		DefaultDuration:       3600,
		RefreshBeforeExpiry:   5 * time.Minute,
		EnableAutoRefresh:     true,
		RetryAttempts:         3,
		RetryDelay:            2 * time.Second,
		ConcurrentAssumptions: false,
	}

	// Define the role chain
	roleChain := []*cloud.RoleChainStep{
		{
			RoleArn:     "arn:aws:iam::123456789012:role/Step1Role",
			SessionName: "enhanced-step-1",
		},
		{
			RoleArn:     "arn:aws:iam::987654321098:role/Step2Role",
			SessionName: "enhanced-step-2",
			ExternalID:  "security-token-xyz",
		},
	}

	fmt.Println("Using enhanced chain manager with:")
	fmt.Printf("  - Auto-refresh: %v\n", config.EnableAutoRefresh)
	fmt.Printf("  - Retry attempts: %d\n", config.RetryAttempts)
	fmt.Printf("  - Max chain length: %d\n", config.MaxSteps)

	// Execute the chain
	session, err := chainManager.AssumeRoleChain(ctx, roleChain, config)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\nChain session created:\n")
	fmt.Printf("  Chain ID: %s\n", session.ChainID)
	fmt.Printf("  Steps completed: %d\n", len(session.Credentials))
	fmt.Printf("  Created at: %v\n", session.CreatedAt)

	// Demonstrate session management
	fmt.Println("\nActive sessions:")
	for _, s := range chainManager.ListSessions() {
		fmt.Printf("  - %s (created: %v)\n", s.ChainID, s.CreatedAt)
	}

	// Validate individual steps
	fmt.Println("\nValidating chain steps:")
	for i, step := range roleChain {
		var prevCreds *cloud.Credentials
		if i > 0 && i <= len(session.Credentials) {
			prevCreds = session.Credentials[i-1].Credentials
		}
		
		err := chainManager.ValidateChainStep(ctx, step, prevCreds)
		if err != nil {
			fmt.Printf("  Step %d validation failed: %v\n", i+1, err)
		} else {
			fmt.Printf("  Step %d validation passed\n", i+1)
		}
	}
}

// Example of using the enhanced assume role chain method
func demonstrateEnhancedMethod(ctx context.Context, provider *cloud.AWSProvider) {
	roleChain := []*cloud.RoleChainStep{
		{
			RoleArn: "arn:aws:iam::123456789012:role/FirstRole",
		},
		{
			RoleArn:    "arn:aws:iam::987654321098:role/SecondRole",
			ExternalID: "partner-id",
		},
	}

	config := &cloud.RoleChainConfig{
		DefaultDuration:   3600,
		EnableAutoRefresh: true,
		RetryAttempts:     3,
	}

	credentials, err := provider.AssumeRoleChainEnhanced(ctx, roleChain, config)
	if err != nil {
		log.Printf("Failed to assume role chain: %v", err)
		return
	}

	fmt.Printf("Successfully assumed enhanced role chain\n")
	fmt.Printf("Final credentials expire at: %v\n", credentials.Expiry)
}