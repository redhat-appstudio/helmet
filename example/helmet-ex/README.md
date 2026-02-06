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

# View all targets
make example-help
```

The build process:
1. Creates an uncompressed tarball from the `installer/` directory
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

### Standard Integrations

```bash
# List available integrations
./bin/helmet-ex integration --help

# Get help on a specific integration
./bin/helmet-ex integration acs --help
```

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

### Custom Integrations Configuration

**GitHub integration:** Webhook and homepage URLs are required; callback URL is optional. This example wires a **CustomURLProvider** (`integrations/custom_url_provider.go`) so the GitHub integration gets URLs without requiring flags. Command-line flags (e.g. `--webhook-url`, `--homepage-url`) override the provider when set.

### MCP Server

```bash
# Start MCP server (STDIO mode)
./bin/helmet-ex mcp-server

# Start with custom image
./bin/helmet-ex mcp-server --image quay.io/myorg/myimage:v1.0.0
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

The application embeds the `installer/` directory contents as an uncompressed tarball at build time:

```
installer/
├── config.yaml           # Default configuration schema
├── values.yaml.tpl       # Go template for Helm values
└── charts/               # Helm charts demonstrating topology
    ├── helmet-foundation/
    ├── helmet-infrastructure/
    ├── helmet-integrations/
    ├── helmet-operators/
    ├── helmet-networking/
    ├── helmet-storage/
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
helmet-ex/
├── cmd/
│   └── helmet-ex/
│       └── main.go             # Application entry point
├── installer/
│   ├── charts/                 # Folder with the installer's Helm charts
│   ├── config.yaml             # Default installer configuration
│   ├── embed.go                # Embed directives
│   ├── installer.tar           # Generated tarball (git-ignored)
│   ├── instructions.md         # MCP server guidance
│   └── values.yaml.tpl         # Template file rendered as `values.yaml` and passed to Helm at deployment time
├── integrations/
│   └── custom_url_provider.go  # GitHub URLProvider for this example
├── .gitignore                  # Git ignore rules
└── Makefile                    # Build automation
```

## Troubleshooting

### Error: "cluster configmap not found"

This is expected when running topology or deploy commands without cluster configuration.

**Solution:** Create configuration first:
```bash
./bin/helmet-ex config --create
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

## License

Same as parent Helmet project.
