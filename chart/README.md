
# vcluster

## **[GitHub](https://github.com/loft-sh/vcluster)** • **[Website](https://www.vcluster.com)** • **[Quickstart](https://www.vcluster.com/docs/getting-started/setup)** • **[Documentation](https://www.vcluster.com/docs/what-are-virtual-clusters)** • **[Blog](https://loft.sh/blog)** • **[Twitter](https://twitter.com/loft_sh)** • **[Slack](https://slack.loft.sh/)**

Create fully functional virtual Kubernetes clusters - Each vcluster runs inside a namespace of the underlying k8s cluster. It's cheaper than creating separate full-blown clusters and it offers better multi-tenancy and isolation than regular namespaces.

## Prerequisites

- Kubernetes 1.18+
- Helm 3+

## Get Helm Repository Info

```bash
helm repo add loft-sh https://charts.loft.sh
helm repo update
```

See [helm repo](https://helm.sh/docs/helm/helm_repo/) for command documentation.

## Install Helm Chart

```bash
helm upgrade [RELEASE_NAME] loft-sh/vcluster -n [RELEASE_NAMESPACE] --create-namespace --install
```

See [vcluster docs](https://vcluster.com/docs) for configuration options.

See [helm upgrade](https://helm.sh/docs/helm/helm_upgrade/) for command documentation.

## Connect to the vcluster

In order to connect to the installed vcluster, please install [vcluster cli](https://www.vcluster.com/docs/getting-started/setup) and run:

```bash
vcluster connect [RELEASE_NAME] -n [RELEASE_NAMESPACE]
```

## Uninstall Helm Chart

```bash
helm uninstall [RELEASE_NAME]
```

This removes all the Kubernetes components associated with the chart and deletes the release.

See [helm uninstall](https://helm.sh/docs/helm/helm_uninstall/) for command documentation.

### Why Virtual Kubernetes Clusters?

- **Cluster Scoped Resources**: much more powerful than simple namespaces (virtual clusters allow users to use CRDs, namespaces, cluster roles etc.)
- **Ease of Use**: usable in any Kubernetes cluster and created in seconds either via a single command or [cluster-api](https://github.com/loft-sh/cluster-api-provider-vcluster)
- **Cost Efficient**: much cheaper and efficient than "real" clusters (single pod and shared resources just like for namespaces)
- **Lightweight**: built upon the ultra-fast k3s distribution with minimal overhead per virtual cluster (other distributions work as well)
- **Strict isolation**: complete separate Kubernetes control plane and access point for each vcluster while still being able to share certain services of the underlying host cluster
- **Cluster Wide Permissions**: allow users to install apps which require cluster-wide permissions while being limited to actually just one namespace within the host cluster
- **Great for Testing**: allow you to test different Kubernetes versions inside a single host cluster which may have a different version than the virtual clusters

Learn more on [www.vcluster.com](https://vcluster.com).

![vcluster Intro](https://github.com/loft-sh/vcluster/raw/main/docs/static/media/vcluster-comparison.png)

Learn more in the [documentation](https://vcluster.com/docs/what-are-virtual-clusters).
