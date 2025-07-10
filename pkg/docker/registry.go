package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/docker/docker/api/types/registry"
	"golang.org/x/oauth2/google"
)

// RegistryAuthenticator handles authentication for different container registries
type RegistryAuthenticator struct {
	credentials map[RegistryType]RegistryCredentials
}

// NewRegistryAuthenticator creates a new registry authenticator
func NewRegistryAuthenticator() *RegistryAuthenticator {
	return &RegistryAuthenticator{
		credentials: make(map[RegistryType]RegistryCredentials),
	}
}

// AddCredentials adds credentials for a registry type
func (r *RegistryAuthenticator) AddCredentials(registryType RegistryType, creds RegistryCredentials) {
	r.credentials[registryType] = creds
}

// GetAuthConfig returns Docker auth configuration for a registry
func (r *RegistryAuthenticator) GetAuthConfig(registryType RegistryType, registryURL string) (registry.AuthConfig, error) {
	switch registryType {
	case RegistryTypeDockerHub:
		return r.getDockerHubAuth()
	case RegistryTypeECR:
		return r.getECRAuth(registryURL)
	case RegistryTypeGCR:
		return r.getGCRAuth()
	case RegistryTypeACR:
		return r.getACRAuth(registryURL)
	default:
		return r.getCustomAuth(registryURL)
	}
}

// Docker Hub authentication
func (r *RegistryAuthenticator) getDockerHubAuth() (registry.AuthConfig, error) {
	creds, ok := r.credentials[RegistryTypeDockerHub]
	if !ok {
		return registry.AuthConfig{}, fmt.Errorf("Docker Hub credentials not configured")
	}

	return registry.AuthConfig{
		Username:      creds.Username,
		Password:      creds.Password,
		Email:         creds.Email,
		ServerAddress: "https://index.docker.io/v1/",
	}, nil
}

// Amazon ECR authentication
func (r *RegistryAuthenticator) getECRAuth(registryURL string) (registry.AuthConfig, error) {
	// Extract region from registry URL
	// Format: <account>.dkr.ecr.<region>.amazonaws.com
	parts := strings.Split(registryURL, ".")
	if len(parts) < 5 || parts[1] != "dkr" || parts[2] != "ecr" {
		return registry.AuthConfig{}, fmt.Errorf("invalid ECR registry URL: %s", registryURL)
	}
	region := parts[3]

	// Create AWS session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		return registry.AuthConfig{}, fmt.Errorf("failed to create AWS session: %w", err)
	}

	// Get ECR authorization token
	svc := ecr.New(sess)
	result, err := svc.GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return registry.AuthConfig{}, fmt.Errorf("failed to get ECR auth token: %w", err)
	}

	if len(result.AuthorizationData) == 0 {
		return registry.AuthConfig{}, fmt.Errorf("no ECR authorization data received")
	}

	// Decode authorization token
	authData := result.AuthorizationData[0]
	decodedToken, err := base64.StdEncoding.DecodeString(*authData.AuthorizationToken)
	if err != nil {
		return registry.AuthConfig{}, fmt.Errorf("failed to decode ECR auth token: %w", err)
	}

	// Token format is username:password
	tokenParts := strings.SplitN(string(decodedToken), ":", 2)
	if len(tokenParts) != 2 {
		return registry.AuthConfig{}, fmt.Errorf("invalid ECR auth token format")
	}

	return registry.AuthConfig{
		Username:      tokenParts[0],
		Password:      tokenParts[1],
		ServerAddress: *authData.ProxyEndpoint,
	}, nil
}

// Google Container Registry authentication
func (r *RegistryAuthenticator) getGCRAuth() (registry.AuthConfig, error) {
	// Try to get credentials from environment or metadata service
	ctx := context.Background()
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return registry.AuthConfig{}, fmt.Errorf("failed to find GCP credentials: %w", err)
	}

	// Get access token
	token, err := creds.TokenSource.Token()
	if err != nil {
		return registry.AuthConfig{}, fmt.Errorf("failed to get GCP access token: %w", err)
	}

	return registry.AuthConfig{
		Username:      "_token",
		Password:      token.AccessToken,
		ServerAddress: "https://gcr.io",
	}, nil
}

