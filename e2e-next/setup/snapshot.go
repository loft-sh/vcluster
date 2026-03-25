package setup

import (
	"context"
	"fmt"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// SnapshotPreSetup returns a PreSetupFunc that installs the CSI hostpath driver
// and snapshot infrastructure, then creates the snapshot-data PVC in the
// vCluster's host namespace.
func SnapshotPreSetup(vclusterName string) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		kubeContext := "kind-" + constants.GetHostClusterName()

		// Install CSI hostpath driver + snapshot CRDs + controller
		if err := InstallCSIHostpath(kubeContext)(ctx); err != nil {
			return fmt.Errorf("install CSI hostpath: %w", err)
		}

		// Create snapshot-data PVC
		return createSnapshotDataPVC(vclusterName)(ctx)
	}
}

// createSnapshotDataPVC creates the snapshot-data PVC in the vCluster's host
// namespace before the vCluster is provisioned.
func createSnapshotDataPVC(vclusterName string) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		hostCluster := cluster.From(ctx, constants.GetHostClusterName())
		if hostCluster == nil {
			return fmt.Errorf("host cluster %s not found in context", constants.GetHostClusterName())
		}
		hostClient, err := kubernetes.NewForConfig(hostCluster.KubernetesRestConfig())
		if err != nil {
			return fmt.Errorf("create host client: %w", err)
		}

		snapshotNS := "vcluster-" + vclusterName
		_, err = hostClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: snapshotNS},
		}, metav1.CreateOptions{})
		if err != nil && !kerrors.IsAlreadyExists(err) {
			return fmt.Errorf("create namespace %s: %w", snapshotNS, err)
		}

		_, err = hostClient.CoreV1().PersistentVolumeClaims(snapshotNS).Create(ctx, &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: "snapshot-data", Namespace: snapshotNS},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("100Mi"),
					},
				},
			},
		}, metav1.CreateOptions{})
		if err != nil && !kerrors.IsAlreadyExists(err) {
			return fmt.Errorf("create snapshot-data PVC: %w", err)
		}

		return nil
	}
}
