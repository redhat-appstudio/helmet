package integrations

import (
	"context"

	"github.com/redhat-appstudio/helmet/internal/config"
)

// CustomURLProvider supplies GitHub App URLs for the helmet-ex example.
// Use with framework.WithGitHubURLProvider.
type CustomURLProvider struct{}

func (CustomURLProvider) GetCallbackURL(_ context.Context, _ *config.Config) string {
	return ""
}

func (CustomURLProvider) GetWebhookURL(_ context.Context, _ *config.Config) string {
	return "https://webhook.mycluster.com"
}

func (CustomURLProvider) GetHomepageURL(_ context.Context, _ *config.Config) string {
	return "https://homepage.mycluster.com"
}
