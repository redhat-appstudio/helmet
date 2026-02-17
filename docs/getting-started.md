# Getting Started

This guide walks you through creating a Kubernetes installer using the Helmet framework. The [`helmet-ex`](../example/helmet-ex/) example application is the canonical reference — this guide explains the patterns it uses so you can replicate them in your own project.

By the end of this guide, you will understand the project structure, how to embed Helm charts, build the binary, and run the generated CLI.

**What this guide covers:**
- Project structure and required files
- Embedding charts as a tarball
- Building the installer binary
- Running the generated CLI commands

**What this guide does NOT cover:**
- Integration modules — see [integrations.md](integrations.md)
- MCP server — see [mcp.md](mcp.md)
- Advanced topology patterns — see [topology.md](topology.md)
- Template engine details — see [templating.md](templating.md)

## Prerequisites

- **Go 1.25 or later**
- **GNU tar** (macOS users: `brew install gnu-tar`)
- **Helm 3** basic knowledge (charts, values, dependencies)
- **Kubernetes cluster** access for deployment testing (optional for initial build)

## Project Structure

Every Helmet installer follows this layout. The [`helmet-ex`](../example/helmet-ex/) example demonstrates it:

```text
helmet-ex/
├── main.go                         # Application entry point
├── custom_url_provider.go          # Optional: custom integration URLs
└── installer/
    ├── embed.go                    # go:embed directives
    ├── config.yaml                 # Configuration schema (required)
    ├── values.yaml.tpl             # Go template for Helm values (required)
    ├── instructions.md             # MCP server context (optional)
    ├── installer.tar               # Generated tarball (git-ignored)
    └── charts/                     # Helm charts with framework annotations
```

See [installer-structure.md](installer-structure.md) for full details on each file's role.

## Key Files

### Configuration: `installer/config.yaml`

Defines settings and products. From [`helmet-ex/installer/config.yaml`](../example/helmet-ex/installer/config.yaml):

```yaml
---
tssc:
  settings:
    crc: false
    ci:
      debug: false
  products:
    - name: Product A
      enabled: true
      namespace: helmet-product-a
```

The top-level key (`tssc`) must match across your configuration. See [configuration.md](configuration.md) for the full schema.

### Values Template: `installer/values.yaml.tpl`

A Go template that renders Helm values from configuration. See [templating.md](templating.md) for available functions and context variables.

### Embed Directives: `installer/embed.go`

Embeds the installer tarball into the binary. From [`helmet-ex/installer/embed.go`](../example/helmet-ex/installer/embed.go):

```go
package installer

import _ "embed"

//go:embed installer.tar
var InstallerTarball []byte

//go:embed instructions.md
var Instructions string
```

### Application Entry Point: `main.go`

The [`helmet-ex/main.go`](../example/helmet-ex/main.go) demonstrates the standard pattern:

```go
appCtx := api.NewAppContext(
    "helmet-ex",
    api.WithVersion(version),
    api.WithCommitID(commitID),
    api.WithNamespace("helmet-ex-system"),
    api.WithShortDescription("Helmet Framework Example Application"),
)

app, err := framework.NewAppFromTarball(
    appCtx,
    installer.InstallerTarball,
    cwd,
    framework.WithIntegrations(appIntegrations...),
    framework.WithMCPImage(mcpImage),
)
if err != nil {
    fmt.Fprintf(os.Stderr, "Error: %v\n", err)
    os.Exit(1)
}

if err := app.Run(); err != nil {
    fmt.Fprintf(os.Stderr, "Error: %v\n", err)
    os.Exit(1)
}
```

**Key points:**
- `api.NewAppContext()` creates application metadata with functional options
- `framework.NewAppFromTarball()` constructs the app from the embedded tarball
- The `cwd` parameter enables the [overlay filesystem](installer-structure.md#overlay-filesystem) for development
- `framework.WithMCPImage()` sets the container image for [MCP Job-based deployments](mcp.md#container-image-for-job-based-deployment)

## Building

### 1. Create the Installer Tarball

Package the installer directory into a tarball. Use `--dereference` to follow symlinks:

```bash
tar cpf installer/installer.tar \
    --dereference \
    --exclude="*.go" \
    --exclude="installer.tar" \
    -C installer .
```

On macOS, use GNU tar (`gtar`) instead of the system `tar`.

### 2. Build the Binary

Build with ldflags to inject version metadata:

```bash
COMMIT_ID="$(git rev-parse --short HEAD)"

go build \
    -ldflags "-X main.version=v1.0.0 -X main.commitID=${COMMIT_ID}" \
    -o helmet-ex .
```

## Running the Generated CLI

The built binary has a full CLI with framework-generated commands:

```bash
helmet-ex --help
```

### Available Commands

| Command | Purpose |
|---------|---------|
| `config` | Configuration management (create, view, update, delete) |
| `deploy` | Deploy all enabled products to the cluster |
| `topology` | Inspect the dependency graph and deployment order |
| `integration` | Configure external service integrations |
| `mcp-server` | Start the MCP server for AI assistant integration |
| `template` | Render values.yaml.tpl (debug) |
| `installer` | Extract or list embedded installer resources |

See [cli-reference.md](cli-reference.md) for full command documentation.

### First Deployment

```bash
# Create configuration in the cluster
helmet-ex config --create

# Review the topology
helmet-ex topology

# Configure necessary integrations
helmet-ex integration --help

# Dry-run the deployment
helmet-ex deploy --dry-run

# Deploy to the cluster
helmet-ex deploy
```

The framework will parse configuration, resolve chart dependencies, render Helm values from the template, deploy charts in topological order, and wait for resources to become ready.

## Next Steps

- [Architecture](architecture.md) — framework design and extension points
- [Installer Structure](installer-structure.md) — tarball layout, overlay filesystem
- [Configuration](configuration.md) — config.yaml schema, ConfigMap persistence
- [Integrations](integrations.md) — add GitHub, GitLab, Quay, and custom integrations
- [Template Engine](templating.md) — values.yaml.tpl syntax, custom functions
- [MCP Server](mcp.md) — AI-assisted workflows
- [Topology](topology.md) — dependency resolution and chart ordering
- [Example Charts](example-charts.md) — reference charts demonstrating framework patterns
- [CLI Reference](cli-reference.md) — generated commands, flags, custom commands
