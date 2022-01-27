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
{{- $defaultEnabled := list "services" "configmaps" "secrets" "endpoints" "pods" "events" "persistentvolumeclaims" "ingresses" "fake-nodes" "fake-persistentvolumes" -}}
{{- range $key, $val := .Values.sync }}
{{- if and (has $key $defaultEnabled) (not $val.enabled) }}
- --sync=-{{ $key }}
{{- else if and (not (has $key $defaultEnabled)) ($val.enabled)}}
- --sync={{ $key }}
{{- end -}}
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