---
title: "vcluster create --help"
sidebar_label: vcluster create
---


Create a new virtual cluster

## Synopsis


```
vcluster create VCLUSTER_NAME [flags]
```

```
#######################################################
################### vcluster create ###################
#######################################################
Creates a new virtual cluster

Example:
vcluster create test --namespace test
#######################################################
```


## Flags

```
      --chart-name string                 The virtual cluster chart name to use (default "vcluster")
      --chart-repo string                 The virtual cluster chart repo to use (default "https://charts.loft.sh")
      --chart-version string              The virtual cluster chart version to use (e.g. v0.9.1)
      --connect                           If true will run vcluster connect directly after the vcluster was created (default true)
      --create-cluster-role               DEPRECATED: cluster role is now automatically created if it is required by one of the resource syncers that are enabled by the .sync.RESOURCE.enabled=true helm value, which is set in a file that is passed via --extra-values argument.
      --create-namespace                  If true the namespace will be created if it does not exist (default true)
      --disable-ingress-sync              If true the virtual cluster will not sync any ingresses
      --distro string                     Kubernetes distro to use for the virtual cluster. Allowed distros: k3s, k0s, k8s, eks (default "k3s")
      --expose                            If true will create a load balancer service to expose the vcluster endpoint
      --expose-local                      If true and a local Kubernetes distro is detected, will deploy vcluster with a NodePort service. Will be set to false and the passed value will be ignored if --expose is set to true. (default true)
  -f, --extra-values strings              Path where to load extra helm values from
  -h, --help                              help for create
      --isolate                           If true vcluster and its workloads will run in an isolated environment
      --k3s-image string                  DEPRECATED: use --extra-values instead
      --kube-config-context-name string   If set, will override the context name of the generated virtual cluster kube config with this name
      --kubernetes-version string         The kubernetes version to use (e.g. v1.20). Patch versions are not supported
      --local-chart-dir string            The virtual cluster local chart dir to use
      --release-values string             DEPRECATED: use --extra-values instead
      --update-current                    If true updates the current kube config (default true)
      --upgrade                           If true will try to upgrade the vcluster instead of failing if it already exists
```


## Global & Inherited Flags

```
      --context string     The kubernetes config context to use
      --debug              Prints the stack trace if an error occurs
  -n, --namespace string   The kubernetes namespace to use
  -s, --silent             Run in silent mode and prevents any vcluster log output except panics & fatals
```

