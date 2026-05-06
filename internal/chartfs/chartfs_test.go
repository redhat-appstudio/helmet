package chartfs

import (
	"os"
	"path/filepath"
	"testing"

	o "github.com/onsi/gomega"
)

func TestNewChartFS(t *testing.T) {
	g := o.NewWithT(t)

	c := New(os.DirFS("../../test"))
	g.Expect(c).ToNot(o.BeNil())

	t.Run("ReadFile", func(t *testing.T) {
		valuesTmplBytes, err := c.ReadFile("values.yaml.tpl")
		g.Expect(err).To(o.Succeed())
		g.Expect(valuesTmplBytes).ToNot(o.BeEmpty())
	})

	t.Run("GetChartForDep", func(t *testing.T) {
		chart, err := c.GetChartFiles("charts/helmet-product-a")
		g.Expect(err).To(o.Succeed())
		g.Expect(chart).ToNot(o.BeNil())
		g.Expect(chart.Name()).To(o.Equal("helmet-product-a"))
		g.Expect(chart.Templates).ToNot(o.BeEmpty())

		// Asserting the chart templates are present, it should contain at least a
		// few files, plus the presence of the "NOTES.txt" common file.
		names := make([]string, 0, len(chart.Templates))
		for _, tmpl := range chart.Templates {
			names = append(names, tmpl.Name)
		}
		g.Expect(len(names)).To(o.BeNumerically("==", 3))
		g.Expect(names).To(o.ContainElement("templates/NOTES.txt"))
		g.Expect(names).To(o.ContainElement("templates/_copy-scripts.tpl"))
		g.Expect(names).To(o.ContainElement("templates/hooks/deploy-order.yaml"))
	})

	t.Run("GetAllCharts", func(t *testing.T) {
		charts, err := c.GetAllCharts()
		g.Expect(err).To(o.Succeed())
		g.Expect(charts).ToNot(o.BeNil())
		g.Expect(len(charts)).To(o.BeNumerically(">", 1))
	})
}

func TestChartFS_ExtractTo(t *testing.T) {
	g := o.NewWithT(t)

	c := New(os.DirFS("../../test"))

	destDir := t.TempDir()
	err := c.ExtractTo(destDir)
	g.Expect(err).To(o.Succeed())

	t.Run("chart directories are extracted", func(t *testing.T) {
		g := o.NewWithT(t)
		expectedCharts := []string{
			"charts/helmet-foundation",
			"charts/helmet-infrastructure",
			"charts/helmet-integrations",
			"charts/helmet-networking",
			"charts/helmet-operators",
			"charts/helmet-product-a",
			"charts/helmet-product-b",
			"charts/helmet-product-c",
			"charts/helmet-product-d",
			"charts/helmet-storage",
			"charts/testing",
		}
		for _, chartDir := range expectedCharts {
			chartYaml := filepath.Join(destDir, chartDir, "Chart.yaml")
			g.Expect(chartYaml).To(o.BeAnExistingFile())
		}
	})

	t.Run("chart contents are correct", func(t *testing.T) {
		g := o.NewWithT(t)
		srcBytes, err := os.ReadFile("../../test/charts/helmet-product-a/Chart.yaml")
		g.Expect(err).To(o.Succeed())

		dstBytes, err := os.ReadFile(
			filepath.Join(destDir, "charts/helmet-product-a/Chart.yaml"),
		)
		g.Expect(err).To(o.Succeed())
		g.Expect(dstBytes).To(o.Equal(srcBytes))
	})

	t.Run("template files are extracted", func(t *testing.T) {
		g := o.NewWithT(t)
		notesPath := filepath.Join(
			destDir, "charts/helmet-product-a/templates/NOTES.txt",
		)
		g.Expect(notesPath).To(o.BeAnExistingFile())
	})

	t.Run("non-chart files are excluded", func(t *testing.T) {
		g := o.NewWithT(t)
		g.Expect(filepath.Join(destDir, "values.yaml.tpl")).
			ToNot(o.BeAnExistingFile())
		g.Expect(filepath.Join(destDir, "config.yaml")).
			ToNot(o.BeAnExistingFile())
	})

	t.Run("non-chart directories are excluded", func(t *testing.T) {
		g := o.NewWithT(t)
		g.Expect(filepath.Join(destDir, "charts/_common")).
			ToNot(o.BeADirectory())
	})

	t.Run("file permissions are correct", func(t *testing.T) {
		g := o.NewWithT(t)
		chartYaml := filepath.Join(
			destDir, "charts/helmet-product-a/Chart.yaml",
		)
		info, err := os.Stat(chartYaml)
		g.Expect(err).To(o.Succeed())
		g.Expect(info.Mode().Perm()).To(o.Equal(os.FileMode(0o644)))

		chartDir := filepath.Join(destDir, "charts/helmet-product-a")
		dirInfo, err := os.Stat(chartDir)
		g.Expect(err).To(o.Succeed())
		g.Expect(dirInfo.Mode().Perm()).To(o.Equal(os.FileMode(0o755)))
	})
}
