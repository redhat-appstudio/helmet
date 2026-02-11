package e2e

import (
	"fmt"

	"helm.sh/helm/v3/pkg/action"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	// ProjectRoot is the relative path from the test/e2e/cli package to the
	// repository root. Go test runners set the working directory to the
	// package under test, so all paths must be anchored relative to it.
	ProjectRoot = "../../.."

	// BinaryPath is the path to the helmet-ex binary relative to the
	// project root.
	BinaryPath = "example/helmet-ex/helmet-ex"

	// ConfigPath is the path to the test configuration file relative to the
	// project root. Passed to the binary which opens it via io/fs (no
	// absolute paths or ".." traversals allowed).
	ConfigPath = "test/config.yaml"
)

// SharedContext holds common resources for E2E tests.
type SharedContext struct {
	KubeConfig *rest.Config
	KubeClient kubernetes.Interface
	HelmConfig *action.Configuration
	Namespace  string
}

// NewSharedContext initializes the shared E2E test context. It verifies
// KUBECONFIG is set and creates Kubernetes clients.
func NewSharedContext(namespace string) (*SharedContext, error) {
	restConfig, err := GetKubeConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	helmConfig, err := newHelmConfig(namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create helm config: %w", err)
	}

	return &SharedContext{
		KubeConfig: restConfig,
		KubeClient: kubeClient,
		HelmConfig: helmConfig,
		Namespace:  namespace,
	}, nil
}
