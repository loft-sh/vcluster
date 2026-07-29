package setup

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/loft-sh/vcluster/config"
)

func TestExportKubeConfig(t *testing.T) {
	const (
		testCluster          = "test-cluster"
		testControlPlanePort = 8443
		testContext          = "test-context"
		testUser             = "test-user"
		exportedTestContext  = "exported-test-context"
		exportedTestServer   = "exported-test-server"
	)

	testControlPlaneProxy := config.ControlPlaneProxy{
		Port: testControlPlanePort,
	}

	cases := []struct {
		name               string
		syncerConfig       clientcmdapi.Config
		options            CreateKubeConfigOptions
		expectedKubeConfig clientcmdapi.Config
	}{
		{
			name: "Export default kubeconfig",
			syncerConfig: clientcmdapi.Config{
				Clusters: map[string]*clientcmdapi.Cluster{
					testCluster: {},
				},
			},
			options: CreateKubeConfigOptions{
				ControlPlaneProxy: testControlPlaneProxy,
			},
			expectedKubeConfig: clientcmdapi.Config{
				Clusters: map[string]*clientcmdapi.Cluster{
					testCluster: {
						Server: fmt.Sprintf("https://localhost:%d", testControlPlanePort),
					},
				},
			},
		},
		{
			name: "Export default kubeconfig with custom server",
			syncerConfig: clientcmdapi.Config{
				Clusters: map[string]*clientcmdapi.Cluster{
					testCluster: {},
				},
			},
			options: CreateKubeConfigOptions{
				ControlPlaneProxy: testControlPlaneProxy,
				ExportKubeConfig: config.ExportKubeConfigProperties{
					Server: exportedTestServer,
				},
			},
			expectedKubeConfig: clientcmdapi.Config{
				Clusters: map[string]*clientcmdapi.Cluster{
					testCluster: {
						Server:     exportedTestServer,
						Extensions: map[string]runtime.Object{},
					},
				},
			},
		},
		{
			name: "Export default kubeconfig with custom context",
			syncerConfig: clientcmdapi.Config{
				CurrentContext: testContext,
				AuthInfos: map[string]*clientcmdapi.AuthInfo{
					testUser: {},
				},
				Clusters: map[string]*clientcmdapi.Cluster{
					testCluster: {},
				},
				Contexts: map[string]*clientcmdapi.Context{
					testContext: {
						Cluster:  testCluster,
						AuthInfo: testUser,
					},
				},
			},
			options: CreateKubeConfigOptions{
				ControlPlaneProxy: testControlPlaneProxy,
				ExportKubeConfig: config.ExportKubeConfigProperties{
					Context: exportedTestContext,
				},
			},
			expectedKubeConfig: clientcmdapi.Config{
				CurrentContext: exportedTestContext,
				AuthInfos: map[string]*clientcmdapi.AuthInfo{
					exportedTestContext: {},
				},
				Clusters: map[string]*clientcmdapi.Cluster{
					exportedTestContext: {
						Server: fmt.Sprintf("https://localhost:%d", testControlPlanePort),
					},
				},
				Contexts: map[string]*clientcmdapi.Context{
					exportedTestContext: {
						Cluster:  exportedTestContext,
						AuthInfo: exportedTestContext,
					},
				},
			},
		},
		{
			name: "Export default kubeconfig with custom context and server",
			syncerConfig: clientcmdapi.Config{
				CurrentContext: testContext,
				AuthInfos: map[string]*clientcmdapi.AuthInfo{
					testUser: {},
				},
				Clusters: map[string]*clientcmdapi.Cluster{
					testCluster: {},
				},
				Contexts: map[string]*clientcmdapi.Context{
					testContext: {
						Cluster:  testCluster,
						AuthInfo: testUser,
					},
				},
			},
			options: CreateKubeConfigOptions{
				ControlPlaneProxy: testControlPlaneProxy,
				ExportKubeConfig: config.ExportKubeConfigProperties{
					Context: exportedTestContext,
					Server:  exportedTestServer,
				},
			},
			expectedKubeConfig: clientcmdapi.Config{
				CurrentContext: exportedTestContext,
				AuthInfos: map[string]*clientcmdapi.AuthInfo{
					exportedTestContext: {},
				},
				Clusters: map[string]*clientcmdapi.Cluster{
					exportedTestContext: {
						Server:     exportedTestServer,
						Extensions: map[string]runtime.Object{},
					},
				},
				Contexts: map[string]*clientcmdapi.Context{
					exportedTestContext: {
						Cluster:  exportedTestContext,
						AuthInfo: exportedTestContext,
					},
				},
			},
		},
		{
			name: "Export default kubeconfig with insecure",
			syncerConfig: clientcmdapi.Config{
				CurrentContext: testContext,
				AuthInfos: map[string]*clientcmdapi.AuthInfo{
					testUser: {},
				},
				Clusters: map[string]*clientcmdapi.Cluster{
					testCluster: {
						CertificateAuthorityData: []byte("test-ca"),
					},
				},
				Contexts: map[string]*clientcmdapi.Context{
					testContext: {
						Cluster:  testCluster,
						AuthInfo: testUser,
					},
				},
			},
			options: CreateKubeConfigOptions{
				ControlPlaneProxy: testControlPlaneProxy,
				ExportKubeConfig: config.ExportKubeConfigProperties{
					Insecure: true,
				},
			},
			expectedKubeConfig: clientcmdapi.Config{
				CurrentContext: testContext,
				AuthInfos: map[string]*clientcmdapi.AuthInfo{
					testUser: {},
				},
				Clusters: map[string]*clientcmdapi.Cluster{
					testCluster: {
						Server:                fmt.Sprintf("https://localhost:%d", testControlPlanePort),
						InsecureSkipTLSVerify: true,
					},
				},
				Contexts: map[string]*clientcmdapi.Context{
					testContext: {
						Cluster:  testCluster,
						AuthInfo: testUser,
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			virtualConfig := &rest.Config{}

			kubeConfigResult, err := CreateVClusterKubeConfigForExport(ctx, virtualConfig, &tc.syncerConfig, tc.options)
			if err != nil {
				t.Errorf("Unexpected error when creating kubeconfig for export: %v", err)
			}

			if !reflect.DeepEqual(*kubeConfigResult, tc.expectedKubeConfig) {
				t.Errorf("Unexpected kubeconfig, here is the diff: %s", cmp.Diff(tc.expectedKubeConfig, kubeConfigResult))
			}
		})
	}
}
