package setup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	gotemplate "text/template"

	"github.com/loft-sh/e2e-framework/pkg/setup"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	. "github.com/onsi/ginkgo/v2"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/e2e-framework/klient"
	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/third_party/vcluster"
)

type key int

const (
	vclusterNameContextKey key = iota
	vclusterClusterContextKey
	vclusterRestConfigContextKey
	vclusterKlientClientContextKey
	vclusterKubeClientContextKey
)

type vclusterContextKey string

func With(ctx context.Context, name string, vclusterCluster *vcluster.Cluster) context.Context {
	return context.WithValue(ctx, vclusterContextKey(name), vclusterCluster)
}

func From(ctx context.Context, name string) *vcluster.Cluster {
	if k, ok := ctx.Value(vclusterContextKey(name)).(*vcluster.Cluster); ok {
		return k
	}
	return nil
}

func WithVClusterName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, vclusterNameContextKey, name)
}

func VClusterNameFrom(ctx context.Context) string {
	if value := ctx.Value(vclusterNameContextKey); value != nil {
		return value.(string)
	}
	return ""
}

func WithVClusterCluster(ctx context.Context, vclusterCluster *vcluster.Cluster) context.Context {
	return context.WithValue(ctx, vclusterClusterContextKey, vclusterCluster)
}

func VClusterClusterFrom(ctx context.Context) *vcluster.Cluster {
	if value := ctx.Value(vclusterClusterContextKey); value != nil {
		return value.(*vcluster.Cluster)
	}
	return nil
}

func WithVClusterRestConfig(ctx context.Context, restConfig *rest.Config) context.Context {
	return context.WithValue(ctx, vclusterRestConfigContextKey, restConfig)
}

func VClusterRestConfigFrom(ctx context.Context) *rest.Config {
	if value := ctx.Value(vclusterRestConfigContextKey); value != nil {
		return value.(*rest.Config)
	}
	return nil
}

func WithVClusterKlientClient(ctx context.Context, klientClient klient.Client) context.Context {
	return context.WithValue(ctx, vclusterKlientClientContextKey, klientClient)
}

func VClusterKlientClientFrom(ctx context.Context) klient.Client {
	if value := ctx.Value(vclusterKlientClientContextKey); value != nil {
		return value.(klient.Client)
	}
	return nil
}

func WithVClusterKubeClient(ctx context.Context, kubeClient kubernetes.Interface) context.Context {
	return context.WithValue(ctx, vclusterKubeClientContextKey, kubeClient)
}

func GetKubeClientFrom(ctx context.Context) kubernetes.Interface {
	if value := ctx.Value(vclusterKubeClientContextKey); value != nil {
		return value.(kubernetes.Interface)
	}
	return nil
}

type vclusterOptions struct {
	name       string
	valuesYAML string
	valuesFile string
}

type VClusterOptions func(*vclusterOptions)

func WithName(name string) VClusterOptions {
	return func(o *vclusterOptions) {
		o.name = name
	}
}

func WithValuesYAML(valuesYAML string) VClusterOptions {
	return func(o *vclusterOptions) {
		o.valuesYAML = valuesYAML
	}
}

func WithValuesFile(valuesFile string) VClusterOptions {
	return func(o *vclusterOptions) {
		o.valuesFile = valuesFile
	}
}

