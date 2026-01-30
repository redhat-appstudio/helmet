# helmet-ex: Helmet Framework Example Application

A comprehensive example application demonstrating all features of the Helmet framework for building Kubernetes installers.

## Overview

The `helmet-ex` application showcases:
- Application context with build-time metadata injection
- Embedded tarball filesystem with overlay support for local development
- Standard integration modules (GitHub, GitLab, Quay, ACS, etc.)
- MCP server with AI assistant instructions
- Configuration management via embedded config.yaml
- Template rendering via embedded values.yaml.tpl
- Helm chart dependency resolution and deployment
- All framework-generated CLI commands

This example uses test fixtures from the `test/` directory, demonstrating a multi-product topology with foundation, infrastructure, operators, storage, networking, integrations, and product layers.

## Quick Start

### Prerequisites

- Go 1.21 or higher
- GNU tar
- GNU make
- Git

### Building

```bash
# Build with default version
make build

# Build with custom version
make build VERSION=v1.0.0

# View all targets
make help
```

The build process:
1. Creates an uncompressed tarball from `../../test/` directory
2. Embeds the tarball and instructions.md into the binary
3. Injects version and commit ID via ldflags

### Running

```bash
# Show help
./bin/helmet-ex --help

# Show version
./bin/helmet-ex --version

# List embedded installer resources
./bin/helmet-ex installer --list

# Extract installer resources
./bin/helmet-ex installer --extract /path/to/directory
```

## Command Reference

### Configuration Management

```bash
# Create initial configuration (requires Kubernetes cluster)
./bin/helmet-ex config --create

# View current configuration
./bin/helmet-ex config --get

# Delete configuration
./bin/helmet-ex config --delete
```

### Topology Inspection

```bash
# View dependency graph
./bin/helmet-ex topology
```

### Deployment

```bash
# Deploy with dry-run
./bin/helmet-ex deploy --dry-run

# Deploy to cluster
./bin/helmet-ex deploy

# Deploy with debug logging
./bin/helmet-ex deploy --log-level=debug
```

### Integration Configuration

```bash
# List available integrations
./bin/helmet-ex integration --help

# Configure GitHub integration
./bin/helmet-ex integration github

# Configure GitLab integration
./bin/helmet-ex integration gitlab

# Configure Quay registry
./bin/helmet-ex integration quay
```

All integrations:
- `acs` - Red Hat Advanced Cluster Security
- `artifactory` - JFrog Artifactory
- `azure` - Azure cloud provider
- `bitbucket` - Bitbucket
- `github` - GitHub
- `gitlab` - GitLab
- `jenkins` - Jenkins CI
- `nexus` - Sonatype Nexus
- `quay` - Quay container registry
- `trusted-artifact-signer` - Trusted Artifact Signer
- `trustification` - Trustification service

### MCP Server

```bash
# Start MCP server (STDIO mode)
./bin/helmet-ex mcp-server

# Start with custom image
./bin/helmet-ex mcp-server --image quay.io/myorg/helmet-ex:v1.0.0
```

The MCP server provides AI assistants with tools for:
- Configuration management (create, get, update, delete)
- Deployment operations
- Topology inspection
- Integration configuration

### Template Rendering

```bash
# Render Helm chart templates
./bin/helmet-ex template [chart-name]
```

## Architecture

### Embedded Tarball Filesystem

The application embeds `test/` directory contents as an uncompressed tarball at build time:

```
test/
├── config.yaml           # Default configuration schema
├── values.yaml.tpl       # Go template for Helm values
└── charts/               # Helm charts demonstrating topology
    ├── helmet-foundation/
    ├── helmet-infrastructure/
    ├── helmet-operators/
    ├── helmet-storage/
    ├── helmet-networking/
    ├── helmet-integrations/
    ├── helmet-product-a/
    ├── helmet-product-b/
    ├── helmet-product-c/
    ├── helmet-product-d/
    └── testing/
```

### Overlay Filesystem

The overlay filesystem allows local development without rebuilding:

```go
// Base layer: embedded tarball
tfs := framework.NewTarFS(installer.InstallerTarball)

// Overlay layer: current working directory
ofs := chartfs.NewOverlayFS(tfs, os.DirFS(cwd))

// Result: local files override embedded files
```

This enables:
1. Extract installer resources: `./bin/helmet-ex installer --extract ./dev`
2. Modify files in `./dev/`
3. Run from `./dev/` directory - changes take effect immediately
4. No binary rebuild required

