package upgrade

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/blang/semver"
	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/log"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
)

var IsPlugin = "false"

// Version holds the current version tag
var version string

var githubSlug = "loft-sh/loft"
var reVersion = regexp.MustCompile(`\d+\.\d+\.\d+`)

func eraseVersionPrefix(version string) (string, error) {
	indices := reVersion.FindStringIndex(version)
	if indices == nil {
		return version, errors.New("Version not adopting semver")
	}
	if indices[0] > 0 {
		version = version[indices[0]:]
	}

	return version, nil
}

func attachVersionPrefix(version string) string {
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	return version
}

func PrintNewerVersionWarning() {
	if os.Getenv("LOFT_SKIP_VERSION_CHECK") != "true" {
		// Get version of latest binary
		latestVersionStr := NewerVersionAvailable()
		if latestVersionStr != "" {
			checkMajorVersion := true

			latestVersion, err := semver.Parse(latestVersionStr)
			if err != nil {
				log.GetInstance().Debugf("Could not parse current version: %s", latestVersionStr)
				checkMajorVersion = false
			}

			currentVersionStr := GetVersion()
			if currentVersionStr == "" {
				log.GetInstance().Debugf("Current version is not defined")
				checkMajorVersion = false
			}

			currentVersion, err := semver.Parse(currentVersionStr)
			if err != nil {
				log.GetInstance().Debugf("Could not parse current version: %s", currentVersionStr)
				checkMajorVersion = false
			}

			if checkMajorVersion {
				if currentVersion.Major == latestVersion.Major && currentVersion.LT(latestVersion) {
					// Same major version, but upgrade is available
					if IsPlugin == "true" {
						log.GetInstance().Warnf("There is a newer version of the Loft DevSpace plugin: v%s. Run `devspace update plugin loft` to upgrade to the newest version.\n", latestVersionStr)
					} else {
						log.GetInstance().Warnf(product.Replace("There is a newer version of Loft: v%s. Run `loft upgrade` to upgrade to the newest version.\n"), latestVersionStr)
					}
				} else if currentVersion.Major == uint64(2) && latestVersion.Major == uint64(3) {
					// Different major version, link to upgrade guide
					log.GetInstance().Warnf("There is a newer version of Loft: v%s. Please visit https://loft.sh/docs/guides/upgrade-2-to-3 for more information on upgrading.\n", latestVersionStr)
				}
			} else {
				if IsPlugin == "true" {
					log.GetInstance().Warnf("There is a newer version of the Loft DevSpace plugin: v%s. Run `devspace update plugin loft` to upgrade to the newest version.\n", latestVersionStr)
				} else {
					log.GetInstance().Warnf(product.Replace("There is a newer version of Loft: v%s. Run `loft upgrade` to upgrade to the newest version.\n"), latestVersionStr)
				}
			}
		}
	}
}

// GetVersion returns the application version
func GetVersion() string {
	return version
}

// SetVersion sets the application version
func SetVersion(verText string) error {
	if len(verText) > 0 {
		_version, err := eraseVersionPrefix(verText)
		if err != nil {
			return fmt.Errorf("error parsing version: %w", err)
		}

		version = _version
	}

	return nil
}

var (
	latestVersion     string
	errLatestVersion  error
	latestVersionOnce sync.Once
)

// CheckForNewerVersion checks if there is a newer version on github and returns the newer version
func CheckForNewerVersion() (string, error) {
	latestVersionOnce.Do(func() {
		latest, found, err := selfupdate.DetectLatest(githubSlug)
		if err != nil {
			errLatestVersion = err
			return
		}

		v := semver.MustParse(version)
		if !found || latest.Version.Equals(v) {
			return
		}

		latestVersion = latest.Version.String()
	})

	return latestVersion, errLatestVersion
}

// NewerVersionAvailable checks if there is a newer version of loft
func NewerVersionAvailable() string {
	// Get version of current binary
	version := GetVersion()
	if version != "" {
		latestStableVersion, err := CheckForNewerVersion()
		if latestStableVersion != "" && err == nil { // Check versions only if newest version could be determined without errors
			semverVersion, err := semver.Parse(version)
			if err == nil { // Only compare version if version can be parsed
				semverLatestStableVersion, err := semver.Parse(latestStableVersion)
				if err == nil { // Only compare version if latestStableVersion can be parsed
					// If latestStableVersion > version
					if semverLatestStableVersion.Compare(semverVersion) == 1 {
						return latestStableVersion
					}
				}
			}
		}
	}

	return ""
}

// Upgrade downloads the latest release from github and replaces loft if a new version is found
func Upgrade(flagVersion string, log log.Logger) error {
	if flagVersion != "" {
		flagVersion = attachVersionPrefix(flagVersion)

		release, found, err := selfupdate.DetectVersion(githubSlug, flagVersion)
		if err != nil {
			return errors.Wrap(err, "find version")
		} else if !found {
			return fmt.Errorf("loft version %s couldn't be found", flagVersion)
		}

		cmdPath, err := os.Executable()
		if err != nil {
			return err
		}

		log.Infof("Downloading version %s...", flagVersion)
		err = selfupdate.DefaultUpdater().UpdateTo(release, cmdPath)
		if err != nil {
			return err
		}

		log.Donef("Successfully updated Loft to version %s", flagVersion)
		return nil
	}

	newerVersion, err := CheckForNewerVersion()
	if err != nil {
		return err
	}
	if newerVersion == "" {
		log.Infof("Current binary is the latest version: %s", version)
		return nil
	}

	v := semver.MustParse(version)

	log.Info("Downloading newest version...")
	latest, err := selfupdate.UpdateSelf(v, githubSlug)
	if err != nil {
		return err
	}

	if latest.Version.Equals(v) {
		// latest version is the same as current version. It means current binary is up to date.
		log.Infof("Current binary is the latest version: %s", version)
	} else {
		log.Donef("Successfully updated to version %s", latest.Version)
		log.Infof("Release note: \n\n%s", latest.ReleaseNotes)
	}

	return nil
}
