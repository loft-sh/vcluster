package v2

import (
	"testing"
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
		desc                   string
		resource               string
		resourceName           string
		group                  string
		verb                   string
		wantOk                 bool
		wantPort               int
		wantHandler            string
		registeredHandler      string
		registeredGroup        string
		registeredResource     string
		registeredResourceName string
		registeredVerb         string
		registeredPort         int
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
			desc:               "wildcard group right resource right verb",
			registeredGroup:    "*",
			registeredResource: "doner",
			registeredPort:     12345,
			registeredVerb:     "get",
			resource:           "doner",
			group:              "somegroup",
			verb:               "get",
			wantOk:             true,
			registeredHandler:  "superName",
			wantHandler:        "superName",
			wantPort:           12345,
		},
		{
			desc:               "right group wildcard resource right verb",
			registeredGroup:    "somegroup",
			registeredResource: "*",
			registeredPort:     12345,
			registeredVerb:     "get",
			resource:           "someresource",
			group:              "somegroup",
			verb:               "get",
			wantOk:             true,
			registeredHandler:  "superName",
			wantHandler:        "superName",
			wantPort:           12345,
		},
		{
			desc:                   "right group right resource right verb wrong name",
			registeredGroup:        "somegroup",
			group:                  "somegroup",
			registeredResource:     "someresource",
			resource:               "someresource",
			registeredPort:         0,
			registeredVerb:         "get",
			verb:                   "get",
			wantOk:                 false,
			registeredHandler:      "superName",
			wantHandler:            "",
			wantPort:               0,
			registeredResourceName: "john",
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
				ResourceInterceptorsPorts: make(map[string]map[string]map[string]map[string]portHandlerName),
			}
			m.ResourceInterceptorsPorts[tC.registeredGroup] = make(map[string]map[string]map[string]portHandlerName)
			m.ResourceInterceptorsPorts[tC.registeredGroup][tC.registeredResource] = make(map[string]map[string]portHandlerName)
			m.ResourceInterceptorsPorts[tC.registeredGroup][tC.registeredResource][tC.registeredVerb] = make(map[string]portHandlerName)
			if tC.registeredResourceName == "" {
				// register does it this way
				m.ResourceInterceptorsPorts[tC.registeredGroup][tC.registeredResource][tC.registeredVerb]["*"] = portHandlerName{handlerName: tC.registeredHandler, port: tC.registeredPort}
			} else {
				m.ResourceInterceptorsPorts[tC.registeredGroup][tC.registeredResource][tC.registeredVerb][tC.registeredResourceName] = portHandlerName{handlerName: tC.registeredHandler, port: tC.registeredPort}
			}
			ok, port, handlername := m.InterceptorPortForResource(tC.group, tC.resource, tC.verb, "jeff")
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

