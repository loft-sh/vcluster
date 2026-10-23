// Suite: export-kubeconfig-vcluster
// Tests exportKubeConfig additional-secret sync across namespaces.
// Run:      just run-e2e 'exportkubeconfig'
package e2e

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e/clusters"
	"github.com/loft-sh/vcluster/e2e/constants"
	"github.com/loft-sh/vcluster/e2e/labels"
	"github.com/loft-sh/vcluster/e2e/setup/lazyvcluster"
	"github.com/loft-sh/vcluster/e2e/test_core/export_kubeconfig"
	. "github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

//go:embed vcluster-export-kubeconfig.yaml
var exportKubeConfigVClusterYAML string

func init() { suiteExportKubeConfigVCluster() }

func suiteExportKubeConfigVCluster() {
	Describe("export-kubeconfig-vcluster", labels.ExportKubeConfig, Ordered,
		cluster.Use(clusters.HostCluster),
		func() {
			BeforeAll(func(ctx context.Context) context.Context {
				return lazyvcluster.LazyVCluster(ctx,
					export_kubeconfig.VClusterName,
					exportKubeConfigVClusterYAML,
					lazyvcluster.WithPreSetup(exportKubeConfigPreSetup),
				)
			})

			export_kubeconfig.ExportKubeConfigSpec()
			export_kubeconfig.ExportPermissionsSpec()
			export_kubeconfig.ServiceAccountExportSpec()
		},
	)
}

// exportKubeConfigPreSetup provisions the legacy cross-namespace destination.
// The chart provisions the rotating destination and its scoped permissions.
func exportKubeConfigPreSetup(ctx context.Context) error {
	hostCluster := cluster.From(ctx, constants.GetHostClusterName())
	if hostCluster == nil {
		return fmt.Errorf("host cluster %s not found in context", constants.GetHostClusterName())
	}
	hostClient, err := kubernetes.NewForConfig(hostCluster.KubernetesRestConfig())
	if err != nil {
		return fmt.Errorf("create host client: %w", err)
	}

	_, err = hostClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: export_kubeconfig.TargetNS},
	}, metav1.CreateOptions{})
	if err != nil && !kerrors.IsAlreadyExists(err) {
		return fmt.Errorf("create namespace %s: %w", export_kubeconfig.TargetNS, err)
	}
	_, err = hostClient.CoreV1().Secrets(export_kubeconfig.TargetNS).Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: export_kubeconfig.CrossNSSecretName},
	}, metav1.CreateOptions{})
	if err != nil && !kerrors.IsAlreadyExists(err) {
		return fmt.Errorf("create legacy export Secret: %w", err)
	}
	_, err = hostClient.RbacV1().Roles(export_kubeconfig.TargetNS).Create(ctx, &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: export_kubeconfig.CrossNSSecretName},
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{""}, Resources: []string{"secrets"},
			ResourceNames: []string{export_kubeconfig.CrossNSSecretName}, Verbs: []string{"get", "patch"},
		}},
	}, metav1.CreateOptions{})
	if err != nil && !kerrors.IsAlreadyExists(err) {
		return fmt.Errorf("create legacy export Role: %w", err)
	}
	_, err = hostClient.RbacV1().RoleBindings(export_kubeconfig.TargetNS).Create(ctx, &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: export_kubeconfig.CrossNSSecretName},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName, Kind: "Role", Name: export_kubeconfig.CrossNSSecretName,
		},
		Subjects: []rbacv1.Subject{{
			Kind: "ServiceAccount", Name: "vc-" + export_kubeconfig.VClusterName,
			Namespace: "vcluster-" + export_kubeconfig.VClusterName,
		}},
	}, metav1.CreateOptions{})
	if err != nil && !kerrors.IsAlreadyExists(err) {
		return fmt.Errorf("create legacy export RoleBinding: %w", err)
	}

	return nil
}
