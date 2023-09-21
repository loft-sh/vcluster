package version

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/blang/semver"
	storagev1 "github.com/loft-sh/api/v3/pkg/apis/storage/v1"
)

type matchedVersion struct {
	Object  storagev1.VersionAccessor
	Version semver.Version
}

func GetLatestVersion(versions storagev1.VersionsAccessor) storagev1.VersionAccessor {
	// find the latest version
	var latestVersion *matchedVersion
	for _, version := range versions.GetVersions() {
		parsedVersion, err := semver.Parse(strings.TrimPrefix(version.GetVersion(), "v"))
		if err != nil {
			continue
		}

		// latest available version
		if latestVersion == nil || latestVersion.Version.LT(parsedVersion) {
			latestVersion = &matchedVersion{
				Object:  version,
				Version: parsedVersion,
			}
		}
	}
	if latestVersion == nil {
		return nil
	}

	return latestVersion.Object
}

func GetLatestMatchedVersion(versions storagev1.VersionsAccessor, versionPattern string) (latestVersion storagev1.VersionAccessor, latestMatchedVersion storagev1.VersionAccessor, err error) {
	// parse version
	splittedVersion := strings.Split(strings.ToLower(strings.TrimPrefix(versionPattern, "v")), ".")
	if len(splittedVersion) != 3 {
		return nil, nil, fmt.Errorf("couldn't parse version %s, expected version in format: 0.0.0", versionPattern)
	}

	// find latest version that matches our defined version
	var latestVersionObj *matchedVersion
	var latestMatchedVersionObj *matchedVersion
	for _, version := range versions.GetVersions() {
		parsedVersion, err := semver.Parse(strings.TrimPrefix(version.GetVersion(), "v"))
		if err != nil {
			continue
		}

		// does the version match our restrictions?
		if (splittedVersion[0] == "x" || splittedVersion[0] == "X" || strconv.FormatUint(parsedVersion.Major, 10) == splittedVersion[0]) &&
			(splittedVersion[1] == "x" || splittedVersion[1] == "X" || strconv.FormatUint(parsedVersion.Minor, 10) == splittedVersion[1]) &&
			(splittedVersion[2] == "x" || splittedVersion[2] == "X" || strconv.FormatUint(parsedVersion.Patch, 10) == splittedVersion[2]) {
			if latestMatchedVersionObj == nil || latestMatchedVersionObj.Version.LT(parsedVersion) {
				latestMatchedVersionObj = &matchedVersion{
					Object:  version,
					Version: parsedVersion,
				}
			}
		}

		// latest available version
		if latestVersionObj == nil || latestVersionObj.Version.LT(parsedVersion) {
			latestVersionObj = &matchedVersion{
				Object:  version,
				Version: parsedVersion,
			}
		}
	}

	if latestVersionObj != nil {
		latestVersion = latestVersionObj.Object
	}
	if latestMatchedVersionObj != nil {
		latestMatchedVersion = latestMatchedVersionObj.Object
	}

	return latestVersion, latestMatchedVersion, nil
}
