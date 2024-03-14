package specialservices

import (
	"slices"

	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

var (
	DefaultKubernetesSvcKey = types.NamespacedName{
		Name:      DefaultKubernetesSVCName,
		Namespace: DefaultKubernetesSVCNamespace,
	}
)

const (
	DefaultKubernetesSVCName      = "kubernetes"
	DefaultKubernetesSVCNamespace = "default"
)

type ServicePortTranslator func(ports []corev1.ServicePort) []corev1.ServicePort

func SyncKubernetesService(
	ctx *synccontext.SyncContext,
	svcNamespace,
	svcName string,
	vSvcToSync types.NamespacedName,
	svcPortTranslator ServicePortTranslator,
) error {
	// get physical service
	pObj := &corev1.Service{}
	err := ctx.CurrentNamespaceClient.Get(ctx.Context, types.NamespacedName{
		Namespace: svcNamespace,
		Name:      svcName,
	}, pObj)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	// get virtual service
	vObj := &corev1.Service{}
	err = ctx.VirtualClient.Get(ctx.Context, vSvcToSync, vObj)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	// detect if cluster ips changed
	clusterIPsChanged := vObj.Spec.ClusterIP != pObj.Spec.ClusterIP || !slices.Equal(vObj.Spec.ClusterIPs, pObj.Spec.ClusterIPs)

	translatedPorts := svcPortTranslator(pObj.Spec.Ports)
	if clusterIPsChanged || !equality.Semantic.DeepEqual(vObj.Spec.Ports, translatedPorts) {
		newService := vObj.DeepCopy()
		newService.Spec.ClusterIP = pObj.Spec.ClusterIP
		newService.Spec.ClusterIPs = pObj.Spec.ClusterIPs
		newService.Spec.IPFamilies = pObj.Spec.IPFamilies
		newService.Spec.IPFamilyPolicy = pObj.Spec.IPFamilyPolicy
		newService.Spec.Ports = translatedPorts
		if clusterIPsChanged {
			// delete & create with correct ClusterIP
			err = ctx.VirtualClient.Delete(ctx.Context, vObj)
			if err != nil {
				return err
			}

			// make sure we don't set the resource version during create
			newService.ResourceVersion = ""

			// create the new service with the correct cluster ip
			err = ctx.VirtualClient.Create(ctx.Context, newService)
			if err != nil {
				return err
			}
		} else {
			// delete & create with correct ClusterIP
			err = ctx.VirtualClient.Update(ctx.Context, newService)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
