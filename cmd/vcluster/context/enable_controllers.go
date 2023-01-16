package context

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"
)

var ExistingControllers = sets.NewString(
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

var DefaultEnabledControllers = sets.NewString(
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

var schedulerRequiredControllers = sets.NewString(
	"csinodes",
	"csidrivers",
	"csistoragecapacities",
)

func parseControllers(options *VirtualClusterOptions) (sets.String, error) {
	enabledControllers := DefaultEnabledControllers.Clone()
	disabledControllers := sets.NewString()

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
			klog.Warningf("pesistentvolumeclaim syncing and scheduler enabled, but required syncers explicitly disabled: %q. This may result in incorrect pod scheduling.", requiredButDisabled.List())
		}
		if !enabledControllers.Has("storageclasses") {
			klog.Info("persistentvolumeclaim syncing and scheduler enabled, but storageclass sync not enabled. Syncing host storageclasses to vcluster(hoststorageclasses)")
			enabledControllers.Insert("hoststorageclasses")
			if disabledControllers.HasAll("storageclasses", "hoststorageclasses") {
				return nil, fmt.Errorf("pesistentvolumeclaim syncing and scheduler enabled, but both storageclasses and hoststorageclasses syncers disabled")
			}
		}
	}

	// remove explicitly disabled controllers
	enabledControllers = enabledControllers.Difference(disabledControllers)

	// do validations on user configured controllers here (on just enabledControllers):

	// check if nodes controller needs to be enabled
	if (options.SyncAllNodes || options.EnableScheduler) && !enabledControllers.Has("nodes") {
		return nil, fmt.Errorf("you cannot use --sync-all-nodes and --enable-scheduler without enabling nodes sync")
	}

	// check if storage classes and host storage classes are enabled at the same time
	if enabledControllers.HasAll("storageclasses", "hoststorageclasses") {
		return nil, fmt.Errorf("you cannot sync storageclasses and hoststorageclasses at the same time. Choose only one of them")
	}

	return enabledControllers, nil
}

func availableControllers() string {
	return strings.Join(ExistingControllers.List(), ", ")
}
