package filters

import (
	"net/http"
	"strings"
)

// WithLeaderReadiness fails /readyz while serving() is false, so the Service skips this pod.
func WithLeaderReadiness(h http.Handler, serving func() bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if strings.HasPrefix(req.URL.Path, "/readyz") && !serving() {
			http.Error(w, "not serving: another control plane pod leads and the backing store isn't shared", http.StatusServiceUnavailable)
			return
		}

		h.ServeHTTP(w, req)
	})
}
