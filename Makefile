# VeilSupport Development Makefile

.PHONY: build run test test-unit test-integration test-load test-all test-watch clean sqlc-generate migrate-up migrate-down

# Build the application
build:
	go build -o bin/veilsupport cmd/server/main.go

# Run the development server
run:
	go run cmd/server/main.go

# Run all tests
test-all: test-unit test-integration test-load

# Run unit tests only
test-unit:
	go test -v ./tests/unit/... -short

# Run integration tests
test-integration:
	go test -v ./tests/integration/... -timeout=10m

# Run load tests
test-load:
	go test -v ./tests/load/... -timeout=10m

# Run tests with coverage
test-coverage:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html

# Run tests with race detector
test-race:
	go test -race -v ./...

# Watch tests (requires inotify tools)
test-watch:
	@echo "Starting test watch mode..."
	@while inotifywait -r -e modify,create,delete --exclude='\.git|bin|tmp' .; do \
		make test-unit; \
	done

# Generate type-safe SQL code with sqlc
sqlc-generate:
	sqlc generate

# Run database migrations up
migrate-up:
	migrate -path db/migrations -database "postgres://veilsupport:password@localhost:5432/veilsupport?sslmode=disable" up

# Run database migrations down
migrate-down:
	migrate -path db/migrations -database "postgres://veilsupport:password@localhost:5432/veilsupport?sslmode=disable" down

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Install development dependencies
dev-deps:
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Format code
fmt:
	go fmt ./...

# Lint code (requires golangci-lint)
lint:
	golangci-lint run

# Download dependencies
deps:
	go mod tidy
	go mod verify

# Show help
help:
	@echo "Available targets:"
	@echo "  build           Build the application"
	@echo "  run             Run the development server"
	@echo "  test-all        Run all tests"
	@echo "  test-unit       Run unit tests only"
	@echo "  test-integration Run integration tests"
	@echo "  test-load       Run load tests"
	@echo "  test-coverage   Run tests with coverage report"
	@echo "  test-race       Run tests with race detector"
	@echo "  test-watch      Watch and re-run tests on file changes"
	@echo "  sqlc-generate   Generate type-safe SQL code"
	@echo "  migrate-up      Apply database migrations"
	@echo "  migrate-down    Rollback database migrations"
	@echo "  clean           Clean build artifacts"
	@echo "  dev-deps        Install development dependencies"
	@echo "  fmt             Format code"
	@echo "  lint            Lint code"
	@echo "  deps            Download and verify dependencies"