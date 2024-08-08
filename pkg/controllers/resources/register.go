package resources

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/controllers/resources/configmaps"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/csidrivers"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/csinodes"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/csistoragecapacities"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/endpoints"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/events"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/ingressclasses"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/ingresses"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/namespaces"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/networkpolicies"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/persistentvolumeclaims"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/persistentvolumes"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/poddisruptionbudgets"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/pods"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/priorityclasses"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/runtimeclasses"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/secrets"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/serviceaccounts"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/services"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/storageclasses"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/volumesnapshots/volumesnapshotclasses"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/volumesnapshots/volumesnapshotcontents"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/volumesnapshots/volumesnapshots"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/pkg/errors"
)

// ExtraControllers that will be started as well
var ExtraControllers []BuildController

// BuildController is a function to build a new syncer
type BuildController func(ctx *synccontext.RegisterContext) (syncertypes.Object, error)

// getSyncers retrieves all syncers that should get created
func getSyncers(ctx *synccontext.RegisterContext) []BuildController {
	return append([]BuildController{
		isEnabled(ctx.Config.Sync.ToHost.Services.Enabled, services.New),
		isEnabled(ctx.Config.Sync.ToHost.ConfigMaps.Enabled, configmaps.New),
		isEnabled(ctx.Config.Sync.ToHost.Secrets.Enabled, secrets.New),
		isEnabled(ctx.Config.Sync.ToHost.Endpoints.Enabled, endpoints.New),
		isEnabled(ctx.Config.Sync.ToHost.Pods.Enabled, pods.New),
		isEnabled(ctx.Config.Sync.FromHost.Events.Enabled, events.New),
		isEnabled(ctx.Config.Sync.ToHost.PersistentVolumeClaims.Enabled, persistentvolumeclaims.New),
		isEnabled(ctx.Config.Sync.ToHost.Ingresses.Enabled, ingresses.New),
		isEnabled(ctx.Config.Sync.FromHost.IngressClasses.Enabled, ingressclasses.New),
		isEnabled(ctx.Config.Sync.FromHost.RuntimeClasses.Enabled, runtimeclasses.New),
		isEnabled(ctx.Config.Sync.ToHost.StorageClasses.Enabled, storageclasses.New),
		isEnabled(ctx.Config.Sync.FromHost.StorageClasses.Enabled == "true", storageclasses.NewHostStorageClassSyncer),
		isEnabled(ctx.Config.Sync.ToHost.PriorityClasses.Enabled || ctx.Config.Sync.FromHost.PriorityClasses.Enabled, priorityclasses.New),
		isEnabled(ctx.Config.Sync.ToHost.PodDisruptionBudgets.Enabled, poddisruptionbudgets.New),
		isEnabled(ctx.Config.Sync.ToHost.NetworkPolicies.Enabled, networkpolicies.New),
		isEnabled(ctx.Config.Sync.ToHost.VolumeSnapshots.Enabled, volumesnapshotclasses.New),
		isEnabled(ctx.Config.Sync.ToHost.VolumeSnapshots.Enabled, volumesnapshots.New),
		isEnabled(ctx.Config.Sync.ToHost.VolumeSnapshots.Enabled, volumesnapshotcontents.New),
		isEnabled(ctx.Config.Sync.ToHost.ServiceAccounts.Enabled, serviceaccounts.New),
		isEnabled(ctx.Config.Sync.FromHost.CSINodes.Enabled == "true", csinodes.New),
		isEnabled(ctx.Config.Sync.FromHost.CSIDrivers.Enabled == "true", csidrivers.New),
		isEnabled(ctx.Config.Sync.FromHost.CSIStorageCapacities.Enabled == "true", csistoragecapacities.New),
		isEnabled(ctx.Config.Experimental.MultiNamespaceMode.Enabled, namespaces.New),
		persistentvolumes.New,
		nodes.New,
	}, ExtraControllers...)
}

// BuildSyncers builds the syncers
func BuildSyncers(ctx *synccontext.RegisterContext) ([]syncertypes.Object, error) {
	// register controllers for resource synchronization
	syncers := []syncertypes.Object{}
	for _, newSyncer := range getSyncers(ctx) {
		if newSyncer == nil {
			continue
		}

		syncer, err := newSyncer(ctx)

		name := ""
		if syncer != nil {
			name = syncer.Name()
		}

		if err != nil {
			return nil, fmt.Errorf("register %s controller: %w", name, err)
		}

		loghelper.Infof("Created %s syncer", name)

		// execute register indices
		indexRegisterer, ok := syncer.(syncertypes.IndicesRegisterer)
		if ok {
			err := indexRegisterer.RegisterIndices(ctx)
			if err != nil {
				return nil, errors.Wrapf(err, "register indices for %s syncer", name)
			}
		}

		syncers = append(syncers, syncer)
	}

	return syncers, nil
}

func isEnabled[T any](enabled bool, fn T) T {
	if enabled {
		return fn
	}
	var ret T
	return ret
}
