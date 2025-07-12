# Contributing to APM

Thank you for considering contributing to APM! This document outlines the process for contributing to the project and how to get started.

## Code of Conduct

By participating in this project, you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md).

## How to Contribute

### Reporting Bugs

If you find a bug, please report it by creating an issue on our [GitHub issue tracker](https://github.com/chaksack/apm/issues). When filing an issue, please include:

- A clear and descriptive title
- A detailed description of the issue
- Steps to reproduce the bug
- Expected behavior
- Actual behavior
- Screenshots or logs (if applicable)
- Environment information (OS, Go version, etc.)

### Suggesting Enhancements

We welcome suggestions for enhancements! Please create an issue on our [GitHub issue tracker](https://github.com/chaksack/apm/issues) with:

- A clear and descriptive title
- A detailed description of the proposed enhancement
- Any relevant examples or use cases
- If applicable, mock-ups or diagrams

### Pull Requests

1. Fork the repository
2. Create a new branch for your feature or bug fix
3. Make your changes
4. Add or update tests as necessary
5. Ensure all tests pass
6. Update documentation as needed
7. Submit a pull request

#### Pull Request Guidelines

- Follow the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Follow the style and formatting of the existing code
- Include tests for new features or bug fixes
- Update documentation as needed
- Keep pull requests focused on a single topic
- Reference any relevant issues in your PR description

## Development Setup

### Prerequisites

- Go 1.22 or higher
- Docker (for running integration tests)
- Git

### Local Development

1. Clone the repository:
   ```bash
   git clone https://github.com/chaksack/apm.git
   cd apm
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Run tests:
   ```bash
   go test ./...
   ```

4. Build the project:
   ```bash
   go build -o apm ./cmd/apm/
   ```

## License

By contributing to APM, you agree that your contributions will be licensed under the project's [Apache License 2.0](LICENSE).

## Questions?

If you have any questions, feel free to create an issue or reach out to the maintainers:

- Andrew Chakdahah - [chakdahah@gmail.com](mailto:chakdahah@gmail.com)
- Yaw Boateng Kessie - [ybkess@gmail.com](mailto:ybkess@gmail.com)

Thank you for your contribution!