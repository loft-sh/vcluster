package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	utilrequest "github.com/loft-sh/vcluster/pkg/util/request"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	maxRequestBytes = 1 << 20 // 1MB
)

type snapshotRouteHandler func(*synccontext.ControllerContext, http.ResponseWriter, *http.Request)

// WithSnapshots returns an http.Handler that intercepts vCluster snapshot API
// requests and delegates everything else to next.
func WithSnapshots(next http.Handler, ctx *synccontext.ControllerContext) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isSnapshotPath(r.URL.Path) {
			if next != nil {
				next.ServeHTTP(w, r)
			}
			return
		}
		if ctx == nil {
			fail(w, r, http.StatusInternalServerError, errors.New("controller context is nil"))
			return
		}

		newSnapshotMux(ctx).ServeHTTP(w, r)
	})
}

func newSnapshotMux(ctx *synccontext.ControllerContext) *http.ServeMux {
	mux := http.NewServeMux()
	handle := func(pattern string, handler snapshotRouteHandler) {
		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			handler(ctx, w, r)
		})
	}

	handle("HEAD /vcluster/snapshots", handleProbe)
	handle("POST /vcluster/snapshots", handleCreate)
	handle("POST /vcluster/snapshots/list", handleList)
	handle("POST /vcluster/snapshots/request", handleCreateRequest)
	handle("POST /vcluster/snapshots/request/delete", handleDeleteRequest)
	handle("GET /vcluster/snapshots/request/{name}", handleGetRequest)

	return mux
}

func handleProbe(_ *synccontext.ControllerContext, w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func handleCreate(ctx *synccontext.ControllerContext, w http.ResponseWriter, r *http.Request) {
	client, err := snapshotClient(r, false)
	if err != nil {
		fail(w, r, http.StatusBadRequest, err)
		return
	}
	if ctx.Config == nil {
		fail(w, r, http.StatusInternalServerError, fmt.Errorf("snapshot config is nil"))
		return
	}
	config := *ctx.Config
	if err := client.Run(r.Context(), &config); err != nil {
		fail(w, r, http.StatusInternalServerError, err)
		return
	}

	utilrequest.SucceedWithStatus(w)
}

func handleList(_ *synccontext.ControllerContext, w http.ResponseWriter, r *http.Request) {
	client, err := snapshotClient(r, true)
	if err != nil {
		fail(w, r, http.StatusBadRequest, err)
		return
	}
	snapshots, err := client.List(r.Context())
	if err != nil {
		fail(w, r, http.StatusInternalServerError, err)
		return
	}

	utilrequest.SucceedWithObject(w, snapshots)
}

func handleCreateRequest(ctx *synccontext.ControllerContext, w http.ResponseWriter, r *http.Request) {
	options, err := snapshotOptions(r, false)
	if err != nil {
		fail(w, r, http.StatusBadRequest, err)
		return
	}

	kubeClient, requestNamespace, err := snapshotRequestResourcesClient(ctx)
	if err != nil {
		fail(w, r, http.StatusInternalServerError, err)
		return
	}

	request, err := snapshot.CreateSnapshotRequestResources(r.Context(), requestNamespace, ctx.Config.Name, ctx.Config, options, kubeClient)
	if err != nil {
		fail(w, r, http.StatusInternalServerError, err)
		return
	}

	utilrequest.SucceedWithObject(w, request)
}

func handleGetRequest(ctx *synccontext.ControllerContext, w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		fail(w, r, http.StatusBadRequest, fmt.Errorf("snapshot request name is required"))
		return
	}

	kubeClient, requestNamespace, err := snapshotRequestResourcesClient(ctx)
	if err != nil {
		fail(w, r, http.StatusInternalServerError, err)
		return
	}

	configMap, err := kubeClient.CoreV1().ConfigMaps(requestNamespace).Get(r.Context(), name, metav1.GetOptions{})
	if kerrors.IsNotFound(err) {
		fail(w, r, http.StatusNotFound, fmt.Errorf("snapshot request %s not found", name))
		return
	} else if err != nil {
		fail(w, r, http.StatusInternalServerError, err)
		return
	}

	request, err := snapshot.UnmarshalSnapshotRequest(configMap)
	if err != nil {
		fail(w, r, http.StatusInternalServerError, err)
		return
	}

	utilrequest.SucceedWithObject(w, request)
}

func handleDeleteRequest(ctx *synccontext.ControllerContext, w http.ResponseWriter, r *http.Request) {
	options, err := snapshotOptions(r, false)
	if err != nil {
		fail(w, r, http.StatusBadRequest, err)
		return
	}

	kubeClient, requestNamespace, err := snapshotRequestResourcesClient(ctx)
	if err != nil {
		fail(w, r, http.StatusInternalServerError, err)
		return
	}

	if err := snapshot.DeleteSnapshotRequestResources(r.Context(), requestNamespace, ctx.Config.Name, ctx.Config, options, kubeClient); err != nil {
		fail(w, r, http.StatusInternalServerError, err)
		return
	}

	utilrequest.SucceedWithStatus(w)
}

func snapshotRequestResourcesClient(ctx *synccontext.ControllerContext) (*kubernetes.Clientset, string, error) {
	isHostMode, err := snapshot.IsSnapshotRequestCreatedInHostCluster(ctx.Config)
	if err != nil {
		return nil, "", err
	}

	requestNamespace := constants.VClusterStandaloneSnapshotNamespace
	requestManager := ctx.VirtualManager
	if isHostMode {
		requestNamespace = ctx.Config.HostNamespace
		requestManager = ctx.HostManager
	}
	if requestManager == nil {
		return nil, "", fmt.Errorf("snapshot request manager is nil")
	}

	kubeClient, err := kubeClientFromManager(requestManager)
	if err != nil {
		return nil, "", err
	}
	return kubeClient, requestNamespace, nil
}

func kubeClientFromManager(m manager.Manager) (*kubernetes.Clientset, error) {
	kubeClient, err := kubernetes.NewForConfig(m.GetConfig())
	if err != nil {
		return nil, fmt.Errorf("create kube client: %w", err)
	}
	return kubeClient, nil
}

func snapshotClient(r *http.Request, isList bool) (*snapshot.Client, error) {
	options, err := snapshotOptions(r, isList)
	if err != nil {
		return nil, err
	}

	return &snapshot.Client{Options: *options}, nil
}

func snapshotOptions(r *http.Request, isList bool) (*snapshot.Options, error) {
	if r.Body == nil {
		return nil, fmt.Errorf("request body is required")
	}
	defer r.Body.Close()

	body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read request body: %w", err)
	}
	if len(body) > maxRequestBytes {
		return nil, fmt.Errorf("request body is too large")
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("request body is required")
	}

	var request OptionsRequest
	if err := json.Unmarshal(body, &request); err == nil && request.Options != nil {
		if err := snapshot.Validate(request.Options, isList); err != nil {
			return nil, err
		}
		return request.Options, nil
	}

	var options snapshot.Options
	if err := json.Unmarshal(body, &options); err != nil {
		return nil, fmt.Errorf("decode snapshot options: %w", err)
	}
	if err := snapshot.Validate(&options, isList); err != nil {
		return nil, err
	}
	return &options, nil
}

func fail(w http.ResponseWriter, r *http.Request, code int, err error) {
	utilrequest.FailWithStatus(w, r, int32(code), err)
}

func isSnapshotPath(path string) bool {
	return path == "/vcluster/snapshots" || strings.HasPrefix(path, "/vcluster/snapshots/")
}
