#!/bin/bash

# SonarQube Local Scan Script for GoFiber APM Solution
# This script performs local SonarQube scanning with pre-commit integration

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SONAR_PROJECT_KEY="apm-solution"
SONAR_PROJECT_NAME="APM Solution - GoFiber APM Stack"
SONAR_HOST_URL=${SONAR_HOST_URL:-"http://localhost:9000"}
SONAR_LOGIN=${SONAR_LOGIN:-"admin"}
SONAR_PASSWORD=${SONAR_PASSWORD:-"admin"}
COVERAGE_FILE="coverage.out"
GOLANGCI_LINT_REPORT="golangci-lint-report.xml"
GOLANGCI_LINT_CHECKSTYLE="golangci-lint-checkstyle.xml"
GOVET_REPORT="govet-report.out"
GOLINT_REPORT="golint-report.out"
TEST_REPORT="test-report.json"
GOSEC_REPORT="gosec-report.json"

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to install missing tools
install_tools() {
    print_status "Checking and installing required tools..."
    
    # Check Go installation
    if ! command_exists go; then
        print_error "Go is not installed. Please install Go first."
        exit 1
    fi
    
    # Install golangci-lint if not present
    if ! command_exists golangci-lint; then
        print_status "Installing golangci-lint..."
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.54.2
        export PATH=$PATH:$(go env GOPATH)/bin
    fi
    
    # Install golint if not present
    if ! command_exists golint; then
        print_status "Installing golint..."
        go install golang.org/x/lint/golint@latest
    fi
    
    # Install gosec if not present
    if ! command_exists gosec; then
        print_status "Installing gosec..."
        go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
    fi
    
    # Install govulncheck if not present
    if ! command_exists govulncheck; then
        print_status "Installing govulncheck..."
        go install golang.org/x/vuln/cmd/govulncheck@latest
    fi
    
    # Check SonarQube scanner
    if ! command_exists sonar-scanner; then
        print_warning "SonarQube scanner not found. Please install it from https://docs.sonarqube.org/latest/analysis/scan/sonarscanner/"
        print_warning "For macOS: brew install sonar-scanner"
        print_warning "For Linux: Download from SonarQube website"
        exit 1
    fi
}

# Function to clean up previous reports
cleanup_reports() {
    print_status "Cleaning up previous reports..."
    rm -f $COVERAGE_FILE $GOLANGCI_LINT_REPORT $GOLANGCI_LINT_CHECKSTYLE $GOVET_REPORT $GOLINT_REPORT $TEST_REPORT $GOSEC_REPORT
    rm -f coverage.html
}

# Function to run tests and generate coverage
run_tests() {
    print_status "Running tests and generating coverage report..."
    
    if ! go test -v -race -coverprofile=$COVERAGE_FILE -covermode=atomic ./...; then
        print_warning "Some tests failed, but continuing with analysis..."
    fi
    
    if [ -f $COVERAGE_FILE ]; then
        print_success "Coverage report generated: $COVERAGE_FILE"
        go tool cover -html=$COVERAGE_FILE -o coverage.html
        print_status "HTML coverage report generated: coverage.html"
    else
        print_warning "No coverage report generated"
    fi
}

# Function to run golangci-lint
run_golangci_lint() {
    print_status "Running golangci-lint..."
    
    golangci-lint run --out-format=junit-xml > $GOLANGCI_LINT_REPORT 2>/dev/null || true
    golangci-lint run --out-format=checkstyle > $GOLANGCI_LINT_CHECKSTYLE 2>/dev/null || true
    
    if [ -f $GOLANGCI_LINT_REPORT ]; then
        print_success "golangci-lint report generated: $GOLANGCI_LINT_REPORT"
    fi
}

# Function to run go vet
run_govet() {
    print_status "Running go vet..."
    
    go vet ./... 2>&1 | tee $GOVET_REPORT || true
    
    if [ -f $GOVET_REPORT ]; then
        print_success "go vet report generated: $GOVET_REPORT"
    fi
}

# Function to run golint
run_golint() {
    print_status "Running golint..."
    
    golint ./... > $GOLINT_REPORT 2>/dev/null || true
    
    if [ -f $GOLINT_REPORT ]; then
        print_success "golint report generated: $GOLINT_REPORT"
    fi
}

# Function to run gosec
run_gosec() {
    print_status "Running gosec security scanner..."
    
    gosec -fmt sonarqube -out $GOSEC_REPORT ./... 2>/dev/null || true
    
    if [ -f $GOSEC_REPORT ]; then
        print_success "gosec security report generated: $GOSEC_REPORT"
    fi
}

# Function to run govulncheck
run_govulncheck() {
    print_status "Running govulncheck for vulnerability scanning..."
    
    govulncheck ./... || print_warning "govulncheck found some issues or completed with warnings"
}

# Function to generate test report
generate_test_report() {
    print_status "Generating test report..."
    
    go test -v -json ./... > $TEST_REPORT 2>/dev/null || true
    
    if [ -f $TEST_REPORT ]; then
        print_success "Test report generated: $TEST_REPORT"
    fi
}

