# Run Conformance Tests

You will need a cluster with at least 2 nodes.
The steps below assume that you will use a local minikube cluster.
We executed the test on a minikube instance with docker driver
(default auto-detected by system).

## 1. Create a multinode minikube cluster

```bash
minikube start --kubernetes-version 1.33.1 --nodes=2
```

## 2. Start vcluster platform

Start vcluster platform using `vcluster version 0.27.0` (latest) using
the following command:

```bash
vcluster platform start
```

## 3. Create the vcluster

Create a file called `vcluster.yaml` with the following content:

```yaml
controlPlane:
  backingStore:
    etcd:
      deploy:
        enabled: true
        statefulSet:
          image:
            tag: 3.5.21-0
  distro:
    k8s:
      apiServer:
        extraArgs:
          - --service-account-jwks-uri=https://kubernetes.default.svc.cluster.local/openid/v1/jwks
      image:
        tag: v1.33.1
      enabled: true
  service:
    spec:
      type: LoadBalancer
privateNodes:
  enabled: true
```

Create a virtual cluster using `vcluster version 0.27.0` (latest) using
the following command:

```bash
# Create the vCluster
vcluster create vcluster -n vcluster -f vcluster.yaml
```

## 4. Add private nodes to vcluster

Follow the [private nodes](https://www.vcluster.com/docs/vcluster/deploy/worker-nodes/private-nodes/join) documentation to
add 2 private nodes to vcluster.

## 5. Run Tests

Download a binary release of the CLI, or build it yourself by running:

```bash
go install github.com/vmware-tanzu/sonobuoy@latest
```

Deploy a Sonobuoy pod to your cluster with:

```bash
sonobuoy run --mode=certified-conformance --dns-pod-labels=k8s-app=vcluster-kube-dns
```

View actively running pods:

```bash
sonobuoy status
```

To inspect the logs:

```bash
sonobuoy logs
```

Once sonobuoy status shows the run as completed, copy the output directory from
the main Sonobuoy pod to a local directory:

```bash
outfile=$(sonobuoy retrieve)
```

This copies a single .tar.gz snapshot from the Sonobuoy pod into your local .
directory. Extract the contents into ./results with:

```bash
mkdir ./results; tar xzf $outfile -C ./results
```
