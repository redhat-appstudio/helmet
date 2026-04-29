package resolver

import (
	"testing"

	"github.com/redhat-appstudio/helmet/internal/annotations"

	o "github.com/onsi/gomega"
	"helm.sh/helm/v3/pkg/chart"
)

// depWithWeight creates a Dependency with a given name and weight annotation.
func depWithWeight(name, weight string) Dependency {
	ann := map[string]string{}
	if weight != "" {
		ann[annotations.Weight] = weight
	}
	return *NewDependency(&chart.Chart{
		Metadata: &chart.Metadata{
			Name:        name,
			Annotations: ann,
		},
	})
}

func names(t *Topology) []string {
	out := make([]string, 0, len(t.dependencies))
	for _, d := range t.dependencies {
		out = append(out, d.Name())
	}
	return out
}

func TestPrependBefore_WeightOrdering(t *testing.T) {
	t.Run("higher weight lands later among peers", func(t *testing.T) {
		g := o.NewWithT(t)
		top := NewTopology()
		parent := depWithWeight("parent", "")
		top.Append(parent)

		low := depWithWeight("low", "0")
		mid := depWithWeight("mid", "50")
		high := depWithWeight("high", "500")

		top.PrependBefore("parent", low, mid, high)
		g.Expect(names(top)).To(o.Equal([]string{
			"low", "mid", "high", "parent",
		}))
	})

	t.Run("negative weight lands before zero-weight peers", func(t *testing.T) {
		g := o.NewWithT(t)
		top := NewTopology()
		parent := depWithWeight("parent", "")
		top.Append(parent)

		neg := depWithWeight("neg", "-10")
		zero := depWithWeight("zero", "0")
		pos := depWithWeight("pos", "10")

		top.PrependBefore("parent", zero, neg, pos)
		g.Expect(names(top)).To(o.Equal([]string{
			"neg", "zero", "pos", "parent",
		}))
	})
}

func TestAppendAfter_WeightOrdering(t *testing.T) {
	t.Run("higher weight lands later among peers", func(t *testing.T) {
		g := o.NewWithT(t)
		top := NewTopology()
		anchor := depWithWeight("anchor", "")
		top.Append(anchor)

		low := depWithWeight("low", "0")
		high := depWithWeight("high", "99")

		top.AppendAfter("anchor", low, high)
		g.Expect(names(top)).To(o.Equal([]string{
			"anchor", "low", "high",
		}))
	})

	t.Run("negative weight lands before zero-weight peers", func(t *testing.T) {
		g := o.NewWithT(t)
		top := NewTopology()
		anchor := depWithWeight("anchor", "")
		top.Append(anchor)

		neg := depWithWeight("neg", "-5")
		zero := depWithWeight("zero", "0")

		top.AppendAfter("anchor", zero, neg)
		g.Expect(names(top)).To(o.Equal([]string{
			"anchor", "neg", "zero",
		}))
	})
}
