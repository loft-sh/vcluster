package pro

import "fmt"

func NewFeatureError(featureName string) error {
	return fmt.Errorf("you are trying to use a vCluster pro feature '%s' that is not allowed by your current license. Please specify a license that allows using this feature or reach out to support@loft.sh", featureName)
}
