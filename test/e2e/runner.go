package e2e

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Runner executes helmet-ex CLI commands in a subprocess. All paths (binary,
// config) are relative to the project root. The binary is launched with its
// working directory set to the project root so that the embedded ChartFS
// (io/fs based) can resolve paths correctly.
type Runner struct {
	projectRoot string
	binaryPath  string
	configPath  string
	namespace   string
}

// run executes the helmet-ex binary with the specified arguments, capturing
// stdout/stderr for debugging. The child process working directory is set to
// the project root.
func (r *Runner) run(ctx context.Context, args ...string) error {
	bin := filepath.Join(r.projectRoot, r.binaryPath)
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = r.projectRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf(
			"command %q failed: %w\nstdout: %s\nstderr: %s",
			cmd.String(), err, stdout.String(), stderr.String(),
		)
	}
	return nil
}

// ConfigDelete executes: "helmet-ex config --delete".
// Errors are ignored so it can be called even when no config exists yet.
func (r *Runner) ConfigDelete(ctx context.Context) {
	_ = r.run(ctx, "config", "--delete")
}

// ConfigCreate executes: "helmet-ex config --create --namespace <ns>
// <configPath>".
func (r *Runner) ConfigCreate(ctx context.Context) error {
	return r.run(ctx,
		"config", "--create",
		"--namespace", r.namespace,
		r.configPath,
	)
}

// Integration executes: helmet-ex integration <module> <flags...>.
func (r *Runner) Integration(
	ctx context.Context,
	module string,
	flags ...string,
) error {
	args := append([]string{"integration", module}, flags...)
	return r.run(ctx, args...)
}

// Topology executes: "helmet-ex topology".
func (r *Runner) Topology(ctx context.Context) error {
	return r.run(ctx, "topology")
}

// Deploy executes: "helmet-ex deploy".
func (r *Runner) Deploy(ctx context.Context) error {
	return r.run(ctx, "deploy")
}

// NewRunner creates a new CLI command runner. The projectRoot is used as the
// working directory for the child process; it is resolved to an absolute path
// so the runner works regardless of where the test binary executes.
func NewRunner(projectRoot, binaryPath, configPath, namespace string) (*Runner, error) {
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to resolve project root %q: %w", projectRoot, err)
	}
	return &Runner{
		projectRoot: absRoot,
		binaryPath:  binaryPath,
		configPath:  configPath,
		namespace:   namespace,
	}, nil
}
