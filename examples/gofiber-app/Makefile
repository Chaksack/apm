# Makefile for GoFiber Example Application

# Variables
APP_NAME := gofiber-example
VERSION := 1.0.0
DOCKER_IMAGE := $(APP_NAME):$(VERSION)
DOCKER_REGISTRY := localhost:5000
GO_VERSION := 1.24.4
PORT := 3000

# Build variables
BUILD_DIR := ./build
BINARY_NAME := $(APP_NAME)
MAIN_GO := ./main.go

# Go build flags
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)"

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m # No Color

.PHONY: help
help: ## Display this help message
	@echo "$(GREEN)GoFiber Example Application Makefile$(NC)"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-20s$(NC) %s\n", $$1, $$2}'

.PHONY: build
build: ## Build the application binary
	@echo "$(GREEN)Building $(APP_NAME)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_GO)
	@echo "$(GREEN)Build complete: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

.PHONY: run
run: ## Run the application locally
	@echo "$(GREEN)Running $(APP_NAME) on port $(PORT)...$(NC)"
	@go run $(MAIN_GO)

.PHONY: dev
dev: ## Run the application with hot reload (requires air)
	@if ! command -v air &> /dev/null; then \
		echo "$(YELLOW)Installing air for hot reload...$(NC)"; \
		go install github.com/air-verse/air@latest; \
	fi
	@echo "$(GREEN)Running $(APP_NAME) in development mode...$(NC)"
	@air

.PHONY: test
test: ## Run tests
	@echo "$(GREEN)Running tests...$(NC)"
	@go test -v -race -coverprofile=coverage.out ./...
	@echo "$(GREEN)Tests complete$(NC)"

.PHONY: test-coverage
test-coverage: test ## Run tests and display coverage
	@echo "$(GREEN)Generating coverage report...$(NC)"
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

.PHONY: benchmark
benchmark: ## Run benchmarks
	@echo "$(GREEN)Running benchmarks...$(NC)"
	@go test -bench=. -benchmem ./...

.PHONY: lint
lint: ## Run linter (requires golangci-lint)
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "$(YELLOW)Installing golangci-lint...$(NC)"; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s latest; \
	fi
	@echo "$(GREEN)Running linter...$(NC)"
	@golangci-lint run ./...

.PHONY: fmt
fmt: ## Format code
	@echo "$(GREEN)Formatting code...$(NC)"
	@go fmt ./...
	@echo "$(GREEN)Code formatted$(NC)"

.PHONY: tidy
tidy: ## Tidy go modules
	@echo "$(GREEN)Tidying go modules...$(NC)"
	@go mod tidy
	@echo "$(GREEN)Modules tidied$(NC)"

.PHONY: clean
clean: ## Clean build artifacts
	@echo "$(GREEN)Cleaning build artifacts...$(NC)"
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "$(GREEN)Clean complete$(NC)"

# Docker targets
.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "$(GREEN)Building Docker image $(DOCKER_IMAGE)...$(NC)"
	@docker build -t $(DOCKER_IMAGE) .
	@echo "$(GREEN)Docker image built: $(DOCKER_IMAGE)$(NC)"

.PHONY: docker-run
docker-run: docker-build ## Run the application in Docker
	@echo "$(GREEN)Running Docker container...$(NC)"
	@docker run --rm -p $(PORT):$(PORT) \
		--name $(APP_NAME) \
		-e PORT=$(PORT) \
		$(DOCKER_IMAGE)

.PHONY: docker-push
docker-push: docker-build ## Push Docker image to registry
	@echo "$(GREEN)Pushing Docker image to $(DOCKER_REGISTRY)...$(NC)"
	@docker tag $(DOCKER_IMAGE) $(DOCKER_REGISTRY)/$(DOCKER_IMAGE)
	@docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE)
	@echo "$(GREEN)Docker image pushed$(NC)"

