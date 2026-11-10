package filters

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithLeaderReadiness(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		serving  bool
		wantCode int
	}{
		{name: "readyz while not serving", path: "/readyz", serving: false, wantCode: http.StatusServiceUnavailable},
		{name: "readyz sub check while not serving", path: "/readyz/ping", serving: false, wantCode: http.StatusServiceUnavailable},
		{name: "readyz while serving", path: "/readyz", serving: true, wantCode: http.StatusOK},
		{name: "readyz sub check while serving", path: "/readyz/ping", serving: true, wantCode: http.StatusOK},
		{name: "livez while not serving", path: "/livez", serving: false, wantCode: http.StatusOK},
		{name: "healthz while not serving", path: "/healthz", serving: false, wantCode: http.StatusOK},
		{name: "api request while not serving", path: "/api/v1/namespaces", serving: false, wantCode: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			h := WithLeaderReadiness(next, func() bool { return tt.serving })

			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if rec.Code != tt.wantCode {
				t.Fatalf("%s with serving=%t: got %d, want %d", tt.path, tt.serving, rec.Code, tt.wantCode)
			}
		})
	}
}
