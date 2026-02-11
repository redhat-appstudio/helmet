package e2e

import (
	"context"
	"fmt"

	"github.com/redhat-appstudio/helmet/internal/annotations"
	"github.com/redhat-appstudio/helmet/internal/constants"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ConfigChecker validates the cluster configuration ConfigMap exists and
// contains product definitions.
type ConfigChecker struct {
	kubeClient kubernetes.Interface
	namespace  string
	appName    string
}

// Check verifies the ConfigMap exists with the expected label and contains
// valid config.yaml data with at least one product definition.
func (c *ConfigChecker) Check(ctx context.Context) Result {
	cmName := fmt.Sprintf("%s-config", c.appName)
	cm, err := c.kubeClient.CoreV1().ConfigMaps(c.namespace).Get(
		ctx, cmName, metav1.GetOptions{},
	)
	if err != nil {
		return NewFailedResult(
			fmt.Errorf("failed to get ConfigMap %q: %w", cmName, err),
		)
	}

	// Verify the label is present.
	if v, ok := cm.Labels[annotations.Config]; !ok || v != "true" {
		return NewFailedResult(
			fmt.Errorf("ConfigMap %q missing label %q=true",
				cmName, annotations.Config),
		)
	}

	// Verify config.yaml data key exists and is non-empty.
	configData, ok := cm.Data[constants.ConfigFilename]
	if !ok || configData == "" {
		return NewFailedResult(
			fmt.Errorf("ConfigMap %q has no %q data",
				cmName, constants.ConfigFilename),
		)
	}

	// Parse YAML to verify product definitions exist.
	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(configData), &parsed); err != nil {
		return NewFailedResult(
			fmt.Errorf("failed to parse %s from ConfigMap %q: %w",
				constants.ConfigFilename, cmName, err),
		)
	}

	// Look for products in the configuration structure. The config is nested
	// under the app name key (e.g., "tssc").
	for _, top := range parsed {
		topMap, ok := top.(map[string]any)
		if !ok {
			continue
		}
		if products, ok := topMap["products"]; ok {
			if productList, ok := products.([]any); ok {
				if len(productList) > 0 {
					return NewResult(fmt.Sprintf(
						"ConfigMap %q verified: %d products found",
						cmName, len(productList),
					))
				}
			}
		}
	}

	return NewFailedResult(
		fmt.Errorf("ConfigMap %q contains no product definitions", cmName),
	)
}

// NewConfigChecker creates a ConfigChecker for the specified application name.
func NewConfigChecker(
	kubeClient kubernetes.Interface,
	namespace string,
	appName string,
) *ConfigChecker {
	return &ConfigChecker{
		kubeClient: kubeClient,
		namespace:  namespace,
		appName:    appName,
	}
}
