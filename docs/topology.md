# Dependency Topology

The Helmet framework automatically resolves installation order based on chart dependencies, weights, product associations, and integration requirements. The topology resolution pipeline transforms a collection of Helm charts with declared dependencies into a linear deployment sequence that respects all constraints, assigns namespaces, and validates integration availability before deployment.

This page covers the dependency topology system: annotations for declaring dependencies, the resolution algorithm, namespace assignment rules, integration validation, and troubleshooting. For integration details and CEL expression syntax, see [integrations.md](integrations.md). For configuration, see [configuration.md](configuration.md).

## Viewing the Topology

```sh
helmet-ex topology
```

The topology command outputs the resolved installation order with metadata for each chart: installation index, chart name, weight, target namespace, dependencies, and integrations provided/required.

Use this command before deployment to verify all dependencies are resolved, ordering is correct, namespace assignments match expectations, and integration requirements can be satisfied.

## Chart Annotations

Charts declare dependencies and metadata using annotations in `Chart.yaml`. All Helmet annotations use the `helmet.redhat-appstudio.github.com/` prefix.

| Annotation | Purpose | Value Type |
|------------|---------|------------|
| `product-name` | Associates chart with a product | String |
| `use-product-namespace` | Deploy into another product's namespace | String (product name) |
| `depends-on` | Explicit dependency list | Comma-separated chart names |
| `weight` | Installation order | Integer; higher = later, default `0`, negative allowed |
| `integrations-provided` | Integrations this chart creates | Comma-separated integration names |
| `integrations-required` | Integration requirements | CEL expression |

### `product-name`

Associates a chart with a product defined in `config.yaml`. The chart deploys to the product's configured namespace. Only one chart per product should have this annotation.

```yaml
annotations:
  helmet.redhat-appstudio.github.com/product-name: "Product A"
```

### `use-product-namespace`

Deploys a dependency chart into a specific product's namespace (for charts without `product-name`).

```yaml
annotations:
  helmet.redhat-appstudio.github.com/use-product-namespace: "Product A"
```

### `depends-on`

Comma-separated list of chart names that must be deployed before this chart.

```yaml
annotations:
  helmet.redhat-appstudio.github.com/depends-on: "helmet-foundation, helmet-operators"
```

The resolver detects circular dependencies and panics with a detailed cycle trace.

### `weight`

Integer defining installation order within a dependency tier. Higher weights install later; lower weights install earlier. Default is `0` when the annotation is absent. Negative values are allowed — a chart with a negative weight installs before zero-weight peers in the same tier.

The value is parsed with `strconv.Atoi`, so any valid integer string is accepted (e.g., `"-10"`, `"0"`, `"500"`).

```yaml
annotations:
  helmet.redhat-appstudio.github.com/weight: "500"
```

**Recommended Weight Ranges:**

| Range | Purpose | Examples |
|-------|---------|----------|
| Negative | Infrastructure and early-stage charts | CRD installers, namespace setup |
| 0 (default) | Standard deployment order | Most application charts |
| 1-98 | Late-stage application services | Dependent services, post-configuration |
| 99+ | Companion and deferred charts | Post-deployment validation, cleanup |

### `integrations-provided`

Comma-separated list of integrations this chart creates.

```yaml
annotations:
  helmet.redhat-appstudio.github.com/integrations-provided: "acs"
```

### `integrations-required`

CEL expression specifying required integrations. Supports `&&`, `||`, `!`, and parentheses.

```yaml
annotations:
  helmet.redhat-appstudio.github.com/integrations-required: "acs && quay"
```

See [integrations.md](integrations.md) for CEL expression syntax and examples.

## Resolution Algorithm

The resolver operates in two phases, both using recursive dependency resolution with circular detection. All iteration orders are deterministic: Phase 1 processes products in `config.yaml` declaration order, Phase 2 processes remaining charts in alphabetical order by name (`Collection.Walk()` sorts with `slices.Sort`), and `depends-on` values are resolved left-to-right. This guarantees reproducible topology output regardless of filesystem ordering or map iteration order.

