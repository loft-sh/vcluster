package vclusterconfig

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/api/v4/pkg/vclusterconfig/constants"
	"github.com/robfig/cron/v3"
	"golang.org/x/mod/semver"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidatePlatformConfig(fldPath *field.Path, platformConfig PlatformConfig) field.ErrorList {
	var errs field.ErrorList

	errs = append(errs, ValidateSleep(fldPath, platformConfig.Sleep)...)
	errs = append(errs, ValidateSnapshots(fldPath, platformConfig.Snapshots)...)
	errs = append(errs, ValidateDeletion(fldPath, platformConfig.Deletion)...)

	return errs
}

// ValidateSleep validates the new top-level Sleep configuration
func ValidateSleep(fldPath *field.Path, sleep *Sleep) field.ErrorList {
	if sleep == nil || sleep.Auto == nil {
		return nil
	}

	var errs field.ErrorList
	auto := sleep.Auto

	if auto.Schedule != "" {
		if err := validateCronSchedule(auto.Schedule); err != nil {
			errs = append(errs, field.Invalid(fldPath.Child("sleep", "auto", "schedule"), auto.Schedule, err.Error()))
		}
	}

	if auto.Wakeup != nil && auto.Wakeup.Schedule != "" {
		if err := validateCronSchedule(auto.Wakeup.Schedule); err != nil {
			errs = append(errs, field.Invalid(fldPath.Child("sleep", "auto", "wakeup", "schedule"), auto.Wakeup.Schedule, err.Error()))
		}
	}

	if auto.Timezone != "" {
		if err := validateTimezone(auto.Timezone); err != nil {
			errs = append(errs, field.Invalid(fldPath.Child("sleep", "auto", "timezone"), auto.Timezone, err.Error()))
		}
	}

	return errs
}

// ValidateSnapshots validates the new top-level Snapshots configuration
func ValidateSnapshots(fldPath *field.Path, snapshots *Snapshots) field.ErrorList {
	if snapshots == nil || snapshots.Auto == nil {
		return nil
	}

	var errs field.ErrorList
	auto := snapshots.Auto

	if auto.Schedule == "" {
		errs = append(errs, field.Required(fldPath.Child("snapshots", "auto", "schedule"), "schedule is required when snapshots are configured"))
	} else if err := validateCronSchedule(auto.Schedule); err != nil {
		errs = append(errs, field.Invalid(fldPath.Child("snapshots", "auto", "schedule"), auto.Schedule, err.Error()))
	}

	if auto.Timezone != "" {
		if err := validateTimezone(auto.Timezone); err != nil {
			errs = append(errs, field.Invalid(fldPath.Child("snapshots", "auto", "timezone"), auto.Timezone, err.Error()))
		}
	}

	if auto.Retention == nil || auto.Retention.MaxSnapshots == 0 || auto.Retention.Period == 0 {
		errs = append(errs, field.Invalid(
			fldPath.Child("snapshots", "auto", "retention"),
			auto.Retention,
			"retention.period and retention.maxSnapshots must both be greater than 0",
		))
	}

	if auto.Storage == nil {
		errs = append(errs, field.Required(
			fldPath.Child("snapshots", "auto", "storage"),
			"storage is required when snapshots are configured",
		))
	} else {
		switch auto.Storage.Type {
		case constants.StorageTypeContainer:
			container := auto.Storage.Container
			if container.Path == "" || container.Volume.Name == "" || container.Volume.Path == "" {
				errs = append(errs, field.Invalid(
					fldPath.Child("snapshots", "auto", "storage", "container"),
					container,
					"storage type is set to 'container', but container configuration is missing (path, volume.name, volume.path required)",
				))
			}
		case constants.StorageTypeS3:
			s3 := auto.Storage.S3
			if s3.Url == "" {
				errs = append(errs, field.Invalid(
					fldPath.Child("snapshots", "auto", "storage", "s3", "url"),
					s3.Url,
					"storage type is set to 's3', but url is missing",
				))
			}
		case constants.StorageTypeOCI:
			oci := auto.Storage.OCI
			if oci.Repository == "" && (oci.Credential == nil && oci.Username == "" && oci.Password == "") {
				errs = append(errs, field.Invalid(
					fldPath.Child("snapshots", "auto", "storage", "oci"),
					oci,
					"storage type is set to 'oci', but repository or credentials are missing",
				))
			}
		default:
			errs = append(errs, field.Invalid(
				fldPath.Child("snapshots", "auto", "storage", "type"),
				auto.Storage.Type,
				fmt.Sprintf("storage type must be one of: %s, %s, %s", constants.StorageTypeContainer, constants.StorageTypeS3, constants.StorageTypeOCI),
			))
		}
	}

	return errs
}

