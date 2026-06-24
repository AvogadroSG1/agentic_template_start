BINARY := mkproj
BIN_DIR := bin
BIN_PATH := $(BIN_DIR)/$(BINARY)

.DEFAULT_GOAL := help

.PHONY: help build test clean

help: ## Show available targets
	@awk 'BEGIN {FS = ": ## "}; /^[a-zA-Z0-9_-]+: ## / {printf "%-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the mkproj binary into bin/
	@mkdir -p $(BIN_DIR)
	@export GOCACHE=$(PWD)/.cache/go-build; \
	go build -o $(BIN_PATH) ./cmd/mkproj

test: ## Run the full Go test suite
	GOCACHE=$(PWD)/.cache/go-build go test ./... -count=1

clean: ## Remove local build outputs
	rm -rf $(BIN_DIR)
