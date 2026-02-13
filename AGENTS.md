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

**E2E tests** require a Kubernetes cluster via `KUBECONFIG`. Users provide their own cluster or use `make kind-up` / `make kind-down` for local KinD. The MCP suite requires a container image accessible to the cluster; build and push with `make image image-push IMAGE_REPOSITORY="<registry>"` using a registry the cluster can pull from.

## Testing

- **Assertions**: `github.com/onsi/gomega`
- **Coverage**: >80% for `framework/`, `api/`, `internal/resolver`
- **E2E suites**: `test/e2e/cli/` (CLI workflow), `test/e2e/mcp/` (MCP JSON-RPC workflow)
- **Deps changed?** Run `go mod tidy -v && go mod vendor`

## Critical Packages

Consumer-facing — use functional options for extensibility:

| Package | Scope |
|---------|-------|
| `framework/` | App bootstrap, CLI |
| `api/` | `SubCommand`, `IntegrationModule`, `AppContext` |

## Patterns

| Pattern | Usage |
|---------|-------|
| Functional Options | `WithVersion()`, `WithIntegrations()` |
| Interface-Driven | `SubCommand`: Complete → Validate → Run |
| Builder | `TopologyBuilder` |
| DI | Services via constructors |

## Error Handling

```go
return fmt.Errorf("failed to resolve dependencies for product %q: %w", name, err)
```

## Conventions

**Annotations** (`helmet.redhat-appstudio.github.com/`):
`product-name`, `depends-on`, `weight`, `integrations-provided`, `integrations-required`

**Filesystem**: `config.yaml`, `values.yaml.tpl` (required) | `charts/`, `instructions.md`

## Git

Semantic commits: `type(scope): message` + `Assisted-by: Claude`

Types: `feat` | `fix` | `refactor` | `test` | `docs` | `chore`

**No commits unless instructed.**
