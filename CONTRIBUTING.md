# Contributing to vcluster

Thank you for contributing to vcluster! Here you can find common questions around developing vcluster. [See the docs](https://www.vcluster.com/docs/architecture/basics) for architecture details.

## Table of Contents

- [How can I get involved?](#how-can-i-get-involved)
- [Developing the Syncer Container using devspace](#developing-the-syncer-container-using-devspace)
  - [Setup vcluster for Development](#setup-vcluster-for-development)
- [Debug vcluster with Delve](#debug-vcluster-with-delve)
  - [Running vcluster Tests](#running-vcluster-tests)
- [Build vcluster CLI tool](#build-vcluster-cli-tool)
- [Developing the hostpath-mapper component instead of syncer](#developing-the-hostpath-mapper-component-instead-of-syncer)
  - [License](#license)
  - [Copyright notice](#copyright-notice)

## How can I get involved?

There are a number of areas where contributions can be accepted:

- Write Golang code for the vcluster syncer, proxy or other components
- Write examples
- Review pull requests
- Test out new features or work-in-progress
- Get involved in design reviews and technical proof-of-concepts (PoCs)
- Help release and package vcluster including the helm chart, compose files, `kubectl` YAML, marketplaces and stores
- Manage, triage and research Issues and Pull Requests
- Engage with the growing community by providing technical support on GitHub
- Create docs, guides and write blogs

This is just a short list of ideas, if you have other ideas for contributing please make a suggestion.

## Developing the Syncer Container using devspace

See [docs](https://www.vcluster.com/docs/architecture/basics#vcluster-syncer) for explanation of the syncer.
Devspace will set up a vcluster and sync your code changes from the repository to a running container.

### Setup vcluster for Development

We recommend to develop vcluster directly in a Kubernetes cluster as it makes feedback a lot quicker. For the quick setup, you'll need to install [devspace](https://github.com/loft-sh/devspace#1-install-devspace), docker, kubectl, helm and make sure you have a local Kubernetes cluster (such as Docker Desktop, minikube, KinD or similar) installed.

Fork and clone the repo:

- Click Fork button (top right) to establish a cloud-based fork.
- git clone your-fork-url

After adjusting the variables and you can run:

```
devspace dev
```

Which should produce an output similar to:

```
[info]   Using namespace 'vcluster'
[info]   Using kube context 'docker-desktop'
[done] √ Created image pull secret vcluster/devspace-auth-ghcr-io
[info]   Building image 'ghcr.io/loft-sh/loft-enterprise/dev-vcluster:szFLbkA' with engine 'docker'
Sending build context to Docker daemon  52.71MB
Step 1/14 : FROM golang:1.15 as builder
 ---> 53b7b7a65524
...
Step 14/14 : ENTRYPOINT ["go", "run", "-mod", "vendor", "cmd/vcluster/main.go"]
 ---> Running in 7156169fe7d7
 ---> b14eacaa5e29
Successfully built b14eacaa5e29
Successfully tagged ghcr.io/loft-sh/loft-enterprise/dev-vcluster:szFLbkA
[info]   Skip image push for ghcr.io/loft-sh/loft-enterprise/dev-vcluster
[done] √ Done processing image 'ghcr.io/loft-sh/loft-enterprise/dev-vcluster'
[info]   Execute 'helm upgrade vcluster ./chart --namespace vcluster --values /var/folders/bc/qxzrp6f93zncnj1xyz25kyp80000gn/T/079539791 --install --kube-context docker-desktop'
[info]   Execute 'helm list --namespace vcluster --output json --kube-context docker-desktop'
[done] √ Deployed helm chart (Release revision: 1)
[done] √ Successfully deployed vcluster with helm

#########################################################
[info]   DevSpace UI available at: http://localhost:8090
#########################################################

[0:sync] Waiting for pods...
[0:sync] Starting sync...
[0:sync] Sync started on /Users/fabiankramm/Programmieren/go-workspace/src/github.com/loft-sh/vcluster <-> . (Pod: vcluster/vcluster-0)
[0:sync] Waiting for initial sync to complete
[info]   Opening shell to pod:container vcluster-0:syncer
root@vcluster-0:/vcluster#
```

Then you can start vcluster with

```
go run -mod vendor cmd/vcluster/main.go start
```

Now if you change a file locally, DevSpace will automatically sync the file into the container and you just have to rerun `go run -mod vendor cmd/vcluster/main.go start` within the terminal to apply the changes.

You can access the vcluster by running `devspace enter` in a separate terminal:

```
devspace enter -n vcluster

? Which pod do you want to open the terminal for? vcluster-0:syncer
[info]   Opening shell to pod:container vcluster-0:syncer
root@vcluster-0:/vcluster# kubectl get ns
NAME              STATUS   AGE
default           Active   2m18s
kube-system       Active   2m18s
kube-public       Active   2m18s
kube-node-lease   Active   2m18s
root@vcluster-0:/vcluster#
```

To access the virtual cluster, you can use the `vcluster connect` command locally as with any other virtual cluster.

### Debug vcluster with Delve

If you wish to run vcluster in the debug mode with delve, run `devspace dev -n vcluster` and wait until you see the command prompt (`root@vcluster-0:/vcluster#`).
Run `dlv debug ./cmd/vcluster/main.go --listen=0.0.0.0:2345 --api-version=2 --output /tmp/__debug_bin --headless --build-flags="-mod=vendor" -- start`
Wait until the `API server listening at: [::]:2345` message appears.
Start the `Debug vcluster (localhost:2346)` configuration in VSCode to connect your debugger session.
If you are not using VSCode, configure your IDE to connect to `localhost:2346` for the "remote" delve debugging.
**Note:** vcluster won't start until you connect with the debugger.
**Note:** vcluster will be stopped once you detach your debugger session.

### Running vcluster Tests

You can run the unit test suite with `./hack/test.sh` which should execute all the vcluster tests.

The e2e test suite can be run from the e2e folder(`cd e2e`) with this command - `go test -v -ginkgo.v`.
Alternatively, if you [install ginkgo binary](https://github.com/onsi/ginkgo#global-installation), you can run it with `ginkgo -v`.

For running conformance tests, please take a look at [conformance tests](https://github.com/loft-sh/tree/vcluster/main/conformance/v1.21)

## Build vcluster CLI tool

Build:

```
go generate ./... && go build -o vcluster cmd/vclusterctl/main.go # build vcluster cli
```

Run:

```
./vcluster create v1 # create vcluster
```

## Developing the hostpath-mapper component instead of syncer

In case you need to develop the hostpath-mapper daemonset instead of the syncer, you can use the `dev-hostpath-mapper` profile in `devspace.yaml`. You can do this by running the following command:

```
devspace dev -p dev-hostpath-mapper
```

This should deploy the hostpath-mapper as a deployment instead of a daemonset and take care of other modifications to be made and should allow you to develop the hostpath-mapper itself against the syncer.

## License

This project is licensed under the Apache 2.0 License.

## Copyright notice

It is important to state that you retain copyright for your contributions, but agree to license them for usage by the project and author(s) under the Apache 2.0 license. Git retains history of authorship, but we use a catch-all statement rather than individual names.
