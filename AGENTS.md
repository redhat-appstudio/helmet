# Project: `github.com/redhat-appstudio/helmet`

Reusable Helm-based installer framework. Imported as a library to build custom installers for deploying workloads to Kubernetes clusters.

## MCRF — Meta-Cognitive Reasoning Framework

Apply for complex tasks (skip for trivial operations):

1. **DECOMPOSE** — Break into: API changes, chart logic, resolver behavior, consumer impact
2. **SOLVE** — Address each sub-problem; assign confidence (0.0–1.0)
3. **VERIFY** — Check: backward compatibility, error handling, test coverage
4. **SYNTHESIZE** — Integrate solutions weighted by confidence
5. **REFLECT** — If confidence < 0.8, iterate or surface blockers

**Output**: Answer, confidence score, caveats for consumer impact.

## Build & Test

Via [`Makefile`](./Makefile) — always use `make` (ensures build-time injections):

| Target | Purpose |
|--------|---------|
| `make build` | Build executable |
| `make test-unit` | Unit tests |
| `make test-unit ARGS='-run=Test'` | Specific test |
| `make test-e2e-cli` | E2E CLI tests (requires cluster) |
| `make test-e2e-mcp` | E2E MCP tests (requires cluster + image) |
| `make lint` | Linting (`golangci-lint`) |

**E2E tests** require `KUBECONFIG` — see [`CONTRIBUTING.md`](CONTRIBUTING.md) for setup details.

## Testing

- **Coverage**: >80% for `framework/`, `api/`, `internal/resolver`
- **Deps changed?** Run `go mod tidy -v && go mod vendor`

Details (assertions, E2E setup, teardown): [`CONTRIBUTING.md`](CONTRIBUTING.md)

## PR Validation

Run before submitting: `make lint && make test-unit && make security`

Full checklist: [`CONTRIBUTING.md` § Pull Request Checklist](CONTRIBUTING.md#pull-request-checklist)

## Patterns

- **Functional Options** for extensibility (`WithVersion()`, `WithIntegrations()`)
- **Interface-Driven**: `SubCommand`: Complete → Validate → Run
- **Builder**: `TopologyBuilder` for dependency graphs
- **DI**: Services via constructors

Full reference: [`docs/architecture.md`](docs/architecture.md)

## Conventions

**Annotations** (`helmet.redhat-appstudio.github.com/`):
`product-name`, `depends-on`, `weight`, `integrations-provided`, `integrations-required`

**Filesystem**: `config.yaml`, `values.yaml.tpl` (required) | `charts/`, `instructions.md`

See [`docs/topology.md`](docs/topology.md), [`docs/installer-structure.md`](docs/installer-structure.md)

## Documentation

Detailed reference for each area lives in `docs/`. Read the relevant page when working in that area:

| Area | File | When to read |
|------|------|--------------|
| Getting started | [`docs/getting-started.md`](docs/getting-started.md) | Setting up a new installer project |
| Architecture & design | [`docs/architecture.md`](docs/architecture.md) | Component relationships, extension points |
| Installer tarball & embed | [`docs/installer-structure.md`](docs/installer-structure.md) | Embedded resources, overlay FS, tarball layout |
| Configuration system | [`docs/configuration.md`](docs/configuration.md) | Changing config schema, ConfigMap persistence, product properties |
| Dependency topology | [`docs/topology.md`](docs/topology.md) | Modifying resolver, annotations, chart ordering |
| Template engine | [`docs/templating.md`](docs/templating.md) | Editing values.yaml.tpl, adding template functions, cluster introspection |
| Integrations & products | [`docs/integrations.md`](docs/integrations.md) | Integration lifecycle, product coupling, CEL expressions |
| MCP server | [`docs/mcp.md`](docs/mcp.md) | MCP tools, container image for Jobs, instructions.md |
| Example charts | [`docs/example-charts.md`](docs/example-charts.md) | Understanding test fixtures, chart annotation examples |
| CLI reference | [`docs/cli-reference.md`](docs/cli-reference.md) | Adding custom commands, SubCommand lifecycle |
| Hook scripts | [`docs/hooks.md`](docs/hooks.md) | Pre/post deploy scripts, hook execution lifecycle |

## Git

Semantic commits: `type(scope): message` + `Assisted-by: Claude`

Types: `feat` | `fix` | `refactor` | `test` | `docs` | `chore`

**No commits unless instructed.**
