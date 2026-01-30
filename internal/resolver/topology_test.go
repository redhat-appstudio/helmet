package resolver

import (
	"os"
	"testing"

	"github.com/redhat-appstudio/helmet/internal/chartfs"

	o "github.com/onsi/gomega"
)

func TestNewTopology(t *testing.T) {
	g := o.NewWithT(t)

	cfs := chartfs.New(os.DirFS("../../test"))
	g.Expect(cfs).ToNot(o.BeNil())

	ns := "default"
	foundationChart, err := cfs.GetChartFiles("charts/helmet-foundation")
	g.Expect(err).To(o.Succeed())
	foundationDep := NewDependencyWithNamespace(foundationChart, ns)

	operatorsChart, err := cfs.GetChartFiles("charts/helmet-operators")
	g.Expect(err).To(o.Succeed())
	operatorsDep := NewDependencyWithNamespace(operatorsChart, ns)

	infrastructureChart, err := cfs.GetChartFiles("charts/helmet-infrastructure")
	g.Expect(err).To(o.Succeed())
	infrastructureDep := NewDependencyWithNamespace(infrastructureChart, ns)

	networkingChart, err := cfs.GetChartFiles("charts/helmet-networking")
	g.Expect(err).To(o.Succeed())
	networkingDep := NewDependencyWithNamespace(networkingChart, ns)

	topology := NewTopology()

	t.Run("Append", func(t *testing.T) {
		topology.Append(*networkingDep)
	})

	t.Run("PrependBefore", func(t *testing.T) {
		topology.PrependBefore(
			networkingDep.Name(),
			*foundationDep,
			*infrastructureDep,
		)
	})

	t.Run("AppendAfter", func(t *testing.T) {
		topology.AppendAfter(foundationChart.Name(), *operatorsDep)
	})

	t.Run("GetDependencies", func(t *testing.T) {
		deps := topology.Dependencies()
		g.Expect(deps).ToNot(o.BeNil())
		names := []string{}
		for _, d := range deps {
			names = append(names, d.Name())
		}
		g.Expect(names).To(o.Equal([]string{
			"helmet-foundation",
			"helmet-operators",
			"helmet-infrastructure",
			"helmet-networking",
		}))
	})
}
