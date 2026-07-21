### Run Conformance Tests

We recommend to use GKE as host cluster for conformance tests, as you will need a cluster with at least 2 nodes.


### 1. Create GKE cluster

```
export PROJECT_NAME=my-google-cloud-project
export CLUSTER_NAME=conformance-test
export CLUSTER_ZONE=europe-west3-a

# At the time of writing 1.21.5 was newest in
# regular channel
export CLUSTER_VERSION=1.21.5-gke.1302
export CLUSTER_CHANNEL=regular

# Create the cluster
gcloud beta container --project $PROJECT_NAME clusters create $CLUSTER_NAME \
   --zone $CLUSTER_ZONE --no-enable-basic-auth --cluster-version $CLUSTER_VERSION \
   --release-channel $CLUSTER_CHANNEL --enable-ip-alias --no-enable-master-authorized-networks \
   --addons GcePersistentDiskCsiDriver --node-locations $CLUSTER_ZONE
   
# Make sure you have a firewall rule that allows incoming connections or the NodePort
# tests will fail
gcloud compute firewall-rules --project $PROJECT_NAME create conformance-firewall-rules --direction=INGRESS --network=default --action=ALLOW --rules=tcp --source-ranges=0.0.0.0/0 --description="vcluster conformance test firewall rule"
```

### 2. Create the vcluster

Create a file called `values.yaml` with the following content:
```yaml
vcluster:
  image: rancher/k3s:v1.21.5-k3s1
# Tolerate everything as the test will taint some nodes
tolerations:
- operator: "Exists"
rbac:
  clusterRole:
    create: true
syncer:
  extraArgs:
  - --sync-all-nodes
  - --sync-node-changes
  - --fake-nodes=false
  - --fake-kubelets=false
  - --enable-priority-classes
  - --disable-sync-resources=ingresses
```

Now create the vcluster with the [vcluster cli](https://github.com/loft-sh/vcluster/releases) (at least version v0.4.5 or newer):
```
# Create the vcluster
vcluster create vcluster -n vcluster -f values.yaml --expose

# Connect to the vcluster 
vcluster connect vcluster -n vcluster
```

### 3. Run Tests

Install [sonobuoy](https://github.com/vmware-tanzu/sonobuoy) and run:
```
export KUBECONFIG=./kubeconfig.yaml
export CONFORMANCE_VERSION=v1.21.1
export SONOBUOY_IMAGE_VERSION=v0.52.0
export SONOBUOY_LOGS_IMAGE_VERSION=v0.3

sonobuoy run \
  --mode=certified-conformance \
  --kube-conformance-image-version=$CONFORMANCE_VERSION \
  --sonobuoy-image=sonobuoy/sonobuoy:$SONOBUOY_IMAGE_VERSION \
  --systemd-logs-image=sonobuoy/systemd-logs:$SONOBUOY_LOGS_IMAGE_VERSION \
  --wait
```

