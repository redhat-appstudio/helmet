package main

import (
	"fmt"
	"os"

	"github.com/redhat-appstudio/helmet/example/helmet-ex/installer"
	"github.com/redhat-appstudio/helmet/pkg/api"
	"github.com/redhat-appstudio/helmet/pkg/chartfs"
	"github.com/redhat-appstudio/helmet/pkg/framework"
	"github.com/redhat-appstudio/helmet/pkg/subcmd"
)

// Build-time variables (injected via ldflags)
var (
	version  = "v0.0.0-SNAPSHOT"
	commitID = "unknown"
)

func main() {
	// 1. Create application context with metadata
	appCtx := createAppContext()

	// 2. Create filesystem abstraction with overlay support
	cfs, err := createChartFS()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// 3. Build MCP image reference
	mcpImage := buildMCPImage()

	// 4. Create application with framework options
	app, err := framework.NewApp(
		appCtx,
		cfs,
		framework.WithIntegrations(subcmd.StandardModules()...),
		framework.WithMCPImage(mcpImage),
		framework.WithMCPToolsBuilder(subcmd.StandardMCPToolsBuilder()),
		framework.WithInstallerTarball(installer.InstallerTarball),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// 5. Run the application
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// createAppContext initializes the application context with build metadata
// and descriptive information about the example application.
func createAppContext() *api.AppContext {
	return api.NewAppContext(
		"helmet-ex",
		api.WithVersion(version),
		api.WithCommitID(commitID),
		api.WithNamespace("helmet-ex-system"),
		api.WithShortDescription("Helmet Framework Example Application"),
		api.WithLongDescription(`A comprehensive example demonstrating all Helmet framework features.

This example application showcases:
- Application context with build-time metadata injection
- Embedded tarball filesystem with overlay support for local development
- Standard integration modules (GitHub, GitLab, Quay, ACS, etc.)
- MCP server with AI assistant instructions
- Configuration management via test/config.yaml
- Template rendering via test/values.yaml.tpl
- Helm chart dependency resolution and deployment
- All framework-generated CLI commands

The example uses the test fixtures from the test/ directory,
demonstrating a multi-product topology with foundation, infrastructure,
operators, storage, networking, integrations, and product layers.`),
	)
}

// createChartFS creates a ChartFS with overlay support, allowing local
// filesystem overrides of embedded tarball content without rebuilding.
func createChartFS() (*chartfs.ChartFS, error) {
	// Create embedded tarball filesystem
	tfs, err := framework.NewTarFS(installer.InstallerTarball)
	if err != nil {
		return nil, fmt.Errorf("failed to create tarball filesystem: %w", err)
	}

	// Get current working directory for local overrides
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	// Create overlay: embedded (base) + local (override)
	// This allows local development without rebuilding the binary
	ofs := chartfs.NewOverlayFS(tfs, os.DirFS(cwd))

	return chartfs.New(ofs), nil
}

// buildMCPImage constructs the container image reference for the MCP server.
// Uses the commit ID for versioning when available, falls back to 'latest'.
func buildMCPImage() string {
	mcpImage := "quay.io/redhat-appstudio/helmet-ex"
	if commitID != "" && commitID != "unknown" {
		return fmt.Sprintf("%s:%s", mcpImage, commitID)
	}
	return fmt.Sprintf("%s:latest", mcpImage)
}
