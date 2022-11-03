{{/* vim: set filetype=mustache: */}}
{{/*
Returns the desired workload kind (StatefulSet / Deployment) for k3s
*/}}
{{- define "vcluster.k3s.workloadKind" -}}
{{- ternary "Deployment" "StatefulSet" (.Values.enableHA) -}}
{{- end -}}

{{/*
Returns the existing value of the k3s server token stored in the Kubernetes secret.
If the Kubernetes secret does not exist, returns a generated, random server token.
*/}}
{{- define "vcluster.k3s.serverToken" -}}
{{- $secret := (lookup "v1" "Secret" .Release.Namespace .Release.Name ) -}}
  {{- if $secret -}}
    {{-  index $secret "data" "server-token" -}}
  {{- else -}}
    {{- (randAlphaNum 32) | b64enc | quote -}}
  {{- end -}}
{{- end -}}
