# Templating

The framework uses Go's `text/template` engine to render `values.yaml.tpl` into Helm values before chart installation. This enables dynamic value generation based on installer configuration, cluster introspection, and live Kubernetes resources.

This page covers the template context structure, available functions, rendering lifecycle, and common patterns. For configuration schema and ConfigMap storage, see [configuration.md](configuration.md). For installer directory layout and `values.yaml.tpl` placement, see [installer-structure.md](installer-structure.md).

## Rendering Lifecycle

Template rendering occurs during the deployment workflow:

1. **Load Configuration**: `config.Config` reads and validates `config.yaml`
2. **Build Context**: `engine.Variables` populates `.Installer` and `.OpenShift` variables
3. **Render Template**: `engine.Engine` processes `values.yaml.tpl` with the context
4. **Helm Install**: Rendered values pass to `helm install` or `helm upgrade`

Each chart uses the same global values file. Template rendering happens once per deployment, not per chart.

## Template Context

The template context provides two top-level objects: `.Installer` (configuration data) and `.OpenShift` (cluster metadata).

### `.Installer` Structure

| Path | Type | Source | Description |
|------|------|--------|-------------|
| `.Installer.Namespace` | string | CLI flag `--namespace` or `AppContext` default | Target namespace for the installer |
| `.Installer.Settings` | map | `config.yaml` `settings` section | Global installer settings (freeform key-value) |
| `.Installer.Products` | map | `config.yaml` `products` section | Map of products keyed by `KeyName()` |
| `.Installer.Products.<KeyName>` | object | Product configuration | Individual product specification |
| `.Installer.Products.<KeyName>.Name` | string | Product `name` field | Human-readable product name |
| `.Installer.Products.<KeyName>.Enabled` | boolean | Product `enabled` field | Whether product is enabled for deployment |
| `.Installer.Products.<KeyName>.Namespace` | string | Product `namespace` field or installer namespace | Target namespace for product's chart |
| `.Installer.Products.<KeyName>.Properties` | map | Product `properties` field | Product-specific configuration (freeform) |

**KeyName Conversion**: Product names are sanitized for template use. Any character that is not a letter, digit, or underscore is replaced with an underscore. Multiple consecutive underscores are collapsed, and leading/trailing underscores are removed. For example, `Product A` becomes `Product_A`.

### `.OpenShift` Structure

| Path | Type | Description |
|------|------|-------------|
| `.OpenShift.Ingress.Domain` | string | OpenShift ingress domain from `IngressController` CR (empty on vanilla Kubernetes) |
| `.OpenShift.Ingress.RouterCA` | string | Base64-encoded router CA certificate from `router-ca` secret or custom certificate (empty on vanilla Kubernetes) |
| `.OpenShift.Version` | string | OpenShift cluster version from `ClusterVersion` CR (empty on vanilla Kubernetes) |
| `.OpenShift.MinorVersion` | string | Minor version extracted from full version (e.g., `4.18` from `4.18.2`) |

**Vanilla Kubernetes**: All `.OpenShift` fields return empty strings if OpenShift APIs are unavailable. Templates should handle both cases.

### Context Population

The framework populates the context via:

```go
// internal/engine/variables.go
variables := engine.NewVariables()
variables.SetInstaller(cfg)           // Populates .Installer
variables.SetOpenShift(ctx, kube)     // Populates .OpenShift
```

OpenShift detection queries these resources:
- `operator.openshift.io/v1 IngressController` in `openshift-ingress-operator` namespace
- `config.openshift.io/v1 ClusterVersion` named `version`
- `Secret` in `openshift-ingress-operator` or `openshift-ingress` namespace for router CA

Errors are silently handled by returning empty strings. This ensures templates work on both OpenShift and vanilla Kubernetes clusters.

## Available Functions

The template engine provides the full Sprig library plus custom functions for serialization, validation, and cluster introspection.

### Sprig Functions

