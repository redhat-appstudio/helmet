package e2e

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// SecretsChecker validates that integration Secrets exist in the target
// namespace.
type SecretsChecker struct {
	kubeClient  kubernetes.Interface // kubernetes client
	namespace   string               // installer namespace
	secretNames []string             // secret names
}

// Check verifies all expected secrets exist in the namespace.
func (s *SecretsChecker) Check(ctx context.Context) Result {
	var missing []string
	for _, name := range s.secretNames {
		_, err := s.kubeClient.CoreV1().Secrets(s.namespace).Get(
			ctx, name, metav1.GetOptions{},
		)
		if err != nil {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		return NewFailedResult(fmt.Errorf(
			"missing secrets in namespace %q: %s",
			s.namespace, strings.Join(missing, ", "),
		))
	}

	return NewResult(fmt.Sprintf(
		"all %d secrets verified in namespace %q",
		len(s.secretNames), s.namespace,
	))
}

// NewSecretsChecker creates a SecretsChecker for the specified secrets.
func NewSecretsChecker(
	kubeClient kubernetes.Interface,
	namespace string,
	secretNames []string,
) *SecretsChecker {
	return &SecretsChecker{
		kubeClient:  kubeClient,
		namespace:   namespace,
		secretNames: secretNames,
	}
}
