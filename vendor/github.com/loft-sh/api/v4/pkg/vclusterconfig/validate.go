package vclusterconfig

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/api/v4/pkg/vclusterconfig/constants"
	"github.com/robfig/cron/v3"
	"golang.org/x/mod/semver"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidatePlatformConfig(fldPath *field.Path, platformConfig PlatformConfig) field.ErrorList {
	var errs field.ErrorList

	errs = append(errs, ValidateSleep(fldPath, platformConfig.Sleep)...)
	errs = append(errs, ValidateSnapshots(fldPath, platformConfig.Snapshots)...)
	errs = append(errs, ValidateDeletion(fldPath, platformConfig.Deletion)...)
	errs = append(errs, ValidateArgoCD(fldPath, platformConfig.ArgoCDIntegration, platformConfig.ArgoCDDeploy)...)
	errs = append(errs, ValidateObservability(fldPath, platformConfig.ObservabilityIntegration)...)
	errs = append(errs, ValidateStacks(fldPath, platformConfig.Stacks)...)

	return errs
}

// MaxStacks bounds how many stacks one vcluster.yaml may declare, set well above any real config.
const MaxStacks = 50

// ValidateStacks checks the deploy.stacks rules that can be judged before conversion.
func ValidateStacks(fldPath *field.Path, stacks []StackConfig) field.ErrorList {
	errs := ValidateStackList(fldPath, stacks)
	if len(stacks) > MaxStacks {
		// Over the cap, the count is the only error worth reporting.
		return errs
	}

	stacksPath := fldPath.Child("deploy", "stacks")
	for i, stack := range stacks {
		errs = append(errs, ValidateStack(stacksPath.Index(i), stack)...)
	}

	return errs
}

// ValidateStackList checks the cap, missing names and duplicate names. These fail the whole sync:
// a stack without a usable name cannot be told apart from its siblings.
func ValidateStackList(fldPath *field.Path, stacks []StackConfig) field.ErrorList {
	if len(stacks) == 0 {
		return nil
	}

	stacksPath := fldPath.Child("deploy", "stacks")
	if len(stacks) > MaxStacks {
		return field.ErrorList{field.TooMany(stacksPath, len(stacks), MaxStacks)}
	}

	var errs field.ErrorList
	seenNames := map[string]int{}

	for i, stack := range stacks {
		namePath := stacksPath.Index(i).Child("name")
		if stack.Name == "" {
			errs = append(errs, field.Required(namePath, "each stack must have a name"))
			continue
		}

		if previous, ok := seenNames[stack.Name]; ok {
			errs = append(errs, field.Duplicate(namePath, fmt.Sprintf("%s (already used at index %d)", stack.Name, previous)))
			continue
		}

		seenNames[stack.Name] = i
	}

	return errs
}

// ValidateStack checks one entry. ValidateStackList handles missing and duplicate names.
func ValidateStack(stackPath *field.Path, stack StackConfig) field.ErrorList {
	var errs field.ErrorList

	// Not trimmed: the name becomes part of the resource name, so spaces must fail.
	// An empty name is reported once, by ValidateStackList.
	if stack.Name != "" {
		for _, msg := range validation.IsDNS1123Label(stack.Name) {
			errs = append(errs, field.Invalid(stackPath.Child("name"), stack.Name, msg))
		}
	}

	hasTemplate := stack.Template != nil
	hasTemplateRef := stack.TemplateRef != nil
	switch {
	case hasTemplate && hasTemplateRef:
		errs = append(errs, field.Forbidden(stackPath, "exactly one of template or templateRef must be set, not both"))
	case !hasTemplate && !hasTemplateRef:
		errs = append(errs, field.Required(stackPath, "exactly one of template or templateRef must be set"))
	case hasTemplateRef && stack.TemplateRef.Name == "":
		errs = append(errs, field.Required(stackPath.Child("templateRef", "name"), "templateRef.name must be set"))
	}

	switch storagev1.StackPrunePolicy(stack.PrunePolicy) {
	case "", storagev1.StackPrunePolicyRetain, storagev1.StackPrunePolicyPrune:
	default:
		errs = append(errs, field.NotSupported(stackPath.Child("prunePolicy"), stack.PrunePolicy,
			[]string{string(storagev1.StackPrunePolicyRetain), string(storagev1.StackPrunePolicyPrune)}))
	}

	if stack.Defaults != nil {
		errs = append(errs, validateDuration(stackPath.Child("defaults", "taskTimeout"), stack.Defaults.TaskTimeout)...)
	}

	if hasTemplate {
		for j, task := range stack.Template.Tasks {
			taskPath := stackPath.Child("template", "tasks").Index(j)
			errs = append(errs, validateDuration(taskPath.Child("timeout"), task.Timeout)...)
		}
	}

	return errs
}

