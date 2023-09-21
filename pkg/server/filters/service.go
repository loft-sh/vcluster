package filters

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/loft-sh/vcluster/pkg/controllers/resources/services"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/encoding"
	"github.com/loft-sh/vcluster/pkg/util/random"
	requestpkg "github.com/loft-sh/vcluster/pkg/util/request"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metainternalversionscheme "k8s.io/apimachinery/pkg/apis/meta/internalversion/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/endpoints/handlers/negotiation"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func WithServiceCreateRedirect(handler http.Handler, uncachedLocalClient, uncachedVirtualClient client.Client, virtualConfig *rest.Config, syncedLabels []string) http.Handler {
	decoder := encoding.NewDecoder(uncachedLocalClient.Scheme(), false)
	s := serializer.NewCodecFactory(uncachedVirtualClient.Scheme())
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		info, ok := request.RequestInfoFrom(req.Context())
		if !ok {
			requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, fmt.Errorf("request info is missing"))
			return
		}

		userInfo, ok := request.UserFrom(req.Context())
		if !ok {
			requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, fmt.Errorf("user info is missing"))
			return
		}

		if info.APIVersion == corev1.SchemeGroupVersion.Version && info.APIGroup == corev1.SchemeGroupVersion.Group && info.Resource == "services" && info.Subresource == "" {
			if info.Verb == "create" {
				options := &metav1.CreateOptions{}
				if err := metainternalversionscheme.ParameterCodec.DecodeParameters(req.URL.Query(), metav1.SchemeGroupVersion, options); err != nil {
					responsewriters.ErrorNegotiated(err, s, corev1.SchemeGroupVersion, w, req)
					return
				}

				if len(options.DryRun) == 0 {
					uncachedVirtualImpersonatingClient, err := clienthelper.NewImpersonatingClient(virtualConfig, uncachedVirtualClient.RESTMapper(), userInfo, uncachedVirtualClient.Scheme())
					if err != nil {
						responsewriters.ErrorNegotiated(err, s, corev1.SchemeGroupVersion, w, req)
						return
					}

					svc, err := createService(req, decoder, uncachedLocalClient, uncachedVirtualImpersonatingClient, info.Namespace, syncedLabels)
					if err != nil {
						responsewriters.ErrorNegotiated(err, s, corev1.SchemeGroupVersion, w, req)
						return
					}

					responsewriters.WriteObjectNegotiated(s, negotiation.DefaultEndpointRestrictions, corev1.SchemeGroupVersion, w, req, http.StatusCreated, svc, false)
					return
				}
			} else if info.Verb == "update" {
				options := &metav1.UpdateOptions{}
				if err := metainternalversionscheme.ParameterCodec.DecodeParameters(req.URL.Query(), metav1.SchemeGroupVersion, options); err != nil {
					responsewriters.ErrorNegotiated(err, s, corev1.SchemeGroupVersion, w, req)
					return
				}

				if len(options.DryRun) == 0 {
					// the only case we have to intercept this is when the service type changes from ExternalName
					vService := &corev1.Service{}
					err := uncachedVirtualClient.Get(req.Context(), client.ObjectKey{Namespace: info.Namespace, Name: info.Name}, vService)
					if err != nil {
						responsewriters.ErrorNegotiated(err, s, corev1.SchemeGroupVersion, w, req)
						return
					}

					if vService.Spec.Type == corev1.ServiceTypeExternalName {
						uncachedVirtualImpersonatingClient, err := clienthelper.NewImpersonatingClient(virtualConfig, uncachedVirtualClient.RESTMapper(), userInfo, uncachedVirtualClient.Scheme())
						if err != nil {
							responsewriters.ErrorNegotiated(err, s, corev1.SchemeGroupVersion, w, req)
							return
						}

						svc, err := updateService(req, decoder, uncachedLocalClient, uncachedVirtualImpersonatingClient, vService)
						if err != nil {
							responsewriters.ErrorNegotiated(err, s, corev1.SchemeGroupVersion, w, req)
							return
						}

						responsewriters.WriteObjectNegotiated(s, negotiation.DefaultEndpointRestrictions, corev1.SchemeGroupVersion, w, req, http.StatusOK, svc, false)
						return
					}
				}
			}
		}

		handler.ServeHTTP(w, req)
	})
}

