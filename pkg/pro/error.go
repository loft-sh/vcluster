package pro

import (
	"fmt"

	"github.com/loft-sh/admin-apis/pkg/util/features"
)

func NewFeatureError(featureName string) error {
	displayName, err := features.GetFeatureDisplayName(featureName)
	if err != nil {
		displayName = "Unknown Feature"
	}
	return fmt.Errorf("you are trying to use a vCluster pro feature %s (%s) that is not part of the open-source build of vCluster. Please use the vCluster pro image and specify a license that allows using this feature or reach out to support@loft.sh", &displayName, featureName)
}
