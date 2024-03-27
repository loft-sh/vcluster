package v2

import "testing"

func TestInterceptorPortNonResource(t *testing.T) {
	testCases := []struct {
		desc           string
		path           string
		verb           string
		wantOk         bool
		port           int
		handler        string
		registeredVerb string
		registeredPath string
	}{
		{
			desc:           "wildcard path and verb",
			registeredVerb: "*",
			registeredPath: "*",
			wantOk:         true,
			handler:        "superName",
			port:           12345,
		},
		{
			desc:           "wildcard path and wrong verb",
			registeredVerb: "list",
			registeredPath: "*",
			wantOk:         false,
			handler:        "",
			port:           0,
			verb:           "get",
		},
		{
			desc:           "wildcard path and right verb",
			registeredVerb: "list",
			registeredPath: "*",
			wantOk:         true,
			handler:        "",
			port:           12345,
			verb:           "list",
		},
		{
			desc:           "right path and right verb",
			registeredVerb: "list",
			registeredPath: "/healthz",
			wantOk:         true,
			handler:        "superhandler",
			port:           12345,
			verb:           "list",
			path:           "/healthz",
		},
		{
			desc:           "wrong path and right verb",
			registeredVerb: "list",
			registeredPath: "/healthz",
			wantOk:         false,
			verb:           "list",
			path:           "/doner",
		},
		{
			desc:           "semi wildcard path and right verb",
			registeredVerb: "list",
			registeredPath: "/don*",
			wantOk:         true,
			port:           12345,
			handler:        "donerhandler",
			verb:           "list",
			path:           "/doner",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			m := Manager{
				NonResourceInterceptorsPorts: make(map[string]map[string]portHandlerName),
			}
			m.NonResourceInterceptorsPorts[tC.registeredPath] = make(map[string]portHandlerName)
			m.NonResourceInterceptorsPorts[tC.registeredPath][tC.registeredVerb] = portHandlerName{port: tC.port, handlerName: tC.handler}
			ok, port, handlername := m.InterceptorPortForNonResourceURL(tC.path, tC.verb)
			if ok != tC.wantOk {
				t.Errorf("wanted ok to be %t but got %t ", tC.wantOk, ok)
			}
			if port != tC.port {
				t.Errorf("wanted port to be %d but got %d ", tC.port, port)
			}
			if handlername != tC.handler {
				t.Errorf("wanted handler to be %s but got %s ", tC.handler, handlername)
			}

		})
	}
}
