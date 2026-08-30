BINARY := forge
BIN_DIR := bin
BIN_PATH := $(BIN_DIR)/$(BINARY)
BINDIR ?= $(HOME)/.local/bin

.DEFAULT_GOAL := help

.PHONY: help build test install uninstall clean verify-fast verify-slow

help: ## Show available targets
	@awk 'BEGIN {FS = ":.*## "}; /^[a-zA-Z0-9_-]+:.*## / {printf "%-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the forge binary into bin/
	@mkdir -p $(BIN_DIR) $(CURDIR)/.cache/tokf
	# Keep tokf cache/tracking writes inside the repo for restricted environments.
	@export GOCACHE=$(CURDIR)/.cache/go-build TOKF_HOME=$(CURDIR)/.cache/tokf TOKF_DB_PATH=$(CURDIR)/.cache/tokf/tracking.db; \
	go build -o $(BIN_PATH) ./cmd/forge

test: ## Run the full Go test suite
	GOCACHE=$(CURDIR)/.cache/go-build go test ./... -count=1

install: build ## Install forge into BINDIR
	@mkdir -p $(BINDIR)
	install -m 0755 $(BIN_PATH) $(BINDIR)/forge

uninstall: ## Remove installed forge from BINDIR
	rm -f $(BINDIR)/forge

clean: ## Remove local build outputs
	rm -rf $(BIN_DIR)

verify-fast: build ## Run fast-gate verification (CLI stacks + upgrade co-ownership E2E)
	GOCACHE=$(CURDIR)/.cache/go-build go test -tags=integration -count=1 \
		-timeout=10m \
		-run "TestLocalRelease/(go-cli-cobra|python-cli-typer|csharp-cli)" ./test/
	GOCACHE=$(CURDIR)/.cache/go-build go test -tags=integration -count=1 \
		-timeout=5m \
		-run "TestUpgradePreservesCoOwnedConfigEndToEnd" ./test/

verify-slow: build ## Run slow-gate template verification (all stacks)
	GOCACHE=$(CURDIR)/.cache/go-build go test -tags=integration -count=1 \
		-timeout=20m -run "TestLocalRelease" ./test/
