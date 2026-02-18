<p align="center">
    <a alt="Project quality report" href="https://goreportcard.com/report/github.com/redhat-appstudio/helmet">
        <img src="https://goreportcard.com/badge/github.com/redhat-appstudio/helmet">
    </a>
    <a alt="Latest project release" href="https://github.com/redhat-appstudio/helmet/releases/latest">
        <img src="https://img.shields.io/github/v/release/redhat-appstudio/helmet">
    </a>
</p>

# Helmet

**A framework for building Kubernetes installers with Helm**

Helmet is a reusable Go library for creating intelligent Kubernetes installers
that understand dependency relationships, manage configuration, and orchestrate
complex multi-component deployments using Helm. It is designed to be imported
into your own Go project, where it generates a complete CLI with commands for
configuration, deployment, topology inspection, and AI-assisted workflows via
the Model Context Protocol (MCP).

## Key Capabilities

- **Automatic Dependency Resolution** — chart annotations declare dependencies;
  the framework resolves installation order automatically
- **Configuration Management** — YAML-based product configuration with
  Kubernetes ConfigMap persistence
- **Template Engine** — Go templates for dynamic Helm values with cluster
  introspection
- **Integration System** — pluggable integrations for Git providers, registries,
  and external services
- **MCP Support** — built-in MCP server for AI assistant integration
- **Generated CLI** — complete CLI generated from your installer definition
- **Hook Scripts** — execute custom logic before and after chart installations
- **Monitoring** — resource readiness checks and Helm test execution

## Quick Start

Import Helmet and embed your installer resources:

```go
app, _ := framework.NewAppFromTarball(appCtx, installerTarball, cwd)
app.Run()
```

See [Getting Started](docs/getting-started.md) for a complete walkthrough.

## Installation

```bash
go get github.com/redhat-appstudio/helmet/framework
```

## Example Implementation

The [`example/helmet-ex/`](example/helmet-ex/) directory contains a complete
reference implementation demonstrating all framework features:

- Embedded installer tarball with overlay filesystem
- Standard and custom integrations (GitHub, GitLab, Quay, ACS, and more)
- MCP server with AI assistant instructions
- Multi-layer dependency topology (foundation → infrastructure → products)
- Build-time metadata injection via ldflags

See the [example README](example/helmet-ex/README.md) for build instructions,
command reference, and architecture details.

## Documentation

| Topic | Description |
|-------|-------------|
| [Getting Started](docs/getting-started.md) | First installer, prerequisites, build and run |
| [Architecture & Design](docs/architecture.md) | Component relationships, extension points, design principles |
| [Installer Structure](docs/installer-structure.md) | Tarball layout, embedded resources, overlay filesystem |
| [Configuration](docs/configuration.md) | config.yaml schema, ConfigMap persistence, product properties |
| [Dependency Topology](docs/topology.md) | Chart annotations, resolution algorithm, namespace assignment |
| [Template Engine](docs/templating.md) | values.yaml.tpl syntax, custom functions, cluster introspection |
| [Integrations](docs/integrations.md) | Integration system, product coupling, CEL expressions, custom integrations |
| [MCP Server](docs/mcp.md) | MCP tools, container image for Jobs, custom tools, instructions.md |
| [Example Charts](docs/example-charts.md) | Test chart reference, annotations, dependency graph |
| [CLI Reference](docs/cli-reference.md) | Generated commands, flags, custom commands, SubCommand lifecycle |
| [Hook Scripts](docs/hooks.md) | Pre/post deploy scripts in charts |

## Design Principles

Convention over configuration, interface-driven extensibility, API stability via
functional options, Kubernetes-native, Helm-centric. See
[Architecture & Design](docs/architecture.md) for details.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, testing, and pull
request guidelines.

## Resources

- [Project Homepage](https://github.com/redhat-appstudio/helmet)
- [Documentation](docs/)
- [Example Implementation](example/helmet-ex/)
- [Issue Tracker](https://github.com/redhat-appstudio/helmet/issues)
- [Releases](https://github.com/redhat-appstudio/helmet/releases)
