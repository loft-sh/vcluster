# Generic Sync Examples

## Prerequisites

* Whichever example you are aiming to get working with, the basic installation of the same has to be setup on the host cluster.

## Creating vclusters that use the above example configurations

Simply create the vcluster along with an above raw configuration file as an argument:

```bash
vcluster create vcluster -f https://raw.githubusercontent.com/loft-sh/vcluster/main/generic-sync-examples/knative/config.yaml
```
