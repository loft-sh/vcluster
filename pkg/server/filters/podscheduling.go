package filters

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/encoding"
	requestpkg "github.com/loft-sh/vcluster/pkg/util/request"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"k8s.io/apiserver/pkg/endpoints/request"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func WithPodSchedulerCheck(h http.Handler, ctx *synccontext.RegisterContext, cachedVirtualClient client.Client) http.Handler {
	if !ctx.Config.Sync.ToHost.Pods.HybridScheduling.Enabled {
		return h
	}

	scheme := cachedVirtualClient.Scheme()
	decoder := encoding.NewDecoder(scheme, false)
	s := serializer.NewCodecFactory(scheme)

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requestInfo, ok := request.RequestInfoFrom(req.Context())
		if !ok {
			requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, fmt.Errorf("request info is missing"))
			return
		}
		if !isPodBindingRequest(requestInfo) {
			h.ServeHTTP(w, req)
		}

		requestBody, err := io.ReadAll(req.Body)
		if err != nil {
			responsewriters.ErrorNegotiated(err, s, corev1.SchemeGroupVersion, w, req)
			return
		}

		vBinding, err := getBindingResourceFromRequest(requestInfo, requestBody, decoder)
		if err != nil {
			responsewriters.ErrorNegotiated(err, s, corev1.SchemeGroupVersion, w, req)
			return
		}

		pod, err := getPodFromBinding(ctx, ctx.VirtualManager.GetClient(), vBinding)
		if err != nil {
			responsewriters.ErrorNegotiated(err, s, corev1.SchemeGroupVersion, w, req)
			return
		}

		if isSchedulerConfiguredAsHostScheduler(ctx.Config.Sync.ToHost.Pods.HybridScheduling.HostSchedulers, pod.Spec.SchedulerName) {
			err = fmt.Errorf("scheduler %s is configured as a host scheduler, so scheduler with the same name is not allowed to schedule pods in the virtual cluster", pod.Spec.SchedulerName)
			requestpkg.FailWithStatus(w, req, http.StatusMethodNotAllowed, err)
			return
		}

		h.ServeHTTP(w, req)
	})
}

func isPodBindingRequest(r *request.RequestInfo) bool {
	if !r.IsResourceRequest {
		return false
	}

	return r.APIGroup == corev1.SchemeGroupVersion.Group &&
		r.APIVersion == corev1.SchemeGroupVersion.Version &&
		r.Resource == "pods" &&
		r.Subresource == "bind"
}

func getBindingResourceFromRequest(requestInfo *request.RequestInfo, requestBody []byte, decoder encoding.Decoder) (*corev1.Binding, error) {
	if requestInfo == nil {
		return nil, errors.New("requestInfo is nil")
	}
	if decoder == nil {
		return nil, errors.New("decoder is nil")
	}

	bindingGVK := corev1.SchemeGroupVersion.WithKind("Binding")
	vObject, err := decoder.Decode(requestBody, &bindingGVK)
	if err != nil {
		return nil, fmt.Errorf("failed to decode Binding resource from request body: %w", err)
	}

	vBinding, ok := vObject.(*corev1.Binding)
	if !ok {
		return nil, fmt.Errorf("expected binding object")
	}

	return vBinding, nil
}

func getPodFromBinding(ctx context.Context, cachedVirtualClient client.Client, binding *corev1.Binding) (*corev1.Pod, error) {
	namespacedName := types.NamespacedName{
		Namespace: binding.Namespace,
		Name:      binding.Name,
	}
	pod := &corev1.Pod{}
	err := cachedVirtualClient.Get(ctx, namespacedName, pod)
	if err != nil {
		return nil, fmt.Errorf("failed to get pod %s/%s: %w", binding.Namespace, binding.Name, err)
	}
	return pod, nil
}

func isSchedulerConfiguredAsHostScheduler(hostSchedulers []string, schedulerName string) bool {
	return slices.Contains(hostSchedulers, schedulerName)
}
