package controllers

import (
	"fmt"
	"net/http"
	"strings"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/controllers/deploy"
	"github.com/loft-sh/vcluster/pkg/controllers/generic"
	"github.com/loft-sh/vcluster/pkg/controllers/servicesync"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/blockingcacheclient"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/loft-sh/vcluster/pkg/controllers/coredns"
	"github.com/loft-sh/vcluster/pkg/controllers/k8sdefaultendpoint"
	"github.com/loft-sh/vcluster/pkg/controllers/podsecurity"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/pkg/errors"
)

func RegisterControllers(ctx *synccontext.ControllerContext, syncers []syncertypes.Object) error {
	registerContext := ctx.ToRegisterContext()

	// start default endpoint controller
	err := k8sdefaultendpoint.Register(ctx)
	if err != nil {
		return err
	}

	// register controller that maintains pod security standard check
	if ctx.Config.Policies.PodSecurityStandard != "" {
		err := registerPodSecurityController(ctx)
		if err != nil {
			return err
		}
	}

	// register controller that keeps CoreDNS NodeHosts config up to date
	err = registerCoreDNSController(ctx)
	if err != nil {
		return err
	}

	// register init manifests configmap watcher controller
	err = deploy.RegisterInitManifestsController(ctx)
	if err != nil {
		return err
	}

	// register service syncer to map services between host and virtual cluster
	err = registerServiceSyncControllers(ctx)
	if err != nil {
		return err
	}

	// register generic sync controllers
	err = registerGenericSyncController(ctx)
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
		}

		// real syncer?
		realSyncer, ok := v.(syncertypes.Syncer)
		if ok {
			err = syncer.RegisterSyncer(registerContext, realSyncer)
			if err != nil {
				return errors.Wrapf(err, "start %s syncer", v.Name())
			}
		}

		// custom syncer?
		customSyncer, ok := v.(syncertypes.ControllerStarter)
		if ok {
			err = customSyncer.Register(registerContext)
			if err != nil {
				return errors.Wrapf(err, "start %s syncer", v.Name())
			}
		}
	}

	return nil
}

func registerGenericSyncController(ctx *synccontext.ControllerContext) error {
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

func registerServiceSyncControllers(ctx *synccontext.ControllerContext) error {
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
		if globalLocalManager == nil {
			return errors.New("nil globalLocalManager")
		}

		// start the manager
		go func() {
			err := globalLocalManager.Start(ctx)
			if err != nil {
				panic(err)
			}
		}()

		// Wait for caches to be synced
		globalLocalManager.GetCache().WaitForCacheSync(ctx)

		// register controller
		name := "map-host-service-syncer"
		controller := &servicesync.ServiceSyncer{
			Name:            name,
			SyncContext:     ctx.ToRegisterContext().ToSyncContext(name),
			SyncServices:    mapping,
			CreateNamespace: true,
			CreateEndpoints: true,
			From:            globalLocalManager,
			To:              ctx.VirtualManager,
			Log:             loghelper.New(name),
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
		name := "map-virtual-service-syncer"
		controller := &servicesync.ServiceSyncer{
			Name:                  name,
			SyncContext:           ctx.ToRegisterContext().ToSyncContext(name),
			SyncServices:          mapping,
			IsVirtualToHostSyncer: true,
			From:                  ctx.VirtualManager,
			To:                    ctx.LocalManager,
			Log:                   loghelper.New(name),
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

func registerCoreDNSController(ctx *synccontext.ControllerContext) error {
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

func registerPodSecurityController(ctx *synccontext.ControllerContext) error {
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
