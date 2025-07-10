package cloud

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// BaseCLIDetector provides common CLI detection functionality
type BaseCLIDetector struct {
	provider     Provider
	commands     []string
	minVersion   string
	versionFlag  string
	versionRegex *regexp.Regexp
}

// NewBaseCLIDetector creates a new base CLI detector
func NewBaseCLIDetector(provider Provider, commands []string, minVersion, versionFlag string, versionPattern string) *BaseCLIDetector {
	return &BaseCLIDetector{
		provider:     provider,
		commands:     commands,
		minVersion:   minVersion,
		versionFlag:  versionFlag,
		versionRegex: regexp.MustCompile(versionPattern),
	}
}

// Detect attempts to detect the CLI installation
func (d *BaseCLIDetector) Detect() (*CLIStatus, error) {
	for _, cmd := range d.commands {
		path, err := exec.LookPath(cmd)
		if err != nil {
			continue
		}

		// Get version
		version, err := d.getVersion(path)
		if err != nil {
			continue
		}

		// Get config path
		configPath := d.getConfigPath()

		return &CLIStatus{
			Installed:   true,
			Version:     version,
			Path:        path,
			ConfigPath:  configPath,
			MinVersion:  d.minVersion,
			IsSupported: d.ValidateVersion(version),
		}, nil
	}

	return &CLIStatus{
		Installed:   false,
		MinVersion:  d.minVersion,
		IsSupported: false,
	}, nil
}

// getVersion extracts version from CLI output
func (d *BaseCLIDetector) getVersion(cliPath string) (string, error) {
	cmd := exec.Command(cliPath, d.versionFlag)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get version: %w", err)
	}

	matches := d.versionRegex.FindStringSubmatch(string(output))
	if len(matches) < 2 {
		return "", fmt.Errorf("could not parse version from output")
	}

	return matches[1], nil
}

// getConfigPath returns the configuration path for the CLI
func (d *BaseCLIDetector) getConfigPath() string {
	home, _ := os.UserHomeDir()

	switch d.provider {
	case ProviderAWS:
		return filepath.Join(home, ".aws")
	case ProviderAzure:
		return filepath.Join(home, ".azure")
	case ProviderGCP:
		return filepath.Join(home, ".config", "gcloud")
	default:
		return ""
	}
}

// ValidateVersion checks if the version meets minimum requirements
func (d *BaseCLIDetector) ValidateVersion(version string) bool {
	// Simple version comparison - can be enhanced
	return version >= d.minVersion
}

// GetMinVersion returns the minimum required version
func (d *BaseCLIDetector) GetMinVersion() string {
	return d.minVersion
}

// AWSCLIDetector detects AWS CLI with enhanced features
type AWSCLIDetector struct {
	*BaseCLIDetector
	logger CLILogger
}

// CLILogger interface for logging CLI detection events
type CLILogger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
}

// DefaultLogger provides a simple implementation of CLILogger
type DefaultLogger struct{}

func (l *DefaultLogger) Debug(msg string, fields ...interface{}) { /* noop for now */ }
func (l *DefaultLogger) Info(msg string, fields ...interface{})  { fmt.Printf("INFO: %s\n", msg) }
func (l *DefaultLogger) Warn(msg string, fields ...interface{})  { fmt.Printf("WARN: %s\n", msg) }
func (l *DefaultLogger) Error(msg string, fields ...interface{}) { fmt.Printf("ERROR: %s\n", msg) }

// NewAWSCLIDetector creates a new enhanced AWS CLI detector
func NewAWSCLIDetector() *AWSCLIDetector {
	return &AWSCLIDetector{
		BaseCLIDetector: NewBaseCLIDetector(
			ProviderAWS,
			[]string{"aws"},
			"2.0.0",
			"--version",
			`aws-cli/(\d+\.\d+\.\d+)`,
		),
		logger: &DefaultLogger{},
	}
}

