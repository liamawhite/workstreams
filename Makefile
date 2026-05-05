VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
MODULE     := github.com/liamawhite/workspace
BINARY     := workspace
BIN_DIR    := bin

LDFLAGS := -X $(MODULE)/cmd.Version=$(VERSION) \
           -X $(MODULE)/cmd.Commit=$(COMMIT) \
           -X $(MODULE)/cmd.BuildTime=$(BUILD_TIME)

.PHONY: build install test test-unit test-functional lint fmt tidy check clean help

build: ## Build the binary to bin/workspace
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) .

install: ## Install the binary to $GOPATH/bin
	go install -ldflags "$(LDFLAGS)" .

test: test-unit test-functional ## Run all tests

test-unit: ## Run unit tests
	go test ./pkg/...

test-functional: ## Run functional tests
	go test -tags integration ./test/...

lint: ## Run golangci-lint (includes go vet and more)
	go run github.com/golangci/golangci-lint/cmd/golangci-lint run ./...

fmt: ## Run goimports
	go run golang.org/x/tools/cmd/goimports -w .

tidy: ## Run go mod tidy
	go mod tidy

check: fmt test lint tidy ## Run fmt, test, lint, and tidy

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR)

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'