// ValidateDeletion validates the new top-level Deletion configuration
func ValidateDeletion(fldPath *field.Path, deletion *Deletion) field.ErrorList {
	if deletion == nil || deletion.Auto == nil {
		return nil
	}

	var errs field.ErrorList

	if deletion.Auto.AfterInactivity != "" {
		_, err := time.ParseDuration(string(deletion.Auto.AfterInactivity))
		if err != nil {
			errs = append(errs, field.Invalid(
				fldPath.Child("deletion", "auto", "afterInactivity"),
				deletion.Auto.AfterInactivity,
				fmt.Sprintf("invalid duration format: %v (use Go duration format like '720h' or '30d')", err),
			))
		}
	}

	return errs
}

// check if vcluster chart version is compatible with PV snapshots
func IsVolumeSnapshotCompatible(release storagev1.VirtualClusterHelmRelease) bool {
	return semver.Compare("v"+release.Chart.Version, "v0.30.0-alpha.0") == 1
}

// validateCronSchedule validates a cron schedule string using the standard cron parser.
func validateCronSchedule(schedule string) error {
	_, err := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor).Parse(schedule)
	return err
}

func validateTimezone(timezone string) error {
	if strings.Contains(timezone, "#") {
		splitted := strings.Split(timezone, "#")
		if len(splitted) == 2 {
			_, err := strconv.Atoi(splitted[1])
			if err != nil {
				return fmt.Errorf("error parsing offset: %w", err)
			}
		}
	} else {
		_, err := time.LoadLocation(timezone)
		if err != nil {
			return fmt.Errorf("error parsing timezone: %w", err)
		}
	}
	return nil
}

// ValidateLegacySleepMode validates the deprecated top-level sleepMode configuration
func ValidateLegacySleepMode(fldPath *field.Path, sleepMode *LegacySleepMode) field.ErrorList {
	if sleepMode == nil {
		return nil
	}

	var errs field.ErrorList
	autoSleep := sleepMode.AutoSleep
	autoWakeup := sleepMode.AutoWakeup

	if autoSleep != nil && autoSleep.Schedule != "" {
		if err := validateCronSchedule(autoSleep.Schedule); err != nil {
			errs = append(errs, field.Invalid(fldPath.Child("autoSleep", "schedule"), autoSleep.Schedule, err.Error()))
		}
	}

	if autoWakeup != nil && autoWakeup.Schedule != "" {
		if err := validateCronSchedule(autoWakeup.Schedule); err != nil {
			errs = append(errs, field.Invalid(fldPath.Child("autoWakeup", "schedule"), autoWakeup.Schedule, err.Error()))
		}
	}

	if sleepMode.TimeZone != "" {
		if err := validateTimezone(sleepMode.TimeZone); err != nil {
			errs = append(errs, field.Invalid(fldPath.Child("timeZone"), sleepMode.TimeZone, err.Error()))
		}
	}

	return errs
}

