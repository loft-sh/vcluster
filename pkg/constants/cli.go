package constants

import (
	"os"

	"github.com/loft-sh/vcluster/pkg/upgrade"
)

const (
	TokenLabelKey        = "vcluster.loft.sh/token"
	TokenNodeTypeKey     = "vcluster.loft.sh/token-node-type"
	NodeTypeControlPlane = "control-plane"
	NodeTypeWorker       = "worker"
)

const (
	DockerContainerdSocketPath = "/var/run/docker/containerd/containerd.sock"
	DockerSocketPath           = "/var/run/docker/docker.sock"
	DockerControlPlanePrefix   = "vcluster.cp."
	DockerNodePrefix           = "vcluster.node."
	DockerLoadBalancerPrefix   = "vcluster.lb."
	DockerNetworkPrefix        = "vcluster."
)

func DefaultBackgroundProxyImage(version string) string {
	envProxyImage := os.Getenv("VCLUSTER_BACKGROUND_PROXY_IMAGE")
	if envProxyImage != "" {
		return envProxyImage
	}

	if version == upgrade.DevelopmentVersion {
		return "ghcr.io/loft-sh/vcluster:dev-next"
	}

	return "ghcr.io/loft-sh/vcluster-pro:" + version
}
