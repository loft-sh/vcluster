package specialservices

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	DefaultKubernetesSVCName      = "kubernetes"
	DefaultKubernetesSVCNamespace = "default"
)

type ServicePortTranslator func(ports []corev1.ServicePort) []corev1.ServicePort

func SyncKubernetesService(ctx context.Context,
	vClient,
	pClient client.Client,
	svcNamespace,
	svcName string,
	vSvcToSync types.NamespacedName,
	svcPortTranslator ServicePortTranslator) error {
	// get physical service
	pObj := &corev1.Service{}
	err := pClient.Get(ctx, types.NamespacedName{
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
	err = vClient.Get(ctx, vSvcToSync, vObj)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	translatedPorts := svcPortTranslator(pObj.Spec.Ports)
	if vObj.Spec.ClusterIP != pObj.Spec.ClusterIP || !equality.Semantic.DeepEqual(vObj.Spec.Ports, translatedPorts) {
		newService := vObj.DeepCopy()
		newService.Spec.ClusterIP = pObj.Spec.ClusterIP
		newService.Spec.ClusterIPs = pObj.Spec.ClusterIPs
		newService.Spec.IPFamilies = pObj.Spec.IPFamilies
		newService.Spec.Ports = translatedPorts
		if vObj.Spec.ClusterIP != pObj.Spec.ClusterIP || !equality.Semantic.DeepEqual(vObj.Spec.ClusterIPs, pObj.Spec.ClusterIPs) {

			// delete & create with correct ClusterIP
			err = vClient.Delete(ctx, vObj)
			if err != nil {
				return err
			}

			// make sure we don't set the resource version during create
			newService.ResourceVersion = ""

			// create the new service with the correct cluster ip
			err = vClient.Create(ctx, newService)
			if err != nil {
				return err
			}
		} else {
			// delete & create with correct ClusterIP
			err = vClient.Update(ctx, newService)
			if err != nil {
				return err
			}
		}
	}

	return nil

}
