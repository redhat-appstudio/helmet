package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/redhat-appstudio/helmet/api/integrations"
	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/k8s"
	"github.com/redhat-appstudio/helmet/internal/runcontext"
)

type mockPublicURLProvider struct {
	callbackURL string
	homepageURL string
	webhookURL  string
}

func (m *mockPublicURLProvider) GetCallbackURL(_ context.Context, _ integrations.IntegrationContext) (string, error) {
	return m.callbackURL, nil
}

func (m *mockPublicURLProvider) GetHomepageURL(_ context.Context, _ integrations.IntegrationContext) (string, error) {
	return m.homepageURL, nil
}

func (m *mockPublicURLProvider) GetWebhookURL(_ context.Context, _ integrations.IntegrationContext) (string, error) {
	return m.webhookURL, nil
}

func Test_urlProviderAdapter_delegates_to_public_provider(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	provider := &mockPublicURLProvider{
		callbackURL: "https://callback.example.com",
		homepageURL: "https://home.example.com",
		webhookURL:  "https://webhook.example.com",
	}
	// Adapter with nil runCtx/cfg: only delegation is tested; provider is not called with ic methods.
	adapter := newURLProviderAdapter(provider, nil, nil)

	cb, err := adapter.GetCallbackURL(ctx, nil, nil)
	if err != nil {
		t.Fatalf("GetCallbackURL: %v", err)
	}
	if cb != provider.callbackURL {
		t.Errorf("GetCallbackURL: got %q, want %q", cb, provider.callbackURL)
	}

	home, err := adapter.GetHomepageURL(ctx, nil, nil)
	if err != nil {
		t.Fatalf("GetHomepageURL: %v", err)
	}
	if home != provider.homepageURL {
		t.Errorf("GetHomepageURL: got %q, want %q", home, provider.homepageURL)
	}

	web, err := adapter.GetWebhookURL(ctx, nil, nil)
	if err != nil {
		t.Fatalf("GetWebhookURL: %v", err)
	}
	if web != provider.webhookURL {
		t.Errorf("GetWebhookURL: got %q, want %q", web, provider.webhookURL)
	}
}

func Test_urlProviderAdapter_propagates_provider_errors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	wantErr := errors.New("provider error")
	errProvider := &errURLProvider{mockPublicURLProvider{webhookURL: "https://web.example.com"}, wantErr}
	adapter := newURLProviderAdapter(errProvider, nil, nil)

	_, err := adapter.GetWebhookURL(ctx, nil, nil)
	if !errors.Is(err, wantErr) {
		t.Errorf("GetWebhookURL: got err %v, want %v", err, wantErr)
	}
}

type errURLProvider struct {
	mockPublicURLProvider
	err error
}

func (e *errURLProvider) GetWebhookURL(_ context.Context, _ integrations.IntegrationContext) (string, error) {
	return "", e.err
}

func Test_urlProviderAdapter_GetProductNamespace(t *testing.T) {
	t.Parallel()
	const productName = "Test Product"
	const wantNamespace = "test-product-ns"

	cfg, err := config.NewConfigFromBytes([]byte(`
tssc:
  settings: {}
  products:
    - name: "`+productName+`"
      enabled: true
      namespace: "`+wantNamespace+`"
`), "installer-ns")
	if err != nil {
		t.Fatalf("build config: %v", err)
	}

	adapter := newURLProviderAdapter(&mockPublicURLProvider{}, nil, cfg)

	got, err := adapter.GetProductNamespace(productName)
	if err != nil {
		t.Fatalf("GetProductNamespace: %v", err)
	}
	if got != wantNamespace {
		t.Errorf("GetProductNamespace(%q): got %q, want %q", productName, got, wantNamespace)
	}
}

func Test_urlProviderAdapter_GetProductNamespace_notFound(t *testing.T) {
	t.Parallel()
	cfg, err := config.NewConfigFromBytes([]byte(`
tssc:
  settings: {}
  products:
    - name: "Only Product"
      enabled: true
      namespace: "only-ns"
`), "installer-ns")
	if err != nil {
		t.Fatalf("build config: %v", err)
	}

	adapter := newURLProviderAdapter(&mockPublicURLProvider{}, nil, cfg)

	_, err = adapter.GetProductNamespace("Nonexistent")
	if err == nil {
		t.Fatal("GetProductNamespace: expected error for unknown product")
	}
	if err.Error() != "product 'Nonexistent' not found" {
		t.Errorf("GetProductNamespace: got err %v", err)
	}
}

func Test_urlProviderAdapter_GetOpenShiftIngressDomain(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fakeKube := k8s.NewFakeKube()
	runCtx := runcontext.NewRunContext(fakeKube, nil, nil)
	adapter := newURLProviderAdapter(&mockPublicURLProvider{}, runCtx, nil)

	_, err := adapter.GetOpenShiftIngressDomain(ctx)
	if err == nil {
		t.Fatal("GetOpenShiftIngressDomain: expected error when no OpenShift ingress controller")
	}
	// Adapter delegates to k8s.GetOpenShiftIngressDomain; with FakeKube we get
	// ErrIngressDomainNotFound or a connection/API error depending on environment.
	// Asserting err != nil is enough to confirm the IntegrationContext path is used.
}
