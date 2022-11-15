{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "vcluster.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "vcluster.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Whether the ingressclasses syncer should be enabled
*/}}
{{- define "vcluster.syncIngressclassesEnabled" -}}
{{- if or (.Values.sync.ingressclasses).enabled (and .Values.sync.ingresses.enabled (not (hasKey .Values.sync.ingressclasses "enabled"))) -}}
    {{- true -}}
{{- end -}}
{{- end -}}

{{/*
Whether to create a cluster role or not
*/}}
{{- define "vcluster.createClusterRole" -}}
{{- if or (not (empty (include "vcluster.serviceMapping.fromHost" . ))) (not (empty (include "vcluster.plugin.clusterRoleExtraRules" . ))) .Values.rbac.clusterRole.create .Values.sync.hoststorageclasses.enabled (index ((index .Values.sync "legacy-storageclasses") | default (dict "enabled" false)) "enabled") (include "vcluster.syncIngressclassesEnabled" . ) .Values.sync.ingresses.enabled .Values.sync.nodes.enabled .Values.sync.persistentvolumes.enabled .Values.sync.storageclasses.enabled .Values.sync.priorityclasses.enabled .Values.sync.volumesnapshots.enabled -}}
    {{- true -}}
{{- end -}}
{{- end -}}

{{- define "vcluster.clusterRoleName" -}}
{{- printf "vc-%s-v-%s" .Release.Name .Release.Namespace | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "vcluster.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "vcluster.labels" -}}
app.kubernetes.io/name: {{ include "vcluster.name" . }}
helm.sh/chart: {{ include "vcluster.chart" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- else }}
app.kubernetes.io/version: {{ .Chart.Version | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Get
*/}}
{{- $}}
{{- define "vcluster.admin.accessKey" -}}
{{- now | unixEpoch | toString | trunc 8 | sha256sum -}}
{{- end -}}

{{/*
Syncer flags for enabling/disabling controllers
Prints only the flags that modify the defaults:
- when default controller has enabled: false => `- "--sync=-controller`
- when non-default controller has enabled: true => `- "--sync=controller`
*/}}
{{- define "vcluster.syncer.syncArgs" -}}
{{- $defaultEnabled := list "services" "configmaps" "secrets" "endpoints" "pods" "events" "persistentvolumeclaims" "fake-nodes" "fake-persistentvolumes" -}}
{{- if and (hasKey .Values.sync.nodes "enableScheduler") .Values.sync.nodes.enableScheduler -}}
    {{- $defaultEnabled = concat $defaultEnabled (list "csinodes" "csidrivers" "csistoragecapacities" ) -}}
{{- end -}}
{{- range $key, $val := .Values.sync }}
{{- if and (has $key $defaultEnabled) (not $val.enabled) }}
- --sync=-{{ $key }}
{{- else if and (not (has $key $defaultEnabled)) ($val.enabled)}}
{{- if eq $key "legacy-storageclasses" }}
- --sync=hoststorageclasses
{{- else }}
- --sync={{ $key }}
{{- end -}}
{{- end -}}
{{- end -}}
{{- if not (include "vcluster.syncIngressclassesEnabled" . ) }}
- --sync=-ingressclasses
{{- end -}}
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
{{- end -}}


{{/*
Virtual cluster service mapping
*/}}
{{- define "vcluster.serviceMapping.fromVirtual" -}}
{{- range $key, $value := .Values.mapServices.fromVirtual }}
- '--map-virtual-service={{ $value.from }}={{ $value.to }}'
{{- end }}
{{- end -}}

{{/*
Host cluster service mapping
*/}}
{{- define "vcluster.serviceMapping.fromHost" -}}
{{- range $key, $value := .Values.mapServices.fromHost }}
- '--map-host-service={{ $value.from }}={{ $value.to }}'
{{- end }}
{{- end -}}
