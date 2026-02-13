package mcp_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/redhat-appstudio/helmet/test/e2e"
)

// configYAMLProducts extracts product entries from a config.yaml string.
// Returns a map keyed by product name with each product's attributes.
func configYAMLProducts(data string) map[string]map[string]any {
	var raw map[string]any
	Expect(yaml.Unmarshal([]byte(data), &raw)).To(Succeed())

	tssc, ok := raw["tssc"].(map[string]any)
	Expect(ok).To(BeTrue(), "missing tssc top-level key")

	productsList, ok := tssc["products"].([]any)
	Expect(ok).To(BeTrue(), "missing tssc.products list")

	products := make(map[string]map[string]any, len(productsList))
	for _, p := range productsList {
		pm, ok := p.(map[string]any)
		Expect(ok).To(BeTrue(), "product entry is not a map")
		name, ok := pm["name"].(string)
		Expect(ok).To(BeTrue(), "product missing name field")
		products[name] = pm
	}
	return products
}

// phaseConfiguration initializes cluster config, applies mutations (settings,
// product enable/disable, namespace, properties), and verifies the resulting
// ConfigMap state matches all mutations.
func phaseConfiguration(
	ctx context.Context,
	mc *e2e.MCPClient,
	r *e2e.Runner,
	sc *e2e.SharedContext,
) {
	By("cleaning up previous config (if any)")
	r.ConfigDelete(ctx)

	By("checking initial status reports AWAITING_CONFIGURATION")
	result := mc.CallTool(ctx, "helmet_ex_status", nil)
	Expect(result.Text()).To(ContainSubstring("AWAITING_CONFIGURATION"))

	By("retrieving default configuration via MCP")
	result = mc.CallTool(ctx, "helmet_ex_config_get", nil)
	Expect(result.Text()).To(ContainSubstring("Product A"))
	Expect(result.Text()).To(ContainSubstring("Product B"))
	Expect(result.Text()).To(ContainSubstring("Product C"))
	Expect(result.Text()).To(ContainSubstring("Product D"))

	By("initializing cluster configuration via MCP")
	result = mc.CallTool(ctx, "helmet_ex_config_init",
		map[string]any{"namespace": "helmet-ex-system"})
	Expect(result.IsError).To(BeFalse(),
		"config_init failed: %s", result.Text())

	By("verifying status after config_init is not AWAITING_CONFIGURATION")
	result = mc.CallTool(ctx, "helmet_ex_status", nil)
	Expect(result.Text()).NotTo(ContainSubstring("AWAITING_CONFIGURATION"),
		"status should advance past AWAITING_CONFIGURATION after config_init")

	// ── Config Mutations ────────────────────────────────────────

	By("mutating settings: crc=true")
	result = mc.CallTool(ctx, "helmet_ex_config_settings",
		map[string]any{"key": "crc", "value": true})
	Expect(result.IsError).To(BeFalse(),
		"config_settings failed: %s", result.Text())

	By("verifying crc=true via config_get")
	result = mc.CallTool(ctx, "helmet_ex_config_get", nil)
	Expect(result.Text()).To(ContainSubstring("crc: true"))

	By("disabling Product D")
	result = mc.CallTool(ctx, "helmet_ex_config_product_enabled",
		map[string]any{"name": "Product D", "enabled": false})
	Expect(result.IsError).To(BeFalse(),
		"config_product_enabled failed: %s", result.Text())

	By("verifying Product D disabled via config_get")
	result = mc.CallTool(ctx, "helmet_ex_config_get", nil)
	Expect(result.Text()).To(ContainSubstring("Product D"))

	By("changing Product C namespace to custom-ns-c")
	result = mc.CallTool(ctx, "helmet_ex_config_product_namespace",
		map[string]any{"name": "Product C", "namespace": "custom-ns-c"})
	Expect(result.IsError).To(BeFalse(),
		"config_product_namespace failed: %s", result.Text())

	By("verifying Product C namespace via config_get")
	result = mc.CallTool(ctx, "helmet_ex_config_get", nil)
	Expect(result.Text()).To(ContainSubstring("custom-ns-c"))

	By("updating properties on Product B (storageClass → fast)")
	result = mc.CallTool(ctx, "helmet_ex_config_product_properties",
		map[string]any{
			"name":       "Product B",
			"properties": map[string]any{"storageClass": "fast"},
		})
	Expect(result.IsError).To(BeFalse(),
		"config_product_properties failed: %s", result.Text())

	By("verifying Product B properties via config_get")
	result = mc.CallTool(ctx, "helmet_ex_config_get", nil)
	Expect(result.Text()).To(ContainSubstring("storageClass: fast"))

	// ── ConfigMap Verification (Fail-Fast) ─────────────────────

	By("verifying cluster ConfigMap reflects all mutations")
	cm, err := sc.KubeClient.CoreV1().ConfigMaps("helmet-ex-system").
		Get(ctx, "helmet-ex-config", metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())

	configYAML := cm.Data["config.yaml"]
	Expect(configYAML).To(ContainSubstring("crc: true"))

	products := configYAMLProducts(configYAML)

	// Product D: enabled: false
	productD, ok := products["Product D"]
	Expect(ok).To(BeTrue(), "Product D not found in ConfigMap")
	Expect(productD["enabled"]).To(BeFalse(),
		"Product D should be disabled")

	// Product C: namespace: custom-ns-c
	productC, ok := products["Product C"]
	Expect(ok).To(BeTrue(), "Product C not found in ConfigMap")
	Expect(productC["namespace"]).To(Equal("custom-ns-c"),
		"Product C namespace mismatch")

	// Product B: storageClass updated to "fast"
	productB, ok := products["Product B"]
	Expect(ok).To(BeTrue(), "Product B not found in ConfigMap")
	props, ok := productB["properties"].(map[string]any)
	Expect(ok).To(BeTrue(), "Product B has no properties map")
	Expect(props["storageClass"]).To(Equal("fast"),
		"Product B storageClass should be updated to fast")

	By("verifying status after mutations is not AWAITING_CONFIGURATION")
	result = mc.CallTool(ctx, "helmet_ex_status", nil)
	Expect(result.Text()).NotTo(ContainSubstring("AWAITING_CONFIGURATION"),
		"status should not regress to AWAITING_CONFIGURATION after mutations")
}

