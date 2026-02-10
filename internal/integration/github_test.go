package integration

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/redhat-appstudio/helmet/api/integrations"
)

type mockURLProvider struct {
	callbackURL string
	webhookURL  string
	homepageURL string
}

func (m *mockURLProvider) GetCallbackURL(_ context.Context, _ integrations.IntegrationContext) (string, error) {
	return m.callbackURL, nil
}

func (m *mockURLProvider) GetWebhookURL(_ context.Context, _ integrations.IntegrationContext) (string, error) {
	return m.webhookURL, nil
}

func (m *mockURLProvider) GetHomepageURL(_ context.Context, _ integrations.IntegrationContext) (string, error) {
	return m.homepageURL, nil
}

// failingURLProvider wraps mockURLProvider and returns a given error from one of its methods.
type failingURLProvider struct {
	*mockURLProvider
	callbackErr, webhookErr, homepageErr error
}

func (e *failingURLProvider) GetCallbackURL(ctx context.Context, ic integrations.IntegrationContext) (string, error) {
	if e.callbackErr != nil {
		return "", e.callbackErr
	}
	return e.mockURLProvider.GetCallbackURL(ctx, ic)
}

func (e *failingURLProvider) GetWebhookURL(ctx context.Context, ic integrations.IntegrationContext) (string, error) {
	if e.webhookErr != nil {
		return "", e.webhookErr
	}
	return e.mockURLProvider.GetWebhookURL(ctx, ic)
}

func (e *failingURLProvider) GetHomepageURL(ctx context.Context, ic integrations.IntegrationContext) (string, error) {
	if e.homepageErr != nil {
		return "", e.homepageErr
	}
	return e.mockURLProvider.GetHomepageURL(ctx, ic)
}

func TestGitHub_SetURLProvider_AllURLs(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()
	gh := NewGitHub(logger)
	gh.name = "test-app"

	provider := &mockURLProvider{
		callbackURL: "https://custom-callback.example.com",
		webhookURL:  "https://custom-webhook.example.com",
		homepageURL: "https://custom-homepage.example.com",
	}
	gh.SetURLProvider(provider)

	err := gh.setClusterURLs(ctx, nil, nil)
	if err != nil {
		t.Fatalf("setClusterURLs: %v", err)
	}

	manifest := gh.generateAppManifest()
	if v := manifest.HookAttributes["url"]; v != provider.webhookURL {
		t.Errorf("manifest webhook: got %q, want %q", v, provider.webhookURL)
	}
	if manifest.URL == nil || *manifest.URL != provider.homepageURL {
		t.Errorf("manifest homepage: got %v, want %q", manifest.URL, provider.homepageURL)
	}
	if len(manifest.CallbackURLs) != 1 || manifest.CallbackURLs[0] != provider.callbackURL {
		t.Errorf("manifest callback URLs: got %v, want [%q]", manifest.CallbackURLs, provider.callbackURL)
	}
}

func TestGitHub_SetURLProvider_PartialURLs(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	cases := []struct {
		name    string
		webhook string
		home    string
	}{
		{"missing webhook", "", "https://home.example.com"},
		{"missing homepage", "https://webhook.example.com", ""},
		{"both missing", "", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gh := NewGitHub(logger)
			gh.SetURLProvider(&mockURLProvider{
				webhookURL:  tc.webhook,
				homepageURL: tc.home,
			})

			err := gh.setClusterURLs(ctx, nil, nil)
			if err == nil {
				t.Fatal("expected setClusterURLs to fail when required URL is missing")
			}
			if !strings.Contains(err.Error(), "webhook and homepage URLs must be provided") {
				t.Errorf("expected error message about required URLs, got: %v", err)
			}
		})
	}
}

