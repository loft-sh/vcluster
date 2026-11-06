{{/*
This template has no output, only side-effects.
*/}}
{{ define "validate-private-nodes" -}}
  {{ $existing := .existing -}}
  {{ $new := .new -}}

  {{ $oldPrivateNodes := (dig "privateNodes" "enabled" false $existing) -}}
  {{ $newPrivateNodes := (dig "privateNodes" "enabled" false $new) -}}
  {{ if ne $oldPrivateNodes $newPrivateNodes -}}
    {{ fail (printf "privateNodes.enabled cannot be changed (existing: %t, new: %t)" $oldPrivateNodes $newPrivateNodes) }}
  {{ end -}}
{{ end -}}
