Contributing to Helmet Framework
--------------------------------

Helmet is a **reusable Go library** for building Helm-based installers that deploy workloads to Kubernetes clusters. The repository provides the `framework/`, `api/`, and `internal/` packages that consumers import to build their own installer binaries.

The `example/helmet-ex/` directory contains a reference application (`helmet-ex`) that demonstrates how to use the framework. This example serves as both documentation for consumers and the test fixture for the repository's CI pipeline. All build, test, and image targets in the [Makefile](Makefile) operate against this example app.

This project also provides an [`AGENTS.md`](AGENTS.md) file with context for AI-powered code assistants.

Detailed documentation for framework internals lives in [`docs/`](docs/). See the [README](README.md) documentation table for a full index of topic pages.

# Prerequisites

- [Go 1.25 or higher][golang]
- [GNU Make][gnuMake]
- [GNU Tar][gnuTar] (macOS: `brew install gnu-tar`)
- [Docker][docker] or [Podman][podman] (for container images)

# Building

Build the example application with:

```bash
make
```

This compiles `example/helmet-ex/helmet-ex`, automatically packaging the installer resources (`example/helmet-ex/installer/`) into an embedded tarball. The tarball (`installer.tar`) is excluded from version control and regenerated whenever its source files change.

To run the example application:

```bash
make run ARGS='deploy --help'
```

## Container Image

Build and push the container image using the `image` and `image-push` targets. Point `IMAGE_REPOSITORY` at a registry your Kubernetes cluster can pull from:

```bash
make image image-push IMAGE_REPOSITORY="my-registry.example.com:5000"
```

The full image reference is composed from several variables:

| Variable           | Default          | Purpose                                   |
| ------------------ | ---------------- | ----------------------------------------- |
| `IMAGE_REPOSITORY` | `localhost:5000` | Registry host and port                    |
| `IMAGE_NAMESPACE`  | `helmet`         | Image namespace/org                       |
| `IMAGE_TAG`        | `$(COMMIT_ID)`   | Image tag (defaults to current git SHA)   |
| `CONTAINER_CLI`    | `docker`         | Container build tool (`docker`, `podman`) |

These combine into `IMAGE = $IMAGE_REPOSITORY/$IMAGE_NAMESPACE/helmet-ex:$IMAGE_TAG`. With defaults, the image resolves to `localhost:5000/helmet/helmet-ex:<commit>`.

# Testing

## Unit Tests

Unit tests cover the core library packages (`api/`, `framework/`, `internal/`) and the E2E test helpers (`test/e2e/`). Assertions use [`gomega`][gomega]:

```bash
make test-unit
```

To run a specific test:

```bash
make test-unit ARGS='-run=TestName'
```

## E2E Tests

End-to-end tests exercise the full installer workflow against a live Kubernetes cluster using [Ginkgo v2][ginkgo]. Both suites run against the `helmet-ex` example application:

- **CLI** (`test/e2e/cli/`): drives the workflow via the `helmet-ex` CLI binary, validating config-create, integration, topology, deploy, and release-checking.
- **MCP** (`test/e2e/mcp/`): drives the workflow via JSON-RPC 2.0 tool calls over STDIO, exercising all 13 MCP tools across configuration, integration, deployment, and post-deploy validation phases.

### Prerequisites

Both suites require a Kubernetes cluster accessible via `KUBECONFIG`. You can use any cluster you have available, or create a local [KinD][kind] cluster:

```bash
make kind-up       # creates KinD cluster + local registry at localhost:5000
```

The MCP suite additionally requires a container image pushed to a registry the cluster can pull from (see [Container Image](#container-image)):

```bash
make image image-push IMAGE_REPOSITORY="my-registry.example.com:5000"
```

### Running

```bash
make test-e2e-cli                    # CLI workflow suite
make test-e2e-mcp                    # MCP workflow suite (requires image)
```

When using a non-default registry, pass the same `IMAGE_REPOSITORY` so the test knows where to find the image:

```bash
make test-e2e-mcp IMAGE_REPOSITORY="my-registry.example.com:5000"
```

### Teardown

```bash
make kind-down     # deletes KinD cluster and local registry
```

## Linting

Static analysis uses [`golangci-lint`][golangciLint] (version pinned in `go.mod` via the `tool` directive):

```bash
make lint
```

## Security

Vulnerability scanning uses [`govulncheck`][govulncheck] to check dependencies for known CVEs:

```bash
make security
```

# Pull Request Checklist

Before submitting a PR, verify:

- [ ] `make lint` passes
- [ ] `make test-unit` passes
- [ ] `make security` passes (runs `govulncheck`)
- [ ] New or changed public API has test coverage
- [ ] If the PR touches `api/` types, `framework/options.go`, `internal/annotations/`, `internal/constants/`, or build/test Makefile targets: review [`AGENTS.md`](AGENTS.md) and update if affected content changed
- [ ] If the PR adds, removes, or renames a `docs/` page: update the documentation tables in both [`README.md`](README.md) and [`AGENTS.md`](AGENTS.md)
- [ ] If the PR modifies framework behavior documented in `docs/`: update the relevant `docs/` page to match

[docker]: https://docs.docker.com/get-docker
[ginkgo]: https://onsi.github.io/ginkgo
[gnuMake]: https://www.gnu.org/software/make
[gnuTar]: https://www.gnu.org/software/tar
[golang]: https://golang.org/dl
[golangciLint]: https://golangci-lint.run
[gomega]: https://onsi.github.io/gomega
[govulncheck]: https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck
[kind]: https://kind.sigs.k8s.io
[podman]: https://podman.io
