# vCluster AI Conformance — v1.34 (Private Nodes)

CNCF Kubernetes AI Platform Conformance submission for vCluster with Private Nodes.

- **Kubernetes version:** v1.34
- **Platform version:** v0.32.0
- **Submission:** https://github.com/cncf/k8s-ai-conformance/tree/main/v1.34/vcluster-private-nodes
- **k8s-conformance baseline:** https://github.com/cncf/k8s-conformance/tree/master/v1.34/vcluster-private-nodes

## What is vCluster with Private Nodes?

Private Nodes attach real physical Kubernetes nodes exclusively to a virtual cluster.
This gives the vCluster dedicated GPU hardware for AI/ML workloads with node-level isolation —
no other workload on the host cluster can schedule onto those nodes.

## Files

| File | Purpose |
|------|---------|
| `PRODUCT.yaml` | CNCF AI conformance self-assessment — all 8 MUST items with evidence |
| `README.md` | This file |

## Conformance requirements covered

| Category | Requirement | Status |
|----------|-------------|--------|
| Accelerators | DRA support (ResourceClaim, DeviceClass) | Implemented |
| Networking | Gateway API for AI inference traffic management | Implemented |
| Scheduling | Gang scheduling (Volcano, NVIDIA KAI) | Implemented |
| Scheduling | Cluster autoscaling with GPU node groups (Auto Nodes) | Implemented |
| Scheduling | HPA with GPU custom metrics (DCGM + Prometheus Adapter) | Implemented |
| Observability | Accelerator metrics (DCGM-Exporter, OTel) | Implemented |
| Observability | AI service metrics (Prometheus discovery) | Implemented |
| Security | Secure accelerator access (node-level + DRA isolation) | Implemented |
| Operators | AI CRD operators (Kubeflow, KServe, webhooks) | Implemented |

## Re-certification

Run `/k8s-ai-conformance-research` annually when a new Kubernetes minor version is released
to check for spec changes and dead evidence URLs before updating for the next cycle.
