package licenseapi

// This code was generated. Change features.yaml to add, remove, or edit features.

import (
	"errors"
	"time"
)

var errNoAllowBefore = errors.New("feature not allowed before license's issued date")

// featureToAllowBefore maps feature names to their corresponding
// RFC3339-formatted allowBefore timestamps. If a license was issued before this
// timestamp, the feature is allowed even if it is not explicitly included in the license.
var featuresToAllowBefore = map[FeatureName]string{
	ProjectQuotas: "2025-05-31T00:00:00Z",
	DisablePlatformDB: "2025-09-09T00:00:00Z",
}

// GetFeaturesAllowedBefore returns list of features
// to be allowed before license's issued time
func GetFeaturesAllowedBefore() []FeatureName {
	return []FeatureName{
		ProjectQuotas,
		DisablePlatformDB,
	}
}

// AllowedBeforeTime returns the parsed allowBefore time for a given feature.
// If the feature does not have an allowBefore date, it returns errNoAllowBefore.
// If the date is present but invalid, it returns the corresponding parsing error.
func AllowedBeforeTime(featureName FeatureName) (*time.Time, error) {
	if date, exists := featuresToAllowBefore[featureName]; exists {
		t, err := time.Parse(time.RFC3339, date)
		if err != nil {
			return nil, err
		}
		return &t, nil

	}
	return nil, errNoAllowBefore
}

// IsAllowBeforeNotDefined determines whether the provided error is
// errNoAllowBefore, indicating that the feature has no allowBefore date.
func IsAllowBeforeNotDefined(err error) bool {
	return errors.Is(err, errNoAllowBefore)
}
