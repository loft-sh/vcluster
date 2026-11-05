{{- define "vcluster.exportKubeConfig.validate" }}
{{- include "vcluster.exportKubeConfig.serviceAccount.validate" (dict "path" "exportKubeConfig.serviceAccount" "serviceAccount" .Values.exportKubeConfig.serviceAccount) }}
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
{{- range $index, $additionalSecret := .Values.exportKubeConfig.additionalSecrets }}
{{- include "vcluster.exportKubeConfig.serviceAccount.validate" (dict "path" (printf "exportKubeConfig.additionalSecrets[%d].serviceAccount" $index) "serviceAccount" $additionalSecret.serviceAccount) }}
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

{{- define "vcluster.exportKubeConfig.serviceAccount.validate" }}
{{- $serviceAccount := .serviceAccount | default dict }}
{{- if $serviceAccount.tokenRotation }}
{{- $rotation := $serviceAccount.tokenRotation }}
{{- if empty $rotation.expiration }}
{{- fail (printf "%s.tokenRotation.expiration is required when rotation settings are configured" .path) }}
{{- end }}
{{- if empty $serviceAccount.name }}
{{- fail (printf "%s.tokenRotation requires a named ServiceAccount" .path) }}
{{- end }}
{{- if ne $rotation.renewBeforePercentage nil }}
{{- $percentage := int64 $rotation.renewBeforePercentage }}
{{- if or (lt $percentage (int64 1)) (gt $percentage (int64 99)) }}
{{- fail (printf "%s.tokenRotation.renewBeforePercentage must be between 1 and 99" .path) }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