// NewAWSCLIDetectorWithLogger creates a new AWS CLI detector with custom logger
func NewAWSCLIDetectorWithLogger(logger CLILogger) *AWSCLIDetector {
	return &AWSCLIDetector{
		BaseCLIDetector: NewBaseCLIDetector(
			ProviderAWS,
			[]string{"aws"},
			"2.0.0",
			"--version",
			`aws-cli/(\d+\.\d+\.\d+)`,
		),
		logger: logger,
	}
}

// Detect performs enhanced AWS CLI detection with comprehensive logging and error handling
func (d *AWSCLIDetector) Detect() (*CLIStatus, error) {
	d.logger.Info("Starting AWS CLI detection")

	status := &CLIStatus{
		Installed:   false,
		MinVersion:  d.minVersion,
		IsSupported: false,
	}

	// Try multiple detection strategies
	detectionResults := d.detectMultiplePaths()

	if len(detectionResults) == 0 {
		d.logger.Warn("No AWS CLI installations found")
		return status, nil
	}

	// Use the best installation found
	bestResult := d.selectBestInstallation(detectionResults)

	status.Installed = true
	status.Version = bestResult.Version
	status.Path = bestResult.Path
	status.ConfigPath = d.getConfigPath()
	status.IsSupported = d.ValidateVersionSemantic(bestResult.Version)

	// Log warnings for older versions or multiple installations
	if len(detectionResults) > 1 {
		d.logger.Warn(fmt.Sprintf("Multiple AWS CLI installations detected (%d)", len(detectionResults)))
	}

	if !status.IsSupported {
		d.logger.Warn(fmt.Sprintf("AWS CLI version %s is below minimum required %s", status.Version, status.MinVersion))
	}

	d.logger.Info(fmt.Sprintf("AWS CLI detection completed: version=%s, path=%s", status.Version, status.Path))

	return status, nil
}

// AWSCLIInstallation represents a detected AWS CLI installation
type AWSCLIInstallation struct {
	Path          string
	Version       string
	MajorVersion  int
	InstallMethod string
	IsV1          bool
	ExecutionTime time.Duration
}

// detectMultiplePaths attempts to detect AWS CLI installations in various locations
func (d *AWSCLIDetector) detectMultiplePaths() []AWSCLIInstallation {
	var results []AWSCLIInstallation

	// Standard PATH detection
	if path, err := exec.LookPath("aws"); err == nil {
		if installation, err := d.analyzeInstallation(path); err == nil {
			results = append(results, installation)
		}
	}

	// Platform-specific additional paths
	additionalPaths := d.getPlatformSpecificPaths()
	for _, path := range additionalPaths {
		if installation, err := d.analyzeInstallation(path); err == nil {
			// Avoid duplicates
			isDuplicate := false
			for _, existing := range results {
				if existing.Path == installation.Path {
					isDuplicate = true
					break
				}
			}
			if !isDuplicate {
				results = append(results, installation)
			}
		}
	}

	return results
}

// getPlatformSpecificPaths returns additional paths to check for AWS CLI
func (d *AWSCLIDetector) getPlatformSpecificPaths() []string {
	var paths []string

	switch runtime.GOOS {
	case "darwin":
		paths = []string{
			"/usr/local/bin/aws",
			"/opt/homebrew/bin/aws",
			"/usr/local/aws-cli/bin/aws",
		}
	case "linux":
		paths = []string{
			"/usr/local/bin/aws",
			"/usr/bin/aws",
			"/opt/aws-cli/bin/aws",
			"/snap/bin/aws",
		}
	case "windows":
		paths = []string{
			"C:\\Program Files\\Amazon\\AWSCLIV2\\aws.exe",
			"C:\\Program Files (x86)\\Amazon\\AWSCLIV2\\aws.exe",
		}
	}

	return paths
}

