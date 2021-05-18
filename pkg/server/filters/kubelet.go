package filters

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
)

func WithFakeKubelet(h http.Handler, localManager ctrl.Manager, virtualManager ctrl.Manager, targetNamespace string) http.Handler {
	s := serializer.NewCodecFactory(virtualManager.GetScheme())
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
			_, err := handleNodeRequest(localManager.GetConfig(), virtualManager.GetClient(), targetNamespace, w, req)
			if err != nil {
				responsewriters.ErrorNegotiated(err, s, corev1.SchemeGroupVersion, w, req)
				return
			}
			return
		}

		h.ServeHTTP(w, req)
	})
}
