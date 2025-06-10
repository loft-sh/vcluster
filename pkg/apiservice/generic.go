package apiservice

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/server/handler"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	genericapifilters "k8s.io/apiserver/pkg/endpoints/filters"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

func checkExistingAPIService(ctx context.Context, client client.Client, groupVersion schema.GroupVersion) bool {
	var exists bool
	_ = applyOperation(ctx, func(ctx context.Context) (bool, error) {
		err := client.Get(ctx, types.NamespacedName{Name: groupVersion.Version + "." + groupVersion.Group}, &apiregistrationv1.APIService{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return true, nil
			}

			return false, err
		}

		exists = true
		return true, nil
	})

	return exists
}

func applyOperation(ctx context.Context, operationFunc wait.ConditionWithContextFunc) error {
	return wait.ExponentialBackoffWithContext(ctx, wait.Backoff{
		Duration: time.Second,
		Factor:   1.5,
		Cap:      time.Minute,
		Steps:    math.MaxInt32,
	}, operationFunc)
}

func deleteOperation(ctrlCtx *synccontext.ControllerContext, groupVersion schema.GroupVersion) wait.ConditionWithContextFunc {
	return func(ctx context.Context) (bool, error) {
		err := ctrlCtx.VirtualManager.GetClient().Delete(ctx, &apiregistrationv1.APIService{
			ObjectMeta: metav1.ObjectMeta{
				Name: groupVersion.Version + "." + groupVersion.Group,
			},
		})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return true, nil
			}

			klog.Errorf("error deleting api service %v", err)
			return false, nil
		}

		return true, nil
	}
}

func createOperation(ctrlCtx *synccontext.ControllerContext, serviceName string, hostPort int, groupVersion schema.GroupVersion) wait.ConditionWithContextFunc {
	return func(ctx context.Context) (bool, error) {
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName,
				Namespace: "kube-system",
			},
		}
		_, err := controllerutil.CreateOrUpdate(ctx, ctrlCtx.VirtualManager.GetClient(), service, func() error {
			service.Spec.Type = corev1.ServiceTypeExternalName
			service.Spec.ExternalName = "localhost"
			service.Spec.Ports = []corev1.ServicePort{
				{
					Port: int32(hostPort),
				},
			}
			return nil
		})
		if err != nil {
			if kerrors.IsAlreadyExists(err) {
				return true, nil
			}

			klog.Errorf("error creating api service %v", err)
			return false, nil
		}

		apiServiceSpec := apiregistrationv1.APIServiceSpec{
			Service: &apiregistrationv1.ServiceReference{
				Name:      serviceName,
				Namespace: "kube-system",
				Port:      ptr.To(int32(hostPort)),
			},
			InsecureSkipTLSVerify: true,
			Group:                 groupVersion.Group,
			GroupPriorityMinimum:  100,
			Version:               groupVersion.Version,
			VersionPriority:       100,
		}
		apiService := &apiregistrationv1.APIService{
			ObjectMeta: metav1.ObjectMeta{
				Name: groupVersion.Version + "." + groupVersion.Group,
			},
		}
		_, err = controllerutil.CreateOrUpdate(ctx, ctrlCtx.VirtualManager.GetClient(), apiService, func() error {
			apiService.Spec = apiServiceSpec
			return nil
		})
		if err != nil {
			if kerrors.IsAlreadyExists(err) {
				return true, nil
			}

			klog.Errorf("error creating api service %v", err)
			return false, nil
		}

		return true, nil
	}
}

