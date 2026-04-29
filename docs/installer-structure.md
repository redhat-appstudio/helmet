# Installer Structure

This document describes the installer directory layout, how installers are packaged using `go:embed`, the tarball build process, and the overlay filesystem mechanism that enables development workflows without rebuilding the binary.

This page covers **packaging and filesystem abstraction**. For configuration schema and values rendering, see [configuration.md](configuration.md). For the Model Context Protocol integration, see [mcp.md](mcp.md). For dependency resolution and chart ordering, see [topology.md](topology.md).

## Directory Layout

Every Helmet installer follows this convention-based structure:

```
installer/
├── config.yaml          # Configuration schema (required)
├── values.yaml.tpl      # Go template for Helm values (required)
├── charts/              # Helm charts with framework annotations
│   ├── product-a/
│   │   ├── Chart.yaml
│   │   └── templates/
│   └── product-b/
│       ├── Chart.yaml
│       └── templates/
└── instructions.md      # MCP server context (optional)
```

### Required Files

| File | Purpose | Framework Constant |
|------|---------|-------------------|
| `config.yaml` | Defines the configuration schema for settings and products | `constants.ConfigFilename` |
| `values.yaml.tpl` | Go template rendered into Helm values using configuration data | `constants.ValuesFilename` |
| `charts/` | Helm charts annotated with framework metadata for dependency resolution | Scanned via `ChartFS.GetAllCharts()` |

### Optional Files

| File | Purpose | Framework Constant |
|------|---------|-------------------|
| `instructions.md` | Context and guidance for the MCP server, provided to AI assistants | `constants.InstructionsFilename` |

The framework discovers charts automatically by walking the filesystem and looking for directories containing `Chart.yaml`.

## Embedding the Installer

Helmet uses Go's `embed` package to bundle the installer directory into the compiled binary as a tarball. This eliminates external file dependencies at runtime.

### Embed Pattern

Create an `embed.go` file in your installer directory:

```go
package installer

import _ "embed"

// InstallerTarball contains the embedded installer directory as a tarball.
//
//nolint:typecheck,nolintlint
//go:embed installer.tar
var InstallerTarball []byte

// Instructions contains the MCP server guidance markdown (optional).
//
//go:embed instructions.md
var Instructions string
```

The `InstallerTarball` variable holds the tarball bytes, which are passed to `framework.NewAppFromTarball()` during application initialization.

## Building the Tarball

Package the installer directory into a tarball before building the Go binary:

```bash
tar cpf installer/installer.tar \
    --dereference \
    --exclude="*.go" \
    --exclude="installer.tar" \
    -C installer .
```

On macOS, use GNU tar (`gtar`) instead of the system `tar`.

### Key Flags

| Flag | Purpose |
|------|---------|
| `--dereference` | Follow symlinks and include target files (enables sharing charts across projects) |
| `-C installer` | Change to installer directory before adding files (produces clean relative paths) |
| `--exclude` | Skip Go source files and the tarball itself |

The `--dereference` flag is critical: it allows you to symlink shared charts into the installer directory without duplicating files in version control.

### Building the Binary

After creating the tarball, build the Go binary with version injection:

```bash
COMMIT_ID="$(git rev-parse --short HEAD)"

go build \
    -ldflags "-X main.version=v1.0.0 -X main.commitID=${COMMIT_ID}" \
    -o helmet-ex .
```

