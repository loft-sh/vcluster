package cluster

import (
	"context"
	"fmt"
	"slices"

	"github.com/loft-sh/e2e-framework/pkg/setup"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientpkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/pkg/types"
	"sigs.k8s.io/e2e-framework/support"
)

type key int

const (
	currentClusterContextKey key = iota
	currentClusterNameContextKey
	currentClusterClientContextKey
	currentKubeClientContextKey
	listContextKey
)

func With(ctx context.Context, name string, k support.E2EClusterProvider) context.Context {
	return context.WithValue(ctx, support.ClusterNameContextKey(name), k)
}

func From(ctx context.Context, name string) support.E2EClusterProvider {
	if k, ok := envfuncs.GetClusterFromContext(ctx, name); ok {
		return k
	}
	return nil
}

func List(ctx context.Context) []string {
	if l := ctx.Value(listContextKey); l != nil {
		return l.([]string)
	}

	return nil
}

func Add(ctx context.Context, name string) context.Context {
	list := List(ctx)
	list = append(list, name)
	slices.Sort(list)
	return context.WithValue(ctx, listContextKey, slices.Compact(list))
}

func Remove(ctx context.Context, name string) context.Context {
	var newList []string
	list := List(ctx)
	for _, item := range list {
		if item == name {
			continue
		}

		newList = append(newList, item)
	}
	return context.WithValue(ctx, listContextKey, list)
}

func WithCurrentClusterName(ctx context.Context, clusterName string) context.Context {
	return context.WithValue(ctx, currentClusterNameContextKey, clusterName)
}

func CurrentClusterNameFrom(ctx context.Context) string {
	if value := ctx.Value(currentClusterNameContextKey); value != nil {
		return value.(string)
	}
	return ""
}

func WithCurrentCluster(ctx context.Context, cluster types.E2EClusterProvider) context.Context {
	return context.WithValue(ctx, currentClusterContextKey, cluster)
}

func CurrentClusterFrom(ctx context.Context) types.E2EClusterProvider {
	if value := ctx.Value(currentClusterContextKey); value != nil {
		return value.(types.E2EClusterProvider)
	}
	return nil
}

func WithCurrentClusterClient(ctx context.Context, client clientpkg.Client) context.Context {
	return context.WithValue(ctx, currentClusterClientContextKey, client)
}

func CurrentClusterClientFrom(ctx context.Context) clientpkg.Client {
	if value := ctx.Value(currentClusterClientContextKey); value != nil {
		return value.(clientpkg.Client)
	}
	return nil
}

func WithCurrentKubeClient(ctx context.Context, client kubernetes.Interface) context.Context {
	return context.WithValue(ctx, currentKubeClientContextKey, client)
}

func CurrentKubeClientFrom(ctx context.Context) kubernetes.Interface {
	if value := ctx.Value(currentKubeClientContextKey); value != nil {
		return value.(kubernetes.Interface)
	}
	return nil
}

func SetCurrentCluster(clusterName string) setup.Func {
	return func(ctx context.Context) (context.Context, error) {
		clusterVal := From(ctx, clusterName)
		if clusterVal == nil {
			return ctx, fmt.Errorf("cluster not found in context")
		}

		client := ControllerRuntimeClientFrom(ctx, clusterName)
		if client == nil {
			return ctx, fmt.Errorf("cluster client not found in context")
		}

		kubeClient := KubeClientFrom(ctx, clusterName)
		if kubeClient == nil {
			return ctx, fmt.Errorf("cluster kube.Interface client not found in context")
		}

		ctx = WithCurrentClusterName(ctx, clusterName)
		ctx = WithCurrentCluster(ctx, clusterVal)
		ctx = WithCurrentClusterClient(ctx, client)
		ctx = WithCurrentKubeClient(ctx, kubeClient)
		return ctx, nil
	}
}

type clientContextKey string

func WithControllerRuntimeClient(ctx context.Context, cluster string, client clientpkg.Client) context.Context {
	return context.WithValue(ctx, clientContextKey(cluster), client)
}

func ControllerRuntimeClientFrom(ctx context.Context, cluster string) clientpkg.Client {
	if value := ctx.Value(clientContextKey(cluster)); value != nil {
		return value.(clientpkg.Client)
	}
	return nil
}

type controllerRuntimeClientOptions struct {
	clientpkg.Options
	clusterName string
}
type ClientOptions func(t *controllerRuntimeClientOptions)

func WithCluster(clusterName string) ClientOptions {
	return func(t *controllerRuntimeClientOptions) {
		t.clusterName = clusterName
	}
}

func WithScheme(scheme *runtime.Scheme) ClientOptions {
	return func(t *controllerRuntimeClientOptions) {
		t.Scheme = scheme
	}
}

