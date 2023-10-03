---
title: "vcluster login --help"
sidebar_label: vcluster login
---


Login to a vCluster.Pro instance

## Synopsis

```
vcluster login [VCLUSTER_PRO_HOST] [flags]
```

```
########################################################
#################### vcluster login ####################
########################################################
Login into vCluster.Pro

Example:
vcluster login https://my-vcluster-pro.com
vcluster login https://my-vcluster-pro.com --access-key myaccesskey
########################################################
```


## Flags

```
      --access-key string   The access key to use
      --docker-login        If true, will log into the docker image registries the user has image pull secrets for (default true)
  -h, --help                help for login
      --insecure            Allow login into an insecure Loft instance (default true)
```


## Global & Inherited Flags

```
      --context string      The kubernetes config context to use
      --debug               Prints the stack trace if an error occurs
      --log-output string   The log format to use. Can be either plain, raw or json (default "plain")
  -n, --namespace string    The kubernetes namespace to use
  -s, --silent              Run in silent mode and prevents any vcluster log output except panics & fatals
```

