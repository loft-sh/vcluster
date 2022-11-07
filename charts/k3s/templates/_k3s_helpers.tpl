{{/* vim: set filetype=mustache: */}}
{{/*
Returns the desired workload kind (StatefulSet / Deployment) for k3s
*/}}
{{- define "vcluster.k3s.workloadKind" -}}
{{- ternary "Deployment" "StatefulSet" (.Values.enableHA) -}}
{{- end -}}

{{/*
Returns the name of the secret containing the k3s tokens.
*/}}
{{- define "vcluster.k3s.tokenSecretName" -}}
{{- with .Values.serverToken.secretKeyRef.name -}}
{{- . -}}
{{- else -}}
{{- printf "%s-tokens" .Release.Name -}}
{{- end -}}
{{- end -}}

{{/*
Returns the secret key name containing the k3s server token.
*/}}
{{- define "vcluster.k3s.serverTokenKey" -}}
{{- with .Values.serverToken.secretKeyRef.key -}}
{{- . -}}
{{- else -}}
{{- "server-token" -}}
{{- end -}}
{{- end -}}
