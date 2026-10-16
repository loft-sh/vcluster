package setup

import (
	"context"
	"fmt"
	"time"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/util/kubeconfig"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// exportRenewal schedules from the granted expiry, rather than the requested
// lifetime. A failed issuance or write never postpones another attempt.
type exportRenewal struct {
	now                   func() time.Time
	write                 func(context.Context) (time.Time, error)
	repair                func(context.Context) error
	rotating              bool
	renewBeforePercentage *int32
	written               bool
	next                  time.Time
}

func (r *exportRenewal) reconcile(ctx context.Context) error {
	if r.written && (!r.rotating || r.now().Before(r.next)) {
		if r.repair != nil {
			return r.repair(ctx)
		}
		return nil
	}
	issuedAt := r.now()
	expires, err := r.write(ctx)
	if err != nil {
		return err
	}
	if r.rotating {
		if !expires.After(r.now()) {
			return fmt.Errorf("exported ServiceAccount token has no remaining lifetime")
		}
		// Include issuance and write latency so it cannot shorten the renewal window.
		remaining := expires.Sub(issuedAt)
		renewBefore := remaining / 3
		if r.renewBeforePercentage != nil {
			renewBefore = remaining / 100 * time.Duration(*r.renewBeforePercentage)
		}
		delay := remaining - renewBefore
		// Spread requests by advancing renewal by up to 10%, preserving the retry window.
		r.next = issuedAt.Add(delay - (wait.Jitter(delay, 0.1) - delay))
	}
	r.written = true
	return nil
}

func startKubeConfigExports(ctx context.Context, virtualConfig *rest.Config, currentNamespace string, host client.Client, options *config.VirtualClusterConfig, raw *clientcmdapi.Config) error {
	rotating := options.ExportKubeConfig.ServiceAccount.TokenRotation.IsSet()
	for _, extra := range options.ExportKubeConfig.GetAdditionalSecrets() {
		rotating = rotating || extra.ServiceAccount.TokenRotation.IsSet()
	}
	if !rotating {
		// Retain the existing startup-only behavior for all existing configurations.
		go func() {
			_ = wait.PollUntilContextCancel(ctx, 10*time.Second, true, func(ctx context.Context) (bool, error) {
				if err := WriteKubeConfigToSecret(ctx, virtualConfig, currentNamespace, host, options, raw); err != nil {
					klog.Errorf("Error writing kube config to secret: %v", err)
					return false, nil
				}
				return true, nil
			})
		}()
		return nil
	}
	if options.ControlPlane.Standalone.Enabled {
		return fmt.Errorf("rotating ServiceAccount exports require host Secret destinations; standalone mode is unsupported")
	}

	defaultName := kubeconfig.GetDefaultSecretName(translate.VClusterName)
	targets := []vclusterconfig.ExportKubeConfigAdditionalSecretReference{{
		ExportKubeConfigProperties: options.ExportKubeConfig.ExportKubeConfigProperties,
		Name:                       defaultName, Namespace: currentNamespace,
	}}
	targets = append(targets, options.ExportKubeConfig.GetAdditionalSecrets()...)
	seen := map[string]bool{}
	for i := range targets {
		target := &targets[i]
		if target.Name == "" {
			target.Name = defaultName
		}
		if target.Namespace == "" {
			target.Namespace = currentNamespace
		}
		key := target.Namespace + "/" + target.Name
		if seen[key] {
			return fmt.Errorf("rotating kubeconfig exports require distinct destination Secrets: %s", key)
		}
		seen[key] = true
	}
	hostKubeClient, err := kubernetes.NewForConfig(options.HostConfig)
	if err != nil {
		return fmt.Errorf("create kubeconfig export watch client: %w", err)
	}
	for i, target := range targets {
		// The first target is the admin kubeconfig export which is written to /root/.kube/config for backwards compatibility
		writeToFile := i == 0
		renewal := &exportRenewal{
			now: time.Now, rotating: target.ServiceAccount.TokenRotation.IsSet(),
		}
		if renewal.rotating {
			renewal.renewBeforePercentage = target.ServiceAccount.TokenRotation.RenewBeforePercentage
		}
		var published *clientcmdapi.Config
		renewal.write = func(ctx context.Context) (time.Time, error) {
			cfg, expiry, err := CreateVClusterKubeConfigForExport(ctx, virtualConfig, raw.DeepCopy(), CreateKubeConfigOptions{
				ControlPlaneProxy: options.ControlPlane.Proxy,
				ExportKubeConfig:  target.ExportKubeConfigProperties,
			})
			if err != nil {
				return time.Time{}, err
			}
			err = kubeconfig.WriteKubeConfig(ctx, host, target.Name, target.Namespace, cfg, options.Name, writeToFile)
			if err == nil {
				published = cfg
			}
			return expiry, err
		}
		var changes <-chan struct{}
		if renewal.rotating {
			renewal.repair = func(ctx context.Context) error {
				return kubeconfig.PatchKubeConfigSecret(ctx, host, target.Name, target.Namespace, published)
			}
			changes = watchExportSecret(ctx, hostKubeClient, target.Name, target.Namespace)
		}
		// Each destination retries independently under the existing leader context.
		go func() {
			for ctx.Err() == nil {
				err := renewal.reconcile(ctx)
				if err != nil {
					klog.ErrorS(err, "Could not export kubeconfig", "namespace", target.Namespace, "name", target.Name)
				}
				if renewal.written && !renewal.rotating {
					return
				}
				delay := 10 * time.Second
				if remaining := renewal.next.Sub(renewal.now()); err == nil && remaining > 0 {
					delay = remaining
				}
				timer := time.NewTimer(delay)
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-timer.C:
				case <-changes:
					timer.Stop()
				}
			}
		}()
	}
	return nil
}

func watchExportSecret(ctx context.Context, host kubernetes.Interface, name, namespace string) <-chan struct{} {
	changes := make(chan struct{}, 1)
	notify := func() {
		select {
		case changes <- struct{}{}:
		default:
		}
	}
	// Filter both the initial list and every watch/reconnect on the server, so
	// resourceNames RBAC suffices and no unrelated Secret enters this informer.
	_, informer := cache.NewInformerWithOptions(cache.InformerOptions{
		ListerWatcher: cache.NewListWatchFromClient(host.CoreV1().RESTClient(), "secrets", namespace, fields.OneTermEqualSelector("metadata.name", name)),
		ObjectType:    &corev1.Secret{},
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc:    func(any) { notify() },
			UpdateFunc: func(any, any) { notify() },
			DeleteFunc: func(any) { notify() },
		},
	})
	go informer.RunWithContext(ctx)
	return changes
}
