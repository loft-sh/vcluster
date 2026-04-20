package e2e_next

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/setup/lazyvcluster"
	"github.com/loft-sh/vcluster/e2e-next/test_core/export_kubeconfig"
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

// Ordered: the outer Describe owns vCluster lifecycle via BeforeAll +
// DeferCleanup - Ginkgo only allows BeforeAll/AfterAll inside Ordered
// containers.
func suiteExportKubeConfigVCluster() {
	Describe("export-kubeconfig-vcluster", Ordered,
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
		},
	)
}

// exportKubeConfigPreSetup creates the target namespace and RBAC so that
// the vCluster's cross-namespace additional-secret export can reach
// export_kubeconfig.TargetNS before the syncer first runs.
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

	saName := "vc-" + export_kubeconfig.VClusterName
	vClusterNS := "vcluster-" + export_kubeconfig.VClusterName
	roleName := export_kubeconfig.VClusterName + "-secret-access"

	_, err = hostClient.RbacV1().Roles(export_kubeconfig.TargetNS).Create(ctx, &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: export_kubeconfig.TargetNS,
		},
		Rules: []rbacv1.PolicyRule{
			{APIGroups: []string{"*"}, Resources: []string{"*"}, Verbs: []string{"*"}},
		},
	}, metav1.CreateOptions{})
	if err != nil && !kerrors.IsAlreadyExists(err) {
		return fmt.Errorf("create role: %w", err)
	}

	_, err = hostClient.RbacV1().RoleBindings(export_kubeconfig.TargetNS).Create(ctx, &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: export_kubeconfig.TargetNS,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     roleName,
		},
		Subjects: []rbacv1.Subject{
			{Kind: "ServiceAccount", Name: saName, Namespace: vClusterNS},
		},
	}, metav1.CreateOptions{})
	if err != nil && !kerrors.IsAlreadyExists(err) {
		return fmt.Errorf("create role binding: %w", err)
	}

	return nil
}
