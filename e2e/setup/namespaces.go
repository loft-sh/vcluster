package setup

import (
	"context"
	"fmt"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e/constants"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

// HostNamespacesPreSetup returns a PreSetupFunc that creates the given host
// namespaces before the vCluster is provisioned. Namespaces mapped by fromHost
// configmap and secret sync have to exist at install time, because the chart
// creates a role and role binding in each of them.
func HostNamespacesPreSetup(namespaces ...string) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		hostCluster := cluster.From(ctx, constants.GetHostClusterName())
		if hostCluster == nil {
			return fmt.Errorf("host cluster %s not found in context", constants.GetHostClusterName())
		}
		hostClient, err := kubernetes.NewForConfig(hostCluster.KubernetesRestConfig())
		if err != nil {
			return fmt.Errorf("create host client: %w", err)
		}

		for _, namespace := range namespaces {
			if err := EnsureHostNamespace(ctx, hostClient, namespace); err != nil {
				return fmt.Errorf("create namespace %s: %w", namespace, err)
			}
		}
		return nil
	}
}

// EnsureHostNamespace creates the namespace if it doesn't exist, and waits
// until it exists and is not terminating (e.g. left over from a previous run).
func EnsureHostNamespace(ctx context.Context, hostClient kubernetes.Interface, name string) error {
	return wait.PollUntilContextTimeout(ctx, constants.PollingInterval, constants.PollingTimeout, true, func(ctx context.Context) (bool, error) {
		namespace, err := hostClient.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
		if kerrors.IsNotFound(err) {
			_, err = hostClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: name},
			}, metav1.CreateOptions{})
			if kerrors.IsAlreadyExists(err) {
				return false, nil
			}
			return err == nil, err
		}
		if err != nil {
			return false, err
		}
		return namespace.DeletionTimestamp == nil, nil
	})
}
