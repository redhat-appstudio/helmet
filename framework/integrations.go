package framework

import (
	"log/slog"

	"github.com/redhat-appstudio/helmet/api"
	"github.com/redhat-appstudio/helmet/internal/integration"
	"github.com/redhat-appstudio/helmet/internal/integrations"
	"github.com/redhat-appstudio/helmet/internal/k8s"
	"github.com/redhat-appstudio/helmet/internal/subcmd"
)

// GitHubModuleWithURLProvider returns a GitHub integration module that uses the
// given URLProvider for webhook, homepage, and optional callback URLs when not
// set by flags. Use this in custom installers (e.g. helmet-ex) to supply URLs
// from config or environment.
func GitHubModuleWithURLProvider(provider integration.URLProvider) api.IntegrationModule {
	return api.IntegrationModule{
		Name: string(integrations.GitHub),
		Init: func(l *slog.Logger, k *k8s.Kube) integration.Interface {
			gh := integration.NewGitHub(l, k)
			gh.SetURLProvider(provider)
			return gh
		},
		Command: func(appCtx *api.AppContext, l *slog.Logger, k *k8s.Kube, i *integration.Integration) api.SubCommand {
			return subcmd.NewIntegrationGitHub(appCtx, l, k, i)
		},
	}
}

// WithGitHubURLProvider returns a copy of modules with the GitHub integration
// replaced by one that uses the given URLProvider. Use after StandardIntegrations()
// to customize GitHub App URLs (e.g. from env or config) without changing other
// integrations.
func WithGitHubURLProvider(modules []api.IntegrationModule, provider integration.URLProvider) []api.IntegrationModule {
	out := make([]api.IntegrationModule, 0, len(modules))
	for _, m := range modules {
		if m.Name == string(integrations.GitHub) {
			out = append(out, GitHubModuleWithURLProvider(provider))
		} else {
			out = append(out, m)
		}
	}
	return out
}
