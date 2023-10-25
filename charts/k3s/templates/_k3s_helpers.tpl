{{/* vim: set filetype=mustache: */}}
{{/*
Returns the desired workload kind (StatefulSet / Deployment) for k3s
*/}}
{{- define "vcluster.k3s.workloadKind" -}}
{{- ternary "Deployment" "StatefulSet" (.Values.enableHA) -}}
{{- end -}}

{{/*
Returns if we want a persistent volume claim for k3s
*/}}
{{- define "vcluster.k3s.persistence" -}}
{{- if and
    (.Values.storage.persistence)
    (or
        (not .Values.etcd.enabled)
        (and .Values.etcd.enabled .Values.etcd.migrate)) -}}
    {{- true -}}
{{- end -}}
{{- end -}}
