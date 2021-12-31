package volumesnapshots

import (
	"fmt"
	"io/ioutil"
	"math"
	"path"
	"strings"
	"time"

	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/volumesnapshots/volumesnapshotclasses"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/volumesnapshots/volumesnapshotcontents"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/volumesnapshots/volumesnapshots"
	"github.com/loft-sh/vcluster/pkg/util/applier"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

var crdPaths = []string{
	"volumesnapshots/snapshot.storage.k8s.io_volumesnapshotclasses.yaml",
	"volumesnapshots/snapshot.storage.k8s.io_volumesnapshotcontents.yaml",
	"volumesnapshots/snapshot.storage.k8s.io_volumesnapshots.yaml",
}

var crdKinds = []string{"VolumeSnapshotClass", "VolumeSnapshotContent", "VolumeSnapshot"}

var ResourceControllers = []func(*context2.ControllerContext, record.EventBroadcaster) error{
	volumesnapshotclasses.Register,
	volumesnapshotcontents.Register,
	volumesnapshots.Register,
}

var ControllerIndicies = []func(*context2.ControllerContext) error{
	volumesnapshotclasses.RegisterIndices,
	volumesnapshotcontents.RegisterIndices,
	volumesnapshots.RegisterIndices,
}

func EnsurePrerequisites(ctx *context2.ControllerContext) error {
	//install CRDs needed for various VolumeSnapshot* resources
	//and wait for them to become available
	config := ctx.VirtualManager.GetConfig()
	restMapper, err := apiutil.NewDynamicRESTMapper(config)
	if err != nil {
		return fmt.Errorf("unable to initialize NewDynamicRESTMapper")
	}
	a := applier.DirectApplier{}

	err = wait.ExponentialBackoff(wait.Backoff{Duration: time.Second, Factor: 1.5, Cap: time.Minute, Steps: math.MaxInt32}, func() (bool, error) {
		var crds strings.Builder
		for _, crd := range crdPaths {
			body, err := ioutil.ReadFile(path.Join(constants.ContainerManifestsFolder, crd))
			if err != nil {
				return false, fmt.Errorf("unable to read %s CRD file for syncing volumesnapshots: %v", crd, err)
			}
			crds.Write(body)
		}

		opts := applier.ApplierOptions{
			Manifest:   crds.String(),
			RESTConfig: config,
			RESTMapper: restMapper,
		}
		err = a.Apply(ctx.Context, opts)
		if err != nil {
			loghelper.Infof("Failed to apply VolumeSnapshotClasses CRD from the manifest file: %v", err)
			return false, nil
		}
		loghelper.Infof("VolumeSnapshotClasses CRD applied successfully")
		return true, nil
	})
	if err != nil {
		return err
	}

	var lastErr error
	err = wait.ExponentialBackoff(wait.Backoff{Duration: time.Second, Factor: 1.5, Cap: time.Minute, Steps: math.MaxInt32}, func() (bool, error) {
		var exists bool
		exists, lastErr = VolumeSnapshotCRDsExist(config)
		return exists, nil
	})

	if err != nil {
		return fmt.Errorf("failed to find VolumeSnapshot* CRDS: %v: %v", err, lastErr)
	}
	return nil
}

func RegisterIndices(ctx *context2.ControllerContext) error {
	for _, v := range ControllerIndicies {
		err := v(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func Register(ctx *context2.ControllerContext, eventBroadcaster record.EventBroadcaster) error {
	for _, v := range ResourceControllers {
		err := v(ctx, eventBroadcaster)
		if err != nil {
			return err
		}
	}
	return nil
}

func VolumeSnapshotCRDsExist(config *rest.Config) (bool, error) {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return false, err
	}

	resources, err := discoveryClient.ServerResourcesForGroupVersion("snapshot.storage.k8s.io/v1")
	if err != nil {
		if kerrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	for _, kind := range crdKinds {
		found := false
		for _, r := range resources.APIResources {
			if r.Kind == kind {
				found = true
				break
			}
		}
		if !found {
			return false, fmt.Errorf("%s group doesn't contain %s", resources.GroupVersion, kind)
		}
	}

	return true, nil
}
