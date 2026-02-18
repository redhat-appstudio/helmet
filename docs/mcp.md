# Model Context Protocol (MCP) Server

The Helmet framework includes built-in MCP server support, enabling AI assistants to interact programmatically with your Kubernetes installer. The MCP server exposes tools over STDIO that guide users through configuration, integration setup, and deployment workflows.

This page covers the MCP server architecture, container image requirements for Job-based deployments, built-in tools, the `instructions.md` format, custom tool registration, and security considerations. For integration modules, see [integrations.md](integrations.md). For general framework setup, see [getting-started.md](getting-started.md).

## Usage

Configure your MCP client to run your installer binary with the `mcp-server` subcommand:

```sh
helmet-ex mcp-server
```

The MCP server requires a [container image](#container-image-for-job-based-deployment) for Job-based deployments, configured via `WithMCPImage()` at framework initialization or overridden with `--image` at runtime.

### Client Configuration

Using [`helmet-ex`](../example/helmet-ex/) as an example:

**Cursor** (`.cursor/mcp.json`):
```json
{
  "mcpServers": {
    "helmet-ex": {
      "command": "helmet-ex",
      "args": ["mcp-server"]
    }
  }
}
```

**Claude Desktop**:
```json
{
  "mcpServers": {
    "helmet-ex": {
      "command": "/path/to/helmet-ex",
      "args": ["mcp-server"]
    }
  }
}
```

## How It Works

- **STDIO Transport**: JSON-RPC over stdin/stdout following the MCP specification
- **Instructions**: Reads `instructions.md` from the installer filesystem to provide AI context
- **Tool Naming**: Tools are prefixed with the app name (e.g., `helmet-ex_config_get`)
- **Long Operations**: Deployments are delegated to Kubernetes Jobs to keep the server responsive

### Workflow Phases

The MCP server guides users through a phased workflow:

| Phase | Description | Key Tools |
|-------|-------------|-----------|
| `AWAITING_CONFIGURATION` | No config in cluster | `config_get`, `config_init` |
| `AWAITING_INTEGRATIONS` | Config exists, integrations missing | `integration_list`, `integration_scaffold` |
| `READY_TO_DEPLOY` | Config and integrations ready | `deploy` |
| `DEPLOYING` | Job is active | `status` (poll) |
| `COMPLETED` | Deployment succeeded | `notes` |

## Container Image for Job-Based Deployment

The MCP server delegates deployments to Kubernetes Jobs. The container image is the consumer's own application — the same Go binary built with the Helmet framework, packaged into a container image so it can execute asynchronously inside the cluster.

MCP clients expect quick responses over STDIO. A deployment that takes 10-20 minutes would block the server and cause timeouts. Instead, the `deploy` tool creates a Kubernetes Job that runs the installer binary in-cluster, then returns immediately. The AI assistant polls `status` to monitor progress.

### Job Spec

When `deploy` is invoked, the MCP server — authenticated via the user's kubeconfig — creates the following resources:

1. **`ServiceAccount`** named `{appName}` in the installer namespace (via [server-side apply](https://kubernetes.io/docs/reference/using-api/server-side-apply/))
2. **`ClusterRoleBinding`** named `{appName}`, binding the `ServiceAccount` to the `cluster-admin` `ClusterRole` (via server-side apply)
3. **`Job`** named `{appName}-deploy-job` (via `Create` — only one Job is allowed; use `force: true` to replace an existing one) with:
   - Container image: the consumer's installer application image (from `WithMCPImage()` or `--image`)
   - Args: `["deploy"]` (with optional `--debug`, `--dry-run`)
   - Env: `KUBECONFIG=""` (forces [in-cluster authentication](https://kubernetes.io/docs/reference/access-authn-authz/authentication/#service-account-tokens))
   - RestartPolicy: `Never`, BackoffLimit: `0`
   - Labels: `type=installer-job.helmet.redhat-appstudio.github.com`

#### Authentication Delegation

The user running the MCP server authenticates to the cluster via their kubeconfig (user certificate, bearer token, OIDC, or any other method configured in `~/.kube/config` or `--kube-config`). The framework uses these credentials to create the `ServiceAccount` and `ClusterRoleBinding` — this succeeds only if the user already has RBAC permission to create these resources.

When the Job's pod starts, Kubernetes [automatically mounts](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/) the `ServiceAccount`'s token into the pod at `/var/run/secrets/kubernetes.io/serviceaccount/`. Setting `KUBECONFIG=""` forces the Go client libraries to use this mounted token for in-cluster authentication instead of looking for a kubeconfig file. The pod then authenticates to the API server as the `ServiceAccount`, which has `cluster-admin` via the `ClusterRoleBinding`.

This is a standard Kubernetes delegation pattern: the user's own credentials create the RBAC resources, and the Job operates with the same `cluster-admin` access level the user already has. See [Security Model](#security-model) for implications.

### Container Image Setup

The consumer application must be packaged as a container image pullable from within the cluster. Use the project's build tooling or a standard Dockerfile:

```dockerfile
FROM registry.access.redhat.com/ubi9/ubi-minimal:latest
COPY helmet-ex /usr/local/bin/helmet-ex
ENTRYPOINT ["helmet-ex"]
```

Build and push to a registry accessible by the cluster:

```bash
make image IMAGE_REPOSITORY="quay.io/redhat-appstudio"
make image-push
```

`WithMCPImage()` is **required** at framework initialization — the framework returns an error if no image is configured. This value becomes the default for the optional `--image` flag, which overrides it at runtime.

```go
// At framework initialization (required) — from helmet-ex/main.go
framework.WithMCPImage("quay.io/redhat-appstudio/helmet-ex:latest")
```

```sh
# At runtime (optional, overrides the compile-time default)
helmet-ex mcp-server --image="quay.io/redhat-appstudio/helmet-ex:v1.3.0-rc1"
```

**Version alignment**: The container image should match the exact version of the CLI binary. Both the local CLI and the in-cluster Job run the same `deploy` command with the same embedded installer tarball. A version mismatch can cause configuration drift or unexpected behavior. The [`helmet-ex` example](../example/helmet-ex/main.go) demonstrates this via `buildMCPImage()`, which tags the image with the build-time `commitID`. See [architecture.md](architecture.md#extension-points) for the full list of functional options.

### Job State Machine

```text
NotFound (0) → Deploying (1) → Done (3)
                            ↘ Failed (2)
```

- **NotFound**: No Job exists (or dry-run Job exists, which is ignored)
- **Deploying**: Job active (`.status.active > 0`)
- **Failed**: Job failed (`.status.failed > 0`)
- **Done**: Job succeeded (`.status.succeeded > 0`)

### Force Parameter

If a Job already exists (e.g., from a failed deployment), use `force: true` to delete the existing Job and create a new one.

### Following Logs

```sh
oc --namespace=<namespace> logs --follow \
    --selector="type=installer-job.helmet.redhat-appstudio.github.com"
```

## Built-in MCP Tools

All tools are prefixed with `<app-name>_` (e.g., `helmet-ex_config_get`).

### Configuration

| Tool | Arguments | Description |
|------|-----------|-------------|
| `config_get` | None | Returns current or default configuration |
| `config_init` | `namespace` (string) | Initializes default configuration in cluster |
| `config_settings` | `key` (string), `value` (any) | Updates global settings |
| `config_product_enabled` | `name` (string), `enabled` (bool) | Enables/disables a product |
| `config_product_namespace` | `name` (string), `namespace` (string) | Changes product namespace |
| `config_product_properties` | `name` (string), `properties` (object) | Updates product properties |

### Integrations

| Tool | Arguments | Description |
|------|-----------|-------------|
| `integration_list` | None | Lists available integrations |
| `integration_scaffold` | `names` (array of strings) | Generates CLI commands with `OVERWRITE_ME` placeholders |
| `integration_status` | `names` (array of strings) | Checks if integrations are configured |

**Security**: The MCP server never accepts credentials as input. `integration_scaffold` generates command templates for users to execute manually.

### Deployment

| Tool | Arguments | Description |
|------|-----------|-------------|
| `deploy` | `dry_run` (bool, default true), `force` (bool), `debug` (bool) | Creates deployment Job |
| `status` | None | Reports current phase and suggested next action |

### Topology and Notes

| Tool | Arguments | Description |
|------|-----------|-------------|
| `topology` | None | Returns dependency topology table |
| `notes` | `name` (string) | Returns Helm chart NOTES.txt for a deployed product |

## instructions.md Format

The `instructions.md` file provides system-level context to the AI assistant. Place it in your installer's embedded filesystem.

### Purpose

- Describe your products and their purpose
- Explain integration requirements
- Provide workflow guidance for each phase
- Document product-specific quirks

### Structure

```markdown
# {Product Name} Installer

## Overview
Brief description of what this installer deploys.

## Workflow Phases

### Phase 1: Configuration
How to use config_init and config_settings.

### Phase 2: Integrations
Required vs. optional integrations. CEL expressions.

### Phase 3: Deployment
Dry-run vs. production. Expected duration.

### Phase 4: Verification
How to verify success. Common issues.
```

**Content guidelines**: Concise (AI assistants have token limits), actionable (clear next steps per phase), role-aware (address user as platform engineer).

**Important boundary**: `instructions.md` describes *a specific installer's* products and workflow. The `docs/` pages describe *the framework itself*.

## Custom Tool Registration

Extend built-in tools by implementing `mcptools.Interface`:

```go
type MyTools struct {
    appName string
}

func (m *MyTools) Init(s *server.MCPServer) {
    // Register tools with the MCP server
}

func buildCustomTools(ctx mcptools.MCPToolsContext) ([]mcptools.Interface, error) {
    return []mcptools.Interface{
        &MyTools{appName: ctx.AppContext.IdentifierName()},
    }, nil
}
```

Register when creating the app:

```go
app, _ := framework.NewAppFromTarball(
    appCtx, installerTarball, cwd,
    framework.WithMCPToolsBuilder(buildCustomTools),
)
```

The `MCPToolsContext` provides access to `AppContext`, `Flags`, `IntegrationManager`, `Image`, and `RunContext`.

## Security Model

### Credential Boundaries

- **STDIO isolation**: The MCP server runs as a local process communicating over stdin/stdout. It does not expose a network listener
- **No credential inputs**: Integration tools (`integration_scaffold`) generate command templates with `OVERWRITE_ME` placeholders. The MCP server never accepts credentials as tool arguments (see [integrations.md](integrations.md#overwrite_me-placeholders))
- **User's kubeconfig**: All cluster operations (ConfigMap reads, Secret checks, Job creation) authenticate using the user's kubeconfig. The MCP server operates with the same Kubernetes identity and permissions as the user running it

### Job RBAC

The deployment Job delegates the user's cluster access to an in-cluster `ServiceAccount`:

| Resource | Name | Purpose |
|----------|------|---------|
| `ServiceAccount` | `{appName}` | Pod identity for the Job, created in the installer namespace |
| `ClusterRoleBinding` | `{appName}` | Binds the `ServiceAccount` to the `cluster-admin` `ClusterRole` |

**Why `cluster-admin`**: Installers deploy cross-namespace workloads including operators, CRDs, `ClusterRole` definitions, and cluster-scoped resources. A narrower role would require enumerating every resource type the installer might create, which varies per consumer.

**Prerequisite**: The user running the MCP server must already have permission to create `ServiceAccount` and `ClusterRoleBinding` resources. The framework cannot escalate beyond the user's existing RBAC permissions — it delegates them to the `ServiceAccount` so the Job can operate autonomously.

**Mitigations**:
- Dedicated `ServiceAccount` scoped to a single installer (not shared across applications)
- Single-use Jobs (`BackoffLimit: 0`) — a failed Job does not retry
- `dry-run` mode enabled by default — the `deploy` tool creates a dry-run Job unless explicitly set to `dry_run: false`
- Auditable: Job logs are accessible via `kubectl logs` with the label selector `type=installer-job.helmet.redhat-appstudio.github.com`

### Image Security

- Use images from trusted registries
- Prefer immutable tags or digests over `latest` — see [Container Image Setup](#container-image-setup) for commit-ID tagging
- Scan images for vulnerabilities
- The cluster must be able to pull the image — verify registry accessibility and image pull secrets if using a private registry

## Troubleshooting

**MCP server won't start**: Check binary is in `$PATH`, `instructions.md` exists in embedded filesystem, `kubectl` access works.

**Tools not appearing**: Verify client config JSON syntax, command path, and that the server starts without errors.

**Deployment Job fails**: Check Job logs (`kubectl logs job/<appname>-deploy-job`), verify RBAC, confirm image is pullable from the cluster.

**Integration status mismatch**: Verify secret namespace matches configuration, and secret keys match the integration module's expectations.

## Cross-References

- [Getting Started](getting-started.md) — initial setup and embedding installer tarball
- [Architecture](architecture.md) — component interactions and data flow
- [Integrations](integrations.md) — integration modules and credential management
- [CLI Reference](cli-reference.md) — `mcp-server` command usage
