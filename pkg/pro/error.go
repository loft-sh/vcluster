package pro

import (
	"fmt"

	"github.com/loft-sh/admin-apis/pkg/licenseapi"
)

func NewFeatureError(featureName licenseapi.FeatureName) error {
	displayName, err := licenseapi.GetFeatureDisplayName(string(featureName))
	if err != nil {
		displayName = string(featureName) // Fallback to feature name if display name not found
	}
	return fmt.Errorf("you are trying to use a vCluster pro feature '%s' (%s) that is not available in the open source vcluster or disabled in your license. Please use the vCluster pro image and specify a license that allows using this feature or reach out to support@loft.sh",
		displayName, featureName)
}