// analyzeInstallation analyzes a specific AWS CLI installation
func (d *AWSCLIDetector) analyzeInstallation(path string) (AWSCLIInstallation, error) {
	startTime := time.Now()

	// Get version information
	cmd := exec.Command(path, "--version")
	output, err := cmd.Output()
	if err != nil {
		return AWSCLIInstallation{}, fmt.Errorf("failed to get version from %s: %w", path, err)
	}

	executionTime := time.Since(startTime)
	versionOutput := strings.TrimSpace(string(output))

	// Parse version with multiple patterns
	version, majorVersion, isV1 := d.parseVersionOutput(versionOutput)
	if version == "" {
		return AWSCLIInstallation{}, fmt.Errorf("failed to parse version from output: %s", versionOutput)
	}

	// Determine installation method
	installMethod := d.detectInstallMethod(path)

	return AWSCLIInstallation{
		Path:          path,
		Version:       version,
		MajorVersion:  majorVersion,
		InstallMethod: installMethod,
		IsV1:          isV1,
		ExecutionTime: executionTime,
	}, nil
}

// parseVersionOutput parses version information from AWS CLI output with multiple patterns
func (d *AWSCLIDetector) parseVersionOutput(output string) (version string, majorVersion int, isV1 bool) {
	// AWS CLI patterns with comprehensive support for different formats
	patterns := []string{
		`aws-cli/(\d+\.\d+\.\d+\.\w+\d*)`, // Pre-release with alphanumeric: aws-cli/2.5.0.dev0
		`aws-cli/(\d+\.\d+\.\d+)-(\w+)`,   // With build info: aws-cli/2.4.29-1.el8
		`aws-cli/(\d+\.\d+\.\d+)`,         // Standard: aws-cli/2.4.29 or aws-cli/1.20.30
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(output)
		if len(matches) >= 2 {
			version = matches[1]
			// For patterns with build info, use first match group only
			if len(matches) > 2 && strings.Contains(pattern, `)-(\w+)`) {
				version = matches[1] // Keep only version part, ignore build suffix
			}

			if parts := strings.Split(version, "."); len(parts) > 0 {
				if major, err := strconv.Atoi(parts[0]); err == nil {
					majorVersion = major
					isV1 = major == 1
					return
				}
			}
		}
	}

	return "", 0, false
}

// detectInstallMethod attempts to determine how AWS CLI was installed
func (d *AWSCLIDetector) detectInstallMethod(path string) string {
	switch runtime.GOOS {
	case "darwin":
		if strings.Contains(path, "/usr/local/aws-cli/") {
			return "installer"
		}
		if strings.Contains(path, "/opt/homebrew/") || strings.Contains(path, "/usr/local/") {
			return "homebrew"
		}
	case "linux":
		if strings.Contains(path, "/snap/") {
			return "snap"
		}
		if strings.Contains(path, "/opt/aws-cli/") {
			return "installer"
		}
		if strings.Contains(path, "/usr/bin/") {
			return "package-manager"
		}
	case "windows":
		if strings.Contains(path, "Program Files") {
			return "installer"
		}
	}

	// Try to detect pip installation (common method)
	if d.isPipInstallation(path) {
		return "pip"
	}

	return "unknown"
}

// isPipInstallation checks if AWS CLI was installed via pip
func (d *AWSCLIDetector) isPipInstallation(path string) bool {
	// This is a heuristic check - pip installations typically have Python in the path
	cmd := exec.Command("python", "-m", "pip", "show", "awscli")
	err := cmd.Run()
	return err == nil
}

// selectBestInstallation selects the best AWS CLI installation from multiple candidates
func (d *AWSCLIDetector) selectBestInstallation(installations []AWSCLIInstallation) AWSCLIInstallation {
	if len(installations) == 1 {
		return installations[0]
	}

	// Prioritize by version (higher is better)
	best := installations[0]
	for _, installation := range installations[1:] {
		if d.isVersionNewer(installation.Version, best.Version) {
			best = installation
		} else if installation.Version == best.Version {
			// Same version, prefer v2 over v1
			if !installation.IsV1 && best.IsV1 {
				best = installation
			} else if installation.IsV1 == best.IsV1 {
				// Same major version, prefer faster execution
				if installation.ExecutionTime < best.ExecutionTime {
					best = installation
				}
			}
		}
	}

	return best
}

// ValidateVersionSemantic performs semantic version validation
func (d *AWSCLIDetector) ValidateVersionSemantic(version string) bool {
	return d.isVersionNewer(version, d.minVersion) || version == d.minVersion
}

