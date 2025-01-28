{{- define "vcluster.controlPlane.image" -}}
{{- $tag := .Chart.Version -}}
{{- if .Values.controlPlane.statefulSet.image.tag -}}
{{- $tag = .Values.controlPlane.statefulSet.image.tag -}}
{{- end -}}
{{- include "vcluster.image" (dict "defaultImageRegistry" .Values.controlPlane.advanced.defaultImageRegistry "tag" $tag "registry" .Values.controlPlane.statefulSet.image.registry "repository" .Values.controlPlane.statefulSet.image.repository) -}}
{{- end -}}

{{- define "vcluster.image" -}}
{{- if .defaultImageRegistry -}}
{{ .defaultImageRegistry }}/{{ .repository }}:{{ .tag }}
{{- else if .registry -}}
{{ .registry }}/{{ .repository }}:{{ .tag }}
{{- else -}}
{{ .repository }}:{{ .tag }}
{{- end -}}
{{- end -}}

{{- define "extractNamespacesFromHostMappings" -}}
{{- $root := index . 0 -}}
{{- $mappings := index . 1 -}}
{{- $namespaces := list -}}
{{- range $key, $val := $mappings -}}
  {{- $sourceNs := splitList "/" $key | first -}}
  {{- if eq $sourceNs "*" -}}
    {{- $namespaces = append $namespaces $root.Release.Namespace -}}
  {{- else -}}
    {{- $namespaces = append $namespaces $sourceNs -}}
  {{- end -}}
{{- end -}}
{{- $nsList := $namespaces | uniq | sortAlpha -}}
{{- range $namespace := $nsList }}
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: vc-{{ $root.Release.Name }}-from-host
  namespace: {{ $namespace }}
rules:
  - apiGroups: [""]
    resources: ["configmaps"]
    verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: vc-{{ $root.Release.Name }}-from-host
  namespace: {{ $namespace }}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: vc-{{ $root.Release.Name }}-from-host
subjects:
  - kind: ServiceAccount
    {{- if $root.Values.controlPlane.advanced.serviceAccount.name }}
    name: {{ $root.Values.controlPlane.advanced.serviceAccount.name | quote }}
    {{- else }}
    name: vc-{{ $root.Release.Name }}
    {{- end }}
    namespace: {{ $root.Release.Namespace }}
---
{{- end }}
{{- end }}

