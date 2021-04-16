package filters

import (
	"fmt"
	"github.com/loft-sh/vcluster/pkg/util/encoding"
	"github.com/loft-sh/vcluster/pkg/util/random"
	requestpkg "github.com/loft-sh/vcluster/pkg/util/request"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/endpoints/handlers/negotiation"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"k8s.io/apiserver/pkg/endpoints/request"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"sync"
	"time"
)

func WithServiceCreateRedirect(handler http.Handler, localManager ctrl.Manager, virtualManager ctrl.Manager, targetNamespace string, sharedMutex sync.Locker) http.Handler {
	decoder := encoding.NewDecoder(localManager.GetScheme(), false)
	s := serializer.NewCodecFactory(virtualManager.GetScheme())
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		info, ok := request.RequestInfoFrom(req.Context())
		if !ok {
			requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, fmt.Errorf("request info is missing"))
			return
		}

		if info.APIVersion == corev1.SchemeGroupVersion.Version && info.APIGroup == corev1.SchemeGroupVersion.Group && info.Resource == "services" && info.Verb == "create" {
			sharedMutex.Lock()
			svc, err := createService(req, decoder, localManager, virtualManager, info.Namespace, targetNamespace)
			sharedMutex.Unlock()
			if err != nil {
				responsewriters.ErrorNegotiated(err, s, corev1.SchemeGroupVersion, w, req)
				return
			}

			responsewriters.WriteObjectNegotiated(s, negotiation.DefaultEndpointRestrictions, corev1.SchemeGroupVersion, w, req, http.StatusCreated, svc)
			return
		}

		handler.ServeHTTP(w, req)
	})
}

func createService(req *http.Request, decoder encoding.Decoder, localManager ctrl.Manager, virtualManager ctrl.Manager, fromNamespace, targetNamespace string) (runtime.Object, error) {
	// authorization will be done at this point already, so we can redirect the request to the physical cluster
	rawObj, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	svc, err := decoder.Decode(rawObj)
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
		vService.Name = vService.GenerateName + random.RandomString(5)
	}

	newObj, err := translate.SetupMetadata(targetNamespace, vService)
	if err != nil {
		return nil, errors.Wrap(err, "error setting metadata")
	}

	newService := newObj.(*corev1.Service)
	newService.Spec.Selector = nil

	err = localManager.GetClient().Create(req.Context(), newService)
	if err != nil {
		return nil, err
	}

	vService.Spec.ClusterIP = newService.Spec.ClusterIP
	vService.Status = newService.Status

	// now create the service in the virtual cluster
	err = virtualManager.GetClient().Create(req.Context(), vService)
	if err != nil {
		return nil, err
	}

	// wait until caches are synced
	return vService, wait.PollImmediate(time.Millisecond*100, time.Second*5, func() (bool, error) {
		err := localManager.GetClient().Get(req.Context(), types.NamespacedName{Namespace: targetNamespace, Name: newService.Name}, &corev1.Service{})
		if err != nil {
			if kerrors.IsNotFound(err) == false {
				return false, err
			}

			return false, nil
		}

		err = virtualManager.GetClient().Get(req.Context(), types.NamespacedName{Namespace: vService.Namespace, Name: vService.Name}, &corev1.Service{})
		if err != nil {
			if kerrors.IsNotFound(err) == false {
				return false, err
			}

			return false, nil
		}

		return true, nil
	})
}
