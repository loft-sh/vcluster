package constants

import (
	"os"

	"github.com/loft-sh/vcluster/pkg/upgrade"
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
