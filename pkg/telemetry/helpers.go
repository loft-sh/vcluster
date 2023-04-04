package telemetry

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/util/translate"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	SyncerVersion  = "dev"
	TelemetryToken = "vAyMmhLjnNcPrQDXEpo9"
)

const (
	DisabledEnvVar             = "VCLUSTER_TELEMETRY_DISABLED"
	InstanaceCreatorTypeEnvVar = "VCLUSTER_INSTANCE_CREATOR_TYPE"
	InstanaceCreatorUIDEnvVar  = "VCLUSTER_INSTANCE_CREATOR_UID"
)

// getSyncerUID provides instance UID based on the UID of the PVC or SS/Deployment
func getSyncerUID(c client.Client, vclusterNamespace string) func() string {
	cachedUID := ""
	return func() string {
		if cachedUID != "" {
			return cachedUID
		}

		// we primarily use PVC as the source of vcluster instance UID
		pvc := &corev1.PersistentVolumeClaim{}
		err := c.Get(context.Background(), types.NamespacedName{Namespace: vclusterNamespace, Name: fmt.Sprintf("data-%s-0", translate.Suffix)}, pvc)
		if err == nil {
			cachedUID = string(pvc.GetUID())
			return cachedUID
		}
		if !kerrors.IsNotFound(err) {
			return ""
		}

		// If vcluster PVC doesn't exist we try to get UID from the vcluster StatefulSet (k3s or k0s distro)
		ss := &appsv1.StatefulSet{}
		err = c.Get(context.Background(), types.NamespacedName{Namespace: vclusterNamespace, Name: translate.Suffix}, ss)
		if err == nil {
			cachedUID = string(ss.GetUID())
			return cachedUID
		}
		if !kerrors.IsNotFound(err) {
			return ""
		}

		// If vcluster StatefulSet doesn't exist we try to get UID from the vcluster Deployment (k3s or eks distro)
		d := &appsv1.Deployment{}
		err = c.Get(context.Background(), types.NamespacedName{Namespace: vclusterNamespace, Name: translate.Suffix}, d)
		if err == nil {
			cachedUID = string(d.GetUID())
			return cachedUID
		}
		return ""
	}
}
