{{- define "vcluster.vClusterConfigHash" -}}
{{- $vals := deepCopy .Values -}}
{{- (unset $vals.privateNodes "autoNodes") | toYaml | b64enc | sha256sum | quote -}}
{{- end -}}