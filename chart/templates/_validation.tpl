{{/*
This template has no output, only side-effects.
*/}}
{{ define "validate-config" -}}
  {{ $ctx := . -}}
  {{- /* Strip any custom type info from values. */}}
  {{- /* Functions like dig require a plain map. */}}
  {{ $current := (fromYaml ($ctx.Values | toYaml)) -}}
  {{ $release := $ctx.Release -}}

  {{ $existing := dict -}}
  {{ $secretName := printf "vc-config-%s" $release.Name -}}
  {{ $secret := lookup "v1" "Secret" $release.Namespace $secretName -}}
  {{ with dig "data" "config.yaml" (b64enc "{}") $secret -}}
    {{ $decoded := b64dec . -}}
    {{ $existing = (fromYaml $decoded) -}}
  {{ end -}}

  {{ include "validate-private-nodes" (dict "existing" $existing "new" $current) }}

  {{- /*
     Enforce rules for changes that would be destructive
  */}}
{{/* sync.fromHost.nodes.enabled cannot change */}}
  {{ $oldFromHostNodes := (dig "sync" "fromHost" "nodes" "enabled" false $existing) -}}
  {{ $newFromHostNodes := (dig "sync" "fromHost" "nodes" "enabled" false $current) -}}
  {{ if (ne $oldFromHostNodes $newFromHostNodes) -}}
    {{ fail (printf "sync.fromHost.nodes.enabled cannot be changed (existing: %t, new: %t)" $oldFromHostNodes $newFromHostNodes) }}
  {{ end -}}

  {{/* sync.toHost.persistentVolumes.enabled cannot change */}}
  {{ $oldToHostPVs := (dig "sync" "toHost" "persistentVolumes" "enabled" false $existing) -}}
  {{ $newToHostPVs := (dig "sync" "toHost" "persistentVolumes" "enabled" false $current) -}}
  {{ if (ne $oldToHostPVs $newToHostPVs) -}}
    {{ fail (printf "sync.toHost.persistentVolumes.enabled cannot be changed (existing: %t, new: %t)" $oldToHostPVs $newToHostPVs) }}
  {{ end -}}

  {{/* sync.toHost.namespaces.enabled cannot change */}}
  {{ $oldToHostNamespaces := (dig "sync" "toHost" "namespaces" "enabled" false $existing) -}}
  {{ $newToHostNamespaces := (dig "sync" "toHost" "namespaces" "enabled" false $current) -}}
  {{ if (ne $oldToHostNamespaces $newToHostNamespaces) -}}
    {{ fail (printf "sync.toHost.namespaces.enabled cannot be changed (existing: %t, new: %t)" $oldToHostNamespaces $newToHostNamespaces) }}
  {{ end -}}

  {{/* networking.advanced.clusterDomain cannot change */}}
  {{ $oldClusterDomain := (dig "networking" "advanced" "clusterDomain" "cluster.local" $existing) -}}
  {{ $newClusterDomain := (dig "networking" "advanced" "clusterDomain" "cluster.local" $current) -}}
  {{ if (ne $oldClusterDomain $newClusterDomain) -}}
    {{ fail (printf "networking.advanced.clusterDomain cannot be changed (existing: %s, new: %s)" $oldClusterDomain $newClusterDomain) }}
  {{ end -}}

  {{/* networking.podCIDR cannot change */}}
  {{ $oldPrivateNodes := (dig "privateNodes" "enabled" false $existing) -}}
  {{ $oldPodCIDR := (dig "networking" "podCIDR" "10.244.0.0/16" $existing) -}}
  {{ $newPodCIDR := (dig "networking" "podCIDR" "10.244.0.0/16" $current) -}}
  {{ if and $oldPrivateNodes (ne $oldPodCIDR $newPodCIDR) -}}
    {{ fail (printf "networking.podCIDR cannot be changed (existing: %s, new: %s)" $oldPodCIDR $newPodCIDR) }}
  {{ end -}}

  {{/* networking.serviceCIDR cannot change */}}
  {{ $oldPrivateNodes := (dig "privateNodes" "enabled" false $existing) -}}
  {{ $oldPodCIDR := (dig "networking" "serviceCIDR" "10.96.0.0/12" $existing) -}}
  {{ $newPodCIDR := (dig "networking" "serviceCIDR" "10.96.0.0/12" $current) -}}
  {{ if and $oldPrivateNodes (ne $oldPodCIDR $newPodCIDR) -}}
    {{ fail (printf "networking.serviceCIDR cannot be changed (existing: %s, new: %s)" $oldPodCIDR $newPodCIDR) }}
  {{ end -}}

  {{/* controlPlane.proxy.port cannot change */}}
  {{ $oldProxyPort := (dig "controlPlane" "proxy" "port" 8443 $existing) -}}
  {{ $newProxyPort := (dig "controlPlane" "proxy" "port" 8443 $current) -}}
  {{ if (ne (int $oldProxyPort) (int $newProxyPort)) -}}
    {{ fail (printf "controlPlane.proxy.port cannot be changed (existing: %v, new: %v)" $oldProxyPort $newProxyPort) }}
  {{ end -}}

  {{/* controlPlane.distro: only allow k3s -> k8s, forbid other changes */}}
  {{ $oldK8s := (dig "controlPlane" "distro" "k8s" "enabled" true $existing) -}}
  {{ $oldK3s := (dig "controlPlane" "distro" "k3s" "enabled" false $existing) -}}
  {{ $newK8s := (dig "controlPlane" "distro" "k8s" "enabled" true $current) -}}
  {{ $newK3s := (dig "controlPlane" "distro" "k3s" "enabled" false $current) -}}
  {{ if and $newK3s $newK8s -}}
  	{{ fail "cannot enable both k8s and k3s control plane at once" }}
  {{ end -}}
  {{ if and $oldK8s $newK3s -}}
  	{{ fail "cannot change control plane from k3s to k8s" }}
  {{ end -}}
{{ end -}}
