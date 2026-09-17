package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/redhat-appstudio/helmet/internal/annotations"
	"github.com/redhat-appstudio/helmet/internal/constants"
	"github.com/redhat-appstudio/helmet/internal/k8s"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// inClusterNamespaceFile is the ServiceAccount namespace injected into pods.
const inClusterNamespaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

// ConfigNamespaceEnv overrides ConfigMap discovery namespace. When unset,
// the in-cluster ServiceAccount namespace is used, then the kubeconfig context
// namespace. Discovery never lists ConfigMaps cluster-wide.
const ConfigNamespaceEnv = "HELMET_CONFIG_NAMESPACE"

// ConfigMapManager the actor responsible for managing installer configuration in
// the cluster.
//
//nolint:revive
type ConfigMapManager struct {
	kube    k8s.Interface // kubernetes client
	name    string        // configmap name
	appName string        // config root key
}

// Selector label selector for installer configuration.
const Selector = annotations.Config + "=true"

// Name returns the ConfigMap name.
func (m *ConfigMapManager) Name() string {
	return m.name
}

var (
	// ErrConfigMapNotFound when the configmap isn't created in the cluster.
	ErrConfigMapNotFound = errors.New("cluster configmap not found")
	// ErrMultipleConfigMapFound when the label selector find multiple resources.
	ErrMultipleConfigMapFound = errors.New("multiple cluster configmaps found")
	// ErrIncompleteConfigMap when the ConfigMap exists, but doesn't contain the
	// expected payload.
	ErrIncompleteConfigMap = errors.New("invalid configmap found in the cluster")
)

// namespaceFromEnvOrInCluster returns HELMET_CONFIG_NAMESPACE or the pod
// ServiceAccount namespace. Empty means those sources are unavailable.
func namespaceFromEnvOrInCluster() string {
	if ns := strings.TrimSpace(os.Getenv(ConfigNamespaceEnv)); ns != "" {
		return ns
	}
	data, err := os.ReadFile(inClusterNamespaceFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func (m *ConfigMapManager) configListNamespace() (string, error) {
	if ns := namespaceFromEnvOrInCluster(); ns != "" {
		return ns, nil
	}
	ns, _, err := m.kube.RESTClientGetter("").ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return "", fmt.Errorf(
			"unable to determine namespace for installer config (set %s): %w",
			ConfigNamespaceEnv,
			err,
		)
	}
	ns = strings.TrimSpace(ns)
	if ns == "" {
		return "", fmt.Errorf(
			"unable to determine namespace for installer config: set %s or a kubeconfig context namespace",
			ConfigNamespaceEnv,
		)
	}
	return ns, nil
}

func chooseConfigMap(items []corev1.ConfigMap) (*corev1.ConfigMap, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf(
			"%w: using label selector %q",
			ErrConfigMapNotFound,
			Selector,
		)
	}
	if len(items) > 1 {
		configMaps := []string{}
		for _, cm := range items {
			configMaps = append(
				configMaps,
				fmt.Sprintf("%s/%s", cm.GetNamespace(), cm.GetName()),
			)
		}
		return nil, fmt.Errorf(
			"%w: multiple configmaps found on namespace/name pairs: %v",
			ErrMultipleConfigMapFound,
			configMaps,
		)
	}
	return &items[0], nil
}

// GetConfigMap retrieves the installer ConfigMap from the current namespace
// only (never cluster-wide).
func (m *ConfigMapManager) GetConfigMap(
	ctx context.Context,
) (*corev1.ConfigMap, error) {
	ns, err := m.configListNamespace()
	if err != nil {
		return nil, err
	}
	coreClient, err := m.kube.CoreV1ClientSet(ns)
	if err != nil {
		return nil, err
	}

	configMapList, err := coreClient.ConfigMaps(ns).List(ctx, metav1.ListOptions{
		LabelSelector: Selector,
	})
	if err != nil {
		return nil, err
	}
	return chooseConfigMap(configMapList.Items)
}

// GetConfig retrieves configuration from a cluster's ConfigMap.
func (m *ConfigMapManager) GetConfig(ctx context.Context) (*Config, error) {
	configMap, err := m.GetConfigMap(ctx)
	if err != nil {
		return nil, err
	}
	payload, ok := configMap.Data[constants.ConfigFilename]
	if !ok || len(payload) == 0 {
		return nil, fmt.Errorf(
			"%w: key %q not found in ConfigMap %s/%s",
			ErrIncompleteConfigMap,
			constants.ConfigFilename,
			configMap.GetNamespace(),
			configMap.GetName(),
		)
	}

	return NewConfigFromBytes(
		[]byte(payload),
		configMap.GetNamespace(),
		m.appName,
	)
}

// configMapForConfig generate a ConfigMap resource based on informed Config.
func (m *ConfigMapManager) configMapForConfig(cfg *Config) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.name,
			Namespace: cfg.Namespace(),
			Labels: map[string]string{
				annotations.Config: "true",
			},
		},
		Data: map[string]string{
			constants.ConfigFilename: cfg.String(),
		},
	}
}

// Create Bootstrap a ConfigMap with the provided configuration.
func (m *ConfigMapManager) Create(ctx context.Context, cfg *Config) error {
	cm := m.configMapForConfig(cfg)
	coreClient, err := m.kube.CoreV1ClientSet(cfg.Namespace())
	if err != nil {
		return err
	}
	_, err = coreClient.
		ConfigMaps(cfg.Namespace()).
		Create(ctx, cm, metav1.CreateOptions{})
	return err
}

// Update updates a ConfigMap with informed configuration.
func (m *ConfigMapManager) Update(ctx context.Context, cfg *Config) error {
	cm := m.configMapForConfig(cfg)
	coreClient, err := m.kube.CoreV1ClientSet(cfg.Namespace())
	if err != nil {
		return err
	}
	_, err = coreClient.
		ConfigMaps(cfg.Namespace()).
		Update(ctx, cm, metav1.UpdateOptions{})
	return err
}

// Delete find and delete the ConfigMap from the cluster.
func (m *ConfigMapManager) Delete(ctx context.Context) error {
	cm, err := m.GetConfigMap(ctx)
	if err != nil {
		return err
	}

	coreClient, err := m.kube.CoreV1ClientSet(cm.GetNamespace())
	if err != nil {
		return err
	}

	return coreClient.ConfigMaps(cm.GetNamespace()).
		Delete(ctx, cm.GetName(), metav1.DeleteOptions{})
}

// NewConfigMapManager instantiates the ConfigMapManager.
// The appName parameter is used to generate the ConfigMap name as "{appName}-config"
// and, with hyphens replaced by underscores, as the YAML root key for config
// decoding.
func NewConfigMapManager(kube k8s.Interface, appName string) *ConfigMapManager {
	return &ConfigMapManager{
		kube:    kube,
		name:    fmt.Sprintf("%s-config", appName),
		appName: strings.ReplaceAll(appName, "-", "_"),
	}
}
