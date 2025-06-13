package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	vClusterMetadataPrefix = "vcluster.loft.sh/"
)

// CleanupSyncedNamespaces identifies all physical namespaces that were created or managed by vCluster.
//  1. Namespaces created by the vCluster: These are deleted directly.
//  2. Namespaces "imported" into the vCluster: For these, we remove all vCluster-related
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

	managedNamespaces, err := getManagedNamespaces(ctx, k8sClient, mainPhysicalNamespace, vClusterName, logger)
	if err != nil {
		return err
	}
	if len(managedNamespaces) == 0 {
		logger.Infof("No managed namespaces found for vCluster '%s' in namespace '%s'.", vClusterName, mainPhysicalNamespace)
		return nil
	}

	dynClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return err
	}

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
			err := deleteAndLogNamespace(ctx, k8sClient, ns.Name, logger)
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
	if err := cleanupObjectMetadata(ctx, nsInterface, &unstructured.Unstructured{Object: unstructuredNS}, logger); err != nil {
		errs = append(errs, err)
	}

	// Then, iterate over all discovered namespaced resource types and clean them up.
	for _, gvr := range resources {
		err := cleanupObjectsByGVR(ctx, dynClient, ns.Name, gvr, logger)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func getManagedNamespaces(ctx context.Context, k8sClient *kubernetes.Clientset, mainPhysicalNamespace, vClusterName string, logger log.Logger) ([]corev1.Namespace, error) {
	labelSelector := translate.MarkerLabel + "=" + translate.SafeConcatName(mainPhysicalNamespace, "x", vClusterName)
	nsList, err := k8sClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces with selector '%s': %w", labelSelector, err)
	}
	return nsList.Items, nil
}

func deleteAndLogNamespace(ctx context.Context, k8sClient *kubernetes.Clientset, nsName string, logger log.Logger) error {
	logger.Infof("Deleting virtual cluster namespace '%s'", nsName)
	err := k8sClient.CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			logger.Infof("Virtual cluster namespace '%s' was already deleted.", nsName)
			return nil
		}
		return fmt.Errorf("failed to delete namespace '%s': %w", nsName, err)
	}
	logger.Infof("Successfully deleted virtual cluster namespace '%s'", nsName)
	return nil
}

// isImportedNamespace checks if a namespace was imported
func isImportedNamespace(ns *corev1.Namespace) bool {
	return ns.Annotations != nil && ns.Annotations[translate.ImportedMarkerAnnotation] == "true"
}

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

func cleanupObjectsByGVR(ctx context.Context, dynClient dynamic.Interface, namespace string, gvr schema.GroupVersionResource, logger log.Logger) error {
	resourceInterface := dynClient.Resource(gvr).Namespace(namespace)
	objects, err := resourceInterface.List(ctx, metav1.ListOptions{})
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
		if err := cleanupObjectMetadata(ctx, resourceInterface, &objects.Items[i], logger); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func cleanupObjectMetadata(ctx context.Context, resInterface dynamic.ResourceInterface, obj *unstructured.Unstructured, logger log.Logger) error {
	type jsonPatch struct {
		Op   string `json:"op"`
		Path string `json:"path"`
	}
	var patch []jsonPatch

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

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return err
	}

	// Apply the patch to the object.
	_, err = resInterface.Patch(ctx, obj.GetName(), types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			// Object might have been deleted by another process in the meantime, which is fine.
			return nil
		}
		return err
	}

	return nil
}
