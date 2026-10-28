package licenseapi

import "fmt"

// GetFeatureDisplayName returns the display name for a given feature name.
func GetFeatureDisplayName(featureName string) (string, error) {
	features := GetAllFeatures()

	for _, feature := range features {
		if feature.Name == featureName {
			return feature.DisplayName, nil
		}
	}
	return "", fmt.Errorf("no feature found with name %s", featureName)
}
