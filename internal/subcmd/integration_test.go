package subcmd

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/onsi/gomega"
	"github.com/redhat-appstudio/helmet/api"
	"github.com/redhat-appstudio/helmet/internal/annotations"
	"github.com/redhat-appstudio/helmet/internal/chartfs"
	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/constants"
	"github.com/redhat-appstudio/helmet/internal/integrations"
	"github.com/redhat-appstudio/helmet/internal/k8s"
	"github.com/redhat-appstudio/helmet/internal/runcontext"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	testAppName   = "test-app"
	testNamespace = "test-ns"
)

// loadTestConfig loads the test configuration from test/config.yaml.
// Path is relative to internal/subcmd/ (location of this test file).
func loadTestConfig(t *testing.T) *config.Config {
	t.Helper()
	g := gomega.NewWithT(t)
	payload, err := os.ReadFile("../../test/config.yaml")
	g.Expect(err).ToNot(gomega.HaveOccurred())
	cfg, err := config.NewConfigFromBytes(payload, testNamespace)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	return cfg
}

func testAppContext() *api.AppContext {
	return &api.AppContext{
		Name:      testAppName,
		Namespace: testNamespace,
	}
}

func configMapForConfig(cfg *config.Config) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testAppName + "-config",
			Namespace: cfg.Namespace(),
			Labels: map[string]string{
				annotations.Config: "true",
			},
		},
		Data: map[string]string{
			constants.ConfigFilename: cfg.String(),
		},
	}
}

// integrationSecret creates a fake integration secret using the naming
// convention from internal/integration: {appName}-{integrationName}-integration.
func integrationSecret(integrationName string) *corev1.Secret {
	name := fmt.Sprintf("%s-%s-integration", testAppName, integrationName)
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{"key": []byte("value")},
	}
}

func testChartFS(t *testing.T) *chartfs.ChartFS {
	t.Helper()
	return chartfs.New(os.DirFS("../../test/charts"))
}

// testRunContext creates a RunContext with FakeKube seeded with the given
// objects, and the test charts filesystem.
func testRunContext(
	t *testing.T,
	objects ...runtime.Object,
) *runcontext.RunContext {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	fakeKube := k8s.NewFakeKube(objects...)
	return runcontext.NewRunContext(fakeKube, testChartFS(t), logger)
}

// testManager creates and loads an integrations Manager with all standard
// modules.
func testManager(
	t *testing.T,
	runCtx *runcontext.RunContext,
) *integrations.Manager {
	t.Helper()
	g := gomega.NewWithT(t)
	manager := integrations.NewManager()
	err := manager.LoadModules(testAppName, runCtx, StandardModules())
	g.Expect(err).ToNot(gomega.HaveOccurred())
	return manager
}

// findChild finds a child command by name in the integration command.
func findChild(cmd *cobra.Command, name string) *cobra.Command {
	for _, child := range cmd.Commands() {
		if child.Name() == name {
			return child
		}
	}
	return nil
}

// TestNewIntegration_CommandNameInvariant verifies every child command's Name()
// matches the IntegrationModule.Name for every registered module.
func TestNewIntegration_CommandNameInvariant(t *testing.T) {
	g := gomega.NewWithT(t)

	appCtx := testAppContext()
	runCtx := testRunContext(t)
	manager := testManager(t, runCtx)
	cmd := NewIntegration(appCtx, runCtx, manager)

	// Build a set of expected integration names from modules.
	moduleNames := map[string]bool{}
	for _, mod := range StandardModules() {
		moduleNames[mod.Name] = true
	}

	// Every child command name must be present in moduleNames.
	for _, child := range cmd.Commands() {
		g.Expect(moduleNames).To(gomega.HaveKey(child.Name()),
			"child command %q does not match any module name", child.Name())
	}

	// Every module must have a corresponding child command.
	for name := range moduleNames {
		g.Expect(findChild(cmd, name)).ToNot(gomega.BeNil(),
			"no child command found for module %q", name)
	}
}