// Azure Container Registry authentication
func (r *RegistryAuthenticator) getACRAuth(registryURL string) (registry.AuthConfig, error) {
	creds, ok := r.credentials[RegistryTypeACR]
	if !ok {
		return registry.AuthConfig{}, fmt.Errorf("ACR credentials not configured")
	}

	// ACR can use service principal or admin credentials
	return registry.AuthConfig{
		Username:      creds.Username,
		Password:      creds.Password,
		ServerAddress: fmt.Sprintf("https://%s", registryURL),
	}, nil
}

// Custom registry authentication
func (r *RegistryAuthenticator) getCustomAuth(registryURL string) (registry.AuthConfig, error) {
	creds, ok := r.credentials[RegistryTypeCustom]
	if !ok {
		return registry.AuthConfig{}, fmt.Errorf("custom registry credentials not configured")
	}

	authConfig := registry.AuthConfig{
		Username:      creds.Username,
		Password:      creds.Password,
		Email:         creds.Email,
		ServerAddress: registryURL,
	}

	// If auth field is provided, use it directly
	if creds.Auth != "" {
		authConfig.Auth = creds.Auth
	}

	return authConfig, nil
}

// RegistryManager manages container registry operations
type RegistryManager struct {
	authenticator *RegistryAuthenticator
	client        *Client
}

// NewRegistryManager creates a new registry manager
func NewRegistryManager(client *Client) *RegistryManager {
	return &RegistryManager{
		authenticator: NewRegistryAuthenticator(),
		client:        client,
	}
}

// PushImage pushes an image to a specific registry
func (rm *RegistryManager) PushImage(ctx context.Context, image string, registryType RegistryType) error {
	// Parse image reference
	ref := parseImageReference(image)

	// Get authentication
	authConfig, err := rm.authenticator.GetAuthConfig(registryType, ref.Registry)
	if err != nil {
		return fmt.Errorf("failed to get registry auth: %w", err)
	}

	// Tag image for target registry if needed
	targetImage := image
	if registryType != RegistryTypeDockerHub && !strings.Contains(image, "/") {
		targetImage = fmt.Sprintf("%s/%s", ref.Registry, image)
	}

	// Push the image
	return rm.client.PushWithAuth(ctx, targetImage, authConfig)
}

// PullImage pulls an image from a specific registry
func (rm *RegistryManager) PullImage(ctx context.Context, image string, registryType RegistryType) error {
	// Parse image reference
	ref := parseImageReference(image)

	// Get authentication
	authConfig, err := rm.authenticator.GetAuthConfig(registryType, ref.Registry)
	if err != nil {
		return fmt.Errorf("failed to get registry auth: %w", err)
	}

	// Pull the image
	return rm.client.PullWithAuth(ctx, image, authConfig)
}

// ScanImage scans an image in a registry for vulnerabilities
func (rm *RegistryManager) ScanImage(ctx context.Context, image string, registryType RegistryType) (*ScanReport, error) {
	switch registryType {
	case RegistryTypeECR:
		return rm.scanECRImage(ctx, image)
	case RegistryTypeGCR:
		return rm.scanGCRImage(ctx, image)
	case RegistryTypeACR:
		return rm.scanACRImage(ctx, image)
	default:
		// Use generic scanning tool
		return rm.scanWithTrivy(ctx, image)
	}
}

