### Run Conformance Tests

You will need a cluster with at least 2 nodes.  
The steps below assume that you will use a local minikube cluster.  
We executed the test on a minikube instance with docker driver (default auto-detected by system).


### 1. Create a multinode minikube cluster

```
minikube start --kubernetes-version 1.30.2 --nodes=2
```

### 2. Create the vcluster

Create a file called `values.yaml` with the following content:
```yaml
controlPlane:
  advanced:
    virtualScheduler:
      enabled: true
  backingStore:
    etcd:
      deploy:
        enabled: true
        statefulSet:
          image:
            tag: 3.5.13-0
  distro:
    k8s:
      apiServer:
        extraArgs:
        - --service-account-jwks-uri=https://kubernetes.default.svc.cluster.local/openid/v1/jwks
        image:
          tag: v1.30.2
      controllerManager:
        image:
          tag: v1.30.2
      enabled: true
      scheduler:
        image:
          tag: v1.30.2
  statefulSet:
    scheduling:
      podManagementPolicy: OrderedReady
networking:
  advanced:
    proxyKubelets:
      byHostname: false
      byIP: false
sync:
  fromHost:
    csiDrivers:
      enabled: false
    csiStorageCapacities:
      enabled: false
    nodes:
      enabled: true
      selector:
        all: true
  toHost:
    persistentVolumes:
      enabled: true
    priorityClasses:
      enabled: true
    storageClasses:
      enabled: true
```

Create virtual cluster using `vcluster version 0.20.0-beta.15` (latest) using the following command:
```
# Create the vcluster
vcluster create vcluster -n vcluster -f values.yaml --distro k8s
```

### 3. Run Tests

Download a binary release of the CLI, or build it yourself by running:
```
go install github.com/vmware-tanzu/sonobuoy@latest
```

Deploy a Sonobuoy pod to your cluster with:
```
sonobuoy run --mode=certified-conformance
```
View actively running pods:
```
sonobuoy status
```
To inspect the logs:
```
sonobuoy logs
```
Once sonobuoy status shows the run as completed, copy the output directory from the main Sonobuoy pod to a local directory:
```
outfile=$(sonobuoy retrieve)
```
This copies a single .tar.gz snapshot from the Sonobuoy pod into your local . directory. Extract the contents into ./results with:
```
mkdir ./results; tar xzf $outfile -C ./results
```
