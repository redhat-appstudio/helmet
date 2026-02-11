package e2e

import (
	"context"
	"testing"

	o "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestSecretsChecker_Check(t *testing.T) {
	g := o.NewWithT(t)
	ctx := context.Background()
	namespace := "test-ns"

	t.Run("succeeds when all secrets exist", func(t *testing.T) {
		client := fake.NewSimpleClientset(
			&corev1.Secret{ObjectMeta: metav1.ObjectMeta{
				Name: "quay", Namespace: namespace,
			}},
			&corev1.Secret{ObjectMeta: metav1.ObjectMeta{
				Name: "acs", Namespace: namespace,
			}},
			&corev1.Secret{ObjectMeta: metav1.ObjectMeta{
				Name: "nexus", Namespace: namespace,
			}},
			&corev1.Secret{ObjectMeta: metav1.ObjectMeta{
				Name: "artifactory", Namespace: namespace,
			}},
		)
		checker := NewSecretsChecker(
			client, namespace,
			[]string{"quay", "acs", "nexus", "artifactory"},
		)
		result := checker.Check(ctx)

		g.Expect(result.Passed).To(o.BeTrue())
		g.Expect(result.Message).To(o.ContainSubstring("4 secrets verified"))
	})

	t.Run("fails when some secrets are missing", func(t *testing.T) {
		client := fake.NewSimpleClientset(
			&corev1.Secret{ObjectMeta: metav1.ObjectMeta{
				Name: "quay", Namespace: namespace,
			}},
		)
		checker := NewSecretsChecker(
			client, namespace, []string{"quay", "acs", "nexus"},
		)
		result := checker.Check(ctx)

		g.Expect(result.Passed).To(o.BeFalse())
		g.Expect(result.Message).To(o.ContainSubstring("acs"))
		g.Expect(result.Message).To(o.ContainSubstring("nexus"))
		g.Expect(result.Message).ToNot(o.ContainSubstring("quay"))
	})

	t.Run("fails when all secrets are missing", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		checker := NewSecretsChecker(client, namespace, []string{"quay", "acs"})
		result := checker.Check(ctx)

		g.Expect(result.Passed).To(o.BeFalse())
		g.Expect(result.Message).To(o.ContainSubstring("quay"))
		g.Expect(result.Message).To(o.ContainSubstring("acs"))
	})

	t.Run("succeeds with empty secret list", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		checker := NewSecretsChecker(client, namespace, []string{})
		result := checker.Check(ctx)

		g.Expect(result.Passed).To(o.BeTrue())
		g.Expect(result.Message).To(o.ContainSubstring("0 secrets verified"))
	})

	t.Run("fails when secret is in wrong namespace", func(t *testing.T) {
		client := fake.NewSimpleClientset(
			&corev1.Secret{ObjectMeta: metav1.ObjectMeta{
				Name: "quay", Namespace: "other-ns",
			}},
		)
		checker := NewSecretsChecker(client, namespace, []string{"quay"})
		result := checker.Check(ctx)

		g.Expect(result.Passed).To(o.BeFalse())
		g.Expect(result.Message).To(o.ContainSubstring("quay"))
	})
}
