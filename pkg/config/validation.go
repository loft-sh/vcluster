package config

import (
	"crypto/x509"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/loft-sh/vcluster/pkg/constants"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/util/toleration"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/api/validation"
)

var allowedPodSecurityStandards = map[string]bool{
	"privileged": true,
	"baseline":   true,
	"restricted": true,
}

var verbs = []string{"get", "list", "create", "update", "patch", "watch", "delete", "deletecollection"}

func ValidateConfigAndSetDefaults(vConfig *VirtualClusterConfig) error {
	// check the value of pod security standard
	if vConfig.Policies.PodSecurityStandard != "" && !allowedPodSecurityStandards[vConfig.Policies.PodSecurityStandard] {
		return fmt.Errorf("invalid argument enforce-pod-security-standard=%s, must be one of: privileged, baseline, restricted", vConfig.Policies.PodSecurityStandard)
	}

	// parse tolerations
	for _, t := range vConfig.Sync.ToHost.Pods.EnforceTolerations {
		_, err := toleration.ParseToleration(t)
		if err != nil {
			return err
		}
	}

	// check if enable scheduler works correctly
	if vConfig.ControlPlane.Advanced.VirtualScheduler.Enabled && !vConfig.Sync.FromHost.Nodes.Selector.All && len(vConfig.Sync.FromHost.Nodes.Selector.Labels) == 0 {
		vConfig.Sync.FromHost.Nodes.Selector.All = true
	}

	// enable additional controllers required for scheduling with storage
	if vConfig.ControlPlane.Advanced.VirtualScheduler.Enabled && vConfig.Sync.ToHost.PersistentVolumeClaims.Enabled {
		if vConfig.Sync.FromHost.CSINodes.Enabled == "auto" {
			vConfig.Sync.FromHost.CSINodes.Enabled = "true"
		}
		if vConfig.Sync.FromHost.CSIStorageCapacities.Enabled == "auto" {
			vConfig.Sync.FromHost.CSIStorageCapacities.Enabled = "true"
		}
		if vConfig.Sync.FromHost.CSIDrivers.Enabled == "auto" {
			vConfig.Sync.FromHost.CSIDrivers.Enabled = "true"
		}
		if vConfig.Sync.FromHost.StorageClasses.Enabled == "auto" && !vConfig.Sync.ToHost.StorageClasses.Enabled {
			vConfig.Sync.FromHost.StorageClasses.Enabled = "true"
		}
	}

	// check if embedded database and multiple replicas
	if vConfig.Config.BackingStoreType() == config.StoreTypeEmbeddedDatabase && vConfig.ControlPlane.StatefulSet.HighAvailability.Replicas > 1 {
		return fmt.Errorf("embedded database is not supported with multiple replicas")
	}

	// check if custom resources have correct scope
	for key, customResource := range vConfig.Sync.ToHost.CustomResources {
		if customResource.Scope != "" && customResource.Scope != config.ScopeNamespaced {
			return fmt.Errorf("unsupported scope %s for sync.toHost.customResources['%s'].scope. Only 'Namespaced' is allowed", customResource.Scope, key)
		}
		err := validatePatches(patchesValidation{basePath: "sync.toHost.customResources." + key, patches: customResource.Patches})
		if err != nil {
			return err
		}
	}
	if err := validateFromHostSyncCustomResources(vConfig.Sync.FromHost.CustomResources); err != nil {
		return err
	}

	// validate sync patches
	err := validatePatches(
		patchesValidation{basePath: "sync.toHost.configMaps", patches: vConfig.Sync.ToHost.ConfigMaps.Patches},
		patchesValidation{basePath: "sync.toHost.secrets", patches: vConfig.Sync.ToHost.Secrets.Patches},
		patchesValidation{basePath: "sync.toHost.endpoints", patches: vConfig.Sync.ToHost.Endpoints.Patches},
		patchesValidation{basePath: "sync.toHost.services", patches: vConfig.Sync.ToHost.Services.Patches},
		patchesValidation{basePath: "sync.toHost.pods", patches: vConfig.Sync.ToHost.Pods.Patches},
		patchesValidation{basePath: "sync.toHost.serviceAccounts", patches: vConfig.Sync.ToHost.ServiceAccounts.Patches},
		patchesValidation{basePath: "sync.toHost.ingresses", patches: vConfig.Sync.ToHost.Ingresses.Patches},
		patchesValidation{basePath: "sync.toHost.networkPolicies", patches: vConfig.Sync.ToHost.NetworkPolicies.Patches},
		patchesValidation{basePath: "sync.toHost.persistentVolumeClaims", patches: vConfig.Sync.ToHost.PersistentVolumeClaims.Patches},
		patchesValidation{basePath: "sync.toHost.persistentVolumes", patches: vConfig.Sync.ToHost.PersistentVolumes.Patches},
		patchesValidation{basePath: "sync.toHost.podDisruptionBudgets", patches: vConfig.Sync.ToHost.PodDisruptionBudgets.Patches},
		patchesValidation{basePath: "sync.toHost.priorityClasses", patches: vConfig.Sync.ToHost.PriorityClasses.Patches},
		patchesValidation{basePath: "sync.toHost.storageClasses", patches: vConfig.Sync.ToHost.StorageClasses.Patches},
		patchesValidation{basePath: "sync.toHost.volumeSnapshots", patches: vConfig.Sync.ToHost.VolumeSnapshots.Patches},
		patchesValidation{basePath: "sync.toHost.volumeSnapshotContents", patches: vConfig.Sync.ToHost.VolumeSnapshotContents.Patches},
		patchesValidation{basePath: "sync.fromHost.nodes", patches: vConfig.Sync.FromHost.Nodes.Patches},
		patchesValidation{basePath: "sync.fromHost.storageClasses", patches: vConfig.Sync.FromHost.StorageClasses.Patches},
		patchesValidation{basePath: "sync.fromHost.priorityClasses", patches: vConfig.Sync.FromHost.PriorityClasses.Patches},
		patchesValidation{basePath: "sync.fromHost.ingressClasses", patches: vConfig.Sync.FromHost.IngressClasses.Patches},
		patchesValidation{basePath: "sync.fromHost.runtimeClasses", patches: vConfig.Sync.FromHost.RuntimeClasses.Patches},
		patchesValidation{basePath: "sync.fromHost.csiDrivers", patches: vConfig.Sync.FromHost.CSIDrivers.Patches},
		patchesValidation{basePath: "sync.fromHost.csiNodes", patches: vConfig.Sync.FromHost.CSINodes.Patches},
		patchesValidation{basePath: "sync.fromHost.csiStorageCapacities", patches: vConfig.Sync.FromHost.CSIStorageCapacities.Patches},
		patchesValidation{basePath: "sync.fromHost.events", patches: vConfig.Sync.FromHost.Events.Patches},
		patchesValidation{basePath: "sync.fromHost.volumeSnapshotClasses", patches: vConfig.Sync.FromHost.VolumeSnapshotClasses.Patches},
		patchesValidation{basePath: "sync.fromHost.configMaps", patches: vConfig.Sync.FromHost.ConfigMaps.Patches},
	)
	if err != nil {
		return err
	}

	// disallow old and new generic sync to be used together
	if len(vConfig.Sync.ToHost.CustomResources) > 0 || len(vConfig.Sync.FromHost.CustomResources) > 0 {
		// check if generic sync exports are used
		if len(vConfig.Experimental.GenericSync.Exports) > 0 {
			return errors.New("experimental.genericSync.exports is not allowed when using sync.toHost.customResources or sync.fromHost.customResources")
		}

		// check if generic sync imports are used
		if len(vConfig.Experimental.GenericSync.Imports) > 0 {
			return errors.New("experimental.genericSync.imports is not allowed when using sync.toHost.customResources or sync.fromHost.customResources")
		}

		// check if hooks are used
		if vConfig.Experimental.GenericSync.Hooks != nil && (len(vConfig.Experimental.GenericSync.Hooks.HostToVirtual) > 0 || len(vConfig.Experimental.GenericSync.Hooks.VirtualToHost) > 0) {
			return errors.New("experimental.genericSync.hooks is not allowed when using sync.toHost.customResources or sync.fromHost.customResources. Please use sync.*.patches.expression instead")
		}
	}

	// check if nodes controller needs to be enabled
	if vConfig.ControlPlane.Advanced.VirtualScheduler.Enabled && !vConfig.Sync.FromHost.Nodes.Enabled {
		return errors.New("sync.fromHost.nodes.enabled is false, but required if using virtual scheduler")
	}

	// check if storage classes and host storage classes are enabled at the same time
	if vConfig.Sync.FromHost.StorageClasses.Enabled == "true" && vConfig.Sync.ToHost.StorageClasses.Enabled {
		return errors.New("you cannot enable both sync.fromHost.storageClasses.enabled and sync.toHost.storageClasses.enabled at the same time. Choose only one of them")
	}

	if vConfig.Sync.FromHost.PriorityClasses.Enabled && vConfig.Sync.ToHost.PriorityClasses.Enabled {
		return errors.New("cannot sync priorityclasses to and from host at the same time")
	}

	// volumesnapshots and volumesnapshotcontents are dependant on each other
	if vConfig.Sync.ToHost.VolumeSnapshotContents.Enabled && !vConfig.Sync.ToHost.VolumeSnapshots.Enabled {
		return errors.New("when syncing volume snapshots contents to the host, one must set sync.toHost.volumeSnapshots.enabled to true")
	}
	if vConfig.Sync.ToHost.VolumeSnapshots.Enabled && !vConfig.Sync.ToHost.VolumeSnapshotContents.Enabled {
		return errors.New("when syncing volume snapshots to the host, one must set sync.toHost.volumeSnapshotContents.enabled to true")
	}

	// validate central admission control
	err = validateCentralAdmissionControl(vConfig)
	if err != nil {
		return err
	}

	// validate generic sync config
	err = validateGenericSyncConfig(vConfig.Experimental.GenericSync)
	if err != nil {
		return fmt.Errorf("validate experimental.genericSync")
	}

	// validate distro
	err = validateDistro(vConfig)
	if err != nil {
		return err
	}

	err = validateK0sAndNoExperimentalKubeconfig(vConfig)
	if err != nil {
		return err
	}

	// check deny proxy requests
	for _, c := range vConfig.Experimental.DenyProxyRequests {
		err := validateCheck(c)
		if err != nil {
			return err
		}
	}

	// check resolve dns
	err = validateMappings(vConfig.Networking.ResolveDNS)
	if err != nil {
		return err
	}

	// check sync.fromHost.configMaps.selector.mappings
	err = validateFromHostSyncMappings(vConfig.Sync.FromHost.ConfigMaps, "configMaps")
	if err != nil {
		return err
	}

	// set service name
	if vConfig.ControlPlane.Advanced.WorkloadServiceAccount.Name == "" {
		vConfig.ControlPlane.Advanced.WorkloadServiceAccount.Name = "vc-workload-" + vConfig.Name
	}

	// pro validate config
	err = ProValidateConfig(vConfig)
	if err != nil {
		return err
	}

	return nil
}

