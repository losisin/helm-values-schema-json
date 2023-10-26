SHELL := /bin/bash

BINNAME := schema
PLUGIN_SHORTNAME := json-schema

BUILD_DATE := $(shell date -u '+%Y-%m-%d %I:%M:%S UTC' 2> /dev/null)
GIT_HASH := $(shell git rev-parse HEAD 2> /dev/null)

GOPATH ?= $(shell go env GOPATH)
PATH := $(GOPATH)/bin:$(PATH)
GO_BUILD_ENV_VARS = $(if $(GO_ENV_VARS),$(GO_ENV_VARS),CGO_ENABLED=0)
GO_BUILD_ARGS = -buildvcs=false -ldflags "-X main.GitCommit=${GIT_HASH}"

HELM_PLUGINS = $(shell helm env HELM_PLUGINS)
HELM_PLUGIN_DIR = $(HELM_PLUGINS)/$(PLUGIN_SHORTNAME)

.PHONY: build \
	install \
	verify \
	tidy \
	fmt \
	vet \
	check \
	test-unit \
	test-coverage \
	test-all \
	clean \
	help

build: ## Build the plugin
	@echo "Building plugin..."
	@${GO_BUILD_ENV_VARS} go build -o $(BINNAME) ${GO_BUILD_ARGS}

install: build ## Install the plugin
	@echo "Installing plugin..."
	@mkdir -p $(HELM_PLUGIN_DIR)
	@cp $(BINNAME) $(HELM_PLUGIN_DIR)
	@cp plugin.yaml $(HELM_PLUGIN_DIR)

verify: ## Verify the plugin
	@echo
	@echo "Verifying plugin..."
	@go mod verify

tidy: ## Tidy the plugin
	@echo
	@echo "Tidying plugin..."
	@go mod tidy

fmt: ## Format the plugin
	@echo
	@echo "Formatting plugin..."
	@go fmt ./...

vet: ## Vet the plugin
	@echo
	@echo "Vetting plugin..."
	@go vet ./...

check: verify tidy fmt vet ## Verify, tidy, fmt and vet the plugin

test-unit: ## Run unit tests
	@echo
	@echo "Running unit tests..."
	@go test -short ./...

test-coverage: ## Run tests with coverage
	@echo
	@echo "Running tests with coverage..."
	@go test -v -race -covermode=atomic -coverprofile=cover.out ./...

test-all: test-unit test-coverage ## Includes test-unit and test-coverage

clean: ## Clean the plugin
	@echo "Cleaning plugin..."
	@rm -rf $(BINNAME) $(HELM_PLUGIN_DIR)

help: ## Show this help message
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  build              Build the plugin"
	@echo "  install            Install the plugin"
	@echo "  verify             Verify the plugin"
	@echo "  tidy               Tidy the plugin"
	@echo "  fmt                Format the plugin"
	@echo "  vet                Vet the plugin"
	@echo "  check              Includes verify, tidy, fmt, vet"
	@echo "  test-unit          Run unit tests"
	@echo "  test-coverage      Run tests with coverage"
	@echo "  test-all           Includes test-unit, test-coverage"
	@echo "  clean              Clean the plugin"
	@echo "  help               Show this help message"
	@echo ""
	@echo "Variables:"
	@echo "  GOPATH             The GOPATH to use (default: \$$GOPATH)"
	@echo "  PATH               The PATH to use (default: \$$GOPATH/bin:\$$PATH)"
	@echo "  HELM_PLUGINS       The HELM_PLUGINS directory (default: \$$HELM_PLUGINS)"
	@echo "  HELM_PLUGIN_DIR    The HELM_PLUGIN_DIR directory (default: \$$HELM_PLUGIN_DIR)"
	@echo "  BINNAME            The name of the binary to build (default: $(BINNAME))"
	@echo "  PLUGIN_SHORTNAME   The short name of the plugin (default: $(PLUGIN_SHORTNAME))"
	@echo ""

default: help
