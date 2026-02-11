package cli_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/redhat-appstudio/helmet/test/e2e"
)

var (
	sharedCtx       *e2e.SharedContext
	runner          *e2e.Runner
	configChecker   *e2e.ConfigChecker
	secretsChecker  *e2e.SecretsChecker
	releasesChecker *e2e.ReleasesChecker
)

func TestCLI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E CLI Suite")
}

var _ = BeforeSuite(func(ctx context.Context) {
	var err error

	By("initializing shared E2E context")
	sharedCtx, err = e2e.NewSharedContext("helmet-ex-system")
	Expect(err).NotTo(HaveOccurred())

	By("creating CLI runner")
	runner, err = e2e.NewRunner(
		e2e.ProjectRoot,
		e2e.BinaryPath,
		e2e.ConfigPath,
		sharedCtx.Namespace,
	)
	Expect(err).NotTo(HaveOccurred())

	By("creating checkers")
	configChecker = e2e.NewConfigChecker(
		sharedCtx.KubeClient,
		sharedCtx.Namespace,
		"helmet-ex",
	)
	secretsChecker = e2e.NewSecretsChecker(
		sharedCtx.KubeClient,
		sharedCtx.Namespace,
		[]string{
			"helmet-ex-quay-integration",
			"helmet-ex-acs-integration",
			"helmet-ex-nexus-integration",
			"helmet-ex-artifactory-integration",
		},
	)
	// Infrastructure releases deployed in helmet-ex-system. Products that
	// provide integrations (A→acs, B→quay, C→nexus) are disabled by the
	// integration commands, so only Product D (in its own namespace) and
	// the shared infrastructure charts are deployed. Product D is not
	// checked here because it lands in namespace "helmet-product-d".
	releasesChecker = e2e.NewReleasesChecker(
		sharedCtx.HelmConfig,
		sharedCtx.KubeClient,
		sharedCtx.Namespace,
		[]string{
			"helmet-foundation",
			"helmet-operators",
			"helmet-networking",
			"helmet-infrastructure",
			"helmet-storage",
		},
	)
})