type patchesValidation struct {
	basePath string
	patches  []config.TranslatePatch
}

func validatePatches(patchesValidation ...patchesValidation) error {
	for _, p := range patchesValidation {
		patches := p.patches
		basePath := p.basePath
		for idx, patch := range patches {
			used := 0
			if patch.Expression != "" || patch.ReverseExpression != "" {
				used++
			}
			if patch.Labels != nil {
				used++
			}
			if patch.Reference != nil {
				used++
			}
			if used > 1 {
				return fmt.Errorf("%s.patches[%d] can only use one of: expression, labels or reference", basePath, idx)
			} else if used == 0 {
				return fmt.Errorf("%s.patches[%d] need to use one of: expression, labels or reference", basePath, idx)
			}
		}
	}

	return nil
}

func validateDistro(config *VirtualClusterConfig) error {
	enabledDistros := 0
	if config.ControlPlane.Distro.K3S.Enabled {
		enabledDistros++
	}
	if config.ControlPlane.Distro.K0S.Enabled {
		enabledDistros++
	}
	if config.ControlPlane.Distro.K8S.Enabled {
		enabledDistros++
	}

	if enabledDistros > 1 {
		return fmt.Errorf("only one distribution can be enabled")
	}
	return nil
}

func validateGenericSyncConfig(config config.ExperimentalGenericSync) error {
	err := validateExportDuplicates(config.Exports)
	if err != nil {
		return err
	}

	for idx, exp := range config.Exports {
		if exp == nil {
			return fmt.Errorf("exports[%d] is required", idx)
		}

		if exp.Kind == "" {
			return fmt.Errorf("exports[%d].kind is required", idx)
		}

		if exp.APIVersion == "" {
			return fmt.Errorf("exports[%d].APIVersion is required", idx)
		}

		for patchIdx, patch := range exp.Patches {
			err := validatePatch(patch)
			if err != nil {
				return fmt.Errorf("invalid exports[%d].patches[%d]: %w", idx, patchIdx, err)
			}
		}

		for patchIdx, patch := range exp.ReversePatches {
			err := validatePatch(patch)
			if err != nil {
				return fmt.Errorf("invalid exports[%d].reversPatches[%d]: %w", idx, patchIdx, err)
			}
		}
	}

	err = validateImportDuplicates(config.Imports)
	if err != nil {
		return err
	}

	for idx, imp := range config.Imports {
		if imp == nil {
			return fmt.Errorf("imports[%d] is required", idx)
		}

		if imp.Kind == "" {
			return fmt.Errorf("imports[%d].kind is required", idx)
		}

		if imp.APIVersion == "" {
			return fmt.Errorf("imports[%d].APIVersion is required", idx)
		}

		for patchIdx, patch := range imp.Patches {
			err := validatePatch(patch)
			if err != nil {
				return fmt.Errorf("invalid imports[%d].patches[%d]: %w", idx, patchIdx, err)
			}
		}

		for patchIdx, patch := range imp.ReversePatches {
			err := validatePatch(patch)
			if err != nil {
				return fmt.Errorf("invalid imports[%d].reversPatches[%d]: %w", idx, patchIdx, err)
			}
		}
	}

	if config.Hooks != nil {
		// HostToVirtual validation
		for idx, hook := range config.Hooks.HostToVirtual {
			for idy, verb := range hook.Verbs {
				if err := validateVerb(verb); err != nil {
					return fmt.Errorf("invalid hooks.hostToVirtual[%d].verbs[%d]: %w", idx, idy, err)
				}
			}

			for idy, patch := range hook.Patches {
				if err := validatePatch(patch); err != nil {
					return fmt.Errorf("invalid hooks.hostToVirtual[%d].patches[%d]: %w", idx, idy, err)
				}
			}
		}

		// VirtualToHost validation
		for idx, hook := range config.Hooks.VirtualToHost {
			for idy, verb := range hook.Verbs {
				if err := validateVerb(verb); err != nil {
					return fmt.Errorf("invalid hooks.virtualToHost[%d].verbs[%d]: %w", idx, idy, err)
				}
			}

			for idy, patch := range hook.Patches {
				if err := validatePatch(patch); err != nil {
					return fmt.Errorf("invalid hooks.virtualToHost[%d].patches[%d]: %w", idx, idy, err)
				}
			}
		}
	}

	return nil
}

