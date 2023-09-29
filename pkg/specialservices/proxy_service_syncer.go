package specialservices

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

var (
	VclusterProxyMetricsSvcKey = types.NamespacedName{
		Name:      "metrics-server",
		Namespace: "kube-system",
	}
)

func SyncVclusterProxyService(ctx *synccontext.SyncContext,
	svcNamespace,
	svcName string,
	vSvcToSync types.NamespacedName,
	svcPortTranslator ServicePortTranslator) error {

	pClient := ctx.PhysicalClient
	// get physical service
	pObj := &corev1.Service{}
	err := pClient.Get(ctx.Context, types.NamespacedName{
		Namespace: svcNamespace,
		Name:      svcName,
	}, pObj)

	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}

		return err
	}

	vClient := ctx.VirtualClient
	vObj := &corev1.Service{}
	err = vClient.Get(ctx.Context, vSvcToSync, vObj)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}

		return err
	}

	if vObj.Spec.ClusterIP != pObj.Spec.ClusterIP || !equality.Semantic.DeepEqual(vObj.Spec.ClusterIPs, pObj.Spec.ClusterIPs) {
		newService := vObj.DeepCopy()
		newService.Spec.ClusterIP = pObj.Spec.ClusterIP
		newService.Spec.ClusterIPs = pObj.Spec.ClusterIPs
		newService.Spec.IPFamilies = pObj.Spec.IPFamilies

		// delete & create with correct ClusterIP
		err = vClient.Delete(ctx.Context, vObj)
		if err != nil {
			return err
		}

		newService.ResourceVersion = ""

		// create the new service with the correct cluster ip
		err = vClient.Create(ctx.Context, newService)
		if err != nil {
			return err
		}
	}

	return nil
}
