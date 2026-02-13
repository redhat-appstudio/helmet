# Primary source code directories.
PKG ?= ./api/... ./framework/... ./internal/...
# E2E test package.
PKG_E2E ?= ./test/e2e
PKG_E2E_CLI := $(PKG_E2E)/cli/...
PKG_E2E_MCP := $(PKG_E2E)/mcp/...

# Golang general flags for build and testing.
GOFLAGS ?= -v
GOFLAGS_TEST ?= -failfast -v -cover
CGO_ENABLED ?= 0
CGO_LDFLAGS ?=

# Build variables.
VERSION ?= v0.0.0-SNAPSHOT
COMMIT_ID ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Example application paths.
EXAMPLE_APP ?= helmet-ex
EXAMPLE_DIR ?= example/$(EXAMPLE_APP)
EXAMPLE_BIN ?= $(EXAMPLE_DIR)/$(EXAMPLE_APP)

# Installer tarball using "find" to list the included paths.
INSTALLER_DIR = $(EXAMPLE_DIR)/installer
INSTALLER_TARBALL = $(INSTALLER_DIR)/installer.tar
INSTALLER_TARBALL_DATA = $(shell find -L $(INSTALLER_DIR) -type f \
    ! -path "$(INSTALLER_TARBALL)" \
    ! -name embed.go \
    | sort \
)

# Determine the appropriate tar command based on the operating system.
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
    TAR := gtar
else
    TAR := tar
endif

# GitHub action current ref name, provided by the action context environment
# variables, and credentials needed to push the release.
GITHUB_REF_NAME ?= ${GITHUB_REF_NAME:-}
GITHUB_TOKEN ?= ${GITHUB_TOKEN:-}

# Container image configuration, either podman or docker.
CONTAINER_CLI ?= $(shell command -v podman >/dev/null 2>&1 && echo podman || echo docker)
IMAGE_REPOSITORY ?= localhost:5000
IMAGE_NAMESPACE ?= helmet
IMAGE_TAG ?= $(COMMIT_ID)
IMAGE ?= $(IMAGE_REPOSITORY)/$(IMAGE_NAMESPACE)/$(EXAMPLE_APP):$(IMAGE_TAG)

.EXPORT_ALL_VARIABLES:

.DEFAULT_GOAL := build

#
# Build
#

# Build the example application.
.PHONY: build
build: $(EXAMPLE_BIN)
$(EXAMPLE_BIN): installer-tarball
$(EXAMPLE_BIN):
	go build $(GOFLAGS) \
		-ldflags "-X main.version=$(VERSION) -X main.commitID=$(COMMIT_ID)" \
		-o $(EXAMPLE_BIN) ./$(EXAMPLE_DIR)

#
# Example Application
#

# Removes build artifacts.
.PHONY: clean
clean:
	rm -fv "$(EXAMPLE_BIN)" "$(INSTALLER_TARBALL)" || true

# Generates the installer tarball.
.PHONY: installer-tarball
installer-tarball: $(INSTALLER_TARBALL)
$(INSTALLER_TARBALL): $(INSTALLER_TARBALL_DATA)
	@echo "# Generating '$(INSTALLER_TARBALL)'"
	@test -f "$(INSTALLER_TARBALL)" && rm -f "$(INSTALLER_TARBALL)" || true
	$(TAR) \
		--create \
		--dereference \
		--directory "$(INSTALLER_DIR)" \
		--file "$(INSTALLER_TARBALL)" \
		--preserve-permissions \
		$(shell echo "$(INSTALLER_TARBALL_DATA)" \
			| sed "s:$(INSTALLER_DIR)/:./:g")

# Builds and runs the example application.
.PHONY: run
run: build
	$(EXAMPLE_BIN) $(ARGS)

# Builds the container image.
.PHONY: image
image: installer-tarball
	@echo "Building container image: $(IMAGE)"
	$(CONTAINER_CLI) build -t $(IMAGE) -f $(EXAMPLE_DIR)/Dockerfile \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT_ID=$(COMMIT_ID) \
		.
	@echo "Container image built successfully: $(IMAGE)"

# Pushes the container image to the configured registry.
.PHONY: image-push
image-push:
	$(CONTAINER_CLI) push --tls-verify=false $(IMAGE) $(ARGS)

#
# Tools
#

# Executes golangci-lint via go tool (version from go.mod).
.PHONY: tool-golangci-lint
tool-golangci-lint:
	@go tool golangci-lint --version

# Requires GitHub CLI ("gh") to be available in PATH. By default it's installed in
# GitHub Actions workflows, common workstation package managers.
.PHONY: tool-gh
tool-gh:
	@which gh >/dev/null 2>&1 || \
		{ echo "Error: 'gh' not found in PATH."; exit 1; }
	@gh --version

# Executes goreleaser via go tool (version from go.mod).
.PHONY: tool-goreleaser
tool-goreleaser:
	@go tool goreleaser --version

# Executes ginkgo via go tool (version from go.mod).
.PHONY: tool-ginkgo
tool-ginkgo:
	@go tool ginkgo version

# Executes govulncheck via go tool (version from go.mod).
.PHONY: tool-govulncheck
tool-govulncheck:
	@go tool govulncheck -version

#
# Test and Lint
#

# Runs the unit tests.
.PHONY: test-unit
test-unit:
	go test $(GOFLAGS_TEST) \
		-coverprofile=coverage.out \
		-covermode=atomic \
		$(PKG) $(PKG_E2E) \
		$(ARGS)

# Runs the E2E CLI tests (requires KinD cluster).
.PHONY: test-e2e-cli
test-e2e-cli: build
	go tool ginkgo -v --fail-fast $(PKG_E2E_CLI) $(ARGS)

