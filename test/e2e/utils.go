package e2e

import (
	"fmt"
	"os"
	"path/filepath"

	"helm.sh/helm/v3/pkg/action"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetKubeConfig loads kubeconfig from the KUBECONFIG environment variable or the
// default location (~/.kube/config).
func GetKubeConfig() (*rest.Config, error) {
	kubeConfig := os.Getenv("KUBECONFIG")
	if kubeConfig == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		kubeConfig = filepath.Join(home, ".kube", "config")
	}
	return clientcmd.BuildConfigFromFlags("", kubeConfig)
}

// noopLog is a no-op logger for Helm action configuration.
func noopLog(_ string, _ ...any) {}

// NewHelmConfig creates a Helm action.Configuration for the specified namespace
// using the KUBECONFIG environment variable.
func NewHelmConfig(namespace string) (*action.Configuration, error) {
	cfg := new(action.Configuration)
	kubeconfig := os.Getenv("KUBECONFIG")
	cf := genericclioptions.NewConfigFlags(false)
	cf.Namespace = &namespace
	if kubeconfig != "" {
		cf.KubeConfig = &kubeconfig
	}
	if err := cfg.Init(cf, namespace, "", noopLog); err != nil {
		return nil, err
	}
	return cfg, nil
}

// MCPTestImage returns the container image reference for the MCP server. Uses
// IMAGE environment varable if set, falls back to default.
func MCPTestImage() string {
	if img := os.Getenv("IMAGE"); img != "" {
		return img
	}
	return "localhost:5000/helmet/helmet-ex:latest"
}