func updateService(req *http.Request, decoder encoding.Decoder, localClient client.Client, virtualClient client.Client, oldVService *corev1.Service) (runtime.Object, error) {
	// we use a background context from now on as this is a critical operation
	ctx := context.Background()

	// authorization will be done at this point already, so we can redirect the request to the physical cluster
	rawObj, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	serviceGVK := corev1.SchemeGroupVersion.WithKind("Service")
	svc, err := decoder.Decode(rawObj, &serviceGVK)
	if err != nil {
		return nil, err
	}

	newVService, ok := svc.(*corev1.Service)
	if !ok {
		return nil, fmt.Errorf("expected service object")
	}

	// type has not changed, we just update here
	if newVService.ResourceVersion != oldVService.ResourceVersion || newVService.Spec.Type == oldVService.Spec.Type || newVService.Spec.ClusterIP != "" {
		err = virtualClient.Update(req.Context(), newVService)
		if err != nil {
			return nil, err
		}

		return newVService, nil
	}

	// okay now we have to change the physical service
	pService := &corev1.Service{}
	err = localClient.Get(ctx, client.ObjectKey{Namespace: translate.Default.PhysicalNamespace(oldVService.Namespace), Name: translate.Default.PhysicalName(oldVService.Name, oldVService.Namespace)}, pService)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil, kerrors.NewNotFound(corev1.Resource("services"), oldVService.Name)
		}

		return nil, err
	}

	// we try to patch the service as this has the best chances to go through
	originalPService := pService.DeepCopy()
	pService.Spec.Type = newVService.Spec.Type
	pService.Spec.Ports = newVService.Spec.Ports
	pService.Spec.ClusterIP = ""
	err = localClient.Patch(ctx, pService, client.MergeFrom(originalPService))
	if err != nil {
		return nil, err
	}

	// now we have the cluster ip that we can apply to the new service
	newVService.Spec.ClusterIP = pService.Spec.ClusterIP
	// also we need to apply newly allocated node ports
	newVService.Spec.HealthCheckNodePort = pService.Spec.HealthCheckNodePort
	newVService.Spec.Ports = pService.Spec.Ports

	err = virtualClient.Update(ctx, newVService)
	if err != nil {
		// this is actually worst case that can happen, as we have somehow now a really strange
		// state in the cluster. This needs to be cleaned up by the controller via delete and create
		// and we delete the physical service here. Maybe there is a better solution to this, but for
		// now it works
		_ = localClient.Delete(ctx, pService)
		return nil, err
	}

	return newVService, nil
}

func createService(req *http.Request, decoder encoding.Decoder, localClient client.Client, virtualClient client.Client, fromNamespace string, syncedLabels []string) (runtime.Object, error) {
	// we use a background context from now on as this is a critical operation
	ctx := context.Background()

	// authorization will be done at this point already, so we can redirect the request to the physical cluster
	rawObj, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	serviceGVK := corev1.SchemeGroupVersion.WithKind("Service")
	svc, err := decoder.Decode(rawObj, &serviceGVK)
	if err != nil {
		return nil, err
	}

	vService, ok := svc.(*corev1.Service)
	if !ok {
		return nil, fmt.Errorf("expected service object")
	}

	// make sure the namespace is correct and filled
	vService.Namespace = fromNamespace

	// generate a name, because this field is cleared
	if vService.GenerateName != "" && vService.Name == "" {
		vService.Name = vService.GenerateName + random.String(5)
	}

	newService := translate.Default.ApplyMetadata(vService, syncedLabels).(*corev1.Service)
	if newService.Annotations == nil {
		newService.Annotations = map[string]string{}
	}
	newService.Annotations[services.ServiceBlockDeletion] = "true"
	newService.Spec.Selector = translate.Default.TranslateLabels(vService.Spec.Selector, vService.Namespace, nil)
	err = localClient.Create(req.Context(), newService)
	if err != nil {
		klog.Infof("Error creating service in physical cluster: %v", err)
		if kerrors.IsAlreadyExists(err) {
			return nil, kerrors.NewAlreadyExists(corev1.Resource("services"), vService.Name)
		}

		return nil, err
	}

	vService.Spec.ClusterIP = newService.Spec.ClusterIP
	vService.Spec.ClusterIPs = newService.Spec.ClusterIPs
	vService.Spec.Ports = newService.Spec.Ports
	vService.Spec.HealthCheckNodePort = newService.Spec.HealthCheckNodePort
	vService.Status = newService.Status

	// now create the service in the virtual cluster
	err = virtualClient.Create(req.Context(), vService)
	if err != nil {
		// try to cleanup the created physical service
		klog.Infof("Error creating service in virtual cluster: %v", err)
		_ = localClient.Delete(ctx, newService)
		return nil, err
	}

	return vService, nil
}
