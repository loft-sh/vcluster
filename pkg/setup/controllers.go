package setup

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/controllers"
	"github.com/loft-sh/vcluster/pkg/controllers/deploy"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/services"
	"github.com/loft-sh/vcluster/pkg/coredns"
	"github.com/loft-sh/vcluster/pkg/log"
	"github.com/loft-sh/vcluster/pkg/plugin"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/specialservices"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/kubeconfig"
	"github.com/loft-sh/vcluster/pkg/util/serviceaccount"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func StartControllers(controllerContext *synccontext.ControllerContext, syncers []syncertypes.Object) error {
	// exchange control plane client
	controlPlaneClient, err := pro.ExchangeControlPlaneClient(controllerContext)
	if err != nil {
		return err
	}

	// register init manifests configmap watcher controller
	err = deploy.RegisterInitManifestsController(controllerContext)
	if err != nil {
		return err
	}

	// start coredns & create syncers
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
		// migrate mappers
		err = MigrateMappers(controllerContext.ToRegisterContext(), syncers)
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
		_ = wait.PollUntilContextCancel(controllerContext, time.Second*10, true, func(ctx context.Context) (bool, error) {
			err := WriteKubeConfigToSecret(ctx, controllerContext.VirtualManager.GetConfig(), controllerContext.Config.ControlPlaneNamespace, controlPlaneClient, controllerContext.Config, controllerContext.VirtualRawConfig)
			if err != nil {
				klog.Errorf("Error writing kube config to secret: %v", err)
				return false, nil
			}

			return true, nil
		})
	}()

	// set leader
	err = plugin.DefaultManager.SetLeader(controllerContext.Context)
	if err != nil {
		return fmt.Errorf("plugin set leader: %w", err)
	}

	// start mappings store garbage collection
	controllerContext.Mappings.Store().StartGarbageCollection(controllerContext.Context)

	// we are done here
	klog.FromContext(controllerContext).Info("Successfully started vCluster controllers")
	return nil
}

func ApplyCoreDNS(controllerContext *synccontext.ControllerContext) {
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

func SyncKubernetesService(ctx *synccontext.ControllerContext) error {
	err := specialservices.SyncKubernetesService(
		ctx.ToRegisterContext().ToSyncContext("sync-kubernetes-service"),
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

func WriteKubeConfigToSecret(ctx context.Context, virtualConfig *rest.Config, currentNamespace string, currentNamespaceClient client.Client, options *config.VirtualClusterConfig, syncerConfig *clientcmdapi.Config) error {
	syncerConfig, err := CreateVClusterKubeConfig(syncerConfig, options)
	if err != nil {
		return err
	}

	var customSyncerConfig *clientcmdapi.Config
	if options.ExportKubeConfig.Server != "" {
		// Create a deep copy of syncerConfig to modify the server for the additional secret
		customSyncerConfig = syncerConfig.DeepCopy()
		for key, cluster := range customSyncerConfig.Clusters {
			if cluster != nil {
				customSyncerConfig.Clusters[key] = &clientcmdapi.Cluster{
					Server:                   options.ExportKubeConfig.Server,
					Extensions:               make(map[string]runtime.Object),
					CertificateAuthorityData: cluster.CertificateAuthorityData,
					InsecureSkipTLSVerify:    options.ExportKubeConfig.Insecure,
				}
			}
		}
	}

	// Apply service account token if specified
	if options.ExportKubeConfig.ServiceAccount.Name != "" {
		serviceAccountNamespace := options.ExportKubeConfig.ServiceAccount.Namespace
		if serviceAccountNamespace == "" {
			serviceAccountNamespace = "kube-system"
		}

		kubeClient, err := kubernetes.NewForConfig(virtualConfig)
		if err != nil {
			return fmt.Errorf("create kube client: %w", err)
		}

		token, err := serviceaccount.CreateServiceAccountToken(ctx, kubeClient, options.ExportKubeConfig.ServiceAccount.Name, serviceAccountNamespace, options.ExportKubeConfig.ServiceAccount.ClusterRole, 0, log.NewFromExisting(klog.FromContext(ctx), "write-kube-context"))
		if err != nil {
			return fmt.Errorf("create service account token for export kube config: %w", err)
		}

		// Apply the token to both syncerConfig and customSyncerConfig (if it exists)
		applyAuthToken(syncerConfig, token)
		if customSyncerConfig != nil {
			applyAuthToken(customSyncerConfig, token)
		}
	}

	// Write the additional secret if specified
	if options.ExportKubeConfig.Secret.Name != "" {
		secretNamespace := options.ExportKubeConfig.Secret.Namespace
		if secretNamespace == "" {
			secretNamespace = currentNamespace
		}

		// Use customSyncerConfig for the additional secret if it was modified, else syncerConfig
		err = kubeconfig.WriteKubeConfig(ctx, currentNamespaceClient, options.ExportKubeConfig.Secret.Name, secretNamespace, customSyncerConfig, options.Experimental.IsolatedControlPlane.KubeConfig != "", options.Name)
		if err != nil {
			return fmt.Errorf("creating %s secret in the %s ns failed: %w", options.ExportKubeConfig.Secret.Name, secretNamespace, err)
		}
	}

	// Write the default secret using syncerConfig, which retains the original Server value
	return kubeconfig.WriteKubeConfig(ctx, currentNamespaceClient, kubeconfig.GetDefaultSecretName(translate.VClusterName), currentNamespace, syncerConfig, options.Experimental.IsolatedControlPlane.KubeConfig != "", options.Name)
}

// applyAuthToken sets the provided token in all AuthInfos of the given config
func applyAuthToken(config *clientcmdapi.Config, token string) {
	for k := range config.AuthInfos {
		config.AuthInfos[k] = &clientcmdapi.AuthInfo{
			Token:                token,
			Extensions:           make(map[string]runtime.Object),
			ImpersonateUserExtra: make(map[string][]string),
		}
	}
}

func MigrateMappers(ctx *synccontext.RegisterContext, syncers []syncertypes.Object) error {
	mappers := ctx.Mappings.List()
	done := map[schema.GroupVersionKind]bool{}

	// migrate mappers
	for _, mapper := range mappers {
		done[mapper.GroupVersionKind()] = true
		err := mapper.Migrate(ctx, mapper)
		if err != nil {
			return fmt.Errorf("migrate mapper %s: %w", mapper.GroupVersionKind().String(), err)
		}
	}

	// migrate syncers
	for _, syncer := range syncers {
		mapper, ok := syncer.(synccontext.Mapper)
		if !ok || done[mapper.GroupVersionKind()] {
			continue
		}

		done[mapper.GroupVersionKind()] = true
		err := mapper.Migrate(ctx, mapper)
		if err != nil {
			return fmt.Errorf("migrate syncer mapper %s: %w", mapper.GroupVersionKind().String(), err)
		}
	}

	return nil
}
