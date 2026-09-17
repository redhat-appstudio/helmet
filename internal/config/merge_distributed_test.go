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
	g.Expect(string(out)).To(o.ContainSubstring("Advanced Cluster Security"))

	cfg, err := NewConfigFromBytes(out, "tssc", "tssc")
	g.Expect(err).To(o.Succeed())
	g.Expect(cfg.Installer.Products).To(o.HaveLen(1))
	g.Expect(cfg.Installer.Products[0].Name).To(o.Equal("Advanced Cluster Security"))
}

func TestMergeDistributedRejectsIntegrationOnlyUnderProducts(t *testing.T) {
	g := o.NewWithT(t)
	cfs := chartfs.New(os.DirFS("testdata/merge-distributed-invalid-quay-product"))

	_, err := MergeDistributedInstallerYAML(cfs, "tssc")
	g.Expect(err).To(o.HaveOccurred())
	g.Expect(err.Error()).To(o.ContainSubstring("does not support a product bundle"))
	g.Expect(err.Error()).To(o.ContainSubstring("bundle-types-supported"))
}

func TestMapAtDotPath(t *testing.T) {
	g := o.NewWithT(t)
	root := map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": map[string]interface{}{"k": "v"},
			},
		},
	}
	m, err := mapAtDotPath(root, "a.b.c")
	g.Expect(err).To(o.Succeed())
	g.Expect(m).To(o.HaveKeyWithValue("k", "v"))
	_, err = mapAtDotPath(root, "a.missing")
	g.Expect(err).To(o.HaveOccurred())
}