### Phase 1: Product Resolution

For each product enabled in `config.yaml`:

1. Find the chart with matching `product-name` annotation
2. Set the chart's namespace to the product's configured namespace
3. Recursively resolve the chart's `depends-on` dependencies
4. Append the chart to the topology after its dependencies

### Phase 2: Dependency Resolution

After all enabled products are resolved, the framework processes remaining charts (charts without `product-name` that were not pulled in as dependencies):

1. Recursively resolve `depends-on` dependencies
2. Determine namespace (default or `use-product-namespace`)
3. Append to topology after dependencies

### Recursive Dependency Resolution

When a chart declares dependencies via `depends-on`:

1. Parse comma-separated dependency names
2. For each dependency:
   - Look up the chart in the collection
   - If the dependency has its own dependencies, resolve them first (recurse)
   - Set the dependency's namespace
   - Insert the dependency before the current chart
3. Detect circular dependencies using a `visited` map
4. Sort dependencies by weight (higher weight = later position)

### Weight-Based Ordering

When inserting a dependency into the topology:
- **Higher weight** → inserted later in the sequence
- **Lower weight (including negative)** → inserted earlier in the sequence
- **Equal weight** → order determined by dependency relationships
- **Same weight, no dependency relationship** → insertion order from `Collection.Walk()`, which iterates chart names alphabetically

## The TopologyBuilder Pipeline

The topology resolution process is managed by three components:

```text
ChartFS → Collection → TopologyBuilder.Build() → Resolver.Resolve() → Topology → Integration Validation → Validated Topology
```

**1. Collection**: Reads all Helm charts from `ChartFS` and indexes them by name and product association.

**2. Resolver**: Implements the two-phase resolution algorithm producing the ordered `Topology`.

**3. Integration Validation**: For each chart in the topology, evaluates `integrations-required` CEL expressions against configured integrations. Fails with detailed error listing missing integrations.

## Namespace Assignment

Each chart is assigned a target namespace using a three-tier priority system:

| Priority | Condition | Namespace Source |
|----------|-----------|-----------------|
| 1 | Chart has `product-name` | Product's namespace from `config.yaml` |
| 2 | Chart has `use-product-namespace` | Referenced product's namespace from `config.yaml` |
| 3 | Neither annotation | Installer's default namespace |

### Example

From [`helmet-ex/installer/config.yaml`](../example/helmet-ex/installer/config.yaml):

```yaml
# config.yaml
<app_name>:
  products:
    - name: Product A
      namespace: helmet-product-a
      enabled: true
```

```yaml
# charts/helmet-product-a/Chart.yaml — product chart
annotations:
  helmet.redhat-appstudio.github.com/product-name: "Product A"
# → Deploys to: helmet-product-a
```

```yaml
# charts/helmet-infrastructure/Chart.yaml — no product annotations
# → Deploys to: helmet-ex-system (installer's default namespace)
```

## Dependency Graph Example

The [`helmet-ex`](../example/helmet-ex/) example includes 10 charts demonstrating multi-tier dependencies, integration providers and consumers, and weight-based ordering. See [example-charts.md](example-charts.md) for the full chart inventory, dependency graph diagram, and deployment order walkthrough.

## Troubleshooting

### Missing Dependency

```text
Error: chart "missing-chart" not found in collection
```

Verify the dependency chart name matches the `name` field in its `Chart.yaml` and that the chart is present in the charts directory.

### Circular Dependency

```text
panic: Circular dependency detected: chart-a -> chart-b -> chart-a
```

Break the cycle by removing one dependency or introducing a shared dependency chart.

### Integration Requirement Not Satisfied

```text
Error: integration validation failed: missing integrations: quay
```

Enable a product that provides the integration, or create the integration manually via CLI.

### Product Chart Not Found