// isVersionNewer compares two semantic versions
func (d *AWSCLIDetector) isVersionNewer(version1, version2 string) bool {
	v1Parts := d.parseSemanticVersion(version1)
	v2Parts := d.parseSemanticVersion(version2)

	for i := 0; i < 3; i++ {
		if v1Parts[i] > v2Parts[i] {
			return true
		} else if v1Parts[i] < v2Parts[i] {
			return false
		}
	}

	return false // versions are equal
}

// parseSemanticVersion parses a semantic version string into [major, minor, patch]
func (d *AWSCLIDetector) parseSemanticVersion(version string) [3]int {
	parts := strings.Split(version, ".")
	result := [3]int{0, 0, 0}

	for i := 0; i < len(parts) && i < 3; i++ {
		if num, err := strconv.Atoi(parts[i]); err == nil {
			result[i] = num
		}
	}

	return result
}

// GetInstallInstructions returns enhanced installation instructions for AWS CLI
func (d *AWSCLIDetector) GetInstallInstructions() string {
	switch runtime.GOOS {
	case "darwin":
		return `Install AWS CLI v2 on macOS:

Recommended methods:
1. Official installer (recommended):
   curl "https://awscli.amazonaws.com/AWSCLIV2.pkg" -o "AWSCLIV2.pkg"
   sudo installer -pkg AWSCLIV2.pkg -target /
   
2. Using Homebrew:
   brew install awscli
   
3. Verify installation:
   aws --version
   
Note: AWS CLI v1 is deprecated. Please use v2.`
	case "linux":
		return `Install AWS CLI v2 on Linux:

Recommended method:
1. Download and install:
   curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
   unzip awscliv2.zip
   sudo ./aws/install
   
2. Alternative - using package manager:
   # Ubuntu/Debian: sudo apt-get install awscli
   # RHEL/CentOS: sudo yum install awscli
   # Arch: sudo pacman -S aws-cli
   
3. Verify installation:
   aws --version
   
Note: Ensure you install v2.0.0 or higher.`
	case "windows":
		return `Install AWS CLI v2 on Windows:

Recommended method:
1. Download installer from:
   https://awscli.amazonaws.com/AWSCLIV2.msi
   
2. Run the MSI installer

3. Alternative - using package manager:
   # Using Chocolatey: choco install awscli
   # Using Scoop: scoop install aws
   
4. Verify installation in PowerShell:
   aws --version
   
Note: Restart your terminal after installation.`
	default:
		return `Install AWS CLI v2:

Please visit the official installation guide:
https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html

Minimum required version: v2.0.0`
	}
}

// GetDetailedValidationResult provides comprehensive validation with actionable recommendations
func (d *AWSCLIDetector) GetDetailedValidationResult() (*AWSCLIValidationResult, error) {
	result := &AWSCLIValidationResult{
		Timestamp: time.Now(),
		Platform:  runtime.GOOS,
	}

	// Detect all installations
	installations := d.detectMultiplePaths()
	result.TotalInstallations = len(installations)

	if len(installations) == 0 {
		result.Status = CLIStatusNotFound
		result.ErrorType = ErrorTypeCLI
		result.ErrorMessage = "AWS CLI is not installed on this system"
		result.Recommendations = d.getInstallationRecommendations()
		return result, nil
	}

	// Analyze the best installation
	best := d.selectBestInstallation(installations)
	result.SelectedInstallation = &best
	result.AllInstallations = installations

	// Validate version
	if best.IsV1 {
		result.Status = CLIStatusDeprecated
		result.ErrorType = ErrorTypeCLI
		result.ErrorMessage = fmt.Sprintf("AWS CLI v1 (%s) is deprecated. Please upgrade to v2.0.0+", best.Version)
		result.Recommendations = d.getUpgradeRecommendations()
		result.Warnings = append(result.Warnings, "AWS CLI v1 will reach end-of-life support")
	} else if !d.ValidateVersionSemantic(best.Version) {
		result.Status = CLIStatusVersionTooOld
		result.ErrorType = ErrorTypeCLI
		result.ErrorMessage = fmt.Sprintf("AWS CLI version %s is below minimum required %s", best.Version, d.minVersion)
		result.Recommendations = d.getUpgradeRecommendations()
	} else {
		result.Status = CLIStatusOK
		result.SuccessMessage = fmt.Sprintf("AWS CLI v%s is properly installed and supported", best.Version)
	}

	// Add warnings for multiple installations
	if len(installations) > 1 {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Multiple AWS CLI installations detected (%d). Consider cleaning up unused installations.", len(installations)))
	}

	// Performance analysis
	if best.ExecutionTime > 5*time.Second {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("AWS CLI execution is slow (%v). This may impact deployment performance.", best.ExecutionTime))
	}

	return result, nil
}

