package k8s

import (
	"testing"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/util/command"
	"gotest.tools/assert"
)

func TestAppendClusterSigningArgsUsesGenericSignerForNonSplitCA(t *testing.T) {
	kubeConfig := vclusterconfig.VirtualClusterKubeConfig{
		ServerCACert: "/custom/ca.crt",
		ServerCAKey:  "/custom/ca.key",
		ClientCACert: "/custom/ca.crt",
	}

	args := appendClusterSigningArgs(nil, kubeConfig, false)

	assertHasArgs(t, args,
		"--cluster-signing-cert-file=/custom/ca.crt",
		"--cluster-signing-key-file=/custom/ca.key",
	)
	assertMissingFlags(t, args, perSignerClusterSigningFlags...)
}

func TestAppendClusterSigningArgsUsesSplitCASigners(t *testing.T) {
	kubeConfig := vclusterconfig.VirtualClusterKubeConfig{
		ServerCACert: "/data/pki/server-ca.crt",
		ServerCAKey:  "/data/pki/server-ca.key",
		ClientCACert: "/data/pki/client-ca.crt",
		ClientCAKey:  "/data/pki/client-ca.key",
	}

	args := appendClusterSigningArgs(nil, kubeConfig, true)

	assertMissingFlags(t, args, genericClusterSigningFlags...)
	assertHasArgs(t, args,
		"--cluster-signing-kube-apiserver-client-cert-file=/data/pki/client-ca.crt",
		"--cluster-signing-kube-apiserver-client-key-file=/data/pki/client-ca.key",
		"--cluster-signing-kubelet-client-cert-file=/data/pki/client-ca.crt",
		"--cluster-signing-kubelet-client-key-file=/data/pki/client-ca.key",
		"--cluster-signing-kubelet-serving-cert-file=/data/pki/server-ca.crt",
		"--cluster-signing-kubelet-serving-key-file=/data/pki/server-ca.key",
		"--cluster-signing-legacy-unknown-cert-file=/data/pki/server-ca.crt",
		"--cluster-signing-legacy-unknown-key-file=/data/pki/server-ca.key",
	)
}

func TestAppendClusterSigningArgsUsesCustomClientCAKeyForCustomSplitCA(t *testing.T) {
	kubeConfig := vclusterconfig.VirtualClusterKubeConfig{
		ServerCACert: "/custom/server-ca.crt",
		ServerCAKey:  "/custom/server-ca.key",
		ClientCACert: "/custom/client-ca.crt",
		ClientCAKey:  "/custom/client-ca.key",
	}

	args := appendClusterSigningArgs(nil, kubeConfig, true)

	assertMissingFlags(t, args, genericClusterSigningFlags...)
	assertHasArgs(t, args,
		"--cluster-signing-kube-apiserver-client-cert-file=/custom/client-ca.crt",
		"--cluster-signing-kube-apiserver-client-key-file=/custom/client-ca.key",
		"--cluster-signing-kubelet-client-cert-file=/custom/client-ca.crt",
		"--cluster-signing-kubelet-client-key-file=/custom/client-ca.key",
		"--cluster-signing-kubelet-serving-cert-file=/custom/server-ca.crt",
		"--cluster-signing-kubelet-serving-key-file=/custom/server-ca.key",
		"--cluster-signing-legacy-unknown-cert-file=/custom/server-ca.crt",
		"--cluster-signing-legacy-unknown-key-file=/custom/server-ca.key",
	)
}

func TestHasClusterSigningFileArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{
			name:     "generic cert file",
			args:     []string{"--cluster-signing-cert-file=/custom/ca.crt"},
			expected: true,
		},
		{
			name:     "generic key file with split value",
			args:     []string{"--cluster-signing-key-file", "/custom/ca.key"},
			expected: true,
		},
		{
			name:     "per-signer cert file",
			args:     []string{"--cluster-signing-kube-apiserver-client-cert-file=/custom/client-ca.crt"},
			expected: true,
		},
		{
			name:     "per-signer key file",
			args:     []string{"--cluster-signing-kubelet-serving-key-file=/custom/server-ca.key"},
			expected: true,
		},
		{
			name:     "duration does not override signing files",
			args:     []string{"--cluster-signing-duration=24h", "--profiling=true"},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, hasClusterSigningFileArgs(test.args), test.expected)
		})
	}
}

func TestGenericClusterSigningExtraArgsSkipGeneratedSigningArgs(t *testing.T) {
	kubeConfig := vclusterconfig.VirtualClusterKubeConfig{
		ServerCACert: "/data/pki/server-ca.crt",
		ServerCAKey:  "/data/pki/server-ca.key",
		ClientCACert: "/data/pki/client-ca.crt",
		ClientCAKey:  "/data/pki/client-ca.key",
	}
	extraArgs := []string{
		"--cluster-signing-cert-file=/custom/ca.crt",
		"--cluster-signing-key-file=/custom/ca.key",
	}

	args := []string{}
	if !hasClusterSigningFileArgs(extraArgs) {
		args = appendClusterSigningArgs(args, kubeConfig, true)
	}
	args = command.MergeArgs(args, extraArgs)

	assertMissingFlags(t, args, perSignerClusterSigningFlags...)
	assertHasArgs(t, args,
		"--cluster-signing-cert-file=/custom/ca.crt",
		"--cluster-signing-key-file=/custom/ca.key",
	)
}

var genericClusterSigningFlags = []string{
	"--cluster-signing-cert-file",
	"--cluster-signing-key-file",
}

var perSignerClusterSigningFlags = []string{
	"--cluster-signing-kube-apiserver-client-cert-file",
	"--cluster-signing-kube-apiserver-client-key-file",
	"--cluster-signing-kubelet-client-cert-file",
	"--cluster-signing-kubelet-client-key-file",
	"--cluster-signing-kubelet-serving-cert-file",
	"--cluster-signing-kubelet-serving-key-file",
	"--cluster-signing-legacy-unknown-cert-file",
	"--cluster-signing-legacy-unknown-key-file",
}

func assertHasArgs(t *testing.T, args []string, expected ...string) {
	t.Helper()
	for _, expectedArg := range expected {
		assert.Assert(t, hasArg(args, expectedArg), "missing arg %s", expectedArg)
	}
}

func assertMissingFlags(t *testing.T, args []string, flags ...string) {
	t.Helper()
	for _, flag := range flags {
		assert.Assert(t, !command.ContainsFlag(args, flag), "unexpected flag %s", flag)
	}
}

func hasArg(args []string, expected string) bool {
	for _, arg := range args {
		if arg == expected {
			return true
		}
	}

	return false
}
