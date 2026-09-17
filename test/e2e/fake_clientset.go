package e2e

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

// newFakeClientset wraps fake.NewSimpleClientset for unit tests. client-go v0.35
// deprecates NewSimpleClientset in favor of NewClientset with apply configs.
func newFakeClientset(objects ...runtime.Object) *fake.Clientset {
	//nolint:staticcheck // SA1019: keep until apply configs are generated for tests
	return fake.NewSimpleClientset(objects...)
}
