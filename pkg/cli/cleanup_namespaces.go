package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	vClusterMetadataPrefix = "vcluster.loft.sh/"

	namespaceDeletionInterval = 2 * time.Second
	namespaceDeletionTimeout  = 3 * time.Minute
)

// CleanupSyncedNamespaces identifies all physical namespaces that were managed by this vCluster.
//  1. Namespaces created on host as result of vCluster syncing will be deleted.
//  2. Namespaces imported into the vCluster from host will be cleaned up by removing all vCluster-related
//     metadata from the namespace itself and all resources within it.
func CleanupSyncedNamespaces(
	ctx context.Context,
	mainPhysicalNamespace string,
	vClusterName string,
	restConfig *rest.Config,
	k8sClient *kubernetes.Clientset,
	logger log.Logger,
) error {
	logger.Infof("Starting cleanup of vCluster '%s' namespaces.", vClusterName)

	// Fetch namespaces managed by this vCluster.
	managedNamespaces, err := getManagedNamespaces(ctx, k8sClient, mainPhysicalNamespace, vClusterName)
	if err != nil {
		return err
	}
	if len(managedNamespaces) == 0 {
		logger.Infof("No managed namespaces found for vCluster '%s' in namespace '%s'.", vClusterName, mainPhysicalNamespace)
		return nil
	}

	// Build dynamic client for cleaning up imported resources.
	dynClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	// Fetch all API namespaced resources.
	namespacedResources, err := discoverNamespacedResources(k8sClient.Discovery(), logger)
	if err != nil {
		return fmt.Errorf("failed to discover API resources: %w", err)
	}

	var errs []error
	for _, ns := range managedNamespaces {
		if isImportedNamespace(&ns) {
			if err := cleanupImportedNamespace(ctx, dynClient, &ns, namespacedResources, logger); err != nil {
				errs = append(errs, err)
			}
		} else {
			err := deleteNamespace(ctx, k8sClient, ns.Name, logger)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("cleanup of vCluster '%s' namespaces finished with errors: %w", vClusterName, errors.Join(errs...))
	}

	logger.Infof("Cleanup of vCluster '%s' namespaces finished.", vClusterName)
	return nil
}

// cleanupImportedNamespace cleans up metadata of namespace resource and all resources running in this namespace.
func cleanupImportedNamespace(ctx context.Context, dynClient dynamic.Interface, ns *corev1.Namespace, resources []schema.GroupVersionResource, logger log.Logger) error {
	logger.Infof("Namespace '%s' was imported, cleaning up its resources and metadata...", ns.Name)

	var errs []error

	// First cleanup metadata of the namespace itself.
	nsGVR := schema.GroupVersionResource{Version: "v1", Resource: "namespaces"}
	nsInterface := dynClient.Resource(nsGVR)
	unstructuredNS, err := runtime.DefaultUnstructuredConverter.ToUnstructured(ns)
	if err != nil {
		return err
	}
	if err := cleanupObjectMetadata(ctx, nsInterface, &unstructured.Unstructured{Object: unstructuredNS}); err != nil {
		errs = append(errs, err)
	}

	// Then, iterate over all discovered namespaced resource types and clean them up.
	for _, gvr := range resources {
		err := cleanupObjectsByGVR(ctx, dynClient, ns.Name, gvr)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// getManagedNamespaces fetches all namespaces managed by this vCluster.
func getManagedNamespaces(ctx context.Context, k8sClient *kubernetes.Clientset, mainPhysicalNamespace, vClusterName string) ([]corev1.Namespace, error) {
	labelSelector := translate.MarkerLabel + "=" + translate.SafeConcatName(mainPhysicalNamespace, "x", vClusterName)
	nsList, err := k8sClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces with selector '%s': %w", labelSelector, err)
	}
	return nsList.Items, nil
}

// deleteNamespace deletes a namespace and waits until it is fully terminated.
func deleteNamespace(
	ctx context.Context,
	k8sClient *kubernetes.Clientset,
	nsName string,
	logger log.Logger,
) error {
	logger.Infof("Issuing delete for namespace '%s'", nsName)
	err := k8sClient.CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			logger.Infof("Namespace '%s' was already deleted.", nsName)
			return nil
		}
		return fmt.Errorf("failed to delete namespace '%s': %w", nsName, err)
	}

	logger.Infof("Waiting for namespace '%s' to be fully terminated...", nsName)
	err = wait.PollUntilContextTimeout(ctx, namespaceDeletionInterval, namespaceDeletionTimeout, true, func(ctx context.Context) (bool, error) {
		// Check for the namespace's existence.
		_, err := k8sClient.CoreV1().Namespaces().Get(ctx, nsName, metav1.GetOptions{})
		if kerrors.IsNotFound(err) {
			// The namespace is gone.
			return true, nil
		}
		if err != nil {
			return false, err
		}
		// The namespace still exists. Continue polling.
		return false, nil
	})

	if err != nil {
		return fmt.Errorf("failed while waiting for namespace '%s' to be deleted: %w", nsName, err)
	}

	logger.Infof("Successfully deleted virtual cluster namespace '%s'", nsName)
	return nil
}

