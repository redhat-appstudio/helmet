APP = helmet-ex

# Build variables
VERSION ?= v0.0.0-SNAPSHOT
COMMIT_ID ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go build flags
GOFLAGS ?= -v
CGO_ENABLED ?= 0

# Output directories and files.
MAKEFILE_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
BIN_DIR := $(MAKEFILE_DIR)bin
BIN = $(BIN_DIR)/$(APP)
CMD_DIR := $(MAKEFILE_DIR)cmd
INSTALLER_DIR := $(MAKEFILE_DIR)installer
INSTALLER_TARBALL := $(INSTALLER_DIR)/installer.tar
# Data to include in the tarball.
INSTALLER_TARBALL_DATA ?= $(shell find -L $(INSTALLER_DIR) -type f \
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

.DEFAULT_GOAL := build

# Build the application
.PHONY: build
build: $(BIN)# Builds the application executable with installer resources embedded.
$(BIN): installer-tarball
$(BIN):
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=$(CGO_ENABLED) go build $(GOFLAGS) -ldflags "-X main.version=$(VERSION) -X main.commitID=$(COMMIT_ID)" -o $(BIN) $(CMD_DIR)/$(APP)

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf $(BIN_DIR)
	rm -f $(INSTALLER_TARBALL)

# Generate installer tarball from test/ directory
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
		$(shell echo "$(INSTALLER_TARBALL_DATA)" | sed "s:$(INSTALLER_DIR)/:./:g")

# Run the application
.PHONY: run
run: build
	$(BIN) $(ARGS)

# Show help for example application
.PHONY: example-help
example-help:
	@echo "Example Targets:"
	@echo "  build           		- Build the application (default)"
	@echo "  clean           		- Remove build artifacts"
	@echo "  installer-tarball		- Create tarball from test/ directory"
	@echo "  run             		- Build and run (use ARGS='...' for arguments)"
	@echo "  example-help    		- Show example help"
