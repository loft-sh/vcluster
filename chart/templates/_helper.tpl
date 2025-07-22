{{- define "vcluster.controlPlane.image" -}}
{{- $tag := .Chart.Version -}}
{{- if .Values.controlPlane.statefulSet.image.tag -}}
{{- $tag = .Values.controlPlane.statefulSet.image.tag -}}
{{- end -}}
{{- include "vcluster.image" (dict "defaultImageRegistry" .Values.controlPlane.advanced.defaultImageRegistry "tag" $tag "registry" .Values.controlPlane.statefulSet.image.registry "repository" .Values.controlPlane.statefulSet.image.repository) -}}
{{- end -}}

{{- define "vcluster.image" -}}
{{- if .defaultImageRegistry -}}
{{ .defaultImageRegistry }}/{{ .repository }}:{{ .tag }}
{{- else if .registry -}}
{{ .registry }}/{{ .repository }}:{{ .tag }}
{{- else -}}
{{ .repository }}:{{ .tag }}
{{- end -}}
{{- end -}}
{{/*
CAST AI patches for pod labels
*/}}
{{- define "vcluster.castai.patches" -}}
{{- if .Values.castai.enabled -}}
- path: metadata.labels["workloads.cast.ai/custom-workload"]
  expression: {{ .Values.castai.workloadName | quote }}
- path: metadata.labels["reports.cast.ai/name"]
  expression: {{ .Values.castai.workloadName | quote }}
{{- end -}}
{{- end -}}
{{/*

Generate vCluster configuration with conditional CAST AI patches
*/}}
{{- define "vcluster.config" -}}
{{- $config := deepCopy .Values -}}
{{- if .Values.castai.enabled -}}
{{- $castaiPatches := list -}}
{{- $castaiPatches = append $castaiPatches (dict "path" "metadata.labels[\"workloads.cast.ai/custom-workload\"]" "expression" .Values.castai.workloadName) -}}
{{- $castaiPatches = append $castaiPatches (dict "path" "metadata.labels[\"reports.cast.ai/name\"]" "expression" .Values.castai.workloadName) -}}
{{- $existingPatches := default (list) $config.sync.toHost.pods.patches -}}
{{- $allPatches := concat $existingPatches $castaiPatches -}}
{{- $_ := set $config.sync.toHost.pods "patches" $allPatches -}}
{{- end -}}
{{- $config | toYaml -}}
{{- end -}}