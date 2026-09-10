package filters

import (
	"net/http"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var WithPodSchedulerCheck = func(h http.Handler, _ *synccontext.RegisterContext, _ client.Client) http.Handler {
	return h
}
