<div align="center">
  <a href="https://www.vcluster.com" target="_blank">


<picture>
      <!-- For Dark Mode -->
      <source media="(prefers-color-scheme: dark)" srcset="docs/static/media/vcluster_horizontal_orange_white.svg">
      <!-- For Light Mode -->
      <source media="(prefers-color-scheme: light)" srcset="docs/static/media/vcluster_horizontal_orange_black.svg">
      <!-- Fallback -->
      <img alt="vCluster Logo" src="docs/static/media/vcluster_horizontal_orange_white.svg" width="600">
</picture>	  

  </a>
</div>

<div align="center">

### **[Website](https://www.vcluster.com)** • **[Quickstart](https://www.vcluster.com/docs/get-started/)** • **[Documentation](https://www.vcluster.com/docs/what-are-virtual-clusters)** • **[Blog](https://loft.sh/blog)** • **[Twitter](https://x.com/vcluster)** • **[Slack](https://slack.loft.sh/)**

</div>



---

### 🚀 Get Started Quickly!

Deploy your first virtual cluster with minimal effort:

#### Step 1: Install vCluster CLI
```bash
brew install loft-sh/tap/vcluster
```
#### Prerequisite: Set Up a Kubernetes Cluster
Before creating a virtual cluster, ensure you have access to a running Kubernetes cluster. 

#### Step 2: Create a Virtual Cluster in the `team-x` namespace

```bash
vcluster create my-vcluster --namespace team-x
```
#### Step 3: Connect to the Virtual Cluster

```bash
vcluster connect my-vcluster --namespace team-x
```

![vCluster gif](./docs/static/media/vcluster-github-gif-1280.gif)

