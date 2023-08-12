### Run Conformance Tests

You will need a cluster with at least 2 nodes.  
The steps below assume that you will use a local minikube cluster.  
We executed the test on a minikube instance with kvm2 driver.


### 1. Create a multinode minikube cluster

```
minikube start --kubernetes-version 1.27.2 --nodes=2
```

### 2. Create the vcluster

Create a file called `values.yaml` with the following content:
```yaml
syncer:
  extraArgs:
  - --disable-fake-kubelets
vcluster:
  image: rancher/k3s:v1.27.2-k3s1
sync:
  pods:
    ephemeralContainers: true
  nodes:
    enabled: true
    syncAllNodes: true
    enableScheduler: true
  priorityclasses:
    enabled: true
  ingresses:
    enabled: false
  csistoragecapacities:
    enabled: false
  csidrivers:
    enabled: false
```

Now create the vcluster with the [vcluster cli](https://github.com/loft-sh/vcluster/releases) (version v0.15.2 or newer):
```
# Create the vcluster
vcluster create vcluster -n vcluster -f values.yaml
```

### 3. Run Tests

Install [sonobuoy](https://github.com/vmware-tanzu/sonobuoy)(version v0.56.15 or newer) and run:
```
export SONOBUOY_IMAGE_VERSION=v0.56.15
export SONOBUOY_LOGS_IMAGE_VERSION=v0.4
sonobuoy run \
  --mode=certified-conformance \
  --kubernetes-version=v1.27.2 \
  --sonobuoy-image=sonobuoy/sonobuoy:$SONOBUOY_IMAGE_VERSION \
  --systemd-logs-image=sonobuoy/systemd-logs:$SONOBUOY_LOGS_IMAGE_VERSION \
  --wait
```
