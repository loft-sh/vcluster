package specialservices

import (
	"slices"

	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
)

var (
	VclusterProxyMetricsSvcKey = types.NamespacedName{
		Name:      "metrics-server",
		Namespace: "kube-system",
	}
)

const (
	PhysicalSvcSelectorKeyApp              = "app"
	PhysicalSvcSelectorKeyRelease          = "release"
	PhysicalMetricsServerServiceNameSuffix = "-metrics-proxy"
)

func SyncVclusterProxyService(ctx *synccontext.SyncContext,
	_,
	svcName string,
	vSvcToSync types.NamespacedName,
	_ ServicePortTranslator,
) error {
	pClient := ctx.PhysicalClient
	// get physical service
	pObj := &corev1.Service{}
	err := pClient.Get(ctx.Context, types.NamespacedName{
		Namespace: translate.Default.PhysicalNamespace(vSvcToSync.Namespace),
		Name:      svcName + PhysicalMetricsServerServiceNameSuffix,
	}, pObj)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}

		return err
	}

	// check if pobject has the expected selectors, if not update
	// and make it point to the syncer pod
	expectedPhysicalSvcSelectors := map[string]string{
		PhysicalSvcSelectorKeyApp:     "vcluster",
		PhysicalSvcSelectorKeyRelease: svcName,
	}

	if !equality.Semantic.DeepEqual(pObj.Spec.Selector, expectedPhysicalSvcSelectors) {
		pObj.Spec.Selector = expectedPhysicalSvcSelectors
		err = pClient.Update(ctx.Context, pObj)
		if err != nil {
			klog.Errorf("error updating physical metrics server service object %v", err)
			return err
		}
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

	if vObj.Spec.ClusterIP != pObj.Spec.ClusterIP || !slices.Equal(vObj.Spec.ClusterIPs, pObj.Spec.ClusterIPs) {
		newService := vObj.DeepCopy()
		newService.Spec.ClusterIP = pObj.Spec.ClusterIP
		newService.Spec.ClusterIPs = pObj.Spec.ClusterIPs
		newService.Spec.IPFamilies = pObj.Spec.IPFamilies
		newService.Spec.IPFamilyPolicy = pObj.Spec.IPFamilyPolicy

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
