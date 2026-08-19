package cli

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"gotest.tools/v3/assert"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const (
	defaultContextName = "my-vcluster"
)

func TestExchangeContextName(t *testing.T) {
	vclusterName := "vcluster-name"
	namespace := "default"
	defaultContext := &clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			defaultContextName: {Server: "foo"},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			defaultContextName: {
				Token: "foo",
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			defaultContextName: {
				Cluster:  defaultContextName,
				AuthInfo: defaultContextName,
			},
		},
		CurrentContext: defaultContextName,
	}
	testTable := []struct {
		config         *clientcmdapi.Config
		expectedConfig *clientcmdapi.Config
		desc           string
		newContextName string
	}{
		{
			desc:           "KubeConfigContextName specified",
			newContextName: "new-context",
			config:         defaultContext.DeepCopy(),
			expectedConfig: &clientcmdapi.Config{
				Clusters: map[string]*clientcmdapi.Cluster{
					"new-context": {Server: "foo"},
				},
				AuthInfos: map[string]*clientcmdapi.AuthInfo{
					"new-context": {
						Token: "foo",
					},
				},
				Contexts: map[string]*clientcmdapi.Context{
					"new-context": {
						Cluster:  "new-context",
						AuthInfo: "new-context",
					},
				},
				CurrentContext: "new-context",
			},
		},
		{
			desc:           "KubeConfigContextName same as default",
			newContextName: defaultContextName,
			config:         defaultContext.DeepCopy(),
			expectedConfig: &clientcmdapi.Config{
				Clusters: map[string]*clientcmdapi.Cluster{
					defaultContextName: {Server: "foo"},
				},
				AuthInfos: map[string]*clientcmdapi.AuthInfo{
					defaultContextName: {
						Token: "foo",
					},
				},
				Contexts: map[string]*clientcmdapi.Context{
					defaultContextName: {
						Cluster:  defaultContextName,
						AuthInfo: defaultContextName,
					},
				},
				CurrentContext: defaultContextName,
			},
		},
	}

	for _, testCase := range testTable {
		cmd := &connectHelm{
			GlobalFlags: &flags.GlobalFlags{
				Namespace: namespace,
			},
			ConnectOptions: &ConnectOptions{
				KubeConfigContextName: testCase.newContextName,
			},
		}
		assert.NilError(t, cmd.exchangeContextName(testCase.config, vclusterName))
		newConfig := testCase.config.DeepCopy()

		assert.DeepEqual(t, newConfig, testCase.expectedConfig)
	}
}

func TestPortForwardServer(t *testing.T) {
	tests := []struct {
		name           string
		server         string
		localPort      int
		expectedServer string
		expectedPort   string
	}{
		{
			name:           "server with explicit port",
			server:         "https://example.com:443",
			localPort:      10443,
			expectedServer: "https://example.com:10443",
			expectedPort:   "443",
		},
		{
			name:           "server without explicit port",
			server:         "https://example.com",
			localPort:      10443,
			expectedServer: "https://example.com:10443",
			expectedPort:   "443",
		},
		{
			name:           "http server without explicit port",
			server:         "http://example.com",
			localPort:      10080,
			expectedServer: "http://example.com:10080",
			expectedPort:   "80",
		},
		{
			name:           "ipv6 server with explicit port",
			server:         "https://[2001:db8::1]:9443",
			localPort:      10443,
			expectedServer: "https://[2001:db8::1]:10443",
			expectedPort:   "9443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, port, err := portForwardServer(tt.server, tt.localPort)
			assert.NilError(t, err)
			assert.Equal(t, server, tt.expectedServer)
			assert.Equal(t, port, tt.expectedPort)
		})
	}
}

func TestPortForwardServerInvalid(t *testing.T) {
	_, _, err := portForwardServer("example.com", 10443)
	assert.Error(t, err, "unexpected server in kubeconfig: example.com")
}
