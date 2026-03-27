# ============================================
# LickLib Makefile
# Shortcuts für Development, Testing, Docker
# ============================================

.PHONY: help

# Default target: Zeigt Hilfe
help:
	@echo "=========================================="
	@echo "LickLib - Available Commands"
	@echo "=========================================="
	@echo ""
	@echo "Development:"
	@echo "  make dev           - Start dev environment (Postgres, MinIO, Keycloak)"
	@echo "  make dev-down      - Stop dev environment"
	@echo "  make dev-logs      - Show dev logs"
	@echo ""
	@echo "Testing:"
	@echo "  make test          - Run all tests (starts test DB automatically)"
	@echo "  make test-unit     - Run only unit tests (fast, no DB)"
	@echo "  make test-integration - Run integration tests (with DB)"
	@echo "  make test-coverage - Run tests with coverage report"
	@echo ""
	@echo "Database:"
	@echo "  make migrate-up    - Run migrations (dev DB)"
	@echo "  make migrate-down  - Rollback migrations (dev DB)"
	@echo "  make migrate-test  - Run migrations (test DB)"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build  - Build backend Docker image"
	@echo "  make prod-start    - Build + start production-like stack"
	@echo "  make prod-stop     - Stop production stack"
	@echo "  make docker-logs   - Show backend logs"
	@echo ""
	@echo "Cleanup:"
	@echo "  make clean         - Clean all Docker resources"
	@echo "  make clean-dev     - Clean only dev environment"
	@echo "  make clean-test    - Clean only test environment"
	@echo ""

# ============================================
# DEVELOPMENT ENVIRONMENT
# Startet: Postgres, MinIO, Keycloak
# ============================================

dev:
	@echo "Starting dev environment..."
	cd .extras/docker/dev && docker compose up -d
	@echo " (Dev is running) Dev environment started!"
	@echo "   Postgres:  localhost:5432 (user: postgres, db: licks)"
	@echo "   MinIO:     http://localhost:9000 (admin: minio_admin_user)"
	@echo "   MinIO UI:  http://localhost:9001"
	@echo "   Keycloak:  http://localhost:8081"

dev-down:
	@echo "🛑 Stopping dev environment..."
	cd extras/docker/dev && docker compose down

dev-logs:
	cd extras/docker/dev && docker compose logs -f

dev-restart: dev-down dev

# ============================================
# TESTING
# ============================================

# Start test DB, run tests, cleanup
test: test-up
	@echo  "Running tests..."
	@sleep 3
	go test ./... -v
	@$(MAKE) test-down

# Nur Unit Tests (ohne DB)
test-unit:
	@echo "Running unit tests..."
	go test ./... -v -short

# Integration Tests (mit DB Tag)
test-integration: test-up
	@echo "Running integration tests..."
	@sleep 3
	go test ./... -v -tags=integration
	@$(MAKE) test-down

# Test Coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test ./... -cover -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Test DB starten
test-up:
	@echo "Starting test database..."
	cd extras/docker/test && docker compose up -d
	@sleep 2

# Test DB stoppen
test-down:
	cd extras/docker/test && docker compose down

# ============================================
# DATABASE MIGRATIONS
# ============================================

migrate-up:
	@echo "Running migrations (dev)..."
	migrate -path migrations \
		-database "postgres://postgres:postgres@localhost:5432/licks?sslmode=disable" \
		up

migrate-down:
	@echo "Rolling back migrations (dev)..."
	migrate -path migrations \
		-database "postgres://postgres:postgres@localhost:5432/licks?sslmode=disable" \
		down

migrate-test:
	@echo "Running migrations (test DB)..."
	migrate -path migrations \
		-database "postgres://test:test@localhost:5434/test?sslmode=disable" \
		up

# ============================================
# DOCKER & PRODUCTION
# ============================================

# Build Backend Image
docker-build:
	@echo "Building Docker image..."
	docker build -t licklib:latest .
	@echo "Image built: licklib:latest"

# Production-like Stack (Backend + Dependencies)
prod-start: docker-build
	@echo "Starting production stack..."
	cd extras/docker/prod && docker compose up -d
	@echo "Production stack started!"
	@echo "   Backend:   http://localhost:8000"
	@echo "   Health:    http://localhost:8000/health"
	@echo "   MinIO UI:  http://localhost:9001"

prod-stop:
	@echo "Stopping production stack..."
	cd extras/docker/prod && docker compose down

docker-logs:
	cd extras/docker/prod && docker compose logs -f backend

docker-shell:
	docker exec -it licklib-backend sh

# ============================================
# CLEANUP
# ============================================

clean-dev:
	@echo "Cleaning dev environment..."
	cd extras/docker/dev && docker compose down -v

clean-test:
	@echo "Cleaning test environment..."
	cd extras/docker/test && docker compose down -v

clean-prod:
	@echo "Cleaning production stack..."
	cd extras/docker/prod && docker compose down -v

clean: clean-dev clean-test clean-prod
	@echo "Cleaning Docker resources..."
	docker network prune -f
	docker volume prune -f
	docker image prune -f
	@echo "Cleanup complete!"

# ============================================
# RUN APPLICATION (local development)
# ============================================

run:
	@echo "Running application locally..."
	go run cmd/main.go

# ============================================
# BUILD (local binary)
# ============================================

build:
	@echo "Building binary..."
	go build -o licklib cmd/main.go
	@echo "Binary created: ./licklib"