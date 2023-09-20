---
title: "vcluster resume --help"
sidebar_label: vcluster resume
---


Resumes a virtual cluster

## Synopsis


```
vcluster resume VCLUSTER_NAME [flags]
```

```
#######################################################
################### vcluster resume ###################
#######################################################
Resume will start a vcluster after it was paused. 
vcluster will recreate all the workloads after it has 
started automatically.

Example:
vcluster resume test --namespace test
#######################################################
```


## Flags

```
  -h, --help   help for resume
```


## Global & Inherited Flags

```
      --context string     The kubernetes config context to use
      --debug              Prints the stack trace if an error occurs
  -n, --namespace string   The kubernetes namespace to use
  -s, --silent             Run in silent mode and prevents any vcluster log output except panics & fatals
```

