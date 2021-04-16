package serverhelper

import (
	"net/http"
	"strings"
)

// HandleRoute handles a route and strips off the route path
func HandleRoute(mux *http.ServeMux, route string, handler http.Handler) {
	mux.Handle(route, StripLeaveSlash(route, handler))
}

// like http.StripPrefix, but always leaves an initial slash. (so that our
// regexps will work.)
func StripLeaveSlash(prefix string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		p := strings.TrimPrefix(req.URL.Path, prefix)
		if len(p) >= len(req.URL.Path) {
			http.NotFound(w, req)
			return
		}
		if len(p) > 0 && p[:1] != "/" {
			p = "/" + p
		}
		req.URL.Path = p
		h.ServeHTTP(w, req)
	})
}
