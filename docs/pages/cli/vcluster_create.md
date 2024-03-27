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
      --annotations strings               [PRO] Comma separated annotations to apply to the virtualclusterinstance
      --chart-name string                 The virtual cluster chart name to use (default "vcluster")
      --chart-repo string                 The virtual cluster chart repo to use (default "https://charts.loft.sh")
      --chart-version string              The virtual cluster chart version to use (e.g. v0.9.1)
      --cluster string                    [PRO] The vCluster.Pro connected cluster to use
      --connect                           If true will run vcluster connect directly after the vcluster was created (default true)
      --create-namespace                  If true the namespace will be created if it does not exist (default true)
      --disable-pro                       If true vcluster will not try to create a vCluster.Pro. You can also use 'vcluster logout' to prevent vCluster from creating any pro clusters
      --distro string                     Kubernetes distro to use for the virtual cluster. Allowed distros: k3s, k0s, k8s, eks (default "k3s")
      --expose                            If true will create a load balancer service to expose the vcluster endpoint
  -h, --help                              help for create
      --kube-config-context-name string   If set, will override the context name of the generated virtual cluster kube config with this name
      --kubernetes-version string         The kubernetes version to use (e.g. v1.20). Patch versions are not supported
  -l, --labels strings                    [PRO] Comma separated labels to apply to the virtualclusterinstance
      --link stringArray                  [PRO] A link to add to the vCluster. E.g. --link 'prod=http://exampleprod.com'
      --params string                     [PRO] If a template is used, this can be used to use a file for the parameters. E.g. --params path/to/my/file.yaml
      --project string                    [PRO] The vCluster.Pro project to use
      --set stringArray                   Set values for helm. E.g. --set 'persistence.enabled=true'
      --set-param stringArray             [PRO] If a template is used, this can be used to set a specific parameter. E.g. --set-param 'my-param=my-value'
      --template string                   [PRO] The vCluster.Pro template to use
      --template-version string           [PRO] The vCluster.Pro template version to use
      --update-current                    If true updates the current kube config (default true)
      --upgrade                           If true will try to upgrade the vcluster instead of failing if it already exists
  -f, --values stringArray                Path where to load extra helm values from
```


## Global & Inherited Flags

```
      --context string      The kubernetes config context to use
      --debug               Prints the stack trace if an error occurs
      --log-output string   The log format to use. Can be either plain, raw or json (default "plain")
  -n, --namespace string    The kubernetes namespace to use
  -s, --silent              Run in silent mode and prevents any vcluster log output except panics & fatals
```

