---
title: "vcluster import --help"
sidebar_label: vcluster import
---


Imports a vcluster into a vCluster.Pro project

## Synopsis

```
vcluster import VCLUSTER_NAME [flags]
```

```
########################################################
################### vcluster import ####################
########################################################
Imports a vcluster into a vCluster.Pro project.

Example:
vcluster import my-vcluster --cluster connected-cluster \
--namespace vcluster-my-vcluster --project my-project --importname my-vcluster
#######################################################
```


## Flags

```
      --cluster string      Cluster name of the cluster the virtual cluster is running on
      --disable-upgrade     If true, will disable auto-upgrade of the imported vcluster to vCluster.Pro
  -h, --help                help for import
      --importname string   The name of the vcluster under projects. If unspecified, will use the vcluster name
      --project string      The project to import the vcluster into
```


## Global & Inherited Flags

```
      --context string      The kubernetes config context to use
      --debug               Prints the stack trace if an error occurs
      --log-output string   The log format to use. Can be either plain, raw or json (default "plain")
  -n, --namespace string    The kubernetes namespace to use
  -s, --silent              Run in silent mode and prevents any vcluster log output except panics & fatals
```

