---
title: "vcluster pro connect cluster --help"
sidebar_label: vcluster pro connect cluster
sidebar_class_name: "pro-feature-sidebar-item"
---

:::info Note:
`vcluster pro connect cluster` is only available in the enterprise-ready [vCluster.Pro](https://vcluster.pro) offering.
:::


connect current cluster to vCluster.Pro

## Synopsis

```
vcluster pro connect cluster [flags]
```

```
#######################################################
############ vcluster pro connect cluster #############
#######################################################
Connect a cluster to the vCluster.Pro instance.

Example:
vcluster pro connect cluster my-cluster
########################################################
```


## Flags

```
      --context string           The kube context to use for installation
      --display-name string      The display name to show in the UI for this cluster
  -h, --help                     help for cluster
      --namespace string         The namespace to generate the service account in. The namespace will be created if it does not exist (default "loft")
      --service-account string   The service account name to create (default "loft-admin")
      --wait                     If true, will wait until the cluster is initialized
```


## Global & Inherited Flags

```
      --debug               Prints the stack trace if an error occurs
      --log-output string   The log format to use. Can be either plain, raw or json (default "plain")
  -s, --silent              Run in silent mode and prevents any vcluster log output except panics & fatals
```

