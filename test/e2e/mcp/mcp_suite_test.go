package mcp_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/redhat-appstudio/helmet/test/e2e"
)

var (
	sharedCtx *e2e.SharedContext
	runner    *e2e.Runner
	client    *e2e.MCPClient
)

func TestMCP(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E MCP Suite")
}

var _ = BeforeSuite(func(ctx context.Context) {
	var err error

	By("initializing shared E2E context")
	sharedCtx, err = e2e.NewSharedContext("helmet-ex-system")
	Expect(err).NotTo(HaveOccurred())

	By("creating CLI runner (for integration commands)")
	runner, err = e2e.NewRunner(
		e2e.ProjectRoot,
		e2e.BinaryPath,
		e2e.ConfigPath,
		sharedCtx.Namespace,
	)
	Expect(err).NotTo(HaveOccurred())

	// Use context.Background for the MCP server subprocess: Ginkgo cancels the
	// BeforeSuite ctx when this node completes, but the server must survive until
	// AfterSuite calls Shutdown.
	By("starting MCP server subprocess via Runner")
	client, err = runner.StartMCPServer(context.Background(), e2e.MCPTestImage())
	Expect(err).NotTo(HaveOccurred())

	By("performing MCP initialize handshake")
	Expect(client.Initialize(ctx)).To(Succeed())

	By("verifying all 13 tools are registered")
	tools, err := client.ListTools(ctx)
	Expect(err).NotTo(HaveOccurred())
	Expect(tools).To(HaveLen(13))
})

var _ = AfterSuite(func() {
	if client != nil {
		_ = client.Shutdown()
	}
})
