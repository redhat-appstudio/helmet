package config

import (
	"os"
	"testing"

	"github.com/redhat-appstudio/helmet/internal/chartfs"

	o "github.com/onsi/gomega"
)

func TestMergeDistributedInstallerYAML(t *testing.T) {
	g := o.NewWithT(t)
	cfs := chartfs.New(os.DirFS("testdata/merge-distributed"))

	out, err := MergeDistributedInstallerYAML(cfs, "tssc")
	g.Expect(err).To(o.Succeed())
	g.Expect(string(out)).To(o.ContainSubstring("tssc:"))
	// Default merged products must not emit enabled: false (listing in blueprint = active).
	g.Expect(string(out)).NotTo(o.ContainSubstring("enabled: false"))
	g.Expect(string(out)).To(o.ContainSubstring("Red Hat Quay"))
	g.Expect(string(out)).To(o.ContainSubstring("Advanced Cluster Security"))

	cfg, err := NewConfigFromBytes(out, "tssc", "tssc")
	g.Expect(err).To(o.Succeed())
	g.Expect(cfg.Installer.Integrations).To(o.HaveLen(1))
	g.Expect(cfg.Installer.Integrations[0].Integration).To(o.Equal("quay"))
}
