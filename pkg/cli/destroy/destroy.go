package destroy

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/survey"
	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/cli/start"
	"github.com/loft-sh/vcluster/pkg/platform/clihelper"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1clientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
)

// define the order of resource deletion
var resourceOrder = []string{
	// instances
	"virtualclusterinstances",
	"virtualclusters",
	"devpodworkspaceinstances",
	"spaceinstances",

	// templates
	"virtualclustertemplates",
	"devpodenvironmenttemplates",
	"devpodworkspacepresets",
	"devpodworkspacetemplates",
	"clusterroletemplates",
	"spacetemplates",
	"apps",
	"spaceconstraints",

	// infra
	"tasks",
	"clusterquotas",
	"projects",
	"runners",
	"clusters",
	"clusteraccesses",
	"networkpeers",

	// access
	"teams",
	"users",
	"sharedsecrets",
	"accesskeys",
	"localclusteraccesses",
	"localteams",
	"localusers",
}

// things listed here should also be included in resourceOrder
var legacyResources = []string{
	"virtualclusters",
	"spaceconstraints",
	"localclusteraccesses",
	"localteams",
	"localusers",
}

// DeleteOptions holds cli options for the delete command
type DeleteOptions struct {
	start.Options
	// cli options
	DeleteNamespace       bool
	IgnoreNotFound        bool
	Force                 bool
	ForceRemoveFinalizers bool
	NonInteractive        bool
	TimeoutMinutes        int
}

var backoffFactor = 1.2

func Destroy(ctxWithoutTimeout context.Context, opts DeleteOptions) error {
	err := destroy(ctxWithoutTimeout, opts)
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("timed out: %w", err)
	}
	return err
}

