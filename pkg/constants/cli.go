package constants

import "github.com/loft-sh/vcluster/pkg/upgrade"

func DefaultBackgroundProxyImage(version string) string {
	if version == upgrade.DevelopmentVersion {
		return "ghcr.io/loft-sh/vcluster:dev-next"
	}
	return "ghcr.io/loft-sh/vcluster-pro:" + version
}
