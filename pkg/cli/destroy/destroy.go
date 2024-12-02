package destroy

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/start"
	"github.com/loft-sh/vcluster/pkg/platform/clihelper"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1clientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	DeleteNamespace bool
	IgnoreNotFound  bool
	Force           bool
	NonInteractive  bool
	TimeoutMinutes  int
}

var backoffFactor = 1.2

func Destroy(ctx context.Context, opts DeleteOptions) error {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(opts.TimeoutMinutes)*time.Minute)
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
		// list and delete all resources
		err = deleteAllResourcesAndWait(ctx, dynamicClient, opts.Log, opts.TimeoutMinutes, "storage.loft.sh", "v1", resourceName)
		if err != nil {
			return fmt.Errorf("failed to delete resource %q: %w", resourceName, err)
		}
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

func deleteAllResourcesAndWait(ctx context.Context, dynamicClient dynamic.Interface, log log.Logger, timeoutMinutes int, group, version, resource string) error {
	gvr := schema.GroupVersionResource{Group: group, Version: version, Resource: resource}
	err := wait.ExponentialBackoffWithContext(ctx, wait.Backoff{Duration: time.Second, Factor: backoffFactor, Cap: time.Duration(timeoutMinutes) * time.Minute, Steps: math.MaxInt32}, func(ctx context.Context) (bool, error) {
		log.Debugf("checking all %q", resource)

		resourceClient := dynamicClient.Resource(gvr)
		list, err := resourceClient.List(ctx, metav1.ListOptions{})
		if err != nil {
			return false, err
		}
		if len(list.Items) == 0 {
			return true, nil
		}
		for _, object := range list.Items {
			if !object.GetDeletionTimestamp().IsZero() {
				return false, nil
			}
			if object.GetNamespace() == "" {
				log.Infof("deleting %v: %v", resource, object.GetName())
			} else {
				log.Infof("deleting %v: %v/%v", resource, object.GetNamespace(), object.GetName())
			}
			err := resourceClient.Namespace(object.GetNamespace()).Delete(ctx, object.GetName(), metav1.DeleteOptions{})
			if err != nil && !kerrors.IsNotFound(err) {
				return false, err
			}
		}
		return false, nil
	})
	if err != nil {
		return err
	}

	return nil
}
