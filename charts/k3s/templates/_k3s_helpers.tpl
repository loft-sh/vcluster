{{/* vim: set filetype=mustache: */}}
{{/*
Returns the desired workload kind (StatefulSet / Deployment) for k3s
*/}}
{{- define "vcluster.k3s.workloadKind" -}}
{{- ternary "Deployment" "StatefulSet" (.Values.enableHA) -}}
{{- end -}}

