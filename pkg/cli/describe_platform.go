package cli

import (
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

	if len(proVClusters) == 0 {
		return fmt.Errorf("couldn't find vcluster %s", name)
	}

	if len(proVClusters) > 1 {
		return fmt.Errorf("found multiple vclusters with name %s. Please use --project flag to narrow down the search", name)
	}

	vCluster := proVClusters[0].VirtualCluster
	if vCluster.Status.VirtualCluster == nil {
		return fmt.Errorf("vcluster %s status is not available", name)
	}

	values := vCluster.Status.VirtualCluster.HelmRelease.Values
	version := vCluster.Status.VirtualCluster.HelmRelease.Chart.Version

	// Return only the user supplied vcluster.yaml, if configOnly is set
	if configOnly {
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
		BackingStore:   string(conf.BackingStoreType()),
		Images:         images,
		UserConfigYaml: &values,
	}

	return writeWithFormat(output, format, describeOutput)
}