For detailed steps, visit our [Quickstart Documentation](https://www.vcluster.com/docs/get-started).

---

### 🌟Why vCluster?

<details>
<summary><strong>Robust Security and Isolation</strong></summary>

- **Granular Permissions**:  
  vCluster users operate with minimized permissions in the host cluster, significantly reducing the risk of privileged access misuse. Within their vCluster, users have admin-level control, enabling them to manage CRDs, RBAC, and other security policies independently.

- **Isolated Control Plane**:  
  Each vCluster comes with its own dedicated API server and control plane, creating a strong isolation boundary.

- **Customizable Security Policies**:  
  Tenants can implement additional vCluster-specific governance, including OPA policies, network policies, resource quotas, limit ranges, and admission control, in addition to the existing policies and security measures in the underlying physical host cluster.

- **Enhanced Data Protection**:  
  With options for separate backing stores, including embedded SQLite, etcd, or external databases, virtual clusters allow for isolated data management, reducing the risk of data leakage between tenants.

</details>

<details>
<summary><strong>Access for Tenants</strong></summary>

- **Full Admin Access per Tenant**:  
  Tenants can freely deploy CRDs, create namespaces, taint, and label nodes, and manage cluster-scoped resources typically restricted in standard Kubernetes namespaces.

- **Isolated yet Integrated Networking**:  
  While ensuring automatic isolation (for example, pods in different virtual clusters cannot communicate by default), vCluster allows for configurable network policies and service sharing, supporting both separation and sharing as needed.

- **Node Management**:  
  Assign static nodes to specific virtual clusters or share node pools among multiple virtual clusters, providing flexibility in resource allocation.

</details>

<details>
<summary><strong>Cost-Effectiveness and Reduced Overhead</strong></summary>

- **Lightweight Infrastructure**:  
  Virtual clusters are significantly more lightweight than physical clusters, able to spin up in seconds, which contrasts sharply with the lengthy provisioning times often seen in environments like EKS (~45 minutes).

- **Resource Efficiency**:  
  By sharing the underlying host cluster's resources, virtual clusters minimize the need for additional physical infrastructure, reducing costs and environmental impact.

- **Simplified Management**:  
  The vCluster control plane, running inside a single pod, along with optional integrated CoreDNS, minimizes the operational overhead, making virtual clusters especially suitable for large-scale deployments and multi-tenancy scenarios.

</details>

<details>
<summary><strong>Enhanced Flexibility and Compatibility</strong></summary>

- **Diverse Kubernetes Environments**:  
  vCluster supports different Kubernetes versions and distributions (including K8s and K3s), allowing version skews. This makes it possible to tailor each virtual cluster to specific requirements without impacting others.

- **Adaptable Backing Stores**:  
  Choose from a range of data stores, from lightweight (SQLite) to enterprise-grade options (embedded etcd, external data stores like Global RDS), catering to various scalability and durability needs.

- **Runs Anywhere**:  
  Virtual clusters can run on EKS, GKE, AKS, OpenShift, RKE, K3s, cloud, edge, and on-prem. As long as it's a K8s cluster, you can run a virtual cluster on top of it.

</details>

<details>
<summary><strong>Improved Scalability</strong></summary>

- **Reduced API Server Load**:  
  Virtual clusters, each with their own dedicated API server, significantly reduce the operational load on the host cluster's Kubernetes API server by isolating and handling requests internally.

- **Conflict-Free CRD Management**:  
  Independent management of CRDs within each virtual cluster eliminates the potential for CRD conflicts and version discrepancies, ensuring smoother operations and easier scaling as the user base expands.

</details>


---

### 📚 Expand Your Knowledge
#### Conference Talks
| Event             | Speaker         | Title                                           | YouTube Link                          |
|--------------------|----------------|-------------------------------------------------|---------------------------------------|
| CNCF Book Club 2024| Marc Boorshtein| Kubernetes - An Enterprise Guide (vCluster) | [Watch Here](https://www.youtube.com/watch?v=8vwnDlkkuJM) |
| KCD NYC 2024   | Lukas Gentele    | Tenant Autonomy & Isolation In Multi-Tenant Kubernetes Clusters | [Watch Here](https://www.youtube.com/watch?v=AKJVLbXsUmE&t=758s)| 
| KubeCon Eu 2023   | Ilia Medvedev & Kostis Kapelonis | How We Securely Scaled Multi-Tenancy with VCluster, Crossplane, and Argo CD | [Watch Here](https://www.youtube.com/watch?v=hFiHU6W4_z0) |
|Solo Webinar 2022 | Rich and Fabian | Speed your Istio development environment with vCluster | [Watch Here](https://www.youtube.com/watch?v=b7OkYjvLf4Y)|
|Mirantis Tech Talks 2022| Mirantis |Multi-tenancy & Isolation using Virtual Clusters (vCluster) in K8s| [Watch Here](https://www.youtube.com/watch?v=CoqRXdJbCwY) |
| TGI 2022 | TGI | TGI Kubernetes 188: vCluster | [Watch Here](https://www.youtube.com/watch?v=EaoxUDGpARE)|
| KubeCon NA 2022 | Whitney Lee & Mauricio Salatino | What a RUSH! Let's Deploy Straight to Production! | [Watch Here](https://www.youtube.com/watch?v=eJG7uIU9NpM) | 
| KubeCon NA 2022   | Joseph Sandoval & Dan Garfield       | How Adobe Planned For Scale With Argo CD, Cluster API, And VCluster| [Watch Here](https://www.youtube.com/watch?v=p8BluR5WT5w)| 
| KubeCon NA 2021    | Lukas Gentele  | Beyond Namespaces: Virtual Clusters are the Future of Multi-Tenancy | [Watch Here](https://www.youtube.com/watch?v=QddWNqchD9I) |

#### Community Voice
| Youtube Channel             | Speaker         | Title                                           | YouTube Link                          |
|--------------------|----------------|-------------------------------------------------|---------------------------------------|
|TeKanAid 2024|TeKanAid|Getting Started with vCluster: Build Your IDP with Backstage, Crossplane, and ArgoCD | [Watch Here](https://www.youtube.com/watch?v=nIxl2PcEs-0)|
| DevOps Toolkit 2021 | Viktor Farcic |  How To Create Virtual Kubernetes Clusters | [Watch Here](https://www.youtube.com/watch?v=JqBjpvp268Y&t=82s) |
| TechWorld with Nana 2021 | Nana | Build your Self-Service Kubernetes Platform with Virtual Clusters  | [Watch Here](https://www.youtube.com/watch?v=tt7hope6zU0)
| Kubesimplify 2021 | Saiyam Pathak and Lukas Gentele | Let's Learn vCluster| [Watch Here](https://www.youtube.com/watch?v=I4mztvnRCjs&t=1s) |
| Rawkode 2021 | David and Lukas | Hands on Introduction to vCluster | [Watch Here](https://www.youtube.com/watch?v=IMdMvn2_LeI) | 

Explore more vCluster tips on our [Youtube Channel](https://www.youtube.com/@vcluster) and [Blogs](https://loft.sh/blog).

---

### 💻 Contribute to vCluster
We love contributions! Check out our [Contributing Guide](https://github.com/loft-sh/vcluster/blob/main/CONTRIBUTING.md).

For quick local development, use [![Open in DevPod!](https://devpod.sh/assets/open-in-devpod.svg)](https://devpod.sh/open#https://github.com/loft-sh/vcluster)

---

### 🔗 Useful Links
- [Documentation](https://www.vcluster.com/docs/what-are-virtual-clusters)
- [Slack Community](https://slack.loft.sh/)
- [vCluster Website](https://www.vcluster.com)

---
### Adopters

We're glad to see vCluster being adopted by organizations around the world! Below are just a few examples of how vCluster is being used in production environments:
- **[Atlan](https://www.vcluster.com/case-studies/atlan)**: Atlan Reduced Their Infrastructure From 100 Kubernetes Clusters To 1 Using vCluster.
- **[Adobe](https://www.youtube.com/watch?v=p8BluR5WT5w)**: Enhancing development environments with virtual clusters.
- **[Aussie Broadband](https://www.vcluster.com/case-studies/aussie-broadband)**:  Aussie Broadband Achieved 99% Faster Cluster Provisioning with vCluster.
- **[Codefresh](https://www.loft.sh/blog/how-codefresh-uses-vcluster-to-provide-hosted-argo-cd)**: Codefresh uses vCluster to provide hosted ArgoCD.
- **[Coreweave](https://www.coreweave.com/blog/coreweave-and-loft-labs-leverage-vcluster-in-kubernetes-at-scale)**: CoreWeave and Loft Labs Leverage vCluster to Run Virtual Clusters in Kubernetes at Scale.
- **[Scanmetrics](https://www.vcluster.com/case-studies/scanmetrix)**: Scanmetrix Achieved 99% Faster Customer Deployments with vCluster
- **[Trade Connectors](https://www.vcluster.com/case-studies/trade-connectors)**: Trade Connectors Optimized Kubernetes Cost with Multi-Tenancy from vCluster.
- **ABBYY**
- **Aera**
- **Lintasarta**
- **Precisely**
- **Shipwire**

Are you using vCluster? We'd love to hear your story! Please [open a pull request](https://github.com/loft-sh/vcluster/pulls) to add your name here, or [contact us](mailto:contact@loft.sh).

---

### 📜 License
vCluster is licensed under the [Apache 2.0 License](http://www.apache.org/licenses/LICENSE-2.0).

### Copyright

© 2025 [Loft Labs](https://loft.sh). All rights reserved.
This project and its maintainers are committed to fostering a welcoming, inclusive, and respectful community.

