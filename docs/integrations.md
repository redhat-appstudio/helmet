# Integrations

An integration is a structured Kubernetes Secret used for configuration and connecting services internal and external to the cluster. Each Secret carries both sensitive credentials (tokens, private keys, client secrets) and plain configuration attributes (endpoints, hostnames, ports, group names) — everything a consuming component needs to connect to the integrated service.

Helmet's integration system provides pluggable connectors for these services. Integrations couple bidirectionally with products through chart annotations and CEL expressions.

This page covers the integration architecture, standard integrations, product-integration coupling, CEL expression syntax, custom integration development, and credential security. For how integrations interact with the deployment topology, see [topology.md](topology.md). For MCP-assisted integration workflows, see [mcp.md](mcp.md).

## Architecture

The integration system uses a three-layer architecture:

1. **IntegrationModule** (`api.IntegrationModule`): Public API contract defining the integration's identity, initialization, and CLI command
2. **Integration** (`internal/integration.Integration`): Secret wrapper managing lifecycle (create, delete, exists) with force-overwrite support
3. **integration.Interface** (`internal/integration.Interface`): Business logic provider implementing validation, secret data generation, and Kubernetes secret type. The `Data()` method returns `map[string][]byte` containing both sensitive credentials and plain configuration attributes

Secret naming convention: `{appName}-{moduleName}-integration`

## Standard Integrations

Helmet provides 11 standard integrations:

| Name | Type | Description |
|------|------|-------------|
| `acs` | Security | Red Hat Advanced Cluster Security |
| `artifactory` | Registry | JFrog Artifactory container registry |
| `azure` | Cloud | Microsoft Azure cloud services |
| `bitbucket` | SCM | Bitbucket Git provider |
| `github` | SCM | GitHub Git provider (supports GitHub Apps) |
| `gitlab` | SCM | GitLab Git provider |
| `jenkins` | CI/CD | Jenkins automation server |
| `nexus` | Registry | Sonatype Nexus repository manager |
| `quay` | Registry | Red Hat Quay container registry |
| `tas` | Security | Trusted Artifact Signer (Sigstore) |
| `trustification` | Security | Supply chain security platform |

Access standard integrations via:

```go
framework.StandardIntegrations() // Returns []api.IntegrationModule
```

## Product-Integration Coupling

Products and integrations form a bidirectional relationship through chart annotations.

### Providing Integrations

A chart declares integrations it creates using `integrations-provided`:

```yaml
# charts/quay-operator/Chart.yaml
annotations:
  helmet.redhat-appstudio.github.com/product-name: "Product B"
  helmet.redhat-appstudio.github.com/integrations-provided: "quay"
```

### Requiring Integrations

A chart declares integration dependencies using `integrations-required` with CEL expressions:

```yaml
# charts/tekton-chains/Chart.yaml
annotations:
  helmet.redhat-appstudio.github.com/product-name: "Product C"
  helmet.redhat-appstudio.github.com/integrations-required: "acs"
```

### Dependency Cascade

Disabling a product that provides an integration affects all dependent products:

**Example**:
- Product A provides `acs`
- Product B provides `quay`
- Product C provides `nexus`, requires `acs`
- Product D requires `quay && nexus`

**Scenario**: Disable Product A → `acs` no longer provided → Product C fails validation → `nexus` no longer provided → Product D fails validation.

**Mitigation**: Create integrations manually via CLI before disabling provider products:

```bash
helmet-ex integration acs --endpoint=acs.example.com:443 --token=...
```

Now Product C's requirement is satisfied even with Product A disabled.

## CEL Expression Syntax

