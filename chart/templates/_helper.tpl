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
