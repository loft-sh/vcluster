package controllers

import (
	"fmt"
	"net/http"
	"strings"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/controllers/deploy"
	"github.com/loft-sh/vcluster/pkg/controllers/generic"
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
	"github.com/loft-sh/vcluster/pkg/controllers/resources/secrets"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/serviceaccounts"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/storageclasses"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/volumesnapshots/volumesnapshotclasses"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/volumesnapshots/volumesnapshotcontents"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/volumesnapshots/volumesnapshots"
	"github.com/loft-sh/vcluster/pkg/controllers/servicesync"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/util/blockingcacheclient"
	util "github.com/loft-sh/vcluster/pkg/util/context"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/loft-sh/vcluster/pkg/controllers/coredns"
	"github.com/loft-sh/vcluster/pkg/controllers/k8sdefaultendpoint"
	"github.com/loft-sh/vcluster/pkg/controllers/podsecurity"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/services"
	syncertypes "github.com/loft-sh/vcluster/pkg/types"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// ExtraControllers that will be started as well
var ExtraControllers []BuildController

// BuildController is a function to build a new syncer
type BuildController func(ctx *synccontext.RegisterContext) (syncertypes.Object, error)

func getSyncers(ctx *config.ControllerContext) []BuildController {
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
		isEnabled(ctx.Config.Sync.ToHost.StorageClasses.Enabled, storageclasses.New),
		isEnabled(ctx.Config.Sync.FromHost.StorageClasses.Enabled == "true", storageclasses.NewHostStorageClassSyncer),
		isEnabled(ctx.Config.Sync.ToHost.PriorityClasses.Enabled, priorityclasses.New),
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

func isEnabled(enabled bool, fn BuildController) BuildController {
	if enabled {
		return fn
	}
	return nil
}

func Create(ctx *config.ControllerContext) ([]syncertypes.Object, error) {
	registerContext := util.ToRegisterContext(ctx)

	// register controllers for resource synchronization
	syncers := []syncertypes.Object{}
	for _, newSyncer := range getSyncers(ctx) {
		if newSyncer == nil {
			continue
		}

		createdController, err := newSyncer(registerContext)
		if err != nil {
			return nil, fmt.Errorf("register controller: %w", err)
		}

		loghelper.Infof("Start %s sync controller", createdController.Name())
		syncers = append(syncers, createdController)
	}

	return syncers, nil
}

func ExecuteInitializers(controllerCtx *config.ControllerContext, syncers []syncertypes.Object) error {
	registerContext := util.ToRegisterContext(controllerCtx)

	// execute in parallel because each one might be time-consuming
	errorGroup, ctx := errgroup.WithContext(controllerCtx.Context)
	registerContext.Context = ctx
	for _, s := range syncers {
		name := s.Name()
		initializer, ok := s.(syncertypes.Initializer)
		if ok {
			errorGroup.Go(func() error {
				err := initializer.Init(registerContext)
				if err != nil {
					return errors.Wrapf(err, "ensure prerequisites for %s syncer", name)
				}
				return nil
			})
		}
	}

	return errorGroup.Wait()
}

func RegisterIndices(ctx *config.ControllerContext, syncers []syncertypes.Object) error {
	registerContext := util.ToRegisterContext(ctx)
	for _, s := range syncers {
		indexRegisterer, ok := s.(syncertypes.IndicesRegisterer)
		if ok {
			err := indexRegisterer.RegisterIndices(registerContext)
			if err != nil {
				return errors.Wrapf(err, "register indices for %s syncer", s.Name())
			}
		}
	}

	return nil
}

func RegisterControllers(ctx *config.ControllerContext, syncers []syncertypes.Object) error {
	registerContext := util.ToRegisterContext(ctx)

	err := k8sdefaultendpoint.Register(ctx)
	if err != nil {
		return err
	}

	// register controller that maintains pod security standard check
	if ctx.Config.Policies.PodSecurityStandard != "" {
		err := RegisterPodSecurityController(ctx)
		if err != nil {
			return err
		}
	}

	// register controller that keeps CoreDNS NodeHosts config up to date
	err = RegisterCoreDNSController(ctx)
	if err != nil {
		return err
	}

	// register init manifests configmap watcher controller
	err = deploy.RegisterInitManifestsController(ctx)
	if err != nil {
		return err
	}

	// register service syncer to map services between host and virtual cluster
	err = RegisterServiceSyncControllers(ctx)
	if err != nil {
		return err
	}

	err = RegisterGenericSyncController(ctx)
	if err != nil {
		return err
	}

	// register controllers for resource synchronization
	for _, v := range syncers {
		// fake syncer?
		fakeSyncer, ok := v.(syncertypes.FakeSyncer)
		if ok {
			err = syncer.RegisterFakeSyncer(registerContext, fakeSyncer)
			if err != nil {
				return errors.Wrapf(err, "start %s syncer", v.Name())
			}
		} else {
			// real syncer?
			realSyncer, ok := v.(syncertypes.Syncer)
			if ok {
				err = syncer.RegisterSyncer(registerContext, realSyncer)
				if err != nil {
					return errors.Wrapf(err, "start %s syncer", v.Name())
				}
			} else {
				return fmt.Errorf("syncer %s does not implement fake syncer or syncer interface", v.Name())
			}
		}
	}

	return nil
}

func RegisterGenericSyncController(ctx *config.ControllerContext) error {
	err := generic.CreateExporters(ctx)
	if err != nil {
		return err
	}

	err = generic.CreateImporters(ctx)
	if err != nil {
		return err
	}

	return nil
}

func RegisterServiceSyncControllers(ctx *config.ControllerContext) error {
	hostNamespace := ctx.Config.WorkloadTargetNamespace
	if ctx.Config.Experimental.MultiNamespaceMode.Enabled {
		hostNamespace = ctx.Config.WorkloadNamespace
	}

	if len(ctx.Config.Networking.ReplicateServices.FromHost) > 0 {
		mapping, err := parseMapping(ctx.Config.Networking.ReplicateServices.FromHost, hostNamespace, "")
		if err != nil {
			return errors.Wrap(err, "parse physical service mapping")
		}

		// sync we are syncing from arbitrary physical namespaces we need to create a new
		// manager that listens on global services
		globalLocalManager, err := ctrl.NewManager(ctx.LocalManager.GetConfig(), ctrl.Options{
			Scheme: ctx.LocalManager.GetScheme(),
			MapperProvider: func(_ *rest.Config, _ *http.Client) (meta.RESTMapper, error) {
				return ctx.LocalManager.GetRESTMapper(), nil
			},
			Metrics:        metricsserver.Options{BindAddress: "0"},
			LeaderElection: false,
			NewClient:      blockingcacheclient.NewCacheClient,
		})
		if err != nil {
			return err
		}

		// start the manager
		go func() {
			err := globalLocalManager.Start(ctx.Context)
			if err != nil {
				panic(err)
			}
		}()

		// Wait for caches to be synced
		globalLocalManager.GetCache().WaitForCacheSync(ctx.Context)

		// register controller
		controller := &servicesync.ServiceSyncer{
			SyncServices:    mapping,
			CreateNamespace: true,
			CreateEndpoints: true,
			From:            globalLocalManager,
			To:              ctx.VirtualManager,
			Log:             loghelper.New("map-host-service-syncer"),
		}
		err = controller.Register()
		if err != nil {
			return errors.Wrap(err, "register physical service sync controller")
		}
	}

	if len(ctx.Config.Networking.ReplicateServices.ToHost) > 0 {
		mapping, err := parseMapping(ctx.Config.Networking.ReplicateServices.ToHost, "", hostNamespace)
		if err != nil {
			return errors.Wrap(err, "parse physical service mapping")
		}

		controller := &servicesync.ServiceSyncer{
			SyncServices:          mapping,
			IsVirtualToHostSyncer: true,
			From:                  ctx.VirtualManager,
			To:                    ctx.LocalManager,
			Log:                   loghelper.New("map-virtual-service-syncer"),
		}

		if ctx.Config.Experimental.MultiNamespaceMode.Enabled {
			controller.CreateEndpoints = true
		}

		err = controller.Register()
		if err != nil {
			return errors.Wrap(err, "register virtual service sync controller")
		}
	}

	return nil
}

func parseMapping(mappings []vclusterconfig.ServiceMapping, fromDefaultNamespace, toDefaultNamespace string) (map[string]types.NamespacedName, error) {
	ret := map[string]types.NamespacedName{}
	for _, m := range mappings {
		from := m.From
		to := m.To

		fromSplitted := strings.Split(from, "/")
		if len(fromSplitted) == 1 {
			if fromDefaultNamespace == "" {
				return nil, fmt.Errorf("invalid service mapping, please use namespace1/service1=service2")
			}

			from = fromDefaultNamespace + "/" + from
		} else if len(fromSplitted) != 2 {
			return nil, fmt.Errorf("invalid service mapping, please use namespace1/service1=service2")
		}

		toSplitted := strings.Split(to, "/")
		if len(toSplitted) == 1 {
			if toDefaultNamespace == "" {
				return nil, fmt.Errorf("invalid service mapping, please use namespace1/service1=namespace2/service2")
			}

			ret[from] = types.NamespacedName{
				Namespace: toDefaultNamespace,
				Name:      to,
			}
		} else if len(toSplitted) == 2 {
			if toDefaultNamespace != "" {
				return nil, fmt.Errorf("invalid service mapping, please use namespace1/service1=service2")
			}

			ret[from] = types.NamespacedName{
				Namespace: toSplitted[0],
				Name:      toSplitted[1],
			}
		} else {
			return nil, fmt.Errorf("invalid service mapping, please use namespace1/service1=service2")
		}
	}

	return ret, nil
}

func RegisterCoreDNSController(ctx *config.ControllerContext) error {
	controller := &coredns.NodeHostsReconciler{
		Client: ctx.VirtualManager.GetClient(),
		Log:    loghelper.New("corednsnodehosts-controller"),
	}
	err := controller.SetupWithManager(ctx.VirtualManager)
	if err != nil {
		return fmt.Errorf("unable to setup CoreDNS NodeHosts controller: %w", err)
	}
	return nil
}

func RegisterPodSecurityController(ctx *config.ControllerContext) error {
	controller := &podsecurity.Reconciler{
		Client:              ctx.VirtualManager.GetClient(),
		PodSecurityStandard: ctx.Config.Policies.PodSecurityStandard,
		Log:                 loghelper.New("podSecurity-controller"),
	}
	err := controller.SetupWithManager(ctx.VirtualManager)
	if err != nil {
		return fmt.Errorf("unable to setup pod security controller: %w", err)
	}
	return nil
}
