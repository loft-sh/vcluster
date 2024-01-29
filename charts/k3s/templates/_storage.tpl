{{/*
  storage size
*/}}
{{- define "vcluster.storage.size" -}}
{{if .Values.storage }}{{ .Values.storage.size }}{{ else }}{{ .Values.syncer.storage.size }}{{ end }}
{{- end -}}

{{/*
  storage persistence
*/}}
{{- define "vcluster.storage.persistence" -}}
{{if .Values.storage }}{{ .Values.storage.persistence }}{{ else }}{{ .Values.syncer.storage.persistence }}{{ end }}
{{- end -}}

{{/*
  storage classname
*/}}
{{- define "vcluster.storage.className" -}}
{{if .Values.storage }}{{ .Values.storage.className }}{{ else }}{{ .Values.syncer.storage.className }}{{ end }}
{{- end -}}
