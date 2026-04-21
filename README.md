<div align="center">
  <a href="https://www.vcluster.com">
    <picture>
      <source media="(prefers-color-scheme: dark)" srcset="docs/static/media/vcluster_horizontal_orange_white.svg">
      <source media="(prefers-color-scheme: light)" srcset="docs/static/media/vcluster_horizontal_orange_black.svg">
      <img alt="vCluster" src="docs/static/media/vcluster_horizontal_orange_white.svg" width="400">
    </picture>
  </a>
  <p><strong>Tenant Clusters for Production Kubernetes and AI Infrastructure</strong></p>
  <p><em>Virtual control planes, real isolation — from a single node to 100K-GPU superclusters.</em></p>

[![GitHub stars](https://img.shields.io/github/stars/loft-sh/vcluster?style=for-the-badge&logo=github&color=orange)](https://github.com/loft-sh/vcluster/stargazers)
[![Slack](https://img.shields.io/badge/Slack-5K+-4A154B?style=for-the-badge&logo=slack&logoColor=white)](https://slack.loft.sh/)
[![LinkedIn](https://img.shields.io/badge/LinkedIn-28K+-0A66C2?style=for-the-badge&logo=linkedin&logoColor=white)](https://www.linkedin.com/company/vcluster)
[![X](https://img.shields.io/badge/X-3.7K+-000000?style=for-the-badge&logo=x&logoColor=white)](https://x.com/loft_sh)

**[Website](https://www.vcluster.com)** • **[Quickstart](https://www.vcluster.com/docs/get-started/)** • **[Documentation](https://www.vcluster.com/docs/vcluster/introduction/what-are-virtual-clusters)** • **[Blog](https://loft.sh/blog)** • **[Slack](https://slack.loft.sh/)**

</div>

---

## What is vCluster?

**vCluster** creates **Tenant Clusters** — fully isolated Kubernetes environments that run on top of a Control Plane Cluster, on dedicated infrastructure, or standalone on bare metal. Each tenant gets its own API server, CRDs, and RBAC, with a cluster experience indistinguishable from a dedicated Kubernetes cluster.

Built for production. Trusted in production. **40M+ Tenant Clusters deployed** by teams at Adobe, CoreWeave, NVIDIA, Lintasarta, Atlan, Deloitte, and hundreds of AI clouds, AI factories, and Fortune 500 platform organizations.

> **The public-cloud experience, on your own infrastructure.** Give every team the Kubernetes they need — with strict isolation, hardware-aware scheduling, and zero tenant sprawl — whether you run one region or 100K GPUs.

<div align="center">

![vCluster demo — create a Tenant Cluster locally with vind, in seconds](./docs/static/media/vcluster-github-demo.gif)

</div>

---

## 🚀 Quick Start

```bash
# Install vCluster CLI
brew install loft-sh/tap/vcluster

# Create a Tenant Cluster
vcluster create my-vcluster --namespace team-x

# Use kubectl as usual — you're now in your Tenant Cluster
kubectl get namespaces
```

**Prerequisites:** A running Kubernetes cluster and `kubectl` configured. Or go straight to bare metal with [vCluster Standalone](https://www.vcluster.com/docs/vcluster/deploy/control-plane/binary/).

👉 **[Full Quickstart Guide](https://www.vcluster.com/docs/get-started)**

### 🐳 Run Locally with Docker — [vind](https://github.com/loft-sh/vind)

No Kubernetes cluster? Run vCluster directly on Docker with **vind** (vCluster in Docker) — like `kind`, but with the full vCluster feature set (UI, sleep/resume, LoadBalancer, image cache):

```bash
vcluster create my-vcluster --driver docker
kubectl get namespaces
```

### 🎮 Try in the Browser

[![Try on Killercoda](https://img.shields.io/badge/Try%20on-Killercoda-22B573?style=for-the-badge&logo=kubernetes&logoColor=white)](https://killercoda.com/vcluster)

### 🎁 vCluster Free Tier

Real usage, not a gated demo. Unlimited Tenant Clusters up to 64 CPUs / 32 GPUs, Private Nodes, Auto Nodes, Standalone, and the Platform UI — for free. **[Get Started Free →](https://www.vcluster.com/free)**

---

## 🆕 What's New

| Version | Feature | Description |
|---------|---------|-------------|
| **v0.33** | [Enterprise Reliability & Storage](https://github.com/loft-sh/vcluster/releases/tag/v0.33.0) | Automatic leaf-cert regeneration, Azure Blob snapshot destinations, workload-level sleep annotations |
| **v0.32** | [Docker Driver & DRA](https://github.com/loft-sh/vcluster/releases/tag/v0.32.0) | Run vCluster on Docker, Dynamic Resource Allocation (DRA) for GPU workloads, in-place pod resizing |
| **v0.31** | [Snapshots & Cross-Cluster APIs](https://github.com/loft-sh/vcluster/releases/tag/v0.31.0) | Expanded snapshot/restore lifecycle, PDBs for Tenant Cluster control planes, cross-cluster resource proxying |
| **v0.30** | [vCluster VPN & Netris Integration](https://www.vcluster.com/releases/en/changelog/platform-v45-and-vcluster-v030-secure-cloud-bursting-on-prem) | Tailscale-powered overlay networking and automated hardware isolation via Netris |
| **v0.27–v0.29** | [Architecture Foundations](https://www.vcluster.com/docs/vcluster/introduction/architecture/) | [Private Nodes](https://www.vcluster.com/docs/vcluster/deploy/worker-nodes/private-nodes) (v0.27, CNI/CSI isolation), [Auto Nodes](https://www.vcluster.com/docs/vcluster/deploy/worker-nodes/private-nodes/auto-nodes/) (v0.28, Karpenter autoscaling), [Standalone Mode](https://www.vcluster.com/docs/vcluster/deploy/control-plane/binary/) (v0.29, bare metal / no Control Plane Cluster) |

👉 **[Full Changelog](https://www.vcluster.com/releases)**

---

## 🎯 Use Cases

| Use Case | Description | Learn More |
|----------|-------------|------------|
| **AI Factory** | Run AI on-prem where your data and GPUs live. Give every team the GPU access they need without multiplying infrastructure. | [View →](https://www.vcluster.com/solutions/ai-factory) |
| **AI Cloud Providers** | Launch a hyperscaler-like Kubernetes experience for your GPU customers. Isolated, production-grade, in minutes. | [View →](https://www.vcluster.com/solutions/gpu-cloud-providers) |
| **Internal GPU Platform** | Maximize GPU utilization without sacrificing isolation. Self-service Kubernetes for AI/ML teams. | [View →](https://www.vcluster.com/solutions/internal-gpu-platform) |
| **Bare Metal Kubernetes** | Run production Kubernetes on bare metal with zero VMs. Isolation without expensive virtualization overhead. | [View →](https://www.vcluster.com/solutions/bare-metal-kubernetes) |
| **Software Vendors** | Ship Kubernetes-native products. Each customer gets their own isolated Tenant Cluster. | [View →](https://www.vcluster.com/solutions/software-vendors) |
| **Environments & Cost Savings** | Consolidate clusters, pause idle workloads with sleep mode, and cut Kubernetes cost at scale. | [View →](https://www.vcluster.com/cost-savings) |

---

## 🏗️ Architectures

vCluster supports multiple deployment architectures. Each builds on the previous, offering progressively stronger isolation — from dense shared infrastructure to fully standalone bare metal.

### Architecture Comparison

| | **Shared Nodes** | **Dedicated Nodes** | **Private Nodes** | **Standalone** |
|---|:---:|:---:|:---:|:---:|
| **Control Plane Cluster** | Required | Required | Required | Not Required |
| **Node Isolation** | ❌ | ✅ | ✅ | ✅ |
| **CNI/CSI Isolation** | ❌ | ❌ | ✅ | ✅ |
| **Bare Metal Ready** | — | — | ✅ | ✅ |
| **Best For** | Dev/test, density | Production tenants | Compliance, GPU | AI factories, edge |

👉 **[Full Architecture Guide](https://www.vcluster.com/docs/vcluster/introduction/architecture/)**

### Minimal Configuration

<details>
<summary>🔹 Shared Nodes — Maximum density, minimum cost</summary>
Tenant Clusters share the Control Plane Cluster's nodes. Workloads run as regular pods in a namespace.
<div align="center">
<img src="./assets/vcluster-architecture-shared-nodes.png" alt="Shared Nodes Architecture" width="600">
</div>

```yaml
sync:
  fromHost:
    nodes:
      enabled: false  # Uses pseudo nodes
```
</details>
<details>
<summary>🔹 Dedicated Nodes — Isolated compute on labeled node pools</summary>
Tenant Clusters get their own set of labeled nodes on the Control Plane Cluster. Workloads are isolated but still managed by the Control Plane Cluster.
<div align="center">
<img src="./assets/vcluster-architecture-dedicated-nodes.png" alt="Dedicated Nodes Architecture" width="600">
</div>

```yaml
sync:
  fromHost:
    nodes:
      enabled: true
      selector:
        labels:
          tenant: my-tenant
```
</details>
<details>
<summary>🔹 Private Nodes <sup>v0.27+</sup> — Full CNI/CSI isolation</summary>
External nodes join the Tenant Cluster directly with their own CNI, CSI, and networking stack. Complete workload isolation from the Control Plane Cluster.
<div align="center">
<img src="./assets/vcluster-architecture-private-nodes.png" alt="Private Nodes Architecture" width="600">
</div>

```yaml
privateNodes:
  enabled: true
controlPlane:
  service:
    spec:
      type: NodePort
```
</details>
<details>
<summary>🔹 vCluster Standalone <sup>v0.29+</sup> — No Control Plane Cluster required</summary>
Run vCluster without any Control Plane Cluster. Deploy the Virtual Control Plane directly on bare metal or VMs. The highest level of isolation — vCluster becomes the cluster.
<div align="center">
<img src="./assets/vcluster-architecture-standalone.png" alt="Standalone Architecture" width="600">
</div>

```yaml
controlPlane:
  standalone:
    enabled: true
    joinNode:
      enabled: true
privateNodes:
  enabled: true
```
</details>
<details>
<summary>⚡ Auto Nodes <sup>v0.28+</sup> — Karpenter-powered dynamic autoscaling</summary>
Automatically provision and deprovision private nodes based on workload demand. Works across public cloud, private cloud, hybrid, and bare metal environments.
<div align="center">
<img src="./assets/vcluster-architecture-auto-nodes.png" alt="Auto Nodes Architecture" width="600">
</div>

```yaml
autoNodes:
  enabled: true
  nodeProvider: <provider>
privateNodes:
  enabled: true
```
</details>

---

## ✨ Key Features

| Feature | Description |
|---------|-------------|
| **🎛️ Isolated Virtual Control Plane** | Each Tenant Cluster gets its own API server, controller manager, and data store — complete Kubernetes API isolation |
| **🔗 Shared Platform Stack** | Leverage the Control Plane Cluster's CNI, CSI, ingress, and other infrastructure — no duplicate platform components |
| **🔒 Strong Tenant Isolation** | Tenants get admin access inside their Tenant Cluster while having minimal permissions on the Control Plane Cluster |
| **🔄 Resource Syncing** | Bidirectional sync of any Kubernetes resource — pods, services, secrets, configmaps, CRDs, and more |
| **💤 Sleep Mode** | Pause inactive Tenant Clusters to save resources. Instant wake when needed |
| **🖥️ Bare Metal & Standalone** | Run with or without a Control Plane Cluster. Purpose-built for AI factories and on-prem GPU fleets |
| **🧩 Integrations** | Native support for cert-manager, external-secrets, KubeVirt, Istio, and metrics-server |
| **📊 High Availability** | Multiple replicas with leader election. Embedded etcd or external databases (PostgreSQL, MySQL, RDS) |

---

## 🌐 The vCluster Platform

vCluster is the foundation of a broader platform for running production Kubernetes and AI infrastructure on your own hardware — from a single rack to 100K-GPU supercomputers.

| Product | What it does |
|---------|--------------|
| **[vCluster](https://www.vcluster.com)** | Tenant Clusters — Virtual Control Planes with API, data, and (optionally) network isolation |
| **[vNode](https://www.vnode.com/)** | Runtime-level tenant isolation. Kernel-enforced boundaries (seccomp, cgroups, namespaces, AppArmor) without VM overhead |
| **[vMetal](https://www.vmetal.ai/)** | Zero-touch bare metal provisioning for GPU fleets. Turns GPU racks into a cloud platform |
| **[Netris](https://www.vcluster.com/solutions/netris-kubernetes-network-automation)** *(integration)* | Hardware-enforced network isolation via programmatic VLANs, VRFs, and ACLs |

Together these deliver the four layers of an AI factory: **Certified Stacks → Tenant Isolation → Tenant Clusters → GPU Infrastructure Operations** — the same pattern used to run production AI on hundreds of GPU clouds and Fortune 500 on-prem platforms.

---

## 🏢 Trusted By

<table>
<tr>
<td align="center"><a href="https://www.vcluster.com/case-studies/atlan"><strong>Atlan</strong></a><br/>100 → 1 clusters</td>
<td align="center"><a href="https://www.vcluster.com/case-studies/aussie-broadband"><strong>Aussie Broadband</strong></a><br/>99% faster provisioning</td>
<td align="center"><a href="https://www.vcluster.com/case-studies/coreweave"><strong>CoreWeave</strong></a><br/>GPU cloud at scale</td>
</tr>
<tr>
<td align="center"><a href="https://www.vcluster.com/case-studies/lintasarta"><strong>Lintasarta</strong></a><br/>170+ Tenant Clusters in prod</td>
<td align="center"><a href="https://www.vcluster.com/case-studies/fortune-500-insurance-company"><strong>Fortune 500 Insurance</strong></a><br/>70% reduction in Kubernetes cost</td>
<td align="center"><a href="https://www.vcluster.com/case-studies/scanmetrix"><strong>Scanmetrix</strong></a><br/>99% faster deployments</td>
</tr>
<tr>
<td align="center"><a href="https://www.vcluster.com/case-studies/deloitte"><strong>Deloitte</strong></a><br/>Enterprise K8s platform</td>
<td align="center"><a href="https://www.vcluster.com/case-studies/ada-cx"><strong>Ada</strong></a><br/>10x Developer Productivity</td>
<td align="center"><a href="https://www.vcluster.com/case-studies/trade-connectors"><strong>Trade Connectors</strong></a><br/>50% reduction in K8s ops cost</td>
</tr>
</table>

**Also used by:** NVIDIA, ABBYY, Precisely, Shipwire, and many more — with 50+ GPU clouds and Fortune 500s running vCluster in production.

👉 **[View All Case Studies](https://www.vcluster.com/case-studies)**

---

## 📚 Learn More

<details>
<summary><strong>🎤 Conference Talks</strong></summary>

| Event | Speaker | Title | Link |
|-------|---------|-------|------|
| KubeCon NA 2025 (Keynote) | Lukas Gentele | Autoscaling GPU Clusters Anywhere — Hyperscalers, Neoclouds & Baremetal | [Watch](https://www.youtube.com/watch?v=LGOELO-ah30) |
| Platform Engineering Day NA 2025 (Keynote) | Saiyam Pathak | AI-Ready Platforms: Scaling Teams Without Scaling Costs | [Watch](https://www.youtube.com/watch?v=sn5kIBS9Xfg) |
| Rejekts NA 2025 | Hrittik Roy, Saiyam Pathak | Beyond the Default Scheduler: Navigating GPU MultiTenancy in AI Era | [Watch](https://www.youtube.com/watch?v=tROp-nmNYxo) |
| KubeCon EU 2025 | Paco Xu, Saiyam Pathak | A Huge Cluster or Multi-Clusters? Identifying the Bottleneck | [Watch](https://www.youtube.com/watch?v=6l5zCt5QsdY) |
| HashiConf 2025 | Scott McAllister | GPU sharing done right: Secrets, security, and scaling with Vault and vCluster | [Watch](https://www.youtube.com/watch?v=zWx17azSqyU) |
| FOSDEM 2025 | Hrittik Roy, Saiyam Pathak | Accelerating CI Pipelines: Rapid Kubernetes Testing with vCluster | [Watch](https://archive.fosdem.org/2025/schedule/event/fosdem-2025-5569-accelerating-ci-pipelines-rapid-kubernetes-testing-with-vcluster/) |
| KubeCon India 2024 (Keynote) | Saiyam Pathak | From Outage To Observability: Lessons From a Kubernetes Meltdown | [Watch](https://www.youtube.com/watch?v=7JCZ688cWpY) |
| CNCF Book Club 2024 | Marc Boorshtein | Kubernetes - An Enterprise Guide (vCluster) | [Watch](https://www.youtube.com/watch?v=8vwnDlkkuJM) |
| KCD NYC 2024 | Lukas Gentele | Tenant Autonomy & Isolation In Multi-Tenant Kubernetes Clusters | [Watch](https://www.youtube.com/watch?v=AKJVLbXsUmE) |
| KubeCon EU 2023 | Ilia Medvedev, Kostis Kapelonis | How We Securely Scaled Multi-Tenancy with VCluster, Crossplane, and Argo CD | [Watch](https://www.youtube.com/watch?v=hFiHU6W4_z0) |
| KubeCon NA 2022 | Joseph Sandoval, Dan Garfield | How Adobe Planned For Scale With Argo CD, Cluster API, And VCluster | [Watch](https://www.youtube.com/watch?v=p8BluR5WT5w) |
| KubeCon NA 2022 | Whitney Lee, Mauricio Salatino | What a RUSH! Let's Deploy Straight to Production! | [Watch](https://www.youtube.com/watch?v=eJG7uIU9NpM) |
| TGI Kubernetes 2022 | TGI | TGI Kubernetes 188: vCluster | [Watch](https://www.youtube.com/watch?v=EaoxUDGpARE) |
| Mirantis Tech Talks 2022 | Mirantis | Multi-tenancy & Isolation using Virtual Clusters (vCluster) in K8s | [Watch](https://www.youtube.com/watch?v=CoqRXdJbCwY) |
| Solo Webinar 2022 | Rich Burroughs, Fabian Keller | Speed your Istio development environment with vCluster | [Watch](https://www.youtube.com/watch?v=b7OkYjvLf4Y) |
| KubeCon NA 2021 | Lukas Gentele | Beyond Namespaces: Virtual Clusters are the Future of Multi-Tenancy | [Watch](https://www.youtube.com/watch?v=QddWNqchD9I) |

</details>

<details>
<summary><strong>🎬 Community Voice</strong></summary>

| Channel | Speaker | Title | Link |
|---------|---------|-------|------|
| TeKanAid 2024 | TeKanAid | Getting Started with vCluster: Build Your IDP with Backstage, Crossplane, and ArgoCD | [Watch](https://www.youtube.com/watch?v=nIxl2PcEs-0) |
| Rawkode 2021 | David McKay, Lukas Gentele | Hands on Introduction to vCluster | [Watch](https://www.youtube.com/watch?v=IMdMvn2_LeI) |
| Kubesimplify 2021 | Saiyam Pathak, Lukas Gentele | Let's Learn vCluster | [Watch](https://www.youtube.com/watch?v=I4mztvnRCjs) |
| TechWorld with Nana 2021 | Nana | Build your Self-Service Kubernetes Platform with Virtual Clusters | [Watch](https://www.youtube.com/watch?v=tt7hope6zU0) |
| DevOps Toolkit 2021 | Viktor Farcic | How To Create Virtual Kubernetes Clusters | [Watch](https://www.youtube.com/watch?v=JqBjpvp268Y) |

</details>

👉 **[YouTube Channel](https://www.youtube.com/@vcluster)** • **[Blog](https://loft.sh/blog)**

---

## 🤝 Contributing

We welcome contributions! Check out our **[Contributing Guide](https://github.com/loft-sh/vcluster/blob/main/CONTRIBUTING.md)** to get started.

---

## 🔗 Links

| Resource | Link |
|----------|------|
| 📖 Documentation | [vcluster.com/docs](https://www.vcluster.com/docs/vcluster/introduction/what-are-virtual-clusters) |
| 💬 Slack Community | [slack.loft.sh](https://slack.loft.sh/) |
| 🌐 Website | [vcluster.com](https://www.vcluster.com) |
| 🐦 X (Twitter) | [@vcluster](https://x.com/vcluster) |
| 💼 LinkedIn | [vCluster](https://www.linkedin.com/company/vcluster) |
| 💬 Chat with Expert | [Start Chat](https://start-chat.com/slack/Loft-Labs/NnQl1M) |

---

## 📜 License

vCluster is licensed under the **[Apache 2.0 License](http://www.apache.org/licenses/LICENSE-2.0)**.

---

<div align="center">

**© 2026 [Loft Labs](https://loft.sh). All rights reserved.**

Made with ❤️ by the vCluster community.

⭐ **Star us on GitHub** — it helps!

</div>
