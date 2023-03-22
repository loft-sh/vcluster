package filters

import (
	"context"
	"net/http"
	"strings"

	"github.com/loft-sh/vcluster/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type nodeName int

// nodeNameKey is the NodeName key for the context. It's of private type here. Because
// keys are interfaces and interfaces are equal when the type and the value is equal, this
// does not conflict with the keys defined in pkg/api.
const nodeNameKey nodeName = iota

func WithNodeName(h http.Handler, cli client.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		nodeName := nodeNameFromHost(req, cli)
		if nodeName != "" {
			req = req.WithContext(context.WithValue(req.Context(), nodeNameKey, nodeName))
		}
		h.ServeHTTP(w, req)
	})
}

// NodeNameFrom returns a node name if there is any
func NodeNameFrom(ctx context.Context) (string, bool) {
	info, ok := ctx.Value(nodeNameKey).(string)
	return info, ok
}

func nodeNameFromHost(req *http.Request, cli client.Client) string {
	splitted := strings.Split(req.Host, ":")
	if len(splitted) == 2 {
		hostname := splitted[0]
		nodeList := &corev1.NodeList{}
		err := cli.List(req.Context(), nodeList, client.MatchingFields{constants.IndexByHostName: hostname})
		if err != nil && !errors.IsNotFound(err) {
			klog.Error(err, "couldn't fetch nodename for hostname")
			return ""
		}
		if len(nodeList.Items) == 1 {
			nodeName := nodeList.Items[0].Name
			return nodeName
		}
	}
	return ""

}
