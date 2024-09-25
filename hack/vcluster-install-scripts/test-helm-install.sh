#!/bin/bash


VCLUSTER_NAME="${VCLUSTER_NAME}"      
VCLUSTER_NAMESPACE="${VCLUSTER_NAMESPACE}"
POLL_INTERVAL=10  
MAX_WAIT_TIME=180
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

if [ ! -f "$PATH_TO_VALUES_FILE" ]; then
    echo "Values file $PATH_TO_VALUES_FILE does not exist."
    exit 1
fi

sed -i "s|REPLACE_REPOSITORY_NAME|${REPOSITORY_NAME}|g" $PATH_TO_VALUES_FILE
sed -i "s|REPLACE_TAG_NAME|${TAG_NAME}|g" $PATH_TO_VALUES_FILE

echo "Creating namespace"
kubectl create namespace $VCLUSTER_NAMESPACE

echo "Installing or upgrading vCluster $VCLUSTER_NAME in namespace $VCLUSTER_NAMESPACE..."
helm upgrade --install $VCLUSTER_NAME $HELM_CHART_DIR \
  --values $PATH_TO_VALUES_FILE \
  --namespace $VCLUSTER_NAMESPACE \
  --repository-config=''

if [ $? -ne 0 ]; then
    echo "Failed to install or upgrade vCluster."
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

echo "Deleting vCluster $VCLUSTER_NAME..."
helm uninstall $VCLUSTER_NAME -n $VCLUSTER_NAMESPACE

if [ $? -ne 0 ]; then
    echo "Failed to delete vCluster."
    exit 1
fi

echo "vCluster $VCLUSTER_NAME has been deleted."
kubectl delete namespace $VCLUSTER_NAMESPACE