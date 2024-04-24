package pro

import (
	"context"
	"net/http"

	"github.com/loft-sh/vcluster/pkg/config"
)

var ConnectToPlatform = func(context.Context, *config.VirtualClusterConfig, http.RoundTripper) error {
	return nil
}
