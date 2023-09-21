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
      --delete-namespace        If enabled, vcluster will delete the namespace of the vcluster. In the case of multi-namespace mode, will also delete all other namespaces created by vcluster
  -h, --help                    help for delete
      --keep-pvc                If enabled, vcluster will not delete the persistent volume claim of the vcluster
```


## Global & Inherited Flags

```
      --context string     The kubernetes config context to use
      --debug              Prints the stack trace if an error occurs
  -n, --namespace string   The kubernetes namespace to use
  -s, --silent             Run in silent mode and prevents any vcluster log output except panics & fatals
```

