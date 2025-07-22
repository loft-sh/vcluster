#!/bin/bash

# Test script to verify CAST AI integration in vCluster Helm chart

echo "Testing vCluster CAST AI integration..."

# Test 1: CAST AI enabled
echo "Test 1: CAST AI enabled"
helm template test-vcluster ./chart \
  --set castai.enabled=true \
  --set castai.workloadName="test-workload" \
  --show-only templates/config-secret.yaml | \
  grep -A 10 -B 10 "workloads.cast.ai"

echo ""

# Test 2: CAST AI disabled
echo "Test 2: CAST AI disabled"
helm template test-vcluster ./chart \
  --set castai.enabled=false \
  --show-only templates/config-secret.yaml | \
  grep -A 5 -B 5 "workloads.cast.ai" || echo "No CAST AI labels found (expected)"

echo ""
echo "Testing complete!"