// validateDuration checks a Go duration string such as "30m" or "720h". Empty means unset.
func validateDuration(fldPath *field.Path, value string) field.ErrorList {
	if value == "" {
		return nil
	}

	if _, err := time.ParseDuration(value); err != nil {
		return field.ErrorList{field.Invalid(fldPath, value,
			fmt.Sprintf("invalid duration format: %v (use a Go duration string like '30m', or '720h' for 30 days)", err))}
	}

	return nil
}

// ValidateObservability validates the observability integration configuration.
func ValidateObservability(fldPath *field.Path, integration *ObservabilityIntegration) field.ErrorList {
	if integration == nil || !integration.Enabled {
		return nil
	}

	var errs field.ErrorList
	path := fldPath.Child("integrations", "observability")

	if integration.Connector == "" {
		errs = append(errs, field.Required(path.Child("connector"), "connector is required when observability is enabled"))
	}

	if integration.GatewaySecret != nil {
		gsPath := path.Child("gatewaySecret")
		if integration.GatewaySecret.Namespace == "" {
			errs = append(errs, field.Required(gsPath.Child("namespace"), "namespace is required"))
		}
		if integration.GatewaySecret.Name == "" {
			errs = append(errs, field.Required(gsPath.Child("name"), "name is required"))
		}
	}

	return errs
}