// ValidateLegacyPlatformConfig validates the deprecated external.platform configuration
func ValidateLegacyPlatformConfig(fldPath *field.Path, legacyPlatformConfig LegacyPlatformConfig) field.ErrorList {
	var errs field.ErrorList

	if legacyPlatformConfig.AutoSleep != nil {
		if legacyPlatformConfig.AutoSleep.Schedule != "" {
			if err := validateCronSchedule(legacyPlatformConfig.AutoSleep.Schedule); err != nil {
				errs = append(errs, field.Invalid(fldPath.Child("autoSleep", "schedule"), legacyPlatformConfig.AutoSleep.Schedule, err.Error()))
			}
		}

		if legacyPlatformConfig.AutoSleep.AutoWakeup != nil && legacyPlatformConfig.AutoSleep.AutoWakeup.Schedule != "" {
			if err := validateCronSchedule(legacyPlatformConfig.AutoSleep.AutoWakeup.Schedule); err != nil {
				errs = append(errs, field.Invalid(fldPath.Child("autoSleep", "autoWakeup", "schedule"), legacyPlatformConfig.AutoSleep.AutoWakeup.Schedule, err.Error()))
			}
		}

		if legacyPlatformConfig.AutoSleep.Timezone != "" {
			if err := validateTimezone(legacyPlatformConfig.AutoSleep.Timezone); err != nil {
				errs = append(errs, field.Invalid(fldPath.Child("autoSleep", "timezone"), legacyPlatformConfig.AutoSleep.Timezone, err.Error()))
			}
		}
	}

	if legacyPlatformConfig.AutoSnapshot != nil {
		if legacyPlatformConfig.AutoSnapshot.Schedule == "" {
			errs = append(errs, field.Invalid(fldPath.Child("autoSnapshot", "schedule"), legacyPlatformConfig.AutoSnapshot.Schedule, "scheduled field is required."))
		} else if err := validateCronSchedule(legacyPlatformConfig.AutoSnapshot.Schedule); err != nil {
			errs = append(errs, field.Invalid(fldPath.Child("autoSnapshot", "schedule"), legacyPlatformConfig.AutoSnapshot.Schedule, err.Error()))
		}

		if legacyPlatformConfig.AutoSnapshot.Timezone != "" {
			if err := validateTimezone(legacyPlatformConfig.AutoSnapshot.Timezone); err != nil {
				errs = append(errs, field.Invalid(
					fldPath.Child("autoSnapshot", "timezone"),
					legacyPlatformConfig.AutoSnapshot.Timezone,
					err.Error(),
				))
			}
		}

		if legacyPlatformConfig.AutoSnapshot.Retention.MaxSnapshots == 0 || legacyPlatformConfig.AutoSnapshot.Retention.Period == 0 {
			errs = append(errs, field.Invalid(
				fldPath.Child("autoSnapshot", "retention"),
				legacyPlatformConfig.AutoSnapshot.Retention,
				"retention should be configured",
			))
		}

		switch legacyPlatformConfig.AutoSnapshot.Storage.Type {
		case constants.StorageTypeContainer:
			container := legacyPlatformConfig.AutoSnapshot.Storage.Container
			if container.Path == "" ||
				container.Volume.Name == "" ||
				container.Volume.Path == "" {
				errs = append(errs, field.Invalid(
					fldPath.Child("autoSnapshot", "storage", "container"),
					container,
					"storage type is set to 'container', but container configuration is missing.",
				))
			}
		case constants.StorageTypeS3:
			s3 := legacyPlatformConfig.AutoSnapshot.Storage.S3
			if s3.Url == "" {
				errs = append(errs, field.Invalid(
					fldPath.Child("autoSnapshot", "storage", "s3", "url"),
					s3,
					"storage type is set to 's3', but s3 configuration is missing.",
				))
			}
		case constants.StorageTypeOCI:
			oci := legacyPlatformConfig.AutoSnapshot.Storage.OCI
			if oci.Repository == "" && (oci.Credential == nil || (oci.Username == "" && oci.Password == "")) {
				errs = append(errs, field.Invalid(
					fldPath.Child("autoSnapshot", "storage", "oci"),
					oci,
					"storage type is set to 'oci', but oci configuration is missing.",
				))
			}
		default:
			errs = append(errs, field.Invalid(
				fldPath.Child("autoSnapshot", "storage", "type"),
				legacyPlatformConfig.AutoSnapshot.Storage,
				fmt.Sprintf("storage type is not set or is not equal to %s, %s or %s", constants.StorageTypeContainer, constants.StorageTypeS3, constants.StorageTypeOCI),
			))
		}
	}

	return errs
}
