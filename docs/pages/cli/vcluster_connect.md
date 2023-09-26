---
title: "vcluster connect --help"
sidebar_label: vcluster connect
---


Connect to a virtual cluster

## Synopsis


```
vcluster connect VCLUSTER_NAME [flags]
```

```
#######################################################
################## vcluster connect ###################
#######################################################
Connect to a virtual cluster

Example:
vcluster connect test --namespace test
# Open a new bash with the vcluster KUBECONFIG defined
vcluster connect test -n test -- bash
vcluster connect test -n test -- kubectl get ns
#######################################################
```


## Flags

```
      --address string                    The local address to start port forwarding under
      --background-proxy                  If specified, vcluster will create the background proxy in docker [its mainly used for vclusters with no nodeport service.]
      --cluster-role string               If specified, vcluster will create the service account if it does not exist and also add a cluster role binding for the given cluster role to it. Requires --service-account to be set
  -h, --help                              help for connect
      --insecure                          If specified, vcluster will create the kube config with insecure-skip-tls-verify
      --kube-config string                Writes the created kube config to this file (default "./kubeconfig.yaml")
      --kube-config-context-name string   If set, will override the context name of the generated virtual cluster kube config with this name
      --local-port int                    The local port to forward the virtual cluster to. If empty, vcluster will use a random unused port
      --pod string                        The pod to connect to
      --print                             When enabled prints the context to stdout
      --project string                    [PRO] The pro project the vcluster is in
      --server string                     The server to connect to
      --service-account string            If specified, vcluster will create a service account token to connect to the virtual cluster instead of using the default client cert / key. Service account must exist and can be used as namespace/name.
      --token-expiration int              If specified, vcluster will create the service account token for the given duration in seconds. Defaults to eternal
      --update-current                    If true updates the current kube config (default true)
```


## Global & Inherited Flags

```
      --context string      The kubernetes config context to use
      --debug               Prints the stack trace if an error occurs
      --log-output string   The log format to use. Can be either plain, raw or json (default "plain")
  -n, --namespace string    The kubernetes namespace to use
  -s, --silent              Run in silent mode and prevents any vcluster log output except panics & fatals
```

