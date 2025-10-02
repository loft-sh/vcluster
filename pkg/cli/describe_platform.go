package cli

import (
	"cmp"
	"context"
	"fmt"
	"io"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
)

func DescribePlatform(ctx context.Context, globalFlags *flags.GlobalFlags, output io.Writer, l log.Logger, name string, projectName string, configOnly bool, format string) error {
	platformClient, err := platform.InitClientFromConfig(ctx, globalFlags.LoadedConfig(l))
	if err != nil {
		return err
	}

	proVClusters, err := platform.ListVClusters(ctx, platformClient, name, projectName, false)
	if err != nil {
		return err
	}

	if len(proVClusters) != 1 {
		return fmt.Errorf("couldn't find vcluster %s", name)
	}

	// provclusters should be len(1), because 0 exits beforehand, and there's only 1
	// vcluster with a name in a project
	vCluster := proVClusters[0].VirtualCluster
	values := vCluster.Status.VirtualCluster.HelmRelease.Values
	version := vCluster.Status.VirtualCluster.HelmRelease.Chart.Version

	// Return only the user supplied vcluster.yaml, if configOnly is set
	if configOnly {
		if cmp.Or(format, "yaml") != "yaml" {
			return fmt.Errorf("--config-only output supports only yaml format")
		}

		if _, err := output.Write([]byte(values)); err != nil {
			return err
		}

		return nil
	}

	conf, err := configPartialUnmarshal([]byte(values))
	if err != nil {
		return err
	}

	images := getImagesFromConfig(conf, version)
	if _, found := images["syncer"]; !found {
		images["syncer"] = fmt.Sprintf("ghcr.io/loft-sh/vcluster-pro:%s", version)
	}

	describeOutput := &DescribeOutput{
		Name:           vCluster.Name,
		Namespace:      vCluster.Namespace,
		Created:        vCluster.CreationTimestamp,
		Status:         string(vCluster.Status.Phase),
		Version:        version,
		Distro:         conf.Distro(),
		BackingStore:   string(conf.BackingStoreType()),
		Images:         images,
		UserConfigYaml: &values,
	}

	return writeWithFormat(output, format, describeOutput)
}
