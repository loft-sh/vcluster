package config

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/url"
	"slices"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/samber/lo"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/loft-sh/vcluster/config"
	cliconfig "github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/util/namespaces"
	"github.com/loft-sh/vcluster/pkg/util/toleration"
)

var allowedPodSecurityStandards = map[string]bool{
	"privileged": true,
	"baseline":   true,
	"restricted": true,
}

var (
	errExportKubeConfigBothSecretAndAdditionalSecretsSet       = errors.New("exportKubeConfig.Secret and exportKubeConfig.AdditionalSecrets cannot be set at the same time")
	errExportKubeConfigAdditionalSecretWithoutNameAndNamespace = errors.New("additional secret must have name and/or namespace set")
	errExportKubeConfigServerNotValid                          = errors.New("exportKubeConfig.Server has to be set to a valid URL (with https:// or http:// prefix)")
)

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
	if vConfig.SchedulingInVirtualClusterEnabled() && !vConfig.Sync.FromHost.Nodes.Selector.All && len(vConfig.Sync.FromHost.Nodes.Selector.Labels) == 0 {
		vConfig.Sync.FromHost.Nodes.Selector.All = true
	}

	// enable additional controllers required for scheduling with storage
	if vConfig.SchedulingInVirtualClusterEnabled() && vConfig.Sync.ToHost.PersistentVolumeClaims.Enabled {
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

	// disallow listing integration CRDs in sync.*.customResources if the corresponding integrations are enabled
	if err := validateEnabledIntegrations(
		vConfig.Sync.ToHost.CustomResources,
		vConfig.Sync.FromHost.CustomResources,
		vConfig.Integrations); err != nil {
		return err
	}

	// validate custom resources are not configured for both sync and proxy
	if err := ValidateCustomResourceSyncProxyConflicts(
		vConfig.Sync.ToHost.CustomResources,
		vConfig.Sync.FromHost.CustomResources,
		vConfig.Experimental.Proxy.CustomResources); err != nil {
		return err
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
	err := ValidateAllSyncPatches(vConfig.Sync)
	if err != nil {
		return err
	}

	// check if nodes controller needs to be enabled
	if vConfig.SchedulingInVirtualClusterEnabled() && !vConfig.Sync.FromHost.Nodes.Enabled {
		return errors.New("sync.fromHost.nodes.enabled is false, but required if using hybrid scheduling or virtual scheduler")
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

	err = validateFromHostSyncMappings(vConfig.Sync.FromHost.Secrets, "secrets")
	if err != nil {
		return err
	}

	// sync.toHost.namespaces validation
	err = namespaces.ValidateNamespaceSyncConfig(&vConfig.Config, vConfig.Name, vConfig.HostNamespace)
	if err != nil {
		return fmt.Errorf("namespace sync: %w", err)
	}

	// if we're running in with namespace sync enabled, we want to sync all objects.
	// otherwise, objects created on host in synced namespaces won't get imported into vCluster.
	if vConfig.Sync.ToHost.Namespaces.Enabled {
		vConfig.Sync.ToHost.Secrets.All = true
		vConfig.Sync.ToHost.ConfigMaps.All = true
	}

	// set service name
	if vConfig.ControlPlane.Advanced.WorkloadServiceAccount.Name == "" {
		vConfig.ControlPlane.Advanced.WorkloadServiceAccount.Name = "vc-workload-" + vConfig.Name
	}

	err = validateAdvancedControlPlaneConfig(vConfig.ControlPlane.Advanced)
	if err != nil {
		return err
	}

	// check config for exporting kubeconfig Secrets
	err = validateExportKubeConfig(vConfig.ExportKubeConfig)
	if err != nil {
		return err
	}

	// pro validate config
	err = ProValidateConfig(vConfig)
	if err != nil {
		return err
	}

	// validate dedicated nodes mode
	err = validatePrivatedNodesMode(vConfig)
	if err != nil {
		return err
	}

	// validate sync.fromHost classes
	err = ValidateSyncFromHostClasses(vConfig.Config.Sync.FromHost)
	if err != nil {
		return err
	}

	// validate deploy.volumeSnapshotController
	err = ValidateVolumeSnapshotController(vConfig.Config.Deploy.VolumeSnapshotController, vConfig.PrivateNodes)
	if err != nil {
		return err
	}
	// auto-enable volume snapshot rules in shared mode
	if !vConfig.Config.PrivateNodes.Enabled && vConfig.RBAC.EnableVolumeSnapshotRules.Enabled == "auto" {
		vConfig.RBAC.EnableVolumeSnapshotRules.Enabled = "true"
	}

	return nil
}

type patchesValidation struct {
	basePath string
	patches  []config.TranslatePatch
}

// ValidateAllSyncPatches validates all sync patches
func ValidateAllSyncPatches(sync config.Sync) error {
	return validatePatches(
		[]patchesValidation{
			{"sync.toHost.configMaps", sync.ToHost.ConfigMaps.Patches},
			{"sync.toHost.secrets", sync.ToHost.Secrets.Patches},
			{"sync.toHost.endpoints", sync.ToHost.Endpoints.Patches},
			{"sync.toHost.services", sync.ToHost.Services.Patches},
			{"sync.toHost.pods", sync.ToHost.Pods.Patches},
			{"sync.toHost.serviceAccounts", sync.ToHost.ServiceAccounts.Patches},
			{"sync.toHost.ingresses", sync.ToHost.Ingresses.Patches},
			{"sync.toHost.namespaces", sync.ToHost.Namespaces.Patches},
			{"sync.toHost.networkPolicies", sync.ToHost.NetworkPolicies.Patches},
			{"sync.toHost.persistentVolumeClaims", sync.ToHost.PersistentVolumeClaims.Patches},
			{"sync.toHost.persistentVolumes", sync.ToHost.PersistentVolumes.Patches},
			{"sync.toHost.podDisruptionBudgets", sync.ToHost.PodDisruptionBudgets.Patches},
			{"sync.toHost.priorityClasses", sync.ToHost.PriorityClasses.Patches},
			{"sync.toHost.resourceClaims", sync.ToHost.ResourceClaims.Patches},
			{"sync.toHost.resourceClaimTemplates", sync.ToHost.ResourceClaimTemplates.Patches},
			{"sync.toHost.storageClasses", sync.ToHost.StorageClasses.Patches},
			{"sync.toHost.volumeSnapshots", sync.ToHost.VolumeSnapshots.Patches},
			{"sync.toHost.volumeSnapshotContents", sync.ToHost.VolumeSnapshotContents.Patches},
			{"sync.fromHost.nodes", sync.FromHost.Nodes.Patches},
			{"sync.fromHost.storageClasses", sync.FromHost.StorageClasses.Patches},
			{"sync.fromHost.priorityClasses", sync.FromHost.PriorityClasses.Patches},
			{"sync.fromHost.ingressClasses", sync.FromHost.IngressClasses.Patches},
			{"sync.fromHost.csiDrivers", sync.FromHost.CSIDrivers.Patches},
			{"sync.fromHost.runtimeClasses", sync.FromHost.RuntimeClasses.Patches},
			{"sync.fromHost.csiNodes", sync.FromHost.CSINodes.Patches},
			{"sync.fromHost.csiStorageCapacities", sync.FromHost.CSIStorageCapacities.Patches},
			{"sync.fromHost.events", sync.FromHost.Events.Patches},
			{"sync.fromHost.volumeSnapshotClasses", sync.FromHost.VolumeSnapshotClasses.Patches},
			{"sync.fromHost.configMaps", sync.FromHost.ConfigMaps.Patches},
			{"sync.fromHost.deviceClasses", sync.FromHost.DeviceClasses.Patches},
		}...,
	)
}

func validatePatches(patchesValidation ...patchesValidation) error {
	for _, p := range patchesValidation {
		patches := p.patches
		basePath := p.basePath
		usedPaths := map[string]int{}
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
			if j, ok := usedPaths[patch.Path]; ok {
				return fmt.Errorf("%s.patches[%d] and %s.patches[%d] have the same path %q", basePath, j, basePath, idx, patch.Path)
			}
			usedPaths[patch.Path] = idx
		}
	}

	return nil
}

func ValidatePlatformProject(ctx context.Context, config *config.Config, loadedConfig *cliconfig.CLI) error {
	platformConfig := config.GetPlatformConfig()
	if platformConfig.Project != "" {
		management, err := platform.NewClientFromConfig(loadedConfig).Management()
		if err != nil {
			return err
		}
		_, err = management.Loft().ManagementV1().Projects().Get(ctx, platformConfig.Project, metav1.GetOptions{})
		if kerrors.IsNotFound(err) {
			return fmt.Errorf("platform project %q not found", platformConfig.Project)
		}
	}

	return nil
}

func ValidateSyncFromHostClasses(fromHost config.SyncFromHost) error {
	errorFn := func(sls config.StandardLabelSelector, path string) error {
		if _, err := sls.ToSelector(); err != nil {
			return fmt.Errorf("invalid sync.fromHost.%s.selector: %w", path, err)
		}
		return nil
	}
	if err := errorFn(fromHost.RuntimeClasses.Selector, "runtimeClasses"); err != nil {
		return err
	}
	if err := errorFn(fromHost.IngressClasses.Selector, "ingressClasses"); err != nil {
		return err
	}
	if err := errorFn(fromHost.PriorityClasses.Selector, "priorityClasses"); err != nil {
		return err
	}
	if err := errorFn(fromHost.StorageClasses.Selector, "storageClasses"); err != nil {
		return err
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

func validateFromHostSyncMappings(s config.EnableSwitchWithResourcesMappings, resourceNamePlural string) error {
	if !s.Enabled {
		return nil
	}
	if len(s.Mappings.ByName) == 0 {
		return fmt.Errorf("config.sync.fromHost.%s.mappings are empty", resourceNamePlural)
	}
	for key, value := range s.Mappings.ByName {
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
		if err := validateFromHostMappingEntry(key, value, resourceNamePlural); err != nil {
			return err
		}
	}
	return nil
}

func validateFromHostMappingEntry(key, value, resourceNamePlural string) error {
	if strings.Count(key, "/") > 1 || strings.Count(value, "/") > 1 {
		return fmt.Errorf("config.sync.fromHost.%s.selector.mappings has key:value pair in invalid format: %s:%s (expected NAMESPACE_NAME/NAME, NAMESPACE_NAME/*, /NAME or \"\")", resourceNamePlural, key, value)
	}
	hostRef := strings.Split(key, "/")
	virtualRef := strings.Split(value, "/")
	if key != "" && len(hostRef) > 0 {
		errs := validation.ValidateNamespaceName(hostRef[0], false)
		if len(errs) > 0 && hostRef[0] != "" {
			return fmt.Errorf("config.sync.fromHost.%s.selector.mappings parsed host namespace is not valid namespace name %s", resourceNamePlural, errs)
		}
		if err := validateFromHostSyncMappingObjectName(hostRef, resourceNamePlural); err != nil {
			return err
		}
	}
	if len(virtualRef) > 0 {
		errs := validation.ValidateNamespaceName(virtualRef[0], false)
		if len(errs) > 0 {
			return fmt.Errorf("config.sync.fromHost.%s.selector.mappings parsed virtual namespace is not valid namespace name %s", resourceNamePlural, errs)
		}
		if err := validateFromHostSyncMappingObjectName(virtualRef, resourceNamePlural); err != nil {
			return err
		}
	}
	return nil
}

func validateFromHostSyncCustomResources(customResources map[string]config.SyncFromHostCustomResource) error {
	for key, customResource := range customResources {
		if customResource.Scope != "" && customResource.Scope != config.ScopeCluster && customResource.Scope != config.ScopeNamespaced {
			return fmt.Errorf("unsupported scope %s for sync.fromHost.customResources['%s'].scope. Only 'Cluster' and 'Namespaced' are allowed", customResource.Scope, key)
		}
		if len(customResource.Mappings.ByName) > 0 && customResource.Scope != config.ScopeNamespaced {
			return fmt.Errorf(".selector.mappings are only supported for sync.fromHost.customResources['%s'] with scope 'Namespaced'", key)
		}
		if customResource.Scope == config.ScopeNamespaced && len(customResource.Mappings.ByName) == 0 {
			return fmt.Errorf(".selector.mappings is required for Namespaced scope sync.fromHost.customResources['%s']", key)
		}
		err := validatePatches(patchesValidation{basePath: "sync.fromHost.customResources." + key, patches: customResource.Patches})
		if err != nil {
			return err
		}

		if customResource.Scope == config.ScopeNamespaced {
			for host, virtual := range customResource.Mappings.ByName {
				if err := validateFromHostMappingEntry(host, virtual, key); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func validateFromHostSyncMappingObjectName(objRef []string, resourceNamePlural string) error {
	var errs []string
	if len(objRef) == 2 && objRef[1] != "" && objRef[1] != "*" {
		errs = validation.NameIsDNSSubdomain(objRef[1], false)
	}
	if len(errs) > 0 {
		return fmt.Errorf("config.sync.fromHost.%s.selector.mappings parsed object name from key (%s) is not valid name %s", resourceNamePlural, strings.Join(objRef, "/"), errs)
	}
	return nil
}

func validateExportKubeConfig(exportKubeConfig config.ExportKubeConfig) error {
	// You cannot set both Secret and AdditionalSecrets at the same time.
	if exportKubeConfig.Secret.IsSet() && len(exportKubeConfig.AdditionalSecrets) > 0 {
		return errExportKubeConfigBothSecretAndAdditionalSecretsSet
	}
	for _, additionalSecret := range exportKubeConfig.AdditionalSecrets {
		// You must set at least Name or Namespace for every additional kubeconfig secret.
		if additionalSecret.Name == "" && additionalSecret.Namespace == "" {
			return errExportKubeConfigAdditionalSecretWithoutNameAndNamespace
		}
	}

	if err := validateExportKubeConfigServer(exportKubeConfig.Server); err != nil {
		return errExportKubeConfigServerNotValid
	}
	return nil
}

func validateExportKubeConfigServer(server string) error {
	if server == "" {
		return nil
	}
	hasProto := strings.HasPrefix(server, "https://") || strings.HasPrefix(server, "http://")
	if _, err := url.Parse(server); err != nil || !hasProto {
		return errExportKubeConfigServerNotValid
	}
	return nil
}

func validateEnabledIntegrations(
	toHostCustomResources map[string]config.SyncToHostCustomResource,
	fromHostCustomResources map[string]config.SyncFromHostCustomResource,
	integrations config.Integrations) error {
	if err := validateIstioEnabled(toHostCustomResources, integrations.Istio); err != nil {
		return err
	}
	if err := validateCertManagerEnabled(toHostCustomResources, fromHostCustomResources, integrations.CertManager); err != nil {
		return err
	}
	if err := validateExternalSecretsEnabled(toHostCustomResources, integrations.ExternalSecrets); err != nil {
		return err
	}
	if err := validateKubeVirtEnabled(toHostCustomResources, integrations.KubeVirt); err != nil {
		return err
	}
	return nil
}

func validateIstioEnabled(
	toHostCustomResources map[string]config.SyncToHostCustomResource,
	istioIntegration config.Istio) error {
	if !istioIntegration.Enabled {
		return nil
	}
	for crdName, crdConfig := range toHostCustomResources {
		if crdConfig.Enabled &&
			(crdName == "destinationrules.networking.istio.io" && istioIntegration.Sync.ToHost.DestinationRules.Enabled ||
				crdName == "virtualservices.networking.istio.io" && istioIntegration.Sync.ToHost.VirtualServices.Enabled ||
				crdName == "gateways.networking.istio.io" && istioIntegration.Sync.ToHost.Gateways.Enabled) {
			return fmt.Errorf("istio integration is enabled but istio custom resource (%s) is also set in the sync.toHost.customResources. "+
				"This is not supported, please remove the entry from sync.toHost.customResources", crdName)
		}
	}
	return nil
}

func validateCertManagerEnabled(
	toHostCustomResources map[string]config.SyncToHostCustomResource,
	fromHostCustomResource map[string]config.SyncFromHostCustomResource,
	certManagerIntegration config.CertManager) error {
	if !certManagerIntegration.Enabled {
		return nil
	}
	errMsg := "cert-manager integration is enabled but cert-manager custom resource (%s) is also set in the sync.%[2]s.customResources. " +
		"This is not supported, please remove the entry from sync.%[2]s.customResources"
	for crdName, crdConfig := range toHostCustomResources {
		if crdConfig.Enabled &&
			(crdName == "certificates.cert-manager.io" && certManagerIntegration.Sync.ToHost.Certificates.Enabled ||
				crdName == "issuers.cert-manager.io" && certManagerIntegration.Sync.ToHost.Issuers.Enabled) {
			return fmt.Errorf(errMsg, crdName, "toHost")
		}
	}
	for crdName, crdConfig := range fromHostCustomResource {
		if crdConfig.Enabled && crdName == "clusterissuers.cert-manager.io" && certManagerIntegration.Sync.FromHost.ClusterIssuers.Enabled {
			return fmt.Errorf(errMsg, crdName, "fromHost")
		}
	}
	return nil
}

func validateExternalSecretsEnabled(
	toHostCustomResources map[string]config.SyncToHostCustomResource,
	externalSecretsIntegration config.ExternalSecrets) error {
	if !externalSecretsIntegration.Enabled {
		return nil
	}
	for crdName, crdConfig := range toHostCustomResources {
		if crdConfig.Enabled &&
			(crdName == "externalsecrets.external-secrets.io" && externalSecretsIntegration.Enabled ||
				crdName == "secretstores.external-secrets.io" && externalSecretsIntegration.Sync.ToHost.Stores.Enabled ||
				crdName == "clustersecretstores.external-secrets.io" && externalSecretsIntegration.Sync.FromHost.ClusterStores.Enabled) {
			return fmt.Errorf("external-secrets integration is enabled but external-secrets custom resource (%s) is also set in the sync.toHost.customResources. "+
				"This is not supported, please remove the entry from sync.toHost.customResources", crdName)
		}
	}
	return nil
}

func validateKubeVirtEnabled(
	toHostCustomResources map[string]config.SyncToHostCustomResource,
	kubeVirtIntegration config.KubeVirt) error {
	if !kubeVirtIntegration.Enabled {
		return nil
	}
	for crdName, crdConfig := range toHostCustomResources {
		if crdConfig.Enabled &&
			(isIn(crdName, "datavolumes.cdi.kubevirt.io", "datavolumes/status.cdi.kubevirt.io") && kubeVirtIntegration.Sync.DataVolumes.Enabled ||
				isIn(crdName, "virtualmachineinstancemigrations.kubevirt.io", "virtualmachineinstancemigrations/status.kubevirt.io") && kubeVirtIntegration.Sync.VirtualMachineInstanceMigrations.Enabled ||
				isIn(crdName, "virtualmachineinstances.kubevirt.io", "virtualmachineinstances/status.kubevirt.io") && kubeVirtIntegration.Sync.VirtualMachineInstances.Enabled ||
				isIn(crdName, "virtualmachines.kubevirt.io", "virtualmachines/status.kubevirt.io") && kubeVirtIntegration.Sync.VirtualMachines.Enabled ||
				isIn(crdName, "virtualmachineclones.clone.kubevirt.io", "virtualmachineclones/status.clone.kubevirt.io") && kubeVirtIntegration.Sync.VirtualMachineClones.Enabled ||
				isIn(crdName, "virtualmachinepools.pool.kubevirt.io", "virtualmachinepools/status.pool.kubevirt.io") && kubeVirtIntegration.Sync.VirtualMachinePools.Enabled) {
			return fmt.Errorf("kube-virt integration is enabled but kube-virt custom resource (%s) is also set in the sync.toHost.customResources. "+
				"This is not supported, please remove the entry from sync.toHost.customResources", crdName)
		}
	}
	return nil
}

func isIn(crdName string, s ...string) bool {
	return slices.Contains(s, crdName)
}

func validatePrivatedNodesMode(vConfig *VirtualClusterConfig) error {
	if !vConfig.PrivateNodes.Enabled {
		if vConfig.ControlPlane.Endpoint != "" {
			return fmt.Errorf("endpoint is only supported in private nodes mode")
		}

		return nil
	}

	// validate endpoint
	if vConfig.ControlPlane.Endpoint != "" {
		_, _, err := net.SplitHostPort(vConfig.ControlPlane.Endpoint)
		if err != nil {
			return fmt.Errorf("invalid endpoint %s: %w", vConfig.ControlPlane.Endpoint, err)
		}
	}

	// integrations are not supported in private nodes mode
	if vConfig.Integrations.MetricsServer.Enabled {
		return fmt.Errorf("metrics-server integration is not supported in private nodes mode")
	}
	if vConfig.Integrations.CertManager.Enabled {
		return fmt.Errorf("cert-manager integration is not supported in private nodes mode")
	}
	if vConfig.Integrations.ExternalSecrets.Enabled {
		return fmt.Errorf("external-secrets integration is not supported in private nodes mode")
	}
	if vConfig.Integrations.Istio.Enabled {
		return fmt.Errorf("istio integration is not supported in private nodes mode")
	}
	if vConfig.Integrations.KubeVirt.Enabled {
		return fmt.Errorf("kubevirt integration is not supported in private nodes mode")
	}

	// embedded coredns is not supported in private nodes mode
	if vConfig.ControlPlane.CoreDNS.Embedded {
		return fmt.Errorf("coredns is not supported in private nodes mode")
	}

	// host path mapper is not supported in private nodes mode
	if vConfig.ControlPlane.HostPathMapper.Enabled {
		return fmt.Errorf("host path mapper is not supported in private nodes mode")
	}

	// multi-namespace mode is not supported in private nodes mode
	if vConfig.Sync.ToHost.Namespaces.Enabled {
		return fmt.Errorf("multi-namespace mode is not supported in private nodes mode")
	}

	// validate node pools
	nodePoolNames := make(map[string]bool)
	nodePoolProviders := make(map[string]bool)
	for _, nodeProviderConfiguration := range vConfig.PrivateNodes.AutoNodes {
		if nodeProviderConfiguration.Provider == "" {
			return fmt.Errorf("node pool provider is required")
		}

		if nodePoolProviders[nodeProviderConfiguration.Provider] {
			return fmt.Errorf("node pool provider %s is already used. You cannot have two configurations for the same provider", nodeProviderConfiguration.Provider)
		}
		nodePoolProviders[nodeProviderConfiguration.Provider] = true

		for _, staticNodePool := range nodeProviderConfiguration.Static {
			if staticNodePool.Name == "" {
				return fmt.Errorf("node pool name is required")
			}
			if staticNodePool.Quantity < 0 {
				return fmt.Errorf("node pool quantity cannot be negative")
			}
			if nodePoolNames[staticNodePool.Name] {
				return fmt.Errorf("node pool name %s is already used. You cannot have two node pools with the same name", staticNodePool.Name)
			}
			nodePoolNames[staticNodePool.Name] = true

			if err := validateRequirements(staticNodePool.NodeTypeSelector); err != nil {
				return fmt.Errorf("invalid requirements for node pool %s: %w", staticNodePool.Name, err)
			}
		}
		for _, dynamicNodePool := range nodeProviderConfiguration.Dynamic {
			if dynamicNodePool.Name == "" {
				return fmt.Errorf("node pool name is required")
			}
			if nodePoolNames[dynamicNodePool.Name] {
				return fmt.Errorf("node pool name %s is already used. You cannot have two node pools with the same name", dynamicNodePool.Name)
			}
			nodePoolNames[dynamicNodePool.Name] = true

			if err := validateRequirements(dynamicNodePool.NodeTypeSelector); err != nil {
				return fmt.Errorf("invalid requirements for node pool %s: %w", dynamicNodePool.Name, err)
			}
		}
	}

	return nil
}

func ValidateVolumeSnapshotController(volumeSnapshotController config.VolumeSnapshotController, privateNodes config.PrivateNodes) error {
	if volumeSnapshotController.Enabled && !privateNodes.Enabled {
		return fmt.Errorf("volume snapshot-controller is only supported with private nodes")
	}
	return nil
}

var allowedOperators = []string{"", "In", "NotIn", "Exists", "DoesNotExist", "Gt", "Lt"}

func validateRequirements(requirements []config.Requirement) error {
	for _, requirement := range requirements {
		if requirement.Property == "" {
			return fmt.Errorf("requirement property is required")
		}

		if !slices.Contains(allowedOperators, requirement.Operator) {
			return fmt.Errorf("invalid operator %s for property %s, allowed operators are: %s", requirement.Operator, requirement.Property, strings.Join(allowedOperators, ", "))
		}

		if requirement.Value != "" && len(requirement.Values) > 0 {
			return fmt.Errorf("requirement value and values cannot be set at the same time")
		}

		if requirement.Operator == "" || requirement.Operator == "In" || requirement.Operator == "NotIn" {
			if requirement.Value == "" && len(requirement.Values) == 0 {
				return fmt.Errorf("requirement value or values is required if operator is empty, In or NotIn")
			}
		}

		if requirement.Operator == "Exists" || requirement.Operator == "DoesNotExist" {
			if requirement.Value != "" || len(requirement.Values) > 0 {
				return fmt.Errorf("value or values is not allowed for operator %s", requirement.Operator)
			}
		}

		if requirement.Operator == "Gt" || requirement.Operator == "Lt" {
			if requirement.Value == "" && len(requirement.Values) == 0 {
				return fmt.Errorf("value or values is required for operator %s", requirement.Operator)
			}
		}
	}

	return nil
}

func validateAdvancedControlPlaneConfig(controlPlaneAdvanced config.ControlPlaneAdvanced) error {
	if controlPlaneAdvanced.PodDisruptionBudget.Enabled &&
		controlPlaneAdvanced.PodDisruptionBudget.MaxUnavailable != nil &&
		controlPlaneAdvanced.PodDisruptionBudget.MinAvailable != nil {
		return fmt.Errorf("minAvailable and maxUnavailable cannot be used together in a podDisruptionBudget")
	}

	return nil
}

func ValidateCustomResourceSyncProxyConflicts(toHostCustomResources map[string]config.SyncToHostCustomResource, fromHostCustomResources map[string]config.SyncFromHostCustomResource, proxyCustomResources map[string]config.CustomResourceProxy) error {
	// Only consider enabled resources for conflict detection
	enabledToHost := lo.Keys(lo.PickBy(toHostCustomResources, func(_ string, v config.SyncToHostCustomResource) bool { return v.Enabled }))
	enabledFromHost := lo.Keys(lo.PickBy(fromHostCustomResources, func(_ string, v config.SyncFromHostCustomResource) bool { return v.Enabled }))
	enabledProxy := lo.Keys(lo.PickBy(proxyCustomResources, func(_ string, v config.CustomResourceProxy) bool { return v.Enabled }))

	// Check exact key conflicts between toHost and fromHost
	if k := lo.Intersect(enabledToHost, enabledFromHost); len(k) > 0 {
		return fmt.Errorf("custom resource %s exists in sync.toHost.customResources and sync.fromHost.customResources. Syncing is only supported one way", k[0])
	}

	proxyGroups := lo.SliceToMap(enabledProxy, func(key string) (string, string) {
		return extractGroup(key), key
	})
	toHostGroups := lo.SliceToMap(enabledToHost, func(key string) (string, string) {
		return extractGroup(key), key
	})
	fromHostGroups := lo.SliceToMap(enabledFromHost, func(key string) (string, string) {
		return extractGroup(key), key
	})

	// Check toHost groups against proxy groups
	if conflicting := lo.Intersect(lo.Keys(toHostGroups), lo.Keys(proxyGroups)); len(conflicting) > 0 {
		group := conflicting[0]
		return fmt.Errorf("custom resource group %q is used in both sync.toHost.customResources (%s) and proxy.customResources (%s). Resources from the same group cannot be used in both sync and proxy", group, toHostGroups[group], proxyGroups[group])
	}

	// Check fromHost groups against proxy groups
	if conflicting := lo.Intersect(lo.Keys(fromHostGroups), lo.Keys(proxyGroups)); len(conflicting) > 0 {
		group := conflicting[0]
		return fmt.Errorf("custom resource group %q is used in both sync.fromHost.customResources (%s) and proxy.customResources (%s). Resources from the same group cannot be used in both sync and proxy", group, fromHostGroups[group], proxyGroups[group])
	}

	return nil
}

func extractGroup(key string) string {
	// Split by "/" to separate version if present, then parse resource.group
	parts := strings.SplitN(key, "/", 2)
	gr := schema.ParseGroupResource(parts[0])
	return gr.Group
}

func ValidateExperimentalProxyCustomResourcesConfig(cfg map[string]config.CustomResourceProxy) error {
	for resourcePath, resourceConfig := range cfg {
		basePath := fmt.Sprintf("experimental.proxy.customResources['%s']", resourcePath)

		parts := strings.Split(resourcePath, "/")
		if len(parts) != 2 || schema.ParseGroupResource(parts[0]).Resource == "" {
			return fmt.Errorf("%s: invalid resource path %q, expected format 'resource.group/version' (e.g., 'resource.my-org.com/v1')", basePath, resourcePath)
		}
		if resourceConfig.TargetVirtualCluster.Name == "" {
			return fmt.Errorf("%s.targetVirtualCluster is required", basePath)
		}

		if resourceConfig.AccessResources != "" && resourceConfig.AccessResources != config.AccessResourcesModeOwned && resourceConfig.AccessResources != config.AccessResourcesModeAll {
			return fmt.Errorf("%s.accessResources: invalid value %q, must be 'owned' or 'all'", basePath, resourceConfig.AccessResources)
		}
	}

	return nil
}

var ProValidateConfig = func(_ *VirtualClusterConfig) error {
	return nil
}