func StartAPIServiceProxy(
	ctx *synccontext.ControllerContext,
	targetServiceName,
	targetServiceNamespace string,
	targetPort,
	hostPort int,
	extraHandlers ...func(h http.Handler) http.Handler,
) error {
	tlsCertFile := ctx.Config.VirtualClusterKubeConfig().ServerCACert
	tlsKeyFile := ctx.Config.VirtualClusterKubeConfig().ServerCAKey

	hostConfig := rest.CopyConfig(ctx.LocalManager.GetConfig())
	hostConfig.Host = "https://" + targetServiceName + "." + targetServiceNamespace
	if targetPort > 0 {
		hostConfig.Host = hostConfig.Host + ":" + strconv.Itoa(targetPort)
	}
	hostConfig.APIPath = ""
	hostConfig.CAFile = ""
	hostConfig.CAData = nil
	hostConfig.KeyFile = ""
	hostConfig.KeyData = nil
	hostConfig.CertFile = ""
	hostConfig.CertData = nil
	hostConfig.Insecure = true

	proxyHandler, err := handler.Handler("", hostConfig, nil)
	if err != nil {
		return fmt.Errorf("create host proxy handler: %w", err)
	}

	h := serveHandler(ctx, proxyHandler)

	// add custom handlers
	for _, extraHandler := range extraHandlers {
		h = extraHandler(h)
	}

	h = genericapifilters.WithRequestInfo(h, &request.RequestInfoFactory{
		APIPrefixes:          sets.NewString("api", "apis"),
		GrouplessAPIPrefixes: sets.NewString("api"),
	})

	server := &http.Server{
		Addr:    "localhost:" + strconv.Itoa(hostPort),
		Handler: h,
	}

	go func() {
		klog.Infof("Listening apiservice proxy on localhost:%d...", hostPort)
		err = server.ListenAndServeTLS(tlsCertFile, tlsKeyFile)
		if err != nil {
			klog.FromContext(ctx).Error(err, "error listening for apiservice proxy and serve tls")
			os.Exit(1)
		}
	}()

	return nil
}

func serveHandler(ctx context.Context, next http.Handler) http.Handler {
	s := serializer.NewCodecFactory(scheme.Scheme)
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// we only allow traffic to discovery paths
		if !isAPIServiceProxyPathAllowed(request.Method, request.URL.Path) {
			klog.FromContext(ctx).Info("Denied access to api service proxy at path", "path", request.URL.Path, "method", request.Method)
			responsewriters.ErrorNegotiated(
				kerrors.NewForbidden(metav1.SchemeGroupVersion.WithResource("proxy").GroupResource(), "proxy", fmt.Errorf("paths other than discovery paths are not allowed")),
				s,
				corev1.SchemeGroupVersion,
				writer,
				request,
			)
			return
		}

		next.ServeHTTP(writer, request)
	})
}

func isAPIServiceProxyPathAllowed(method, path string) bool {
	if strings.ToUpper(method) != http.MethodGet {
		return false
	}

	path = strings.TrimPrefix(strings.TrimSuffix(path, "/"), "/")
	if strings.HasPrefix(path, "openapi") {
		return true
	}

	if path == "" {
		return true
	}
	if path == "version" {
		return true
	}
	if path == "api" || path == "apis" {
		return true
	}

	splitPath := strings.Split(path, "/")
	if splitPath[0] == "apis" && len(splitPath) <= 3 {
		return true
	} else if splitPath[0] == "api" && len(splitPath) <= 2 {
		return true
	} else if splitPath[0] == ".well-known" {
		return true
	} else if splitPath[0] == "readyz" {
		return true
	} else if splitPath[0] == "livez" {
		return true
	}

	return false
}

func RegisterAPIService(ctx *synccontext.ControllerContext, serviceName string, hostPort int, groupVersion schema.GroupVersion) error {
	return applyOperation(ctx, createOperation(ctx, serviceName, hostPort, groupVersion))
}

func DeregisterAPIService(ctx *synccontext.ControllerContext, groupVersion schema.GroupVersion) error {
	// check if the api service should get created
	exists := checkExistingAPIService(ctx, ctx.VirtualManager.GetClient(), groupVersion)
	if exists {
		return applyOperation(ctx, deleteOperation(ctx, groupVersion))
	}

	return nil
}
