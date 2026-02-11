package e2e

import (
	"context"
	"io"
	"testing"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	o "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/fake"
)

// newTestHelmConfig creates an action.Configuration backed by in-memory
// storage, suitable for unit tests. Releases can be pre-populated via the
// returned storage.Storage.
func newTestHelmConfig() (*action.Configuration, *storage.Storage) {
	mem := driver.NewMemory()
	store := storage.Init(mem)
	return &action.Configuration{
		Releases:   store,
		KubeClient: &kubefake.PrintingKubeClient{Out: io.Discard},
		Log:        func(_ string, _ ...any) {},
	}, store
}

// addRelease adds a release to the in-memory Helm storage.
func addRelease(
	t *testing.T,
	store *storage.Storage,
	name string,
	status release.Status,
) {
	t.Helper()
	err := store.Create(&release.Release{
		Name:    name,
		Version: 1,
		Info:    &release.Info{Status: status},
		Chart:   &chart.Chart{Metadata: &chart.Metadata{Name: name}},
	})
	if err != nil {
		t.Fatalf("failed to add release %q: %v", name, err)
	}
}

func TestReleasesChecker_Check(t *testing.T) {
	ctx := context.Background()
	namespace := "test-ns"

	expectedOrder := []string{
		"helmet-foundation",
		"helmet-operators",
		"helmet-infrastructure",
	}

	t.Run("succeeds with all releases deployed and correct sequence", func(t *testing.T) {
		g := o.NewWithT(t)

		helmCfg, store := newTestHelmConfig()
		for _, name := range expectedOrder {
			addRelease(t, store, name, release.StatusDeployed)
		}

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "deploy-sequence",
				Namespace: namespace,
			},
			Data: map[string]string{
				"sequence": "helmet-foundation\nhelmet-operators\nhelmet-infrastructure",
			},
		}

		client := fake.NewSimpleClientset(cm)
		checker := NewReleasesChecker(helmCfg, client, namespace, expectedOrder)
		result := checker.Check(ctx)

		g.Expect(result.Passed).To(o.BeTrue())
		g.Expect(result.Message).To(o.ContainSubstring("3 releases verified"))
	})

	t.Run("fails when a release is missing", func(t *testing.T) {
		g := o.NewWithT(t)

		helmCfg, store := newTestHelmConfig()
		addRelease(t, store, "helmet-foundation", release.StatusDeployed)
		// helmet-operators missing
		addRelease(t, store, "helmet-infrastructure", release.StatusDeployed)

		client := fake.NewSimpleClientset()
		checker := NewReleasesChecker(helmCfg, client, namespace, expectedOrder)
		result := checker.Check(ctx)

		g.Expect(result.Passed).To(o.BeFalse())
		g.Expect(result.Message).To(o.ContainSubstring("missing helm releases"))
		g.Expect(result.Message).To(o.ContainSubstring("helmet-operators"))
	})

	t.Run("fails when a release is not deployed", func(t *testing.T) {
		g := o.NewWithT(t)

		helmCfg, store := newTestHelmConfig()
		addRelease(t, store, "helmet-foundation", release.StatusDeployed)
		addRelease(t, store, "helmet-operators", release.StatusFailed)
		addRelease(t, store, "helmet-infrastructure", release.StatusDeployed)

		client := fake.NewSimpleClientset()
		checker := NewReleasesChecker(helmCfg, client, namespace, expectedOrder)
		result := checker.Check(ctx)

		g.Expect(result.Passed).To(o.BeFalse())
		g.Expect(result.Message).To(o.ContainSubstring("not in deployed status"))
		g.Expect(result.Message).To(o.ContainSubstring("helmet-operators"))
	})

	t.Run("fails when deploy-sequence ConfigMap is missing", func(t *testing.T) {
		g := o.NewWithT(t)

		helmCfg, store := newTestHelmConfig()
		for _, name := range expectedOrder {
			addRelease(t, store, name, release.StatusDeployed)
		}

		client := fake.NewSimpleClientset() // no ConfigMap
		checker := NewReleasesChecker(helmCfg, client, namespace, expectedOrder)
		result := checker.Check(ctx)

		g.Expect(result.Passed).To(o.BeFalse())
		g.Expect(result.Message).To(
			o.ContainSubstring("failed to get deploy-sequence ConfigMap"),
		)
	})

	t.Run("fails when sequence key is missing from ConfigMap", func(t *testing.T) {
		g := o.NewWithT(t)

		helmCfg, store := newTestHelmConfig()
		for _, name := range expectedOrder {
			addRelease(t, store, name, release.StatusDeployed)
		}

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "deploy-sequence",
				Namespace: namespace,
			},
			Data: map[string]string{"wrong-key": "data"},
		}

		client := fake.NewSimpleClientset(cm)
		checker := NewReleasesChecker(helmCfg, client, namespace, expectedOrder)
		result := checker.Check(ctx)

		g.Expect(result.Passed).To(o.BeFalse())
		g.Expect(result.Message).To(o.ContainSubstring("no 'sequence' key"))
	})

	t.Run("fails when deploy order is wrong", func(t *testing.T) {
		g := o.NewWithT(t)

		helmCfg, store := newTestHelmConfig()
		for _, name := range expectedOrder {
			addRelease(t, store, name, release.StatusDeployed)
		}

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "deploy-sequence",
				Namespace: namespace,
			},
			Data: map[string]string{
				"sequence": "helmet-operators\nhelmet-foundation\nhelmet-infrastructure",
			},
		}

		client := fake.NewSimpleClientset(cm)
		checker := NewReleasesChecker(helmCfg, client, namespace, expectedOrder)
		result := checker.Check(ctx)

		g.Expect(result.Passed).To(o.BeFalse())
		g.Expect(result.Message).To(o.ContainSubstring("deploy order mismatch"))
	})

	t.Run("fails when sequence length does not match", func(t *testing.T) {
		g := o.NewWithT(t)

		helmCfg, store := newTestHelmConfig()
		for _, name := range expectedOrder {
			addRelease(t, store, name, release.StatusDeployed)
		}

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "deploy-sequence",
				Namespace: namespace,
			},
			Data: map[string]string{
				"sequence": "helmet-foundation\nhelmet-operators",
			},
		}

		client := fake.NewSimpleClientset(cm)
		checker := NewReleasesChecker(helmCfg, client, namespace, expectedOrder)
		result := checker.Check(ctx)

		g.Expect(result.Passed).To(o.BeFalse())
		g.Expect(result.Message).To(o.ContainSubstring("length mismatch"))
	})

	t.Run("handles empty lines in sequence data", func(t *testing.T) {
		g := o.NewWithT(t)

		helmCfg, store := newTestHelmConfig()
		for _, name := range expectedOrder {
			addRelease(t, store, name, release.StatusDeployed)
		}

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "deploy-sequence",
				Namespace: namespace,
			},
			Data: map[string]string{
				"sequence": "\nhelmet-foundation\n\nhelmet-operators\nhelmet-infrastructure\n",
			},
		}

		client := fake.NewSimpleClientset(cm)
		checker := NewReleasesChecker(helmCfg, client, namespace, expectedOrder)
		result := checker.Check(ctx)

		g.Expect(result.Passed).To(o.BeTrue())
	})
}