func validatePatch(patch *config.Patch) error {
	switch patch.Operation {
	case config.PatchTypeRemove, config.PatchTypeReplace, config.PatchTypeAdd:
		if patch.FromPath != "" {
			return fmt.Errorf("fromPath is not supported for this operation")
		}

		return nil
	case config.PatchTypeRewriteName, config.PatchTypeRewriteLabelSelector, config.PatchTypeRewriteLabelKey, config.PatchTypeRewriteLabelExpressionsSelector:
		return nil
	case config.PatchTypeCopyFromObject:
		if patch.FromPath == "" {
			return fmt.Errorf("fromPath is required for this operation")
		}

		return nil
	default:
		return fmt.Errorf("unsupported patch type %s", patch.Operation)
	}
}

func validateVerb(verb string) error {
	if !slices.Contains(verbs, verb) {
		return fmt.Errorf("invalid verb \"%s\"; expected on of %q", verb, verbs)
	}

	return nil
}

func validateExportDuplicates(exports []*config.Export) error {
	gvks := map[string]bool{}
	for _, e := range exports {
		k := fmt.Sprintf("%s|%s", e.APIVersion, e.Kind)
		_, found := gvks[k]
		if found {
			return fmt.Errorf("duplicate export for APIVersion %s and %s Kind, only one export for each APIVersion+Kind is permitted", e.APIVersion, e.Kind)
		}
		gvks[k] = true
	}

	return nil
}

