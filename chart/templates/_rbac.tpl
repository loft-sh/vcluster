{{- define "vcluster.clusterRoleName" -}}
{{- printf "vc-%s-v-%s" .Release.Name .Release.Namespace | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "vcluster.clusterRoleNameMultinamespace" -}}
{{- printf "vc-mn-%s-v-%s" .Release.Name .Release.Namespace | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
  Whether to create a cluster role or not
*/}}
{{- define "vcluster.createClusterRole" -}}
{{- if eq (toString .Values.rbac.clusterRole.enabled) "true" -}}
{{- true -}}
{{- else if eq (toString .Values.rbac.clusterRole.enabled) "auto" -}}
{{- if or
    .Values.rbac.clusterRole.overwriteRules
    (not (empty (include "vcluster.rbac.clusterRoleExtraRules" . )))
    (not (empty (include "vcluster.plugin.clusterRoleExtraRules" . )))
    (not (empty (include "vcluster.generic.clusterRoleExtraRules" . )))
    .Values.networking.replicateServices.fromHost
    .Values.pro
    .Values.sync.toHost.storageClasses.enabled
    .Values.experimental.isolatedControlPlane.enabled
    .Values.sync.toHost.persistentVolumes.enabled
    .Values.sync.toHost.priorityClasses.enabled
    .Values.sync.fromHost.priorityClasses.enabled
    .Values.sync.toHost.volumeSnapshotContents.enabled
    .Values.sync.fromHost.volumeSnapshotClasses.enabled
    .Values.controlPlane.advanced.virtualScheduler.enabled
    .Values.sync.fromHost.ingressClasses.enabled
    .Values.sync.fromHost.runtimeClasses.enabled
    (eq (toString .Values.sync.fromHost.storageClasses.enabled) "true")
    (eq (toString .Values.sync.fromHost.csiNodes.enabled) "true")
    (eq (toString .Values.sync.fromHost.csiDrivers.enabled) "true")
    (eq (toString .Values.sync.fromHost.csiStorageCapacities.enabled) "true")
    .Values.sync.fromHost.nodes.enabled
    .Values.sync.toHost.customResources
    .Values.sync.fromHost.customResources
    .Values.integrations.kubeVirt.enabled
    .Values.integrations.externalSecrets.enabled
    (and .Values.integrations.certManager.enabled .Values.integrations.certManager.sync.fromHost.clusterIssuers.enabled)
    (and .Values.integrations.metricsServer.enabled .Values.integrations.metricsServer.nodes)
    .Values.experimental.multiNamespaceMode.enabled
	.Values.sync.fromHost.configMaps.enabled
    .Values.sync.fromHost.secrets.enabled -}}
{{- true -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
  Role rules defined on global level
*/}}
{{- define "vcluster.rbac.roleExtraRules" -}}
{{- if .Values.rbac.role.extraRules }}
{{- range $ruleIndex, $rule := .Values.rbac.role.extraRules }}
- {{ toJson $rule }}
{{- end }}
{{- end }}
{{- end -}}

{{/*
  Role rules defined by plugins
*/}}
{{- define "vcluster.plugin.roleExtraRules" -}}
{{- range $key, $container := .Values.plugin }}
{{- if $container.rbac }}
{{- if $container.rbac.role }}
{{- if $container.rbac.role.extraRules }}
{{- range $ruleIndex, $rule := $container.rbac.role.extraRules }}
- {{ toJson $rule }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- range $key, $container := .Values.plugins }}
{{- if $container.rbac }}
{{- if $container.rbac.role }}
{{- if $container.rbac.role.extraRules }}
{{- range $ruleIndex, $rule := $container.rbac.role.extraRules }}
- {{ toJson $rule }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- end -}}

{{/*
  Cluster role rules defined by plugins
*/}}
{{- define "vcluster.plugin.clusterRoleExtraRules" -}}
{{- range $key, $container := .Values.plugin }}
{{- if $container.rbac }}
{{- if $container.rbac.clusterRole }}
{{- if $container.rbac.clusterRole.extraRules }}
{{- range $ruleIndex, $rule := $container.rbac.clusterRole.extraRules }}
- {{ toJson $rule }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- range $key, $container := .Values.plugins }}
{{- if $container.rbac }}
{{- if $container.rbac.clusterRole }}
{{- if $container.rbac.clusterRole.extraRules }}
{{- range $ruleIndex, $rule := $container.rbac.clusterRole.extraRules }}
- {{ toJson $rule }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- end -}}

{{/*
  Role rules defined in generic syncer
*/}}
{{- define "vcluster.generic.roleExtraRules" -}}
{{- if .Values.experimental.genericSync.role }}
{{- if .Values.experimental.genericSync.role.extraRules }}
{{- range $ruleIndex, $rule := .Values.experimental.genericSync.role.extraRules }}
- {{ toJson $rule }}
{{- end }}
{{- end }}
{{- end }}
{{- end -}}

