package specialservices

import (
	"fmt"
	"slices"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
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
	err := ctx.CurrentNamespaceClient.Get(ctx, types.NamespacedName{
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
	err = ctx.VirtualClient.Get(ctx, vSvcToSync, vObj)
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
			err = recreateService(ctx, vSvcToSync, vObj, newService)
			if err != nil {
				return err
			}
		} else {
			// delete & create with correct ClusterIP
			err = ctx.VirtualClient.Update(ctx, newService)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// recreateAttempts bounds how often the delete and create is repeated when the kube-apiserver
// recreates the kubernetes service in between.
const recreateAttempts = 5

// recreateService deletes the virtual service and creates it again with the cluster ips of the host
// service. The kube-apiserver recreates default/kubernetes on its own as soon as it is gone, so the
// create can lose that race and fail with AlreadyExists. In that case the recreated service is
// deleted and the create is repeated until the service that exists carries the wanted cluster ips.
func recreateService(ctx *synccontext.SyncContext, key types.NamespacedName, vObj, newService *corev1.Service) error {
	var err error
	for attempt := 0; attempt < recreateAttempts; attempt++ {
		err = ctx.VirtualClient.Delete(ctx, vObj)
		if err != nil && !kerrors.IsNotFound(err) {
			return err
		}

		// make sure we don't set the resource version during create
		toCreate := newService.DeepCopy()
		toCreate.ResourceVersion = ""
		toCreate.UID = ""
		err = ctx.VirtualClient.Create(ctx, toCreate)
		if err == nil {
			return nil
		} else if !kerrors.IsAlreadyExists(err) {
			return err
		}

		// somebody else created the service in between, check whether it already has the wanted cluster ips
		current := &corev1.Service{}
		getErr := ctx.VirtualClient.Get(ctx, key, current)
		if getErr != nil {
			if kerrors.IsNotFound(getErr) {
				continue
			}
			return getErr
		}
		if current.Spec.ClusterIP == newService.Spec.ClusterIP && slices.Equal(current.Spec.ClusterIPs, newService.Spec.ClusterIPs) {
			return nil
		}

		vObj = current
	}

	return fmt.Errorf("recreate service %s after %d attempts: %w", key.String(), recreateAttempts, err)
}