// ECR image scanning
func (rm *RegistryManager) scanECRImage(ctx context.Context, image string) (*ScanReport, error) {
	ref := parseImageReference(image)

	// Create ECR client
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(ref.Region),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	svc := ecr.New(sess)

	// Start image scan
	_, err = svc.StartImageScan(&ecr.StartImageScanInput{
		RepositoryName: aws.String(ref.Repository),
		ImageId: &ecr.ImageIdentifier{
			ImageTag: aws.String(ref.Tag),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start ECR image scan: %w", err)
	}

	// Wait for scan to complete
	time.Sleep(5 * time.Second)

	// Get scan findings
	findings, err := svc.DescribeImageScanFindings(&ecr.DescribeImageScanFindingsInput{
		RepositoryName: aws.String(ref.Repository),
		ImageId: &ecr.ImageIdentifier{
			ImageTag: aws.String(ref.Tag),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get ECR scan findings: %w", err)
	}

	// Convert to standard scan report
	return convertECRScanReport(findings), nil
}

// GCR image scanning (placeholder)
func (rm *RegistryManager) scanGCRImage(ctx context.Context, image string) (*ScanReport, error) {
	// GCR uses Container Analysis API
	// Implementation would use Google Cloud SDK
	return rm.scanWithTrivy(ctx, image)
}

// ACR image scanning (placeholder)
func (rm *RegistryManager) scanACRImage(ctx context.Context, image string) (*ScanReport, error) {
	// ACR uses Azure Defender for container registries
	// Implementation would use Azure SDK
	return rm.scanWithTrivy(ctx, image)
}

// Generic image scanning with Trivy
func (rm *RegistryManager) scanWithTrivy(ctx context.Context, image string) (*ScanReport, error) {
	// This would integrate with Trivy or another scanning tool
	// For now, return a placeholder
	return &ScanReport{
		ImageID:  image,
		ScanTime: time.Now(),
		Critical: 0,
		High:     0,
		Medium:   0,
		Low:      0,
		Unknown:  0,
	}, nil
}

// ImageReference represents a parsed container image reference
type ImageReference struct {
	Registry   string
	Repository string
	Tag        string
	Digest     string
	Region     string // For cloud registries
}

// parseImageReference parses a container image reference
func parseImageReference(image string) ImageReference {
	ref := ImageReference{
		Tag: "latest",
	}

	// Simple parsing logic
	parts := strings.Split(image, "/")
	if len(parts) > 1 && strings.Contains(parts[0], ".") {
		ref.Registry = parts[0]
		image = strings.Join(parts[1:], "/")
	}

	// Extract tag
	if idx := strings.LastIndex(image, ":"); idx > 0 {
		ref.Repository = image[:idx]
		ref.Tag = image[idx+1:]
	} else {
		ref.Repository = image
	}

	// Extract region for ECR
	if strings.Contains(ref.Registry, "ecr") && strings.Contains(ref.Registry, "amazonaws.com") {
		registryParts := strings.Split(ref.Registry, ".")
		if len(registryParts) >= 4 {
			ref.Region = registryParts[3]
		}
	}

	return ref
}

// convertECRScanReport converts ECR scan findings to standard format
func convertECRScanReport(findings *ecr.DescribeImageScanFindingsOutput) *ScanReport {
	report := &ScanReport{
		ScanTime:        time.Now(),
		Vulnerabilities: make([]Vulnerability, 0),
	}

	if findings.ImageScanFindings == nil {
		return report
	}

	// Count vulnerabilities by severity
	severityCounts := findings.ImageScanFindings.FindingSeverityCounts
	if severityCounts != nil {
		for severity, count := range severityCounts {
			switch severity {
			case "CRITICAL":
				report.Critical = int(*count)
			case "HIGH":
				report.High = int(*count)
			case "MEDIUM":
				report.Medium = int(*count)
			case "LOW":
				report.Low = int(*count)
			case "INFORMATIONAL", "UNDEFINED":
				report.Unknown += int(*count)
			}
		}
	}

	// Convert findings to vulnerabilities
	for _, finding := range findings.ImageScanFindings.Findings {
		vuln := Vulnerability{
			ID:       *finding.Name,
			Severity: *finding.Severity,
		}

		if finding.Description != nil {
			vuln.Description = *finding.Description
		}

		if len(finding.Attributes) > 0 {
			for _, attr := range finding.Attributes {
				if *attr.Key == "package_name" {
					vuln.Package = *attr.Value
				} else if *attr.Key == "package_version" {
					vuln.Version = *attr.Value
				}
			}
		}

		report.Vulnerabilities = append(report.Vulnerabilities, vuln)
	}

	return report
}
