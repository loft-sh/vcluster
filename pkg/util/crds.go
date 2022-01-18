package util

import (
	"context"
	"fmt"
	"math"
	"path"
	"time"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/util/applier"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

func EnsureCRD(ctx context.Context, config *rest.Config, crdFilePath, groupVersion, kind string) error {
	// TODO: remove retries once this is implemented - https://github.com/loft-sh/vcluster/issues/276
	err := wait.ExponentialBackoffWithContext(ctx, wait.Backoff{Duration: time.Second, Factor: 1.5, Cap: 5 * time.Minute, Steps: math.MaxInt32}, func() (bool, error) {
		err := applier.ApplyManifestFile(config, path.Join(constants.ContainerManifestsFolder, crdFilePath))
		if err != nil {
			loghelper.Infof("Failed to apply VolumeSnapshotClasses CRD from the manifest file: %v", err)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("failed to apply VolumeSnapshotClasses CRD: %v", err)
	}

	var lastErr error
	err = wait.ExponentialBackoffWithContext(ctx, wait.Backoff{Duration: time.Second, Factor: 1.5, Cap: time.Minute, Steps: math.MaxInt32}, func() (bool, error) {
		var missingKinds *[]string
		_, missingKinds, lastErr = CRDsExist(config, kind)
		return len(*missingKinds) == 0, nil
	})

	if err != nil {
		return fmt.Errorf("failed to find VolumeSnapshot* CRDS: %v: %v", err, lastErr)
	}
	return nil
}

// CRDsExist checks if given CRDs exist in the given group.
// Returns foundKinds, notFoundKinds, error
func CRDsExist(config *rest.Config, groupVersion string, kinds ...string) (*[]string, *[]string, error) {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, nil, err
	}

	resources, err := discoveryClient.ServerResourcesForGroupVersion(groupVersion)
	if err != nil {
		return nil, nil, err
	}

	foundKinds := []string{}
	notFoundKinds := []string{}
	for _, kind := range kinds {
		found := false
		for _, r := range resources.APIResources {
			if r.Kind == kind {
				found = true
				break
			}
		}
		if found {
			foundKinds = append(foundKinds, kind)
		} else {
			notFoundKinds = append(notFoundKinds, kind)
		}
	}
	return &foundKinds, &notFoundKinds, nil
}
