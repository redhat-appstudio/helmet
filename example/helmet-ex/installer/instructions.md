# Helmet-Ex MCP Server Instructions

## Overview

You are interacting with the Helmet-Ex example application via the Model Context Protocol (MCP).
This example demonstrates all features of the Helmet framework for building Kubernetes installers.

## Available MCP Tools

### Configuration Management

- **helmet_ex_config_create**: Create initial configuration
- **helmet_ex_config_delete**: Delete configuration
- **helmet_ex_config_get**: Retrieve current configuration
- **helmet_ex_config_update**: Update configuration (product enable/disable, properties)

### Deployment Operations

- **helmet_ex_deploy**: Deploy products to Kubernetes cluster
  - Supports dry-run mode
  - Handles dependency resolution automatically
  - Monitors installation progress

### Topology Inspection

- **helmet_ex_topology**: View chart dependency graph and installation order
  - Shows product associations
  - Displays dependency relationships
  - Indicates integration requirements

### Status Monitoring

- **helmet_ex_status**: Get deployment status
  - Shows Helm release status
  - Reports resource readiness
  - Displays integration status

### Integration Configuration

- **helmet_ex_integration_acs**: Configure Red Hat ACS
- **helmet_ex_integration_artifactory**: Configure Artifactory
- **helmet_ex_integration_azure**: Configure Azure
- **helmet_ex_integration_bitbucket**: Configure Bitbucket
- **helmet_ex_integration_github**: Configure GitHub integration
- **helmet_ex_integration_gitlab**: Configure GitLab integration
- **helmet_ex_integration_jenkins**: Configure Jenkins
- **helmet_ex_integration_nexus**: Configure Nexus
- **helmet_ex_integration_quay**: Configure Quay registry
- **helmet_ex_integration_tas**: Configure Trusted Artifact Signer
- **helmet_ex_integration_trustification**: Configure Trustification

## Workflow Patterns

### Initial Setup

1. Create configuration: `helmet_ex_config_create`
2. View topology: `helmet_ex_topology`
3. Configure required integrations (e.g., `helmet_ex_integration_github`)
4. Deploy: `helmet_ex_deploy` (or `helmet_ex_deploy --dry-run` to preview)

### Updating Configuration

1. Get current config: `helmet_ex_config_get`
2. Update products/properties: `helmet_ex_config_update`
3. Redeploy: `helmet_ex_deploy`

### Troubleshooting

1. Check status: `helmet_ex_status`
2. Review topology: `helmet_ex_topology`
3. Verify integrations: Check individual integration tools

## Example Architecture

The helmet-ex example uses a multi-layered product topology:

- **Foundation Layer**: helmet-foundation (base dependencies)
- **Infrastructure Layer**: helmet-infrastructure (depends on foundation)
- **Operators Layer**: helmet-operators (depends on foundation)
- **Storage Layer**: helmet-storage (depends on foundation)
- **Networking Layer**: helmet-networking (depends on foundation)
- **Integrations Layer**: helmet-integrations (depends on foundation)
- **Product A**: Depends on foundation, operators, infrastructure
- **Product B**: Depends on storage, networking
- **Product C**: Depends on Product A, storage
- **Product D**: Depends on Product C, integrations

## Configuration Structure

Configuration is stored in Kubernetes ConfigMaps with this schema:

```yaml
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
        catalogURL: https://example.com/catalog.yaml
        manageSubscription: true
        authProvider: oidc
```

## Best Practices

1. Always check topology before deployment to understand installation order
2. Use dry-run mode to preview changes before applying
3. Configure integrations before deploying charts that require them
4. Monitor status during and after deployment
5. Update configuration incrementally and test each change

## Error Handling

Common issues and solutions:

- **Missing Configuration**: Run `helmet_ex_config_create` first
- **Integration Required**: Configure the required integration before deployment
- **Dependency Not Met**: Check topology and ensure dependent charts are deployed
- **Namespace Conflict**: Verify unique namespaces per product in configuration
