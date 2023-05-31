package filters

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/loft-sh/vcluster/pkg/authorization/delegatingauthorizer"
	"github.com/loft-sh/vcluster/pkg/server/handler"
	requestpkg "github.com/loft-sh/vcluster/pkg/util/request"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func WithRedirect(h http.Handler, localConfig *rest.Config, localScheme *runtime.Scheme, uncachedVirtualClient client.Client, admit admission.Interface, resources []delegatingauthorizer.GroupVersionResourceVerb) http.Handler {
	s := serializer.NewCodecFactory(localScheme)
	parameterCodec := runtime.NewParameterCodec(uncachedVirtualClient.Scheme())
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		info, ok := request.RequestInfoFrom(req.Context())
		if !ok {
			requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, fmt.Errorf("request info is missing"))
			return
		}

		if applies(info, resources) {
			// call admission webhooks
			err := callAdmissionWebhooks(req, info, parameterCodec, admit, uncachedVirtualClient)
			if err != nil {
				responsewriters.ErrorNegotiated(err, s, corev1.SchemeGroupVersion, w, req)
				return
			}

			// we have to change the request url
			if info.Resource != "nodes" {
				if info.Namespace == "" {
					responsewriters.ErrorNegotiated(kerrors.NewBadRequest("namespace required"), s, corev1.SchemeGroupVersion, w, req)
					return
				}

				splitted := strings.Split(req.URL.Path, "/")
				if len(splitted) < 6 {
					responsewriters.ErrorNegotiated(kerrors.NewBadRequest("unexpected url"), s, corev1.SchemeGroupVersion, w, req)
					return
				}

				// exchange namespace & name
				splitted[4] = translate.Default.PhysicalNamespace(info.Namespace)
				splitted[6] = translate.Default.PhysicalName(splitted[6], info.Namespace)
				req.URL.Path = strings.Join(splitted, "/")

				// we have to add a trailing slash here, because otherwise the
				// host api server would redirect us to a wrong path
				if len(splitted) == 8 {
					req.URL.Path += "/"
				}
			}

			h, err := handler.Handler("", localConfig, nil)
			if err != nil {
				requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, err)
				return
			}

			req.Header.Del("Authorization")
			h.ServeHTTP(w, req)
			return
		}

		h.ServeHTTP(w, req)
	})
}

func callAdmissionWebhooks(req *http.Request, info *request.RequestInfo, parameterCodec runtime.ParameterCodec, admit admission.Interface, uncachedVirtualClient client.Client) error {
	if info.Resource != "pods" {
		return nil
	} else if info.Subresource != "exec" && info.Subresource != "portforward" && info.Subresource != "attach" {
		return nil
	}

	if admit != nil && admit.Handles(admission.Connect) {
		userInfo, _ := request.UserFrom(req.Context())
		if validatingAdmission, ok := admit.(admission.ValidationInterface); ok {
			var opts runtime.Object
			var kind schema.GroupVersionKind
			if info.Subresource == "exec" {
				kind = corev1.SchemeGroupVersion.WithKind("PodExecOptions")
				opts = &corev1.PodExecOptions{}
				if err := parameterCodec.DecodeParameters(req.URL.Query(), corev1.SchemeGroupVersion, opts); err != nil {
					return err
				}
			} else if info.Subresource == "attach" {
				kind = corev1.SchemeGroupVersion.WithKind("PodAttachOptions")
				opts = &corev1.PodAttachOptions{}
				if err := parameterCodec.DecodeParameters(req.URL.Query(), corev1.SchemeGroupVersion, opts); err != nil {
					return err
				}
			} else if info.Subresource == "portforward" {
				kind = corev1.SchemeGroupVersion.WithKind("PodPortForwardOptions")
				opts = &corev1.PodPortForwardOptions{}
				if err := parameterCodec.DecodeParameters(req.URL.Query(), corev1.SchemeGroupVersion, opts); err != nil {
					return err
				}
			}

			err := validatingAdmission.Validate(req.Context(), admission.NewAttributesRecord(opts, nil, kind, info.Namespace, info.Name, corev1.SchemeGroupVersion.WithResource(info.Resource), info.Subresource, admission.Connect, nil, false, userInfo), NewFakeObjectInterfaces(uncachedVirtualClient.Scheme(), uncachedVirtualClient.RESTMapper()))
			if err != nil {
				klog.Infof("Admission validate failed for %s: %v", info.Path, err)
				return err
			}
		}
	}

	return nil
}

func applies(r *request.RequestInfo, resources []delegatingauthorizer.GroupVersionResourceVerb) bool {
	if !r.IsResourceRequest {
		return false
	}

	for _, gv := range resources {
		if (gv.Group == "*" || gv.Group == r.APIGroup) && (gv.Version == "*" || gv.Version == r.APIVersion) && (gv.Resource == "*" || gv.Resource == r.Resource) && (gv.Verb == "*" || gv.Verb == r.Verb) && (gv.SubResource == "*" || gv.SubResource == r.Subresource) {
			return true
		}
	}

	return false
}
