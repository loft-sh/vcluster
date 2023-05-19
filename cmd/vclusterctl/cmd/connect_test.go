package cmd

import (
	"testing"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"gotest.tools/v3/assert"
	"k8s.io/client-go/tools/clientcmd/api"
)

const (
	defaultContextName = "my-vcluster"
)

func TestExchangeContextName(t *testing.T) {
	vclusterName := "vcluster-name"
	namespace := "default"
	defaultContext := &api.Config{
		Clusters: map[string]*api.Cluster{
			defaultContextName: {Server: "foo"},
		},
		AuthInfos: map[string]*api.AuthInfo{
			defaultContextName: {
				Token: "foo",
			},
		},
		Contexts: map[string]*api.Context{
			defaultContextName: {
				Cluster:  defaultContextName,
				AuthInfo: defaultContextName,
			},
		},
		CurrentContext: defaultContextName,
	}
	testTable := []struct {
		desc           string
		newContextName string
		config         *api.Config
		expectedConfig *api.Config
	}{
		{
			desc:           "KubeConfigContextName specified",
			newContextName: "new-context",
			config:         defaultContext.DeepCopy(),
			expectedConfig: &api.Config{
				Clusters: map[string]*api.Cluster{
					"new-context": {Server: "foo"},
				},
				AuthInfos: map[string]*api.AuthInfo{
					"new-context": {
						Token: "foo",
					},
				},
				Contexts: map[string]*api.Context{
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
			expectedConfig: &api.Config{
				Clusters: map[string]*api.Cluster{
					defaultContextName: &api.Cluster{Server: "foo"},
				},
				AuthInfos: map[string]*api.AuthInfo{
					defaultContextName: &api.AuthInfo{
						Token: "foo",
					},
				},
				Contexts: map[string]*api.Context{
					defaultContextName: &api.Context{
						Cluster:  defaultContextName,
						AuthInfo: defaultContextName,
					},
				},
				CurrentContext: defaultContextName,
			},
		},
	}
	for _, testCase := range testTable {
		cmd := &ConnectCmd{
			GlobalFlags: &flags.GlobalFlags{
				Namespace: namespace,
			},
			KubeConfigContextName: testCase.newContextName,
		}
		assert.NilError(t, cmd.exchangeContextName(testCase.config, vclusterName))
		newConfig := testCase.config.DeepCopy()

		assert.DeepEqual(t, newConfig, testCase.expectedConfig)
	}
}
