<br>
<a href="https://www.vcluster.com"><img src="docs/static/media/vcluster-logo-dark.svg"></a>

### **[Website](https://www.vcluster.com)** • **[Quickstart](https://www.vcluster.com/docs/getting-started/setup)** • **[Documentation](https://www.vcluster.com/docs/what-are-virtual-clusters)** • **[Blog](https://loft.sh/blog)** • **[Twitter](https://twitter.com/loft_sh)** • **[Slack](https://slack.loft.sh/)**

![Latest Release](https://img.shields.io/github/v/release/loft-sh/vcluster?style=for-the-badge&label=Latest%20Release&color=%23007ec6)
![License: Apache-2.0](https://img.shields.io/github/license/loft-sh/vcluster?style=for-the-badge&color=%23007ec6)

[![Join us on Slack!](docs/static/media/slack.svg)](https://slack.loft.sh/)

### vcluster - Virtual Clusters For Kubernetes
Create fully functional virtual Kubernetes clusters - Each vcluster runs inside a namespace of the underlying k8s cluster. It's cheaper than creating separate full-blown clusters and it offers better multi-tenancy and isolation than regular namespaces.
- **Certified Kubernetes Distribution** - vcluster itself is a [certified Kubernetes distribution](https://www.cncf.io/certification/software-conformance/) and is 100% Kubernetes API conform. Everything that works in a regular Kubernetes cluster works in vcluster
- **Lightweight & Low-Overhead** - Based on k3s, bundled in a single pod and with super-low resource consumption
- **No Performance Degradation** - Pods are scheduled in the underlying host cluster, so they get no performance hit at all while running
- **Reduced Overhead On Host Cluster** - Split up large multi-tenant clusters into smaller vcluster to reduce complexity and increase scalability. Since most vcluster api requests and objects will not reach the host cluster at all, vcluster can greatly decrease pressure on the underlying Kubernetes cluster
- **Easy Provisioning** - Create via vcluster CLI, helm, kubectl, Argo or any of your favorite tools (it is basically just a StatefulSet)
- **No Admin Privileges Required** - If you can deploy a web app to a Kubernetes namespace, you will be able to deploy a vcluster as well
- **Single Namespace Encapsulation** - Every vcluster and all of its workloads are inside a single namespace of the underlying host cluster
- **Easy Cleanup** - Delete the host namespace and the vcluster plus all of its workloads will be gone immediately
- **Flexible & Versatile** - vcluster supports different storage backends (such as sqlite, mysql, postgresql & etcd), customizable sync behaviour, vcluster within vcluster setups, rewriting of kubelet metrics and has many more additional configuration options to fit a multitude of use cases

Learn more on [www.vcluster.com](https://vcluster.com).

<br>

## Architecture 
[![vcluster Intro](docs/static/media/diagrams/vcluster-architecture.svg)](https://www.vcluster.com)

![vcluster Compatibility](docs/static/media/cluster-compatibility.png)


Learn more in the [documentation](https://vcluster.com/docs/what-are-virtual-clusters).

<br>

<p align="center">
⭐️ <strong>Do you like vcluster? Support the project with a star</strong> ⭐️
</p>

<br>

## Quick Start
To learn more about vcluster, [**open the full getting started guide**](https://www.vcluster.com/docs/getting-started/setup).

### 1. Download vcluster CLI
Use one of the following commands to download the vcluster CLI binary from GitHub:

<details>
<summary>Mac (Intel/AMD)</summary>

```bash
curl -s -L "https://github.com/loft-sh/vcluster/releases/latest" | sed -nE 's!.*"([^"]*vcluster-darwin-amd64)".*!https://github.com\1!p' | xargs -n 1 curl -L -o vcluster && chmod +x vcluster;
sudo mv vcluster /usr/local/bin;
```

</details>

<details>
<summary>Mac (Silicon/ARM)</summary>

```bash
curl -s -L "https://github.com/loft-sh/vcluster/releases/latest" | sed -nE 's!.*"([^"]*vcluster-darwin-arm64)".*!https://github.com\1!p' | xargs -n 1 curl -L -o vcluster && chmod +x vcluster;
sudo mv vcluster /usr/local/bin;
```

</details>

<details>
<summary>Linux (AMD)</summary>

```bash
curl -s -L "https://github.com/loft-sh/vcluster/releases/latest" | sed -nE 's!.*"([^"]*vcluster-linux-amd64)".*!https://github.com\1!p' | xargs -n 1 curl -L -o vcluster && chmod +x vcluster;
sudo mv vcluster /usr/local/bin;
```

</details>

<details>
<summary>Linux (ARM)</summary>

```bash
curl -s -L "https://github.com/loft-sh/vcluster/releases/latest" | sed -nE 's!.*"([^"]*vcluster-linux-arm64)".*!https://github.com\1!p' | xargs -n 1 curl -L -o vcluster && chmod +x vcluster;
sudo mv vcluster /usr/local/bin;
```

</details>

<details>
<summary>Windows (Powershell)</summary>

```bash
md -Force "$Env:APPDATA\vcluster"; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.SecurityProtocolType]'Tls,Tls11,Tls12';
Invoke-WebRequest -UseBasicParsing ((Invoke-WebRequest -URI "https://github.com/loft-sh/vcluster/releases/latest" -UseBasicParsing).Content -replace "(?ms).*`"([^`"]*vcluster-windows-amd64.exe)`".*","https://github.com/`$1") -o $Env:APPDATA\vcluster\vcluster.exe;
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
# By default vcluster will connect via port-forwarding
vcluster create vcluster-1 -n host-namespace-1 --connect

# OR: Use --expose to create a vcluster with an externally accessible LoadBalancer
vcluster create vcluster-1 -n host-namespace-1 --connect --expose 
```

Take a look at the [vcluster docs](https://www.vcluster.com/docs/getting-started/deployment) to see how to deploy a vcluster using Helm or Kubectl instead.

### 3. Use the vcluster

Run in a separate terminal:
```bash
export KUBECONFIG=./kubeconfig.yaml

# Run any kubectl, helm, etc. command in your vcluster
kubectl get namespace
kubectl get pods -n kube-system
kubectl create namespace demo-nginx
kubectl create deployment nginx-deployment -n demo-nginx --image=nginx
kubectl get pods -n demo-nginx
```

### 4. Cleanup
```bash
vcluster delete vcluster-1 -n host-namespace-1
```

Alternatively, you could also delete the host-namespace using kubectl.

## Contributing

Thank you for your interest in contributing! Please refer to
[CONTRIBUTING.md](https://github.com/loft-sh/vcluster/blob/main/CONTRIBUTING.md) for guidance.

<br>

---

This project is open-source and licensed under Apache 2.0, so you can use it in any private or commercial projects.
