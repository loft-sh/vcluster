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
	"github.com/loft-sh/vcluster/pkg/plugin"
	"github.com/loft-sh/vcluster/pkg/setup/options"
	"github.com/loft-sh/vcluster/pkg/specialservices"
	syncertypes "github.com/loft-sh/vcluster/pkg/types"
	"github.com/loft-sh/vcluster/pkg/util/kubeconfig"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func StartControllers(controllerContext *options.ControllerContext) error {
	// setup CoreDNS according to the manifest file
	go ApplyCoreDNS(controllerContext)

	// instantiate controllers
	syncers, err := controllers.Create(controllerContext)
	if err != nil {
		return errors.Wrap(err, "instantiate controllers")
	}

	// start managers
	err = StartManagers(controllerContext, syncers)
	if err != nil {
		return err
	}

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

	// write the kube config to secret
	go func() {
		wait.Until(func() {
			err := WriteKubeConfigToSecret(controllerContext.Context, controllerContext.CurrentNamespace, controllerContext.CurrentNamespaceClient, controllerContext.Options, controllerContext.VirtualRawConfig, false)
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

func SetGlobalOwner(ctx context.Context, currentNamespaceClient client.Client, currentNamespace, targetNamespace string, setOwner bool, serviceName string) error {
	if currentNamespace != targetNamespace {
		if setOwner {
			klog.Warningf("Skip setting owner, because current namespace %s != target namespace %s", currentNamespace, targetNamespace)
		}

		return nil
	}

	if setOwner {
		service := &corev1.Service{}
		err := currentNamespaceClient.Get(ctx, types.NamespacedName{Namespace: currentNamespace, Name: serviceName}, service)
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

	// make sure owner is set if it is there
	err = SetGlobalOwner(
		controllerContext.Context,
		controllerContext.CurrentNamespaceClient,
		controllerContext.CurrentNamespace,
		controllerContext.Options.TargetNamespace,
		controllerContext.Options.SetOwner,
		controllerContext.Options.ServiceName,
	)
	if err != nil {
		return errors.Wrap(err, "finding vcluster pod owner")
	}

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
