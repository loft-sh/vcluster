package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
)

func DescribePlatform(ctx context.Context, globalFlags *flags.GlobalFlags, output io.Writer, l log.Logger, name, projectName, format string) error {
	platformClient, err := platform.InitClientFromConfig(ctx, globalFlags.LoadedConfig(l))
	if err != nil {
		return err
	}

	proVClusters, err := platform.ListVClusters(ctx, platformClient, name, projectName)
	if err != nil {
		return err
	}

	// provclusters should be len(1), because 0 exits beforehand, and there's only 1
	// vcluster with a name in a project
	values := proVClusters[0].VirtualCluster.Status.VirtualCluster.HelmRelease.Values
	version := proVClusters[0].VirtualCluster.Status.VirtualCluster.HelmRelease.Chart.Version

	switch format {
	case "yaml":
		_, err = output.Write([]byte(values))
		return err
	case "json":
		b, err := yaml.YAMLToJSON([]byte(values))
		if err != nil {
			return err
		}
		_, err = output.Write(b)
		return err
	}
	describeOutput := &DescribeOutput{}

	describeOutput.Version = version

	err = extractFromValues(describeOutput, []byte(values), format, version, output)
	if err != nil {
		return err
	}

	if describeOutput.ImageTags.Syncer == "" {
		describeOutput.ImageTags.Syncer = fmt.Sprintf("ghcr.io/loft-sh/vcluster-pro:%s", version)
	}

	b, err := yaml.Marshal(describeOutput)
	if err != nil {
		return err
	}
	_, err = output.Write(b)

	return err
}
