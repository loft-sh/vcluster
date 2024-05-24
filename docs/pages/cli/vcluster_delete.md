---
title: "vcluster delete --help"
sidebar_label: vcluster delete
---


Deletes a virtual cluster

## Synopsis


```
vcluster delete VCLUSTER_NAME [flags]
```

```
#######################################################
################### vcluster delete ###################
#######################################################
Deletes a virtual cluster

Example:
vcluster delete test --namespace test
#######################################################
```


## Flags

```
      --auto-delete-namespace   If enabled, vcluster will delete the namespace of the vcluster if it was created by vclusterctl. In the case of multi-namespace mode, will also delete all other namespaces created by vcluster (default true)
      --delete-configmap        If enabled, vCluster will delete the ConfigMap of the vCluster
      --delete-namespace        If enabled, vcluster will delete the namespace of the vcluster. In the case of multi-namespace mode, will also delete all other namespaces created by vcluster
  -h, --help                    help for delete
      --ignore-not-found        If enabled, vcluster will not error out in case the target vcluster does not exist
      --keep-pvc                If enabled, vcluster will not delete the persistent volume claim of the vcluster
      --project string          [PRO] The pro project the vcluster is in
      --wait                    If enabled, vcluster will wait until the vcluster is deleted (default true)
```


## Global & Inherited Flags

```
      --context string      The kubernetes config context to use
      --debug               Prints the stack trace if an error occurs
      --log-output string   The log format to use. Can be either plain, raw or json (default "plain")
  -n, --namespace string    The kubernetes namespace to use
  -s, --silent              Run in silent mode and prevents any vcluster log output except panics & fatals
```

