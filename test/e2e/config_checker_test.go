package e2e

import (
	"context"
	"testing"

	o "github.com/onsi/gomega"
	"github.com/redhat-appstudio/helmet/internal/annotations"
	"github.com/redhat-appstudio/helmet/internal/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestConfigChecker_Check(t *testing.T) {
	g := o.NewWithT(t)
	ctx := context.Background()
	namespace := "test-ns"
	appName := "helmet-ex"

	t.Run("succeeds with valid ConfigMap", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "helmet-ex-config",
				Namespace: namespace,
				Labels: map[string]string{
					annotations.Config: "true",
				},
			},
			Data: map[string]string{
				constants.ConfigFilename: `tssc:
  products:
    - name: Product A
      enabled: true`,
			},
		}

		client := fake.NewSimpleClientset(cm)
		checker := NewConfigChecker(client, namespace, appName)
		result := checker.Check(ctx)

		g.Expect(result.Passed).To(o.BeTrue())
		g.Expect(result.Message).To(o.ContainSubstring("1 products found"))
	})

	t.Run("fails when ConfigMap is missing", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		checker := NewConfigChecker(client, namespace, appName)
		result := checker.Check(ctx)

		g.Expect(result.Passed).To(o.BeFalse())
		g.Expect(result.Message).To(o.ContainSubstring("failed to get ConfigMap"))
	})

	t.Run("fails when label is missing", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "helmet-ex-config",
				Namespace: namespace,
			},
			Data: map[string]string{
				constants.ConfigFilename: "products: []",
			},
		}

		client := fake.NewSimpleClientset(cm)
		checker := NewConfigChecker(client, namespace, appName)
		result := checker.Check(ctx)

		g.Expect(result.Passed).To(o.BeFalse())
		g.Expect(result.Message).To(o.ContainSubstring("missing label"))
	})

	t.Run("fails when config.yaml data key is absent", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "helmet-ex-config",
				Namespace: namespace,
				Labels: map[string]string{
					annotations.Config: "true",
				},
			},
			Data: map[string]string{},
		}

		client := fake.NewSimpleClientset(cm)
		checker := NewConfigChecker(client, namespace, appName)
		result := checker.Check(ctx)

		g.Expect(result.Passed).To(o.BeFalse())
		g.Expect(result.Message).To(o.ContainSubstring(`no "config.yaml" data`))
	})

	t.Run("fails when config.yaml is empty string", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "helmet-ex-config",
				Namespace: namespace,
				Labels: map[string]string{
					annotations.Config: "true",
				},
			},
			Data: map[string]string{constants.ConfigFilename: ""},
		}

		client := fake.NewSimpleClientset(cm)
		checker := NewConfigChecker(client, namespace, appName)
		result := checker.Check(ctx)

		g.Expect(result.Passed).To(o.BeFalse())
		g.Expect(result.Message).To(o.ContainSubstring(`no "config.yaml" data`))
	})

	t.Run("fails when no products are defined", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "helmet-ex-config",
				Namespace: namespace,
				Labels: map[string]string{
					annotations.Config: "true",
				},
			},
			Data: map[string]string{
				constants.ConfigFilename: `tssc:
  products: []`,
			},
		}

		client := fake.NewSimpleClientset(cm)
		checker := NewConfigChecker(client, namespace, appName)
		result := checker.Check(ctx)

		g.Expect(result.Passed).To(o.BeFalse())
		g.Expect(result.Message).To(o.ContainSubstring("no product definitions"))
	})

	t.Run("fails when config.yaml is invalid YAML", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "helmet-ex-config",
				Namespace: namespace,
				Labels: map[string]string{
					annotations.Config: "true",
				},
			},
			Data: map[string]string{
				constants.ConfigFilename: "{{invalid yaml",
			},
		}

		client := fake.NewSimpleClientset(cm)
		checker := NewConfigChecker(client, namespace, appName)
		result := checker.Check(ctx)

		g.Expect(result.Passed).To(o.BeFalse())
		g.Expect(result.Message).To(o.ContainSubstring("failed to parse"))
	})

	t.Run("fails when top-level value is not a map", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "helmet-ex-config",
				Namespace: namespace,
				Labels: map[string]string{
					annotations.Config: "true",
				},
			},
			Data: map[string]string{
				constants.ConfigFilename: `scalar_key: "just a string"`,
			},
		}

		client := fake.NewSimpleClientset(cm)
		checker := NewConfigChecker(client, namespace, appName)
		result := checker.Check(ctx)

		g.Expect(result.Passed).To(o.BeFalse())
		g.Expect(result.Message).To(o.ContainSubstring("no product definitions"))
	})
}
