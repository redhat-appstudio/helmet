package e2e

import (
	"context"
	"fmt"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ReleasesChecker validates Helm releases are installed and deployed in the
// correct topology order.
type ReleasesChecker struct {
	helmConfig      *action.Configuration
	kubeClient      kubernetes.Interface
	namespace       string
	expectedOrder   []string
	deploySeqCMName string
}

// Check verifies:
//  1. All expected releases exist (via helm list).
//  2. All releases are in "deployed" status.
//  3. Deploy order matches expected topology (via deploy-sequence ConfigMap).
func (r *ReleasesChecker) Check(ctx context.Context) Result {
	// 1. List all Helm releases.
	listAction := action.NewList(r.helmConfig)
	listAction.All = true
	releases, err := listAction.Run()
	if err != nil {
		return NewFailedResult(
			fmt.Errorf("failed to list helm releases: %w", err),
		)
	}

	releaseMap := make(map[string]*release.Release, len(releases))
	for _, rel := range releases {
		releaseMap[rel.Name] = rel
	}

	// 2. Verify all expected releases exist and are deployed.
	var missing []string
	var notDeployed []string
	for _, name := range r.expectedOrder {
		rel, ok := releaseMap[name]
		if !ok {
			missing = append(missing, name)
			continue
		}
		if rel.Info.Status != release.StatusDeployed {
			notDeployed = append(notDeployed, fmt.Sprintf(
				"%s (status: %s)", name, rel.Info.Status,
			))
		}
	}

	if len(missing) > 0 {
		return NewFailedResult(fmt.Errorf(
			"missing helm releases: %s", strings.Join(missing, ", "),
		))
	}
	if len(notDeployed) > 0 {
		return NewFailedResult(fmt.Errorf(
			"releases not in deployed status: %s",
			strings.Join(notDeployed, ", "),
		))
	}

	// 3. Verify deploy order via the deploy-sequence ConfigMap.
	cm, err := r.kubeClient.CoreV1().ConfigMaps(r.namespace).Get(
		ctx, r.deploySeqCMName, metav1.GetOptions{},
	)
	if err != nil {
		return NewFailedResult(fmt.Errorf(
			"failed to get deploy-sequence ConfigMap %q: %w",
			r.deploySeqCMName, err,
		))
	}

	sequenceData, ok := cm.Data["sequence"]
	if !ok {
		return NewFailedResult(fmt.Errorf(
			"deploy-sequence ConfigMap has no 'sequence' key",
		))
	}

	// Parse the newline-separated sequence and filter out empty lines.
	var actualOrder []string
	for line := range strings.SplitSeq(sequenceData, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			actualOrder = append(actualOrder, trimmed)
		}
	}

	if len(actualOrder) != len(r.expectedOrder) {
		return NewFailedResult(fmt.Errorf(
			"deploy sequence length mismatch: expected %d, got %d\n"+
				"expected: %v\nactual: %v",
			len(r.expectedOrder), len(actualOrder),
			r.expectedOrder, actualOrder,
		))
	}

	for i, expected := range r.expectedOrder {
		if actualOrder[i] != expected {
			return NewFailedResult(fmt.Errorf(
				"deploy order mismatch at position %d: expected %q, got %q\n"+
					"expected: %v\nactual: %v",
				i, expected, actualOrder[i],
				r.expectedOrder, actualOrder,
			))
		}
	}

	return NewResult(fmt.Sprintf(
		"all %d releases verified in correct topology order",
		len(r.expectedOrder),
	))
}

// NewReleasesChecker creates a ReleasesChecker. The expectedOrder slice
// defines the topology-sorted deployment order. The deploy-sequence ConfigMap
// name defaults to "deploy-sequence".
func NewReleasesChecker(
	helmConfig *action.Configuration,
	kubeClient kubernetes.Interface,
	namespace string,
	expectedOrder []string,
) *ReleasesChecker {
	return &ReleasesChecker{
		helmConfig:      helmConfig,
		kubeClient:      kubeClient,
		namespace:       namespace,
		expectedOrder:   expectedOrder,
		deploySeqCMName: "deploy-sequence",
	}
}