// phaseIntegrations lists available integrations, scaffolds them, configures
// each via CLI, and verifies their status reports Configured.
func phaseIntegrations(ctx context.Context, mc *e2e.MCPClient, r *e2e.Runner) {
	By("listing available integrations via MCP")
	result := mc.CallTool(ctx, "helmet_ex_integration_list", nil)
	Expect(result.IsError).To(BeFalse())
	Expect(result.Text()).To(ContainSubstring("acs"))
	Expect(result.Text()).To(ContainSubstring("quay"))

	By("scaffolding integration commands via MCP")
	result = mc.CallTool(ctx, "helmet_ex_integration_scaffold",
		map[string]any{"names": []string{"acs", "quay"}})
	Expect(result.Text()).To(ContainSubstring("OVERWRITE_ME"))

	By("configuring acs integration via CLI")
	Expect(r.Integration(ctx, "acs",
		"--force",
		"--endpoint=acs.test.local:443",
		"--token=test-token",
	)).To(Succeed())

	By("configuring quay integration via CLI")
	Expect(r.Integration(ctx, "quay",
		"--force",
		"--url=https://quay.io",
		"--token=test-token",
		"--organization=test-org",
	)).To(Succeed())

	By("verifying integration status via MCP")
	result = mc.CallTool(ctx, "helmet_ex_integration_status",
		map[string]any{"names": []string{"acs", "quay"}})
	Expect(result.Text()).To(ContainSubstring("Configured"))
}

