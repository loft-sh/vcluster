package setup

import (
	"context"
	"math"
	"time"

	"github.com/loft-sh/vcluster/pkg/controllers"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/services"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/coredns"
	"github.com/loft-sh/vcluster/pkg/metricsapiservice"
	"github.com/loft-sh/vcluster/pkg/plugin"
	"github.com/loft-sh/vcluster/pkg/setup/options"
	"github.com/loft-sh/vcluster/pkg/specialservices"
	"github.com/loft-sh/vcluster/pkg/telemetry"
	telemetrytypes "github.com/loft-sh/vcluster/pkg/telemetry/types"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
)

func StartControllers(controllerContext *options.ControllerContext) error {
	if telemetry.Collector.IsEnabled() {
		telemetry.Collector.RecordEvent(telemetry.Collector.NewEvent(telemetrytypes.EventLeadershipStarted))
	}

	// setup CoreDNS according to the manifest file
	go func() {
		_ = wait.ExponentialBackoffWithContext(controllerContext.Context, wait.Backoff{Duration: time.Second, Factor: 1.5, Cap: time.Minute, Steps: math.MaxInt32}, func(ctx context.Context) (bool, error) {
			err := coredns.ApplyManifest(ctx, controllerContext.Options.DefaultImageRegistry, controllerContext.VirtualManager.GetConfig(), controllerContext.VirtualClusterVersion)
			if err != nil {
				if errors.Is(err, coredns.ErrNoCoreDNSManifests) {
					klog.Infof("No CoreDNS manifests found, skipping CoreDNS configuration")
					return true, nil
				}
				klog.Infof("Failed to apply CoreDNS configuration from the manifest file: %v", err)
				return false, nil
			}
			klog.Infof("CoreDNS configuration from the manifest file applied successfully")
			return true, nil
		})
	}()

	// instantiate controllers
	syncers, err := controllers.Create(controllerContext)
	if err != nil {
		return errors.Wrap(err, "instantiate controllers")
	}

	// execute controller initializers to setup prereqs, etc.
	err = controllers.ExecuteInitializers(controllerContext, syncers)
	if err != nil {
		return errors.Wrap(err, "execute initializers")
	}

	// register indices
	err = controllers.RegisterIndices(controllerContext, syncers)
	if err != nil {
		return err
	}

	// start the local manager
	go func() {
		err := controllerContext.LocalManager.Start(controllerContext.Context)
		if err != nil {
			panic(err)
		}
	}()

	// start the virtual cluster manager
	go func() {
		err := controllerContext.VirtualManager.Start(controllerContext.Context)
		if err != nil {
			panic(err)
		}
	}()

	// Wait for caches to be synced
	controllerContext.LocalManager.GetCache().WaitForCacheSync(controllerContext.Context)
	controllerContext.VirtualManager.GetCache().WaitForCacheSync(controllerContext.Context)

	// register APIService
	go RegisterOrDeregisterAPIService(controllerContext)

	// make sure owner is set if it is there
	err = FindOwner(controllerContext)
	if err != nil {
		return errors.Wrap(err, "finding vcluster pod owner")
	}

	// make sure the kubernetes service is synced
	err = SyncKubernetesService(controllerContext)
	if err != nil {
		return errors.Wrap(err, "sync kubernetes service")
	}

	// write the kube config to secret
	go func() {
		wait.Until(func() {
			err := WriteKubeConfigToSecret(controllerContext.Context, controllerContext.CurrentNamespace, controllerContext.CurrentNamespaceClient, controllerContext.Options, controllerContext.VirtualRawConfig)
			if err != nil {
				klog.Errorf("Error writing kube config to secret: %v", err)
			}
		}, time.Minute, controllerContext.StopChan)
	}()

	// register controllers
	err = controllers.RegisterControllers(controllerContext, syncers)
	if err != nil {
		return err
	}

	// set leader
	if !controllerContext.Options.DisablePlugins {
		plugin.DefaultManager.SetLeader(true)
	}

	return nil
}

func FindOwner(ctx *options.ControllerContext) error {
	if ctx.CurrentNamespace != ctx.Options.TargetNamespace {
		if ctx.Options.SetOwner {
			klog.Warningf("Skip setting owner, because current namespace %s != target namespace %s", ctx.CurrentNamespace, ctx.Options.TargetNamespace)
		}
		return nil
	}

	if ctx.Options.SetOwner {
		service := &corev1.Service{}
		err := ctx.CurrentNamespaceClient.Get(ctx.Context, types.NamespacedName{Namespace: ctx.CurrentNamespace, Name: ctx.Options.ServiceName}, service)
		if err != nil {
			return errors.Wrap(err, "get vcluster service")
		}

		translate.Owner = service
		return nil
	}

	return nil
}

func SyncKubernetesService(ctx *options.ControllerContext) error {
	err := specialservices.SyncKubernetesService(
		&synccontext.SyncContext{
			Context:                ctx.Context,
			Log:                    loghelper.New("sync-kubernetes-service"),
			PhysicalClient:         ctx.LocalManager.GetClient(),
			VirtualClient:          ctx.VirtualManager.GetClient(),
			CurrentNamespace:       ctx.CurrentNamespace,
			CurrentNamespaceClient: ctx.CurrentNamespaceClient,
		},
		ctx.CurrentNamespace,
		ctx.Options.ServiceName,
		types.NamespacedName{
			Name:      specialservices.DefaultKubernetesSVCName,
			Namespace: specialservices.DefaultKubernetesSVCNamespace,
		},
		services.TranslateServicePorts)
	if err != nil {
		if kerrors.IsConflict(err) {
			klog.Errorf("Error syncing kubernetes service: %v", err)
			time.Sleep(time.Second)
			return SyncKubernetesService(ctx)
		}

		return errors.Wrap(err, "sync kubernetes service")
	}
	return nil
}

func RegisterOrDeregisterAPIService(ctx *options.ControllerContext) {
	// check api-service for metrics server
	err := metricsapiservice.RegisterOrDeregisterAPIService(ctx.Context, ctx.Options, ctx.VirtualManager.GetClient())
	if err != nil {
		klog.Errorf("Error registering metrics apiservice: %v", err)
	}
}