{{/*
  Role rules defined in generic syncer
*/}}
{{- define "vcluster.customResources.roleExtraRules" -}}
{{- if .Values.sync.toHost.customResources }}
{{- range $crdName, $rule := .Values.sync.toHost.customResources }}
{{- if $rule.enabled }}
- resources: [ "{{ (splitn "." 2 $crdName)._0 }}" ]
  apiGroups: [ "{{ (splitn "." 2 $crdName)._1 }}" ]
  verbs: ["create", "delete", "patch", "update", "get", "list", "watch"]
{{- end }}
{{- end }}
{{- end }}
{{- end -}}

{{/*
  Cluster role rules defined in generic syncer
*/}}
{{- define "vcluster.customResources.clusterRoleExtraRules" -}}
{{- if .Values.sync.fromHost.customResources }}
{{- range $crdName, $rule := .Values.sync.fromHost.customResources }}
{{- if $rule.enabled }}
- resources: [ "{{ (splitn "." 2 $crdName)._0 }}" ]
  apiGroups: [ "{{ (splitn "." 2 $crdName)._1 }}" ]
  verbs: ["get", "list", "watch"]
{{- end }}
{{- end }}
{{- end }}
{{- end -}}

{{/*
  Cluster role rules defined in generic syncer
*/}}
{{- define "vcluster.generic.clusterRoleExtraRules" -}}
{{- if .Values.experimental.genericSync.clusterRole }}
{{- if .Values.experimental.genericSync.clusterRole.extraRules }}
{{- range $ruleIndex, $rule := .Values.experimental.genericSync.clusterRole.extraRules }}
- {{ toJson $rule }}
{{- end }}
{{- end }}
{{- end }}
{{- end -}}

{{/*
  Cluster Role rules defined on global level
*/}}
{{- define "vcluster.rbac.clusterRoleExtraRules" -}}
{{- if .Values.rbac.clusterRole.extraRules }}
{{- range $ruleIndex, $rule := .Values.rbac.clusterRole.extraRules }}
- {{ toJson $rule }}
{{- end }}
{{- end }}
{{- end -}}

{{/*
  Whether to create a role and role binding to access the platform API key secret
*/}}
{{- define "vcluster.rbac.createPlatformSecretRole" -}}
{{- $createRBAC := dig "platform" "apiKey" "createRBAC" true .Values.external -}}
{{- if and $createRBAC (ne (include "vcluster.rbac.platformSecretNamespace" .) .Release.Namespace) }}
{{- true -}}
{{- end }}
{{- end -}}

{{/*
  Namespace containing the vCluster platform secret
*/}}
{{- define "vcluster.rbac.platformSecretNamespace" -}}
{{- dig "platform" "apiKey" "namespace" .Release.Namespace .Values.external | default .Release.Namespace -}}
{{- end -}}

{{/*
  Name specifies the secret name containing the vCluster platform licenses and tokens
*/}}
{{- define "vcluster.rbac.platformSecretName" -}}
{{- dig "platform" "apiKey" "secretName" "vcluster-platform-api-key" .Values.external | quote -}}
{{- end -}}

{{- define "vcluster.rbac.platformRoleName" -}}
{{- printf "vc-%s-v-%s-platform-role" .Release.Name .Release.Namespace | trunc 63 | trimSuffix "-" -}}
{{- end -}}


{{- define "vcluster.rbac.platformRoleBindingName" -}}
{{- printf "vc-%s-v-%s-platform-role-binding" .Release.Name .Release.Namespace | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
  Cluster role rules needed for fromHost sync (containing namespaces + configmaps/secret/other core resources)
*/}}
{{- define "vcluster.rbac.rulesForFromHostSyncerForGivenCoreResource" -}}
{{- $root := index . 0 -}}
{{- $mappings := index . 1 -}}
{{- $kind := index . 2 -}}
{{- $enabled := index . 3 -}}
{{- if and $enabled $mappings -}}
{{- $namespaces := list -}}
{{- $objNames := list -}}
{{- $addResourceNames := true -}}
{{- range $key, $val := $mappings -}}
  {{- $sourceNs := splitList "/" $key | first -}}
  {{- $sourceObjName := splitList "/" $key | last }}
  {{- if eq $sourceNs "" -}}
    {{- $namespaces = append $namespaces (quote $root.Release.Namespace) -}}
  {{- else -}}
    {{- $namespaces = append $namespaces (quote $sourceNs) -}}
  {{- end -}}
  {{- if eq $sourceObjName "*" -}}
  	{{- $addResourceNames = false -}}
  {{- else -}}
  	{{- $objNames = append $objNames (quote $sourceObjName) -}}
  {{- end -}}
{{- end -}}
{{- $objList := $objNames | uniq | sortAlpha -}}
{{- $nsList := $namespaces | uniq | sortAlpha -}}
- apiGroups: [""]
  resources: [ "namespaces" ]
  resourceNames: [ {{ join "," $nsList }} ]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: [ {{ $kind | quote }} ]
  verbs: ["list", "watch"]
- apiGroups: [""]
  resources: [ {{ $kind | quote }} ]
  verbs: ["get"]
{{- if $addResourceNames }}
  resourceNames: [ {{ join "," $objList }} ]
{{- end }}
{{- end }}
{{- end }}