func SetupControllerRuntimeClient(opts ...ClientOptions) setup.Func {
	return func(ctx context.Context) (context.Context, error) {
		o := &controllerRuntimeClientOptions{}
		for _, opt := range opts {
			opt(o)
		}

		cluster := From(ctx, o.clusterName)
		if cluster == nil {
			return ctx, fmt.Errorf("cluster %s not found", o.clusterName)
		}

		client, err := clientpkg.New(cluster.KubernetesRestConfig(), o.Options)
		if err != nil {
			return ctx, err
		}

		return WithControllerRuntimeClient(ctx, o.clusterName, client), nil
	}
}

type kubeClientContextKey string

func WithKubeClient(ctx context.Context, cluster string, client kubernetes.Interface) context.Context {
	return context.WithValue(ctx, kubeClientContextKey(cluster), client)
}

func KubeClientFrom(ctx context.Context, cluster string) kubernetes.Interface {
	if value := ctx.Value(kubeClientContextKey(cluster)); value != nil {
		return value.(kubernetes.Interface)
	}
	return nil
}

func SetupKubeClient(clusterName string) setup.Func {
	return func(ctx context.Context) (context.Context, error) {
		cluster := From(ctx, clusterName)
		if cluster == nil {
			return ctx, fmt.Errorf("cluster %s not found", clusterName)
		}

		clientSet, err := kubernetes.NewForConfig(cluster.KubernetesRestConfig())
		if err != nil {
			return ctx, err
		}

		return WithKubeClient(ctx, clusterName, clientSet), nil
	}
}

type cluster struct {
	name       string
	provider   support.E2EClusterProvider
	configFile string
	envCfg     *envconf.Config
	opts       []support.ClusterOpts
}

type Options func(c *cluster)

func WithProvider(p support.E2EClusterProvider) Options {
	return func(c *cluster) {
		c.provider = p
	}
}

func WithName(name string) Options {
	return func(c *cluster) {
		c.name = name
	}
}

func WithConfigFile(path string) Options {
	return func(c *cluster) {
		c.configFile = path
	}
}

func WithEnvConfig(envCfg *envconf.Config) Options {
	return func(c *cluster) {
		c.envCfg = envCfg
	}
}

func WithOptions(opts ...support.ClusterOpts) Options {
	return func(c *cluster) {
		c.opts = opts
	}
}

func Create(options ...Options) setup.Func {
	return func(ctx context.Context) (context.Context, error) {
		c := &cluster{}
		for _, o := range options {
			o(c)
		}

		if c.envCfg == nil {
			c.envCfg = envconf.New()
		}

		if c.configFile == "" {
			// Likely an existing cluster
			var err error
			ctx, err = envfuncs.CreateClusterWithOpts(c.provider, c.name, c.opts...)(ctx, c.envCfg)
			if err != nil {
				return ctx, err
			}
		} else {
			var err error
			ctx, err = envfuncs.CreateClusterWithConfig(c.provider, c.name, c.configFile, c.opts...)(ctx, c.envCfg)
			if err != nil {
				return ctx, err
			}
		}

		return Add(ctx, c.name), nil
	}
}

func Destroy(clusterName string) setup.Func {
	return func(ctx context.Context) (context.Context, error) {
		if clusterName == "" {
			return ctx, fmt.Errorf("cluster name is required")
		}

		var err error
		ctx, err = envfuncs.DestroyCluster(clusterName)(ctx, nil)
		if err != nil {
			return ctx, err
		}

		return Remove(ctx, clusterName), nil
	}
}

func DestroyAll() setup.Func {
	return func(ctx context.Context) (context.Context, error) {
		clusters := List(ctx)
		for _, c := range clusters {
			var err error
			ctx, err = Destroy(c)(ctx)
			if err != nil {
				return ctx, err
			}

			ctx = Remove(ctx, c)
		}
		return ctx, nil
	}
}

func LoadImage(name, image string, args ...string) setup.Func {
	return func(ctx context.Context) (context.Context, error) {
		if hasImage(ctx, name, image) {
			return ctx, nil
		}

		clusterVal := ctx.Value(support.ClusterNameContextKey(name))
		if clusterVal == nil {
			return ctx, fmt.Errorf("load image func: context cluster is nil")
		}

		cluster, ok := clusterVal.(support.E2EClusterProviderWithImageLoader)
		if !ok {
			return ctx, fmt.Errorf("load image archive func: cluster provider does not support LoadImage helper")
		}

		if err := cluster.LoadImage(ctx, image, args...); err != nil {
			return ctx, fmt.Errorf("load image: %w", err)
		}

		return ctx, nil
	}
}

func hasImage(ctx context.Context, clusterName, image string) bool {
	client := ControllerRuntimeClientFrom(ctx, clusterName)
	if client == nil {
		return false
	}

	nodeList := &corev1.NodeList{}
	if err := client.List(ctx, nodeList); err != nil {
		return false
	}

	found := map[string]bool{}
	for _, node := range nodeList.Items {
		found[node.Name] = false
	imageLoop:
		for _, images := range node.Status.Images {
			if slices.Contains(images.Names, image) {
				found[node.Name] = true
				break imageLoop
			}
		}
	}

	for _, v := range found {
		if !v {
			return false
		}
	}

	return true
}
