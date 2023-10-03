---
title: "vcluster disconnect --help"
sidebar_label: vcluster disconnect
---


Disconnects from a virtual cluster

## Synopsis


```
vcluster disconnect [flags]
```

```
#######################################################
################# vcluster disconnect #################
#######################################################
Disconnect switches back the kube context if
vcluster connect --update-current was used

Example:
vcluster connect --update-current
vcluster disconnect
#######################################################
```


## Flags

```
  -h, --help   help for disconnect
```


## Global & Inherited Flags

```
      --context string      The kubernetes config context to use
      --debug               Prints the stack trace if an error occurs
      --log-output string   The log format to use. Can be either plain, raw or json (default "plain")
  -n, --namespace string    The kubernetes namespace to use
  -s, --silent              Run in silent mode and prevents any vcluster log output except panics & fatals
```

