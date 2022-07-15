package ephemeral_containers

import (
	"encoding/json"
	"fmt"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// AddEphemeralContainer runs an EphemeralContainer in the target Pod for use as a debug container
func AddEphemeralContainer(ctx *synccontext.SyncContext, physicalClusterClient kubernetes.Interface, physicalPod *corev1.Pod, virtualPod *corev1.Pod, profile string) (*corev1.Pod, string, error) {
	if len(virtualPod.Spec.EphemeralContainers) > 0 {
		podJS, err := json.Marshal(physicalPod)
		if err != nil {
			return nil, "", fmt.Errorf("error creating JSON for physicalPod: %v", err)
		}
		debugPod, debugContainer, err := getEphemeralContainer(physicalPod, virtualPod, profile)
		if err != nil {
			return nil, "", err
		}
		klog.V(2).Infof("new ephemeral container: %#v", debugContainer)

		debugJS, err := json.Marshal(debugPod)
		if err != nil {
			return nil, "", fmt.Errorf("error creating JSON for debug container: %v", err)
		}

		patch, err := strategicpatch.CreateTwoWayMergePatch(podJS, debugJS, physicalPod)
		if err != nil {
			return nil, "", fmt.Errorf("error creating patch to add debug container: %v", err)
		}
		klog.V(2).Infof("generated strategic merge patch for debug container: %s", patch)

		pods := physicalClusterClient.CoreV1().Pods(physicalPod.Namespace)
		result, err := pods.Patch(ctx.Context, physicalPod.Name, types.StrategicMergePatchType, patch, metav1.PatchOptions{}, "ephemeralcontainers")
		if err != nil {
			// The apiserver will return a 404 when the EphemeralContainers feature is disabled because the `/ephemeralcontainers` subresource
			// is missing. Unlike the 404 returned by a missing physicalPod, the status details will be empty.
			if serr, ok := err.(*errors.StatusError); ok && serr.Status().Reason == metav1.StatusReasonNotFound && serr.ErrStatus.Details.Name == "" {
				return nil, "", fmt.Errorf("ephemeral containers are disabled for this cluster (error from server: %q)", err)
			}
			// The Kind used for the /ephemeralcontainers subresource changed in 1.22. When presented with an unexpected
			// Kind the api server will respond with a not-registered error. When this happens we can optimistically try
			// using the old API.
			if runtime.IsNotRegisteredError(err) {
				klog.V(1).Infof("Falling back to legacy API because server returned error: %v", err)
				return addEphemeralContainerLegacy(ctx, physicalClusterClient, physicalPod, debugContainer)
			}
			return nil, "", err
		}
		return result, debugContainer.Name, nil
	}
	return nil, "", nil
}

// addEphemeralContainerLegacy adds an ephemeral container using the pre-1.22 /ephemeralcontainers API
// This may be removed when we no longer wish to support releases prior to 1.22.
func addEphemeralContainerLegacy(ctx *synccontext.SyncContext, physicalClusterClient kubernetes.Interface, physicalPod *corev1.Pod, debugContainer *corev1.EphemeralContainer) (*corev1.Pod, string, error) {
	// We no longer have the v1.EphemeralContainers Kind since it was removed in 1.22, but
	// we can present a JSON 6902 patch that the api server will apply.
	patch, err := json.Marshal([]map[string]interface{}{{
		"op":    "add",
		"path":  "/ephemeralContainers/-",
		"value": debugContainer,
	}})
	if err != nil {
		return nil, "", fmt.Errorf("error creating JSON 6902 patch for old /ephemeralcontainers API: %s", err)
	}

	result := physicalClusterClient.CoreV1().RESTClient().Patch(types.JSONPatchType).
		Namespace(physicalPod.Namespace).
		Resource("pods").
		Name(physicalPod.Name).
		SubResource("ephemeralcontainers").
		Body(patch).
		Do(ctx.Context)
	if err := result.Error(); err != nil {
		return nil, "", err
	}

	newPod, err := physicalClusterClient.CoreV1().Pods(physicalPod.Namespace).Get(ctx.Context, physicalPod.Name, metav1.GetOptions{})
	if err != nil {
		return nil, "", err
	}

	return newPod, debugContainer.Name, nil
}

// getEphemeralContainer returns a debugging pod and an EphemeralContainer suitable for use as a debug container
// in the given pod.
func getEphemeralContainer(physicalPod *corev1.Pod, virtualPod *corev1.Pod, profile string) (*corev1.Pod, *corev1.EphemeralContainer, error) {
	var ephemeralContainer corev1.EphemeralContainer
	ephemeralContainer = virtualPod.Spec.EphemeralContainers[len(virtualPod.Spec.EphemeralContainers)-1]
	copied := physicalPod.DeepCopy()
	ephemeralContainer.TargetContainerName = ""
	copied.Spec.EphemeralContainers = append(copied.Spec.EphemeralContainers, ephemeralContainer)

	if profile != "" {
		applier, err := NewProfileApplier(profile)
		if err != nil {
			return nil, nil, err
		}
		if err := applier.Apply(copied, ephemeralContainer.Name, copied); err != nil {
			return nil, nil, err
		}
	}

	return copied, &ephemeralContainer, nil
}