// TestNewIntegration_TASAliasPreserved verifies the TAS command is renamed to
// "tas" and "trusted-artifact-signer" is kept as an alias.
func TestNewIntegration_TASAliasPreserved(t *testing.T) {
	g := gomega.NewWithT(t)

	appCtx := testAppContext()
	runCtx := testRunContext(t)
	manager := testManager(t, runCtx)
	cmd := NewIntegration(appCtx, runCtx, manager)

	tasCmd := findChild(cmd, "tas")
	g.Expect(tasCmd).ToNot(gomega.BeNil(), "child command 'tas' not found")
	g.Expect(tasCmd.Aliases).To(gomega.ContainElement("trusted-artifact-signer"))
}

// TestDisableProductForIntegration_ScopedDisablement verifies that only Product A
// (which provides acs) is disabled, while Products B and C (which provide quay
// and nexus) remain enabled, even though their integration secrets also exist.
func TestDisableProductForIntegration_ScopedDisablement(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()

	cfg := loadTestConfig(t)
	runCtx := testRunContext(t,
		configMapForConfig(cfg),
		integrationSecret("acs"),
		integrationSecret("quay"),
		integrationSecret("nexus"),
	)
	manager := testManager(t, runCtx)
	appCtx := testAppContext()

	err := disableProductForIntegration(
		ctx, appCtx, runCtx, manager, cfg, integrations.ACS)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	productA, err := cfg.GetProduct("Product A")
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(productA.Enabled).To(gomega.BeFalse(),
		"Product A should be disabled (provides acs)")

	productB, err := cfg.GetProduct("Product B")
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(productB.Enabled).To(gomega.BeTrue(),
		"Product B should remain enabled (provides quay, not touched)")

	productC, err := cfg.GetProduct("Product C")
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(productC.Enabled).To(gomega.BeTrue(),
		"Product C should remain enabled (provides nexus, not touched)")
}

// TestDisableProductForIntegration_NoProvidingProduct verifies that when no
// product provides the active integration (artifactory), the config is unchanged.
func TestDisableProductForIntegration_NoProvidingProduct(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()

	cfg := loadTestConfig(t)
	runCtx := testRunContext(t, integrationSecret("artifactory"))
	manager := testManager(t, runCtx)
	appCtx := testAppContext()

	err := disableProductForIntegration(
		ctx, appCtx, runCtx, manager, cfg, integrations.Artifactory)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	for _, name := range []string{
		"Product A",
		"Product B",
		"Product C",
		"Product D",
	} {
		spec, err := cfg.GetProduct(name)
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(spec.Enabled).To(gomega.BeTrue(),
			"%s should remain enabled", name)
	}
}

// TestDisableProductForIntegration_SecretNotCreated verifies that when the active
// integration's secret doesn't exist, the config is unchanged.
func TestDisableProductForIntegration_SecretNotCreated(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()

	cfg := loadTestConfig(t)
	// No integration secrets seeded.
	runCtx := testRunContext(t)
	manager := testManager(t, runCtx)
	appCtx := testAppContext()

	err := disableProductForIntegration(
		ctx, appCtx, runCtx, manager, cfg, integrations.ACS)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	productA, err := cfg.GetProduct("Product A")
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(productA.Enabled).To(gomega.BeTrue(),
		"Product A should remain enabled (acs secret not created)")
}

// TestDisableProductForIntegration_AlreadyDisabled verifies that when the product
// is already disabled, no redundant update is performed.
func TestDisableProductForIntegration_AlreadyDisabled(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()

	cfg := loadTestConfig(t)

	// Disable Product A before calling.
	productA, err := cfg.GetProduct("Product A")
	g.Expect(err).ToNot(gomega.HaveOccurred())
	productA.Enabled = false
	err = cfg.SetProduct("Product A", *productA)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	runCtx := testRunContext(t, integrationSecret("acs"))
	manager := testManager(t, runCtx)
	appCtx := testAppContext()

	err = disableProductForIntegration(
		ctx, appCtx, runCtx, manager, cfg, integrations.ACS)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	updatedA, err := cfg.GetProduct("Product A")
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(updatedA.Enabled).To(gomega.BeFalse(),
		"Product A should remain disabled")

	productB, err := cfg.GetProduct("Product B")
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(productB.Enabled).To(gomega.BeTrue(),
		"Product B should remain enabled")
}
