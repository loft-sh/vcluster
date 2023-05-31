package context

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/discovery"
	"k8s.io/klog/v2"
)

var ExistingControllers = sets.New(
	"services",
	"configmaps",
	"secrets",
	"endpoints",
	"pods",
	"events",
	"fake-nodes",
	"fake-persistentvolumes",
	"persistentvolumeclaims",
	"ingresses",
	"ingressclasses",
	"nodes",
	"persistentvolumes",
	"storageclasses",
	"hoststorageclasses",
	"priorityclasses",
	"networkpolicies",
	"volumesnapshots",
	"poddisruptionbudgets",
	"serviceaccounts",
	"csinodes",
	"csidrivers",
	"csistoragecapacities",
	"namespaces",
)

var DefaultEnabledControllers = sets.New(
	// helm charts need to be updated when changing this!
	// values.yaml and template/_helpers.tpl reference these
	"services",
	"configmaps",
	"secrets",
	"endpoints",
	"pods",
	"events",
	"persistentvolumeclaims",
	"fake-nodes",
	"fake-persistentvolumes",
)

var schedulerRequiredControllers = sets.New(
	"csinodes",
	"csidrivers",
	"csistoragecapacities",
)

const (
	storageV1GroupVersion = "storage.k8s.io/v1"
)

// map from groupversion to list of resources in that groupversion
// the syncers will be disabled unless that resource is advertised in that groupversion
var possibleMissing = map[string][]string{
	storageV1GroupVersion: schedulerRequiredControllers.UnsortedList(),
}

func parseControllers(options *VirtualClusterOptions) (sets.Set[string], error) {
	enabledControllers := DefaultEnabledControllers.Clone()
	disabledControllers := sets.New[string]()

	// migrate deprecated flags
	if len(options.DeprecatedDisableSyncResources) > 0 {
		disabledControllers.Insert(strings.Split(options.DeprecatedDisableSyncResources, ",")...)
	}
	if options.DeprecatedEnablePriorityClasses {
		enabledControllers.Insert("priorityclasses")
	}
	if !options.DeprecatedUseFakePersistentVolumes {
		enabledControllers.Insert("persistentvolumes")
	}
	if !options.DeprecatedUseFakeNodes {
		enabledControllers.Insert("nodes")
	}
	if options.DeprecatedEnableStorageClasses {
		enabledControllers.Insert("storageclasses")
	}

	for _, c := range options.Controllers {
		controller := strings.TrimSpace(c)
		if len(controller) == 0 {
			return nil, fmt.Errorf("unrecognized controller %s, available controllers: %s", c, availableControllers())
		}

		if controller[0] == '-' {
			controller = controller[1:]
			disabledControllers.Insert(controller)
		} else {
			enabledControllers.Insert(controller)
		}

		if !ExistingControllers.Has(controller) {
			return nil, fmt.Errorf("unrecognized controller %s, available controllers: %s", controller, availableControllers())
		}
	}
	// enable ingressclasses if ingress syncing is enabled and incressclasses not explicitly disabled
	if enabledControllers.Has("ingresses") && !disabledControllers.Has("ingressesclasses") {
		enabledControllers.Insert("ingressclasses")
	}

	// enable namespaces controller in MultiNamespaceMode
	if options.MultiNamespaceMode {
		enabledControllers.Insert("namespaces")
	}

	// do validations on dynamically added controllers here (to take into acount disabledControllers):

	// enable additional controllers required for scheduling with storage
	if options.EnableScheduler && enabledControllers.Has("persistentvolumeclaims") {
		klog.Infof("persistentvolumeclaim syncing and scheduler enabled, enabling required controllers: %q", schedulerRequiredControllers)
		enabledControllers = enabledControllers.Union(schedulerRequiredControllers)
		requiredButDisabled := disabledControllers.Intersection(schedulerRequiredControllers)
		if requiredButDisabled.Len() > 0 {
			klog.Warningf("persistentvolumeclaim syncing and scheduler enabled, but required syncers explicitly disabled: %q. This may result in incorrect pod scheduling.", sets.List(requiredButDisabled))
		}
		if !enabledControllers.Has("storageclasses") {
			klog.Info("persistentvolumeclaim syncing and scheduler enabled, but storageclass sync not enabled. Syncing host storageclasses to vcluster(hoststorageclasses)")
			enabledControllers.Insert("hoststorageclasses")
			if disabledControllers.HasAll("storageclasses", "hoststorageclasses") {
				return nil, fmt.Errorf("persistentvolumeclaim syncing and scheduler enabled, but both storageclasses and hoststorageclasses syncers disabled")
			}
		}
	}

	// remove explicitly disabled controllers
	enabledControllers = enabledControllers.Difference(disabledControllers)

	// do validations on user configured controllers here (on just enabledControllers):

	// check if nodes controller needs to be enabled
	if (options.SyncAllNodes || options.EnableScheduler) && !enabledControllers.Has("nodes") {
		return nil, fmt.Errorf("node sync needs to be enabled when using --sync-all-nodes OR --enable-scheduler flags")
	}

	// check if storage classes and host storage classes are enabled at the same time
	if enabledControllers.HasAll("storageclasses", "hoststorageclasses") {
		return nil, fmt.Errorf("you cannot sync storageclasses and hoststorageclasses at the same time. Choose only one of them")
	}

	return enabledControllers, nil
}

func availableControllers() string {
	return strings.Join(sets.List(ExistingControllers), ", ")
}

// disableMissingAPIs checks if the  apis are enabled, if any are missing, disable the syncer and print a log
func disableMissingAPIs(discoveryClient discovery.DiscoveryInterface, controllers sets.Set[string]) (sets.Set[string], error) {
	enabledControllers := controllers.Clone()
	for groupVersion, resourceList := range possibleMissing {
		resources, err := discoveryClient.ServerResourcesForGroupVersion(groupVersion)
		if err != nil && !errors.IsNotFound(err) {
			return nil, err
		}
		for _, resourcePlural := range resourceList {
			found := false
			// search the resourses for a match
			if resources != nil {
				for _, r := range resources.APIResources {
					if r.Name == resourcePlural {
						found = true
						break
					}
				}
			}
			if !found {
				enabledControllers.Delete(resourcePlural)
				klog.Warningf("host kubernetes apiserver not advertising resource %q in GroupVersion %q, disabling the syncer", resourcePlural, storageV1GroupVersion)
			}
		}
	}
	return enabledControllers, nil
}