# Function to run SonarQube analysis
run_sonar_analysis() {
    print_status "Running SonarQube analysis..."
    
    # Get current git branch
    CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "main")
    
    # Build sonar-scanner command
    SONAR_ARGS="-Dsonar.projectKey=$SONAR_PROJECT_KEY"
    SONAR_ARGS="$SONAR_ARGS -Dsonar.projectName=\"$SONAR_PROJECT_NAME\""
    SONAR_ARGS="$SONAR_ARGS -Dsonar.host.url=$SONAR_HOST_URL"
    SONAR_ARGS="$SONAR_ARGS -Dsonar.login=$SONAR_LOGIN"
    
    # Add coverage if available
    if [ -f $COVERAGE_FILE ]; then
        SONAR_ARGS="$SONAR_ARGS -Dsonar.go.coverage.reportPaths=$COVERAGE_FILE"
    fi
    
    # Add golangci-lint reports if available
    if [ -f $GOLANGCI_LINT_REPORT ]; then
        SONAR_ARGS="$SONAR_ARGS -Dsonar.go.golangci-lint.reportPaths=$GOLANGCI_LINT_REPORT"
    fi
    
    # Add other reports if available
    if [ -f $GOVET_REPORT ]; then
        SONAR_ARGS="$SONAR_ARGS -Dsonar.go.govet.reportPaths=$GOVET_REPORT"
    fi
    
    if [ -f $GOLINT_REPORT ]; then
        SONAR_ARGS="$SONAR_ARGS -Dsonar.go.golint.reportPaths=$GOLINT_REPORT"
    fi
    
    if [ -f $TEST_REPORT ]; then
        SONAR_ARGS="$SONAR_ARGS -Dsonar.go.test.reportPaths=$TEST_REPORT"
    fi
    
    # Add branch information
    SONAR_ARGS="$SONAR_ARGS -Dsonar.branch.name=$CURRENT_BRANCH"
    
    # Run the scanner
    print_status "Executing: sonar-scanner $SONAR_ARGS"
    
    if eval "sonar-scanner $SONAR_ARGS"; then
        print_success "SonarQube analysis completed successfully!"
        
        # Display results if available
        if [ -f .sonarqube/report-task.txt ]; then
            print_status "SonarQube analysis results:"
            cat .sonarqube/report-task.txt
        fi
    else
        print_error "SonarQube analysis failed!"
        exit 1
    fi
}

# Function to format and display results
display_results() {
    print_status "Analysis Results Summary:"
    echo "========================"
    
    if [ -f $COVERAGE_FILE ]; then
        COVERAGE_PERCENT=$(go tool cover -func=$COVERAGE_FILE | grep total | awk '{print $3}')
        echo "Coverage: $COVERAGE_PERCENT"
    fi
    
    if [ -f $GOLANGCI_LINT_REPORT ]; then
        GOLANGCI_ISSUES=$(grep -c "testcase" $GOLANGCI_LINT_REPORT 2>/dev/null || echo "0")
        echo "golangci-lint issues: $GOLANGCI_ISSUES"
    fi
    
    if [ -f $GOVET_REPORT ]; then
        GOVET_ISSUES=$(wc -l < $GOVET_REPORT 2>/dev/null || echo "0")
        echo "go vet issues: $GOVET_ISSUES"
    fi
    
    if [ -f $GOLINT_REPORT ]; then
        GOLINT_ISSUES=$(wc -l < $GOLINT_REPORT 2>/dev/null || echo "0")
        echo "golint issues: $GOLINT_ISSUES"
    fi
    
    echo "========================"
    print_success "Local SonarQube scan completed!"
}

# Function to setup pre-commit hook
setup_precommit() {
    print_status "Setting up pre-commit hook..."
    
    HOOK_FILE=".git/hooks/pre-commit"
    
    cat > $HOOK_FILE << 'EOF'
#!/bin/bash
# Pre-commit hook for SonarQube analysis

echo "Running pre-commit SonarQube analysis..."

# Change to repository root
cd "$(git rev-parse --show-toplevel)"

# Run quick analysis (without full SonarQube scan)
if [ -f scripts/sonar-scan.sh ]; then
    # Run only linting and tests for pre-commit
    export QUICK_SCAN=true
    ./scripts/sonar-scan.sh --quick
else
    echo "SonarQube scan script not found, skipping analysis"
fi
EOF
    
    chmod +x $HOOK_FILE
    print_success "Pre-commit hook installed successfully!"
}

# Function to run quick scan (for pre-commit)
run_quick_scan() {
    print_status "Running quick scan (pre-commit mode)..."
    
    install_tools
    cleanup_reports
    
    # Run only essential checks
    run_golangci_lint
    run_govet
    
    # Quick test run
    print_status "Running quick tests..."
    if ! go test -short ./...; then
        print_error "Tests failed! Commit aborted."
        exit 1
    fi
    
    print_success "Quick scan completed successfully!"
}

# Main execution function
main() {
    print_status "Starting SonarQube local scan for GoFiber APM Solution..."
    
    # Parse command line arguments
    case "${1:-}" in
        --quick)
            run_quick_scan
            exit 0
            ;;
        --setup-precommit)
            setup_precommit
            exit 0
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo "Options:"
            echo "  --quick            Run quick scan (for pre-commit)"
            echo "  --setup-precommit  Install pre-commit hook"
            echo "  --help, -h         Show this help message"
            exit 0
            ;;
    esac
    
    # Full scan
    install_tools
    cleanup_reports
    run_tests
    run_golangci_lint
    run_govet
    run_golint
    run_gosec
    run_govulncheck
    generate_test_report
    run_sonar_analysis
    display_results
    
    print_success "SonarQube local scan completed successfully!"
}

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    print_error "This script must be run from within a git repository"
    exit 1
fi

# Run main function
main "$@"