// getInstallationRecommendations returns platform-specific installation recommendations
func (d *AWSCLIDetector) getInstallationRecommendations() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{
			"Install using the official installer: curl 'https://awscli.amazonaws.com/AWSCLIV2.pkg' -o 'AWSCLIV2.pkg' && sudo installer -pkg AWSCLIV2.pkg -target /",
			"Alternative: Install using Homebrew: brew install awscli",
			"Verify installation: aws --version",
		}
	case "linux":
		return []string{
			"Install using the official installer: curl 'https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip' -o 'awscliv2.zip' && unzip awscliv2.zip && sudo ./aws/install",
			"Alternative: Use your package manager (apt-get, yum, pacman)",
			"Verify installation: aws --version",
		}
	case "windows":
		return []string{
			"Download and run the MSI installer from: https://awscli.amazonaws.com/AWSCLIV2.msi",
			"Alternative: Use Chocolatey: choco install awscli",
			"Verify installation in PowerShell: aws --version",
		}
	default:
		return []string{
			"Visit the official installation guide: https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html",
			"Ensure you install version 2.0.0 or higher",
		}
	}
}

// getUpgradeRecommendations returns upgrade recommendations
func (d *AWSCLIDetector) getUpgradeRecommendations() []string {
	return []string{
		"Uninstall the current AWS CLI version",
		"Install AWS CLI v2 using the official installer",
		"Update your PATH if necessary",
		"Verify the upgrade: aws --version",
		"Test your existing AWS configurations still work",
	}
}

// AWSCLIValidationResult contains comprehensive validation results
type AWSCLIValidationResult struct {
	Timestamp            time.Time            `json:"timestamp"`
	Platform             string               `json:"platform"`
	Status               CLIStatusType        `json:"status"`
	ErrorType            ErrorType            `json:"error_type,omitempty"`
	ErrorMessage         string               `json:"error_message,omitempty"`
	SuccessMessage       string               `json:"success_message,omitempty"`
	TotalInstallations   int                  `json:"total_installations"`
	SelectedInstallation *AWSCLIInstallation  `json:"selected_installation,omitempty"`
	AllInstallations     []AWSCLIInstallation `json:"all_installations,omitempty"`
	Warnings             []string             `json:"warnings,omitempty"`
	Recommendations      []string             `json:"recommendations,omitempty"`
}

// CLIStatusType represents the status of CLI detection
type CLIStatusType string

const (
	CLIStatusOK            CLIStatusType = "ok"
	CLIStatusNotFound      CLIStatusType = "not_found"
	CLIStatusVersionTooOld CLIStatusType = "version_too_old"
	CLIStatusDeprecated    CLIStatusType = "deprecated"
	CLIStatusCorrupted     CLIStatusType = "corrupted"
	CLIStatusMultiple      CLIStatusType = "multiple_installations"
)

// CLI-specific error classifications using existing ErrorType
const (
	CLIErrorNotInstalled  = "cli_not_installed"
	CLIErrorVersionTooOld = "cli_version_too_old"
	CLIErrorCorrupted     = "cli_corrupted"
	CLIErrorPermission    = "cli_permission_denied"
)

// AzureCLIDetector detects Azure CLI
type AzureCLIDetector struct {
	*BaseCLIDetector
}

