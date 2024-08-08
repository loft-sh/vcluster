#!/bin/bash


VCLUSTER_NAME="${VCLUSTER_NAME}"      
VCLUSTER_NAMESPACE="${VCLUSTER_NAMESPACE}" 
POLL_INTERVAL=10  
MAX_WAIT_TIME=180 
MANIFEST_FILE="vcluster-manifest.yaml"
PATH_TO_VALUES_FILE="./test/functional_tests/commonValues.yaml"
HELM_CHART_DIR="./chart"

if [ -z "$VCLUSTER_NAME" ]; then
    echo "VCLUSTER_NAME environment variable is not set."
    exit 1
fi

if [ -z "$VCLUSTER_NAMESPACE" ]; then
    echo "VCLUSTER_NAMESPACE environment variable is not set."
    exit 1
fi

sed -i "s|REPLACE_REPOSITORY_NAME|${REPOSITORY_NAME}|g" $PATH_TO_VALUES_FILE
sed -i "s|REPLACE_TAG_NAME|${TAG_NAME}|g" $PATH_TO_VALUES_FILE

echo "Creating namespace"
kubectl create namespace $VCLUSTER_NAMESPACE

echo "Generating vCluster manifest for $VCLUSTER_NAME in namespace $VCLUSTER_NAMESPACE..."
helm template $VCLUSTER_NAME $HELM_CHART_DIR -n $VCLUSTER_NAMESPACE -f $PATH_TO_VALUES_FILE > $MANIFEST_FILE

echo "$PATH_TO_VALUES_FILE"

if [ $? -ne 0 ]; then
    echo "Failed to generate vCluster manifest."
    exit 1
fi

echo "Applying vCluster manifest..."
kubectl apply -f $MANIFEST_FILE

if [ $? -ne 0 ]; then
    echo "Failed to create vCluster."
    exit 1
fi

echo "Polling for vCluster $VCLUSTER_NAME to be in Running state..."
check_vcluster_running() {
    vcluster list -n $VCLUSTER_NAMESPACE | grep -q "Running"
}

elapsed_time=0
while [ $elapsed_time -le $MAX_WAIT_TIME ]; do
    if check_vcluster_running; then
        echo "vCluster $VCLUSTER_NAME is in Running state."
        break
    fi
    
    echo "vCluster $VCLUSTER_NAME is not in Running state yet. Waiting..."
    sleep $POLL_INTERVAL
    elapsed_time=$((elapsed_time + POLL_INTERVAL))
done

if ! check_vcluster_running; then
    echo "vCluster $VCLUSTER_NAME did not reach Running state within $MAX_WAIT_TIME seconds."
    exit 1
fi

echo "Deleting vCluster $VCLUSTER_NAME using manifest..."
kubectl delete -f $MANIFEST_FILE

if [ $? -ne 0 ]; then
    echo "Failed to delete vCluster."
    exit 1
fi

echo "vCluster $VCLUSTER_NAME has been deleted."

echo "Delete namespace $VCLUSTER_NAMESPACE"
kubectl delete namespace $VCLUSTER_NAMESPACE