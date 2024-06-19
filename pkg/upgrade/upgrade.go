package upgrade

import (
	"fmt"
	"os"
	"regexp"
	"sync"

	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"k8s.io/klog/v2"

	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
)

const DevelopmentVersion = "0.0.1"

// Version holds the current version tag
var version string

var githubSlug = "loft-sh/vcluster"
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

func PrintNewerVersionWarning() {
	if os.Getenv("VCLUSTER_SKIP_VERSION_CHECK") != "true" {
		// Get version of current binary
		latestVersion := NewerVersionAvailable()
		if latestVersion != "" {
			log.GetInstance().Warnf("There is a newer version of vcluster: v%s. Run `vcluster upgrade` to upgrade to the newest version.\n", latestVersion)
		}
	}
}

// GetVersion returns the application version
func GetVersion() string {
	return version
}

// SetVersion sets the application version
func SetVersion(verText string) {
	if len(verText) > 0 {
		_version, err := eraseVersionPrefix(verText)
		if err != nil {
			klog.Errorf("Error parsing version: %v", err)
			return
		}

		version = _version
	}
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

// NewerVersionAvailable checks if there is a newer version of vcluster
func NewerVersionAvailable() string {
	if GetVersion() == DevelopmentVersion {
		return ""
	}

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

// Upgrade downloads the latest release from github and replaces vcluster if a new version is found
func Upgrade(flagVersion string, log log.Logger) error {
	updater, err := selfupdate.NewUpdater(selfupdate.Config{
		Filters: []string{"vcluster"},
	})
	if err != nil {
		return fmt.Errorf("failed to initialize updater: %w", err)
	}
	if flagVersion != "" {
		release, found, err := updater.DetectVersion(githubSlug, flagVersion)
		if err != nil {
			return errors.Wrap(err, "find version")
		} else if !found {
			return fmt.Errorf("vcluster version %s couldn't be found", flagVersion)
		}

		cmdPath, err := os.Executable()
		if err != nil {
			return err
		}

		log.Infof("Downloading version %s...", flagVersion)
		err = updater.UpdateTo(release, cmdPath)
		if err != nil {
			return err
		}

		log.Donef("Successfully updated vcluster to version %s", flagVersion)
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
	latest, err := updater.UpdateSelf(v, githubSlug)
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
