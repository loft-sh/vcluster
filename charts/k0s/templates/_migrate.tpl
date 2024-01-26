{{/*
  handles both replicas and syncer.replicas
*/}}
{{- define "vcluster.replicas" -}}
{{ if .Values.replicas }}{{ .Values.replicas }}{{ else }}{{ .Values.syncer.replicas }}{{ end }}
{{- end }}
