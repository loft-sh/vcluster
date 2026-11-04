{{- define "vcluster.exportKubeConfig.validate" }}
{{- /*
  Verify that exportKubeConfig.secret and exportKubeConfig.additionalSecrets are
  not set at the same time.
*/}}
{{- $secretSet := false }}
{{- if .Values.exportKubeConfig.secret }}
{{- $secretSet = or (.Values.exportKubeConfig.secret.name | trim | ne "") (.Values.exportKubeConfig.secret.namespace | trim | ne "") }}
{{- end }}
{{- $additionalSecretsSet := false }}
{{- if .Values.exportKubeConfig.additionalSecrets }}
{{- $additionalSecretsSet = gt (len .Values.exportKubeConfig.additionalSecrets) 0 }}
{{- end }}
{{- if and $secretSet $additionalSecretsSet }}
{{- fail "exportKubeConfig.secret and exportKubeConfig.additionalSecrets cannot be set at the same time" }}
{{- end }}
{{- /*
  Verify that additional secrets have name or namespace set.
*/}}
{{- range $_, $additionalSecret := .Values.exportKubeConfig.additionalSecrets }}
{{- $nameSet := false }}
{{- $namespaceSet := false }}
{{- if $additionalSecret.name }}
{{- if $additionalSecret.name | trim | ne "" }}
{{- $nameSet = true }}
{{- end }}
{{- end }}
{{- if $additionalSecret.namespace }}
{{- if $additionalSecret.namespace | trim | ne "" }}
{{- $namespaceSet = true }}
{{- end }}
{{- end }}
{{- if not (or $nameSet $namespaceSet) }}
{{- fail (cat "additional secret must have name and/or namespace set, found:" (toJson $additionalSecret)) }}
{{- end }}
{{- end }}
{{- end }}
