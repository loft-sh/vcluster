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
  -h, --help             help for resume
      --project string   [PRO] The pro project the vcluster is in
```


## Global & Inherited Flags

```
      --context string      The kubernetes config context to use
      --debug               Prints the stack trace if an error occurs
      --log-output string   The log format to use. Can be either plain, raw or json (default "plain")
  -n, --namespace string    The kubernetes namespace to use
  -s, --silent              Run in silent mode and prevents any vcluster log output except panics & fatals
```

