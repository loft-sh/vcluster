---
title: "vcluster pro generate admin-kube-config --help"
sidebar_label: vcluster pro generate admin-kube-config
sidebar_class_name: "pro-feature-sidebar-item"
---

:::info Note:
`vcluster pro generate admin-kube-config` is only available in the enterprise-ready [vCluster.Pro](https://vcluster.pro) offering.
:::


Generates a new kube config for connecting a cluster

## Synopsis


```
vcluster pro generate admin-kube-config [flags]
```

```
#######################################################
######### vcluster pro generate admin-kube-config ###########
#######################################################
Creates a new kube config that can be used to connect
a cluster to vCluster.Pro

Example:
vcluster pro generate admin-kube-config
#######################################################
```


## Flags

```
  -h, --help                     help for admin-kube-config
      --namespace string         The namespace to generate the service account in. The namespace will be created if it does not exist (default "loft")
      --service-account string   The service account name to create (default "loft-admin")
```


## Global & Inherited Flags

```
      --context string      The kubernetes config context to use
      --debug               Prints the stack trace if an error occurs
      --log-output string   The log format to use. Can be either plain, raw or json (default "plain")
  -s, --silent              Run in silent mode and prevents any vcluster log output except panics & fatals
```