If a product is enabled but its chart doesn't appear in the topology, verify the product name in `config.yaml` matches the `product-name` annotation exactly (case-sensitive).

## Infrastructure Charts & Derived Products

Not all charts map to products in config.yaml. Infrastructure and foundation charts handle shared concerns — namespace setup, operator subscriptions, shared databases, IAM, integration aggregation. In production installers, 40–60% of charts are infrastructure with no product entry.

Infrastructure charts are **outside the triad**: they have a root-key and a values.yaml.tpl section but no `product-name` annotation and no config.yaml entry. They may have no Helmet annotations at all (serving as base layers), or use only `depends-on` and integration annotations.

| Chart Pattern | Purpose | Typical Annotations |
|---|---|---|
| Namespace setup | Create OpenShift projects | None |
| Operator subscriptions | Install OLM operators | None or `depends-on` |
| Shared databases | PostgreSQL for multiple products | `depends-on` |
| IAM / Identity | Keycloak realm, routes, services | `depends-on`, `integrations-provided` |
| Integration aggregation | Collect integration secrets | `depends-on` |

**Derived products** are charts whose enablement is computed from other products' state rather than configured directly. The canonical example is IAM/Keycloak, where enablement grows with the number of dependent products:

```yaml
{{- $keycloakEnabled := or $tpa.Enabled $tas.Enabled -}}
```

Derived products have no entry in config.yaml. Their enablement and configuration are derived in `values.yaml.tpl` from other products. Namespace assignment follows the standard topology resolution (installer default namespace, or inherited via `use-product-namespace` annotation). When adding or removing products, trace derived enablement logic -- disabling all upstream products implicitly disables the derived chart.

See [example-charts.md](example-charts.md) for concrete infrastructure chart examples from the test fixtures.

## Companion Charts & Multi-Root-Key Patterns

**Companion charts** are satellite charts that extend a product chart. They deploy in their parent product's namespace using `use-product-namespace` and control ordering with `weight`:

```yaml
# charts/my-product-test/Chart.yaml
annotations:
  helmet.redhat-appstudio.github.com/use-product-namespace: "My Product"
  helmet.redhat-appstudio.github.com/depends-on: "my-product"
  helmet.redhat-appstudio.github.com/weight: "99"
```

The `weight: "99"` ensures the companion deploys after its parent within the same dependency tier (higher weight = later). Companion charts inherit their parent's values via YAML anchors in values.yaml.tpl:

```yaml
myProduct: &myProduct
  enabled: true
  namespace: {{ $myProduct.Namespace }}

myProductTest: *myProduct
```

**Prerequisite splits** are different from companions — they are charts split out for lifecycle separation. A prerequisite deploys *before* its dependent chart (no `use-product-namespace`, no `weight`). It is structurally an infrastructure chart.

**Multi-root-key charts** have more than one top-level key in values.yaml. This is acceptable for:

1. **Sub-components** with distinct value trees but shared config (bridged via YAML anchors, e.g., `trustedProfileAnalyzer` + `trustification`)
2. **Cross-cutting concerns** like debug flags that don't belong under the primary key

Each chart should have one identifiable primary root key. Secondary keys are acceptable when justified — warn on unintentional cases.

See [example-charts.md](example-charts.md) for chart annotation examples.

## Best Practices

- Use low or negative weights for infrastructure charts that must deploy first
- Prefer explicit `depends-on` over implicit weight ordering
- One `product-name` per product
- Validate topology before production: `helmet-ex topology`
- Document complex dependencies in Chart.yaml comments

## Cross-References

- [Integrations](integrations.md) — integration configuration, CEL expression syntax
- [Configuration](configuration.md) — product and namespace configuration, [the triad](configuration.md#the-triad)
- [Templating](templating.md) — values.yaml.tpl syntax, [YAML anchors and production patterns](templating.md#production-patterns)
- [Example Charts](example-charts.md) — complete chart examples with annotations
- [CLI Reference](cli-reference.md) — `topology` command usage
