.PHONY: build clean test docker-build docker-push deploy undeploy logs

# Variables
APP_NAME := k8s-external-ip-powerdns
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
DOCKER_REGISTRY := ghcr.io
DOCKER_IMAGE := $(DOCKER_REGISTRY)/$(shell git config --get remote.origin.url | sed 's/.*[:/]\([^/]*\/[^/]*\)\.git$$/\1/' | tr '[:upper:]' '[:lower:]')/$(APP_NAME)
DOCKER_TAG := $(VERSION)

# Go build variables
GOOS := linux
GOARCH := amd64
CGO_ENABLED := 0

# Build flags
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE)

# Build the Go binary
build:
	@echo "Building $(APP_NAME) version $(VERSION)..."
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) go build \
		-ldflags "-s -w -X main.version=$(VERSION)" \
		-o bin/$(APP_NAME) .

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Build Docker image
docker-build:
	@echo "Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest

# Push Docker image
docker-push: docker-build
	@echo "Pushing Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	docker push $(DOCKER_IMAGE):latest

# Deploy to Kubernetes
deploy:
	@echo "Deploying to Kubernetes..."
	kubectl apply -f k8s-deployment.yaml

# Remove from Kubernetes
undeploy:
	@echo "Removing from Kubernetes..."
	kubectl delete -f k8s-deployment.yaml

# Show application logs
logs:
	@echo "Showing application logs..."
	kubectl logs -f deployment/$(APP_NAME)

# Run locally with environment variables
run-local: build
	@echo "Running locally..."
	@if [ ! -f .env ]; then \
		echo "Creating .env from .env.example..."; \
		cp .env.example .env; \
		echo "Please edit .env with your configuration before running"; \
		exit 1; \
	fi
	@set -a && source .env && set +a && ./bin/$(APP_NAME)

# Development setup
dev-setup:
	@echo "Setting up development environment..."
	go mod download
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "Created .env file. Please edit it with your configuration."; \
	fi

# Check for required tools
check-tools:
	@echo "Checking required tools..."
	@command -v go >/dev/null 2>&1 || { echo "Go is required but not installed. Aborting." >&2; exit 1; }
	@command -v docker >/dev/null 2>&1 || { echo "Docker is required but not installed. Aborting." >&2; exit 1; }
	@command -v kubectl >/dev/null 2>&1 || { echo "kubectl is required but not installed. Aborting." >&2; exit 1; }
	@echo "All required tools are available."

# Show help
help:
	@echo "Available targets:"
	@echo "  build         - Build the Go binary"
	@echo "  clean         - Clean build artifacts"
	@echo "  test          - Run tests"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-push   - Build and push Docker image"
	@echo "  deploy        - Deploy to Kubernetes"
	@echo "  undeploy      - Remove from Kubernetes"
	@echo "  logs          - Show application logs"
	@echo "  run-local     - Run locally with .env configuration"
	@echo "  dev-setup     - Set up development environment"
	@echo "  check-tools   - Check for required tools"
	@echo "  help          - Show this help message"