The tarball must be regenerated whenever the installer directory contents change. See the [`helmet-ex` build instructions](../example/helmet-ex/README.md#building) for a complete working example.

## TarFS: Converting Tarball to Filesystem

The framework provides `NewTarFS()` to convert embedded tarball bytes into an `fs.FS` interface:

```go
// framework/tarfs.go
func NewTarFS(tarball []byte) (fs.FS, error)
```

This uses `github.com/quay/claircore/pkg/tarfs` internally to create a read-only filesystem backed by the tarball contents. The returned `fs.FS` supports standard operations: `Open()`, `ReadFile()`, `ReadDir()`.

## Overlay Filesystem

The `OverlayFS` layers the embedded tarball with a local filesystem. Files are resolved with **embedded first, local second** priority.

### Implementation

```go
// internal/chartfs/overlay.go
type OverlayFS struct {
    Embedded fs.FS  // first priority
    Local    fs.FS  // fallback
}

func NewOverlayFS(embedded, local fs.FS) *OverlayFS
```

When `Open(name)` is called:
1. Check if the file exists in the **Embedded** filesystem (tarball)
2. If not found, check the **Local** filesystem (current working directory)
3. Return `fs.ErrNotExist` only if both lookups fail

### Resolution Order

| Priority | Source | Description |
|----------|--------|-------------|
| 1 (first) | Embedded tarball | Files bundled into the binary at build time |
| 2 (fallback) | Local filesystem | Files in the current working directory |

### Development Workflow

Because the embedded tarball has first priority, local files in the working directory are only used for files **not present** in the tarball. To iterate on installer content during development, rebuild the tarball after each change:

1. **Edit** files in the `installer/` source directory (config, charts, templates)

2. **Rebuild** the tarball:
   ```bash
   tar cpf installer/installer.tar \
       --dereference \
       --exclude="*.go" \
       --exclude="installer.tar" \
       -C installer .
   ```

3. **Rebuild** the binary and test:
   ```bash
   go build -o helmet-ex . && ./helmet-ex deploy --dry-run
   ```

The local filesystem fallback is useful for files that are intentionally **excluded from the tarball** — for example, a new chart directory that hasn't been packaged yet. It does not provide a general override mechanism for files that already exist in the tarball.

### File Resolution Example

Given an embedded tarball containing `config.yaml`, `values.yaml.tpl`, and `charts/product-a/Chart.yaml`, and a working directory with `config.yaml` and a new `charts/product-b/` directory:

| File | Source | Reason |
|------|--------|--------|
| `config.yaml` | Embedded tarball | Present in tarball (first priority) |
| `values.yaml.tpl` | Embedded tarball | Present in tarball |
| `charts/product-a/Chart.yaml` | Embedded tarball | Present in tarball (first priority) |
| `charts/product-b/Chart.yaml` | Local workspace | Not in tarball, falls back to local |
| `instructions.md` | Embedded tarball | Present in tarball |

## NewAppFromTarball: Standard Constructor

The recommended way to create an installer application:

```go
func NewAppFromTarball(
    appCtx *api.AppContext,
    tarball []byte,
    cwd string,
    opts ...Option,
) (*App, error)
```

| Parameter | Type | Purpose |
|-----------|------|---------|
| `appCtx` | `*api.AppContext` | Application metadata (name, version, namespace, descriptions) |
| `tarball` | `[]byte` | Embedded installer tarball bytes |
| `cwd` | `string` | Current working directory for local filesystem overlay |
| `opts` | `...Option` | Functional options (integrations, MCP image, etc.) |

**Internal flow**:
1. Convert tarball bytes to `fs.FS` via `NewTarFS(tarball)`
2. Create overlay: `chartfs.NewOverlayFS(tfs, os.DirFS(cwd))`
3. Wrap in `ChartFS`: `chartfs.New(ofs)`
4. Pass to `NewApp()` with all options applied

## Build Pipeline

The installer build process has a strict dependency: the tarball must be created before the Go binary is compiled, because `go:embed` requires the tarball file to exist at build time.

### Makefile Targets

| Target | Purpose | Notes |
|---|---|---|
| `installer-tarball` | Package installer directory into `installer.tar` | Must run before `build` |
| `build` | Compile Go binary with version injection | Depends on `installer-tarball` |
| `image` | Build container image | Multi-stage Containerfile |
| `lint` | Run linters (`golangci-lint`) | Code quality checks |
| `security` | Run security scanners (`govulncheck`) | Vulnerability detection |

### Tarball-Before-Build Dependency

The `installer-tarball` target must run before `go build`. The `build` target should declare this dependency explicitly:

```makefile
build: installer-tarball
	go build -ldflags "-X main.version=$(VERSION) -X main.commitID=$(COMMIT_ID)" -o bin/my-app ./cmd
```

If `installer.tar` does not exist when `go build` runs, the `go:embed` directive in `embed.go` fails with a compilation error.

### GNU tar on macOS

On macOS, use GNU tar (`gtar`) instead of the system BSD tar for consistent behavior. BSD tar and GNU tar handle symlinks, permissions, and extended attributes differently. Install via `brew install gnu-tar`.

### Container Image

The Containerfile uses a multi-stage build:

1. **Builder stage**: Compiles the Go binary with build args for `BUILD_VERSION` and `COMMIT_ID` injected via ldflags
2. **Runtime stage**: Copies the compiled binary and required tools (`kubectl`, `oc`, `jq`) into a minimal base image

Build args enable version tracking in the final image without carrying the full build toolchain.

## Chart Provenance

A Helmet installer is a **composition of Helm charts**. Most installers combine external charts (for products with existing Helm charts) and authored charts (for infrastructure and integration glue). The framework is chart-agnostic — it loads whatever is in the `charts/` directory.

### Bringing External Charts

The primary workflow for building an installer:

1. **Copy** the chart directory into `charts/<name>/`
2. **Add Helmet annotations** to `Chart.yaml` — `product-name`, `depends-on`, and integration annotations as needed
3. **Set the root key** in `values.yaml` — conventionally derived from the chart name, must be unique across all charts
4. **Add `__OVERWRITE_ME__` placeholders** in `values.yaml` for values that should be dynamically rendered by `values.yaml.tpl`
5. **Wire up** a corresponding section in `values.yaml.tpl` and a product entry in `config.yaml` (if the chart maps to a product)

**Helm `dependencies:` handling**: Upstream charts may use `dependencies:` in `Chart.yaml` for sub-chart resolution. The framework ignores this field for topology resolution — it reads `depends-on` annotations instead. However, Helm itself still processes `dependencies:` during install (pulling and rendering sub-charts with values merging). Only remove `dependencies:` entries when you intend to surface the sub-chart as a separate Helmet-managed topology node; otherwise, keep the upstream `dependencies:` so Helm continues to handle sub-chart rendering and values merging. Express inter-chart relationships via `depends-on` annotations regardless.

### Authoring New Charts

A secondary workflow for installer-specific charts that don't exist upstream:

- **Namespace charts**: Create OpenShift projects, set labels and annotations
- **Operator subscription charts**: Deploy OLM Subscriptions for operators the installer manages
- **Integration aggregation charts**: Collect and validate integration Secrets
- **Companion and test charts**: Validation or post-deploy checks for a product

These charts are typically thin — a few templates, minimal values — and are authored from scratch. See [topology.md](topology.md#infrastructure-charts--derived-products) for infrastructure chart patterns.

### Composition Model

The filesystem is the composition mechanism. Copy charts into `charts/`, add annotations, wire up `config.yaml` and `values.yaml.tpl`. No catalog or registry feature is needed — the directory structure is the source of truth.

A realistic installer combines both paths. Plan the composition by considering:

- **Inventory**: What products does the installer deploy? Which have existing Helm charts?
- **Gap analysis**: What infrastructure charts are needed (namespaces, operators, shared databases)?
- **Dependency design**: How do the charts relate? What deployment order is required?
- **Integration mapping**: Which charts provide or require integrations?

## The `instructions.md` File

The optional `instructions.md` provides context to AI assistants when the installer runs as an MCP server. The MCP client (e.g., Claude Desktop, Cursor) receives this content, helping the AI understand available products, workflow phases, and tool usage.

**Important boundary**: `instructions.md` describes *a specific installer's* products and workflow. The `docs/` pages describe *the framework itself*. Do not put framework internals in `instructions.md` or installer-specific content in `docs/`.

See [mcp.md](mcp.md) for MCP server implementation details.

## Cross-References

- [Configuration](configuration.md) — config.yaml schema, values rendering, [the triad](configuration.md#the-triad)
- [Topology](topology.md) — chart dependency resolution, deployment ordering, [infrastructure charts](topology.md#infrastructure-charts--derived-products)
- [Templating](templating.md) — values.yaml.tpl syntax and [production patterns](templating.md#production-patterns)
- [MCP Server](mcp.md) — Model Context Protocol server implementation
- [Getting Started](getting-started.md) — building your first installer