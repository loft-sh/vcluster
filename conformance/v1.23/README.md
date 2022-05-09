### Run Conformance Tests

We recommend to use GKE as host cluster for conformance tests, as you will need a cluster with at least 2 nodes.


### 1. Create GKE cluster

```
export PROJECT_NAME=my-google-cloud-project
export CLUSTER_NAME=conformance-test
export CLUSTER_ZONE=europe-west3-a

# At the time of writing v1.23.5-gke.200 was newest in
# rapid channel
export CLUSTER_VERSION=v1.23.5-gke.200
export CLUSTER_CHANNEL=rapid

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
syncer:
  extraArgs:
  - --disable-fake-kubelets
vcluster:
  image: rancher/k3s:v1.23.6-k3s1
sync:
  nodes:
    enabled: true
    syncAllNodes: true
    enableScheduler: true
  priorityclasses:
    enabled: true
  ingresses:
    enabled: false
service:
  type: LoadBalancer
```

Now create the vcluster with the [vcluster cli](https://github.com/loft-sh/vcluster/releases) (at least version v0.8.0 or newer):
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
export SONOBUOY_IMAGE_VERSION=v0.55.1
export SONOBUOY_LOGS_IMAGE_VERSION=v0.4
sonobuoy run \
  --mode=certified-conformance \
  --kubernetes-version=v1.23.6 
  --sonobuoy-image=sonobuoy/sonobuoy:$SONOBUOY_IMAGE_VERSION \
  --systemd-logs-image=sonobuoy/systemd-logs:$SONOBUOY_LOGS_IMAGE_VERSION \
  --wait
```
