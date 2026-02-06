APP = helmet

# Primary source code directories.
PKG ?= ./api/... ./framework/... ./internal/...

# Golang general flags for build and testing.
GOFLAGS ?= -v
GOFLAGS_TEST ?= -failfast -v -cover
CGO_ENABLED ?= 0
CGO_LDFLAGS ?= 


# GitHub action current ref name, provided by the action context environment
# variables, and credentials needed to push the release.
GITHUB_REF_NAME ?= ${GITHUB_REF_NAME:-}
GITHUB_TOKEN ?= ${GITHUB_TOKEN:-}

.EXPORT_ALL_VARIABLES:

.default: build

include example/helmet-ex/Makefile

#
# Build
#

# Build the application
.PHONY: build
build:
	go build $(GOFLAGS) ./...

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

# Executes govulncheck via go tool (version from go.mod).
.PHONY: tool-govulncheck
tool-govulncheck:
	@go tool govulncheck -version

#
# Test and Lint
#

test: test-unit

# Runs the unit tests.
.PHONY: test-unit
test-unit:
	go test $(GOFLAGS_TEST) -coverprofile=coverage.out -covermode=atomic $(PKG) $(ARGS)

# Uses golangci-lint to inspect the code base.
.PHONY: lint
lint: build
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

#
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
# Show help
#
.PHONY: help
help: example-help
	@echo ""
	@echo "Targets:"
	@echo "  build                   - Build the package (default)"
	@echo "  github-release-create   - Create GitHub release (requires 'gh' in PATH)"
	@echo "  goreleaser-snapshot     - Build release assets for current platform"
	@echo "  goreleaser-release      - Create full release (CI only)"
	@echo "  lint                    - Run linting (includes gosec)"
	@echo "  security                - Run govulncheck vulnerability scan"
	@echo "  test                    - Run tests"
	@echo "  help                    - Show help"
