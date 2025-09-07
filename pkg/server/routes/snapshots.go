package routes

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	PostSnapshotsURL = "/vcluster/snapshots"
)

func WithSnapshotsCreate(ctx context.Context, uncachedLocalClient, uncachedVirtualClient client.Client, vConfig *config.VirtualClusterConfig) http.Handler {
	logger := log.GetInstance()
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// read the snapshot request and options from the HTTP request body
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var snapshotRequestData map[string]json.RawMessage
		err = json.Unmarshal(body, &snapshotRequestData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var snapshotRequest snapshot.Request
		err = json.Unmarshal(snapshotRequestData[snapshot.RequestKey], &snapshotRequest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var snapshotOptions snapshot.Options
		err = json.Unmarshal(snapshotRequestData[snapshot.OptionsKey], &snapshotOptions)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// create the snapshot request Secret and ConfigMap
		// - for shared nodes, create resources in the host cluster in the vCluster namespace
		// - for private nodes, create resources in the virtual cluster in the kube-system namespace
		var kubeClient client.Client
		var snapshotRequestNamespace string
		if vConfig.Config.PrivateNodes.Enabled {
			kubeClient = uncachedVirtualClient
			snapshotRequestNamespace = "kube-system"
		} else {
			kubeClient = uncachedLocalClient
			snapshotRequestNamespace = vConfig.HostNamespace

		}

		// create snapshot request Secret
		secret, err := snapshot.CreateSnapshotOptionsSecret(snapshotRequestNamespace, &snapshotOptions)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if snapshotRequest.Name != "" {
			// snapshot request name is specified
			secret.Name = snapshotRequest.Name
		} else {
			// snapshot request name is auto-generated
			secret.GenerateName = "snapshot-request-"
		}
		err = kubeClient.Create(ctx, secret)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// create snapshot request ConfigMap
		if snapshotRequest.Name == "" {
			snapshotRequest.Name = secret.Name
		}
		configMap, err := snapshot.CreateSnapshotRequestConfigMap(snapshotRequestNamespace, &snapshotRequest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		configMap.Name = secret.Name
		err = kubeClient.Create(ctx, configMap)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// finally, return snapshot request JSON as a response
		logger.Infof("Snapshot request %s/%s created", configMap.Namespace, configMap.Name)
		snapshotRequestJSON, err := json.Marshal(snapshotRequest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, err = w.Write(snapshotRequestJSON)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}
