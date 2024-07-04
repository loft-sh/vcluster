package filters

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/loft-sh/vcluster/pkg/server/handler"
	"github.com/loft-sh/vcluster/pkg/util/encoding"
	requestpkg "github.com/loft-sh/vcluster/pkg/util/request"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metainternalversionscheme "k8s.io/apimachinery/pkg/apis/meta/internalversion/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/endpoints/handlers/negotiation"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func WithNodeChanges(ctx context.Context, h http.Handler, uncachedLocalClient, uncachedVirtualClient client.Client, virtualConfig *rest.Config) http.Handler {
	decoder := encoding.NewDecoder(uncachedLocalClient.Scheme(), false)
	s := serializer.NewCodecFactory(uncachedVirtualClient.Scheme())
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		info, ok := request.RequestInfoFrom(req.Context())
		if !ok {
			requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, fmt.Errorf("request info is missing"))
			return
		}

		if info.APIVersion == corev1.SchemeGroupVersion.Version && info.APIGroup == corev1.SchemeGroupVersion.Group && info.Resource == "nodes" {
			if info.Verb == "update" {
				options := &metav1.UpdateOptions{}
				if err := metainternalversionscheme.ParameterCodec.DecodeParameters(req.URL.Query(), metav1.SchemeGroupVersion, options); err != nil {
					responsewriters.ErrorNegotiated(err, s, corev1.SchemeGroupVersion, w, req)
					return
				}

				if len(options.DryRun) == 0 {
					// authorization will be done at this point already, so we can redirect the request to the physical cluster
					rawObj, err := io.ReadAll(req.Body)
					if err != nil {
						responsewriters.ErrorNegotiated(err, s, corev1.SchemeGroupVersion, w, req)
						return
					}

					updatedNode, err := updateNode(ctx, decoder, uncachedLocalClient, uncachedVirtualClient, rawObj, info.Subresource == "status")
					if err != nil {
						responsewriters.ErrorNegotiated(err, s, corev1.SchemeGroupVersion, w, req)
						return
					}

					responsewriters.WriteObjectNegotiated(s, negotiation.DefaultEndpointRestrictions, corev1.SchemeGroupVersion, w, req, http.StatusOK, updatedNode, false)
					return
				}
			} else if info.Verb == "patch" {
				options := &metav1.PatchOptions{}
				if err := metainternalversionscheme.ParameterCodec.DecodeParameters(req.URL.Query(), metav1.SchemeGroupVersion, options); err != nil {
					responsewriters.ErrorNegotiated(err, s, corev1.SchemeGroupVersion, w, req)
					return
				}

				if len(options.DryRun) == 0 {
					patchNode(ctx, w, req, s, decoder, uncachedLocalClient, uncachedVirtualClient, virtualConfig, info.Subresource == "status")
					return
				}
			}
		}

		h.ServeHTTP(w, req)
	})
}

func patchNode(ctx context.Context, w http.ResponseWriter, req *http.Request, s runtime.NegotiatedSerializer, decoder encoding.Decoder, localClient client.Client, virtualClient client.Client, virtualConfig *rest.Config, status bool) {
	h, err := handler.Handler("", virtualConfig, nil)
	if err != nil {
		responsewriters.ErrorNegotiated(err, s, corev1.SchemeGroupVersion, w, req)
		return
	}

	// make sure its a dry run
	q := req.URL.Query()
	q.Add("dryRun", "All")
	req.URL.RawQuery = q.Encode()
	code, header, data, err := ExecuteRequest(req, h)
	if err != nil {
		responsewriters.ErrorNegotiated(err, s, corev1.SchemeGroupVersion, w, req)
		return
	} else if code != http.StatusOK {
		WriteWithHeader(w, code, header, data)
		return
	}

	vObj, err := updateNode(ctx, decoder, localClient, virtualClient, data, status)
	if err != nil {
		responsewriters.ErrorNegotiated(err, s, corev1.SchemeGroupVersion, w, req)
		return
	}

	responsewriters.WriteObjectNegotiated(s, negotiation.DefaultEndpointRestrictions, corev1.SchemeGroupVersion, w, req, http.StatusOK, vObj, false)
}

func updateNode(ctx context.Context, decoder encoding.Decoder, localClient client.Client, virtualClient client.Client, rawObj []byte, status bool) (runtime.Object, error) {
	nodeGVK := corev1.SchemeGroupVersion.WithKind("Node")
	vObj, err := decoder.Decode(rawObj, &nodeGVK)
	if err != nil {
		return nil, err
	}

	vNode, ok := vObj.(*corev1.Node)
	if !ok {
		return nil, fmt.Errorf("expected node object")
	}

	// get the current virtual node
	curVNode := &corev1.Node{}
	err = virtualClient.Get(ctx, client.ObjectKey{Name: vNode.Name}, curVNode)
	if err != nil {
		return nil, err
	} else if curVNode.ResourceVersion != vNode.ResourceVersion {
		return nil, kerrors.NewConflict(corev1.Resource("nodes"), vNode.Name, fmt.Errorf("the object has been modified; please apply your changes to the latest version and try again"))
	}

	// get the corresponding physical node
	pNode := &corev1.Node{}
	err = localClient.Get(ctx, client.ObjectKey{Name: vNode.Name}, pNode)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil, kerrors.NewNotFound(corev1.Resource("nodes"), vNode.Name)
		}

		return nil, err
	}

	// apply the changes to from the vNode
	newNode := pNode.DeepCopy()
	newNode.Labels = vNode.Labels
	newNode.Spec.Taints = vNode.Spec.Taints
	newNode.Status.Capacity = vNode.Status.Capacity

	// if there are no changes, just return the provided object
	patch := client.MergeFrom(pNode)
	data, err := patch.Data(newNode)
	if err != nil || string(data) == "{}" {
		return vNode, nil
	}

	// patch the physical node
	if status {
		err = localClient.Status().Patch(ctx, newNode, patch)
	} else {
		err = localClient.Patch(ctx, newNode, patch)
	}
	if err != nil {
		return nil, err
	} else if newNode.ResourceVersion == pNode.ResourceVersion {
		return vNode, nil
	}

	// now let's wait for the virtual node to update
	err = wait.PollUntilContextTimeout(ctx, time.Second*4, time.Millisecond*200, true, func(ctx context.Context) (bool, error) {
		updatedNode := &corev1.Node{}
		err := virtualClient.Get(ctx, client.ObjectKey{Name: vNode.Name}, updatedNode)
		if err != nil {
			return false, nil
		} else if updatedNode.ResourceVersion == vNode.ResourceVersion {
			return false, nil
		}

		vNode = updatedNode
		return true, nil
	})
	return vNode, err
}
