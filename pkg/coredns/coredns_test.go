package coredns

import (
	"strings"
	"testing"

	"github.com/loft-sh/vcluster/config"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/version"
)

func TestGetManifestVariablesImage(t *testing.T) {
	tests := []struct {
		name                 string
		major                string
		minor                string
		defaultImageRegistry string
		expectedImage        string
	}{
		{
			name:          "known version is fully qualified with docker.io",
			major:         "1",
			minor:         "34",
			expectedImage: "docker.io/coredns/coredns:1.12.1",
		},
		{
			name:          "unknown version falls back to fully qualified default image",
			major:         "1",
			minor:         "99",
			expectedImage: "docker.io/" + DefaultImage,
		},
		{
			name:                 "custom registry takes precedence over docker.io",
			major:                "1",
			minor:                "34",
			defaultImageRegistry: "my.registry:5000",
			expectedImage:        "my.registry:5000/coredns/coredns:1.12.1",
		},
		{
			name:                 "custom registry trailing slash is trimmed",
			major:                "1",
			minor:                "34",
			defaultImageRegistry: "my.registry:5000/",
			expectedImage:        "my.registry:5000/coredns/coredns:1.12.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := getManifestVariables(tt.defaultImageRegistry, &version.Info{Major: tt.major, Minor: tt.minor})
			assert.Equal(t, vars[VarImage], tt.expectedImage)
		})
	}
}

func TestManifestReplicas(t *testing.T) {
	liveDeployment := func(replicas int32) *appsv1.Deployment {
		return &appsv1.Deployment{Spec: appsv1.DeploymentSpec{Replicas: &replicas}}
	}
	staticReplicas := func(c *config.Config) {
		c.ControlPlane.CoreDNS.Deployment.Replicas = 3
	}
	scaledWithNodes := func(c *config.Config) {
		c.ControlPlane.CoreDNS.Deployment.Replicas = 3
		c.ControlPlane.CoreDNS.Deployment.Autoscaling.Enabled = true
	}
	scaledExternally := func(c *config.Config) {
		scaledWithNodes(c)
		c.ControlPlane.CoreDNS.Deployment.Autoscaling.Mode = config.CoreDNSAutoscalingModeExternal
	}

	tests := []struct {
		name      string
		configure func(*config.Config)
		options   []ManifestOption
		expected  string
	}{
		{
			name:      "static count comes from the config",
			configure: staticReplicas,
			expected:  "  replicas: 3",
		},
		{
			name:      "static count overrides the live deployment",
			configure: staticReplicas,
			options:   []ManifestOption{WithLiveDeployment(liveDeployment(7))},
			expected:  "  replicas: 3",
		},
		{
			name: "static count never renders zero",
			configure: func(c *config.Config) {
				c.ControlPlane.CoreDNS.Deployment.Replicas = 0
			},
			expected: "  replicas: 1",
		},
		{
			name:      "mode Nodes keeps the count of the live deployment",
			configure: scaledWithNodes,
			options:   []ManifestOption{WithLiveDeployment(liveDeployment(7))},
			expected:  "  replicas: 7",
		},
		{
			name:      "mode Nodes starts with one replica while there is no deployment",
			configure: scaledWithNodes,
			options:   []ManifestOption{WithLiveDeployment(nil)},
			expected:  "  replicas: 1",
		},
		{
			name:      "mode Nodes starts with one replica when the live deployment has no count",
			configure: scaledWithNodes,
			options:   []ManifestOption{WithLiveDeployment(&appsv1.Deployment{})},
			expected:  "  replicas: 1",
		},
		{
			name:      "mode Nodes starts with one replica for a caller that passes no deployment",
			configure: scaledWithNodes,
			expected:  "  replicas: 1",
		},
		{
			name:      "mode External keeps the count of the live deployment",
			configure: scaledExternally,
			options:   []ManifestOption{WithLiveDeployment(liveDeployment(7))},
			expected:  "  replicas: 7",
		},
		{
			name:      "mode External starts with the configured count while there is no deployment",
			configure: scaledExternally,
			options:   []ManifestOption{WithLiveDeployment(nil)},
			expected:  "  replicas: 3",
		},
		{
			name: "mode External never starts without a replica",
			configure: func(c *config.Config) {
				scaledExternally(c)
				c.ControlPlane.CoreDNS.Deployment.Replicas = 0
			},
			expected: "  replicas: 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vConfig := mustNewDefaultConfig(tt.configure)
			vars := manifestVariables(vConfig, "", &version.Info{Major: "1", Minor: "34"}, tt.options...)

			manifests, err := processManifests(vars, vConfig)
			assert.NilError(t, err)

			var replicasLines []string
			for _, line := range strings.Split(string(manifests), "\n") {
				if strings.Contains(line, "replicas:") {
					replicasLines = append(replicasLines, line)
				}
			}
			assert.DeepEqual(t, replicasLines, []string{tt.expected})
		})
	}
}
