package setup

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers"
	"github.com/loft-sh/vcluster/pkg/controllers/deploy"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/services"
	"github.com/loft-sh/vcluster/pkg/coredns"
	"github.com/loft-sh/vcluster/pkg/k8s"
	"github.com/loft-sh/vcluster/pkg/log"
	"github.com/loft-sh/vcluster/pkg/plugin"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/specialservices"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/kubeconfig"
	"github.com/loft-sh/vcluster/pkg/util/serviceaccount"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/mitchellh/go-homedir"
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
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func StartControllers(controllerContext *synccontext.ControllerContext, syncers []syncertypes.Object) error {
	// exchange control plane client
	controlPlaneClient := controllerContext.HostNamespaceClient

	// migrate k3s to k8s if needed
	err := k8s.MigrateK3sToK8sStateless(controllerContext.Context, controllerContext.Config.HostClient, controllerContext.Config.HostNamespace, controllerContext.VirtualManager.GetClient(), controllerContext.Config)
	if err != nil {
		return err
	}

	// register init manifests configmap watcher controller
	err = deploy.RegisterInitManifestsController(controllerContext)
	if err != nil {
		return err
	}

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

	// migrate mappers
	if !controllerContext.Config.PrivateNodes.Enabled {
		err = MigrateMappers(controllerContext.ToRegisterContext(), syncers)
		if err != nil {
			return err
		}
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
			err := WriteKubeConfigToSecret(ctx, controllerContext.VirtualManager.GetConfig(), controllerContext.Config.HostNamespace, controlPlaneClient, controllerContext.Config, controllerContext.VirtualRawConfig)
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

	if !controllerContext.Config.PrivateNodes.Enabled {
		// start mappings store garbage collection
		controllerContext.Mappings.Store().StartGarbageCollection(controllerContext.Context)
	}

	// When the user disables from host syncing for some kind, the previously synced resources will
	// stay in the virtual cluster. Since the controllers for those resources do not exist anymore,
	// here we delete those stale virtual resources that were synced from host but should not be
	// synced anymore.
	err = deletePreviouslySyncedResources(controllerContext)
	if err != nil {
		return fmt.Errorf("failed to delete previouly synced resources: %w", err)
	}

	// ensure kubeadm setup
	err = pro.StartPrivateNodesMode(controllerContext)
	if err != nil {
		return fmt.Errorf("ensure kubeadm setup: %w", err)
	}

	// we are done here
	klog.FromContext(controllerContext).Info("Successfully started vCluster controllers")
	return nil
}

func ApplyCoreDNS(controllerContext *synccontext.ControllerContext) {
	_ = wait.ExponentialBackoffWithContext(controllerContext.Context, wait.Backoff{Duration: time.Second, Factor: 1.5, Cap: time.Minute, Steps: math.MaxInt32}, func(ctx context.Context) (bool, error) {
		dnsDeployment := &appsv1.Deployment{}
		err := controllerContext.VirtualManager.GetClient().Get(controllerContext.Context, types.NamespacedName{Namespace: "kube-system", Name: "coredns"}, dnsDeployment)
		if err != nil && !kerrors.IsNotFound(err) {
			return false, err
		}
		if err == nil {
			// dns pod labels were changed to avoid conflict with apps running in the host cluster that select for the "kube-dns" label, e.g. cilium.
			// If the deployment already exists with a label selector that is not "vcluster-kube-dns" then it needs to be deleted because the selector field is immutable.
			// Otherwise, dns will break because the dns service will target the updated label but not match any deployments.
			if dnsDeployment.Spec.Selector.MatchLabels[constants.CoreDNSLabelKey] != constants.CoreDNSLabelValue {
				err = controllerContext.VirtualManager.GetClient().Delete(controllerContext.Context, dnsDeployment)
				if err != nil && !kerrors.IsNotFound(err) {
					return false, err
				}
			}
		}

		// apply coredns manifests
		err = coredns.ApplyManifest(ctx, &controllerContext.Config.Config, controllerContext.Config.ControlPlane.Advanced.DefaultImageRegistry, controllerContext.VirtualManager.GetConfig(), controllerContext.VirtualClusterVersion)
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
	// don't sync kubernetes service in dedicated mode
	var err error
	if ctx.Config.PrivateNodes.Enabled {
		err = pro.SyncKubernetesServiceDedicated(ctx.ToRegisterContext().ToSyncContext("sync-kubernetes-service"))
	} else {
		err = specialservices.SyncKubernetesService(
			ctx.ToRegisterContext().ToSyncContext("sync-kubernetes-service"),
			ctx.Config.HostNamespace,
			ctx.Config.Name,
			types.NamespacedName{
				Name:      specialservices.DefaultKubernetesSVCName,
				Namespace: specialservices.DefaultKubernetesSVCNamespace,
			},
			services.TranslateServicePorts)
	}
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

func CreateVClusterKubeConfigForExport(ctx context.Context, virtualConfig *rest.Config, syncerConfig *clientcmdapi.Config, options CreateKubeConfigOptions) (*clientcmdapi.Config, error) {
	syncerConfigToExport, err := CreateVClusterKubeConfig(syncerConfig, options)
	if err != nil {
		return nil, err
	}

	// should use special server?
	if options.ExportKubeConfig.Server != "" {
		// exchange kube config server & resolve certificate
		for key, cluster := range syncerConfigToExport.Clusters {
			if cluster == nil {
				continue
			}

			syncerConfigToExport.Clusters[key] = &clientcmdapi.Cluster{
				Server:                   options.ExportKubeConfig.Server,
				Extensions:               make(map[string]runtime.Object),
				CertificateAuthorityData: cluster.CertificateAuthorityData,
			}
		}
	}

	// is insecure?
	if options.ExportKubeConfig.Insecure {
		// set insecure skip tls verify and remove certificate authority data
		for key, cluster := range syncerConfigToExport.Clusters {
			if cluster == nil {
				continue
			}

			syncerConfigToExport.Clusters[key].InsecureSkipTLSVerify = true
			syncerConfigToExport.Clusters[key].CertificateAuthorityData = nil
		}
	}

	// should use service account token for secret?
	if options.ExportKubeConfig.ServiceAccount.Name != "" {
		serviceAccountNamespace := options.ExportKubeConfig.ServiceAccount.Namespace
		if serviceAccountNamespace == "" {
			serviceAccountNamespace = "kube-system"
		}

		kubeClient, err := kubernetes.NewForConfig(virtualConfig)
		if err != nil {
			return nil, fmt.Errorf("create kube client: %w", err)
		}

		token, err := serviceaccount.CreateServiceAccountToken(ctx, kubeClient, options.ExportKubeConfig.ServiceAccount.Name, serviceAccountNamespace, options.ExportKubeConfig.ServiceAccount.ClusterRole, 0, log.NewFromExisting(klog.FromContext(ctx), "write-kube-context"))
		if err != nil {
			return nil, fmt.Errorf("create service account token for export kube config: %w", err)
		}

		for k := range syncerConfigToExport.AuthInfos {
			syncerConfigToExport.AuthInfos[k] = &clientcmdapi.AuthInfo{
				Token:                token,
				Extensions:           make(map[string]runtime.Object),
				ImpersonateUserExtra: make(map[string][]string),
			}
		}
	}

	return syncerConfigToExport, nil
}

func WriteKubeConfigToSecret(ctx context.Context, virtualConfig *rest.Config, currentNamespace string, currentNamespaceClient client.Client, options *config.VirtualClusterConfig, syncerConfig *clientcmdapi.Config) error {
	// Write the default kubeconfig secret.
	createKubeConfigOptions := CreateKubeConfigOptions{
		ControlPlaneProxy: options.ControlPlane.Proxy,
		ExportKubeConfig:  options.ExportKubeConfig.ExportKubeConfigProperties,
	}
	defaultKubeConfig, err := CreateVClusterKubeConfigForExport(ctx, virtualConfig, syncerConfig.DeepCopy(), createKubeConfigOptions)
	if err != nil {
		return fmt.Errorf("failed to create kubeconfig that is exported to the default kubeconfig secret: %w", err)
	}

	// if standalone mode is enabled, we don't need to write any kubeconfig secrets and instead write it to a file
	if options.ControlPlane.Standalone.Enabled {
		klog.FromContext(ctx).Info("Writing kubeconfig to", "path", filepath.Join(constants.DataDir, "kubeconfig.yaml"))
		err = clientcmd.WriteToFile(*defaultKubeConfig, filepath.Join(constants.DataDir, "kubeconfig.yaml"))
		if err != nil {
			return fmt.Errorf("failed to write kubeconfig to file: %w", err)
		}

		// also check if we can write it to ~/.kube/config
		home, err := homedir.Dir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		homeKubeConfig := filepath.Join(home, ".kube", "config")
		_, err = os.Stat(homeKubeConfig)
		if err == nil {
			// does exist so we skip writing it to the home kubeconfig
			return nil
		}

		err = clientcmd.WriteToFile(*defaultKubeConfig, homeKubeConfig)
		if err != nil {
			return fmt.Errorf("failed to write kubeconfig to file: %w", err)
		}

		return nil
	}

	err = kubeconfig.WriteKubeConfig(ctx, currentNamespaceClient, kubeconfig.GetDefaultSecretName(translate.VClusterName), currentNamespace, defaultKubeConfig, options.Name)
	if err != nil {
		return fmt.Errorf("creating the default kubeconfig secret in the %s ns failed: %w", currentNamespace, err)
	}

	// Write the additional kubeconfig secrets. Here we get the additional secrets with the GetAdditionalSecrets() func
	// which will return either the deprecated ExportKubeConfig.Secret config or the new ExportKubeConfig.AdditionalSecrets
	// config.
	for _, additionalSecret := range options.ExportKubeConfig.GetAdditionalSecrets() {
		createKubeConfigOptions = CreateKubeConfigOptions{
			ControlPlaneProxy: options.ControlPlane.Proxy,
			ExportKubeConfig:  additionalSecret.ExportKubeConfigProperties,
		}
		additionalKubeConfig, err := CreateVClusterKubeConfigForExport(ctx, virtualConfig, syncerConfig.DeepCopy(), createKubeConfigOptions)
		if err != nil {
			return fmt.Errorf("failed to create kubeconfig that is exported to the additional kubeconfig secret: %w", err)
		}

		// if the additional secret name is not specified, fallback to the default secret name
		secretName := additionalSecret.Name
		if secretName == "" {
			secretName = kubeconfig.GetDefaultSecretName(translate.VClusterName)
		}
		// if the additional secret namespace is not specified, fallback to the current namespace
		secretNamespace := additionalSecret.Namespace
		if secretNamespace == "" {
			secretNamespace = currentNamespace
		}

		// write the additional kubeconfig secret
		err = kubeconfig.WriteKubeConfig(ctx, currentNamespaceClient, secretName, secretNamespace, additionalKubeConfig, options.Name)
		if err != nil {
			return fmt.Errorf("creating additional secret %s in the %s ns failed: %w", secretName, secretNamespace, err)
		}
	}

	return nil
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
