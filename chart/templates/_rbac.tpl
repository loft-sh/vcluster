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
    .Values.sync.toHost.volumeSnapshots.enabled
    .Values.controlPlane.advanced.virtualScheduler.enabled
    .Values.sync.fromHost.ingressClasses.enabled
    (eq (toString .Values.sync.fromHost.storageClasses.enabled) "true")
    (eq (toString .Values.sync.fromHost.csiNodes.enabled) "true")
    (eq (toString .Values.sync.fromHost.csiDrivers.enabled) "true")
    (eq (toString .Values.sync.fromHost.csiStorageCapacities.enabled) "true")
    .Values.sync.fromHost.nodes.enabled
    .Values.observability.metrics.proxy.nodes
    .Values.experimental.multiNamespaceMode.enabled -}}
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