func validateImportDuplicates(imports []*config.Import) error {
	gvks := map[string]bool{}
	for _, e := range imports {
		k := fmt.Sprintf("%s|%s", e.APIVersion, e.Kind)
		_, found := gvks[k]
		if found {
			return fmt.Errorf("duplicate import for APIVersion %s and %s Kind, only one import for each APIVersion+Kind is permitted", e.APIVersion, e.Kind)
		}
		gvks[k] = true
	}

	return nil
}

func validateCentralAdmissionControl(config *VirtualClusterConfig) error {
	_, _, err := ParseExtraHooks(config.Policies.CentralAdmission.ValidatingWebhooks, config.Policies.CentralAdmission.MutatingWebhooks)
	return err
}

func ParseExtraHooks(valHooks []config.ValidatingWebhookConfiguration, mutHooks []config.MutatingWebhookConfiguration) ([]admissionregistrationv1.ValidatingWebhookConfiguration, []admissionregistrationv1.MutatingWebhookConfiguration, error) {
	decodedVal := make([]string, 0, len(valHooks))
	for _, v := range valHooks {
		bytes, err := yaml.Marshal(v)
		if err != nil {
			return nil, nil, err
		}
		decodedVal = append(decodedVal, string(bytes))
	}
	decodedMut := make([]string, 0, len(mutHooks))
	for _, v := range mutHooks {
		bytes, err := yaml.Marshal(v)
		if err != nil {
			return nil, nil, err
		}
		decodedMut = append(decodedMut, string(bytes))
	}

	validateConfs := make([]admissionregistrationv1.ValidatingWebhookConfiguration, 0, len(valHooks))
	mutateConfs := make([]admissionregistrationv1.MutatingWebhookConfiguration, 0, len(mutHooks))
	for _, v := range decodedVal {
		var valHook admissionregistrationv1.ValidatingWebhookConfiguration
		err := yaml.Unmarshal([]byte(v), &valHook)
		if err != nil {
			return nil, nil, err
		}
		for _, v := range valHook.Webhooks {
			err := validateWebhookClientCfg(v.ClientConfig)
			if err != nil {
				return nil, nil, fmt.Errorf("webhook client config was not valid for ValidatingWebhookConfiguration %s: %w", v.Name, err)
			}
		}
		validateConfs = append(validateConfs, valHook)
	}
	for _, v := range decodedMut {
		var mutHook admissionregistrationv1.MutatingWebhookConfiguration
		err := yaml.Unmarshal([]byte(v), &mutHook)
		if err != nil {
			return nil, nil, err
		}
		for _, v := range mutHook.Webhooks {
			err := validateWebhookClientCfg(v.ClientConfig)
			if err != nil {
				return nil, nil, fmt.Errorf("webhook client config was not valid for MutatingWebhookConfiguration %s: %w", v.Name, err)
			}
		}
		mutateConfs = append(mutateConfs, mutHook)
	}

	return validateConfs, mutateConfs, nil
}

