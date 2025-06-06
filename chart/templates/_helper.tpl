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
Returns the probe configuration merged with defaults.
Usage:
{{ include "vcluster.probe" (dict "ctx" . "type" "liveness") }}
*/}}
{{- define "vcluster.probe" -}}
{{- $ctx := .ctx -}}
{{- $type := .type -}}

{{- $user := dict -}}
{{- if hasKey $ctx.Values.controlPlane.statefulSet.probes $type }}
  {{- $user = index $ctx.Values.controlPlane.statefulSet.probes $type | default dict -}}
{{- end }}

{{- $defaults := dict -}}
{{- if eq $type "livenessProbe" }}
  {{- $_ := set $defaults "enabled" false -}}
  {{- $_ := set $defaults "initialDelaySeconds" 60 -}}
  {{- $_ := set $defaults "periodSeconds" 2 -}}
  {{- $_ := set $defaults "timeoutSeconds" 3 -}}
  {{- $_ := set $defaults "failureThreshold" 60 -}}
{{- else if eq $type "readinessProbe" }}
  {{- $_ := set $defaults "enabled" false -}}
  {{- $_ := set $defaults "periodSeconds" 2 -}}
  {{- $_ := set $defaults "timeoutSeconds" 3 -}}
  {{- $_ := set $defaults "failureThreshold" 60 -}}
{{- else if eq $type "startupProbe" }}
  {{- $_ := set $defaults "enabled" false -}}
  {{- $_ := set $defaults "periodSeconds" 6 -}}
  {{- $_ := set $defaults "timeoutSeconds" 3 -}}
  {{- $_ := set $defaults "failureThreshold" 300 -}}
{{- end }}

{{- $merged := mergeOverwrite (deepCopy $defaults) $user -}}
{{- toYaml $merged -}}
{{- end }}
