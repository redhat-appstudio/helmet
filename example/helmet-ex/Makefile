# Build variables
VERSION ?= v0.0.0-SNAPSHOT
COMMIT_ID ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go build flags
GOFLAGS ?= -v
CGO_ENABLED ?= 0

# ldflags for build-time variable injection
LDFLAGS := -X main.version=$(VERSION) -X main.commitID=$(COMMIT_ID)

# Output binary
BIN_DIR = bin
BIN = $(BIN_DIR)/helmet-ex

.PHONY: all
all: build

# Generate installer tarball from test/ directory
.PHONY: installer-tarball
installer-tarball:
	@echo "Creating installer tarball from test/ directory..."
	tar -cf installer/installer.tar -C ../../test .
	@echo "Created installer/installer.tar"

# Build the application
.PHONY: build
build: installer-tarball
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=$(CGO_ENABLED) go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/helmet-ex

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf $(BIN_DIR)
	rm -f installer/installer.tar

# Run the application
.PHONY: run
run: build
	./$(BIN) $(ARGS)

# Show help
.PHONY: help
help:
	@echo "Targets:"
	@echo "  all              - Build the application (default)"
	@echo "  installer-tarball- Create tarball from test/ directory"
	@echo "  build            - Build the application with version injection"
	@echo "  clean            - Remove build artifacts"
	@echo "  run              - Build and run (use ARGS='...' for arguments)"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION          - Version string (default: v0.0.0-SNAPSHOT)"
	@echo "  COMMIT_ID        - Git commit ID (default: git rev-parse --short HEAD)"
	@echo ""
	@echo "Examples:"
	@echo "  make build VERSION=v1.0.0"
	@echo "  make run ARGS='--help'"
	@echo "  make run ARGS='config --create'"
	@echo "  make run ARGS='topology'"
