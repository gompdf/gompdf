.PHONY: all build test lint clean

# Default target
all: lint test build

# Build the CLI tool
build:
	go build -o bin/gompdf ./cmd/gompdf

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run linter
lint:
	golangci-lint run

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Install development dependencies
dev-deps:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Generate documentation
docs:
	@echo "Generating documentation..."
	# Add documentation generation commands here

# Run example
example:
	go run ./examples/invoice-basic/main.go

# Install the CLI tool
install:
	go install ./cmd/gompdf

# Format code
fmt:
	go fmt ./...

# Vet code
vet:
	go vet ./...

# Run all checks (lint, vet, test)
check: lint vet test
