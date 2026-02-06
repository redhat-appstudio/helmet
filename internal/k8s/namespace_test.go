package k8s

import (
	"context"
	"log/slog"
	"testing"

	"github.com/redhat-appstudio/helmet/test/stubs"

	"k8s.io/apimachinery/pkg/runtime"

	o "github.com/onsi/gomega"
)

// TestEnsureNamespace tests the EnsureNamespace function with various scenarios.
func TestEnsureNamespace(t *testing.T) {
	tests := []struct {
		name         string
		namespace    string
		existingObjs []runtime.Object
		wantErr      bool
	}{
		{
			name:         "namespace already exists",
			namespace:    "existing-namespace",
			existingObjs: []runtime.Object{stubs.NamespaceRuntimeObject("existing-namespace")},
			wantErr:      false,
		},
		{
			name:         "namespace does not exist",
			namespace:    "new-namespace",
			existingObjs: []runtime.Object{},
			wantErr:      false,
		},
		{
			name:         "namespace created when other namespaces exist",
			namespace:    "another-namespace",
			existingObjs: []runtime.Object{stubs.NamespaceRuntimeObject("existing-namespace")},
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := o.NewWithT(t)

			kube := NewFakeKube(tt.existingObjs...)
			logger := slog.Default()
			ctx := context.TODO()

			err := EnsureNamespace(ctx, logger, kube, tt.namespace)

			if tt.wantErr {
				g.Expect(err).To(o.HaveOccurred())
				return
			}

			g.Expect(err).ToNot(o.HaveOccurred())
		})
	}
}
