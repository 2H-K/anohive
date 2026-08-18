.PHONY: all build test clean run-server run-cli frontend

BINARY_NAME=anohive
CLI_NAME=anohive-cli
BUILD_DIR=./build
GO=go
GOFLAGS=-v

all: build

build:
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(CLI_NAME) ./cmd/cli
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME), $(BUILD_DIR)/$(CLI_NAME)"

test:
	$(GO) test -v -race -coverprofile=coverage.out ./internal/...
	$(GO) tool cover -func=coverage.out

test-short:
	$(GO) test -short ./internal/...

clean:
	rm -rf $(BUILD_DIR) coverage.out anohive.db
	@echo "Cleaned"

run-server: build
	$(BUILD_DIR)/$(BINARY_NAME) -port 8080

run-cli: build
	$(BUILD_DIR)/$(CLI_NAME)

deps:
	$(GO) mod download
	$(GO) mod tidy

lint:
	@which golangci-lint > /dev/null 2>&1 && golangci-lint ./... || $(GO) vet ./...

fmt:
	$(GO) fmt ./...

frontend:
	cd web && npm install && npm run build

frontend-dev:
	cd web && npm install && npm run dev

.DEFAULT_GOAL := build
