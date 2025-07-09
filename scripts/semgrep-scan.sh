#!/bin/bash

# Semgrep Security Scanning Script
# This script runs Semgrep security analysis on the APM codebase

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
SEMGREP_CONFIG="./.semgrep.yml"
OUTPUT_DIR="./security-reports"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Print usage
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo "Options:"
    echo "  -c, --config <file>    Use custom Semgrep config file (default: .semgrep.yml)"
    echo "  -f, --format <format>  Output format: text, json, sarif, junit (default: text)"
    echo "  -s, --severity <level> Minimum severity level: INFO, WARNING, ERROR (default: all)"
    echo "  -o, --output <file>    Output file path (default: stdout)"
    echo "  -q, --quiet           Suppress non-error output"
    echo "  -h, --help            Show this help message"
    exit 1
}

# Parse command line arguments
FORMAT="text"
SEVERITY=""
OUTPUT=""
QUIET=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -c|--config)
            SEMGREP_CONFIG="$2"
            shift 2
            ;;
        -f|--format)
            FORMAT="$2"
            shift 2
            ;;
        -s|--severity)
            SEVERITY="--severity=$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT="$2"
            shift 2
            ;;
        -q|--quiet)
            QUIET=true
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo "Unknown option: $1"
            usage
            ;;
    esac
done

# Check if Semgrep is installed
check_semgrep() {
    if ! command -v semgrep &> /dev/null; then
        echo -e "${RED}Error: Semgrep is not installed${NC}"
        echo "Install Semgrep using one of the following methods:"
        echo "  pip install semgrep"
        echo "  brew install semgrep"
        echo "  docker pull returntocorp/semgrep"
        exit 1
    fi
}

# Create output directory if needed
setup_output_dir() {
    if [[ -n "$OUTPUT" ]] && [[ "$OUTPUT" == *"/"* ]]; then
        mkdir -p "$(dirname "$OUTPUT")"
    elif [[ -z "$OUTPUT" ]] && [[ "$FORMAT" != "text" ]]; then
        mkdir -p "$OUTPUT_DIR"
    fi
}

# Run Semgrep scan
run_scan() {
    local cmd="semgrep --config=$SEMGREP_CONFIG"
    
    # Add severity filter if specified
    if [[ -n "$SEVERITY" ]]; then
        cmd="$cmd $SEVERITY"
    fi
    
    # Add format option
    case $FORMAT in
        json)
            cmd="$cmd --json"
            ;;
        sarif)
            cmd="$cmd --sarif"
            ;;
        junit)
            cmd="$cmd --junit-xml"
            ;;
    esac
    
    # Add output file if specified
    if [[ -n "$OUTPUT" ]]; then
        cmd="$cmd --output=$OUTPUT"
    elif [[ "$FORMAT" != "text" ]]; then
        OUTPUT="$OUTPUT_DIR/semgrep-report-$TIMESTAMP.$FORMAT"
        cmd="$cmd --output=$OUTPUT"
    fi
    
    # Add quiet flag if specified
    if [[ "$QUIET" == true ]] && [[ "$FORMAT" == "text" ]]; then
        cmd="$cmd --quiet"
    fi
    
    # Add target directory
    cmd="$cmd ."
    
    if [[ "$QUIET" != true ]]; then
        echo -e "${GREEN}Running Semgrep security scan...${NC}"
        echo "Command: $cmd"
    fi
    
    # Run the scan
    eval $cmd
    local exit_code=$?
    
    return $exit_code
}

# Generate summary report
generate_summary() {
    if [[ "$FORMAT" == "json" ]] && [[ -n "$OUTPUT" ]] && [[ -f "$OUTPUT" ]]; then
        echo -e "\n${YELLOW}Scan Summary:${NC}"
        
        # Parse JSON output for summary
        if command -v jq &> /dev/null; then
            local total=$(jq '.results | length' "$OUTPUT")
            local errors=$(jq '.results | map(select(.extra.severity == "ERROR")) | length' "$OUTPUT")
            local warnings=$(jq '.results | map(select(.extra.severity == "WARNING")) | length' "$OUTPUT")
            local info=$(jq '.results | map(select(.extra.severity == "INFO")) | length' "$OUTPUT")
            
            echo "Total findings: $total"
            echo "  - Errors: $errors"
            echo "  - Warnings: $warnings"
            echo "  - Info: $info"
            
            # Show top rules
            echo -e "\nTop violated rules:"
            jq -r '.results | group_by(.check_id) | map({rule: .[0].check_id, count: length}) | sort_by(.count) | reverse | .[:5] | .[] | "  - \(.rule): \(.count) findings"' "$OUTPUT"
        else
            echo "Install jq for detailed summary: brew install jq"
        fi
    fi
}

# Main execution
main() {
    check_semgrep
    setup_output_dir
    
    # Run the scan
    run_scan
    local exit_code=$?
    
    # Generate summary if applicable
    if [[ "$QUIET" != true ]]; then
        generate_summary
        
        if [[ -n "$OUTPUT" ]]; then
            echo -e "\n${GREEN}Report saved to: $OUTPUT${NC}"
        fi
        
        if [[ $exit_code -eq 0 ]]; then
            echo -e "${GREEN}✓ Security scan completed successfully${NC}"
        else
            echo -e "${YELLOW}⚠ Security scan found issues${NC}"
        fi
    fi
    
    exit $exit_code
}

# Run main function
main