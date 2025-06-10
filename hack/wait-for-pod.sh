#!/bin/bash

NAMESPACE=""
LABEL_SELECTOR=""
POD_NAME=""
CONTEXT=""

# Parse command-line arguments
while [[ $# -gt 0 ]]; do
  case "$1" in
  -n | --namespace)
    NAMESPACE="$2"
    shift 2
    ;;
  -l | --label-selector)
    LABEL_SELECTOR="$2"
    shift 2
    ;;
  -c | --context)
    CONTEXT="$2"
    shift 2
    ;;
  *)
    echo "Invalid argument: $1"
    exit 1
    ;;
  esac
done

# Validate required arguments
if [[ -z "$NAMESPACE" ]]; then
  echo "Namespace is required. Use the '-n' or '--namespace' flag."
  exit 1
fi

if [[ -z "$LABEL_SELECTOR" ]]; then
  echo "Label selector is required. Use the '-l' or '--label-selector' flag."
  exit 1
fi

# Loop until a pod with the given label selector is created
while [[ -z "$POD_NAME" ]]; do
  POD_NAME=$(kubectl get pod --context="$CONTEXT" -n "$NAMESPACE" -l "$LABEL_SELECTOR" --output=jsonpath='{.items[0].metadata.name}' 2>/dev/null)
  if [[ -z "$POD_NAME" ]]; then
    echo "Pod with label selector '$LABEL_SELECTOR' not found in context '$CONTEXT'. Waiting..."
    sleep 5
  fi
done

# Wait for the pod to be ready
kubectl wait --context="$CONTEXT" --for=condition=Ready -n $NAMESPACE pod -l $LABEL_SELECTOR --timeout=5m
