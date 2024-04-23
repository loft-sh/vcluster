package pro

import (
	"context"
	"net/http"

	"github.com/loft-sh/vcluster/pkg/config"
	"k8s.io/client-go/kubernetes"
)

var ConnectToPlatform = func(context.Context, kubernetes.Interface, http.RoundTripper, *config.VirtualClusterConfig) error {
	return nil
}
