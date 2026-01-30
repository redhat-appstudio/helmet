package installer

import _ "embed"

// InstallerTarball contains the embedded test/ directory as a tarball.
// This tarball is generated at build time from test/ directory contents
// by the Makefile target 'installer-tarball'.
//
// The tarball contains:
//   - config.yaml: Default configuration schema
//   - values.yaml.tpl: Go template for Helm values rendering
//   - charts/: All Helm charts demonstrating the framework topology
//
//go:embed installer.tar
var InstallerTarball []byte

// Instructions contains the MCP server guidance markdown for AI assistants.
// This content is served by the MCP server to provide context about
// available tools and workflow patterns.
//
//go:embed instructions.md
var Instructions string
