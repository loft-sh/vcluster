package clusters

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ExportKubeConfigVCluster has exportKubeConfig.additionalSecrets configured
// with two entries: one same-namespace and one cross-namespace secret.
// PreSetup creates the target namespace and RBAC for cross-namespace export.

//go:embed vcluster-export-kubeconfig.yaml
var exportKubeConfigVClusterYAML string

const (
	ExportKubeConfigVClusterName = "export-kubeconfig-vcluster"
	ExportKubeConfigTargetNS     = "export-kubeconfig-target"

	// Same-namespace additional secret config (must match YAML).
	ExportKubeConfigSameNSSecretName = "export-kubeconfig-same-ns"
	ExportKubeConfigSameNSServer     = "https://export-kubeconfig-vcluster.vcluster-export-kubeconfig-vcluster.svc:443"
	ExportKubeConfigSameNSContext    = "same-ns-context"

	// Cross-namespace additional secret config (must match YAML).
	ExportKubeConfigCrossNSSecretName = "export-kubeconfig-cross-ns"
	ExportKubeConfigCrossNSServer     = "https://export-kubeconfig-vcluster.export-kubeconfig-target.svc:443"
	ExportKubeConfigCrossNSContext    = "cross-ns-context"
)

var ExportKubeConfigVCluster = registerWith(
	ExportKubeConfigVClusterName,
	exportKubeConfigVClusterYAML,
	[]RegisterOption{WithPreSetup(exportKubeConfigPreSetup)},
)

func exportKubeConfigPreSetup(ctx context.Context) error {
	hostCluster := cluster.From(ctx, constants.GetHostClusterName())
	if hostCluster == nil {
		return fmt.Errorf("host cluster %s not found in context", constants.GetHostClusterName())
	}
	hostClient, err := kubernetes.NewForConfig(hostCluster.KubernetesRestConfig())
	if err != nil {
		return fmt.Errorf("create host client: %w", err)
	}

	// Create the target namespace for cross-namespace export.
	_, err = hostClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: ExportKubeConfigTargetNS},
	}, metav1.CreateOptions{})
	if err != nil && !kerrors.IsAlreadyExists(err) {
		return fmt.Errorf("create namespace %s: %w", ExportKubeConfigTargetNS, err)
	}

	// Grant the vCluster service account permissions to manage secrets
	// in the target namespace.
	saName := "vc-" + ExportKubeConfigVClusterName
	vClusterNS := "vcluster-" + ExportKubeConfigVClusterName
	roleName := ExportKubeConfigVClusterName + "-secret-access"

	_, err = hostClient.RbacV1().Roles(ExportKubeConfigTargetNS).Create(ctx, &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: ExportKubeConfigTargetNS,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
		},
	}, metav1.CreateOptions{})
	if err != nil && !kerrors.IsAlreadyExists(err) {
		return fmt.Errorf("create role: %w", err)
	}

	_, err = hostClient.RbacV1().RoleBindings(ExportKubeConfigTargetNS).Create(ctx, &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: ExportKubeConfigTargetNS,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     roleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      saName,
				Namespace: vClusterNS,
			},
		},
	}, metav1.CreateOptions{})
	if err != nil && !kerrors.IsAlreadyExists(err) {
		return fmt.Errorf("create role binding: %w", err)
	}

	return nil
}
