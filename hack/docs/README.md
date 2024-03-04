# Generated Documentation

This folder contains scripts that are used to generate a JSON schema based on the vcluster values.yaml, and a Documentation partials for browsing the schema via the hosted documentation (/docs)

## Generate Schema

To update `vcluster-schema.json` after making changes, run the following command:
```shell
> go run ./hack/docs/schema/main.go
```

To update `docs/pages/configuration/_partials` after making changes, run the following command:
```shell
> go run ./hack/docs/partials/main.go
```
