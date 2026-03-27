package kubeclient

import (
	"context"
	"net/http"
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// --- ContextName ---

func TestContextName(t *testing.T) {
	tests := []struct {
		name          string
		vclusterName  string
		namespace     string
		parentContext string
		want          string
	}{
		{
			name:          "basic",
			vclusterName:  "mycluster",
			namespace:     "myns",
			parentContext: "docker-desktop",
			want:          "vcluster_mycluster_myns_docker-desktop",
		},
		{
			name:          "empty parent context",
			vclusterName:  "vc",
			namespace:     "ns",
			parentContext: "",
			want:          "vcluster_vc_ns_",
		},
		{
			name:          "parent context with underscores",
			vclusterName:  "vc",
			namespace:     "ns",
			parentContext: "kind_kind",
			want:          "vcluster_vc_ns_kind_kind",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ContextName(tc.vclusterName, tc.namespace, tc.parentContext)
			assert.Equal(t, got, tc.want)
		})
	}
}

// --- PlatformContextName ---

func TestPlatformContextName(t *testing.T) {
	tests := []struct {
		name          string
		vclusterName  string
		project       string
		parentContext string
		want          string
	}{
		{
			name:          "basic",
			vclusterName:  "mycluster",
			project:       "myproject",
			parentContext: "docker-desktop",
			want:          "vcluster-platform_mycluster_myproject_docker-desktop",
		},
		{
			name:          "empty parent context",
			vclusterName:  "vc",
			project:       "proj",
			parentContext: "",
			want:          "vcluster-platform_vc_proj_",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := PlatformContextName(tc.vclusterName, tc.project, tc.parentContext)
			assert.Equal(t, got, tc.want)
		})
	}
}

// --- SpaceContextName ---

func TestSpaceContextName(t *testing.T) {
	tests := []struct {
		name          string
		clusterName   string
		namespaceName string
		want          string
	}{
		{
			name:          "with namespace",
			clusterName:   "mycluster",
			namespaceName: "myns",
			want:          "vcluster-platform_myns_mycluster",
		},
		{
			name:          "without namespace",
			clusterName:   "mycluster",
			namespaceName: "",
			want:          "vcluster-platform_mycluster",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SpaceContextName(tc.clusterName, tc.namespaceName)
			assert.Equal(t, got, tc.want)
		})
	}
}

// --- SpaceInstanceContextName ---

func TestSpaceInstanceContextName(t *testing.T) {
	tests := []struct {
		name              string
		projectName       string
		spaceInstanceName string
		want              string
	}{
		{
			name:              "basic",
			projectName:       "myproject",
			spaceInstanceName: "myspace",
			want:              "vcluster-platform_myspace_myproject",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SpaceInstanceContextName(tc.projectName, tc.spaceInstanceName)
			assert.Equal(t, got, tc.want)
		})
	}
}

// --- VirtualClusterInstanceContextName ---

func TestVirtualClusterInstanceContextName(t *testing.T) {
	tests := []struct {
		name                   string
		projectName            string
		virtualClusterInstance string
		want                   string
	}{
		{
			name:                   "basic",
			projectName:            "myproject",
			virtualClusterInstance: "myvc",
			want:                   "vcluster-platform-vcluster_myvc_myproject",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := VirtualClusterInstanceContextName(tc.projectName, tc.virtualClusterInstance)
			assert.Equal(t, got, tc.want)
		})
	}
}

// --- ManagementContextName ---

func TestManagementContextName(t *testing.T) {
	assert.Equal(t, ManagementContextName(), "vcluster-platform_management")
}

// --- BackgroundProxyName ---

