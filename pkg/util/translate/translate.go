package translate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/util/stringutil"
	"github.com/pkg/errors"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1clientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

const (
	SkipBackSyncInMultiNamespaceMode = "vcluster.loft.sh/skip-backsync"
)

var Owner client.Object

func CopyObjectWithName[T client.Object](obj T, name types.NamespacedName, setOwner bool, excludedAnnotations ...string) T {
	target := obj.DeepCopyObject().(T)

	// reset metadata & translate name and namespace
	ResetObjectMetadata(target)
	target.SetName(name.Name)
	if obj.GetNamespace() != "" {
		target.SetNamespace(name.Namespace)

		// set owning stateful set if defined
		if setOwner && Owner != nil {
			target.SetOwnerReferences(GetOwnerReference(obj))
		}
	}

	stripExcludedAnnotations(target, excludedAnnotations...)
	return target
}

func HostMetadata[T client.Object](vObj T, name types.NamespacedName, excludedAnnotations ...string) T {
	pObj := CopyObjectWithName(vObj, name, true, excludedAnnotations...)
	stripExcludedAnnotations(vObj, excludedAnnotations...)
	pObj.SetAnnotations(HostAnnotations(vObj, pObj, excludedAnnotations...))
	pObj.SetLabels(HostLabels(vObj, nil))
	return pObj
}

func VirtualMetadata[T client.Object](pObj T, name types.NamespacedName, excludedAnnotations ...string) T {
	vObj := CopyObjectWithName(pObj, name, false, excludedAnnotations...)
	vObj.SetAnnotations(VirtualAnnotations(pObj, nil, excludedAnnotations...))
	vObj.SetLabels(VirtualLabels(pObj, nil))
	return vObj
}

func stripExcludedAnnotations(obj client.Object, excludedAnnotations ...string) {
	annotations := obj.GetAnnotations()
	for k := range annotations {
		if stringutil.Contains(excludedAnnotations, k) {
			delete(annotations, k)
		}
	}
	obj.SetAnnotations(annotations)
}

func VirtualAnnotations(pObj, vObj client.Object, excluded ...string) map[string]string {
	excluded = append(excluded, NameAnnotation, NamespaceAnnotation, HostNameAnnotation, HostNamespaceAnnotation, UIDAnnotation, KindAnnotation, ManagedAnnotationsAnnotation, ManagedLabelsAnnotation)
	var toAnnotations map[string]string
	if vObj != nil {
		toAnnotations = vObj.GetAnnotations()
	}

	return copyMaps(pObj.GetAnnotations(), toAnnotations, func(key string) bool {
		return exists(excluded, key)
	})
}

func copyMaps(fromMap, toMap map[string]string, excludeKey func(string) bool) map[string]string {
	retMap := map[string]string{}
	for k, v := range fromMap {
		if excludeKey != nil && excludeKey(k) {
			continue
		}

		retMap[k] = v
	}

	for key := range toMap {
		if excludeKey != nil && excludeKey(key) {
			value, ok := toMap[key]
			if ok {
				retMap[key] = value
			}
		}
	}

	return retMap
}

func HostAnnotations(vObj, pObj client.Object, excluded ...string) map[string]string {
	excluded = append(excluded, NameAnnotation, HostNameAnnotation, HostNamespaceAnnotation, UIDAnnotation, KindAnnotation, NamespaceAnnotation)
	toAnnotations := map[string]string{}
	if pObj != nil {
		toAnnotations = pObj.GetAnnotations()
		if toAnnotations == nil {
			toAnnotations = map[string]string{}
		}
	}

	retMap := applyAnnotations(vObj.GetAnnotations(), toAnnotations, excluded...)
	addHostAnnotations(retMap, vObj, pObj)

	return retMap
}

func addHostAnnotations(retMap map[string]string, vObj, pObj client.Object) {
	retMap[NameAnnotation] = vObj.GetName()
	retMap[UIDAnnotation] = string(vObj.GetUID())
	if pObj != nil {
		retMap[HostNameAnnotation] = pObj.GetName()
		if pObj.GetNamespace() != "" {
			retMap[HostNamespaceAnnotation] = pObj.GetNamespace()
		}
	}
	if vObj.GetNamespace() == "" {
		delete(retMap, NamespaceAnnotation)
	} else {
		retMap[NamespaceAnnotation] = vObj.GetNamespace()
	}

	gvk, err := apiutil.GVKForObject(vObj, scheme.Scheme)
	if err == nil {
		retMap[KindAnnotation] = gvk.String()
	}
}

