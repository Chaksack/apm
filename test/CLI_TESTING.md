# APM CLI Testing

This document describes the testing of the APM CLI commands.

## Test Script

The test script `cli_test.sh` tests the following APM CLI commands:

1. **Version Command**: `apm --version`
   - Displays the version information of the APM CLI.

2. **Help Command**: `apm --help`
   - Shows the help information for the root command, listing all available subcommands and flags.

3. **Init Command**:
   - `apm init --help`: Shows help for the init command.
   - `apm init --name test-project --type generic --env local`: Initializes a new APM configuration with specified options.

4. **Run Command**:
   - `apm run --help`: Shows help for the run command.
   - `apm run --stack-only --detach`: Runs only the monitoring stack in detached mode.

5. **Test Command**:
   - `apm test --help`: Shows help for the test command.
   - `apm test --config-only`: Validates the configuration without testing.

6. **Dashboard Command**:
   - `apm dashboard --help`: Shows help for the dashboard command.
   - `apm dashboard --list`: Lists all available dashboards.

## Test Results

All tests passed successfully, indicating that the CLI commands are working properly.

## Running the Tests

To run the tests:

1. Navigate to the test directory:
   ```
   cd test
   ```

2. Make the test script executable (if not already):
   ```
   chmod +x cli_test.sh
   ```

3. Run the test script:
   ```
   ./cli_test.sh
   ```

## Notes

- The test script rebuilds the CLI binary for the current system to ensure compatibility.
- The tests only verify that the commands execute without errors, not that they perform their intended functions correctly.
- For more comprehensive testing, additional tests should be added to verify the actual functionality of each command.