func TestBackgroundProxyName(t *testing.T) {
	tests := []struct {
		name          string
		vclusterName  string
		namespace     string
		parentContext string
		want          string
	}{
		{
			name:          "clean inputs",
			vclusterName:  "mycluster",
			namespace:     "myns",
			parentContext: "docker-desktop",
			// ContextName = "vcluster_mycluster_myns_docker-desktop"
			// + "_background_proxy" = "vcluster_mycluster_myns_docker-desktop_background_proxy"
			// strip [^a-zA-Z0-9\-_] → no change (already clean)
			want: "vcluster_mycluster_myns_docker-desktop_background_proxy",
		},
		{
			name:          "parent context with dots",
			vclusterName:  "vc",
			namespace:     "ns",
			parentContext: "kind.cluster.local",
			// "vcluster_vc_ns_kind.cluster.local_background_proxy"
			// dots stripped → "vcluster_vc_ns_kindclusterlocal_background_proxy"
			want: "vcluster_vc_ns_kindclusterlocal_background_proxy",
		},
		{
			name:          "parent context with spaces",
			vclusterName:  "vc",
			namespace:     "ns",
			parentContext: "my context",
			// "vcluster_vc_ns_my context_background_proxy"
			// space stripped → "vcluster_vc_ns_mycontext_background_proxy"
			want: "vcluster_vc_ns_mycontext_background_proxy",
		},
		{
			name:          "special chars in vcluster name",
			vclusterName:  "my.vc",
			namespace:     "ns",
			parentContext: "ctx",
			// "vcluster_my.vc_ns_ctx_background_proxy"
			// dot stripped → "vcluster_myvc_ns_ctx_background_proxy"
			want: "vcluster_myvc_ns_ctx_background_proxy",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := BackgroundProxyName(tc.vclusterName, tc.namespace, tc.parentContext)
			assert.Equal(t, got, tc.want)
		})
	}
}

// --- FromContext ---

func TestFromContext(t *testing.T) {
	tests := []struct {
		name            string
		originalContext string
		wantName        string
		wantNamespace   string
		wantParent      string
	}{
		{
			name:            "valid vcluster context",
			originalContext: "vcluster_mycluster_myns_docker-desktop",
			wantName:        "mycluster",
			wantNamespace:   "myns",
			wantParent:      "docker-desktop",
		},
		{
			name:            "parent context with underscores",
			originalContext: "vcluster_vc_ns_kind_kind",
			wantName:        "vc",
			wantNamespace:   "ns",
			wantParent:      "kind_kind",
		},
		{
			name:            "not a vcluster context",
			originalContext: "docker-desktop",
			wantName:        "",
			wantNamespace:   "",
			wantParent:      "",
		},
		{
			name:            "platform context rejected",
			originalContext: "vcluster-platform_vc_proj_ctx",
			wantName:        "",
			wantNamespace:   "",
			wantParent:      "",
		},
		{
			name:            "too few parts returns original",
			originalContext: "vcluster_vc_ns",
			wantName:        "vcluster_vc_ns",
			wantNamespace:   "",
			wantParent:      "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			name, ns, parent := FromContext(tc.originalContext)
			assert.Equal(t, name, tc.wantName)
			assert.Equal(t, ns, tc.wantNamespace)
			assert.Equal(t, parent, tc.wantParent)
		})
	}
}

// --- PlatformFromContext ---

func TestPlatformFromContext(t *testing.T) {
	tests := []struct {
		name            string
		originalContext string
		wantName        string
		wantProject     string
		wantParent      string
	}{
		{
			name:            "valid platform context",
			originalContext: "vcluster-platform_mycluster_myproject_docker-desktop",
			wantName:        "mycluster",
			wantProject:     "myproject",
			wantParent:      "docker-desktop",
		},
		{
			name:            "parent context with underscores",
			originalContext: "vcluster-platform_vc_proj_kind_kind",
			wantName:        "vc",
			wantProject:     "proj",
			wantParent:      "kind_kind",
		},
		{
			name:            "not a platform context",
			originalContext: "docker-desktop",
			wantName:        "",
			wantProject:     "",
			wantParent:      "",
		},
		{
			name:            "standalone vcluster context rejected",
			originalContext: "vcluster_vc_ns_ctx",
			wantName:        "",
			wantProject:     "",
			wantParent:      "",
		},
		{
			name:            "too few parts returns original",
			originalContext: "vcluster-platform_vc_proj",
			wantName:        "vcluster-platform_vc_proj",
			wantProject:     "",
			wantParent:      "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			name, project, parent := PlatformFromContext(tc.originalContext)
			assert.Equal(t, name, tc.wantName)
			assert.Equal(t, project, tc.wantProject)
			assert.Equal(t, parent, tc.wantParent)
		})
	}
}

// --- DockerFromContext ---

