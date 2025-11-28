.PHONY: build run test test-verbose clean docker-build docker-run fmt vet help

BINARY_NAME=server
DOCKER_IMAGE=collaborative-docs
DOCKER_TAG=latest

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*##"; printf ""} /^[a-zA-Z_-]+:.*?##/ { printf "  %-15s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## Build the server binary
	go build -o $(BINARY_NAME) cmd/server/main.go

run: build ## Build and run the server locally
	./$(BINARY_NAME)

test: ## Run all tests
	go test ./...

test-verbose: ## Run tests with verbose output
	go test -v ./...

clean: ## Remove built binaries
	rm -f $(BINARY_NAME)
	go clean

docker-build: ## Build Docker image
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-run: ## Run Docker container
	docker run -p 8080:8080 $(DOCKER_IMAGE):$(DOCKER_TAG)

fmt: ## Format Go code
	go fmt ./...

vet: ## Run go vet
	go vet ./...