All functions from [Sprig v3](https://masterminds.github.io/sprig/) are available, including:
- String manipulation (`trim`, `upper`, `lower`, `replace`)
- Type conversion (`toString`, `toInt`)
- Collections (`list`, `dict`, `merge`)
- Flow control (`default`, `ternary`, `coalesce`)
- Encoding (`b64enc`, `b64dec`)
- Cryptography (`sha256sum`, `randAlpha`)
- Math (`add`, `sub`, `mul`, `div`)
- Deep access (`dig`, `pluck`, `pick`, `omit`)

Refer to the [Sprig documentation](https://masterminds.github.io/sprig/) for the complete reference.

### Custom Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `toYaml` | `(data interface{}) string` | Serializes value to YAML string, trimming trailing newline |
| `fromYaml` | `(str string) map[string]interface{}` | Deserializes YAML string to map; returns `{"Error": "<msg>"}` on failure |
| `fromYamlArray` | `(str string) []interface{}` | Deserializes YAML string to array; returns `["<msg>"]` on failure |
| `toJson` | `(v interface{}) string` | Serializes value to JSON string |
| `fromJson` | `(str string) map[string]interface{}` | Deserializes JSON string to map; returns `{"Error": "<msg>"}` on failure |
| `fromJsonArray` | `(str string) []interface{}` | Deserializes JSON string to array; returns `["<msg>"]` on failure |
| `required` | `(name string, value interface{}) (interface{}, error)` | Returns value if non-nil; returns error with name if nil |
| `lookup` | `(apiVersion, kind, namespace, name string) (map[string]interface{}, error)` | Queries Kubernetes resource; returns unstructured content or empty map |

### Serialization Functions

**`toYaml`**: Converts Go data structures to YAML strings. Useful for passing nested configuration to Helm charts.

```yaml
# Convert product properties to YAML
productA:
  config: |
{{- .Installer.Products.Product_A.Properties | toYaml | nindent 4 }}
```

**`fromYaml` / `fromYamlArray`**: Parse YAML strings into Go structures. Use for processing YAML embedded in configuration properties.

**`toJson` / `fromJson` / `fromJsonArray`**: JSON equivalents of the YAML functions.

### Validation Functions

**`required`**: Fails template rendering with a descriptive error if a value is nil. Use to enforce required configuration fields.

```yaml
catalogURL: {{ required "Product_D.Properties.catalogURL" .Installer.Products.Product_D.Properties.catalogURL }}
```

Error message: `Product_D.Properties.catalogURL is required`

### Cluster Introspection

**`lookup`**: Queries live Kubernetes resources during template rendering. Returns unstructured content as `map[string]interface{}`.

**Signature**:
```go
lookup(apiVersion, kind, namespace, name string) (map[string]interface{}, error)
```

**Parameters**:
- `apiVersion`: Resource API version (e.g., `v1`, `apps/v1`, `config.openshift.io/v1`)
- `kind`: Resource kind (e.g., `ConfigMap`, `Secret`, `ClusterVersion`)
- `namespace`: Namespace for namespaced resources (use `""` for cluster-scoped)
- `name`: Resource name (use `""` to list all resources of this type)

**Return values**:
- If `name` is specified: Returns the resource's unstructured content (`obj.UnstructuredContent()`)
- If `name` is empty: Returns list content (`objList.UnstructuredContent()`)
- If resource not found: Returns empty map `{}`
- On error (other than NotFound): Returns error

**Example: Get a ConfigMap**:

```yaml
{{- $cm := lookup "v1" "ConfigMap" "default" "my-config" }}
{{- if $cm }}
myValue: {{ $cm.data.key }}
{{- end }}
```

**Example: List all Secrets in a namespace**:

```yaml
{{- $secrets := lookup "v1" "Secret" "kube-system" "" }}
secretCount: {{ len $secrets.items }}
```

**Example: Check for OpenShift-specific resources**:

```yaml
{{- $ingress := lookup "operator.openshift.io/v1" "IngressController" "openshift-ingress-operator" "default" }}
{{- if $ingress }}
openshiftDomain: {{ $ingress.status.domain }}
{{- end }}
```

**Important**: The `lookup` function executes during template rendering, before Helm charts are deployed. Resources queried via `lookup` must already exist in the cluster.

## Common Patterns

### Conditional Rendering Based on Product Enablement

Only include chart values when a product is enabled:

```yaml
{{- if .Installer.Products.Product_A.Enabled }}
productA:
  namespace: {{ .Installer.Products.Product_A.Namespace }}
  enabled: true
{{- end }}
```

### Iterating Over All Products

Generate values for all products dynamically:

```yaml
products:
{{- range $key, $product := .Installer.Products }}
  {{- if $product.Enabled }}
  {{ $key }}:
    namespace: {{ $product.Namespace }}
  {{- end }}
{{- end }}
```

### Accessing Nested Settings

Use Sprig's `dig` function to safely access nested configuration with defaults:

```yaml
debug:
  ci: {{ dig "ci" "debug" false .Installer.Settings }}
```

### Combining Product Properties

Merge global settings with product-specific configuration:

```yaml
productB:
  namespace: {{ .Installer.Products.Product_B.Namespace }}
  storageClass: {{ .Installer.Products.Product_B.Properties.storageClass | default "standard" }}
  size: {{ .Installer.Products.Product_B.Properties.size | default "10Gi" }}
```

### Platform-Specific Configuration

Adjust values based on OpenShift availability:

```yaml
ingress:
{{- if .OpenShift.Ingress.Domain }}
  domain: {{ .OpenShift.Ingress.Domain }}
  routerCA: {{ .OpenShift.Ingress.RouterCA }}
  platform: openshift
{{- else }}
  domain: cluster.local
  platform: kubernetes
{{- end }}
```

### Building Namespace Lists

Collect namespaces for all enabled products:

```yaml
projects:
{{- range .Installer.Products }}
  {{- if and .Enabled .Namespace }}
  - {{ .Namespace }}
  {{- end }}
{{- end }}
```

This pattern is useful for namespace provisioning charts.

### Validating Required Properties

Use `required` to enforce configuration constraints:

```yaml
productD:
  catalogURL: {{ required "Product D catalogURL" .Installer.Products.Product_D.Properties.catalogURL }}
  authProvider: {{ .Installer.Products.Product_D.Properties.authProvider | default "oauth" }}
```

## Complete Example

Combining patterns into a realistic `values.yaml.tpl`:

```yaml
---
# Global installer settings
global:
  namespace: {{ .Installer.Namespace }}
  debug: {{ dig "ci" "debug" false .Installer.Settings }}
  platform: {{ if .OpenShift.Version }}openshift{{ else }}kubernetes{{ end }}

# OpenShift-specific values
{{- if .OpenShift.Ingress.Domain }}
openshift:
  domain: {{ .OpenShift.Ingress.Domain }}
  routerCA: {{ .OpenShift.Ingress.RouterCA }}
  version: {{ .OpenShift.Version }}
  minorVersion: {{ .OpenShift.MinorVersion }}
{{- end }}

# Product-specific values
{{- if .Installer.Products.Product_A.Enabled }}
productA:
  namespace: {{ .Installer.Products.Product_A.Namespace }}
  enabled: true
{{- end }}

{{- if .Installer.Products.Product_B.Enabled }}
productB:
  namespace: {{ .Installer.Products.Product_B.Namespace }}
  enabled: true
  properties:
{{- .Installer.Products.Product_B.Properties | toYaml | nindent 4 }}
{{- end }}

{{- if .Installer.Products.Product_D.Enabled }}
productD:
  namespace: {{ .Installer.Products.Product_D.Namespace }}
  enabled: true
  catalogURL: {{ required "catalogURL" .Installer.Products.Product_D.Properties.catalogURL }}
  authProvider: {{ .Installer.Products.Product_D.Properties.authProvider | default "oidc" }}
{{- end }}

# Namespace list for foundation chart
foundation:
  projects:
{{- range .Installer.Products }}
  {{- if and .Enabled .Namespace }}
    - {{ .Namespace }}
  {{- end }}
{{- end }}
```

## Scope Boundaries

This document covers template rendering mechanics and the `values.yaml.tpl` syntax. Related topics:

- **Configuration schema**: See [configuration.md](configuration.md) for `config.yaml` structure, product specifications, and ConfigMap persistence
- **Installer packaging**: See [installer-structure.md](installer-structure.md) for directory layout, embedding, and the overlay filesystem
- **Dependency resolution**: See [topology.md](topology.md) for chart ordering and annotation-based dependencies
- **OpenShift introspection**: Implemented in `internal/k8s/openshift.go`