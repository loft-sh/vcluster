package filters

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"strings"
)

func WithPprof(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		prefix := "/debug/pprof/"
		if strings.HasPrefix(req.URL.Path, prefix) {
			switch req.URL.Path {
			case fmt.Sprintf("%scmdline", prefix):
				pprof.Cmdline(w, req)
			case fmt.Sprintf("%sprofile", prefix):
				pprof.Profile(w, req)
			case fmt.Sprintf("%strace", prefix):
				pprof.Trace(w, req)
			case fmt.Sprintf("%ssymbol", prefix):
				pprof.Symbol(w, req)
			default:
				pprof.Index(w, req)
			}
			return
		}

		h.ServeHTTP(w, req)
	})
}