func GetOwnerReference(object client.Object) []metav1.OwnerReference {
	if Owner == nil || Owner.GetName() == "" || Owner.GetUID() == "" {
		return nil
	}

	typeAccessor, err := meta.TypeAccessor(Owner)
	if err != nil || typeAccessor.GetAPIVersion() == "" || typeAccessor.GetKind() == "" {
		return nil
	}

	isController := false
	if object != nil {
		ctrl := metav1.GetControllerOf(object)
		isController = ctrl != nil
	}
	return []metav1.OwnerReference{
		{
			APIVersion: typeAccessor.GetAPIVersion(),
			Kind:       typeAccessor.GetKind(),
			Name:       Owner.GetName(),
			UID:        Owner.GetUID(),
			Controller: &isController,
		},
	}
}

func SafeConcatName(name ...string) string {
	fullPath := strings.Join(name, "-")
	if len(fullPath) > 63 {
		digest := sha256.Sum256([]byte(fullPath))
		return strings.ReplaceAll(fullPath[0:52]+"-"+hex.EncodeToString(digest[0:])[0:10], ".-", "-")
	}
	return fullPath
}

func Split(s, sep string) (string, string) {
	parts := strings.SplitN(s, sep, 2)
	return strings.TrimSpace(parts[0]), strings.TrimSpace(safeIndex(parts, 1))
}

func safeIndex(parts []string, idx int) string {
	if len(parts) <= idx {
		return ""
	}
	return parts[idx]
}

func exists(a []string, k string) bool {
	for _, i := range a {
		if i == k {
			return true
		}
	}

	return false
}

// ResetObjectMetadata resets the objects metadata except name, namespace and annotations
func ResetObjectMetadata(obj metav1.Object) {
	obj.SetGenerateName("")
	obj.SetSelfLink("")
	obj.SetUID("")
	obj.SetResourceVersion("")
	obj.SetGeneration(0)
	obj.SetCreationTimestamp(metav1.Time{})
	obj.SetDeletionTimestamp(nil)
	obj.SetDeletionGracePeriodSeconds(nil)
	obj.SetOwnerReferences(nil)
	obj.SetFinalizers(nil)
	obj.SetManagedFields(nil)
}

func ApplyMetadata(fromAnnotations map[string]string, toAnnotations map[string]string, fromLabels map[string]string, toLabels map[string]string, excludeAnnotations ...string) (labels map[string]string, annotations map[string]string) {
	mergedAnnotations := applyAnnotations(fromAnnotations, toAnnotations, excludeAnnotations...)
	return applyLabels(fromLabels, toLabels, mergedAnnotations)
}

func applyAnnotations(fromAnnotations map[string]string, toAnnotations map[string]string, excludeAnnotations ...string) map[string]string {
	if toAnnotations == nil {
		toAnnotations = map[string]string{}
	}

	excludedKeys := []string{ManagedAnnotationsAnnotation, ManagedLabelsAnnotation}
	excludedKeys = append(excludedKeys, excludeAnnotations...)
	mergedAnnotations, managedKeys := applyMaps(fromAnnotations, toAnnotations, ApplyMapsOptions{
		ManagedKeys: strings.Split(toAnnotations[ManagedAnnotationsAnnotation], "\n"),
		ExcludeKeys: excludedKeys,
	})
	if managedKeys == "" {
		delete(mergedAnnotations, ManagedAnnotationsAnnotation)
	} else {
		mergedAnnotations[ManagedAnnotationsAnnotation] = managedKeys
	}

	return mergedAnnotations
}

