---
title: "vcluster get service-cidr --help"
sidebar_label: vcluster get service-cidr
---


Prints Service CIDR of the cluster

## Synopsis


```
vcluster get service-cidr [flags]
```

```
#######################################################
############### vcluster get service-cidr  ############
#######################################################
Prints Service CIDR of the cluster

Ex:
vcluster get service-cidr
10.96.0.0/12
#######################################################
```


## Flags

```
  -h, --help   help for service-cidr
```


## Global & Inherited Flags

```
      --context string      The kubernetes config context to use
      --debug               Prints the stack trace if an error occurs
      --log-output string   The log format to use. Can be either plain, raw or json (default "plain")
  -n, --namespace string    The kubernetes namespace to use
  -s, --silent              Run in silent mode and prevents any vcluster log output except panics & fatals
```