// phaseReadyToDeploy asserts the system reached a deployable state and inspects
// the deployment topology.
func phaseReadyToDeploy(ctx context.Context, mc *e2e.MCPClient) {
	By("verifying status is READY_TO_DEPLOY or COMPLETED")
	result := mc.CallTool(ctx, "helmet_ex_status", nil)
	Expect(result.Text()).To(SatisfyAny(
		ContainSubstring("READY_TO_DEPLOY"),
		ContainSubstring("COMPLETED"),
	), "expected deployable status, got: %s", result.Text())

	By("viewing topology via MCP")
	result = mc.CallTool(ctx, "helmet_ex_topology", nil)
	Expect(result.IsError).To(BeFalse(),
		"topology failed: %s", result.Text())
}

// phaseDeploy triggers deployment and polls status until COMPLETED.
func phaseDeploy(ctx context.Context, mc *e2e.MCPClient) {
	By("deploying via MCP (dry-run=false, force=true)")
	result := mc.CallTool(ctx, "helmet_ex_deploy",
		map[string]any{"dry-run": false, "force": true})
	Expect(result.IsError).To(BeFalse(),
		"deploy failed: %s", result.Text())

	By("verifying status transitions to DEPLOYING or COMPLETED")
	result = mc.CallTool(ctx, "helmet_ex_status", nil)
	Expect(result.Text()).To(SatisfyAny(
		ContainSubstring("DEPLOYING"),
		ContainSubstring("COMPLETED"),
	), "expected deploy-phase status, got: %s", result.Text())

	By("polling status until COMPLETED")
	Eventually(ctx, func() string {
		r := mc.CallTool(ctx, "helmet_ex_status", nil)
		return r.Text()
	}).WithPolling(5 * time.Second).
		WithTimeout(300 * time.Second).
		Should(ContainSubstring("COMPLETED"))
}

// phasePostDeployValidation checks Helm releases are present and healthy, and
// retrieves product notes.
func phasePostDeployValidation(
	ctx context.Context,
	mc *e2e.MCPClient,
	sc *e2e.SharedContext,
) {
	By("validating Helm releases in helmet-ex-system namespace")
	expectedReleases := []string{
		"helmet-foundation",
		"helmet-operators",
		"helmet-networking",
		"helmet-infrastructure",
		"helmet-storage",
	}
	Eventually(ctx, func() error {
		listAction := action.NewList(sc.HelmConfig)
		listAction.All = true
		releases, err := listAction.Run()
		if err != nil {
			return fmt.Errorf("failed to list helm releases: %w", err)
		}

		deployed := make(map[string]bool, len(releases))
		for _, rel := range releases {
			if rel.Info.Status == release.StatusDeployed {
				deployed[rel.Name] = true
			}
		}

		var missing []string
		for _, name := range expectedReleases {
			if !deployed[name] {
				missing = append(missing, name)
			}
		}
		if len(missing) > 0 {
			return fmt.Errorf("missing or not deployed: %v", missing)
		}
		return nil
	}).WithPolling(5 * time.Second).
		WithTimeout(15 * 5 * time.Second).
		Should(Succeed())

	By("retrieving product notes via MCP")
	result := mc.CallTool(ctx, "helmet_ex_notes",
		map[string]any{"name": "Product A"})
	Expect(result.Text()).NotTo(BeEmpty(),
		"notes tool should return content")
}

var _ = Describe("MCP Installer Workflow", func() {
	It("executes complete workflow via JSON-RPC and validates cluster state",
		func(ctx context.Context) {
			phaseConfiguration(ctx, client, runner, sharedCtx)
			phaseIntegrations(ctx, client, runner)
			phaseReadyToDeploy(ctx, client)
			phaseDeploy(ctx, client)
			phasePostDeployValidation(ctx, client, sharedCtx)
		})
})