func validateWebhookClientCfg(clientCfg admissionregistrationv1.WebhookClientConfig) error {
	if len(clientCfg.CABundle) != 0 {
		ok := x509.NewCertPool().AppendCertsFromPEM(clientCfg.CABundle)
		if !ok {
			return errors.New("could not parse the CABundle")
		}
	}

	if clientCfg.Service == nil && clientCfg.URL == nil {
		return errors.New("there is no service config")
	}

	if clientCfg.Service != nil && (clientCfg.Service.Name == "" || clientCfg.Service.Namespace == "") {
		return errors.New("namespace or name of the service is missing")
	}

	if clientCfg.URL != nil {
		_, err := url.Parse(*clientCfg.URL)
		if err != nil {
			return errors.New("the url was not valid")
		}
	}

	return nil
}

func validateCheck(check config.DenyRule) error {
	for _, ns := range check.Namespaces {
		errors := validation.ValidateNamespaceName(ns, false)
		if len(errors) != 0 {
			return fmt.Errorf("invalid Namespaces in %q check: %v", check.Name, errors)
		}
	}
	var err error
	for _, r := range check.Rules {
		err = validateWildcardOrExact(r.Verbs, "create", "get", "update", "patch", "delete")
		if err != nil {
			return fmt.Errorf("invalid Verb defined in the %q check: %w", check.Name, err)
		}

		err = validateWildcardOrAny(r.APIGroups)
		if err != nil {
			return fmt.Errorf("invalid APIGroup defined in the %q check: %w", check.Name, err)
		}

		err = validateWildcardOrAny(r.APIVersions)
		if err != nil {
			return fmt.Errorf("invalid APIVersion defined in the %q check: %w", check.Name, err)
		}

		if r.Scope != nil {
			switch *r.Scope {
			case string(admissionregistrationv1.ClusterScope):
			case string(admissionregistrationv1.NamespacedScope):
			case string(admissionregistrationv1.AllScopes):
			default:
				return fmt.Errorf("invalid Scope defined in the %q check: %q", check.Name, *r.Scope)
			}
		}
	}
	return nil
}