func applyLabels(fromLabels map[string]string, toLabels map[string]string, toAnnotations map[string]string) (labels map[string]string, annotations map[string]string) {
	if toAnnotations == nil {
		toAnnotations = map[string]string{}
	}

	mergedLabels, managedKeys := applyMaps(fromLabels, toLabels, ApplyMapsOptions{
		ManagedKeys: strings.Split(toAnnotations[ManagedLabelsAnnotation], "\n"),
		ExcludeKeys: []string{ManagedAnnotationsAnnotation, ManagedLabelsAnnotation},
	})
	mergedAnnotations := map[string]string{}
	for k, v := range toAnnotations {
		mergedAnnotations[k] = v
	}
	if managedKeys == "" {
		delete(mergedAnnotations, ManagedLabelsAnnotation)
	} else {
		mergedAnnotations[ManagedLabelsAnnotation] = managedKeys
	}

	return mergedLabels, mergedAnnotations
}

type ApplyMapsOptions struct {
	ManagedKeys []string
	ExcludeKeys []string
}

func applyMaps(fromMap, toMap map[string]string, opts ApplyMapsOptions) (map[string]string, string) {
	retMap := map[string]string{}
	managedKeys := []string{}
	for k, v := range fromMap {
		if exists(opts.ExcludeKeys, k) {
			continue
		}

		retMap[k] = v
		managedKeys = append(managedKeys, k)
	}

	for key, value := range toMap {
		if exists(opts.ExcludeKeys, key) {
			retMap[key] = value
			continue
		} else if exists(managedKeys, key) || exists(opts.ManagedKeys, key) {
			continue
		}

		retMap[key] = value
	}

	sort.Strings(managedKeys)
	managedKeysStr := strings.Join(managedKeys, "\n")
	return retMap, managedKeysStr
}

