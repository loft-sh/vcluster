package util

import (
	"context"
	"fmt"
	"math"
	"time"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/loft-sh/vcluster/pkg/util/applier"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

func EnsureCRD(ctx context.Context, config *rest.Config, manifest []byte, groupVersionKind schema.GroupVersionKind) error {
	exists, err := KindExists(config, groupVersionKind)
	if err != nil {
		return err
	} else if exists {
		return nil
	}

	err = wait.ExponentialBackoffWithContext(ctx, wait.Backoff{Duration: time.Second, Factor: 1.5, Cap: 5 * time.Minute, Steps: math.MaxInt32}, func(ctx context.Context) (bool, error) {
		err := applier.ApplyManifest(ctx, config, manifest)
		if err != nil {
			loghelper.Infof("Failed to apply CRD %s: %v", groupVersionKind.String(), err)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("failed to apply CRD %s: %w", groupVersionKind.String(), err)
	}

	var lastErr error
	err = wait.ExponentialBackoffWithContext(ctx, wait.Backoff{Duration: time.Second, Factor: 1.5, Cap: time.Minute, Steps: math.MaxInt32}, func(_ context.Context) (bool, error) {
		var found bool
		found, lastErr = KindExists(config, groupVersionKind)
		return found, nil
	})
	if err != nil {
		return fmt.Errorf("failed to find CRD %s: %w: %w", groupVersionKind.String(), err, lastErr)
	}

	return nil
}

// KindExists checks if given CRDs exist in the given group.
// Returns foundKinds, notFoundKinds, error
func KindExists(config *rest.Config, groupVersionKind schema.GroupVersionKind) (bool, error) {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return false, err
	}

	resources, err := discoveryClient.ServerResourcesForGroupVersion(groupVersionKind.GroupVersion().String())
	if err != nil {
		if kerrors.IsNotFound(err) {
			return false, nil
		}

		return false, err
	}

	for _, r := range resources.APIResources {
		if r.Kind == groupVersionKind.Kind {
			return true, nil
		}
	}

	return false, nil
}
