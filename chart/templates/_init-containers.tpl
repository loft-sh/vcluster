{{/* Bump $defaultTag value whenever k8s version is bumped */}}
{{- define "vcluster.k8s.image.tag" -}}
{{- if not (empty .Values.controlPlane.distro.k8s.version) -}}
{{- if .Values.privateNodes.enabled -}}
{{ .Values.controlPlane.distro.k8s.version }}-full
{{- else -}}
{{ .Values.controlPlane.distro.k8s.version }}
{{- end -}}
{{- else -}}
{{- if .Values.privateNodes.enabled -}}
{{ .Values.controlPlane.distro.k8s.image.tag }}-full
{{- else -}}
{{ .Values.controlPlane.distro.k8s.image.tag }}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "vcluster.k8s.initContainers" -}}
{{- include "vcluster.oldPlugins.initContainers" . }}
{{- include "vcluster.plugins.initContainers" . }}
- name: kubernetes
  image: "{{ include "vcluster.image" (dict "defaultImageRegistry" .Values.controlPlane.advanced.defaultImageRegistry "registry" .Values.controlPlane.distro.k8s.image.registry "repository" .Values.controlPlane.distro.k8s.image.repository "tag" (include "vcluster.k8s.image.tag" .)) }}"
  volumeMounts:
    - mountPath: /binaries
      name: binaries
  command:
    - cp
  args:
    - -r 
    - /kubernetes/.
    - /binaries/
  {{- if .Values.controlPlane.distro.k8s.imagePullPolicy }}
  imagePullPolicy: {{ .Values.controlPlane.distro.k8s.imagePullPolicy }}
  {{- end }}
  securityContext:
{{ toYaml .Values.controlPlane.distro.k8s.securityContext | indent 4 }}
  resources:
{{ toYaml .Values.controlPlane.distro.k8s.resources | indent 4 }}
{{- if .Values.controlPlane.statefulSet.initContainers }}
{{ toYaml .Values.controlPlane.statefulSet.initContainers }}
{{- end }}
{{- end -}}

{{/*
  Plugin init container definition
*/}}
{{- define "vcluster.plugins.initContainers" -}}
{{- range $key, $container := .Values.plugins }}
{{- if not $container.image }}
{{- continue }}
{{- end }}
- {{- if $.Values.controlPlane.advanced.defaultImageRegistry }}
  image: {{ $.Values.controlPlane.advanced.defaultImageRegistry }}/{{ $container.image }}
  {{- else }}
  image: {{ $container.image }}
  {{- end }}
  {{- if $container.name }}
  name: {{ $container.name | quote }}
  {{- else }}
  name: {{ $key | quote }}
  {{- end }}
  {{- if $container.imagePullPolicy }}
  imagePullPolicy: {{ $container.imagePullPolicy }}
  {{- end }}
  {{- if or $container.command $container.args }}
  {{- if $container.command }}
  command:
    {{- range $commandIndex, $command := $container.command }}
    - {{ $command | quote }}
    {{- end }}
  {{- end }}
  {{- if $container.args }}
  args:
    {{- range $argIndex, $arg := $container.args }}
    - {{ $arg | quote }}
    {{- end }}
  {{- end }}
  {{- else }}
  command: ["sh"]
  args: ["-c", "cp -r /plugin /plugins/{{ $key }}"]
  {{- end }}
  {{- if $container.securityContext }}
  securityContext:
{{ toYaml $container.securityContext | indent 4 }}
  {{- end }}
  {{- if $container.volumeMounts }}
  volumeMounts:
{{ toYaml $container.volumeMounts | indent 4 }}
  {{- else }}
  volumeMounts:
    - mountPath: /plugins
      name: plugins
  {{- end }}
  {{- if $container.resources }}
  resources:
{{ toYaml $container.resources | indent 4 }}
  {{- end }}
{{- end }}
{{- end -}}

{{/*
  Old Plugin init container definition
*/}}
{{- define "vcluster.oldPlugins.initContainers" -}}
{{- range $key, $container := .Values.plugin }}
{{- if or (ne $container.version "v2") (not $container.image) -}}
{{- continue -}}
{{- end -}}
- {{- if $.Values.controlPlane.advanced.defaultImageRegistry }}
  image: {{ $.Values.controlPlane.advanced.defaultImageRegistry }}/{{ $container.image }}
  {{- else }}
  image: {{ $container.image }}
  {{- end }}
  {{- if $container.name }}
  name: {{ $container.name | quote }}
  {{- else }}
  name: {{ $key | quote }}
  {{- end }}
  {{- if $container.imagePullPolicy }}
  imagePullPolicy: {{ $container.imagePullPolicy }}
  {{- end }}
  {{- if or $container.command $container.args }}
  {{- if $container.command }}
  command:
    {{- range $commandIndex, $command := $container.command }}
    - {{ $command | quote }}
    {{- end }}
  {{- end }}
  {{- if $container.args }}
  args:
    {{- range $argIndex, $arg := $container.args }}
    - {{ $arg | quote }}
    {{- end }}
  {{- end }}
  {{- else }}
  command: ["sh"]
  args: ["-c", "cp -r /plugin /plugins/{{ $key }}"]
  {{- end }}
  securityContext:
{{ toYaml $container.securityContext | indent 4 }}
  {{- if $container.volumeMounts }}
  volumeMounts:
{{ toYaml $container.volumeMounts | indent 4 }}
  {{- else }}
  volumeMounts:
    - mountPath: /plugins
      name: plugins
  {{- end }}
  {{- if $container.resources }}
  resources:
{{ toYaml $container.resources | indent 4 }}
  {{- end }}
{{- end }}
{{- end -}}
