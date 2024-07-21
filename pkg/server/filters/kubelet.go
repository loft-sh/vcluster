package filters

import (
	"net/http"

	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
)

func WithFakeKubelet(h http.Handler, registerCtx *synccontext.RegisterContext) http.Handler {
	s := serializer.NewCodecFactory(scheme.Scheme)
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		nodeName, found := NodeNameFrom(req.Context())
		if found {
			// make sure there is a leading slash
			if req.URL.Path[0] != '/' {
				req.URL.Path = "/" + req.URL.Path
			}

			// construct the actual path
			req.URL.Path = "/api/v1/nodes/" + nodeName + "/proxy" + req.URL.Path

			// execute the request
			_, err := handleNodeRequest(registerCtx, w, req)
			if err != nil {
				responsewriters.ErrorNegotiated(err, s, corev1.SchemeGroupVersion, w, req)
				return
			}

			return
		}

		h.ServeHTTP(w, req)
	})
}
