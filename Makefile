# Makefile for Anker Solix Exporter

# Variables
BINARY_NAME=anker-solix-exporter
VERSION?=dev
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags="-w -s -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}"

# Docker
DOCKER_IMAGE?=ghcr.io/yourusername/anker-solix-exporter
DOCKER_TAG?=latest

# Helm
HELM_CHART=./deployments/helm/anker-solix-exporter

.PHONY: all build test clean docker-build docker-push helm-lint helm-install

all: test build

## Build the binary
build:
	@echo "Building ${BINARY_NAME}..."
	CGO_ENABLED=0 go build ${LDFLAGS} -o ${BINARY_NAME} ./cmd/exporter

## Run tests
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

## Run tests with coverage
coverage: test
	go tool cover -html=coverage.out

## Run linters
lint:
	@echo "Running linters..."
	go vet ./...
	gofmt -s -w .

## Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f ${BINARY_NAME}
	rm -f coverage.out

## Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build \
		--build-arg VERSION=${VERSION} \
		--build-arg COMMIT=${COMMIT} \
		--build-arg DATE=${DATE} \
		-t ${DOCKER_IMAGE}:${DOCKER_TAG} \
		.

## Build multi-arch Docker images
docker-buildx:
	@echo "Building multi-arch Docker image..."
	docker buildx build \
		--platform linux/amd64,linux/arm64,linux/arm/v7 \
		--build-arg VERSION=${VERSION} \
		--build-arg COMMIT=${COMMIT} \
		--build-arg DATE=${DATE} \
		-t ${DOCKER_IMAGE}:${DOCKER_TAG} \
		--push \
		.

## Push Docker image
docker-push: docker-build
	@echo "Pushing Docker image..."
	docker push ${DOCKER_IMAGE}:${DOCKER_TAG}

## Lint Helm chart
helm-lint:
	@echo "Linting Helm chart..."
	helm lint ${HELM_CHART}

## Package Helm chart
helm-package:
	@echo "Packaging Helm chart..."
	helm package ${HELM_CHART}

## Install Helm chart (for development)
helm-install:
	@echo "Installing Helm chart..."
	helm install anker-solix-exporter ${HELM_CHART} \
		--create-namespace \
		--namespace monitoring \
		-f values.yaml

## Upgrade Helm chart
helm-upgrade:
	@echo "Upgrading Helm chart..."
	helm upgrade anker-solix-exporter ${HELM_CHART} \
		--namespace monitoring \
		-f values.yaml

## Uninstall Helm chart
helm-uninstall:
	@echo "Uninstalling Helm chart..."
	helm uninstall anker-solix-exporter --namespace monitoring

## Run locally
run:
	@echo "Running locally..."
	go run ${LDFLAGS} ./cmd/exporter -config config.yaml

## Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

## Show help
help:
	@echo "Available targets:"
	@echo "  build          - Build the binary"
	@echo "  test           - Run tests"
	@echo "  coverage       - Run tests with coverage report"
	@echo "  lint           - Run linters"
	@echo "  clean          - Clean build artifacts"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-buildx  - Build multi-arch Docker images"
	@echo "  docker-push    - Push Docker image"
	@echo "  helm-lint      - Lint Helm chart"
	@echo "  helm-package   - Package Helm chart"
	@echo "  helm-install   - Install Helm chart"
	@echo "  helm-upgrade   - Upgrade Helm chart"
	@echo "  helm-uninstall - Uninstall Helm chart"
	@echo "  run            - Run locally"
	@echo "  deps           - Install dependencies"
	@echo "  help           - Show this help"
