.PHONY: help build run test test-verbose clean docker-up docker-down docker-logs migrate-up migrate-down

# Make test script executable
$(shell chmod +x test-xmpp.sh 2>/dev/null)

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Setup:'
	@echo '  cp .env.example .env    # Create environment file'
	@echo '  make docker-up          # Start Docker services'
	@echo ''
	@echo 'Quick testing:'
	@echo '  make test-docker        # Run tests with Docker database'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	go build -o bin/veilsupport cmd/server/main.go

run: ## Run the application
	go run cmd/server/main.go

test: ## Run tests (requires TEST_DATABASE_URL)
	@if [ -z "$$TEST_DATABASE_URL" ]; then \
		echo "ERROR: TEST_DATABASE_URL environment variable is required"; \
		echo "Example: export TEST_DATABASE_URL='postgres://user:pass@localhost:5432/dbname?sslmode=disable'"; \
		echo "For Docker Compose: make test-docker"; \
		exit 1; \
	fi
	go test ./tests -v

test-docker: ## Run tests using Docker Compose database (loads from .env)
	@if [ ! -f .env ]; then \
		echo "ERROR: .env file not found. Copy .env.example to .env first:"; \
		echo "  cp .env.example .env"; \
		exit 1; \
	fi
	@echo "Loading environment from .env file..."
	@set -a; source .env; set +a; go test ./tests -v

test-db: ## Run database tests specifically (requires TEST_DATABASE_URL)
	@if [ -z "$$TEST_DATABASE_URL" ]; then \
		echo "ERROR: TEST_DATABASE_URL environment variable is required"; \
		echo "For Docker Compose: make test-docker"; \
		exit 1; \
	fi
	go test ./tests -run TestUser -v
	go test ./tests -run TestMessage -v

test-auth: ## Run auth tests specifically (requires TEST_DATABASE_URL)
	@if [ -z "$$TEST_DATABASE_URL" ]; then \
		echo "ERROR: TEST_DATABASE_URL environment variable is required"; \
		echo "For Docker Compose: make test-docker"; \
		exit 1; \
	fi
	go test ./tests -run "TestPassword|TestJWT|^TestRegistration$$|^TestLogin$$" -v

test-xmpp: ## Run XMPP tests specifically (requires TEST_DATABASE_URL)
	@if [ -z "$TEST_DATABASE_URL" ]; then \
		echo "ERROR: TEST_DATABASE_URL environment variable is required"; \
		echo "For Docker Compose: make test-docker"; \
		exit 1; \
	fi
	go test ./tests -run TestXMPP -v

xmpp-test: ## Test XMPP connection and send test messages to Conversations
	@if [ ! -f .env ]; then \
		echo "ERROR: .env file not found. Copy .env.example to .env and configure XMPP settings"; \
		exit 1; \
	fi
	@echo "Testing XMPP connection..."
	@set -a; source .env; set +a; go run cmd/xmpp-test/main.go

gateway-test: ## Test XMPP Gateway - multiple users as separate contacts
	@if [ ! -f .env ]; then \
		echo "ERROR: .env file not found. Copy .env.example to .env and configure XMPP settings"; \
		exit 1; \
	fi
	@echo "Testing XMPP Gateway (multiple users)..."
	@set -a; source .env; set +a; go run cmd/gateway-test/main.go

realistic-bot: ## Test REALISTIC implementation - formatted single conversation
	@if [ ! -f .env ]; then \
		echo "ERROR: .env file not found. Copy .env.example to .env and configure XMPP settings"; \
		exit 1; \
	fi
	@echo "Testing REALISTIC bot with better formatting..."
	@set -a; source .env; set +a; go run cmd/realistic-bot/main.go

test-api: ## Run API tests specifically (requires TEST_DATABASE_URL)
	@if [ -z "$$TEST_DATABASE_URL" ]; then \
		echo "ERROR: TEST_DATABASE_URL environment variable is required"; \
		echo "For Docker Compose: make test-docker"; \
		exit 1; \
	fi
	go test ./tests -run "TestRegisterEndpoint|TestLoginEndpoint|TestSendMessage|TestGetHistory" -v

test-verbose: ## Run tests with verbose output (requires TEST_DATABASE_URL)
	@if [ -z "$$TEST_DATABASE_URL" ]; then \
		echo "ERROR: TEST_DATABASE_URL environment variable is required"; \
		echo "For Docker Compose: make test-docker"; \
		exit 1; \
	fi
	go test ./... -v

test-coverage: ## Run tests with coverage (requires TEST_DATABASE_URL)
	@if [ -z "$$TEST_DATABASE_URL" ]; then \
		echo "ERROR: TEST_DATABASE_URL environment variable is required"; \
		echo "For Docker Compose: make test-docker"; \
		exit 1; \
	fi
	go test ./... -cover -coverprofile=coverage.out
	go tool cover -html=coverage.out

test-coverage-docker: ## Run tests with coverage using Docker Compose database (loads from .env)
	@if [ ! -f .env ]; then \
		echo "ERROR: .env file not found. Copy .env.example to .env first:"; \
		echo "  cp .env.example .env"; \
		exit 1; \
	fi
	@echo "Loading environment from .env file..."
	@set -a; source .env; set +a; go test ./... -cover -coverprofile=coverage.out
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

setup: ## Setup development environment (copy .env.example to .env)
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "✅ Created .env file from .env.example"; \
		echo "📝 Edit .env to customize your configuration"; \
	else \
		echo "⚠️  .env file already exists"; \
	fi

dev-setup: setup docker-up deps ## Complete development setup
	@echo "✅ Development environment ready!"
	@echo "📋 Next steps:"
	@echo "  make test-docker      # Run tests"
	@echo "  make run              # Start the server"

all: clean deps build test ## Clean, install deps, build and test