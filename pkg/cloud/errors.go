package cloud

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// CloudError represents a cloud provider-specific error
type CloudError struct {
	Provider    Provider  `json:"provider"`
	Operation   string    `json:"operation"`
	Code        string    `json:"code"`
	Message     string    `json:"message"`
	Details     string    `json:"details,omitempty"`
	Cause       error     `json:"cause,omitempty"`
	Retryable   bool      `json:"retryable"`
	StatusCode  int       `json:"status_code,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	UserMessage string    `json:"user_message,omitempty"`
}

// Error implements the error interface
func (ce *CloudError) Error() string {
	if ce.UserMessage != "" {
		return ce.UserMessage
	}

	if ce.Details != "" {
		return fmt.Sprintf("[%s] %s: %s - %s", ce.Provider, ce.Operation, ce.Message, ce.Details)
	}

	return fmt.Sprintf("[%s] %s: %s", ce.Provider, ce.Operation, ce.Message)
}

// Unwrap returns the underlying cause error
func (ce *CloudError) Unwrap() error {
	return ce.Cause
}

// Is checks if the error matches a target error
func (ce *CloudError) Is(target error) bool {
	if target == nil {
		return false
	}

	if targetCloudError, ok := target.(*CloudError); ok {
		return ce.Provider == targetCloudError.Provider &&
			ce.Code == targetCloudError.Code
	}

	return false
}

// ErrorBuilder helps build CloudError instances
type ErrorBuilder struct {
	provider  Provider
	operation string
}

// NewErrorBuilder creates a new error builder
func NewErrorBuilder(provider Provider, operation string) *ErrorBuilder {
	return &ErrorBuilder{
		provider:  provider,
		operation: operation,
	}
}

// Build creates a CloudError with the given parameters
func (eb *ErrorBuilder) Build(code, message string) *CloudError {
	return &CloudError{
		Provider:  eb.provider,
		Operation: eb.operation,
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
		Retryable: eb.isRetryable(code),
	}
}

// BuildWithCause creates a CloudError with an underlying cause
func (eb *ErrorBuilder) BuildWithCause(code, message string, cause error) *CloudError {
	err := eb.Build(code, message)
	err.Cause = cause
	err.Retryable = eb.isRetryableWithCause(code, cause)
	return err
}

// BuildWithDetails creates a CloudError with additional details
func (eb *ErrorBuilder) BuildWithDetails(code, message, details string) *CloudError {
	err := eb.Build(code, message)
	err.Details = details
	return err
}

// BuildUserFriendly creates a CloudError with a user-friendly message
func (eb *ErrorBuilder) BuildUserFriendly(code, message, userMessage string) *CloudError {
	err := eb.Build(code, message)
	err.UserMessage = userMessage
	return err
}

// BuildHTTP creates a CloudError from an HTTP response
func (eb *ErrorBuilder) BuildHTTP(statusCode int, message string) *CloudError {
	err := eb.Build(fmt.Sprintf("HTTP_%d", statusCode), message)
	err.StatusCode = statusCode
	err.Retryable = eb.isHTTPRetryable(statusCode)

	// Add user-friendly messages for common HTTP errors
	switch statusCode {
	case 401:
		err.UserMessage = "Authentication failed. Please check your credentials and try again."
	case 403:
		err.UserMessage = "Access denied. You don't have permission to perform this operation."
	case 404:
		err.UserMessage = "Resource not found. Please check the resource name and try again."
	case 429:
		err.UserMessage = "Rate limit exceeded. Please wait a moment and try again."
	case 500:
		err.UserMessage = "Server error occurred. Please try again later."
	case 503:
		err.UserMessage = "Service unavailable. Please try again later."
	}

	return err
}

// isRetryable determines if an error code is retryable
func (eb *ErrorBuilder) isRetryable(code string) bool {
	retryableCodes := map[string]bool{
		"RATE_LIMITED":        true,
		"THROTTLED":           true,
		"TIMEOUT":             true,
		"NETWORK_ERROR":       true,
		"SERVICE_UNAVAILABLE": true,
		"TEMPORARY_FAILURE":   true,
		"CONNECTION_ERROR":    true,
		"DNS_ERROR":           true,
		"HTTP_500":            true,
		"HTTP_502":            true,
		"HTTP_503":            true,
		"HTTP_504":            true,
		"HTTP_429":            true,
		"TOKEN_EXPIRED":       true,
		"SESSION_EXPIRED":     true,
	}

	return retryableCodes[code]
}

// isRetryableWithCause determines if an error is retryable based on the underlying cause
func (eb *ErrorBuilder) isRetryableWithCause(code string, cause error) bool {
	if eb.isRetryable(code) {
		return true
	}

	// Check underlying error types
	if cause != nil {
		// Network errors are generally retryable
		if _, ok := cause.(net.Error); ok {
			return true
		}

		// Context deadline exceeded is retryable
		if errors.Is(cause, context.DeadlineExceeded) {
			return true
		}

		// DNS errors are retryable
		if _, ok := cause.(*net.DNSError); ok {
			return true
		}

		// Connection errors are retryable
		if _, ok := cause.(*net.OpError); ok {
			return true
		}
	}

	return false
}

// isHTTPRetryable determines if an HTTP status code is retryable
func (eb *ErrorBuilder) isHTTPRetryable(statusCode int) bool {
	switch statusCode {
	case 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}

// Common error codes
const (
	// Authentication errors
	ErrCodeAuthenticationFailed = "AUTHENTICATION_FAILED"
	ErrCodeInvalidCredentials   = "INVALID_CREDENTIALS"
	ErrCodeTokenExpired         = "TOKEN_EXPIRED"
	ErrCodeSessionExpired       = "SESSION_EXPIRED"
	ErrCodePermissionDenied     = "PERMISSION_DENIED"
	ErrCodeNotAuthenticated     = "NOT_AUTHENTICATED"

	// Configuration errors
	ErrCodeInvalidConfiguration  = "INVALID_CONFIGURATION"
	ErrCodeMissingConfiguration  = "MISSING_CONFIGURATION"
	ErrCodeConfigurationNotFound = "CONFIGURATION_NOT_FOUND"

	// CLI errors
	ErrCodeCLINotInstalled     = "CLI_NOT_INSTALLED"
	ErrCodeCLIVersionMismatch  = "CLI_VERSION_MISMATCH"
	ErrCodeCLIExecutionFailed  = "CLI_EXECUTION_FAILED"
	ErrCodeCLINotAuthenticated = "CLI_NOT_AUTHENTICATED"

	// Resource errors
	ErrCodeResourceNotFound    = "RESOURCE_NOT_FOUND"
	ErrCodeResourceExists      = "RESOURCE_EXISTS"
	ErrCodeResourceInUse       = "RESOURCE_IN_USE"
	ErrCodeResourceUnavailable = "RESOURCE_UNAVAILABLE"

	// Network errors
	ErrCodeNetworkError      = "NETWORK_ERROR"
	ErrCodeConnectionTimeout = "CONNECTION_TIMEOUT"
	ErrCodeDNSError          = "DNS_ERROR"
	ErrCodeConnectionRefused = "CONNECTION_REFUSED"

	// Rate limiting errors
	ErrCodeRateLimited = "RATE_LIMITED"
	ErrCodeThrottled   = "THROTTLED"

	// Service errors
	ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
	ErrCodeInternalError      = "INTERNAL_ERROR"
	ErrCodeTemporaryFailure   = "TEMPORARY_FAILURE"

	// Validation errors
	ErrCodeValidationFailed = "VALIDATION_FAILED"
	ErrCodeInvalidInput     = "INVALID_INPUT"
	ErrCodeMissingParameter = "MISSING_PARAMETER"
)

// Predefined errors
var (
	ErrCLINotInstalled   = errors.New("cloud CLI not installed")
	ErrCLINotFound       = errors.New("cloud CLI not found in PATH")
	ErrNotAuthenticated  = errors.New("not authenticated with cloud provider")
	ErrInvalidProvider   = errors.New("invalid or unsupported cloud provider")
	ErrResourceNotFound  = errors.New("cloud resource not found")
	ErrOperationTimeout  = errors.New("cloud operation timed out")
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)

// ErrorClassifier helps classify errors for appropriate handling
type ErrorClassifier struct{}

// NewErrorClassifier creates a new error classifier
func NewErrorClassifier() *ErrorClassifier {
	return &ErrorClassifier{}
}

// Classify classifies an error and returns appropriate handling strategy
func (ec *ErrorClassifier) Classify(err error) ErrorClassification {
	if err == nil {
		return ErrorClassification{
			Type:      ErrorTypeNone,
			Severity:  SeverityInfo,
			Retryable: false,
		}
	}

	// Check if it's a CloudError
	if cloudErr, ok := err.(*CloudError); ok {
		return ec.classifyCloudError(cloudErr)
	}

	// Classify standard errors
	return ec.classifyStandardError(err)
}

// classifyCloudError classifies a CloudError
func (ec *ErrorClassifier) classifyCloudError(err *CloudError) ErrorClassification {
	classification := ErrorClassification{
		Type:      ec.getErrorType(err.Code),
		Severity:  ec.getSeverity(err.Code),
		Retryable: err.Retryable,
		Code:      err.Code,
		Provider:  err.Provider,
	}

	// Add retry strategy if retryable
	if err.Retryable {
		classification.RetryStrategy = ec.getRetryStrategy(err.Code)
	}

	return classification
}

// classifyStandardError classifies a standard error
func (ec *ErrorClassifier) classifyStandardError(err error) ErrorClassification {
	classification := ErrorClassification{
		Type:     ErrorTypeUnknown,
		Severity: SeverityError,
	}

	// Check for network errors
	if netErr, ok := err.(net.Error); ok {
		classification.Type = ErrorTypeNetwork
		classification.Retryable = true
		classification.RetryStrategy = RetryStrategyExponential

		if netErr.Timeout() {
			classification.Severity = SeverityWarning
		}
		return classification
	}

	// Check for context errors
	if errors.Is(err, context.DeadlineExceeded) {
		classification.Type = ErrorTypeTimeout
		classification.Severity = SeverityWarning
		classification.Retryable = true
		classification.RetryStrategy = RetryStrategyLinear
		return classification
	}

	if errors.Is(err, context.Canceled) {
		classification.Type = ErrorTypeCanceled
		classification.Severity = SeverityInfo
		classification.Retryable = false
		return classification
	}

	// Check error message for common patterns
	errMsg := strings.ToLower(err.Error())

	if strings.Contains(errMsg, "not found") {
		classification.Type = ErrorTypeNotFound
		classification.Severity = SeverityWarning
		classification.Retryable = false
	} else if strings.Contains(errMsg, "permission denied") || strings.Contains(errMsg, "unauthorized") {
		classification.Type = ErrorTypeAuthentication
		classification.Severity = SeverityError
		classification.Retryable = false
	} else if strings.Contains(errMsg, "rate limit") || strings.Contains(errMsg, "throttle") {
		classification.Type = ErrorTypeRateLimit
		classification.Severity = SeverityWarning
		classification.Retryable = true
		classification.RetryStrategy = RetryStrategyExponential
	}

	return classification
}

// getErrorType returns the error type for a code
func (ec *ErrorClassifier) getErrorType(code string) ErrorType {
	switch code {
	case ErrCodeAuthenticationFailed, ErrCodeInvalidCredentials, ErrCodePermissionDenied:
		return ErrorTypeAuthentication
	case ErrCodeInvalidConfiguration, ErrCodeMissingConfiguration, ErrCodeConfigurationNotFound:
		return ErrorTypeConfiguration
	case ErrCodeCLINotInstalled, ErrCodeCLIVersionMismatch, ErrCodeCLIExecutionFailed:
		return ErrorTypeCLI
	case ErrCodeResourceNotFound:
		return ErrorTypeNotFound
	case ErrCodeNetworkError, ErrCodeConnectionTimeout, ErrCodeDNSError, ErrCodeConnectionRefused:
		return ErrorTypeNetwork
	case ErrCodeRateLimited, ErrCodeThrottled:
		return ErrorTypeRateLimit
	case ErrCodeServiceUnavailable, ErrCodeInternalError, ErrCodeTemporaryFailure:
		return ErrorTypeService
	case ErrCodeValidationFailed, ErrCodeInvalidInput, ErrCodeMissingParameter:
		return ErrorTypeValidation
	default:
		if strings.HasPrefix(code, "HTTP_") {
			return ErrorTypeHTTP
		}
		return ErrorTypeUnknown
	}
}

// getSeverity returns the severity for a code
func (ec *ErrorClassifier) getSeverity(code string) ErrorSeverity {
	switch code {
	case ErrCodeTokenExpired, ErrCodeSessionExpired, ErrCodeResourceNotFound:
		return SeverityWarning
	case ErrCodeRateLimited, ErrCodeThrottled, ErrCodeConnectionTimeout:
		return SeverityWarning
	case ErrCodeAuthenticationFailed, ErrCodeInvalidCredentials, ErrCodePermissionDenied:
		return SeverityError
	case ErrCodeCLINotInstalled, ErrCodeCLIVersionMismatch:
		return SeverityError
	case ErrCodeInvalidConfiguration, ErrCodeMissingConfiguration:
		return SeverityError
	case ErrCodeServiceUnavailable, ErrCodeInternalError:
		return SeverityCritical
	default:
		if strings.HasPrefix(code, "HTTP_5") {
			return SeverityCritical
		}
		if strings.HasPrefix(code, "HTTP_4") {
			return SeverityError
		}
		return SeverityWarning
	}
}

// getRetryStrategy returns the retry strategy for a code
func (ec *ErrorClassifier) getRetryStrategy(code string) RetryStrategy {
	switch code {
	case ErrCodeRateLimited, ErrCodeThrottled:
		return RetryStrategyExponential
	case ErrCodeNetworkError, ErrCodeConnectionTimeout, ErrCodeDNSError:
		return RetryStrategyExponential
	case ErrCodeServiceUnavailable, ErrCodeTemporaryFailure:
		return RetryStrategyLinear
	case ErrCodeTokenExpired, ErrCodeSessionExpired:
		return RetryStrategyImmediate
	default:
		return RetryStrategyExponential
	}
}

// ErrorClassification represents the classification of an error
type ErrorClassification struct {
	Type          ErrorType     `json:"type"`
	Severity      ErrorSeverity `json:"severity"`
	Retryable     bool          `json:"retryable"`
	RetryStrategy RetryStrategy `json:"retry_strategy,omitempty"`
	Code          string        `json:"code,omitempty"`
	Provider      Provider      `json:"provider,omitempty"`
}

// ErrorType represents the type of error
type ErrorType string

const (
	ErrorTypeNone           ErrorType = "none"
	ErrorTypeAuthentication ErrorType = "authentication"
	ErrorTypeConfiguration  ErrorType = "configuration"
	ErrorTypeCLI            ErrorType = "cli"
	ErrorTypeNotFound       ErrorType = "not_found"
	ErrorTypeNetwork        ErrorType = "network"
	ErrorTypeRateLimit      ErrorType = "rate_limit"
	ErrorTypeService        ErrorType = "service"
	ErrorTypeValidation     ErrorType = "validation"
	ErrorTypeHTTP           ErrorType = "http"
	ErrorTypeTimeout        ErrorType = "timeout"
	ErrorTypeCanceled       ErrorType = "canceled"
	ErrorTypeUnknown        ErrorType = "unknown"
)

// ErrorSeverity represents the severity of an error
type ErrorSeverity string

const (
	SeverityInfo     ErrorSeverity = "info"
	SeverityWarning  ErrorSeverity = "warning"
	SeverityError    ErrorSeverity = "error"
	SeverityCritical ErrorSeverity = "critical"
)

// RetryStrategy represents the retry strategy for an error
type RetryStrategy string

const (
	RetryStrategyNone        RetryStrategy = "none"
	RetryStrategyImmediate   RetryStrategy = "immediate"
	RetryStrategyLinear      RetryStrategy = "linear"
	RetryStrategyExponential RetryStrategy = "exponential"
)

// UserMessageGenerator generates user-friendly error messages
type UserMessageGenerator struct{}

// NewUserMessageGenerator creates a new user message generator
func NewUserMessageGenerator() *UserMessageGenerator {
	return &UserMessageGenerator{}
}

// GenerateMessage generates a user-friendly error message
func (umg *UserMessageGenerator) GenerateMessage(err error) string {
	if err == nil {
		return ""
	}

	// Check if it's already a CloudError with a user message
	if cloudErr, ok := err.(*CloudError); ok && cloudErr.UserMessage != "" {
		return cloudErr.UserMessage
	}

	classifier := NewErrorClassifier()
	classification := classifier.Classify(err)

	return umg.generateByClassification(classification, err)
}

// generateByClassification generates a message based on error classification
func (umg *UserMessageGenerator) generateByClassification(classification ErrorClassification, err error) string {
	switch classification.Type {
	case ErrorTypeAuthentication:
		return umg.generateAuthMessage(classification, err)
	case ErrorTypeConfiguration:
		return umg.generateConfigMessage(classification, err)
	case ErrorTypeCLI:
		return umg.generateCLIMessage(classification, err)
	case ErrorTypeNotFound:
		return umg.generateNotFoundMessage(classification, err)
	case ErrorTypeNetwork:
		return umg.generateNetworkMessage(classification, err)
	case ErrorTypeRateLimit:
		return umg.generateRateLimitMessage(classification, err)
	case ErrorTypeService:
		return umg.generateServiceMessage(classification, err)
	case ErrorTypeValidation:
		return umg.generateValidationMessage(classification, err)
	default:
		return umg.generateGenericMessage(classification, err)
	}
}

// generateAuthMessage generates authentication error messages
func (umg *UserMessageGenerator) generateAuthMessage(classification ErrorClassification, err error) string {
	switch classification.Provider {
	case ProviderAWS:
		return "Authentication with AWS failed. Please run 'aws configure' or check your AWS credentials."
	case ProviderAzure:
		return "Authentication with Azure failed. Please run 'az login' or check your Azure credentials."
	case ProviderGCP:
		return "Authentication with Google Cloud failed. Please run 'gcloud auth login' or check your GCP credentials."
	default:
		return "Authentication failed. Please check your cloud provider credentials and try again."
	}
}

// generateConfigMessage generates configuration error messages
func (umg *UserMessageGenerator) generateConfigMessage(classification ErrorClassification, err error) string {
	return "Configuration error detected. Please check your cloud provider configuration and try again."
}

// generateCLIMessage generates CLI error messages
func (umg *UserMessageGenerator) generateCLIMessage(classification ErrorClassification, err error) string {
	switch classification.Provider {
	case ProviderAWS:
		return "AWS CLI is not properly installed or configured. Please install the AWS CLI and run 'aws configure'."
	case ProviderAzure:
		return "Azure CLI is not properly installed or configured. Please install the Azure CLI and run 'az login'."
	case ProviderGCP:
		return "Google Cloud CLI is not properly installed or configured. Please install gcloud and run 'gcloud auth login'."
	default:
		return "Cloud CLI is not properly installed or configured. Please install the appropriate CLI tool."
	}
}

// generateNotFoundMessage generates not found error messages
func (umg *UserMessageGenerator) generateNotFoundMessage(classification ErrorClassification, err error) string {
	return "The requested resource was not found. Please check the resource name and try again."
}

// generateNetworkMessage generates network error messages
func (umg *UserMessageGenerator) generateNetworkMessage(classification ErrorClassification, err error) string {
	if classification.Retryable {
		return "Network connection issue detected. The operation will be retried automatically."
	}
	return "Network connection failed. Please check your internet connection and try again."
}

// generateRateLimitMessage generates rate limit error messages
func (umg *UserMessageGenerator) generateRateLimitMessage(classification ErrorClassification, err error) string {
	return "Rate limit exceeded. Please wait a moment before trying again."
}

// generateServiceMessage generates service error messages
func (umg *UserMessageGenerator) generateServiceMessage(classification ErrorClassification, err error) string {
	if classification.Retryable {
		return "Cloud service is temporarily unavailable. The operation will be retried automatically."
	}
	return "Cloud service error occurred. Please try again later."
}

// generateValidationMessage generates validation error messages
func (umg *UserMessageGenerator) generateValidationMessage(classification ErrorClassification, err error) string {
	return "Input validation failed. Please check your parameters and try again."
}

// generateGenericMessage generates generic error messages
func (umg *UserMessageGenerator) generateGenericMessage(classification ErrorClassification, err error) string {
	if classification.Retryable {
		return "An error occurred, but the operation will be retried automatically."
	}
	return "An unexpected error occurred. Please try again or contact support if the problem persists."
}

// GetSuggestions returns suggestions for resolving an error
func (umg *UserMessageGenerator) GetSuggestions(err error) []string {
	if err == nil {
		return nil
	}

	classifier := NewErrorClassifier()
	classification := classifier.Classify(err)

	switch classification.Type {
	case ErrorTypeAuthentication:
		return umg.getAuthSuggestions(classification)
	case ErrorTypeConfiguration:
		return umg.getConfigSuggestions(classification)
	case ErrorTypeCLI:
		return umg.getCLISuggestions(classification)
	case ErrorTypeNetwork:
		return umg.getNetworkSuggestions(classification)
	case ErrorTypeRateLimit:
		return umg.getRateLimitSuggestions(classification)
	default:
		return umg.getGenericSuggestions(classification)
	}
}

// getAuthSuggestions returns authentication error suggestions
func (umg *UserMessageGenerator) getAuthSuggestions(classification ErrorClassification) []string {
	switch classification.Provider {
	case ProviderAWS:
		return []string{
			"Run 'aws configure' to set up your credentials",
			"Check if your AWS access keys are valid",
			"Verify your AWS region is correct",
			"Try using 'aws sts get-caller-identity' to test authentication",
		}
	case ProviderAzure:
		return []string{
			"Run 'az login' to authenticate",
			"Check if your Azure subscription is active",
			"Verify you have the required permissions",
			"Try using 'az account show' to verify authentication",
		}
	case ProviderGCP:
		return []string{
			"Run 'gcloud auth login' to authenticate",
			"Check if your GCP project is set correctly",
			"Verify your service account has the required permissions",
			"Try using 'gcloud auth list' to check authentication status",
		}
	default:
		return []string{
			"Check your cloud provider credentials",
			"Verify you have the required permissions",
			"Try re-authenticating with your cloud provider",
		}
	}
}

// getConfigSuggestions returns configuration error suggestions
func (umg *UserMessageGenerator) getConfigSuggestions(classification ErrorClassification) []string {
	return []string{
		"Check your configuration file syntax",
		"Verify all required fields are present",
		"Review the configuration documentation",
		"Try using the default configuration",
	}
}

// getCLISuggestions returns CLI error suggestions
func (umg *UserMessageGenerator) getCLISuggestions(classification ErrorClassification) []string {
	switch classification.Provider {
	case ProviderAWS:
		return []string{
			"Install AWS CLI: https://aws.amazon.com/cli/",
			"Update AWS CLI to the latest version",
			"Add AWS CLI to your PATH",
		}
	case ProviderAzure:
		return []string{
			"Install Azure CLI: https://docs.microsoft.com/en-us/cli/azure/install-azure-cli",
			"Update Azure CLI to the latest version",
			"Add Azure CLI to your PATH",
		}
	case ProviderGCP:
		return []string{
			"Install Google Cloud CLI: https://cloud.google.com/sdk/docs/install",
			"Update gcloud to the latest version",
			"Add gcloud to your PATH",
		}
	default:
		return []string{
			"Install the appropriate cloud CLI tool",
			"Update CLI to the latest version",
			"Add CLI to your PATH",
		}
	}
}

// getNetworkSuggestions returns network error suggestions
func (umg *UserMessageGenerator) getNetworkSuggestions(classification ErrorClassification) []string {
	return []string{
		"Check your internet connection",
		"Verify firewall settings",
		"Try again in a few moments",
		"Check if the cloud service is experiencing issues",
	}
}

// getRateLimitSuggestions returns rate limit error suggestions
func (umg *UserMessageGenerator) getRateLimitSuggestions(classification ErrorClassification) []string {
	return []string{
		"Wait a few moments before retrying",
		"Reduce the frequency of requests",
		"Consider implementing exponential backoff",
		"Check your rate limit quotas",
	}
}

// getGenericSuggestions returns generic error suggestions
func (umg *UserMessageGenerator) getGenericSuggestions(classification ErrorClassification) []string {
	suggestions := []string{
		"Try the operation again",
		"Check the logs for more details",
	}

	if classification.Retryable {
		suggestions = append(suggestions, "The operation will be retried automatically")
	}

	return suggestions
}