// isImportedNamespace checks if a namespace was imported
func isImportedNamespace(ns *corev1.Namespace) bool {
	return ns.Annotations != nil && ns.Annotations[translate.ImportedMarkerAnnotation] == "true"
}

// discoverNamespacedResources retrieves list of namespaced GVRs from API.
func discoverNamespacedResources(discoveryClient discovery.DiscoveryInterface, logger log.Logger) ([]schema.GroupVersionResource, error) {
	resourceList, err := discoveryClient.ServerPreferredResources()
	if err != nil {
		if discovery.IsGroupDiscoveryFailedError(err) {
			logger.Infof("Warning: Could not discover all API groups, some resources may not be cleaned up: %v", err)
		} else {
			return nil, fmt.Errorf("failed to discover server preferred resources: %w", err)
		}
	}

	var patchableResources []schema.GroupVersionResource
	for _, list := range resourceList {
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			logger.Infof("Skipping resource list with invalid GroupVersion '%s': %v", list.GroupVersion, err)
			continue
		}

		for _, resource := range list.APIResources {
			// Resource must be namespaced.
			if !resource.Namespaced {
				continue
			}

			// Resource must be listable and patchable.
			var hasList, hasPatch bool
			for _, verb := range resource.Verbs {
				if verb == "list" {
					hasList = true
				}
				if verb == "patch" {
					hasPatch = true
				}
			}

			if hasList && hasPatch {
				patchableResources = append(patchableResources, gv.WithResource(resource.Name))
			}
		}
	}

	logger.Debugf("Discovered %d namespaced resource types that can be cleaned up.", len(patchableResources))
	return patchableResources, nil
}

// cleanupObjectsByGVR lists all objects of given GVR and runs metadata cleanup on them.
func cleanupObjectsByGVR(ctx context.Context, dynClient dynamic.Interface, namespace string, gvr schema.GroupVersionResource) error {
	resourceClient := dynClient.Resource(gvr).Namespace(namespace)
	objects, err := resourceClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) || kerrors.IsForbidden(err) {
			return nil
		}
		return err
	}

	if len(objects.Items) == 0 {
		return nil
	}

	var errs []error
	for i := range objects.Items {
		if err := cleanupObjectMetadata(ctx, resourceClient, &objects.Items[i]); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// cleanupObjectMetadata checks given object for vCluster labels and annotations and runs patch operation removing them.
func cleanupObjectMetadata(ctx context.Context, client dynamic.ResourceInterface, obj *unstructured.Unstructured) error {
	type jsonPatch struct {
		Op   string `json:"op"`
		Path string `json:"path"`
	}
	var patch []jsonPatch

	// labels/annotations need to be escaped for patches
	jsonpointer := strings.NewReplacer("~", "~0", "/", "~1")

	// Check all labels for the prefix and add a "remove" operation to the patch if found.
	for labelKey := range obj.GetLabels() {
		if strings.HasPrefix(labelKey, vClusterMetadataPrefix) {
			escapedLabel := jsonpointer.Replace(labelKey)
			patch = append(patch, jsonPatch{Op: "remove", Path: fmt.Sprintf("/metadata/labels/%s", escapedLabel)})
		}
	}

	// Check all annotations for the prefix and add a "remove" operation to the patch if found.
	for annotationKey := range obj.GetAnnotations() {
		if strings.HasPrefix(annotationKey, vClusterMetadataPrefix) {
			escapedAnnotation := jsonpointer.Replace(annotationKey)
			patch = append(patch, jsonPatch{Op: "remove", Path: fmt.Sprintf("/metadata/annotations/%s", escapedAnnotation)})
		}
	}

	// If no matching metadata was found, there's nothing to do.
	if len(patch) == 0 {
		return nil
	}

	// Create patch.
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return err
	}

	// Apply the patch to the object.
	_, err = client.Patch(ctx, obj.GetName(), types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			// Object might have been deleted by another process in the meantime, which is fine.
			return nil
		}
		return err
	}

	return nil
}
