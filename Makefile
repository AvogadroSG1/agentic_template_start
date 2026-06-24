BINARY := mkproj
BIN_DIR := bin
BIN_PATH := $(BIN_DIR)/$(BINARY)
BINDIR ?= $(HOME)/.local/bin

.DEFAULT_GOAL := help

.PHONY: help build test install uninstall clean

help: ## Show available targets
	@awk 'BEGIN {FS = ":.*## "}; /^[a-zA-Z0-9_-]+:.*## / {printf "%-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the mkproj binary into bin/
	@mkdir -p $(BIN_DIR) $(PWD)/.cache/tokf
	# Keep tokf cache/tracking writes inside the repo for restricted environments.
	@export GOCACHE=$(PWD)/.cache/go-build TOKF_HOME=$(PWD)/.cache/tokf TOKF_DB_PATH=$(PWD)/.cache/tokf/tracking.db; \
	go build -o $(BIN_PATH) ./cmd/mkproj

test: ## Run the full Go test suite
	GOCACHE=$(PWD)/.cache/go-build go test ./... -count=1

install: build ## Install mkproj into BINDIR
	@mkdir -p $(BINDIR)
	install -m 0755 $(BIN_PATH) $(BINDIR)/mkproj

uninstall: ## Remove installed mkproj from BINDIR
	rm -f $(BINDIR)/mkproj

clean: ## Remove local build outputs
	rm -rf $(BIN_DIR)
