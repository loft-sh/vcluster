package config

import (
	"context"
	"net/http"

	"k8s.io/apimachinery/pkg/version"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ControllerContext struct {
	Context context.Context

	LocalManager          ctrl.Manager
	VirtualManager        ctrl.Manager
	VirtualRawConfig      *clientcmdapi.Config
	VirtualClusterVersion *version.Info

	WorkloadNamespaceClient client.Client

	Config   *VirtualClusterConfig
	StopChan <-chan struct{}

	// PreServerHooks are extra filters to inject into the server before everything else
	PreServerHooks []Filter

	// PostServerHooks are extra filters to inject into the server after everything else
	PostServerHooks []Filter

	// AcquiredLeaderHooks are hooks to start after vCluster acquired leader
	AcquiredLeaderHooks []Hook
}

type Filter func(http.Handler, *ControllerContext) http.Handler

type Hook func(ctx *ControllerContext) error