func validateWildcardOrExact(values []string, validValues ...string) error {
	if len(values) == 1 && values[0] == "*" {
		return nil
	}
	for _, val := range values {
		if val == "*" {
			return fmt.Errorf("when wildcard(*) is used, it must be the only value in the list")
		}

		// empty list of validValues means any value is valid
		valid := len(validValues) == 0
		for _, v := range validValues {
			if val == v {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid value %q", val)
		}
	}
	return nil
}

func validateWildcardOrAny(values []string) error {
	if len(values) == 1 && values[0] == "*" {
		return nil
	}
	for _, val := range values {
		if val == "*" {
			return fmt.Errorf("when wildcard(*) is used, it must be the only value in the list")
		}
	}
	return nil
}

func validateK0sAndNoExperimentalKubeconfig(c *VirtualClusterConfig) error {
	if c.Distro() != config.K0SDistro {
		return nil
	}
	virtualclusterconfig := c.Experimental.VirtualClusterKubeConfig
	empty := config.VirtualClusterKubeConfig{}
	if virtualclusterconfig != empty {
		return errors.New("config.experimental.VirtualClusterConfig cannot be set for k0s")
	}
	return nil
}

func validateFromHostSyncMappings(s config.EnableSwitchWithResourcesMappings, resourceNamePlural string) error {
	if !s.Enabled {
		return nil
	}
	if len(s.Selector.Mappings) == 0 {
		return fmt.Errorf("config.sync.fromHost.%s.mappings are empty", resourceNamePlural)
	}
	for key, value := range s.Selector.Mappings {
		if !strings.Contains(key, "/") && key != constants.VClusterNamespaceInHostMappingSpecialCharacter {
			return fmt.Errorf("config.sync.fromHost.%s.selector.mappings has key in invalid format: %s (expected NAMESPACE_NAME/NAME, NAMESPACE_NAME/*, /NAME or \"\")", resourceNamePlural, key)
		}
		if !strings.Contains(value, "/") && key != constants.VClusterNamespaceInHostMappingSpecialCharacter {
			return fmt.Errorf("config.sync.fromHost.%s.selector.mappings has value in invalid format: %s (expected NAMESPACE_NAME/NAME or NAMESPACE_NAME/* or NAMESPACE if key is \"\")", resourceNamePlural, value)
		}
		if key == "*" && strings.Contains(value, "/") && !strings.HasSuffix(value, "/*") {
			return fmt.Errorf("config.sync.fromHost.%s.selector.mappings has key \"\" that matches vCluster host namespace but the value is not in NAMESPACE_NAME or NAMESPACE_NAME/* format (value: %s)", resourceNamePlural, value)
		}
		if strings.HasSuffix(key, "/*") && !strings.HasSuffix(value, "/*") {
			return fmt.Errorf(
				"config.sync.fromHost.%s.selector.mappings has key that matches all objects in the namespace: %s "+
					"but value does not: %s. Please make sure that value for this key is in the format of NAMESPACE_NAME/*",
				resourceNamePlural, key, value,
			)
		}
	}
	return nil
}

func validateFromHostSyncCustomResources(customResources map[string]config.SyncFromHostCustomResource) error {
	for key, customResource := range customResources {
		if customResource.Scope != "" && customResource.Scope != config.ScopeCluster && customResource.Scope != config.ScopeNamespaced {
			return fmt.Errorf("unsupported scope %s for sync.fromHost.customResources['%s'].scope. Only 'Cluster' and 'Namespaced' are allowed", customResource.Scope, key)
		}
		if len(customResource.Selector.Mappings) > 0 && customResource.Scope != config.ScopeNamespaced {
			return fmt.Errorf(".selector.mappings are only supported for sync.fromHost.customResources['%s'] with scope 'Namespaced'", key)
		}
		if customResource.Scope == config.ScopeNamespaced && len(customResource.Selector.Mappings) == 0 {
			return fmt.Errorf(".selector.mappings is required for Namespaced scope sync.fromHost.customResources['%s']", key)
		}
		err := validatePatches(patchesValidation{basePath: "sync.fromHost.customResources." + key, patches: customResource.Patches})
		if err != nil {
			return err
		}
	}
	return nil
}

var ProValidateConfig = func(_ *VirtualClusterConfig) error {
	return nil
}