// ValidateArgoCD validates the Argo CD integration and deploy configuration.
func ValidateArgoCD(fldPath *field.Path, integration *ArgoCDIntegration, deploy *ArgoCDDeploy) field.ErrorList {
	if deploy == nil || len(deploy.Applications) == 0 {
		return nil
	}

	var errs field.ErrorList
	deployPath := fldPath.Child("deploy", "argoCD")
	integrationPath := fldPath.Child("integrations", "argoCD")

	if integration == nil || !integration.Enabled {
		errs = append(errs, field.Invalid(deployPath.Child("applications"), deploy.Applications, "argoCD integration must be enabled when applications are configured"))
	}
	if integration == nil || integration.Connector == "" {
		errs = append(errs, field.Required(integrationPath.Child("connector"), "connector is required when argoCD applications are configured"))
	}

	seenNames := map[string]int{}
	seenDisplayNames := map[string]int{}
	for i, application := range deploy.Applications {
		appPath := deployPath.Child("applications").Index(i)
		name := strings.TrimSpace(application.Name)
		displayName := strings.TrimSpace(application.DisplayName)
		if name == "" && displayName == "" {
			errs = append(errs, field.Required(appPath, "either name or displayName must be set"))
		}
		if name != "" {
			if previousIndex, ok := seenNames[name]; ok {
				errs = append(errs, field.Duplicate(appPath.Child("name"), fmt.Sprintf("%s (already used at index %d)", name, previousIndex)))
			} else {
				seenNames[name] = i
			}
		}
		if displayName != "" {
			if previousIndex, ok := seenDisplayNames[displayName]; ok {
				errs = append(errs, field.Duplicate(appPath.Child("displayName"), fmt.Sprintf("%s (already used at index %d)", displayName, previousIndex)))
			} else {
				seenDisplayNames[displayName] = i
			}
		}

		switch application.Target {
		case "", "vCluster", "host":
		default:
			errs = append(errs, field.NotSupported(appPath.Child("target"), application.Target, []string{"vCluster", "host"}))
		}

		if len(application.Inline) == 0 && application.Template == nil {
			errs = append(errs, field.Required(appPath, "either inline or template must be set"))
		}
		if len(application.Inline) > 0 && application.Template != nil {
			errs = append(errs, field.Invalid(appPath, application, "inline and template are mutually exclusive"))
		}
		if application.Template != nil && application.Template.Name == "" {
			errs = append(errs, field.Required(appPath.Child("template", "name"), "template.name is required"))
		}
	}

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
		case constants.StorageTypeAzure:
			azure := auto.Storage.Azure
			if azure.BlobURL == "" {
				errs = append(errs, field.Invalid(
					fldPath.Child("snapshots", "auto", "storage", "azure", "blobUrl"),
					azure.BlobURL,
					"storage type is set to 'azure', but url is missing",
				))
			}
		default:
			errs = append(errs, field.Invalid(
				fldPath.Child("snapshots", "auto", "storage", "type"),
				auto.Storage.Type,
				fmt.Sprintf("storage type must be one of: %s, %s, %s, %s", constants.StorageTypeContainer, constants.StorageTypeS3, constants.StorageTypeOCI, constants.StorageTypeAzure),
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

	errs = append(errs, validateDuration(
		fldPath.Child("deletion", "auto", "afterInactivity"),
		string(deletion.Auto.AfterInactivity),
	)...)

	return errs
}

// check if vcluster chart version is compatible with snapshots requests
func SupportsSnapshotRequests(release storagev1.VirtualClusterHelmRelease) bool {
	return semver.Compare("v"+release.Chart.Version, "v0.30.0-alpha.0") == 1
}

// ValidateStandaloneSnapshots rejects snapshot storage a standalone instance cannot work with: the
// platform reads the storage itself there, so anything reachable only from inside the tenant is
// unusable. Separate from ValidateSnapshots because the same config is fine for a pod-based instance.
func ValidateStandaloneSnapshots(fldPath *field.Path, snapshots *Snapshots) field.ErrorList {
	if snapshots == nil || snapshots.Auto == nil || snapshots.Auto.Storage == nil {
		return nil
	}

	storage := snapshots.Auto.Storage
	storagePath := fldPath.Child("snapshots", "auto", "storage")

	if storage.Type == constants.StorageTypeContainer {
		return field.ErrorList{field.Invalid(
			storagePath.Child("type"),
			storage.Type,
			"container storage is not supported on a standalone virtual cluster: the volume exists only inside the tenant, which the platform cannot list or prune",
		)}
	}

	if usesAmbientCloudIdentity(storage) {
		return field.ErrorList{field.Required(
			storagePath.Child(storage.Type, "credential"),
			"a credential secret is required on a standalone virtual cluster: an ambient cloud identity exists only inside the tenant, and the platform resolves the credentials itself",
		)}
	}

	// the URL is the snapshot location, so it is written into the tenant's request ConfigMap in
	// plaintext; a SAS token there is a credential persisted where every other backend keeps none
	if storage.Type == constants.StorageTypeAzure && blobURLContainsSAS(storage.Azure.BlobURL) {
		return field.ErrorList{field.Invalid(
			storagePath.Child("azure", "blobUrl"),
			"[redacted]",
			"a SAS token in the blob URL is not supported on a standalone virtual cluster, use the credential secret instead",
		)}
	}

	return nil
}

// blobURLContainsSAS mirrors the check in pkg/snapshot/storage/azure, duplicated rather than imported
// so validation does not pull the Azure SDK in.
func blobURLContainsSAS(blobURL string) bool {
	if blobURL == "" {
		return false
	}

	parsedURL, err := url.Parse(blobURL)
	if err != nil {
		// an unparseable URL fails later where it is used
		return false
	}

	return parsedURL.Query().Has("sig") || strings.Contains(parsedURL.RawQuery, "sig=")
}

// usesAmbientCloudIdentity reports whether the storage relies on an identity attached to the workload
// (IRSA, an instance profile, a managed identity) rather than a configured credential.
func usesAmbientCloudIdentity(storage *SnapshotStorage) bool {
	switch storage.Type {
	case constants.StorageTypeS3:
		return storage.S3.Credential == nil
	case constants.StorageTypeOCI:
		return storage.OCI.Credential == nil && storage.OCI.Username == ""
	case constants.StorageTypeAzure:
		return storage.Azure.Credential == nil
	}

	return false
}

// SupportsStandaloneSnapshots reports whether the tenant can take platform-scheduled snapshots while
// standalone. This needs more than SupportsSnapshotRequests: standalone gets no options Secret pushed
// into it, so the runtime has to pull its storage credentials from the platform, which it learned to do
// in v0.37. An older standalone reconciles the request, finds no Secret and fails the snapshot.
//
// Compared on major.minor so every 0.37 build qualifies, prereleases included. Comparing full versions
// would order "alpha" before "beta" before "next" and split the 0.37 development stream in three.
func SupportsStandaloneSnapshots(release storagev1.VirtualClusterHelmRelease) bool {
	return semver.Compare(semver.MajorMinor(chartSemver(release.Chart.Version)), "v0.37") >= 0
}

// chartSemver puts a chart version into the form x/mod/semver expects: trimmed, with exactly one
// leading "v". Blindly prefixing breaks on a version that already carries one — MajorMinor("vv0.37.0")
// is "", which compares below every real version. Chart versions come from whatever a registering
// vCluster reports (registervirtualcluster/rest.go), so that shape is not hypothetical.
func chartSemver(version string) string {
	version = strings.TrimSpace(version)
	if len(version) > 0 && (version[0] == 'v' || version[0] == 'V') {
		version = version[1:]
	}

	return "v" + version
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