func destroy(ctxWithoutTimeout context.Context, opts DeleteOptions) error {
	ctx, cancel := context.WithTimeout(ctxWithoutTimeout, time.Duration(opts.TimeoutMinutes)*time.Minute)
	defer cancel()

	// create time.Duration(opts.TimeoutMinutes) * time.Minutea dynamic client
	dynamicClient, err := dynamic.NewForConfig(opts.RestConfig)
	if err != nil {
		return err
	}

	// create a discovery client
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(opts.RestConfig)
	if err != nil {
		return err
	}

	apiextensionclientset, err := apiextensionsv1clientset.NewForConfig(opts.RestConfig)
	if err != nil {
		return err
	}

	// to compare resources advertised by server vs ones explicitly handled by us
	clusterResourceSet := sets.New[string]()
	handledResourceSet := sets.New(resourceOrder...)
	legacyHandledResourceSet := sets.New(legacyResources...)

	// get all custom resource definitions in storage.loft.sh
	resourceList, err := discoveryClient.ServerResourcesForGroupVersion("storage.loft.sh/v1")
	if err != nil && !kerrors.IsNotFound(err) {
		return fmt.Errorf("failed to discover resources in storage.loft.sh/v1 group: %w", err)
	}

	// resource list may be nil if all resources in the group are already deleted
	if resourceList == nil {
		resourceList = &metav1.APIResourceList{APIResources: []metav1.APIResource{}}
	}
	// populate the set
	for _, resource := range resourceList.APIResources {
		// don't insert subresources
		if strings.Contains(resource.Name, "/") {
			continue
		}
		clusterResourceSet.Insert(resource.Name)
	}

	unhandledResourceSet := clusterResourceSet.Difference(handledResourceSet)
	if unhandledResourceSet.Len() != 0 {
		opts.Log.Errorf("some storage.loft.sh resources are unhandled: %v. Try a newer cli version", unhandledResourceSet.UnsortedList())
		return err
	}

	for _, resourceName := range resourceOrder {
		if !clusterResourceSet.Has(resourceName) {
			// only debug output if legacy resource
			if legacyHandledResourceSet.Has(resourceName) {
				opts.Log.Debugf("legacy resource %q not found in discovery, skipping", resourceName)
			} else {
				opts.Log.Infof("resource %q not found in discovery, skipping", resourceName)
			}
			continue
		}
		// list and delete all resources. If times out because of resources, the timeout will be repeated and new context will be created
		ctx, cancel, err = deleteAllResourcesAndWait(ctxWithoutTimeout, ctx, dynamicClient, opts.Log, opts.NonInteractive, opts.ForceRemoveFinalizers, opts.TimeoutMinutes, "storage.loft.sh", "v1", resourceName)
		defer cancel()
		if err != nil {
			return fmt.Errorf("failed to delete resource %q: %w", resourceName, err)
		}
		defer cancel()
	}

	// helm uninstall and others
	err = clihelper.UninstallLoft(ctx, opts.KubeClient, opts.RestConfig, opts.Context, opts.Namespace, opts.Log)
	if err != nil {
		return err
	}

	opts.Log.Info("deleting CRDS")
	err = wait.ExponentialBackoffWithContext(ctx, wait.Backoff{Duration: time.Second, Factor: backoffFactor, Cap: time.Duration(opts.TimeoutMinutes) * time.Minute, Steps: math.MaxInt32}, func(ctx context.Context) (bool, error) {
		list, err := apiextensionclientset.ApiextensionsV1().CustomResourceDefinitions().List(ctx, metav1.ListOptions{})
		if err != nil {
			return false, err
		}

		crdList := []apiextensionsv1.CustomResourceDefinition{}

		for _, object := range list.Items {
			if strings.HasSuffix(object.Name, ".storage.loft.sh") {
				crdList = append(crdList, object)
			}
		}
		if len(crdList) == 0 {
			return true, nil
		}
		for _, object := range crdList {
			crdSuffix := ".storage.loft.sh"
			if !strings.HasSuffix(object.Name, crdSuffix) {
				continue
			}
			opts.Log.Debugf("checking CRD %q", object.Name)
			expectedResourceName := strings.TrimSuffix(object.Name, crdSuffix)
			if !handledResourceSet.Has(expectedResourceName) {
				opts.Log.Errorf("unhandled CRD: %q", object.Name)
				continue
			}
			// retry later if object already deleted
			if !object.GetDeletionTimestamp().IsZero() {
				opts.Log.Infof("deleted CRD still found: %q", object.GetName())
				continue
			}
			opts.Log.Infof("deleting customresourcedefinition %v", object.GetName())
			err := apiextensionclientset.ApiextensionsV1().CustomResourceDefinitions().Delete(ctx, object.Name, metav1.DeleteOptions{})
			if err != nil {
				return false, err
			}
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("failed to delete CRDs: %w", err)
	}

	if opts.DeleteNamespace {
		opts.Log.Infof("deleting namespace %q", opts.Namespace)
		err = wait.ExponentialBackoffWithContext(ctx, wait.Backoff{Duration: time.Second, Factor: backoffFactor, Cap: time.Duration(opts.TimeoutMinutes) * time.Minute, Steps: math.MaxInt32}, func(ctx context.Context) (bool, error) {
			ns, err := opts.KubeClient.CoreV1().Namespaces().Get(ctx, opts.Namespace, metav1.GetOptions{})
			if kerrors.IsNotFound(err) {
				return true, nil
			} else if err != nil {
				return false, err
			}

			if ns.GetDeletionTimestamp().IsZero() {
				err = opts.KubeClient.CoreV1().Namespaces().Delete(ctx, opts.Namespace, metav1.DeleteOptions{})
				if err != nil {
					return false, err
				}
			}
			return false, nil
		})
		if err != nil {
			return err
		}
	}

	for _, name := range clihelper.DefaultClusterRoles {
		name := name + "-binding"
		opts.Log.Infof("deleting clusterrolebinding %q", name)
		err := opts.KubeClient.RbacV1().ClusterRoleBindings().Delete(ctx, name, metav1.DeleteOptions{})
		if err != nil && !kerrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete clusterrole: %w", err)
		}
	}
	for _, name := range clihelper.DefaultClusterRoles {
		opts.Log.Infof("deleting clusterrole %q", name)
		err := opts.KubeClient.RbacV1().ClusterRoles().Delete(ctx, name, metav1.DeleteOptions{})
		if err != nil && !kerrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete clusterrole: %w", err)
		}
	}

	return nil
}

