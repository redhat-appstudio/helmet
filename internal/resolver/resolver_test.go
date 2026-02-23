package resolver

import (
	"os"
	"testing"

	"github.com/redhat-appstudio/helmet/api"
	"github.com/redhat-appstudio/helmet/internal/chartfs"
	"github.com/redhat-appstudio/helmet/internal/config"

	o "github.com/onsi/gomega"
)

// resolveTopology creates a new Topology and resolves it using the provided
// config and collection. Each sub-test calling this gets an independent topology.
func resolveTopology(
	g o.Gomega,
	cfg *config.Config,
	c *Collection,
) *Topology {
	topology := NewTopology()
	r := NewResolver(cfg, c, topology)
	err := r.Resolve()
	g.Expect(err).To(o.Succeed())
	return topology
}

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
		topology := resolveTopology(g, cfg, c)

		// Extracting the Helm chart names and namespaces from the topology.
		deps := topology.Dependencies()
		dependencyNamespaceMap := map[string]string{}
		dependencySlice := make([]string, 0, len(deps))
		for _, d := range deps {
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
			"helmet-networking",
			"helmet-product-c",
			"helmet-infrastructure",
			"helmet-product-a",
			"helmet-storage",
			"helmet-product-b",
			"helmet-integrations",
			"helmet-product-d",
		}))
	})

	t.Run("Inspect", func(t *testing.T) {
		topology := resolveTopology(g, cfg, c)

		// Construct the CEL environment with the integration names used
		// by the test charts.
		cel, err := NewCEL("acs", "quay", "nexus")
		g.Expect(err).To(o.Succeed())

		// All integrations start unconfigured (no cluster secrets).
		i := &Integrations{
			configured: map[string]bool{
				"acs":   false,
				"quay":  false,
				"nexus": false,
			},
			cel: cel,
		}

		// Inspect the resolved topology — after the fix this passes;
		// before the fix it fails on helmet-product-c requiring "acs"
		// because helmet-product-a (which provides "acs") appears later
		// in the topology walk.
		err = i.Inspect(topology)
		g.Expect(err).To(o.Succeed())
	})

	t.Run("Inspect/missing integration", func(t *testing.T) {
		topology := resolveTopology(g, cfg, c)

		cel, err := NewCEL("acs", "quay", "nexus")
		g.Expect(err).To(o.Succeed())

		// All integrations unconfigured, and we build a topology where
		// the provider of "acs" is absent so the requirement fails.
		i := &Integrations{
			configured: map[string]bool{
				"acs":   false,
				"quay":  false,
				"nexus": false,
			},
			cel: cel,
		}

		// Build a minimal topology with only the consumer (product-c)
		// and its non-product dependencies, but without product-a which
		// provides "acs".
		topologyWithoutProvider := NewTopology()
		for _, d := range topology.Dependencies() {
			// Skip product-a (the acs provider) to simulate it being
			// absent from the topology.
			if d.Name() == "helmet-product-a" {
				continue
			}
			topologyWithoutProvider.Append(d)
		}

		err = i.Inspect(topologyWithoutProvider)
		g.Expect(err).To(o.HaveOccurred())
	})
}
