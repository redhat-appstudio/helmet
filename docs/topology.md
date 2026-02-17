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
| `weight` | Installation priority | Integer; higher = earlier, default `0`, negative allowed |
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

Integer defining installation priority within a dependency tier. Higher weights install earlier; lower weights install later. Default is `0` when the annotation is absent. Negative values are allowed — a chart with a negative weight installs after all zero-weight peers in the same tier.

The value is parsed with `strconv.Atoi`, so any valid integer string is accepted (e.g., `"-10"`, `"0"`, `"500"`).

```yaml
annotations:
  helmet.redhat-appstudio.github.com/weight: "500"
```

**Recommended Weight Ranges:**

| Range | Purpose | Examples |
|-------|---------|----------|
| 1000+ | Infrastructure and cluster operators | CRD installers, namespaces |
| 500-999 | Platform services | Databases, message queues |
| 100-499 | Application services | APIs, web services |
| 0-99 | User applications and integrations | Business applications |
| Negative | Deferred charts within a tier | Post-deployment validation, cleanup |

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
4. Sort dependencies by weight (higher weight = earlier position)

### Weight-Based Ordering

When inserting a dependency into the topology:
- **Higher weight** → inserted earlier in the sequence
- **Lower weight (including negative)** → inserted later in the sequence
- **Equal weight** → order determined by dependency relationships
- **Same weight, no dependency relationship** → alphabetical order by chart name (guaranteed by `Collection.Walk()` sorting chart names with `slices.Sort` before resolution)

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
tssc:
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

## Best Practices

- Use high weights (1000+) for infrastructure charts
- Prefer explicit `depends-on` over implicit weight ordering
- One `product-name` per product
- Validate topology before production: `helmet-ex topology`
- Document complex dependencies in Chart.yaml comments

## Cross-References

- [Integrations](integrations.md) — integration configuration, CEL expression syntax
- [Configuration](configuration.md) — product and namespace configuration
- [Example Charts](example-charts.md) — complete chart examples with annotations
- [CLI Reference](cli-reference.md) — `topology` command usage
