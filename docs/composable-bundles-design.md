# Composable chart bundles — design overview

This document describes the goals and shape of **composable bundles**: treating each installer chart as an independent unit with its own configuration surface, while allowing a single **blueprint** (`helmet.yaml`) to declare how charts participate in an install.

---

## 1. Problem we are solving

Installers today often ship as **one monolithic config** and **one root values template**, which couples charts together and makes it hard to:

- Enable only part of the stack (for example “integration-only” registry credentials vs full product install).
- Evolve one chart’s config or templating without touching unrelated charts.
- Reuse the same chart in **different roles** (minimal integration bundle vs full product) without duplicating installers.

We want **composition**: each chart remains a **bundle** with its own **`config.yaml`** (schema fragment), **`values.yaml.tpl`**, and Helm templates, while the overall install is **declared** in one place and stays **predictable** for CI/CD and humans.

---

## 2. Composable bundles per chart

### Per-chart ownership

Each chart under `charts/<chart>/` is treated as an independent bundle that owns:

| Artifact | Role |
|----------|------|
| **`config.yaml`** | Declares that chart’s slice of installer configuration (products, settings keys, or integration defaults as applicable). |
| **`values.yaml.tpl`** | Renders Helm values for **that chart only**, using installer context (namespace, settings, products, integrations). |
| **`Chart.yaml`** | Framework annotations: dependencies, bundle behavior, integration names for topology/CEL, optional documentation of installer integration **id** vs topology name. |

The **root** installer may still provide shared defaults or glue (`settings`, shared `values.yaml.tpl` for cross-cutting keys), but **bundle-specific** behavior and templates live **with the chart**.

### Why it matters

- Teams can change one chart’s templates or config contract without a giant merge conflict in a single root template.
- Testing and review can focus on **one bundle** at a time.

---

## 3. Driving the install from `helmet.yaml`

For installers that adopt a **distributed layout**, a **`helmet.yaml`** blueprint lists **which charts participate** and **how**:

- **`products`**: `local://` references to charts that should appear as **products** (full bundle path when the chart supports it).
- **`integrations`**: `local://` references to charts that should be deployed as **integration bundles** when the chart supports that mode.

The suffix of each reference (for example `local://quay`, `local://tpa`) aligns with the chart directory naming convention (`charts/tssc-quay`, `charts/tssc-tpa`) and drives generated **`installer.integrations`** entries (including merged **properties** defaults) where applicable.

This gives a **single declarative file** for “what is in this install” without encoding all low-level YAML by hand in one mega-file.

---

## 4. Bundle mode / bundle type (concept)

Each chart advertises **which placements are valid** for that bundle — for example:

- **Integration**: minimal footprint (often secrets, connectivity checks, or slim resources) suitable for listing under **`integrations`** in `helmet.yaml`.
- **Product**: full install path suitable for listing under **`products`**.
- **Both**: the chart may be listed in either list; **runtime behavior** (integration vs product) follows config and template logic, not only the list name.

We refer to this informally as **bundle mode** or **bundle type** support on the chart (declared via framework annotations). The blueprint (`helmet.yaml`) must not place a chart in a role the chart does not support (the framework validates this during merge / resolution).

---

## 5. Integration bundles and the ConfigMap

Charts that contribute **integration** configuration need a stable place for **non-secret** or **placeholder** values that operators replace before or after **`deploy`**:

- Merged installer configuration is persisted in the cluster (typically a **ConfigMap**).
- **Integration** entries gain **`properties`** (and display metadata) merged from chart **`values.yaml`** defaults and/or the blueprint flow.
- Operators still replace placeholders (URLs, tokens, flags) with **real environment-specific values** before a successful deploy — same operational model as classic installs, but **scoped per integration** in config.

So: **integration charts both participate in topology and populate config surfaces that must be filled with real values**; the design does not assume secrets live in Git — only that the **shape** of config is clear and mergeable.

---

## 6. Backward compatibility (classic structure)

**Helmet remains compatible with the classic installer layout**: a single embedded **`config.yaml`**, root **`values.yaml.tpl`**, and **`charts/`** described only by framework annotations — **without** `helmet.yaml`, **without** distributed merge, and **without** listing integrations in config when operators manage integrations solely via CLI or cluster secrets.

Composable bundles and `helmet.yaml` are **additive**; consumers adopt them incrementally. Existing semantics for chart discovery, dependency resolution, and ConfigMap-backed configuration continue to apply unless a consumer explicitly enables the new paths.

---

## References

- [Configuration](configuration.md) — schema, integrations list, template variables.
- [Topology](topology.md) — annotations, namespace assignment, `install-release-in-installer-namespace`.
- [Installer structure](installer-structure.md) — directory layout, optional distributed merge files.