func Create(opts ...VClusterOptions) setup.Func {
	return func(ctx context.Context) (context.Context, error) {
		o := &vclusterOptions{}
		for _, opt := range opts {
			opt(o)
		}

		if o.name == "" {
			return ctx, fmt.Errorf("vcluster name is required")
		}

		// Check if vcluster already exists in context
		existingVCluster := From(ctx, o.name)
		if existingVCluster != nil {
			return ctx, nil
		}

		var valuesFile string
		var err error

		if o.valuesYAML != "" {
			valuesFile, err = createTempValuesFile(o.name, o.valuesYAML)
			if err != nil {
				return ctx, fmt.Errorf("failed to create temp values file: %w", err)
			}

			DeferCleanup(func(ctx context.Context) {
				_ = os.Remove(valuesFile)
			})
		} else if o.valuesFile != "" {
			valuesFile = o.valuesFile
		} else {
			return ctx, fmt.Errorf("either valuesYAML or valuesFile must be provided")
		}
		// Create vcluster instance
		vclusterCluster := vcluster.NewCluster(o.name)
		// Create the vcluster with the values file
		_, err = vclusterCluster.CreateWithConfig(ctx, valuesFile)
		if err != nil {
			return ctx, fmt.Errorf("failed to create vcluster: %w", err)
		}
		// Get the kubeconfig and create clients
		restConfig, err := setupRestConfig(vclusterCluster)
		if err != nil {
			return ctx, fmt.Errorf("failed to get rest config: %w", err)
		}
		klientClient, err := setupKlientClient(restConfig)
		if err != nil {
			return ctx, fmt.Errorf("failed to get klient client: %w", err)
		}
		kubeClient, err := setupKubeClient(restConfig)
		if err != nil {
			return ctx, fmt.Errorf("failed to get kubernetes client: %w", err)
		}
		// Store vcluster data in context
		ctx = With(ctx, o.name, vclusterCluster)
		ctx = WithVClusterName(ctx, o.name)
		ctx = WithVClusterCluster(ctx, vclusterCluster)
		ctx = WithVClusterRestConfig(ctx, restConfig)
		ctx = WithVClusterKlientClient(ctx, klientClient)
		ctx = WithVClusterKubeClient(ctx, kubeClient)

		return ctx, nil
	}
}

func Destroy(vclusterName string) setup.Func {
	return func(ctx context.Context) (context.Context, error) {
		if vclusterName == "" {
			return ctx, fmt.Errorf("vcluster name is required")
		}

		vclusterCluster := From(ctx, vclusterName)
		if vclusterCluster == nil {
			return ctx, fmt.Errorf("vcluster %s not found in context", vclusterName)
		}

		if err := vclusterCluster.Destroy(ctx); err != nil {
			return ctx, fmt.Errorf("failed to destroy vcluster: %w", err)
		}

		return ctx, nil
	}
}

func WaitForControlPlane(ctx context.Context) error {
	vclusterCluster := VClusterClusterFrom(ctx)
	if vclusterCluster == nil {
		return fmt.Errorf("vcluster cluster not found in context")
	}

	klientClient := VClusterKlientClientFrom(ctx)
	if klientClient == nil {
		return fmt.Errorf("vcluster klient client not found in context")
	}

	return vclusterCluster.WaitForControlPlane(ctx, klientClient)
}

func createTempValuesFile(vclusterName string, valuesYAML string) (string, error) {
	tmpDir := os.TempDir()
	valuesFile := filepath.Join(tmpDir, fmt.Sprintf("vcluster-values-%s.yaml", vclusterName))

	data := map[string]string{
		"Repository": constants.GetRepository(),
		"Tag":        constants.GetTag(),
	}

	tmpl, err := gotemplate.New("values").Parse(valuesYAML)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	f, err := os.Create(valuesFile)
	if err != nil {
		return "", fmt.Errorf("failed to create values file: %w", err)
	}
	defer f.Close()

	err = tmpl.Execute(f, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return valuesFile, nil
}

func setupRestConfig(vclusterCluster *vcluster.Cluster) (*rest.Config, error) {
	kubeconfigPath := vclusterCluster.GetKubeconfig()
	if kubeconfigPath == "" {
		return nil, fmt.Errorf("vcluster kubeconfig path should not be empty")
	}

	restConfig, err := conf.New(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create vcluster rest config: %w", err)
	}
	if restConfig == nil {
		return nil, fmt.Errorf("vcluster config should not be nil")
	}

	return restConfig, nil
}

func setupKlientClient(restConfig *rest.Config) (klient.Client, error) {
	klientClient, err := klient.New(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create klient client: %w", err)
	}
	if klientClient == nil {
		return nil, fmt.Errorf("klient client should not be nil")
	}

	return klientClient, nil
}

func setupKubeClient(restConfig *rest.Config) (kubernetes.Interface, error) {
	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}
	if kubeClient == nil {
		return nil, fmt.Errorf("kubernetes client should not be nil")
	}

	return kubeClient, nil
}
