# E2E Test Makefile for APM Stack

# Variables
COMPOSE_FILE := docker-compose.test.yml
GO_TEST_FLAGS := -v -timeout 30m
SEMGREP_IMAGE := returntocorp/semgrep:latest

# Colors for output
GREEN := \033[0;32m
RED := \033[0;31m
YELLOW := \033[0;33m
NC := \033[0m # No Color

.PHONY: all help setup teardown test test-prometheus test-grafana test-loki test-jaeger test-alertmanager test-semgrep test-health clean logs

# Default target
all: help

# Help target
help:
	@echo "$(GREEN)APM E2E Test Suite$(NC)"
	@echo ""
	@echo "Available targets:"
	@echo "  $(YELLOW)setup$(NC)              - Start all APM services using docker-compose"
	@echo "  $(YELLOW)teardown$(NC)           - Stop and remove all APM services"
	@echo "  $(YELLOW)test$(NC)               - Run all e2e tests"
	@echo "  $(YELLOW)test-prometheus$(NC)    - Run Prometheus tests only"
	@echo "  $(YELLOW)test-grafana$(NC)       - Run Grafana tests only"
	@echo "  $(YELLOW)test-loki$(NC)          - Run Loki tests only"
	@echo "  $(YELLOW)test-jaeger$(NC)        - Run Jaeger tests only"
	@echo "  $(YELLOW)test-alertmanager$(NC)  - Run AlertManager tests only"
	@echo "  $(YELLOW)test-semgrep$(NC)       - Run Semgrep tests only"
	@echo "  $(YELLOW)test-health$(NC)        - Run health check tests only"
	@echo "  $(YELLOW)test-integration$(NC)   - Run integration tests only"
	@echo "  $(YELLOW)clean$(NC)              - Clean up test artifacts and volumes"
	@echo "  $(YELLOW)logs$(NC)               - Show logs from all services"
	@echo "  $(YELLOW)logs-<service>$(NC)     - Show logs from specific service"

# Setup - Start all services
setup:
	@echo "$(GREEN)Starting APM services...$(NC)"
	@docker-compose -f $(COMPOSE_FILE) up -d
	@echo "$(GREEN)Waiting for services to be ready...$(NC)"
	@sleep 30
	@echo "$(GREEN)Services are ready!$(NC)"

# Teardown - Stop all services
teardown:
	@echo "$(YELLOW)Stopping APM services...$(NC)"
	@docker-compose -f $(COMPOSE_FILE) down -v
	@echo "$(GREEN)Services stopped!$(NC)"

# Run all tests
test: setup
	@echo "$(GREEN)Running all E2E tests...$(NC)"
	@go test $(GO_TEST_FLAGS) -run Test ./...
	@echo "$(GREEN)All tests completed!$(NC)"

# Run specific test suites
test-prometheus:
	@echo "$(GREEN)Running Prometheus tests...$(NC)"
	@go test $(GO_TEST_FLAGS) -run TestPrometheusMetricsCollection ./...

test-grafana:
	@echo "$(GREEN)Running Grafana tests...$(NC)"
	@go test $(GO_TEST_FLAGS) -run TestGrafanaDashboardConnectivity ./...

test-loki:
	@echo "$(GREEN)Running Loki tests...$(NC)"
	@go test $(GO_TEST_FLAGS) -run TestLokiLogAggregation ./...

test-jaeger:
	@echo "$(GREEN)Running Jaeger tests...$(NC)"
	@go test $(GO_TEST_FLAGS) -run TestJaegerTraceCollection ./...

test-alertmanager:
	@echo "$(GREEN)Running AlertManager tests...$(NC)"
	@go test $(GO_TEST_FLAGS) -run TestAlertManagerNotifications ./...

test-semgrep:
	@echo "$(GREEN)Running Semgrep tests...$(NC)"
	@docker pull $(SEMGREP_IMAGE)
	@go test $(GO_TEST_FLAGS) -run TestSemgrepSecurityAnalysis ./...

test-health:
	@echo "$(GREEN)Running health check tests...$(NC)"
	@go test $(GO_TEST_FLAGS) -run TestHealthCheckEndpoints ./...

test-integration:
	@echo "$(GREEN)Running integration tests...$(NC)"
	@go test $(GO_TEST_FLAGS) -run TestIntegrationBetweenServices ./...

# Clean up
clean:
	@echo "$(YELLOW)Cleaning up test artifacts...$(NC)"
	@docker-compose -f $(COMPOSE_FILE) down -v
	@rm -rf test-results/
	@rm -rf coverage/
	@rm -f /tmp/test-security.go
	@echo "$(GREEN)Cleanup completed!$(NC)"

# Logs
logs:
	@docker-compose -f $(COMPOSE_FILE) logs -f

logs-prometheus:
	@docker-compose -f $(COMPOSE_FILE) logs -f prometheus

logs-grafana:
	@docker-compose -f $(COMPOSE_FILE) logs -f grafana

logs-loki:
	@docker-compose -f $(COMPOSE_FILE) logs -f loki

logs-jaeger:
	@docker-compose -f $(COMPOSE_FILE) logs -f jaeger

logs-alertmanager:
	@docker-compose -f $(COMPOSE_FILE) logs -f alertmanager

logs-app:
	@docker-compose -f $(COMPOSE_FILE) logs -f apm-app

# Development helpers
.PHONY: dev-setup dev-test dev-teardown

# Quick development setup
dev-setup:
	@echo "$(GREEN)Starting services in development mode...$(NC)"
	@docker-compose -f $(COMPOSE_FILE) up -d prometheus grafana loki jaeger alertmanager
	@echo "$(GREEN)Development services ready!$(NC)"

# Run tests without setup/teardown
dev-test:
	@echo "$(GREEN)Running tests (assuming services are already running)...$(NC)"
	@go test $(GO_TEST_FLAGS) ./...

# Quick teardown without volume removal
dev-teardown:
	@echo "$(YELLOW)Stopping services (keeping volumes)...$(NC)"
	@docker-compose -f $(COMPOSE_FILE) stop

# Service health checks
.PHONY: check-prometheus check-grafana check-loki check-jaeger check-alertmanager

check-prometheus:
	@curl -s http://localhost:9090/-/ready > /dev/null && echo "$(GREEN)✓ Prometheus is ready$(NC)" || echo "$(RED)✗ Prometheus is not ready$(NC)"

check-grafana:
	@curl -s http://localhost:3000/api/health > /dev/null && echo "$(GREEN)✓ Grafana is ready$(NC)" || echo "$(RED)✗ Grafana is not ready$(NC)"

check-loki:
	@curl -s http://localhost:3100/ready > /dev/null && echo "$(GREEN)✓ Loki is ready$(NC)" || echo "$(RED)✗ Loki is not ready$(NC)"

check-jaeger:
	@curl -s http://localhost:16686 > /dev/null && echo "$(GREEN)✓ Jaeger is ready$(NC)" || echo "$(RED)✗ Jaeger is not ready$(NC)"

check-alertmanager:
	@curl -s http://localhost:9093/-/ready > /dev/null && echo "$(GREEN)✓ AlertManager is ready$(NC)" || echo "$(RED)✗ AlertManager is not ready$(NC)"

check-all: check-prometheus check-grafana check-loki check-jaeger check-alertmanager

# Coverage report
.PHONY: test-coverage

test-coverage: setup
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	@mkdir -p coverage
	@go test $(GO_TEST_FLAGS) -coverprofile=coverage/coverage.out ./...
	@go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "$(GREEN)Coverage report generated at coverage/coverage.html$(NC)"

# Benchmark tests
.PHONY: benchmark

benchmark: setup
	@echo "$(GREEN)Running benchmark tests...$(NC)"
	@go test -bench=. -benchmem ./...

# Continuous test mode (watches for changes)
.PHONY: watch

watch:
	@echo "$(GREEN)Starting continuous test mode...$(NC)"
	@while true; do \
		go test $(GO_TEST_FLAGS) ./... || true; \
		echo "$(YELLOW)Waiting for changes...$(NC)"; \
		sleep 5; \
	done