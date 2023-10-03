---
title: "vcluster list --help"
sidebar_label: vcluster list
---


Lists all virtual clusters

## Synopsis


```
vcluster list [flags]
```

```
#######################################################
#################### vcluster list ####################
#######################################################
Lists all virtual clusters

Example:
vcluster list
vcluster list --output json
vcluster list --namespace test
#######################################################
```


## Flags

```
  -h, --help            help for list
      --output string   Choose the format of the output. [table|json] (default "table")
```


## Global & Inherited Flags

```
      --context string      The kubernetes config context to use
      --debug               Prints the stack trace if an error occurs
      --log-output string   The log format to use. Can be either plain, raw or json (default "plain")
  -n, --namespace string    The kubernetes namespace to use
  -s, --silent              Run in silent mode and prevents any vcluster log output except panics & fatals
```

