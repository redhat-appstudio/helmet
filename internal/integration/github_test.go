package integration

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/redhat-appstudio/helmet/internal/config"
)

type mockURLProvider struct {
	callbackURL string
	webhookURL  string
	homepageURL string
}

func (m *mockURLProvider) GetCallbackURL(ctx context.Context, cfg *config.Config) string {
	return m.callbackURL
}

func (m *mockURLProvider) GetWebhookURL(ctx context.Context, cfg *config.Config) string {
	return m.webhookURL
}

func (m *mockURLProvider) GetHomepageURL(ctx context.Context, cfg *config.Config) string {
	return m.homepageURL
}

func TestGitHub_SetURLProvider_AllURLs(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	gh := NewGitHub(logger, nil)

	provider := &mockURLProvider{
		callbackURL: "https://custom-callback.example.com",
		webhookURL:  "https://custom-webhook.example.com",
		homepageURL: "https://custom-homepage.example.com",
	}

	gh.SetURLProvider(provider)

	if gh.urlProvider == nil {
		t.Error("expected urlProvider to be set")
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
			gh := NewGitHub(logger, nil)
			gh.SetURLProvider(&mockURLProvider{
				webhookURL:  tc.webhook,
				homepageURL: tc.home,
			})

			err := gh.setClusterURLs(ctx, nil)
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
	gh := NewGitHub(logger, nil)

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
		gh := NewGitHub(logger, nil)
		// no SetURLProvider, no URLs set

		err := gh.setClusterURLs(ctx, nil)
		if err == nil {
			t.Fatal("expected setClusterURLs to fail when webhook and homepage are not provided")
		}
		if !strings.Contains(err.Error(), "webhook and homepage URLs must be provided via flags or URLProvider") {
			t.Errorf("expected documented error message, got: %v", err)
		}
	})

	t.Run("provider returns empty for required URL", func(t *testing.T) {
		t.Parallel()
		gh := NewGitHub(logger, nil)
		gh.SetURLProvider(&mockURLProvider{
			webhookURL:  "",
			homepageURL: "",
		})

		err := gh.setClusterURLs(ctx, nil)
		if err == nil {
			t.Fatal("expected setClusterURLs to fail when provider returns empty for required URLs")
		}
		if !strings.Contains(err.Error(), "webhook and homepage URLs must be provided via flags or URLProvider") {
			t.Errorf("expected documented error message, got: %v", err)
		}
	})
}
