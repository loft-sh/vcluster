package v2

import (
	"testing"

	plugintypes "github.com/loft-sh/vcluster/pkg/plugin/types"
)

func TestInterceptorPortNonResource(t *testing.T) {
	testCases := []struct {
		desc              string
		path              string
		verb              string
		wantOk            bool
		wantHandler       string
		wantPort          int
		registeredPort    int
		registeredHandler string
		registeredVerb    string
		registeredPath    string
	}{
		{
			desc:              "wildcard path and verb",
			registeredVerb:    "*",
			registeredPath:    "*",
			wantOk:            true,
			registeredHandler: "superName",
			wantHandler:       "superName",
			registeredPort:    12345,
			wantPort:          12345,
		},
		{
			desc:              "wildcard path and wrong verb",
			registeredVerb:    "list",
			registeredPath:    "*",
			wantOk:            false,
			registeredHandler: "superName",
			wantHandler:       "",
			registeredPort:    12345,
			wantPort:          0,
			verb:              "get",
		},
		{
			desc:              "wildcard path and right verb",
			registeredVerb:    "list",
			registeredPath:    "*",
			wantOk:            true,
			registeredHandler: "superName",
			wantHandler:       "superName",
			registeredPort:    12345,
			wantPort:          12345,
			verb:              "list",
		},
		{
			desc:              "right path and right verb",
			registeredVerb:    "list",
			registeredPath:    "/healthz",
			wantOk:            true,
			registeredHandler: "superhandler",
			wantHandler:       "superhandler",
			registeredPort:    12345,
			wantPort:          12345,
			verb:              "list",
			path:              "/healthz",
		},
		{
			desc:              "wrong path and right verb",
			registeredVerb:    "list",
			registeredPath:    "/healthz",
			registeredHandler: "superName",
			registeredPort:    12345,
			wantPort:          0,
			wantHandler:       "",
			wantOk:            false,
			verb:              "list",
			path:              "/doner",
		},
		{
			desc:              "semi wildcard path and right verb",
			registeredVerb:    "list",
			registeredPath:    "/don*",
			wantOk:            true,
			registeredPort:    12345,
			wantPort:          12345,
			registeredHandler: "donerhandler",
			wantHandler:       "donerhandler",
			verb:              "list",
			path:              "/doner",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			m := Manager{
				NonResourceInterceptorsPorts: make(map[string]map[string]portHandlerName),
			}
			m.NonResourceInterceptorsPorts[tC.registeredPath] = make(map[string]portHandlerName)
			m.NonResourceInterceptorsPorts[tC.registeredPath][tC.registeredVerb] = portHandlerName{port: tC.registeredPort, handlerName: tC.registeredHandler}
			ok, port, handlername := m.InterceptorPortForNonResourceURL(tC.path, tC.verb)
			if ok != tC.wantOk {
				t.Errorf("wanted ok to be %t but got %t ", tC.wantOk, ok)
			}
			if port != tC.wantPort {
				t.Errorf("wanted port to be %d but got %d ", tC.wantPort, port)
			}
			if handlername != tC.wantHandler {
				t.Errorf("wanted handler to be %s but got %s ", tC.wantHandler, handlername)
			}

		})
	}
}

func TestResourcePort(t *testing.T) {
	testCases := []struct {
		desc               string
		resource           string
		group              string
		verb               string
		wantOk             bool
		wantPort           int
		wantHandler        string
		registeredHandler  string
		registeredGroup    string
		registeredResource string
		registeredVerb     string
		registeredPort     int
	}{
		{
			desc:               "right group and resource wildcard verb",
			registeredGroup:    "",
			group:              "",
			registeredResource: "doner",
			resource:           "doner",
			verb:               "get",
			registeredVerb:     "*",
			wantOk:             true,
			registeredHandler:  "superName",
			wantHandler:        "superName",
			registeredPort:     12345,
			wantPort:           12345,
		},
		{
			desc:               "right group and resource specific verb",
			registeredGroup:    "",
			group:              "",
			registeredResource: "doner",
			registeredPort:     12345,
			resource:           "doner",
			verb:               "get",
			registeredVerb:     "get",
			wantOk:             true,
			wantPort:           12345,
			registeredHandler:  "superName",
			wantHandler:        "superName",
		},
		{
			desc:               "right group wrong resource right verb",
			registeredGroup:    "",
			registeredResource: "doner",
			registeredPort:     12345,
			registeredVerb:     "get",
			resource:           "gyros",
			group:              "",
			verb:               "get",
			wantOk:             false,
			registeredHandler:  "superName",
			wantHandler:        "",
			wantPort:           0,
		},
		{
			desc:               "wrong group right resource right verb",
			registeredGroup:    "",
			registeredResource: "doner",
			registeredPort:     12345,
			registeredVerb:     "get",
			resource:           "doner",
			group:              "kebab",
			verb:               "get",
			wantOk:             false,
			registeredHandler:  "superName",
			wantHandler:        "",
			wantPort:           0,
		},
		{
			desc:               "right group and resource wrong verb",
			registeredGroup:    "",
			registeredResource: "doner",
			registeredPort:     12345,
			registeredVerb:     "list",
			resource:           "doner",
			group:              "",
			verb:               "get",
			wantOk:             false,
			registeredHandler:  "superName",
			wantHandler:        "",
			wantPort:           0,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {

			m := Manager{
				ResourceInterceptorsPorts: make(map[plugintypes.GroupedResource]map[string]portHandlerName),
			}
			resource := plugintypes.GroupedResource{
				Resource: tC.registeredResource,
				APIGroup: tC.registeredGroup,
			}
			m.ResourceInterceptorsPorts[resource] = make(map[string]portHandlerName)
			m.ResourceInterceptorsPorts[resource][tC.registeredVerb] = portHandlerName{port: tC.registeredPort, handlerName: tC.registeredHandler}
			ok, port, handlername := m.InterceptorPortForResource(tC.group, tC.resource, tC.verb)
			if ok != tC.wantOk {
				t.Errorf("wanted ok to be %t but got %t ", tC.wantOk, ok)
			}
			if port != tC.wantPort {
				t.Errorf("wanted port to be %d but got %d ", tC.wantPort, port)
			}
			if handlername != tC.wantHandler {
				t.Errorf("wanted handler to be %s but got %s ", tC.registeredHandler, handlername)
			}
		})
	}
}
