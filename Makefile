.PHONY: help build up down logs ps clean volumes restart build-app run dev test

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