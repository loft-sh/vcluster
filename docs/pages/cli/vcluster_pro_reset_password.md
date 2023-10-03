---
title: "vcluster pro reset password --help"
sidebar_label: vcluster pro reset password
sidebar_class_name: "pro-feature-sidebar-item"
---

:::info Note:
`vcluster pro reset password` is only available in the enterprise-ready [vCluster.Pro](https://vcluster.pro) offering.
:::


Resets the password of a user

## Synopsis

```
vcluster pro reset password [flags]
```

```
########################################################
############## vcluster pro reset password #############
########################################################
Resets the password of a user.

Example:
vcluster pro reset password
vcluster pro reset password --user admin
#######################################################
```


## Flags

```
      --create            Creates the user if it does not exist
      --force             If user had no password will create one
  -h, --help              help for password
      --password string   The new password to use
      --user string       The name of the user to reset the password (default "admin")
```


## Global & Inherited Flags

```
      --context string      The kubernetes config context to use
      --debug               Prints the stack trace if an error occurs
      --log-output string   The log format to use. Can be either plain, raw or json (default "plain")
  -n, --namespace string    The kubernetes namespace to use
  -s, --silent              Run in silent mode and prevents any vcluster log output except panics & fatals
```

