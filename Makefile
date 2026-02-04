APP = helmet

# Primary source code directories.
PKG ?= ./internal/...

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

# Installs golangci-lint.
tool-golangci-lint: GOFLAGS =
tool-golangci-lint:
	which golangci-lint || \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

# Installs GitHub CLI ("gh").
tool-gh: GOFLAGS =
tool-gh:
	which gh || \
		go install github.com/cli/cli/v2/cmd/gh@latest

#
# Test and Lint
#

test: test-unit

# Runs the unit tests.
.PHONY: test-unit
test-unit:
	go test $(GOFLAGS_TEST) $(PKG) $(ARGS)

# Uses golangci-lint to inspect the code base.
.PHONY: lint
lint: tool-golangci-lint
	golangci-lint run ./...

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
# Show help
#
.PHONY: help
help: example-help
	@echo ""
	@echo "Targets:"
	@echo "  build           		- Build the package (default)"
	@echo "  github-release-create	- Create GitHub release"
	@echo "  lint            		- Run linting"
	@echo "  test            		- Run tests"
	@echo "  help            		- Show help"
