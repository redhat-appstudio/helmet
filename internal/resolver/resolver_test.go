package resolver

import (
	"os"
	"testing"

	"github.com/redhat-appstudio/helmet/api"
	"github.com/redhat-appstudio/helmet/internal/chartfs"
	"github.com/redhat-appstudio/helmet/internal/config"

	o "github.com/onsi/gomega"
)

func TestNewResolver(t *testing.T) {
	g := o.NewWithT(t)

	cfs := chartfs.New(os.DirFS("../../test"))

	installerNamespace := "test-namespace"
	cfg, err := config.NewConfigFromFile(cfs, "config.yaml", installerNamespace)
	g.Expect(err).To(o.Succeed())

	charts, err := cfs.GetAllCharts()
	g.Expect(err).To(o.Succeed())

	appCtx := api.NewAppContext("tssc")
	c, err := NewCollection(appCtx, charts)
	g.Expect(err).To(o.Succeed())

	t.Run("Resolve", func(t *testing.T) {
		topology := NewTopology()
		r := NewResolver(cfg, c, topology)
		err := r.Resolve()
		g.Expect(err).To(o.Succeed())

		// Extracting the Helm chart names and namespaces from the topology.
		dependencyNamespaceMap := map[string]string{}
		dependencySlice := []string{}
		for _, d := range topology.Dependencies() {
			dependencyNamespaceMap[d.Name()] = d.Namespace()
			dependencySlice = append(dependencySlice, d.Name())
		}
		// Showing the resolved dependencies.
		t.Logf("Resolved dependencies (%d)", len(dependencySlice))
		i := 1
		for name, ns := range dependencyNamespaceMap {
			t.Logf("(%2d) %s -> %s", i, name, ns)
			i++
		}
		g.Expect(len(dependencySlice)).To(o.Equal(10))

		// Validating the order of the resolved dependencies, as well as the
		// namespace of each dependency.
		g.Expect(dependencyNamespaceMap).To(o.Equal(map[string]string{
			"helmet-product-a":      "helmet-product-a",
			"helmet-product-b":      "helmet-product-b",
			"helmet-product-c":      "helmet-product-c",
			"helmet-product-d":      "helmet-product-d",
			"helmet-foundation":     installerNamespace,
			"helmet-operators":      installerNamespace,
			"helmet-infrastructure": installerNamespace,
			"helmet-integrations":   installerNamespace,
			"helmet-networking":     installerNamespace,
			"helmet-storage":        installerNamespace,
		}))
		g.Expect(dependencySlice).To(o.Equal([]string{
			"helmet-foundation",
			"helmet-operators",
			"helmet-infrastructure",
			"helmet-product-a",
			"helmet-storage",
			"helmet-product-b",
			"helmet-integrations",
			"helmet-networking",
			"helmet-product-c",
			"helmet-product-d",
		}))
	})
}
