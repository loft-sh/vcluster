package vclusterconfig

import (
	"encoding/json"
	"time"

	"sigs.k8s.io/yaml"
)

// LegacyPlatformMutator is a function that can mutate a LegacyPlatformConfig before conversion.
// This allows callers to inject platform-specific logic (e.g., sleep mode migration from annotations).
type LegacyPlatformMutator func(*LegacyPlatformConfig)

// ConvertPlatformConfig converts a LegacyPlatformConfig to the new PlatformConfig format
// and returns it as a YAML string.
func ConvertPlatformConfig(legacyConfig *LegacyPlatformConfig) (string, error) {
	cfg := &PlatformConfig{}

	if legacyConfig.AutoSnapshot != nil {
		cfg.Snapshots = convertAutoSnapshotToSnapshots(legacyConfig.AutoSnapshot)
	}
	if legacyConfig.AutoDelete != nil {
		cfg.Deletion = convertAutoDeleteToDeletion(legacyConfig.AutoDelete)
	}
	if legacyConfig.AutoSleep != nil {
		cfg.Sleep = convertAutoSleepToSleep(legacyConfig.AutoSleep)
	}
	if legacyConfig.APIKey != nil || legacyConfig.Project != "" {
		cfg.Platform = convertLegacyPlatformToPlatform(legacyConfig)
	}

	return marshalPruned(cfg)
}

// ConvertSleepMode converts a LegacySleepMode to the new Sleep format
// and returns it as a YAML string.
func ConvertSleepMode(legacySleepMode *LegacySleepMode) (string, error) {
	if legacySleepMode == nil {
		return "", nil
	}

	cfg := &PlatformConfig{
		Sleep: convertSleepModeToSleep(legacySleepMode),
	}

	return marshalPruned(cfg)
}

// ConvertExternalPlatformValues extracts the external.platform section from values,
// applies any mutators, converts it to the new format, and returns the result as a map.
// Returns nil if no external.platform section exists.
func ConvertExternalPlatformValues(values map[string]any, mutators ...LegacyPlatformMutator) (map[string]any, error) {
	external, ok := values["external"].(map[string]any)
	if !ok {
		return nil, nil
	}
	platform, ok := external["platform"]
	if !ok {
		return nil, nil
	}

	rawPlatform, err := yaml.Marshal(platform)
	if err != nil {
		return nil, err
	}

	legacyConfig := &LegacyPlatformConfig{}
	if err := yaml.UnmarshalStrict(rawPlatform, legacyConfig); err != nil {
		return nil, err
	}

	// Apply mutators (e.g., sleep mode migration from annotations)
	for _, mutator := range mutators {
		mutator(legacyConfig)
	}

	rawConverted, err := ConvertPlatformConfig(legacyConfig)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := yaml.Unmarshal([]byte(rawConverted), &result); err != nil {
		return nil, err
	}

	return result, nil
}

// ConvertSleepModeValues extracts the sleepMode section from values,
// converts it to the new format, and returns the result as a map.
// Returns nil if no sleepMode section exists.
func ConvertSleepModeValues(values map[string]any) (map[string]any, error) {
	sleepMode, ok := values["sleepMode"]
	if !ok {
		return nil, nil
	}

	rawSleepMode, err := yaml.Marshal(sleepMode)
	if err != nil {
		return nil, err
	}

	legacySleepMode := &LegacySleepMode{}
	if err := yaml.UnmarshalStrict(rawSleepMode, legacySleepMode); err != nil {
		return nil, err
	}

	rawConverted, err := ConvertSleepMode(legacySleepMode)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := yaml.Unmarshal([]byte(rawConverted), &result); err != nil {
		return nil, err
	}

	return result, nil
}

// convertSleepModeToSleep converts deprecated SleepMode to new Sleep type
func convertSleepModeToSleep(sm *LegacySleepMode) *Sleep {
	if sm == nil {
		return nil
	}

	var wakeup *SleepAutoWakeup
	if sm.AutoWakeup != nil && sm.AutoWakeup.Schedule != "" {
		wakeup = &SleepAutoWakeup{
			Schedule: sm.AutoWakeup.Schedule,
		}
	}

	auto := &SleepAuto{
		Wakeup:   wakeup,
		Timezone: sm.TimeZone,
	}

	if sm.AutoSleep != nil {
		auto.AfterInactivity = sm.AutoSleep.AfterInactivity
		auto.Schedule = sm.AutoSleep.Schedule
		auto.Exclude = sm.AutoSleep.Exclude
	}

	return &Sleep{
		Auto: auto,
	}
}

