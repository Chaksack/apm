#!/bin/bash
set -e

# Test script for APM CLI commands
echo "Testing APM CLI commands..."

# Always build the CLI for the current system
echo "Building APM CLI..."
cd ..
go build -o apm ./cmd/cli
cd - > /dev/null

# Function to run a command and check its exit status
run_test() {
    local cmd="$1"
    local description="$2"

    echo "----------------------------------------------"
    echo "Testing: $description"
    echo "Command: $cmd"

    if eval "$cmd"; then
        echo "✅ PASS: $description"
        return 0
    else
        echo "❌ FAIL: $description"
        return 1
    fi
}

# Test the version command
run_test "../apm --version" "Version command"

# Test the help command
run_test "../apm --help" "Help command"

# Test init command
run_test "../apm init --help" "Init help command"
run_test "../apm init --name test-project --type generic --env local" "Init command with options"

# Test run command
run_test "../apm run --help" "Run help command"
run_test "../apm run --stack-only --detach" "Run stack-only command"

# Test test command
run_test "../apm test --help" "Test help command"
run_test "../apm test --config-only" "Test config-only command"

# Test dashboard command
run_test "../apm dashboard --help" "Dashboard help command"
run_test "../apm dashboard --list" "Dashboard list command"

echo "----------------------------------------------"
echo "All tests completed."
