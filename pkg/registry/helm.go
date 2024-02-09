package registry

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/helm"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
)

func GetChartInfo(ctx context.Context, hostClient kubernetes.Interface, vClusterNamespace string) (*ChartInfo, error) {
	if hostClient == nil {
		return nil, fmt.Errorf("host client is empty")
	}

	release, err := helm.NewSecrets(hostClient).Get(ctx, translate.VClusterName, vClusterNamespace)
	if err != nil {
		return nil, err
	} else if release == nil {
		return &ChartInfo{}, nil
	} else if kerrors.IsNotFound(err) {
		return &ChartInfo{}, nil
	}

	if release.Config == nil {
		release.Config = map[string]interface{}{}
	}

	name := "unknown"
	chartVersion := ""
	if release.Chart != nil && release.Chart.Metadata != nil && release.Chart.Metadata.Name != "" {
		name = release.Chart.Metadata.Name
		chartVersion = release.Chart.Metadata.Version
	}

	return &ChartInfo{
		Name:    name,
		Version: chartVersion,
		Values:  release.Config,
	}, nil
}