func TestDockerFromContext(t *testing.T) {
	tests := []struct {
		name            string
		originalContext string
		wantName        string
	}{
		{
			name:            "valid docker context",
			originalContext: "vcluster-docker_mycluster",
			wantName:        "mycluster",
		},
		{
			name:            "not a docker context",
			originalContext: "docker-desktop",
			wantName:        "",
		},
		{
			name:            "vcluster standalone context rejected",
			originalContext: "vcluster_vc_ns_ctx",
			wantName:        "",
		},
		{
			name:            "too many parts returns empty",
			originalContext: "vcluster-docker_vc_extra",
			wantName:        "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := DockerFromContext(tc.originalContext)
			assert.Equal(t, got, tc.wantName)
		})
	}
}

// --- WithWrapTransport ---

func TestWithWrapTransport(t *testing.T) {
	t.Run("single wrapper is applied", func(t *testing.T) {
		called := false
		wrapper := func(rt http.RoundTripper) http.RoundTripper {
			called = true
			return rt
		}

		o := applyOptions([]Option{WithWrapTransport(wrapper)})
		assert.Assert(t, o.wrapTransport != nil)

		// invoke the wrapper
		o.wrapTransport(http.DefaultTransport)
		assert.Assert(t, called)
	})

	t.Run("two wrappers compose with inner applied first", func(t *testing.T) {
		var order []string

		first := func(rt http.RoundTripper) http.RoundTripper {
			order = append(order, "first")
			return rt
		}
		second := func(rt http.RoundTripper) http.RoundTripper {
			order = append(order, "second")
			return rt
		}

		o := applyOptions([]Option{
			WithWrapTransport(first),
			WithWrapTransport(second),
		})

		o.wrapTransport(http.DefaultTransport)
		// first is the "prior", second wraps it — second(first(rt))
		// execution order: first is called first, then second wraps the result
		assert.DeepEqual(t, order, []string{"first", "second"})
	})
}

// --- GetDefaultSecretName ---

func TestGetDefaultSecretName(t *testing.T) {
	tests := []struct {
		suffix string
		want   string
	}{
		{suffix: "mycluster", want: "vc-mycluster"},
		{suffix: "", want: "vc-"},
		{suffix: "a-b-c", want: "vc-a-b-c"},
	}

	for _, tc := range tests {
		t.Run(tc.suffix, func(t *testing.T) {
			got := GetDefaultSecretName(tc.suffix)
			assert.Equal(t, got, tc.want)
		})
	}
}

// --- ReadKubeConfig ---

func minimalKubeconfig(t *testing.T) []byte {
	t.Helper()
	cfg := clientcmdapi.NewConfig()
	cfg.CurrentContext = "test-ctx"
	cfg.Clusters["test-ctx"] = &clientcmdapi.Cluster{Server: "https://localhost:6443"}
	cfg.AuthInfos["test-ctx"] = clientcmdapi.NewAuthInfo()
	cfg.Contexts["test-ctx"] = &clientcmdapi.Context{
		Cluster:  "test-ctx",
		AuthInfo: "test-ctx",
	}
	data, err := clientcmd.Write(*cfg)
	assert.NilError(t, err)
	return data
}

func TestReadKubeConfig(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		kubeconfigData := minimalKubeconfig(t)
		secretName := GetDefaultSecretName("mycluster")
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: "myns",
			},
			Data: map[string][]byte{
				KubeconfigSecretKey: kubeconfigData,
			},
		}

		client := fakeclientset.NewClientset(secret)
		cfg, err := ReadKubeConfig(context.Background(), client, "mycluster", "myns")
		assert.NilError(t, err)
		assert.Equal(t, cfg.CurrentContext, "test-ctx")
	})

	t.Run("secret not found", func(t *testing.T) {
		client := fakeclientset.NewClientset()
		_, err := ReadKubeConfig(context.Background(), client, "mycluster", "myns")
		assert.ErrorContains(t, err, "vc-mycluster")
	})

	t.Run("key missing in secret", func(t *testing.T) {
		secretName := GetDefaultSecretName("mycluster")
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: "myns",
			},
			Data: map[string][]byte{
				"wrong-key": []byte("data"),
			},
		}

		client := fakeclientset.NewClientset(secret)
		_, err := ReadKubeConfig(context.Background(), client, "mycluster", "myns")
		assert.ErrorContains(t, err, KubeconfigSecretKey)
	})
}

// --- NewVClusterClient ---