.PHONY: docker-compose-up
docker-compose-up: ## Start services with docker-compose
	@echo "$(GREEN)Starting services with docker-compose...$(NC)"
	@docker-compose up -d
	@echo "$(GREEN)Services started$(NC)"

.PHONY: docker-compose-down
docker-compose-down: ## Stop services with docker-compose
	@echo "$(GREEN)Stopping services with docker-compose...$(NC)"
	@docker-compose down
	@echo "$(GREEN)Services stopped$(NC)"

.PHONY: docker-compose-logs
docker-compose-logs: ## View docker-compose logs
	@docker-compose logs -f

# Monitoring targets
.PHONY: metrics
metrics: ## View application metrics
	@echo "$(GREEN)Opening metrics endpoint...$(NC)"
	@open http://localhost:$(PORT)/metrics || xdg-open http://localhost:$(PORT)/metrics

.PHONY: health
health: ## Check application health
	@echo "$(GREEN)Checking application health...$(NC)"
	@curl -s http://localhost:$(PORT)/health | jq '.' || echo "$(RED)Application is not running$(NC)"

.PHONY: readiness
readiness: ## Check application readiness
	@echo "$(GREEN)Checking application readiness...$(NC)"
	@curl -s http://localhost:$(PORT)/ready | jq '.' || echo "$(RED)Application is not ready$(NC)"

# Load testing
.PHONY: load-test
load-test: ## Run load test (requires hey)
	@if ! command -v hey &> /dev/null; then \
		echo "$(YELLOW)Installing hey...$(NC)"; \
		go install github.com/rakyll/hey@latest; \
	fi
	@echo "$(GREEN)Running load test...$(NC)"
	@hey -z 30s -c 10 http://localhost:$(PORT)/api/v1/users

# Database targets
.PHONY: migrate-up
migrate-up: ## Run database migrations up
	@echo "$(GREEN)Running migrations up...$(NC)"
	@go run ./cmd/migrate up

.PHONY: migrate-down
migrate-down: ## Run database migrations down
	@echo "$(GREEN)Running migrations down...$(NC)"
	@go run ./cmd/migrate down

.PHONY: migrate-create
migrate-create: ## Create a new migration (usage: make migrate-create name=migration_name)
	@echo "$(GREEN)Creating migration: $(name)...$(NC)"
	@go run ./cmd/migrate create $(name)

# Utility targets
.PHONY: deps
deps: ## Download dependencies
	@echo "$(GREEN)Downloading dependencies...$(NC)"
	@go mod download
	@echo "$(GREEN)Dependencies downloaded$(NC)"

.PHONY: update-deps
update-deps: ## Update dependencies
	@echo "$(GREEN)Updating dependencies...$(NC)"
	@go get -u ./...
	@go mod tidy
	@echo "$(GREEN)Dependencies updated$(NC)"

.PHONY: generate
generate: ## Run go generate
	@echo "$(GREEN)Running go generate...$(NC)"
	@go generate ./...
	@echo "$(GREEN)Generation complete$(NC)"

.PHONY: install
install: build ## Install the application
	@echo "$(GREEN)Installing $(APP_NAME)...$(NC)"
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/
	@echo "$(GREEN)Installation complete$(NC)"

.PHONY: uninstall
uninstall: ## Uninstall the application
	@echo "$(GREEN)Uninstalling $(APP_NAME)...$(NC)"
	@rm -f $(GOPATH)/bin/$(BINARY_NAME)
	@echo "$(GREEN)Uninstallation complete$(NC)"

# Development setup
.PHONY: setup
setup: deps ## Setup development environment
	@echo "$(GREEN)Setting up development environment...$(NC)"
	@go install github.com/air-verse/air@latest
	@go install github.com/rakyll/hey@latest
	@echo "$(GREEN)Development environment ready$(NC)"

# All target
.PHONY: all
all: clean deps lint test build ## Run all targets (clean, deps, lint, test, build)
	@echo "$(GREEN)All targets completed successfully$(NC)"

# Default target
.DEFAULT_GOAL := help