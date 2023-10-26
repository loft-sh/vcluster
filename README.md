<p align="center">
  <a href="https://www.vcluster.com">
    <img src="docs/static/media/vcluster_horizontal_black.svg" width="500">
  </a>
</p>

<div align="center">
  <strong>Create fully functional virtual Kubernetes clusters.</strong>
</div>

---

## Table of Contents

- [Introduction](#introduction)
- [Why Virtual Kubernetes Clusters?](#why-virtual-kubernetes-clusters)
- [Features](#features)
- [Quick Start](#quick-start)
- [Architecture](#architecture)
- [Contributing](#contributing)
- [License](#license)

---

## Introduction

[vcluster](https://www.vcluster.com) allows you to create fully functional virtual Kubernetes clusters. Each vcluster runs inside a namespace of the underlying Kubernetes cluster. It's a cost-effective solution that offers better multi-tenancy and isolation than regular namespaces.

**[Website](https://www.vcluster.com)** • **[Quickstart](https://www.vcluster.com/docs/getting-started/setup)** • **[Documentation](https://www.vcluster.com/docs/what-are-virtual-clusters)** • **[Blog](https://loft.sh/blog)** • **[Twitter](https://twitter.com/loft_sh)** • **[Slack](https://slack.loft.sh/)**

![Latest Release](https://img.shields.io/github/v/release/loft-sh/vcluster?style=for-the-badge&label=Latest%20Release&color=%23007ec6)
![License: Apache-2.0](https://img.shields.io/github/license/loft-sh/vcluster?style=for-the-badge&color=%23007ec6)

[![Join us on Slack!](docs/static/media/slack.svg)](https://slack.loft.sh/) [![Open in DevPod!](https://devpod.sh/assets/open-in-devpod.svg)](https://devpod.sh/open#https://github.com/loft-sh/vcluster)

---

## Why Virtual Kubernetes Clusters?

- **Cluster Scoped Resources**: vcluster allows users to use CRDs, namespaces, cluster roles, etc.
- **Ease of Use**: Create virtual clusters in seconds via a single command or [cluster-api](https://github.com/loft-sh/cluster-api-provider-vcluster).
- **Cost Efficient**: More cost-effective and efficient than creating separate full-blown clusters.
- **Lightweight**: Built upon the ultra-fast k3s distribution with minimal overhead.
- **Strict Isolation**: Each vcluster has a separate Kubernetes control plane and access point.
- **Cluster Wide Permissions**: Install apps requiring cluster-wide permissions while being limited to one namespace.
- **Great for Testing**: Test different Kubernetes versions inside a single host cluster.

Learn more on [www.vcluster.com](https://vcluster.com).

---

## Features

- **Certified Kubernetes Distribution**: vcluster is a [certified Kubernetes distribution](https://www.cncf.io/certification/software-conformance/), 100% Kubernetes API conform.
- **Lightweight & Low-Overhead**: Based on k3s, with super-low resource consumption.
- **No Performance Degradation**: Pods are scheduled in the underlying host cluster.
- **Reduced Overhead On Host Cluster**: Split up large multi-tenant clusters into smaller vclusters.
- **Easy Provisioning**: Create via vcluster CLI, helm, kubectl, cluster-api, or any favorite tools.
- **No Admin Privileges Required**: Deploy a vcluster with web app privileges.
- **Single Namespace Encapsulation**: All vcluster workloads are inside a single namespace.
- **Easy Cleanup**: Delete the host namespace and the vcluster plus all of its workloads will be gone immediately.
- **Flexible & Versatile**: vcluster supports different storage backends, plugins, and many more configuration options.

Learn more in the [documentation](https://vcluster.com/docs/what-are-virtual-clusters).

---

## Quick Start (~ 1 minute)

To learn more about vcluster, [**open the full getting started guide**](https://www.vcluster.com/docs/getting-started/setup).

### 1. Download vcluster CLI

Use one of the following commands to download the vcluster CLI binary from GitHub:

<details>
<summary>Mac (Intel/AMD)</summary>

```bash
curl -L -o vcluster "https://github.com/loft-sh/vcluster/releases/latest/download/vcluster-darwin-amd64" && sudo install -c -m 0755 vcluster /usr/local/bin
```

</details>

<details>
<summary>Linux (AMD)</summary>

```bash
curl -L -o vcluster "https://github.com/loft-sh/vcluster/releases/latest/download/vcluster-linux-amd64" && sudo install -c -m 0755 vcluster /usr/local/bin
```

</details>

<details>
<summary>Linux (ARM)</summary>

```bash
curl -L -o vcluster "https://github.com/loft-sh/vcluster/releases/latest/download/vcluster-linux-arm64" && sudo install -c -m 0755 vcluster /usr/local/bin
```

</details>

<details>
<summary>Windows (Powershell)</summary>

```bash
md -Force "$Env:APPDATA\vcluster"; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.SecurityProtocolType]'Tls,Tls11,Tls12';
Invoke-WebRequest -URI "https://github.com/loft-sh/vcluster/releases/latest/download/vcluster-windows-amd64.exe" -o $Env:APPDATA\vcluster\vcluster.exe;
$env:Path += ";" + $Env:APPDATA + "\vcluster";
[Environment]::SetEnvironmentVariable("Path", $env:Path, [System.EnvironmentVariableTarget]::User);
```

> If you get the error that Windows cannot find vcluster after installing it, you will need to restart your computer, so that the changes to the `PATH` variable will be applied.

</details>

<br>

Alternatively, you can download the binary for your platform from the [GitHub Releases](https://github.com/loft-sh/vcluster/releases) page and add this binary to your PATH.

<br>

### 2. Create a vcluster

```vash
vcluster create my-vcluster

# OR: Use --expose to create a vcluster with an externally accessible LoadBalancer
vcluster create my-vcluster --expose

# OR: Use --isolate to create an isolated environment for the vcluster workloads
vcluster create my-vcluster --isolate
```

Take a look at the [vcluster docs](https://www.vcluster.com/docs/getting-started/deployment) to see how to deploy a vcluster using Helm or Kubectl instead.

### 3. Use the vcluster

Run in a terminal:

```bash
# Run any kubectl, helm, etc. command in your vcluster
kubectl get namespace
kubectl get pods -n kube-system
kubectl create namespace demo-nginx
kubectl create deployment nginx-deployment -n demo-nginx --image=nginx
kubectl get pods -n demo-nginx
```

### 4. Cleanup

```bash
vcluster delete my-vcluster
```

Alternatively, you could also delete the host-namespace using kubectl.

## Architecture

[![vcluster Intro](docs/static/media/diagrams/vcluster-architecture.svg)](https://www.vcluster.com)

## Contributing

Thank you for your interest in contributing! Please refer to
[CONTRIBUTING.md](https://github.com/loft-sh/vcluster/blob/main/CONTRIBUTING.md) for guidance.

<br>

---
## License
This project is open-source and licensed under Apache 2.0, so you can use it in any private or commercial projects.