// NewAzureCLIDetector creates a new Azure CLI detector
func NewAzureCLIDetector() *AzureCLIDetector {
	return &AzureCLIDetector{
		BaseCLIDetector: NewBaseCLIDetector(
			ProviderAzure,
			[]string{"az"},
			"2.30.0",
			"--version",
			`azure-cli\s+(\d+\.\d+\.\d+)`,
		),
	}
}

// GetInstallInstructions returns installation instructions for Azure CLI
func (d *AzureCLIDetector) GetInstallInstructions() string {
	switch runtime.GOOS {
	case "darwin":
		return `Install Azure CLI on macOS:
1. Using Homebrew: brew update && brew install azure-cli
2. Verify: az --version`
	case "linux":
		return `Install Azure CLI on Linux:
1. Run: curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash
2. Verify: az --version`
	case "windows":
		return `Install Azure CLI on Windows:
1. Download installer from: https://aka.ms/installazurecliwindows
2. Run the installer
3. Verify in PowerShell: az --version`
	default:
		return "Please visit https://docs.microsoft.com/en-us/cli/azure/install-azure-cli for installation instructions"
	}
}

// GCPCLIDetector detects Google Cloud CLI
type GCPCLIDetector struct {
	*BaseCLIDetector
}

// NewGCPCLIDetector creates a new GCP CLI detector
func NewGCPCLIDetector() *GCPCLIDetector {
	return &GCPCLIDetector{
		BaseCLIDetector: NewBaseCLIDetector(
			ProviderGCP,
			[]string{"gcloud"},
			"400.0.0",
			"--version",
			`Google Cloud SDK (\d+\.\d+\.\d+)`,
		),
	}
}

// GetInstallInstructions returns installation instructions for Google Cloud CLI
func (d *GCPCLIDetector) GetInstallInstructions() string {
	switch runtime.GOOS {
	case "darwin":
		return `Install Google Cloud CLI on macOS:
1. Download from: https://cloud.google.com/sdk/docs/install#mac
2. Extract and run: ./google-cloud-sdk/install.sh
3. Initialize: ./google-cloud-sdk/bin/gcloud init
4. Verify: gcloud --version`
	case "linux":
		return `Install Google Cloud CLI on Linux:
1. Run: curl https://sdk.cloud.google.com | bash
2. Restart shell: exec -l $SHELL
3. Initialize: gcloud init
4. Verify: gcloud --version`
	case "windows":
		return `Install Google Cloud CLI on Windows:
1. Download installer from: https://cloud.google.com/sdk/docs/install#windows
2. Run the installer
3. Initialize: gcloud init
4. Verify in PowerShell: gcloud --version`
	default:
		return "Please visit https://cloud.google.com/sdk/docs/install for installation instructions"
	}
}

// DetectorFactory creates CLI detectors for different providers
type DetectorFactory struct{}

// NewDetectorFactory creates a new detector factory
func NewDetectorFactory() *DetectorFactory {
	return &DetectorFactory{}
}

