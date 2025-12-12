package cluster

import (
	"context"
	"fmt"

	"github.com/loft-sh/e2e-framework/pkg/setup"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientpkg "sigs.k8s.io/controller-runtime/pkg/client"
)

type kubeClientContextKey string

func WithKubeClient(ctx context.Context, cluster string, client kubernetes.Interface) context.Context {
	return context.WithValue(ctx, kubeClientContextKey(cluster), client)
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

func KubeClientFrom(ctx context.Context, clusterName string) kubernetes.Interface {
	if value := ctx.Value(kubeClientContextKey(clusterName)); value != nil {
		return value.(kubernetes.Interface)
	}

	return nil
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
