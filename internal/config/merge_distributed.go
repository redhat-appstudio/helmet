package config

import (
	"fmt"
	"path"
	"strings"

	"github.com/redhat-appstudio/helmet/internal/chartfs"

	"gopkg.in/yaml.v3"
)

const (
	distributedSettingsPath  = "config/settings.yaml"
	distributedBlueprintPath = "helmet.yaml"
	distributedChartsDir     = "charts"
	localRefPrefix           = "local://"
)

// helmInstallerBlueprint mirrors the installer helmet.yaml blueprint.
type helmInstallerBlueprint struct {
	Name         string   `yaml:"name"`
	Products     []string `yaml:"products"`
	Integrations []string `yaml:"integrations"`
}

// MergeDistributedInstallerYAML builds one config.yaml payload from settings,
// helmet blueprint, per-chart config.yaml files, and generated integration stubs.
func MergeDistributedInstallerYAML(cfs *chartfs.ChartFS, appIdentifier string) ([]byte, error) {
	settingsBytes, err := cfs.ReadFile(distributedSettingsPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", distributedSettingsPath, err)
	}
	helmBytes, err := cfs.ReadFile(distributedBlueprintPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", distributedBlueprintPath, err)
	}

	var blueprint helmInstallerBlueprint
	if err := yaml.Unmarshal(helmBytes, &blueprint); err != nil {
		return nil, fmt.Errorf("parse helmet blueprint: %w", err)
	}

	var settings map[string]interface{}
	if err := yaml.Unmarshal(settingsBytes, &settings); err != nil {
		return nil, fmt.Errorf("parse settings: %w", err)
	}

	var products []Product
	for _, ref := range blueprint.Products {
		chartName, err := chartNameFromLocalRef(ref)
		if err != nil {
			return nil, err
		}
		cfgPath := path.Join(distributedChartsDir, chartName, "config.yaml")
		body, err := cfs.ReadFile(cfgPath)
		if err != nil {
			return nil, fmt.Errorf("read product config %s: %w", cfgPath, err)
		}
		var p Product
		if err := yaml.Unmarshal(body, &p); err != nil {
			return nil, fmt.Errorf("parse product config %s: %w", cfgPath, err)
		}
		products = append(products, p)
	}

	var integrations []IntegrationSpec
	for _, ref := range blueprint.Integrations {
		id, err := integrationIDFromLocalRef(ref)
		if err != nil {
			return nil, err
		}
		entry, ok := integrationDummyByID(id)
		if !ok {
			return nil, fmt.Errorf("unknown integration reference %q", ref)
		}
		integrations = append(integrations, entry)
	}

	doc := map[string]interface{}{
		appIdentifier: map[string]interface{}{
			"settings":     settings,
			"integrations": integrations,
			"products":     products,
		},
	}

	out, err := yaml.Marshal(doc)
	if err != nil {
		return nil, err
	}
	return append([]byte("---\n"), out...), nil
}

func chartNameFromLocalRef(ref string) (string, error) {
	id, err := integrationIDFromLocalRef(ref)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("tssc-%s", id), nil
}

func integrationIDFromLocalRef(ref string) (string, error) {
	if !strings.HasPrefix(ref, localRefPrefix) {
		return "", fmt.Errorf("expected %q reference, got %q", localRefPrefix, ref)
	}
	return strings.TrimPrefix(ref, localRefPrefix), nil
}

func integrationDummyByID(id string) (IntegrationSpec, bool) {
	switch id {
	case "quay":
		return IntegrationSpec{
			Integration: "quay",
			Name:        "Red Hat Quay",
			Properties: map[string]interface{}{
				"url":                      "https://quay.example.com",
				"token":                    "REPLACE_ME",
				"organization":             "REPLACE_ME",
				"dockerconfigjson":         "{}",
				"dockerconfigjsonreadonly": "{}",
			},
		}, true
	case "github":
		return IntegrationSpec{
			Integration: "github",
			Name:        "GitHub App",
			Properties: map[string]interface{}{
				"clientId":      "REPLACE_ME",
				"clientSecret":  "REPLACE_ME",
				"createdAt":     "",
				"externalURL":   "",
				"htmlURL":       "",
				"host":          "github.com",
				"id":            "",
				"name":          "",
				"nodeId":        "",
				"ownerLogin":    "",
				"ownerId":       "",
				"pem":           "REPLACE_ME",
				"slug":          "",
				"updatedAt":     "",
				"webhookSecret": "REPLACE_ME",
				"token":         "REPLACE_ME",
			},
		}, true
	default:
		return IntegrationSpec{}, false
	}
}
