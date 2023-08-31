# How to use vcluster's generic-crd-sync feature to sync certain Istio resources

**Note**: The configuration provided here is an example. Please review and update it to match your environment and use case.

Prerequisite for installation are the Istio CRDs installed in the host cluster where vcluster will be installed.

To enable this config you need to pass the ./config.yaml file as additional source of Helm values during the installation.
For testing purposes you can refer to the config.yaml for the Istio resources like so: `https://raw.githubusercontent.com/loft-sh/vcluster/main/generic-sync-examples/istio/config.yaml`.  
Example installation command:

```bash
vcluster create -n vcluster vcluster -f https://raw.githubusercontent.com/loft-sh/vcluster/main/generic-sync-examples/istio/config.yaml
```

Once configured, vcluster will copy the CRDs from the host cluster into vcluster based on the configuration. Currently the configuration includes sync mappings for the VirtualService, DestinationRule and Gateway CRDs.

## Supported features

Provided configuration supports only limited number of Istio features. The VirtualServices and DestinationRules are recognized only for the purpose of directing traffic from the Gateway into a sevice. The intra mesh communication is not currently unsupported. The workloads from the vcluster are connected to the host mesh, but they can not resolve VirtualServices from the vcluster correctly, and as such it will usually fallback to usual Kubernetes Service based communication.

## Alternative configuration

If you would like to allow vcluster users to use Gateways from the host cluster, instead of letting them create Gateways, then you can use the `host-only-gateways.yaml` configuration file.

In this mode the Gateway CRD is not synced to the vcluster and the `.spec.gateways` values of the VirtualServices are not overwritten when syncing the resource to the host cluster. This will allow vcluster users to use a Gateway that exists in the host cluster, but they would have to know the name of the Gateway resource.
You can also set a specific Gateway to be used for all VirtualServices synced to the host by updating the configuration based on comments inside the `host-only-gateways.yaml` file (find a comment with instructions in the VirtualService mapping).

## Controlling sidecar injection with labels

By default, vcluster modifies pod labels before scheduling a pod in the host cluster. If you want to allow the vcluster users to use the `sidecar.istio.io/inject` label to control sidecar injection for their pod then you need to (or uncomment in the config.yaml file) these helm values:

```yaml
syncer:
  extraArgs:
    - "--sync-labels=sidecar.istio.io/inject"
```