func buildTestClientConfig(contextName string) clientcmd.ClientConfig {
	rawCfg := clientcmdapi.NewConfig()
	rawCfg.CurrentContext = contextName
	rawCfg.Clusters[contextName] = &clientcmdapi.Cluster{
		Server: "https://localhost:6443",
	}
	rawCfg.AuthInfos[contextName] = clientcmdapi.NewAuthInfo()
	rawCfg.Contexts[contextName] = &clientcmdapi.Context{
		Cluster:  contextName,
		AuthInfo: contextName,
	}
	return clientcmd.NewDefaultClientConfig(*rawCfg, &clientcmd.ConfigOverrides{})
}

func TestNewVClusterClient(t *testing.T) {
	t.Run("with explicit context name", func(t *testing.T) {
		cfg := buildTestClientConfig("my-context")
		client, err := NewVClusterClient(cfg, "my-context")
		assert.NilError(t, err)
		assert.Assert(t, client != nil)
	})

	t.Run("with empty context name uses current context", func(t *testing.T) {
		cfg := buildTestClientConfig("my-context")
		client, err := NewVClusterClient(cfg, "")
		assert.NilError(t, err)
		assert.Assert(t, client != nil)
	})

	t.Run("unknown context name returns error", func(t *testing.T) {
		cfg := buildTestClientConfig("my-context")
		_, err := NewVClusterClient(cfg, "nonexistent-context")
		assert.ErrorContains(t, err, "nonexistent-context")
	})
}

// --- NewVClusterClientFromConfig ---

func TestNewVClusterClientFromConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		contextName := "test-ctx"
		rawCfg := clientcmdapi.NewConfig()
		rawCfg.CurrentContext = contextName
		rawCfg.Clusters[contextName] = &clientcmdapi.Cluster{
			Server: "https://localhost:6443",
		}
		rawCfg.AuthInfos[contextName] = clientcmdapi.NewAuthInfo()
		rawCfg.Contexts[contextName] = &clientcmdapi.Context{
			Cluster:  contextName,
			AuthInfo: contextName,
		}

		client, err := NewVClusterClientFromConfig(*rawCfg)
		assert.NilError(t, err)
		assert.Assert(t, client != nil)
	})
}

// --- ConvertRestConfigToClientConfig ---

func TestConvertRestConfigToClientConfig(t *testing.T) {
	t.Run("host and bearer token", func(t *testing.T) {
		restCfg := &rest.Config{
			Host:        "https://localhost:6443",
			BearerToken: "my-token",
		}

		clientCfg, err := ConvertRestConfigToClientConfig(restCfg)
		assert.NilError(t, err)

		restOut, err := clientCfg.ClientConfig()
		assert.NilError(t, err)
		assert.Equal(t, restOut.Host, "https://localhost:6443")
		assert.Equal(t, restOut.BearerToken, "my-token")
	})

	t.Run("inline CA data and client cert/key", func(t *testing.T) {
		restCfg := &rest.Config{
			Host:        "https://localhost:6443",
			BearerToken: "token",
			TLSClientConfig: rest.TLSClientConfig{
				CAData:   []byte("ca-data"),
				CertData: []byte("cert-data"),
				KeyData:  []byte("key-data"),
			},
		}

		clientCfg, err := ConvertRestConfigToClientConfig(restCfg)
		assert.NilError(t, err)

		rawCfg, err := clientCfg.RawConfig()
		assert.NilError(t, err)

		cluster := rawCfg.Clusters["local"]
		assert.Assert(t, cluster != nil)
		assert.DeepEqual(t, cluster.CertificateAuthorityData, []byte("ca-data"))

		authInfo := rawCfg.AuthInfos["local"]
		assert.Assert(t, authInfo != nil)
		assert.DeepEqual(t, authInfo.ClientCertificateData, []byte("cert-data"))
		assert.DeepEqual(t, authInfo.ClientKeyData, []byte("key-data"))
	})

	t.Run("current context is local", func(t *testing.T) {
		restCfg := &rest.Config{
			Host:        "https://localhost:6443",
			BearerToken: "token",
		}

		clientCfg, err := ConvertRestConfigToClientConfig(restCfg)
		assert.NilError(t, err)

		rawCfg, err := clientCfg.RawConfig()
		assert.NilError(t, err)
		assert.Equal(t, rawCfg.CurrentContext, "local")
	})
}