func TestGitHub_FlagsTakePrecedenceOverURLProvider(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()
	gh := NewGitHub(logger)

	flagWebhook := "https://flag-webhook.example.com"
	flagHome := "https://flag-home.example.com"
	flagCallback := "https://flag-callback.example.com"
	gh.webhookURL = flagWebhook
	gh.homepageURL = flagHome
	gh.callbackURL = flagCallback
	gh.name = "test-app"

	gh.SetURLProvider(&mockURLProvider{
		callbackURL: "https://provider-callback.example.com",
		webhookURL:  "https://provider-webhook.example.com",
		homepageURL: "https://provider-homepage.example.com",
	})

	err := gh.setClusterURLs(ctx, nil, nil)
	if err != nil {
		t.Fatalf("setClusterURLs: %v", err)
	}

	manifest := gh.generateAppManifest()

	if v := manifest.HookAttributes["url"]; v != flagWebhook {
		t.Errorf("manifest webhook: got %q, want %q", v, flagWebhook)
	}
	if manifest.URL == nil || *manifest.URL != flagHome {
		t.Errorf("manifest homepage: got %v, want %q", manifest.URL, flagHome)
	}
	if len(manifest.CallbackURLs) != 1 || manifest.CallbackURLs[0] != flagCallback {
		t.Errorf("manifest callback URLs: got %v, want [%q]", manifest.CallbackURLs, flagCallback)
	}
}

func TestGitHub_ErrorWhenRequiredURLsMissing(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	t.Run("no provider no flags", func(t *testing.T) {
		t.Parallel()
		gh := NewGitHub(logger)
		// no SetURLProvider, no URLs set

		err := gh.setClusterURLs(ctx, nil, nil)
		if err == nil {
			t.Fatal("expected setClusterURLs to fail when webhook and homepage are not provided")
		}
		if !strings.Contains(err.Error(), "webhook and homepage URLs must be provided via flags or URLProvider") {
			t.Errorf("expected documented error message, got: %v", err)
		}
	})

	t.Run("provider returns empty for required URL", func(t *testing.T) {
		t.Parallel()
		gh := NewGitHub(logger)
		gh.SetURLProvider(&mockURLProvider{
			webhookURL:  "",
			homepageURL: "",
		})

		err := gh.setClusterURLs(ctx, nil, nil)
		if err == nil {
			t.Fatal("expected setClusterURLs to fail when provider returns empty for required URLs")
		}
		if !strings.Contains(err.Error(), "webhook and homepage URLs must be provided via flags or URLProvider") {
			t.Errorf("expected documented error message, got: %v", err)
		}
	})
}

var errProviderSentinel = errors.New("urlprovider error")

func TestGitHub_SetClusterURLs_URLProviderErrors(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()
	base := &mockURLProvider{
		callbackURL: "https://cb.example.com",
		webhookURL:  "https://wh.example.com",
		homepageURL: "https://hp.example.com",
	}

	t.Run("GetCallbackURL error", func(t *testing.T) {
		t.Parallel()
		gh := NewGitHub(logger)
		gh.SetURLProvider(&failingURLProvider{mockURLProvider: base, callbackErr: errProviderSentinel})

		err := gh.setClusterURLs(ctx, nil, nil)
		if err == nil {
			t.Fatal("expected setClusterURLs to fail when GetCallbackURL returns error")
		}
		if !strings.Contains(err.Error(), "get callback URL") {
			t.Errorf("expected error to mention callback URL, got: %v", err)
		}
		if !errors.Is(err, errProviderSentinel) {
			t.Errorf("expected wrapped errProviderSentinel, got: %v", err)
		}
	})

	t.Run("GetWebhookURL error", func(t *testing.T) {
		t.Parallel()
		gh := NewGitHub(logger)
		gh.SetURLProvider(&failingURLProvider{mockURLProvider: base, webhookErr: errProviderSentinel})

		err := gh.setClusterURLs(ctx, nil, nil)
		if err == nil {
			t.Fatal("expected setClusterURLs to fail when GetWebhookURL returns error")
		}
		if !strings.Contains(err.Error(), "get webhook URL") {
			t.Errorf("expected error to mention webhook URL, got: %v", err)
		}
		if !errors.Is(err, errProviderSentinel) {
			t.Errorf("expected wrapped errProviderSentinel, got: %v", err)
		}
	})

	t.Run("GetHomepageURL error", func(t *testing.T) {
		t.Parallel()
		gh := NewGitHub(logger)
		gh.SetURLProvider(&failingURLProvider{mockURLProvider: base, homepageErr: errProviderSentinel})

		err := gh.setClusterURLs(ctx, nil, nil)
		if err == nil {
			t.Fatal("expected setClusterURLs to fail when GetHomepageURL returns error")
		}
		if !strings.Contains(err.Error(), "get homepage URL") {
			t.Errorf("expected error to mention homepage URL, got: %v", err)
		}
		if !errors.Is(err, errProviderSentinel) {
			t.Errorf("expected wrapped errProviderSentinel, got: %v", err)
		}
	})
}
