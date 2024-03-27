---
title: "vcluster pro token --help"
sidebar_label: vcluster pro token
sidebar_class_name: "pro-feature-sidebar-item"
---

:::info Note:
`vcluster pro token` is only available in the enterprise-ready [vCluster.Pro](https://vcluster.pro) offering.
:::


Token prints the access token to a vCluster.Pro instance

## Synopsis

```
vcluster pro token [flags]
```

```
########################################################
################## vcluster pro token ##################
########################################################

Prints an access token to a vCluster.Pro instance. This
can be used as an ExecAuthenticator for kubernetes

Example:
vcluster pro token
########################################################
```


## Flags

```
      --direct-cluster-endpoint   When enabled prints a direct cluster endpoint token
  -h, --help                      help for token
      --project string            The project containing the virtual cluster
      --virtual-cluster string    The virtual cluster
```


## Global & Inherited Flags

```
      --context string      The kubernetes config context to use
      --debug               Prints the stack trace if an error occurs
      --log-output string   The log format to use. Can be either plain, raw or json (default "plain")
  -n, --namespace string    The kubernetes namespace to use
  -s, --silent              Run in silent mode and prevents any vcluster log output except panics & fatals
```

