package setup

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/controllers"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/services"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/coredns"
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
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func StartControllers(controllerContext *config.ControllerContext) error {
	// exchange control plane client
	controlPlaneClient, err := pro.ExchangeControlPlaneClient(controllerContext)
	if err != nil {
		return err
	}

	// start coredns & create syncers
	var syncers []syncertypes.Object
	if !controllerContext.Config.Experimental.SyncSettings.DisableSync {
		// setup CoreDNS according to the manifest file
		// skip this if both integrated and dedicated coredns
		// deployments are explicitly disabled
		go func() {
			// apply coredns
			ApplyCoreDNS(controllerContext)

			// delete coredns deployment if integrated core dns
			if controllerContext.Config.ControlPlane.CoreDNS.Embedded {
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
	if controllerContext.Config.Experimental.IsolatedControlPlane.KubeConfig != "" {
		err := pro.SyncRemoteEndpoints(
			controllerContext.Context,
			types.NamespacedName{
				Namespace: controllerContext.Config.ControlPlaneNamespace,
				Name:      controllerContext.Config.ControlPlaneService,
			},
			controlPlaneClient,
			types.NamespacedName{
				Namespace: controllerContext.Config.WorkloadNamespace,
				Name:      controllerContext.Config.WorkloadService,
			},
			controllerContext.WorkloadNamespaceClient,
		)
		if err != nil {
			return errors.Wrap(err, "sync remote endpoints")
		}
	}

	// sync endpoints for noop syncer
	if controllerContext.Config.Experimental.SyncSettings.DisableSync && controllerContext.Config.Experimental.SyncSettings.RewriteKubernetesService {
		err := pro.SyncNoopSyncerEndpoints(
			controllerContext,
			types.NamespacedName{
				Namespace: controllerContext.Config.ControlPlaneNamespace,
				Name:      controllerContext.Config.ControlPlaneService,
			},
			controlPlaneClient,
			types.NamespacedName{
				Namespace: controllerContext.Config.ControlPlaneNamespace,
				Name:      controllerContext.Config.ControlPlaneService + "-proxy",
			},
			controllerContext.Config.ControlPlaneService,
		)
		if err != nil {
			return errors.Wrap(err, "sync proxied cluster endpoints")
		}
	}

	// if not noop syncer
	if !controllerContext.Config.Experimental.SyncSettings.DisableSync {
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

	// register pro controllers
	if err := pro.RegisterProControllers(controllerContext); err != nil {
		return fmt.Errorf("register pro controllers: %w", err)
	}

	// run leader hooks
	for _, hook := range controllerContext.AcquiredLeaderHooks {
		err = hook(controllerContext)
		if err != nil {
			return fmt.Errorf("execute controller hook: %w", err)
		}
	}

	// write the kube config to secret
	go func() {
		wait.Until(func() {
			err := WriteKubeConfigToSecret(controllerContext.Context, controllerContext.Config.ControlPlaneNamespace, controlPlaneClient, controllerContext.Config, controllerContext.VirtualRawConfig)
			if err != nil {
				klog.Errorf("Error writing kube config to secret: %v", err)
			}
		}, time.Minute, controllerContext.StopChan)
	}()

	// set leader
	err = plugin.DefaultManager.SetLeader(controllerContext.Context)
	if err != nil {
		return fmt.Errorf("plugin set leader: %w", err)
	}

	return nil
}

func ApplyCoreDNS(controllerContext *config.ControllerContext) {
	_ = wait.ExponentialBackoffWithContext(controllerContext.Context, wait.Backoff{Duration: time.Second, Factor: 1.5, Cap: time.Minute, Steps: math.MaxInt32}, func(ctx context.Context) (bool, error) {
		err := coredns.ApplyManifest(ctx, controllerContext.Config.ControlPlane.Advanced.DefaultImageRegistry, controllerContext.VirtualManager.GetConfig(), controllerContext.VirtualClusterVersion)
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

func SyncKubernetesService(ctx *config.ControllerContext) error {
	err := specialservices.SyncKubernetesService(
		&synccontext.SyncContext{
			Context:                ctx.Context,
			Log:                    loghelper.New("sync-kubernetes-service"),
			PhysicalClient:         ctx.LocalManager.GetClient(),
			VirtualClient:          ctx.VirtualManager.GetClient(),
			CurrentNamespace:       ctx.Config.WorkloadNamespace,
			CurrentNamespaceClient: ctx.WorkloadNamespaceClient,
		},
		ctx.Config.WorkloadNamespace,
		ctx.Config.WorkloadService,
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

func StartManagers(controllerContext *config.ControllerContext, syncers []syncertypes.Object) error {
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

	return nil
}

func WriteKubeConfigToSecret(ctx context.Context, currentNamespace string, currentNamespaceClient client.Client, options *config.VirtualClusterConfig, syncerConfig *clientcmdapi.Config) error {
	syncerConfig, err := CreateVClusterKubeConfig(syncerConfig, options)
	if err != nil {
		return err
	}

	if options.ExportKubeConfig.Context != "" {
		syncerConfig.CurrentContext = options.ExportKubeConfig.Context
		// update authInfo
		for k := range syncerConfig.AuthInfos {
			syncerConfig.AuthInfos[syncerConfig.CurrentContext] = syncerConfig.AuthInfos[k]
			if k != syncerConfig.CurrentContext {
				delete(syncerConfig.AuthInfos, k)
			}
			break
		}

		// update cluster
		for k := range syncerConfig.Clusters {
			syncerConfig.Clusters[syncerConfig.CurrentContext] = syncerConfig.Clusters[k]
			if k != syncerConfig.CurrentContext {
				delete(syncerConfig.Clusters, k)
			}
			break
		}

		// update context
		for k := range syncerConfig.Contexts {
			tmpCtx := syncerConfig.Contexts[k]
			tmpCtx.Cluster = syncerConfig.CurrentContext
			tmpCtx.AuthInfo = syncerConfig.CurrentContext
			syncerConfig.Contexts[syncerConfig.CurrentContext] = tmpCtx
			if k != syncerConfig.CurrentContext {
				delete(syncerConfig.Contexts, k)
			}
			break
		}
	}

	// check if we need to write the kubeconfig secrete to the default location as well
	if options.ExportKubeConfig.Secret.Name != "" {
		// which namespace should we create the additional secret in?
		secretNamespace := options.ExportKubeConfig.Secret.Namespace
		if secretNamespace == "" {
			secretNamespace = currentNamespace
		}

		// write the extra secret
		err = kubeconfig.WriteKubeConfig(ctx, currentNamespaceClient, options.ExportKubeConfig.Secret.Name, secretNamespace, syncerConfig, options.Experimental.IsolatedControlPlane.KubeConfig != "")
		if err != nil {
			return fmt.Errorf("creating %s secret in the %s ns failed: %w", options.ExportKubeConfig.Secret.Name, secretNamespace, err)
		}
	}

	// write the default Secret
	return kubeconfig.WriteKubeConfig(ctx, currentNamespaceClient, kubeconfig.GetDefaultSecretName(translate.VClusterName), currentNamespace, syncerConfig, options.Experimental.IsolatedControlPlane.KubeConfig != "")
}
