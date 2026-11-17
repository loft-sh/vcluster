# vCluster AI Conformance — v1.35 (Private Nodes)

CNCF Kubernetes AI Platform Conformance submission for vCluster with Private Nodes.

- **Kubernetes version:** v1.35
- **Platform version:** v0.32.0
- **Submission:** https://github.com/cncf/k8s-ai-conformance/tree/main/v1.35/vcluster-private-nodes
- **k8s-conformance baseline:** https://github.com/cncf/k8s-conformance/tree/master/v1.35/vcluster-with-private-nodes

## What is vCluster with Private Nodes?

Private Nodes attach real physical Kubernetes nodes exclusively to a virtual cluster.
This gives the vCluster dedicated GPU hardware for AI/ML workloads with node-level isolation —
no other workload on the host cluster can schedule onto those nodes.

## Files

| File | Purpose |
|------|---------|
| `PRODUCT.yaml` | CNCF AI conformance self-assessment — 8 MUST + 3 SHOULD items with evidence |
| `README.md` | This file |

## Conformance requirements covered

| Category | Requirement | Level | Status |
|----------|-------------|-------|--------|
| Accelerators | DRA support (ResourceClaim, DeviceClass) | MUST | Implemented |
| Accelerators | Driver & runtime management (NVIDIA GPU Operator) | SHOULD | Implemented |
| Accelerators | GPU sharing (MIG, time-slicing) | SHOULD | Implemented |
| Accelerators | Virtualized accelerators (vGPU) | SHOULD | Implemented |
| Networking | Gateway API for AI inference traffic management | MUST | Implemented |
| Scheduling | Gang scheduling (Volcano, NVIDIA KAI) | MUST | Implemented |
| Scheduling | Cluster autoscaling with GPU node groups (Auto Nodes) | MUST | Implemented |
| Scheduling | HPA with GPU custom metrics (DCGM + Prometheus Adapter) | MUST | Implemented |
| Observability | Accelerator metrics (DCGM-Exporter, OTel) | MUST | Implemented |
| Observability | AI service metrics (Prometheus discovery) | MUST | Implemented |
| Security | Secure accelerator access (node-level + DRA isolation) | MUST | Implemented |
| Operators | AI CRD operators (Kubeflow, KServe, webhooks) | MUST | Implemented |

## Re-certification

Run `/k8s-ai-conformance-research` annually when a new Kubernetes minor version is released
to check for spec changes and dead evidence URLs before updating for the next cycle.
