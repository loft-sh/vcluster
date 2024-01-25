{{/*
  handles both replicas and syncer.replicas
*/}}
{{- define "vcluster.replicas" -}}
{{ if .Values.syncer.replicas }}{{ .Values.syncer.replicas }}{{ else }}{{ .Values.replicas }}{{ end }}
{{- end }}