// convertAutoSleepToSleep converts deprecated AutoSleep to new Sleep type
func convertAutoSleepToSleep(as *LegacyAutoSleep) *Sleep {
	if as == nil {
		return nil
	}

	var afterInactivity string
	if as.AfterInactivity > 0 {
		afterInactivity = (time.Duration(as.AfterInactivity) * time.Second).String()
	}

	var wakeup *SleepAutoWakeup
	if as.AutoWakeup != nil && as.AutoWakeup.Schedule != "" {
		wakeup = &SleepAutoWakeup{
			Schedule: as.AutoWakeup.Schedule,
		}
	}

	return &Sleep{
		Auto: &SleepAuto{
			AfterInactivity: Duration(afterInactivity),
			Schedule:        as.Schedule,
			Wakeup:          wakeup,
			Timezone:        as.Timezone,
		},
	}
}

// convertAutoSnapshotToSnapshots converts deprecated AutoSnapshot to new Snapshots type
func convertAutoSnapshotToSnapshots(as *LegacyAutoSnapshot) *Snapshots {
	if as == nil {
		return nil
	}
	return &Snapshots{
		Auto: &SnapshotsAuto{
			Schedule: as.Schedule,
			Timezone: as.Timezone,
			Retention: &SnapshotRetention{
				Period:       as.Retention.Period,
				MaxSnapshots: as.Retention.MaxSnapshots,
			},
			Storage: &SnapshotStorage{
				Type: as.Storage.Type,
				S3: SnapshotStorageS3{
					Url:        as.Storage.S3.Url,
					Credential: as.Storage.S3.Credential,
				},
				OCI: SnapshotStorageOCI{
					Repository: as.Storage.OCI.Repository,
					Credential: as.Storage.OCI.Credential,
					Username:   as.Storage.OCI.Username,
					Password:   as.Storage.OCI.Password,
				},
				Container: SnapshotStorageContainer{
					Path: as.Storage.Container.Path,
					Volume: SnapshotStorageContainerVolume{
						Name: as.Storage.Container.Volume.Name,
						Path: as.Storage.Container.Volume.Path,
					},
				},
			},
			Volumes: &SnapshotVolumes{
				Enabled: as.Volumes.Enabled,
			},
		},
	}
}

// convertAutoDeleteToDeletion converts deprecated AutoDelete to new Deletion type
func convertAutoDeleteToDeletion(ad *LegacyAutoDelete) *Deletion {
	if ad == nil {
		return nil
	}
	var afterInactivity string
	if ad.AfterInactivity > 0 {
		afterInactivity = (time.Duration(ad.AfterInactivity) * time.Second).String()
	}
	return &Deletion{
		Auto: &DeletionAuto{
			AfterInactivity: Duration(afterInactivity),
		},
	}
}

// convertLegacyPlatformToPlatform converts deprecated PlatformConfig fields to new Platform type
func convertLegacyPlatformToPlatform(pc *LegacyPlatformConfig) *Platform {
	if pc == nil {
		return nil
	}
	p := &Platform{
		Project: pc.Project,
	}
	// APIKey is interface{} in legacy, need to handle type assertion
	if apiKey, ok := pc.APIKey.(map[string]any); ok {
		if secretName, ok := apiKey["secretName"].(string); ok {
			p.APIKey.SecretName = secretName
		}
		if namespace, ok := apiKey["namespace"].(string); ok {
			p.APIKey.Namespace = namespace
		}
		if createRBAC, ok := apiKey["createRBAC"].(bool); ok {
			p.APIKey.CreateRBAC = &createRBAC
		}
	}
	return p
}

// marshalPruned marshals the config to YAML, pruning empty nested maps.
func marshalPruned(cfg any) (string, error) {
	raw := map[string]any{}
	data, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return "", err
	}

	pruned := prune(raw)
	if pruned == nil {
		pruned = map[string]any{}
	}

	out, err := yaml.Marshal(pruned)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func prune(in any) any {
	switch v := in.(type) {
	case []any:
		for i, elem := range v {
			v[i] = prune(elem)
		}
		return v
	case map[string]any:
		if len(v) == 0 {
			return nil
		}
		for k, val := range v {
			v[k] = prune(val)
			if v[k] == nil && val != nil {
				delete(v, k)
			}
		}
		if len(v) == 0 {
			return nil
		}
		return v
	default:
		return in
	}
}
