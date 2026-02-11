# Primary source code directories.
PKG ?= ./api/... ./framework/... ./internal/...
# E2E test package.
PKG_E2E ?= ./test/e2e
PKG_E2E_CLI := $(PKG_E2E)/cli/...

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

# Creates a KinD cluster for testing
.PHONY: kind-up
kind-up:
	@echo "Creating KinD cluster '$(KIND_CLUSTER_NAME)'..."
	kind create cluster --name $(KIND_CLUSTER_NAME) --config $(KIND_CONFIG) --wait 60s
	@echo "KinD cluster '$(KIND_CLUSTER_NAME)' is ready!"
	@echo "Run: kubectl cluster-info --context kind-$(KIND_CLUSTER_NAME)"

# Deletes the KinD cluster
.PHONY: kind-down
kind-down:
	@echo "Deleting KinD cluster '$(KIND_CLUSTER_NAME)'..."
	kind delete cluster --name $(KIND_CLUSTER_NAME)
	@echo "KinD cluster '$(KIND_CLUSTER_NAME)' deleted."

# Shows KinD cluster status
.PHONY: kind-status
kind-status:
	@kind get clusters 2>/dev/null | grep -q "$(KIND_CLUSTER_NAME)" && \
		echo "KinD cluster '$(KIND_CLUSTER_NAME)' is running" || \
		echo "KinD cluster '$(KIND_CLUSTER_NAME)' is not running"

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
	@echo "  test                     - Run tests"
	@echo "  test-e2e-cli             - Run E2E CLI tests (requires KinD)"
	@echo "  lint                     - Run linting"
	@echo "  security                 - Run govulncheck vulnerability scan"
	@echo "  github-release-create    - Create GitHub release (requires 'gh')"
	@echo "  goreleaser-snapshot      - Build release assets for current platform"
	@echo "  goreleaser-release       - Create full release (CI only)"
	@echo "  kind-up                  - Create KinD cluster"
	@echo "  kind-down                - Delete the KinD cluster"
	@echo "  kind-status              - Show KinD cluster status"
	@echo "  help                     - Show this help"
