package config

import (
	"os"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	o "github.com/onsi/gomega"
)

func TestChooseConfigMap(t *testing.T) {
	g := o.NewWithT(t)

	t.Run("not found", func(t *testing.T) {
		_, err := chooseConfigMap(nil)
		g.Expect(err).To(o.MatchError(ErrConfigMapNotFound))
	})

	t.Run("single", func(t *testing.T) {
		cm := corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "order-demo-config", Namespace: "ns-a"}}
		got, err := chooseConfigMap([]corev1.ConfigMap{cm})
		g.Expect(err).ToNot(o.HaveOccurred())
		g.Expect(got.Name).To(o.Equal("order-demo-config"))
		g.Expect(got.Namespace).To(o.Equal("ns-a"))
	})

	t.Run("multiple", func(t *testing.T) {
		items := []corev1.ConfigMap{
			{ObjectMeta: metav1.ObjectMeta{Name: "order-demo-config", Namespace: "ns-a"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "order-demo-config", Namespace: "ns-b"}},
		}
		_, err := chooseConfigMap(items)
		g.Expect(err).To(o.MatchError(ErrMultipleConfigMapFound))
		g.Expect(err.Error()).To(o.ContainSubstring("ns-a/order-demo-config"))
		g.Expect(err.Error()).To(o.ContainSubstring("ns-b/order-demo-config"))
	})
}

func TestNamespaceFromEnvOrInCluster(t *testing.T) {
	g := o.NewWithT(t)
	t.Setenv(ConfigNamespaceEnv, "workshop-p02")
	g.Expect(namespaceFromEnvOrInCluster()).To(o.Equal("workshop-p02"))

	t.Setenv(ConfigNamespaceEnv, "  workshop-p03  ")
	g.Expect(namespaceFromEnvOrInCluster()).To(o.Equal("workshop-p03"))

	t.Setenv(ConfigNamespaceEnv, "")
	if _, err := os.Stat(inClusterNamespaceFile); err != nil {
		g.Expect(namespaceFromEnvOrInCluster()).To(o.BeEmpty())
	}
}
