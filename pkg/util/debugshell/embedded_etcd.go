package debugshell

import (
	"fmt"
	"strings"

	"github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// IsEmbeddedEtcdEnabled decodes the vCluster config and checks if embedded etcd is enabled.
func IsEmbeddedEtcdEnabled(rawVClusterConfig []byte) (bool, error) {
	var configValues map[string]any

	if err := yaml.Unmarshal(rawVClusterConfig, &configValues); err != nil {
		return false, err
	}
	enabled, isFound, err := unstructured.NestedBool(configValues, "controlPlane", "backingStore", "etcd", "embedded", "enabled")
	if err != nil {
		return false, err
	}

	return isFound && enabled, nil
}

// AppendEtcdEnvs appends ETCD env vars and enriches the banner env with ETCD tips.
func AppendEtcdEnvs(envs []corev1.EnvVar, bannerEnvName string, vClusterName string, podCount int) []corev1.EnvVar {
	for i := range envs {
		if envs[i].Name == bannerEnvName {
			envs[i].Value += "\n" +
				"etcd environment variables are already configured for you, so you may check the cluster state with etcdctl.\n\n" +
				"Useful etcd debugging commands:\n" +
				"$ etcdctl member list -w table\n" +
				"$ etcdctl endpoint health -w table\n"
		}
	}

	endpoints := BuildEtcdEndpoints(vClusterName, podCount)
	return append(envs, []corev1.EnvVar{
		{
			Name:  "ETCDCTL_CACERT",
			Value: "/proc/1/root/data/pki/etcd/ca.crt",
		},
		{
			Name:  "ETCDCTL_KEY",
			Value: "/proc/1/root/data/pki/etcd/healthcheck-client.key",
		},
		{
			Name:  "ETCDCTL_CERT",
			Value: "/proc/1/root/data/pki/etcd/healthcheck-client.crt",
		},
		{
			Name:  "ETCDCTL_ENDPOINTS",
			Value: endpoints,
		},
	}...)
}

// BuildEtcdEndpoints builds the endpoint list for embedded etcd.
func BuildEtcdEndpoints(vClusterName string, podCount int) string {
	debugEndpoints := "https://localhost:2379"
	if podCount <= 1 {
		return debugEndpoints
	}

	remoteEndpoints := make([]string, 0, podCount)
	for i := 0; i < podCount; i++ {
		remoteEndpoints = append(remoteEndpoints, fmt.Sprintf("https://%s-%d.%s-headless:2379", vClusterName, i, vClusterName))
	}
	return strings.Join(append([]string{debugEndpoints}, remoteEndpoints...), ",")
}
