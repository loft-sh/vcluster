package filters

import (
	"fmt"
	"io"
	"net/http"

	"github.com/loft-sh/vcluster/pkg/controllers/resources/services"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
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
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func WithServiceCreateRedirect(handler http.Handler, registerCtx *synccontext.RegisterContext, uncachedLocalClient, uncachedVirtualClient client.Client) http.Handler {
	decoder := encoding.NewDecoder(scheme.Scheme, false)
	s := serializer.NewCodecFactory(scheme.Scheme)
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
					uncachedVirtualImpersonatingClient, err := clienthelper.NewImpersonatingClient(registerCtx.VirtualManager.GetConfig(), uncachedVirtualClient.RESTMapper(), userInfo, uncachedVirtualClient.Scheme())
					if err != nil {
						responsewriters.ErrorNegotiated(err, s, corev1.SchemeGroupVersion, w, req)
						return
					}

					syncContext := registerCtx.ToSyncContext("create-service")
					syncContext.VirtualClient = uncachedVirtualImpersonatingClient
					syncContext.PhysicalClient = uncachedLocalClient

					svc, err := createService(syncContext, req, decoder, info.Namespace)
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
						uncachedVirtualImpersonatingClient, err := clienthelper.NewImpersonatingClient(registerCtx.VirtualManager.GetConfig(), uncachedVirtualClient.RESTMapper(), userInfo, uncachedVirtualClient.Scheme())
						if err != nil {
							responsewriters.ErrorNegotiated(err, s, corev1.SchemeGroupVersion, w, req)
							return
						}

						syncContext := registerCtx.ToSyncContext("update-service")
						syncContext.VirtualClient = uncachedVirtualImpersonatingClient
						syncContext.PhysicalClient = uncachedLocalClient

						svc, err := updateService(syncContext, req, decoder, vService)
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

func updateService(ctx *synccontext.SyncContext, req *http.Request, decoder encoding.Decoder, oldVService *corev1.Service) (runtime.Object, error) {
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
		err = ctx.VirtualClient.Update(req.Context(), newVService)
		if err != nil {
			return nil, err
		}

		return newVService, nil
	}

	// okay now we have to change the physical service
	pService := &corev1.Service{}
	pServiceName := mappings.VirtualToHost(ctx, oldVService.Name, oldVService.Namespace, mappings.Services())
	err = ctx.PhysicalClient.Get(ctx, client.ObjectKey{Namespace: pServiceName.Namespace, Name: pServiceName.Name}, pService)
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
	err = ctx.PhysicalClient.Patch(ctx, pService, client.MergeFrom(originalPService))
	if err != nil {
		return nil, err
	}

	// now we have the cluster ip that we can apply to the new service
	newVService.Spec.ClusterIP = pService.Spec.ClusterIP
	// also we need to apply newly allocated node ports
	newVService.Spec.HealthCheckNodePort = pService.Spec.HealthCheckNodePort
	newVService.Spec.Ports = pService.Spec.Ports

	err = ctx.VirtualClient.Update(ctx, newVService)
	if err != nil {
		// this is actually worst case that can happen, as we have somehow now a really strange
		// state in the cluster. This needs to be cleaned up by the controller via delete and create
		// and we delete the physical service here. Maybe there is a better solution to this, but for
		// now it works
		_ = ctx.PhysicalClient.Delete(ctx, pService)
		return nil, err
	}

	return newVService, nil
}

func createService(ctx *synccontext.SyncContext, req *http.Request, decoder encoding.Decoder, fromNamespace string) (runtime.Object, error) {
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

	newService := translate.HostMetadata(vService, mappings.VirtualToHost(ctx, vService.Name, vService.Namespace, mappings.Services()))
	if newService.Annotations == nil {
		newService.Annotations = map[string]string{}
	}
	newService.Annotations[services.ServiceBlockDeletion] = "true"
	newService.Spec.Selector = translate.HostLabelsMap(vService.Spec.Selector, nil, vService.Namespace, false)
	err = ctx.PhysicalClient.Create(req.Context(), newService)
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
	err = ctx.VirtualClient.Create(req.Context(), vService)
	if err != nil {
		// try to cleanup the created physical service
		klog.Infof("Error creating service in virtual cluster: %v", err)
		_ = ctx.PhysicalClient.Delete(ctx, newService)
		return nil, err
	}

	return vService, nil
}
