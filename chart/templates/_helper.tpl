{{- define "vcluster.controlPlane.image" -}}
{{- if .Values.controlPlane.statefulSet.image.tag -}}
{{ .Values.controlPlane.advanced.defaultImageRegistry }}{{ .Values.controlPlane.statefulSet.image.repository }}:{{ .Values.controlPlane.statefulSet.image.tag }}
{{- else -}}
{{ .Values.controlPlane.advanced.defaultImageRegistry }}{{ .Values.controlPlane.statefulSet.image.repository }}:{{ .Chart.Version }}
{{- end -}}
{{- end -}}
