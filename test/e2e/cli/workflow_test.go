package cli_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Installer Workflow", func() {
	It("executes complete workflow and validates cluster state", func(ctx context.Context) {
		By("removing previous cluster configuration (if any)")
		runner.ConfigDelete(ctx)

		By("creating cluster configuration")
		Expect(runner.ConfigCreate(ctx)).To(Succeed())

		By("validating configuration ConfigMap")
		configResult := configChecker.Check(ctx)
		Expect(configResult.Passed).To(BeTrue(), configResult.Message)

		By("configuring quay integration")
		Expect(runner.Integration(ctx, "quay",
			"--force",
			"--url=https://quay.io",
			"--token=test-token",
			"--organization=test-org",
		)).To(Succeed())

		By("configuring acs integration")
		Expect(runner.Integration(ctx, "acs",
			"--force",
			"--endpoint=acs.test.local:443",
			"--token=test-token",
		)).To(Succeed())

		By("configuring nexus integration")
		Expect(runner.Integration(ctx, "nexus",
			"--force",
			"--url=https://nexus.test.local",
			"--token=test-token",
		)).To(Succeed())

		By("configuring artifactory integration")
		Expect(runner.Integration(ctx, "artifactory",
			"--force",
			"--url=https://artifactory.test.local",
			"--token=test-token",
		)).To(Succeed())

		By("validating integration Secrets")
		secretsResult := secretsChecker.Check(ctx)
		Expect(secretsResult.Passed).To(BeTrue(), secretsResult.Message)

		By("viewing topology")
		Expect(runner.Topology(ctx)).To(Succeed())

		By("deploying charts")
		Expect(runner.Deploy(ctx)).To(Succeed())

		By("validating Helm releases in topology order (15 attempts, 5s interval)")
		Eventually(ctx, func(ctx context.Context) error {
			result := releasesChecker.Check(ctx)
			if !result.Passed {
				return fmt.Errorf("releases check failed: %s", result.Message)
			}
			return nil
		}).WithPolling(5 * time.Second).WithTimeout(15 * 5 * time.Second).
			Should(Succeed())
	})
})
