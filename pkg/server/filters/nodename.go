package filters

import (
	"context"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"net/http"
	"strings"
)

type nodeName int

// nodeNameKey is the NodeName key for the context. It's of private type here. Because
// keys are interfaces and interfaces are equal when the type and the value is equal, this
// does not conflict with the keys defined in pkg/api.
const nodeNameKey nodeName = iota

func WithNodeName(h http.Handler, currentNamespace string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		nodeName := nodeNameFromHost(req.Host, currentNamespace)
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

func nodeNameFromHost(host, currentNamespace string) string {
	suffix := "." + translate.Suffix + "." + currentNamespace + "." + constants.NodeSuffix

	// retrieve the node name
	splitted := strings.Split(host, ":")
	if len(splitted) == 2 && strings.HasSuffix(splitted[0], suffix) {
		return strings.TrimSuffix(splitted[0], suffix)
	}

	return ""
}
