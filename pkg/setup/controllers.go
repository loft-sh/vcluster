package setup

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/loft-sh/vcluster/pkg/controllers"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/services"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/coredns"
	"github.com/loft-sh/vcluster/pkg/metricsapiservice"
	"github.com/loft-sh/vcluster/pkg/options"
	"github.com/loft-sh/vcluster/pkg/plugin"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/specialservices"
	syncertypes "github.com/loft-sh/vcluster/pkg/types"
	"github.com/loft-sh/vcluster/pkg/util/kubeconfig"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func StartControllers(
	controllerContext *options.ControllerContext,
	controlPlaneNamespace,
	controlPlaneService string,
	controlPlaneConfig *rest.Config,
) error {
	proOptions := controllerContext.Options.ProOptions

	// exchange control plane client
	controlPlaneClient, err := pro.ExchangeControlPlaneClient(controllerContext, controlPlaneNamespace, controlPlaneConfig)
	if err != nil {
		return err
	}

	// start coredns & create syncers
	var syncers []syncertypes.Object
	if !proOptions.NoopSyncer {
		// setup CoreDNS according to the manifest file
		// skip this if both integrated and dedicated coredns
		// deployments are explicitly disabled
		go func() {
			// apply coredns
			ApplyCoreDNS(controllerContext)

			// delete coredns deployment if integrated core dns
			if proOptions.IntegratedCoredns {
				err := controllerContext.VirtualManager.GetClient().Delete(controllerContext.Context, &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "coredns",
						Namespace: "kube-system",
					},
				})
				if err != nil && !kerrors.IsNotFound(err) {
					klog.Errorf("Error deleting coredns deployment: %v", err)
				}
			}
		}()

		// init syncers
		syncers, err = controllers.Create(controllerContext)
		if err != nil {
			return errors.Wrap(err, "instantiate controllers")
		}
	}

	// start managers
	err = StartManagers(controllerContext, syncers)
	if err != nil {
		return err
	}

	// sync remote Endpoints
	if proOptions.RemoteKubeConfig != "" {
		err := pro.SyncRemoteEndpoints(
			controllerContext.Context,
			types.NamespacedName{
				Namespace: controlPlaneNamespace,
				Name:      controlPlaneService,
			},
			controlPlaneClient,
			types.NamespacedName{
				Namespace: controllerContext.CurrentNamespace,
				Name:      controllerContext.Options.ServiceName,
			},
			controllerContext.CurrentNamespaceClient,
		)
		if err != nil {
			return errors.Wrap(err, "sync remote endpoints")
		}
	}

	// sync endpoints for noop syncer
	if proOptions.NoopSyncer && proOptions.SyncKubernetesService {
		err := pro.SyncNoopSyncerEndpoints(
			controllerContext,
			types.NamespacedName{
				Namespace: controlPlaneNamespace,
				Name:      controlPlaneService + "-lb",
			},
			controlPlaneClient,
			types.NamespacedName{
				Namespace: controlPlaneNamespace,
				Name:      controlPlaneService + "-proxy",
			},
			controlPlaneService,
		)
		if err != nil {
			return errors.Wrap(err, "sync proxied cluster endpoints")
		}
	}

	// if not noop syncer
	if !proOptions.NoopSyncer {
		// make sure the kubernetes service is synced
		err = SyncKubernetesService(controllerContext)
		if err != nil {
			return errors.Wrap(err, "sync kubernetes service")
		}

		// register controllers
		err = controllers.RegisterControllers(controllerContext, syncers)
		if err != nil {
			return err
		}
	}

	// write the kube config to secret
	go func() {
		wait.Until(func() {
			err := WriteKubeConfigToSecret(controllerContext.Context, controlPlaneNamespace, controlPlaneClient, controllerContext.Options, controllerContext.VirtualRawConfig, proOptions.RemoteKubeConfig != "")
			if err != nil {
				klog.Errorf("Error writing kube config to secret: %v", err)
			}
		}, time.Minute, controllerContext.StopChan)
	}()

	// set leader
	if !controllerContext.Options.DisablePlugins {
		err = plugin.DefaultManager.SetLeader(controllerContext.Context)
		if err != nil {
			return fmt.Errorf("plugin set leader: %w", err)
		}
	}

	return nil
}

func ApplyCoreDNS(controllerContext *options.ControllerContext) {
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

func StartManagers(controllerContext *options.ControllerContext, syncers []syncertypes.Object) error {
	// execute controller initializers to setup prereqs, etc.
	err := controllers.ExecuteInitializers(controllerContext, syncers)
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
	klog.Infof("Starting local & virtual managers...")
	controllerContext.LocalManager.GetCache().WaitForCacheSync(controllerContext.Context)
	controllerContext.VirtualManager.GetCache().WaitForCacheSync(controllerContext.Context)
	klog.Infof("Successfully started local & virtual manager")

	// register APIService
	go RegisterOrDeregisterAPIService(controllerContext)

	return nil
}

func RegisterOrDeregisterAPIService(ctx *options.ControllerContext) {
	err := metricsapiservice.RegisterOrDeregisterAPIService(ctx)
	if err != nil {
		klog.Errorf("Error registering metrics apiservice: %v", err)
	}
}

func WriteKubeConfigToSecret(ctx context.Context, currentNamespace string, currentNamespaceClient client.Client, options *options.VirtualClusterOptions, config *clientcmdapi.Config, isRemote bool) error {
	config, err := CreateVClusterKubeConfig(config, options)
	if err != nil {
		return err
	}

	if options.KubeConfigContextName != "" {
		config.CurrentContext = options.KubeConfigContextName
		// update authInfo
		for k := range config.AuthInfos {
			config.AuthInfos[options.KubeConfigContextName] = config.AuthInfos[k]
			if k != options.KubeConfigContextName {
				delete(config.AuthInfos, k)
			}
			break
		}

		// update cluster
		for k := range config.Clusters {
			config.Clusters[options.KubeConfigContextName] = config.Clusters[k]
			if k != options.KubeConfigContextName {
				delete(config.Clusters, k)
			}
			break
		}

		// update context
		for k := range config.Contexts {
			tmpCtx := config.Contexts[k]
			tmpCtx.Cluster = options.KubeConfigContextName
			tmpCtx.AuthInfo = options.KubeConfigContextName
			config.Contexts[options.KubeConfigContextName] = tmpCtx
			if k != options.KubeConfigContextName {
				delete(config.Contexts, k)
			}
			break
		}
	}

	// check if we need to write the kubeconfig secrete to the default location as well
	if options.KubeConfigSecret != "" {
		// which namespace should we create the additional secret in?
		secretNamespace := options.KubeConfigSecretNamespace
		if secretNamespace == "" {
			secretNamespace = currentNamespace
		}

		// write the extra secret
		err = kubeconfig.WriteKubeConfig(ctx, currentNamespaceClient, options.KubeConfigSecret, secretNamespace, config, isRemote)
		if err != nil {
			return fmt.Errorf("creating %s secret in the %s ns failed: %w", options.KubeConfigSecret, secretNamespace, err)
		}
	}

	// write the default Secret
	return kubeconfig.WriteKubeConfig(ctx, currentNamespaceClient, kubeconfig.GetDefaultSecretName(translate.VClusterName), currentNamespace, config, isRemote)
}