### Dependency Topology

The example demonstrates a multi-layer product topology:

```
Foundation Layer
└── helmet-foundation (base dependencies)
    ├── Infrastructure Layer
    │   └── helmet-infrastructure
    ├── Operators Layer
    │   └── helmet-operators
    ├── Storage Layer
    │   └── helmet-storage
    ├── Networking Layer
    │   └── helmet-networking
    └── Integrations Layer
        └── helmet-integrations

Product Layer
├── Product A (depends on: foundation, operators, infrastructure)
├── Product B (depends on: storage, networking)
├── Product C (depends on: Product A, storage)
└── Product D (depends on: Product C, integrations)
```

## Development Workflow

### 1. Initial Build

```bash
cd example/helmet-ex
make build
```

### 2. Extract Resources for Local Development

```bash
mkdir dev
./bin/helmet-ex installer --extract dev
cd dev
```

### 3. Modify Configuration

Edit `dev/config.yaml` to customize:
- Product enablement
- Namespace assignments
- Product-specific properties

### 4. Test Changes

```bash
# From dev/ directory, run commands
../bin/helmet-ex topology        # Uses local config.yaml
../bin/helmet-ex deploy --dry-run
```

### 5. Rebuild (when code changes)

```bash
cd ..
make clean build
```

## Testing

### Unit Tests

```bash
# Build and verify
make clean build
./bin/helmet-ex --version | grep -q "v0.0.0-SNAPSHOT" || echo "Version injection failed"
./bin/helmet-ex --version | grep -q "Commit:" || echo "Commit injection failed"
```

### CLI Validation

```bash
# Verify all commands are registered
./bin/helmet-ex --help | grep -E "config|topology|deploy|template|installer|mcp-server|integration"

# Test config command structure
./bin/helmet-ex config --help | grep -E "create|get|delete"

# Test integration listing
./bin/helmet-ex integration --help | grep -E "github|gitlab|quay|acs"

# Test installer extraction
./bin/helmet-ex installer --list
```

### Integration Testing (requires Kubernetes cluster)

```bash
# Prerequisites
kubectl cluster-info

# Create configuration
./bin/helmet-ex config --create

# Verify configuration in cluster
kubectl get configmap -n helmet-ex-system

# Deploy with monitoring
./bin/helmet-ex deploy --log-level=debug

# Check deployment status
helm list --all-namespaces | grep helmet-
```

## Build Variables

Override build-time variables:

```bash
# Custom version
make build VERSION=v1.0.0

# Custom commit ID
make build COMMIT_ID=abc123

# Both
make build VERSION=v2.0.0 COMMIT_ID=def456
```

Injected via ldflags:
- `main.version` - Application version (default: v0.0.0-SNAPSHOT)
- `main.commitID` - Git commit ID (default: git rev-parse --short HEAD)

## Project Structure

```
example/helmet-ex/
├── cmd/
│   └── helmet-ex/
│       └── main.go           # Application entry point
├── installer/
│   ├── installer.go          # Embed directives
│   ├── installer.tar         # Generated tarball (git-ignored)
│   └── instructions.md       # MCP server guidance
├── .gitignore                # Git ignore rules
├── go.mod                    # Go module (uses replace directive)
├── Makefile                  # Build automation
└── README.md                 # This file
```

## Troubleshooting

### Error: "cluster configmap not found"

This is expected when running topology or deploy commands without cluster configuration.

**Solution:** Create configuration first:
```bash
./bin/helmet-ex config --create
```

### Error: "failed to create tarball filesystem"

The tarball must be uncompressed (not gzipped).

**Solution:** Verify Makefile uses `tar -cf` (not `tar -czf`):
```bash
make clean
make installer-tarball
file installer/installer.tar  # Should show "POSIX tar archive"
```

### Error: "go.mod updates needed"

Dependencies need to be synchronized.

**Solution:**
```bash
go mod tidy
make build
```

### MCP Server Not Responding

Ensure STDIO mode is used (default behavior):
```bash
./bin/helmet-ex mcp-server
```

For debugging, check that instructions.md is embedded:
```bash
./bin/helmet-ex installer --list | grep instructions.md
```

## References

- [Helmet Framework Documentation](../../README.md)
- [PLAN.md](../../PLAN.md) - Detailed implementation plan
- [Test Fixtures](../../test/) - Source of embedded resources

## License

Same as parent Helmet project.
