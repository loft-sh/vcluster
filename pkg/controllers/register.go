package controllers

import (
	"fmt"
	"strings"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd"

	"github.com/loft-sh/vcluster/pkg/controllers/servicesync"
	"github.com/loft-sh/vcluster/pkg/helm"
	"github.com/loft-sh/vcluster/pkg/plugin"
	"github.com/loft-sh/vcluster/pkg/util/blockingcacheclient"
	"github.com/loft-sh/vcluster/pkg/util/pluginhookclient"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/loft-sh/vcluster/pkg/controllers/k8sdefaultendpoint"
	"github.com/loft-sh/vcluster/pkg/controllers/manifests"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/csidrivers"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/csinodes"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/csistoragecapacities"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/ingressclasses"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/namespaces"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/serviceaccounts"

	"github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/loft-sh/vcluster/pkg/controllers/coredns"
	"github.com/loft-sh/vcluster/pkg/controllers/podsecurity"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/configmaps"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/endpoints"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/events"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/ingresses"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/networkpolicies"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/persistentvolumeclaims"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/persistentvolumes"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/poddisruptionbudgets"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/pods"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/priorityclasses"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/secrets"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/services"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/storageclasses"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/volumesnapshots/volumesnapshotclasses"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/volumesnapshots/volumesnapshotcontents"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/volumesnapshots/volumesnapshots"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

var ResourceControllers = map[string][]func(*synccontext.RegisterContext) (syncer.Object, error){
	"services":               {services.New},
	"configmaps":             {configmaps.New},
	"secrets":                {secrets.New},
	"endpoints":              {endpoints.New},
	"pods":                   {pods.New},
	"events":                 {events.New},
	"persistentvolumeclaims": {persistentvolumeclaims.New},
	"ingresses":              {ingresses.New},
	"ingressclasses":         {ingressclasses.New},
	"storageclasses":         {storageclasses.New},
	"hoststorageclasses":     {storageclasses.NewHostStorageClassSyncer},
	"priorityclasses":        {priorityclasses.New},
	"nodes,fake-nodes":       {nodes.New},
	"poddisruptionbudgets":   {poddisruptionbudgets.New},
	"networkpolicies":        {networkpolicies.New},
	"volumesnapshots":        {volumesnapshotclasses.New, volumesnapshots.New, volumesnapshotcontents.New},
	"serviceaccounts":        {serviceaccounts.New},
	"csinodes":               {csinodes.New},
	"csidrivers":             {csidrivers.New},
	"csistoragecapacities":   {csistoragecapacities.New},
	"namespaces":             {namespaces.New},
	"persistentvolumes,fake-persistentvolumes": {persistentvolumes.New},
}

func Create(ctx *context.ControllerContext) ([]syncer.Object, error) {
	registerContext := ToRegisterContext(ctx)

	// register controllers for resource synchronization
	syncers := []syncer.Object{}
	for k, v := range ResourceControllers {
		for _, controllerNew := range v {
			controllers := strings.Split(k, ",")
			for _, controller := range controllers {
				if ctx.Controllers.Has(controller) {
					loghelper.Infof("Start %s sync controller", controller)
					ctrl, err := controllerNew(registerContext)
					if err != nil {
						return nil, errors.Wrapf(err, "register %s controller", controller)
					}

					syncers = append(syncers, ctrl)
					break
				}
			}
		}
	}

	return syncers, nil
}

func ExecuteInitializers(controllerCtx *context.ControllerContext, syncers []syncer.Object) error {
	registerContext := ToRegisterContext(controllerCtx)

	// execute in parallel because each one might be time-consuming
	errorGroup, ctx := errgroup.WithContext(controllerCtx.Context)
	registerContext.Context = ctx
	for _, s := range syncers {
		initializer, ok := s.(syncer.Initializer)
		if ok {
			errorGroup.Go(func() error {
				err := initializer.Init(registerContext)
				if err != nil {
					return errors.Wrapf(err, "ensure prerequisites for %s syncer", s.Name())
				}
				return nil
			})
		}
	}

	return errorGroup.Wait()
}

func RegisterIndices(ctx *context.ControllerContext, syncers []syncer.Object) error {
	registerContext := ToRegisterContext(ctx)
	for _, s := range syncers {
		indexRegisterer, ok := s.(syncer.IndicesRegisterer)
		if ok {
			err := indexRegisterer.RegisterIndices(registerContext)
			if err != nil {
				return errors.Wrapf(err, "register indices for %s syncer", s.Name())
			}
		}
	}

	return nil
}

func RegisterControllers(ctx *context.ControllerContext, syncers []syncer.Object) error {
	registerContext := ToRegisterContext(ctx)

	err := k8sdefaultendpoint.Register(ctx)
	if err != nil {
		return err
	}

	// register controller that maintains pod security standard check
	if ctx.Options.EnforcePodSecurityStandard != "" {
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
	err = registerInitManifestsController(ctx)
	if err != nil {
		return err
	}

	// register service syncer to map services between host and virtual cluster
	err = registerServiceSyncControllers(ctx)
	if err != nil {
		return err
	}

	// register controllers for resource synchronization
	for _, v := range syncers {
		// fake syncer?
		fakeSyncer, ok := v.(syncer.FakeSyncer)
		if ok {
			err = syncer.RegisterFakeSyncer(registerContext, fakeSyncer)
			if err != nil {
				return errors.Wrapf(err, "start %s syncer", v.Name())
			}
		} else {
			// real syncer?
			realSyncer, ok := v.(syncer.Syncer)
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

func registerInitManifestsController(ctx *context.ControllerContext) error {
	currentNamespaceManager := ctx.LocalManager
	if ctx.Options.TargetNamespace != ctx.CurrentNamespace {
		var err error
		currentNamespaceManager, err = ctrl.NewManager(ctx.LocalManager.GetConfig(), ctrl.Options{
			Scheme: ctx.LocalManager.GetScheme(),
			MapperProvider: func(c *rest.Config) (meta.RESTMapper, error) {
				return ctx.LocalManager.GetRESTMapper(), nil
			},
			MetricsBindAddress: "0",
			LeaderElection:     false,
			Namespace:          ctx.CurrentNamespace,
			NewClient:          pluginhookclient.NewPhysicalPluginClientFactory(blockingcacheclient.NewCacheClient),
		})
		if err != nil {
			return err
		}

		// start the manager
		go func() {
			err := currentNamespaceManager.Start(ctx.Context)
			if err != nil {
				panic(err)
			}
		}()

		// Wait for caches to be synced
		currentNamespaceManager.GetCache().WaitForCacheSync(ctx.Context)
	}

	vconfig, err := plugin.ConvertRestConfigToClientConfig(ctx.VirtualManager.GetConfig())
	if err != nil {
		return err
	}

	vConfigRaw, err := vconfig.RawConfig()
	if err != nil {
		return err
	}

	helmBinaryPath, err := cmd.GetHelmBinaryPath(log.GetInstance())
	if err != nil {
		return err
	}

	controller := &manifests.InitManifestsConfigMapReconciler{
		LocalClient:    currentNamespaceManager.GetClient(),
		Log:            loghelper.New("init-manifests-controller"),
		VirtualManager: ctx.VirtualManager,

		HelmClient: helm.NewClient(&vConfigRaw, log.GetInstance(), helmBinaryPath),
	}

	err = controller.SetupWithManager(currentNamespaceManager)
	if err != nil {
		return fmt.Errorf("unable to setup init manifests configmap controller: %v", err)
	}

	return nil
}

func registerServiceSyncControllers(ctx *context.ControllerContext) error {
	if len(ctx.Options.MapHostServices) > 0 {
		mapping, err := parseMapping(ctx.Options.MapHostServices, ctx.Options.TargetNamespace, "")
		if err != nil {
			return errors.Wrap(err, "parse physical service mapping")
		}

		// sync we are syncing from arbitrary physical namespaces we need to create a new
		// manager that listens on global services
		globalLocalManager, err := ctrl.NewManager(ctx.LocalManager.GetConfig(), ctrl.Options{
			Scheme: ctx.LocalManager.GetScheme(),
			MapperProvider: func(c *rest.Config) (meta.RESTMapper, error) {
				return ctx.LocalManager.GetRESTMapper(), nil
			},
			MetricsBindAddress: "0",
			LeaderElection:     false,
			NewClient:          blockingcacheclient.NewCacheClient,
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

	if len(ctx.Options.MapVirtualServices) > 0 {
		mapping, err := parseMapping(ctx.Options.MapVirtualServices, "", ctx.Options.TargetNamespace)
		if err != nil {
			return errors.Wrap(err, "parse physical service mapping")
		}

		controller := &servicesync.ServiceSyncer{
			SyncServices: mapping,
			From:         ctx.VirtualManager,
			To:           ctx.LocalManager,
			Log:          loghelper.New("map-virtual-service-syncer"),
		}
		err = controller.Register()
		if err != nil {
			return errors.Wrap(err, "register virtual service sync controller")
		}
	}

	return nil
}

func parseMapping(mappings []string, fromDefaultNamespace, toDefaultNamespace string) (map[string]types.NamespacedName, error) {
	ret := map[string]types.NamespacedName{}
	for _, m := range mappings {
		splitted := strings.Split(m, "=")
		if len(splitted) != 2 {
			return nil, fmt.Errorf("invalid service mapping, please use namespace1/service1=service2")
		} else if len(splitted[0]) == 0 || len(splitted[1]) == 0 {
			return nil, fmt.Errorf("invalid service mapping, please use namespace1/service1=service2")
		}

		fromSplitted := strings.Split(splitted[0], "/")
		if len(fromSplitted) == 1 {
			if fromDefaultNamespace == "" {
				return nil, fmt.Errorf("invalid service mapping, please use namespace1/service1=service2")
			}

			splitted[0] = fromDefaultNamespace + "/" + splitted[0]
		} else if len(fromSplitted) != 2 {
			return nil, fmt.Errorf("invalid service mapping, please use namespace1/service1=service2")
		}

		toSplitted := strings.Split(splitted[1], "/")
		if len(toSplitted) == 1 {
			if toDefaultNamespace == "" {
				return nil, fmt.Errorf("invalid service mapping, please use namespace1/service1=namespace2/service2")
			}

			ret[splitted[0]] = types.NamespacedName{
				Namespace: toDefaultNamespace,
				Name:      splitted[1],
			}
		} else if len(toSplitted) == 2 {
			if toDefaultNamespace != "" {
				return nil, fmt.Errorf("invalid service mapping, please use namespace1/service1=service2")
			}

			ret[splitted[0]] = types.NamespacedName{
				Namespace: toSplitted[0],
				Name:      toSplitted[1],
			}
		} else {
			return nil, fmt.Errorf("invalid service mapping, please use namespace1/service1=service2")
		}
	}

	return ret, nil
}

func registerCoreDNSController(ctx *context.ControllerContext) error {
	controller := &coredns.CoreDNSNodeHostsReconciler{
		Client: ctx.VirtualManager.GetClient(),
		Log:    loghelper.New("corednsnodehosts-controller"),
	}
	err := controller.SetupWithManager(ctx.VirtualManager)
	if err != nil {
		return fmt.Errorf("unable to setup CoreDNS NodeHosts controller: %v", err)
	}
	return nil
}

func registerPodSecurityController(ctx *context.ControllerContext) error {
	controller := &podsecurity.PodSecurityReconciler{
		Client:              ctx.VirtualManager.GetClient(),
		PodSecurityStandard: ctx.Options.EnforcePodSecurityStandard,
		Log:                 loghelper.New("podSecurity-controller"),
	}
	err := controller.SetupWithManager(ctx.VirtualManager)
	if err != nil {
		return fmt.Errorf("unable to setup pod security controller: %v", err)
	}
	return nil
}

func ToRegisterContext(ctx *context.ControllerContext) *synccontext.RegisterContext {
	return &synccontext.RegisterContext{
		Context: ctx.Context,

		Options:     ctx.Options,
		Controllers: ctx.Controllers,

		CurrentNamespace:       ctx.CurrentNamespace,
		CurrentNamespaceClient: ctx.CurrentNamespaceClient,

		VirtualManager:  ctx.VirtualManager,
		PhysicalManager: ctx.LocalManager,
	}
}
