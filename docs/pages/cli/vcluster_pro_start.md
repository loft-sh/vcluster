---
title: "vcluster pro start --help"
sidebar_label: vcluster pro start
sidebar_class_name: "pro-feature-sidebar-item"
---

:::info Note:
`vcluster pro start` is only available in the enterprise-ready [vCluster.Pro](https://vcluster.pro) offering.
:::


Start a vCluster.Pro instance and connect via port-forwarding

## Synopsis

```
vcluster pro start [flags]
```

```
########################################################
################## vcluster pro start ##################
########################################################

Starts a vCluster.Pro instance in your Kubernetes cluster
and then establishes a port-forwarding connection.

Please make sure you meet the following requirements
before running this command:

1. Current kube-context has admin access to the cluster
2. Helm v3 must be installed
3. kubectl must be installed

########################################################
```


## Flags

```
      --chart-name string    The chart name to deploy vCluster.Pro (default "vcluster-control-plane")
      --chart-path string    The vCluster.Pro chart path to deploy vCluster.Pro
      --chart-repo string    The chart repo to deploy vCluster.Pro (default "https://charts.loft.sh/")
      --context string       The kube context to use for installation
      --email string         The email to use for the installation
  -h, --help                 help for start
      --host string          Provide a hostname to enable ingress and configure its hostname
      --local-port string    The local port to bind to if using port-forwarding
      --namespace string     The namespace to install vCluster.Pro into (default "vcluster-pro")
      --no-login             If true, vCluster.Pro will not login to a vCluster.Pro instance on start
      --no-port-forwarding   If true, vCluster.Pro will not do port forwarding after installing it
      --no-tunnel            If true, vCluster.Pro will not create a loft.host tunnel for this installation
      --no-wait              If true, vCluster.Pro will not wait after installing it
      --password string      The password to use for the admin account. (If empty this will be the namespace UID)
      --reset                If true, an existing loft instance will be deleted before installing vCluster.Pro
      --reuse-values         Reuse previous vCluster.Pro helm values on upgrade (default true)
      --upgrade              If true, vCluster.Pro will try to upgrade the release
      --values string        Path to a file for extra vCluster.Pro helm chart values
      --version string       The vCluster.Pro version to install (default "latest")
```


## Global & Inherited Flags

```
      --debug               Prints the stack trace if an error occurs
      --log-output string   The log format to use. Can be either plain, raw or json (default "plain")
  -s, --silent              Run in silent mode and prevents any vcluster log output except panics & fatals
```

