package pods

import (
	"fmt"

	translatepods "github.com/loft-sh/vcluster/pkg/controllers/resources/pods/translate"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// syncEphemeralContainers updates the ephemeral containers in the physical pod
func (s *podSyncer) syncEphemeralContainers(ctx *synccontext.SyncContext, physicalClusterClient kubernetes.Interface, physicalPod *corev1.Pod, virtualPod *corev1.Pod) (bool, error) {
	if len(virtualPod.Spec.EphemeralContainers) == 0 {
		return false, nil
	}

	// make sure when changing the physical pod we don't get any unwanted side effects
	physicalPod = physicalPod.DeepCopy()

	// copy the ephemeral containers not found in the physical pod
	ephemeralContainers := virtualPod.Spec.EphemeralContainers
	newContainers := []corev1.EphemeralContainer{}
	for _, ephemeralContainer := range ephemeralContainers {
		// check if found
		found := false
		for _, existingEphemeralContainer := range physicalPod.Spec.EphemeralContainers {
			if existingEphemeralContainer.Name == ephemeralContainer.Name {
				found = true
				break
			}
		}
		if found {
			continue
		}

		newContainers = append(newContainers, ephemeralContainer)
	}

	// if there are no new ephemeral containers, we skip
	// important to note that we do not check the other way around and sync ephemeral containers from host to virtual
	if len(newContainers) == 0 {
		return false, nil
	}

	// translate the ephemeral containers
	kubeIP, _, ptrServiceList, err := s.getK8sIPDNSIPServiceList(ctx, virtualPod)
	if err != nil {
		return false, err
	}
	serviceEnv := translatepods.ServicesToEnvironmentVariables(virtualPod.Spec.EnableServiceLinks, ptrServiceList, kubeIP)
	for _, ephemeralContainer := range newContainers {
		// not found, we add it to physical
		envVar, envFrom, err := s.podTranslator.TranslateContainerEnv(ctx, ephemeralContainer.Env, ephemeralContainer.EnvFrom, virtualPod, serviceEnv)
		if err != nil {
			return false, fmt.Errorf("translate container env: %w", err)
		}
		ephemeralContainer.Env = envVar
		ephemeralContainer.EnvFrom = envFrom
		physicalPod.Spec.EphemeralContainers = append(physicalPod.Spec.EphemeralContainers, ephemeralContainer)
	}

	// do the actual update
	ctx.Log.Infof("Update ephemeral containers for pod %s/%s", physicalPod.Namespace, physicalPod.Name)
	_, err = physicalClusterClient.CoreV1().Pods(physicalPod.Namespace).UpdateEphemeralContainers(ctx, physicalPod.Name, physicalPod, metav1.UpdateOptions{})
	if err != nil {
		// The api-server will return a 404 when the EphemeralContainers feature is disabled because the `/ephemeralcontainers` subresource
		// is missing. Unlike the 404 returned by a missing physicalPod, the status details will be empty.
		if kerrors.IsNotFound(err) {
			return false, fmt.Errorf("ephemeral containers are disabled for this cluster (error from server: %w)", err)
		}

		return false, fmt.Errorf("update ephemeral containers: %w", err)
	}

	return true, nil
}
