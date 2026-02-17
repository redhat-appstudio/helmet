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

## The `instructions.md` File

The optional `instructions.md` provides context to AI assistants when the installer runs as an MCP server. The MCP client (e.g., Claude Desktop, Cursor) receives this content, helping the AI understand available products, workflow phases, and tool usage.

**Important boundary**: `instructions.md` describes *a specific installer's* products and workflow. The `docs/` pages describe *the framework itself*. Do not put framework internals in `instructions.md` or installer-specific content in `docs/`.

See [mcp.md](mcp.md) for MCP server implementation details.

## Cross-References

- [Configuration](configuration.md) — config.yaml schema and values rendering
- [Topology](topology.md) — chart dependency resolution and deployment ordering
- [MCP Server](mcp.md) — Model Context Protocol server implementation
- [Getting Started](getting-started.md) — building your first installer