# Runs the E2E MCP tests (requires KinD cluster + image pushed).
.PHONY: test-e2e-mcp
test-e2e-mcp: build
	go tool ginkgo -v --fail-fast $(PKG_E2E_MCP) $(ARGS)

# Uses golangci-lint to inspect the code base.
.PHONY: lint
lint: installer-tarball
	go tool golangci-lint run ./...

#
# Security
#

# Scans for known vulnerabilities in dependencies.
.PHONY: govulncheck
govulncheck:
	go tool govulncheck ./...

# Runs all security checks.
.PHONY: security
security: govulncheck

#
# GitHub Release
#

# Asserts the required environment variables are set and the target release
# version starts with "v".
github-preflight:
ifeq ($(strip $(GITHUB_REF_NAME)),)
	$(error variable GITHUB_REF_NAME is not set)
endif
ifeq ($(shell echo ${GITHUB_REF_NAME} |grep -v -E '^v'),)
	@echo GITHUB_REF_NAME=\"${GITHUB_REF_NAME}\"
else
	$(error invalid GITHUB_REF_NAME, it must start with "v")
endif
ifeq ($(strip $(GITHUB_TOKEN)),)
	$(error variable GITHUB_TOKEN is not set)
endif

# Creates a new GitHub release with GITHUB_REF_NAME.
.PHONY: github-release-create
github-release-create: tool-gh
	gh release view $(GITHUB_REF_NAME) >/dev/null 2>&1 || \
		gh release create --generate-notes $(GITHUB_REF_NAME)

# Releases the GITHUB_REF_NAME.
github-release: \
	github-preflight \
	github-release-create

# Goreleaser
#

# Builds release assets for current platform (snapshot mode).
.PHONY: goreleaser-snapshot
goreleaser-snapshot:
	go tool goreleaser build --snapshot --clean $(ARGS)

# Builds release assets for all platforms (snapshot mode).
.PHONY: goreleaser-snapshot-all
goreleaser-snapshot-all:
	go tool goreleaser build --snapshot --clean

# Creates a full release (CI only).
.PHONY: goreleaser-release
goreleaser-release: github-preflight
	go tool goreleaser release --clean

#
# KinD Cluster Management
#

KIND_CLUSTER_NAME ?= helmet-test
KIND_CONFIG ?= test/kind-cluster.yaml
KIND_REGISTRY_NAME ?= kind-registry
KIND_REGISTRY_PORT ?= 5000

# Creates a KinD cluster for testing with local registry.
.PHONY: kind-up
kind-up:
	@echo "Creating registry container '$(KIND_REGISTRY_NAME)'..."
	@docker run -d --restart=always \
		-p 127.0.0.1:$(KIND_REGISTRY_PORT):5000 \
		--network bridge \
		--name $(KIND_REGISTRY_NAME) \
		registry:2 2>/dev/null || \
		docker start $(KIND_REGISTRY_NAME) 2>/dev/null || true
	@echo "Creating KinD cluster '$(KIND_CLUSTER_NAME)'..."
	kind create cluster --name $(KIND_CLUSTER_NAME) --config $(KIND_CONFIG) --wait 60s
	@echo "Connecting registry to cluster network..."
	@docker network connect kind $(KIND_REGISTRY_NAME) 2>/dev/null || true
	@echo "KinD cluster '$(KIND_CLUSTER_NAME)' is ready!"
	@echo "Local registry available at: localhost:$(KIND_REGISTRY_PORT)"

# Deletes the KinD cluster and local registry.
.PHONY: kind-down
kind-down:
	@echo "Deleting KinD cluster '$(KIND_CLUSTER_NAME)'..."
	@kind delete cluster --name $(KIND_CLUSTER_NAME) 2>/dev/null || true
	@echo "Removing registry container '$(KIND_REGISTRY_NAME)'..."
	@docker rm -f $(KIND_REGISTRY_NAME) 2>/dev/null || true

# Shows KinD cluster and registry status.
.PHONY: kind-status
kind-status:
	@kind get clusters 2>/dev/null | grep -q "$(KIND_CLUSTER_NAME)" && \
		echo "KinD cluster '$(KIND_CLUSTER_NAME)' is running" || \
		echo "KinD cluster '$(KIND_CLUSTER_NAME)' is not running"
	@if [ -n "$$(docker ps -q -f name=$(KIND_REGISTRY_NAME))" ]; then \
		echo "Registry '$(KIND_REGISTRY_NAME)' is running at localhost:$(KIND_REGISTRY_PORT)"; \
	else \
		echo "Registry '$(KIND_REGISTRY_NAME)' is not running"; \
	fi

#
# Show help
#

.PHONY: help
help:
	@echo "Targets:"
	@echo "  build                    - Build library and example binary (default)"
	@echo "  clean                    - Remove build artifacts"
	@echo "  installer-tarball        - Generate installer tarball"
	@echo "  run                      - Build and run example (use ARGS='...')"
	@echo "  image                    - Build container image (depends on installer-tarball)"
	@echo "  image-push               - Push container image"
	@echo "  test-unit                - Run unit tests"
	@echo "  test-e2e-cli             - Run E2E CLI tests (requires KinD)"
	@echo "  test-e2e-mcp             - Run E2E MCP tests (requires KinD + image)"
	@echo "  lint                     - Run linting"
	@echo "  security                 - Run govulncheck vulnerability scan"
	@echo "  github-release-create    - Create GitHub release (requires 'gh')"
	@echo "  goreleaser-snapshot      - Build release assets for current platform"
	@echo "  goreleaser-release       - Create full release (CI only)"
	@echo "  kind-up                  - Create KinD cluster with local registry"
	@echo "  kind-down                - Delete KinD cluster and registry"
	@echo "  kind-status              - Show KinD cluster and registry status"
	@echo "  help                     - Show this help"
