.PHONY: help build run test test-verbose clean docker-up docker-down docker-logs migrate-up migrate-down

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	go build -o bin/veilsupport cmd/server/main.go

run: ## Run the application
	go run cmd/server/main.go

test: ## Run tests
	go test ./tests -v

test-db: ## Run database tests specifically
	go test ./tests -run TestUser -v
	go test ./tests -run TestMessage -v

test-auth: ## Run auth tests specifically
	go test ./tests -run "TestPassword|TestJWT|TestRegistration|TestLogin" -v

test-verbose: ## Run tests with verbose output
	go test ./... -v

test-coverage: ## Run tests with coverage
	go test ./... -cover -coverprofile=coverage.out
	go tool cover -html=coverage.out

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out

docker-up: ## Start Docker containers
	docker-compose up -d

docker-down: ## Stop Docker containers
	docker-compose down

docker-logs: ## View Docker logs
	docker-compose logs -f

docker-clean: ## Clean Docker containers and volumes
	docker-compose down -v
	docker system prune -f

deps: ## Install dependencies
	go mod tidy
	go mod download

lint: ## Run linter (if available)
	@if command -v golangci-lint > /dev/null; then golangci-lint run; else echo "golangci-lint not installed"; fi

fmt: ## Format code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

check: fmt vet lint ## Run all code checks

dev-setup: docker-up deps ## Setup development environment
	@echo "Development environment ready!"
	@echo "Database URLs:"
	@echo "  Main: postgres://veiluser:veilpass@localhost:5432/veilsupport"
	@echo "  Test: postgres://veiluser:veilpass@localhost:5433/veilsupport_test"

all: clean deps build test ## Clean, install deps, build and test