func EnsureCRDFromPhysicalCluster(ctx context.Context, pConfig *rest.Config, vConfig *rest.Config, groupVersionKind schema.GroupVersionKind) (bool, bool, error) {
	var isClusterScoped, hasStatusSubresource bool

	vClient, err := apiextensionsv1clientset.NewForConfig(vConfig)
	if err != nil {
		return isClusterScoped, hasStatusSubresource, err
	}

	if apiResource, err := KindExists(vConfig, groupVersionKind); err == nil {
		// (ThomasK33): Check if the CRD has the status subresource
		isClusterScoped = !apiResource.Namespaced

		klog.FromContext(ctx).Info("CRD already exists in virtual cluster, checking for status subresource.", "apiResource", apiResource, "groupVersionKind", groupVersionKind)

		crdName := apiResource.Name
		if apiResource.Group != "" {
			crdName += "." + apiResource.Group
		} else if groupVersionKind.Group != "" {
			crdName += "." + groupVersionKind.Group
		}

		crdDefinition, err := vClient.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, crdName, metav1.GetOptions{})
		if err != nil {
			klog.FromContext(ctx).Error(err, "Error getting CRD in the virtual cluster", "crd", crdName)
			return isClusterScoped, hasStatusSubresource, nil
		}

		for _, version := range crdDefinition.Spec.Versions {
			if version.Name == groupVersionKind.Version {
				if version.Subresources != nil && version.Subresources.Status != nil {
					hasStatusSubresource = true
				}
				break
			}
		}

		return isClusterScoped, hasStatusSubresource, nil
	} else if !kerrors.IsNotFound(err) {
		return isClusterScoped, hasStatusSubresource, fmt.Errorf("check virtual cluster kind: %w", err)
	}

	// get resource from kind name in physical cluster
	groupVersionResource, err := ConvertKindToResource(pConfig, groupVersionKind)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return isClusterScoped, hasStatusSubresource, fmt.Errorf("seems like resource %s is not available in the physical cluster or vcluster has no access to it", groupVersionKind.String())
		}

		return isClusterScoped, hasStatusSubresource, err
	}

	// get crd in physical cluster
	pClient, err := apiextensionsv1clientset.NewForConfig(pConfig)
	if err != nil {
		return isClusterScoped, hasStatusSubresource, err
	}
	crdDefinition, err := pClient.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, groupVersionResource.GroupResource().String(), metav1.GetOptions{})
	if err != nil {
		return isClusterScoped, hasStatusSubresource, errors.Wrap(err, "retrieve crd in host cluster")
	}

	// now create crd in virtual cluster
	crdDefinition.UID = ""
	crdDefinition.ResourceVersion = ""
	crdDefinition.ManagedFields = nil
	crdDefinition.OwnerReferences = nil
	crdDefinition.Status = apiextensionsv1.CustomResourceDefinitionStatus{}
	crdDefinition.Spec.PreserveUnknownFields = false
	crdDefinition.Spec.Conversion = nil

	// make sure we only store the version we care about
	newVersions := []apiextensionsv1.CustomResourceDefinitionVersion{}
	for _, version := range crdDefinition.Spec.Versions {
		if version.Name == groupVersionKind.Version {
			version.Served = true
			version.Storage = true
			newVersions = append(newVersions, version)

			if version.Subresources != nil && version.Subresources.Status != nil {
				hasStatusSubresource = true
			}
			break
		}
	}
	crdDefinition.Spec.Versions = newVersions

	// apply the crd
	klog.FromContext(ctx).Info("Create crd in virtual cluster", "crd", groupVersionKind.String())
	_, err = vClient.ApiextensionsV1().CustomResourceDefinitions().Create(ctx, crdDefinition, metav1.CreateOptions{})
	if err != nil && !kerrors.IsAlreadyExists(err) {
		return isClusterScoped, hasStatusSubresource, errors.Wrap(err, "create crd in virtual cluster")
	}

	// wait for crd to become ready
	klog.FromContext(ctx).Info("Wait for crd to become ready in virtual cluster", "crd", groupVersionKind.String())
	err = wait.ExponentialBackoffWithContext(ctx, wait.Backoff{Duration: time.Second, Factor: 1.5, Cap: time.Minute, Steps: math.MaxInt32}, func(ctx context.Context) (bool, error) {
		crdDefinition, err := vClient.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, groupVersionResource.GroupResource().String(), metav1.GetOptions{})
		if err != nil {
			return false, errors.Wrap(err, "retrieve crd in virtual cluster")
		}
		message := ""
		for _, cond := range crdDefinition.Status.Conditions {
			if cond.Type == apiextensionsv1.Established && cond.Status == apiextensionsv1.ConditionTrue {
				return true, nil
			} else if cond.Type == apiextensionsv1.Established {
				message = cond.String()
			}
		}
		klog.FromContext(ctx).Info("CRD is not ready yet", "crd", groupVersionKind.String(), "message", message)
		return false, nil
	})
	if err != nil {
		return isClusterScoped, hasStatusSubresource, fmt.Errorf("failed to wait for CRD %s to become ready: %w", groupVersionKind.String(), err)
	}

	// check if crd is cluster scoped
	if crdDefinition.Spec.Scope == apiextensionsv1.ClusterScoped {
		isClusterScoped = true
	}

	return isClusterScoped, hasStatusSubresource, nil
}

func ConvertKindToResource(config *rest.Config, groupVersionKind schema.GroupVersionKind) (schema.GroupVersionResource, error) {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}

	resources, err := discoveryClient.ServerResourcesForGroupVersion(groupVersionKind.GroupVersion().String())
	if err != nil {
		return schema.GroupVersionResource{}, err
	}

	for _, r := range resources.APIResources {
		if r.Kind == groupVersionKind.Kind {
			return groupVersionKind.GroupVersion().WithResource(r.Name), nil
		}
	}

	return schema.GroupVersionResource{}, kerrors.NewNotFound(schema.GroupResource{Group: groupVersionKind.Group}, groupVersionKind.Kind)
}

// KindExists returns the api resource for a given CRD.
// If the kind does not exist, it returns an error.
func KindExists(config *rest.Config, groupVersionKind schema.GroupVersionKind) (metav1.APIResource, error) {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return metav1.APIResource{}, err
	}

	resources, err := discoveryClient.ServerResourcesForGroupVersion(groupVersionKind.GroupVersion().String())
	if err != nil {
		return metav1.APIResource{}, err
	}

	for _, r := range resources.APIResources {
		if r.Kind == groupVersionKind.Kind {
			return r, nil
		}
	}

	return metav1.APIResource{}, kerrors.NewNotFound(schema.GroupResource{Group: groupVersionKind.Group}, groupVersionKind.Kind)
}