Integration requirements use the [Common Expression Language (CEL)](https://github.com/google/cel-spec) with boolean variables.

### Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `&&` | Logical AND | `github && quay` |
| `\|\|` | Logical OR | `quay \|\| artifactory` |
| `!` | Logical NOT | `!github` |
| `()` | Grouping | `(github \|\| gitlab) && quay` |

Each standard integration name is a boolean variable: `true` if the integration secret exists in the cluster, `false` otherwise.

### Examples

```yaml
# Single integration
integrations-required: "github"

# Multiple required
integrations-required: "acs && quay && tas"

# Alternative providers
integrations-required: "github || gitlab || bitbucket"

# Complex logic
integrations-required: "(github || gitlab) && (quay || artifactory || nexus) && tas"
```

### Validation

CEL expressions are compiled and validated at deployment time:

1. Parse expression into AST
2. Type-check against known integration names
3. Extract referenced variables
4. Evaluate with current cluster state
5. Report specific missing integrations on failure

```text
Error: integration validation failed for chart "helmet-product-d":
  Required integrations not configured: quay
  Expression: acs && quay
```

## Custom Integration Development

### Step 1: Implement `integration.Interface`

```go
package myintegration

import (
    "context"
    "fmt"
    "log/slog"

    "github.com/redhat-appstudio/helmet/internal/config"
    "github.com/redhat-appstudio/helmet/internal/runcontext"
    "github.com/spf13/cobra"
    corev1 "k8s.io/api/core/v1"
)

type Custom struct {
    endpoint string
    apiKey   string
}

func (c *Custom) PersistentFlags(cmd *cobra.Command) {
    p := cmd.PersistentFlags()
    p.StringVar(&c.endpoint, "endpoint", "", "Service endpoint URL")
    p.StringVar(&c.apiKey, "api-key", "", "API authentication key")
    for _, f := range []string{"endpoint", "api-key"} {
        _ = cmd.MarkPersistentFlagRequired(f)
    }
}

func (c *Custom) LoggerWith(logger *slog.Logger) *slog.Logger {
    return logger.With("endpoint", c.endpoint)
}

func (c *Custom) Validate() error {
    if c.endpoint == "" {
        return fmt.Errorf("--endpoint is required")
    }
    return nil
}

func (c *Custom) Type() corev1.SecretType {
    return corev1.SecretTypeOpaque
}

func (c *Custom) SetArgument(key, value string) error {
    return nil
}

func (c *Custom) Data(
    _ context.Context,
    _ *runcontext.RunContext,
    _ *config.Config,
) (map[string][]byte, error) {
    return map[string][]byte{
        "endpoint": []byte(c.endpoint),
        "api-key":  []byte(c.apiKey),
    }, nil
}
```

### Step 2: Create IntegrationModule and Register

```go
import (
    "log/slog"

    "github.com/redhat-appstudio/helmet/api"
    "github.com/redhat-appstudio/helmet/internal/integration"
    "github.com/redhat-appstudio/helmet/internal/k8s"
)

var CustomModule = api.IntegrationModule{
    Name: "custom",
    Init: func(logger *slog.Logger, kube k8s.Interface) integration.Interface {
        return &Custom{}
    },
    Command: func(appCtx *api.AppContext, runCtx *runcontext.RunContext, i *integration.Integration) api.SubCommand {
        // Return a SubCommand for the CLI
        return NewCustomCommand(appCtx, runCtx, i)
    },
}
```

### Step 3: Register with Framework

```go
integrations := append(
    framework.StandardIntegrations(),
    CustomModule,
)

app, _ := framework.NewAppFromTarball(
    appCtx, installerTarball, cwd,
    framework.WithIntegrations(integrations...),
)
```

The integration CLI command is automatically registered under `helmet-ex integration custom`.

### Step 4: Use in Charts

```yaml
# charts/helmet-product-a/Chart.yaml
annotations:
  helmet.redhat-appstudio.github.com/integrations-required: "custom"
```

## URLProvider Interface

The `URLProvider` interface customizes GitHub App URLs (callback, homepage, webhook) without importing internal packages:

```go
type IntegrationContext interface {
    GetOpenShiftIngressDomain(ctx context.Context) (string, error)
    GetProductNamespace(productName string) (string, error)
}

type URLProvider interface {
    GetCallbackURL(ctx context.Context, ic IntegrationContext) (string, error)
    GetHomepageURL(ctx context.Context, ic IntegrationContext) (string, error)
    GetWebhookURL(ctx context.Context, ic IntegrationContext) (string, error)
}
```

Register with:

```go
integrations := framework.StandardIntegrations()
integrations = framework.WithURLProvider(integrations, MyURLProvider{})

app, _ := framework.NewAppFromTarball(
    appCtx, installerTarball, cwd,
    framework.WithIntegrations(integrations...),
)
```

`WithURLProvider` replaces the GitHub module with one that uses the provided `URLProvider` for URL generation, leaving all other integrations unchanged.

## Credential Security

### Secrets Management

The following applies to Secrets created via the `integration` CLI subcommand:

- **Structured data**: Each Secret contains both sensitive credentials (tokens, private keys) and plain configuration (endpoints, hostnames, ports) as `map[string][]byte` entries
- **Namespaced**: Secrets are created in the installer's configured namespace
- **Opaque Type**: Most integrations use `SecretTypeOpaque`
- **Immutable by Default**: Existing secrets are not overwritten unless `--force` flag is used
- **Naming**: `{appName}-{moduleName}-integration` (e.g., `helmet-ex-github-integration`)

When integration Secrets are managed by Helm charts (via `integrations-provided`), the chart's templates control the Secret lifecycle entirely. Overwrite behavior, naming, secret type, and namespace placement are the chart author's responsibility. The framework only records the `integrations-provided` declaration for topology resolution — it does not manage chart-created Secrets.

### OVERWRITE_ME Placeholders

The MCP server's `integration_scaffold` tool generates shell commands with `OVERWRITE_ME` placeholders for sensitive values:

```bash
helmet-ex integration github \
  --app-id=OVERWRITE_ME \
  --private-key=OVERWRITE_ME \
  --webhook-secret=OVERWRITE_ME
```

**Security policy**: Automated tools (MCP servers, AI assistants) must NOT replace `OVERWRITE_ME` values. Users must manually fill credentials and run commands in their terminal.

## Cross-References

- [Topology](topology.md) — dependency resolution and integration requirements
- [MCP Server](mcp.md) — AI assistant integration tools
- [Example Charts](example-charts.md) — chart annotation examples with integration coupling
