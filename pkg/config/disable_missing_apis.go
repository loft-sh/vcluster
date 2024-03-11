package config

import (
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/klog/v2"
)

// DisableMissingAPIs checks if the  apis are enabled, if any are missing, disable the syncer and print a log
func (v VirtualClusterConfig) DisableMissingAPIs(discoveryClient discovery.DiscoveryInterface) error {
	resources, err := discoveryClient.ServerResourcesForGroupVersion("storage.k8s.io/v1")
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	}

	// check if found
	if v.Sync.FromHost.CSINodes.Enabled && !findResource(resources, "csinodes") {
		v.Sync.FromHost.CSINodes.Enabled = false
		klog.Warningf("host kubernetes apiserver not advertising resource csinodes in GroupVersion storage.k8s.io/v1, disabling the syncer")
	}

	// check if found
	if v.Sync.FromHost.CSIDrivers.Enabled && !findResource(resources, "csidrivers") {
		v.Sync.FromHost.CSIDrivers.Enabled = false
		klog.Warningf("host kubernetes apiserver not advertising resource csidrivers in GroupVersion storage.k8s.io/v1, disabling the syncer")
	}

	// check if found
	if v.Sync.FromHost.CSIStorageCapacities.Enabled && !findResource(resources, "csistoragecapacities") {
		v.Sync.FromHost.CSIStorageCapacities.Enabled = false
		klog.Warningf("host kubernetes apiserver not advertising resource csistoragecapacities in GroupVersion storage.k8s.io/v1, disabling the syncer")

	}

	return nil
}

func findResource(resources *metav1.APIResourceList, resourcePlural string) bool {
	if resources != nil {
		for _, r := range resources.APIResources {
			if r.Name == resourcePlural {
				return true
			}
		}
	}

	return false
}
