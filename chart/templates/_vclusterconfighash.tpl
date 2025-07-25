{{- define "vcluster.vClusterConfigHash" -}}
{{- $vals := deepCopy .Values -}}
{{- (unset $vals.privateNodes "nodePools") | toYaml | b64enc | sha256sum | quote -}}
{{- end -}}