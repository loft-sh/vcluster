package cli

import (
	"cmp"
	"context"
	"fmt"
	"io"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
)

func DescribePlatform(ctx context.Context, globalFlags *flags.GlobalFlags, output io.Writer, l log.Logger, name string, projectName string, showConfig bool, format string) error {
	platformClient, err := platform.InitClientFromConfig(ctx, globalFlags.LoadedConfig(l))
	if err != nil {
		return err
	}

	proVClusters, err := platform.ListVClusters(ctx, platformClient, name, projectName, false)
	if err != nil {
		return err
	}

	// provclusters should be len(1), because 0 exits beforehand, and there's only 1
	// vcluster with a name in a project
	vCluster := proVClusters[0].VirtualCluster
	values := vCluster.Status.VirtualCluster.HelmRelease.Values
	version := vCluster.Status.VirtualCluster.HelmRelease.Chart.Version

	// Return only the user supplied vcluster.yaml, if showConfig is set
	if showConfig {
		var rawValues map[string]interface{}
		if err := yaml.Unmarshal([]byte(values), &rawValues); err != nil {
			return err
		}

		return writeWithFormat(output, cmp.Or(format, "yaml"), rawValues)
	}

	conf, err := configPartialUnmarshal([]byte(values))
	if err != nil {
		return err
	}

	syncer, api := getImageTags(conf, version)

	describeOutput := &DescribeOutput{
		Name:         vCluster.Name,
		Namespace:    vCluster.Namespace,
		Created:      vCluster.CreationTimestamp,
		Status:       string(vCluster.Status.Phase),
		Version:      version,
		Distro:       conf.Distro(),
		BackingStore: string(conf.BackingStoreType()),
		ImageTags: ImageTag{
			APIServer: api,
			Syncer:    syncer,
		},
	}

	if describeOutput.ImageTags.Syncer == "" {
		describeOutput.ImageTags.Syncer = fmt.Sprintf("ghcr.io/loft-sh/vcluster-pro:%s", version)
	}

	return writeWithFormat(output, format, describeOutput)
}
