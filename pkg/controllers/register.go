package controllers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd"
	"github.com/loft-sh/vcluster/pkg/options"
	"github.com/loft-sh/vcluster/pkg/util/kubeconfig"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/controllers/generic"
	"github.com/loft-sh/vcluster/pkg/controllers/servicesync"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	"github.com/loft-sh/vcluster/pkg/helm"
	"github.com/loft-sh/vcluster/pkg/util/blockingcacheclient"
	util "github.com/loft-sh/vcluster/pkg/util/context"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/loft-sh/vcluster/pkg/controllers/k8sdefaultendpoint"
	"github.com/loft-sh/vcluster/pkg/controllers/manifests"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/csidrivers"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/csinodes"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/csistoragecapacities"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/ingressclasses"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/namespaces"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/serviceaccounts"

	"github.com/loft-sh/log"
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
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	syncertypes "github.com/loft-sh/vcluster/pkg/types"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

var ResourceControllers = map[string][]func(*synccontext.RegisterContext) (syncertypes.Object, error){
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

func Create(ctx *options.ControllerContext) ([]syncertypes.Object, error) {
	registerContext := util.ToRegisterContext(ctx)

	// register controllers for resource synchronization
	syncers := []syncertypes.Object{}
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

func ExecuteInitializers(controllerCtx *options.ControllerContext, syncers []syncertypes.Object) error {
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

func RegisterIndices(ctx *options.ControllerContext, syncers []syncertypes.Object) error {
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

func RegisterControllers(ctx *options.ControllerContext, syncers []syncertypes.Object) error {
	registerContext := util.ToRegisterContext(ctx)

	err := k8sdefaultendpoint.Register(ctx)
	if err != nil {
		return err
	}

	// register controller that maintains pod security standard check
	if ctx.Options.EnforcePodSecurityStandard != "" {
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
	err = RegisterInitManifestsController(ctx)
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

func RegisterGenericSyncController(ctx *options.ControllerContext) error {
	// first check if a generic CRD config is provided and we actually need
	// to create any of these syncer controllers
	c := os.Getenv(options.GenericConfig)
	if strings.TrimSpace(c) == "" || strings.TrimSpace(c) == "---" {
		// empty configuration, no need for creating any syncer controllers
		loghelper.Infof("no generic config provided, skipping creating controllers")

		return nil
	}

	configuration, err := config.Parse(c)
	if err != nil {
		loghelper.Infof("error parsing the config %v", err.Error())
		return errors.Wrapf(err, "parsing the config")
	}

	loghelper.Infof("generic config provided, parsed successfully")

	err = generic.CreateExporters(ctx, configuration)
	if err != nil {
		return err
	}

	err = generic.CreateImporters(ctx, configuration)
	if err != nil {
		return err
	}

	return nil
}

func RegisterInitManifestsController(controllerCtx *options.ControllerContext) error {
	vconfig, err := kubeconfig.ConvertRestConfigToClientConfig(controllerCtx.VirtualManager.GetConfig())
	if err != nil {
		return err
	}

	vConfigRaw, err := vconfig.RawConfig()
	if err != nil {
		return err
	}

	helmBinaryPath, err := cmd.GetHelmBinaryPath(controllerCtx.Context, log.GetInstance())
	if err != nil {
		return err
	}

	controller := &manifests.InitManifestsConfigMapReconciler{
		LocalClient:    controllerCtx.CurrentNamespaceClient,
		Log:            loghelper.New("init-manifests-controller"),
		VirtualManager: controllerCtx.VirtualManager,

		HelmClient: helm.NewClient(&vConfigRaw, log.GetInstance(), helmBinaryPath),
	}

	go func() {
		wait.JitterUntilWithContext(controllerCtx.Context, func(ctx context.Context) {
			for {
				result, err := controller.Reconcile(ctx, ctrl.Request{
					NamespacedName: types.NamespacedName{
						Namespace: controllerCtx.CurrentNamespace,
						Name:      translate.VClusterName + manifests.InitManifestSuffix,
					},
				})
				if err != nil {
					klog.Errorf("Error reconciling init_configmap: %v", err)
					break
				} else if !result.Requeue {
					break
				}
			}
		}, time.Second*10, 1.0, true)
	}()

	return nil
}

func RegisterServiceSyncControllers(ctx *options.ControllerContext) error {
	hostNamespace := ctx.Options.TargetNamespace
	if ctx.Options.MultiNamespaceMode {
		hostNamespace = ctx.CurrentNamespace
	}

	if len(ctx.Options.MapHostServices) > 0 {
		mapping, err := parseMapping(ctx.Options.MapHostServices, hostNamespace, "")
		if err != nil {
			return errors.Wrap(err, "parse physical service mapping")
		}

		// sync we are syncing from arbitrary physical namespaces we need to create a new
		// manager that listens on global services
		globalLocalManager, err := ctrl.NewManager(ctx.LocalManager.GetConfig(), ctrl.Options{
			Scheme: ctx.LocalManager.GetScheme(),
			MapperProvider: func(c *rest.Config, httpClient *http.Client) (meta.RESTMapper, error) {
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

	if len(ctx.Options.MapVirtualServices) > 0 {
		mapping, err := parseMapping(ctx.Options.MapVirtualServices, "", hostNamespace)
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

		if ctx.Options.MultiNamespaceMode {
			controller.CreateEndpoints = true
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

func RegisterCoreDNSController(ctx *options.ControllerContext) error {
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

func RegisterPodSecurityController(ctx *options.ControllerContext) error {
	controller := &podsecurity.Reconciler{
		Client:              ctx.VirtualManager.GetClient(),
		PodSecurityStandard: ctx.Options.EnforcePodSecurityStandard,
		Log:                 loghelper.New("podSecurity-controller"),
	}
	err := controller.SetupWithManager(ctx.VirtualManager)
	if err != nil {
		return fmt.Errorf("unable to setup pod security controller: %w", err)
	}
	return nil
}
