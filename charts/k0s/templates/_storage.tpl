{{/*
  storage size
*/}}
{{- define "vcluster.storage.size" -}}
{{if and .Values.storage (hasKey .Values.storage "size") }}{{ .Values.storage.size }}{{ else }}{{ .Values.syncer.storage.size }}{{ end }}
{{- end -}}

{{/*
  storage persistence
*/}}
{{- define "vcluster.storage.persistence" -}}
{{if and .Values.storage (hasKey .Values.storage "persistence") }}{{ .Values.storage.persistence }}{{ else }}{{ .Values.syncer.storage.persistence }}{{ end }}
{{- end -}}

{{/*
  storage classname
*/}}
{{- define "vcluster.storage.className" -}}
{{if and .Values.storage (hasKey .Values.storage "className") }}{{ .Values.storage.className }}{{ else }}{{ .Values.syncer.storage.className }}{{ end }}
{{- end -}}