// CreateDetector creates a detector for the specified provider
func (f *DetectorFactory) CreateDetector(provider Provider) (CLIDetector, error) {
	switch provider {
	case ProviderAWS:
		return NewAWSCLIDetector(), nil
	case ProviderAzure:
		return NewAzureCLIDetector(), nil
	case ProviderGCP:
		return NewGCPCLIDetector(), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// DetectAllCLIs detects all cloud provider CLIs
func DetectAllCLIs(ctx context.Context) map[Provider]*CLIStatus {
	factory := NewDetectorFactory()
	providers := []Provider{ProviderAWS, ProviderAzure, ProviderGCP}

	results := make(map[Provider]*CLIStatus)

	for _, provider := range providers {
		detector, err := factory.CreateDetector(provider)
		if err != nil {
			continue
		}

		status, err := detector.Detect()
		if err != nil {
			status = &CLIStatus{
				Installed:   false,
				IsSupported: false,
			}
		}

		results[provider] = status
	}

	return results
}

// ValidateCLIEnvironment validates the CLI environment for a provider
func ValidateCLIEnvironment(provider Provider) (*ValidationResult, error) {
	factory := NewDetectorFactory()
	detector, err := factory.CreateDetector(provider)
	if err != nil {
		return nil, err
	}

	status, err := detector.Detect()
	if err != nil {
		return &ValidationResult{
			Valid:  false,
			Errors: []string{fmt.Sprintf("Failed to detect CLI: %v", err)},
		}, nil
	}

	result := &ValidationResult{
		Valid:   true,
		Details: make(map[string]string),
	}

	if !status.Installed {
		result.Valid = false
		result.Errors = append(result.Errors, "CLI not installed")
		result.Details["install_instructions"] = detector.GetInstallInstructions()
		return result, nil
	}

	result.Details["version"] = status.Version
	result.Details["path"] = status.Path
	result.Details["config_path"] = status.ConfigPath

	if !status.IsSupported {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("CLI version %s is below minimum required version %s",
				status.Version, status.MinVersion))
	}

	// Check for authentication
	if err := checkAuthentication(provider, status.ConfigPath); err != nil {
		result.Warnings = append(result.Warnings, "No active authentication found")
		result.Details["auth_hint"] = getAuthenticationHint(provider)
	}

	return result, nil
}

// checkAuthentication checks if the CLI is authenticated
func checkAuthentication(provider Provider, configPath string) error {
	switch provider {
	case ProviderAWS:
		// Check for credentials file
		credFile := filepath.Join(configPath, "credentials")
		if _, err := os.Stat(credFile); os.IsNotExist(err) {
			return fmt.Errorf("no credentials file found")
		}
	case ProviderAzure:
		// Check if logged in
		cmd := exec.Command("az", "account", "show")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("not logged in")
		}
	case ProviderGCP:
		// Check for active configuration
		cmd := exec.Command("gcloud", "auth", "list")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("not authenticated")
		}
	}
	return nil
}

// getAuthenticationHint returns authentication hints for a provider
func getAuthenticationHint(provider Provider) string {
	switch provider {
	case ProviderAWS:
		return "Run 'aws configure' to set up credentials"
	case ProviderAzure:
		return "Run 'az login' to authenticate"
	case ProviderGCP:
		return "Run 'gcloud auth login' to authenticate"
	default:
		return "Please authenticate with your cloud provider"
	}
}

// GetPlatformCompatibility returns platform-specific compatibility info
func GetPlatformCompatibility(provider Provider) *PlatformCompatibility {
	home, _ := os.UserHomeDir()

	switch provider {
	case ProviderAWS:
		return &PlatformCompatibility{
			OS:         runtime.GOOS,
			Arch:       runtime.GOARCH,
			CLICommand: "aws",
			ConfigLocations: []string{
				filepath.Join(home, ".aws"),
				"/etc/aws",
			},
			EnvVars: []string{
				"AWS_PROFILE",
				"AWS_ACCESS_KEY_ID",
				"AWS_SECRET_ACCESS_KEY",
				"AWS_SESSION_TOKEN",
				"AWS_REGION",
			},
		}
	case ProviderAzure:
		return &PlatformCompatibility{
			OS:         runtime.GOOS,
			Arch:       runtime.GOARCH,
			CLICommand: "az",
			ConfigLocations: []string{
				filepath.Join(home, ".azure"),
			},
			EnvVars: []string{
				"AZURE_SUBSCRIPTION_ID",
				"AZURE_TENANT_ID",
				"AZURE_CLIENT_ID",
				"AZURE_CLIENT_SECRET",
			},
		}
	case ProviderGCP:
		return &PlatformCompatibility{
			OS:         runtime.GOOS,
			Arch:       runtime.GOARCH,
			CLICommand: "gcloud",
			ConfigLocations: []string{
				filepath.Join(home, ".config", "gcloud"),
			},
			EnvVars: []string{
				"GOOGLE_APPLICATION_CREDENTIALS",
				"GOOGLE_CLOUD_PROJECT",
				"GCLOUD_PROJECT",
			},
		}
	default:
		return nil
	}
}