func deleteAllResourcesAndWait(ctxWithoutDeadline, ctxWithDeadLine context.Context, dynamicClient dynamic.Interface, log log.Logger, nonInteractive bool, deleteFinalizers bool, timeoutMinutes int, group, version, resource string) (context.Context, context.CancelFunc, error) {
	gvr := schema.GroupVersionResource{Group: group, Version: version, Resource: resource}

	//  function to poll with wait.ExponentialBackoffWithContext
	deleteAndWait := func(deleteFinalizers bool) func(ctx context.Context) (bool, error) {
		// log each key as waiting only once on the info level, and continue logging on the debug level
		loggedDeletion := sets.New[string]()
		infofOnceThenDebugf := func(str string, args ...interface{}) {
			logLine := fmt.Sprintf(str, args...)
			if loggedDeletion.Has(logLine) {
				log.Debug(logLine)
				return
			}
			log.Info(logLine)
			loggedDeletion.Insert(logLine)
		}

		return func(ctx context.Context) (bool, error) {
			infofOnceThenDebugf("checking all %q", resource)

			// fetch all
			resourceClient := dynamicClient.Resource(gvr)
			list, err := resourceClient.List(ctx, metav1.ListOptions{})
			if err != nil {
				return false, err
			}
			// succeed when all resources are deleted
			if len(list.Items) == 0 {
				return true, nil
			}

			isVCluster := resource == "virtualclusterinstances"

			// delete all resources and log deleting resources
			for _, object := range list.Items {
				// get namespaced name
				namespacedName := object.GetName()
				namespace := object.GetNamespace()
				if namespace != "" {
					namespacedName += "/" + namespace
				}
				isExternalVCluster := false
				virtualClusterInstance := &storagev1.VirtualClusterInstance{}
				if isVCluster {
					//convert unstructured to VirtualClusterInstance
					err = runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &virtualClusterInstance)
					if err != nil {
						log.Warnf("couldn't cast %q object %q to VirtualClusterInstance: %v", resource, namespacedName, err)
					}
					isExternalVCluster = virtualClusterInstance.Spec.External
				}
				if isExternalVCluster && virtualClusterInstance.Status.VirtualCluster != nil {
					vConfig := &config.Config{}
					err = config.UnmarshalYAMLStrict([]byte(virtualClusterInstance.Status.VirtualCluster.HelmRelease.Values), vConfig)
					if err != nil {
						return false, fmt.Errorf("failed to unmarshal virtual cluster config for %v %q: %w", resource, namespacedName, err)
					}
					if vConfig.ControlPlane.BackingStore.Database.External.Connector != "" {
						log.Warnf("IMPORTANT! You are removing an externally deployed virtual cluster %q from the platform.\n It will not be destroyed as the deployment is managed externally, but its database will be removed rendering it inoperable.", namespacedName)
						if !nonInteractive {
							yesOpt := "yes"
							noOpt := "no"
							out, err := log.Question(&survey.QuestionOptions{
								Options:  []string{yesOpt, noOpt},
								Question: "Do you want to continue?",
							})
							if err != nil {
								return false, fmt.Errorf("failed to prompt for confirmation: %w", err)
							}
							if out != yesOpt {
								return false, fmt.Errorf("destroy cancelled during prompt")
							}
						}
					} else {
						log.Warnf("removing an externally deployed virtual cluster %q from the platform. It will not be destroyed as the deployment is managed externally, but its connection to its database will be removed.", namespacedName)
					}
				}

				// delete object if not already deleted
				if object.GetDeletionTimestamp().IsZero() {
					if !isExternalVCluster {
						log.Infof("deleting %v: %q", resource, namespacedName)
					} else {
						log.Infof("deleting externally deployed %v, the virtual cluster itself will remain: %q", resource, namespacedName)
					}
					err := resourceClient.Namespace(object.GetNamespace()).Delete(ctx, object.GetName(), metav1.DeleteOptions{})
					if kerrors.IsNotFound(err) {
						continue
					} else if err != nil {
						return false, err
					}
				} else {
					infofOnceThenDebugf("deleted resource found, waiting for cleanup: %v", object.GetName())
				}
				// object exists and delete command succeeded
				if deleteFinalizers {
					log.Infof("removing finalizers from %v: %q", resource, namespacedName)
					_, err = resourceClient.Namespace(object.GetNamespace()).Patch(ctx, object.GetName(), types.MergePatchType, []byte(`{"metadata":{"finalizers":[]}}`), metav1.PatchOptions{})
					if err != nil && !kerrors.IsNotFound(err) {
						return false, err
					}
				}
			}
			return false, nil
		}
	}
	err := wait.ExponentialBackoffWithContext(ctxWithDeadLine, wait.Backoff{Duration: time.Second, Factor: backoffFactor, Cap: time.Duration(timeoutMinutes) * time.Minute, Steps: math.MaxInt32}, deleteAndWait(false))
	if !errors.Is(err, context.DeadlineExceeded) {
		// return the err unless timed out. If timed out, remove finalizers and retry
		return ctxWithDeadLine, func() {}, err
	}

	// the timeout is hit, begin removing finalizers and rety
	if !deleteFinalizers {
		return ctxWithDeadLine, func() {}, fmt.Errorf("timed out waiting for %q to be deleted", resource)
	}
	// new context now that the old deadline is exceeded
	ctx, cancel := context.WithTimeout(ctxWithoutDeadline, time.Duration(timeoutMinutes)*time.Minute)
	log.Warn("operation timed out. Removing finalizers from stuck resources, resetting timeout")
	err = wait.ExponentialBackoffWithContext(ctxWithoutDeadline, wait.Backoff{Duration: time.Second, Factor: backoffFactor, Cap: time.Duration(timeoutMinutes) * time.Minute, Steps: math.MaxInt32}, deleteAndWait(true))
	if errors.Is(err, context.DeadlineExceeded) {
		return ctx, cancel, fmt.Errorf("timed out waiting for %q to be deleted", resource)
	}
	return ctx, cancel, err
}
