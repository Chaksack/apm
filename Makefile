.PHONY: help build up down logs ps clean volumes restart build-app run dev test \
	test-coverage lint gosec semgrep semgrep-json security-scan vulncheck \
	sonar-scan quality-check prometheus-reload grafana-logs jaeger-ui \
	prometheus-ui grafana-ui alertmanager-ui sample-app-logs sample-app-restart \
	test-endpoints metrics test-e2e test-e2e-parallel test-e2e-load \
	test-e2e-security test-e2e-monitoring test-e2e-alerts test-e2e-integration \
	test-e2e-report

# Default target
.DEFAULT_GOAL := help

# Go build variables
BINARY_NAME=apm
MAIN_PATH=cmd/apm/main.go

# Help target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Go application commands
build-app: ## Build the GoFiber APM application
	go build -o $(BINARY_NAME) $(MAIN_PATH)

run: build-app ## Build and run the GoFiber APM application
	./$(BINARY_NAME)

dev: ## Run the GoFiber app with air (hot reload)
	@if ! command -v air &> /dev/null; then \
		echo "air is not installed. Installing..."; \
		go install github.com/cosmtrek/air@latest; \
	fi
	air

test: ## Run all tests
	go test -v ./...

test-coverage: ## Run tests with coverage
	go test -v -cover ./...

# Docker Compose commands
build: ## Build all Docker images
	docker-compose build

up: ## Start all services
	docker-compose up -d

down: ## Stop all services
	docker-compose down

logs: ## View logs for all services
	docker-compose logs -f

ps: ## Show running containers
	docker-compose ps

clean: ## Stop services and remove volumes
	docker-compose down -v

volumes: ## List Docker volumes
	docker volume ls | grep apm

restart: ## Restart all services
	docker-compose restart

# Service-specific commands
prometheus-reload: ## Reload Prometheus configuration
	curl -X POST http://localhost:9090/-/reload

grafana-logs: ## View Grafana logs
	docker-compose logs -f grafana

jaeger-ui: ## Open Jaeger UI in browser
	open http://localhost:16686

prometheus-ui: ## Open Prometheus UI in browser
	open http://localhost:9090

grafana-ui: ## Open Grafana UI in browser
	open http://localhost:3000

alertmanager-ui: ## Open AlertManager UI in browser
	open http://localhost:9093

# Development commands
sample-app-logs: ## View sample app logs
	docker-compose logs -f sample-app

sample-app-restart: ## Restart sample app
	docker-compose restart sample-app

test-endpoints: ## Test sample app endpoints
	@echo "Testing health endpoint..."
	@curl -s http://localhost:8080/health
	@echo "\n\nTesting root endpoint..."
	@curl -s http://localhost:8080/
	@echo "\n\nTesting error endpoint..."
	@curl -s http://localhost:8080/error
	@echo "\n\nTesting slow endpoint..."
	@curl -s http://localhost:8080/slow
	@echo "\n"

metrics: ## View sample app metrics
	curl -s http://localhost:8081/metrics | grep -E "^(http_|go_|process_)"

# Security and Quality commands
lint: ## Run golangci-lint
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "golangci-lint is not installed. Installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	golangci-lint run

gosec: ## Run gosec security scanner
	@if ! command -v gosec &> /dev/null; then \
		echo "gosec is not installed. Installing..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
	fi
	gosec ./...

semgrep: ## Run Semgrep security analysis
	@if ! command -v semgrep &> /dev/null; then \
		echo "Semgrep is not installed. Please install it:"; \
		echo "  pip install semgrep"; \
		echo "  or: brew install semgrep"; \
		exit 1; \
	fi
	./scripts/semgrep-scan.sh

semgrep-json: ## Run Semgrep and output JSON report
	./scripts/semgrep-scan.sh --format json --output security-reports/semgrep-report.json

security-scan: lint gosec semgrep ## Run all security scans

vulncheck: ## Run Go vulnerability check
	@if ! command -v govulncheck &> /dev/null; then \
		echo "govulncheck is not installed. Installing..."; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
	fi
	govulncheck ./...

sonar-scan: ## Run SonarQube scan
	./scripts/sonar-scan.sh

quality-check: test-coverage lint security-scan ## Run all quality checks

# E2E Testing commands
test-e2e: ## Run all E2E tests sequentially
	cd test/e2e && make test

test-e2e-parallel: ## Run E2E tests in parallel
	cd test/e2e && ./scripts/run_parallel_tests.sh all

test-e2e-load: ## Run load testing scenario
	cd test/e2e && ./scripts/run_parallel_tests.sh load

test-e2e-security: ## Run security testing scenario
	cd test/e2e && ./scripts/run_parallel_tests.sh security

test-e2e-monitoring: ## Run monitoring pipeline tests
	cd test/e2e && ./scripts/run_parallel_tests.sh monitoring

test-e2e-alerts: ## Run alert testing scenario
	cd test/e2e && ./scripts/run_parallel_tests.sh alerts

test-e2e-integration: ## Run full integration tests
	cd test/e2e && ./scripts/run_parallel_tests.sh integration

test-e2e-report: ## Generate E2E test report
	@echo "Generating E2E test report..."
	@if [ -f test/e2e/test-report.json ]; then \
		cd test/e2e && go run -tags=report ./report_generator.go; \
	else \
		echo "No test report found. Run tests first."; \
	fi