func TestValidateInterceptors(t *testing.T) {
	testCases := []struct {
		desc             string
		interceptorInfos Interceptor
		wantErr          bool
	}{
		{
			desc: "valid interceptor",
			interceptorInfos: Interceptor{
				APIGroups: []string{"core"},
				Resources: []string{"pods"},
				Verbs:     []string{"get"},
			},
			wantErr: false,
		},
		{
			desc: "verb and verb wildcard",
			interceptorInfos: Interceptor{
				APIGroups: []string{"core"},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "*"},
			},
			wantErr: true,
		},
		{
			desc: "resource and resource wildcard",
			interceptorInfos: Interceptor{
				APIGroups: []string{"core"},
				Resources: []string{"pods", "*"},
				Verbs:     []string{"get"},
			},
			wantErr: true,
		},
		{
			desc:             "emtpy interceptor",
			interceptorInfos: Interceptor{},
			wantErr:          true,
		},
		{
			desc: "empty group and empty nonresourceURL",
			interceptorInfos: Interceptor{
				Resources: []string{"pods"},
			},
			wantErr: true,
		},
		{
			desc: "empty resource and empty nonresourceURL",
			interceptorInfos: Interceptor{
				APIGroups: []string{"core"},
			},
			wantErr: true,
		},
		{
			desc: "emtpy verbs",
			interceptorInfos: Interceptor{
				APIGroups: []string{"core"},
				Resources: []string{"pods"},
			},
			wantErr: true,
		},
		{
			desc: "only url and verb",
			interceptorInfos: Interceptor{
				NonResourceURLs: []string{"/healthz"},
				Verbs:           []string{"*"},
			},
			wantErr: false,
		},
		{
			desc: "resource group url and verb",
			interceptorInfos: Interceptor{
				NonResourceURLs: []string{"/healthz"},
				APIGroups:       []string{"core"},
				Resources:       []string{"*"},
				Verbs:           []string{"*"},
			},
			wantErr: false,
		},
		{
			desc: "no nonresource and no group",
			interceptorInfos: Interceptor{
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
			wantErr: true,
		},
		{
			desc: "no nonresource and no resource",
			interceptorInfos: Interceptor{
				APIGroups: []string{"*"},
				Verbs:     []string{"*"},
			},
			wantErr: true,
		},
		{
			desc: "wildcard apigroup and more",
			interceptorInfos: Interceptor{
				APIGroups: []string{"*", "core"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
			wantErr: true,
		},
		{
			desc: "wildcard resource and more",
			interceptorInfos: Interceptor{
				Resources: []string{"*", "pods"},
				APIGroups: []string{"*"},
				Verbs:     []string{"*"},
			},
			wantErr: true,
		},
		{
			desc: "non resource url with wrong wildcard",
			interceptorInfos: Interceptor{
				Verbs:           []string{"*"},
				NonResourceURLs: []string{"doner*kebab"},
			},
			wantErr: true,
		},
		{
			desc: "non resource url with right wildcard",
			interceptorInfos: Interceptor{
				Verbs:           []string{"*"},
				NonResourceURLs: []string{"doner*"},
			},
			wantErr: false,
		},
		{
			desc: "non resource url with right wildcard",
			interceptorInfos: Interceptor{
				Verbs:           []string{"*"},
				NonResourceURLs: []string{"*"},
			},
			wantErr: false,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			err := validateInterceptor(tC.interceptorInfos)
			if (tC.wantErr && err == nil) || (!tC.wantErr && err != nil) {
				if err == nil {
					t.Error("error was nil and shouldn't have been")
				} else {
					t.Errorf("unexpected error value %s", err.Error())
				}
			}

		})
	}
}

func TestRegistrationResource(t *testing.T) {
	testCases := []struct {
		desc                      string
		newInterceptorsInfos      Interceptor
		existingInterceptorsInfos Interceptor
		port                      int
		wantErr                   bool
	}{
		{
			desc: "conflict with resource",
			existingInterceptorsInfos: Interceptor{APIGroups: []string{"core"},
				Resources: []string{"pod"},
				Verbs:     []string{"get"},
			},
			newInterceptorsInfos: Interceptor{APIGroups: []string{"core"},
				Resources: []string{"pod"},
				Verbs:     []string{"get"},
			},
			port:    123,
			wantErr: true,
		},
		{
			desc: "no conflict",
			existingInterceptorsInfos: Interceptor{APIGroups: []string{"core"},
				Resources: []string{"service"},
				Verbs:     []string{"get"},
			},
			newInterceptorsInfos: Interceptor{APIGroups: []string{"core"},
				Resources: []string{"pod"},
				Verbs:     []string{"get"},
			},
			port:    123,
			wantErr: false,
		},
		//{
		//	desc: "conflict with wildcards",
		//	existingInterceptorsInfos: Interceptor{APIGroups: []string{"core"},
		//		Resources: []string{"service"},
		//		Verbs:     []string{"get"},
		//	},
		//	newInterceptorsInfos: Interceptor{APIGroups: []string{"*"},
		//		Resources: []string{"pod"},
		//		Verbs:     []string{"get"},
		//	},
		//	port:    123,
		//	wantErr: true,
		//},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			m := &Manager{
				ResourceInterceptorsPorts: make(map[string]map[string]map[string]map[string]portHandlerName),
			}
			err := m.registerResourceInterceptor(tC.port, tC.existingInterceptorsInfos)
			if err != nil {
				t.Errorf("could not put the existing interceptors")
			}

			err = m.registerResourceInterceptor(tC.port, tC.newInterceptorsInfos)
			if (tC.wantErr && err == nil) || (!tC.wantErr && err != nil) {
				if err == nil {
					t.Error("error was nil and shouldn't have been")
				} else {
					t.Errorf("unexpected error value %s", err.Error())
				}
			}

		})
	}
}
