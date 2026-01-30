package resolver

import (
	"os"
	"testing"

	"github.com/redhat-appstudio/helmet/api"
	"github.com/redhat-appstudio/helmet/internal/chartfs"

	o "github.com/onsi/gomega"
)

func TestNewCollection(t *testing.T) {
	g := o.NewWithT(t)

	cfs := chartfs.New(os.DirFS("../../test"))

	charts, err := cfs.GetAllCharts()
	g.Expect(err).To(o.Succeed())

	appCtx := api.NewAppContext("tssc")
	c, err := NewCollection(appCtx, charts)
	g.Expect(err).To(o.Succeed())
	g.Expect(c).NotTo(o.BeNil())
}
