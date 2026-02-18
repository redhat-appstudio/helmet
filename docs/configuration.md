# Configuration

The framework uses a YAML configuration file to define installer settings and product deployments. Configuration is persisted in the Kubernetes cluster as a ConfigMap and referenced by all installer operations.

This document covers the `config.yaml` schema, default values, ConfigMap storage, CLI operations, and how configuration is exposed to Helm templates. For dependency resolution and installation order, see [topology.md](topology.md). For template engine details, see [templating.md](templating.md).

## Configuration Schema

The configuration file uses a top-level key (matching the installer name) containing two sections: `settings` and `products`.

### Example Configuration

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
    - name: Product B
      enabled: true
      namespace: helmet-product-b
      properties:
        storageClass: standard
    - name: Product C
      enabled: true
      namespace: helmet-product-c
    - name: Product D
      enabled: true
      namespace: helmet-product-d
      properties:
        catalogURL: https://github.com/example/repo/blob/main/catalog.yaml
        manageSubscription: true
        authProvider: oidc
```

### Settings Section

The `settings` section is a freeform key-value map for installer-wide configuration that applies across all products. Settings are accessible in Helm templates as `.Installer.Settings`.

- Must be present (can be empty: `settings: {}`)
- Supports arbitrary nesting

### Products Section

The `products` section is a list of product specifications. Each product represents a deployable component with its own Helm chart and configuration.

## Product Field Reference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Product identifier; must match `product-name` annotation in chart |
| `enabled` | boolean | Yes | Toggle product deployment; only enabled products are installed |
| `namespace` | string | No | Kubernetes namespace for deployment; defaults to installer namespace |
| `properties` | map | No | Product-specific configuration passed to Helm chart as template variables |

### Product Name and KeyName

The `name` field is the human-readable product identifier. The framework converts this to a sanitized `KeyName` for use in templates:

- Non-alphanumeric characters replaced with `_`
- Multiple underscores collapsed to one
- Leading/trailing underscores removed

Examples:
- `Product A` → `Product_A`
- `my-product` → `my_product`

### Namespace Resolution

Products use the following namespace resolution order:

1. Explicit `namespace` field in product definition
2. Installer namespace from `--namespace` flag (applied via `ApplyDefaults()`)
3. Default namespace from `AppContext`

**Validation**: Enabled products must have a namespace. The framework returns an error if an enabled product has no namespace after default application.

## ConfigMap Persistence

Configuration is stored in the cluster as a ConfigMap, allowing consistent access across installer operations and restarts.

### ConfigMap Structure

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: <app-name>-config
  namespace: <installer-namespace>
  labels:
    helmet.redhat-appstudio.github.com/config: "true"
data:
  config.yaml: |
    ---
    tssc:
      settings: ...
      products: ...
```

| Detail | Value |
|--------|-------|
| Name format | `{appName}-config` |
| Label selector | `helmet.redhat-appstudio.github.com/config=true` |
| Data key | `config.yaml` |
| Cardinality | Single ConfigMap per cluster (enforced by label selector) |

### ConfigMap Operations

The `ConfigMapManager` provides CRUD operations:

| Method | Purpose |
|--------|---------|
| `Create(ctx, cfg)` | Creates new ConfigMap |
| `Update(ctx, cfg)` | Updates existing ConfigMap |
| `GetConfig(ctx)` | Retrieves configuration from cluster |
| `Delete(ctx)` | Deletes ConfigMap |

**Error Conditions**:
- `ErrConfigMapNotFound`: No ConfigMap with required label exists
- `ErrMultipleConfigMapFound`: Multiple ConfigMaps with label found (invalid state)
- `ErrIncompleteConfigMap`: ConfigMap exists but missing `config.yaml` key

## CLI Operations

### Create Configuration

```sh
helmet-ex config --create
```

Creates a new ConfigMap using the embedded default `config.yaml`. Fails if ConfigMap already exists.

```sh
# Force update existing ConfigMap
helmet-ex config --create --force

# Create in a specific namespace
helmet-ex config --create --namespace custom-namespace
```

### View Configuration

```sh
helmet-ex config --get
```

Displays the current ConfigMap contents in YAML format.

### Delete Configuration

```sh
helmet-ex config --delete
```

Removes the ConfigMap from the cluster. Does not affect deployed resources.

## Template Variable Access

Configuration is exposed to Helm templates via the `values.yaml.tpl` template system. The framework populates template variables from the Config struct during chart rendering.

### Available Variables

| Path | Type | Description |
|------|------|-------------|
| `.Installer.Namespace` | string | Installer's target namespace |
| `.Installer.Settings` | map | Flattened settings from config |
| `.Installer.Products.<KeyName>` | object | Product by sanitized name |
| `.Installer.Products.<KeyName>.Enabled` | boolean | Product enabled state |
| `.Installer.Products.<KeyName>.Namespace` | string | Product target namespace |
| `.Installer.Products.<KeyName>.Properties` | map | Product-specific properties |

Products are keyed by `KeyName()`, not the original `name` field. See [Product Name and KeyName](#product-name-and-keyname).

### Example Template Usage

```yaml
# values.yaml.tpl
namespace: {{ .Installer.Namespace }}

settings:
  crc: {{ .Installer.Settings.crc }}
  debug: {{ dig "ci" "debug" false .Installer.Settings }}

{{- if .Installer.Products.Product_A.Enabled }}
productA:
  namespace: {{ .Installer.Products.Product_A.Namespace }}
{{- end }}

{{- if .Installer.Products.Product_B.Enabled }}
productB:
  namespace: {{ .Installer.Products.Product_B.Namespace }}
  storageClass: {{ .Installer.Products.Product_B.Properties.storageClass }}
{{- end }}
```

## Product Properties

The `properties` field is a freeform map for product-specific configuration. Common patterns:

```yaml
# Storage configuration
properties:
  storageClass: standard
  storageSize: 10Gi

# External service URLs
properties:
  catalogURL: https://github.com/org/repo/blob/main/catalog.yaml
  apiEndpoint: https://api.example.com

# Feature toggles
properties:
  manageSubscription: true
  enableMetrics: false
  authProvider: oidc
```

## Default Configuration

Each installer embeds a default `config.yaml` at the root of its chart filesystem. This file is used when no custom configuration is provided.

```go
// Load embedded default
cfg, err := config.NewConfigDefault(chartFS, namespace)

// Load from file
cfg, err := config.NewConfigFromFile(chartFS, "config.yaml", namespace)

// Load from bytes (e.g., from cluster)
cfg, err := config.NewConfigFromBytes(payload, namespace)
```

The `ApplyDefaults()` method propagates the installer namespace to products without explicit `namespace` fields. This is called automatically during unmarshaling.

## Validation Rules

| Rule | Error |
|------|-------|
| `settings` section must exist | `missing settings` |
| Enabled products must have namespace | `product <name>: missing namespace` |
| Configuration must unmarshal successfully | `failed to unmarshal configuration` |

## Cross-References

- [Topology](topology.md) — chart annotations, dependency resolution, and installation order
- [Templating](templating.md) — values.yaml.tpl syntax and template functions
- [MCP Server](mcp.md) — configuration tools for AI-assisted workflows
- [CLI Reference](cli-reference.md) — `config` command usage