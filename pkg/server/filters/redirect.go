package filters

import (
	"fmt"
	"github.com/loft-sh/vcluster/pkg/authorization/delegatingauthorizer"
	"github.com/loft-sh/vcluster/pkg/server/handler"
	requestpkg "github.com/loft-sh/vcluster/pkg/util/request"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"k8s.io/apiserver/pkg/endpoints/request"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"strings"
)

func WithRedirect(h http.Handler, localManager ctrl.Manager, targetNamespace string, resources []delegatingauthorizer.GroupVersionResourceVerb) http.Handler {
	s := serializer.NewCodecFactory(localManager.GetScheme())
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		info, ok := request.RequestInfoFrom(req.Context())
		if !ok {
			requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, fmt.Errorf("request info is missing"))
			return
		}

		if applies(info, resources) {
			// authorization was done here already so we will just go forward with the redirect
			req.Header.Del("Authorization")

			// we have to change the request url
			if info.Resource != "nodes" {
				if info.Namespace == "" {
					responsewriters.ErrorNegotiated(kerrors.NewBadRequest("namespace required"), s, corev1.SchemeGroupVersion, w, req)
					return
				}

				targetName := translate.PhysicalName(info.Name, info.Namespace)
				splitted := strings.Split(req.URL.Path, "/")
				if len(splitted) < 6 {
					responsewriters.ErrorNegotiated(kerrors.NewBadRequest("unexpected url"), s, corev1.SchemeGroupVersion, w, req)
					return
				}

				// exchange namespace & name
				splitted[4] = targetNamespace
				splitted[6] = targetName
				req.URL.Path = strings.Join(splitted, "/")
			}

			req.Header.Del("Authorization")
			h, err := handler.Handler("", localManager.GetConfig())
			if err != nil {
				requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, err)
				return
			}

			h.ServeHTTP(w, req)
			return
		}

		h.ServeHTTP(w, req)
	})
}

func applies(r *request.RequestInfo, resources []delegatingauthorizer.GroupVersionResourceVerb) bool {
	if r.IsResourceRequest == false {
		return false
	}

	for _, gv := range resources {
		if (gv.Group == "*" || gv.Group == r.APIGroup) && (gv.Version == "*" || gv.Version == r.APIVersion) && (gv.Resource == "*" || gv.Resource == r.Resource) && (gv.Verb == "*" || gv.Verb == r.Verb) && (gv.SubResource == "*" || gv.SubResource == r.Subresource) {
			return true
		}
	}

	return false
}
