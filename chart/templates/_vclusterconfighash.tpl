{{- define "vcluster.vClusterConfigHash" -}}
{{- $vals := deepCopy .Values -}}
{{- $_ := unset (index $vals "privateNodes") "autoNodes" -}}
{{- toYaml $vals | b64enc | sha256sum | quote -}}
{{- end -}}