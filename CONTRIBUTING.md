Contributing

# Contributing to vcluster

Thank you for contributing to vcluster! Here you can find common questions around developing vcluster.

# Table of Contents

- [How can I get involved?](#how-can-i-get-involved)
- [Developing vCluster](#developing-vcluster)
  - [Pre-requisites for Development](#pre-requisites-for-development)
  - [Developing the Different vCluster Containers](#developing-the-different-vcluster-containers)
  - [Developing and Debugging with DevSpace](#developing-and-debugging-vcluster-containers--with-devspace)
  - [Build and Test the vcluster CLI tool](#build-and-test-the-vcluster-cli-tool)
  - [Developing without DevSpace](#developing-without-devspace)

- [Running vcluster Tests](#running-vcluster-tests)
- [License](#license)
- [Copyright notice](#copyright-notice)

# How can I get involved?

There are a number of areas where contributions can be accepted:

- Write code to fix bugs or implement features
- Review pull requests
- Try out our alphas and betas and give us feedback in our community Slack channel
- Help respond to Github issues to help our community
- Create [docs](https://github.com/loft-sh/vcluster-docs) or guides

# Developing vCluster

We recommend developing vCluster directly on a local Kubernetes cluster as it provides faster feedback. There are two ways that we recommend developing.

- DevSpace
- Locally

## Pre-requisites for Development

### Tools

- Docker needs to be installed (e.g. docker-desktop, orbstack, rancher desktop etc.)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [Helm v3.10.0+](https://helm.sh/docs/intro/install/)
- Local Kubernetes v1.26+ cluster (i.e. Docker Desktop, [minikube](https://minikube.sigs.k8s.io/docs/start/), KinD or similar)

### Fork and Clone the vcluster repo

Click Fork button (top right) to establish a cloud-based fork.

```
git clone YOUR_PERSONAL_FORK_URL
```

## Developing the Different vCluster Containers

One of the primary containers of vCluster is the `syncer` but you can also work on other components like `hostpath-mapper`. The guide focuses on the `syncer` container.

## Developing and Debugging vCluster Containers with DevSpace

The vCluster repo is enabled to develop using [DevSpace](https://www.devspace.sh/). Devspace will automatically set up a vcluster and sync your code changes from the repository to a running container. So, you can easily develop and test.

### Install DevSpace

Follow the guide on how to install [DevSpace](https://github.com/loft-sh/devspace#1-install-devspace)

### Launch DevSpace

Ensure your `kubectl` is connected to the local Kubernetes cluster.

```
kubectl get namespaces
```

In your Github `vcluster` directory, run:

```
devspace dev
```

Which uses the `devspace.yaml` file in the `vcluster` directory to deploy a vCluster and launch DevSpace:

```
info Using namespace 'vcluster'
info Using kube context 'minikube'
info Created namespace: vcluster
build:vcluster Rebuild image ghcr.io/loft-sh/loft-enterprise/dev-vcluster because tag is missing
build:vcluster Building image 'ghcr.io/loft-sh/loft-enterprise/dev-vcluster:YwXtFIF' with engine 'buildkit'
build:vcluster Execute BuildKit command with: docker buildx build --tag ghcr.io/loft-sh/loft-enterprise/dev-vcluster:YwXtFIF --file Dockerfile --target builder -
#0 building with "default" instance using docker driver557.1kB
build:vcluster
build:vcluster #1 [internal] load remote build context
Sending build context to Docker daemon  99.17MBdaemon  17.83MB
...
...
build:vcluster #21 exporting to image
build:vcluster #21 exporting layers
build:vcluster #21 exporting layers 3.9s done
build:vcluster #21 writing image sha256:35a71edf70fe3be275cfd42a775e15621a9067880b5f397e957daedb0ed7b73b
build:vcluster #21 writing image sha256:35a71edf70fe3be275cfd42a775e15621a9067880b5f397e957daedb0ed7b73b done
build:vcluster #21 naming to ghcr.io/loft-sh/loft-enterprise/dev-vcluster:YwXtFIF done
build:vcluster #21 DONE 3.9s
build:vcluster
build:vcluster View build details: docker-desktop://dashboard/build/default/default/q7vc4iv3ip8v24qxnybv4eo5p
build:vcluster Done processing image 'ghcr.io/loft-sh/loft-enterprise/dev-vcluster'
deploy:vcluster-k8s Deploying chart ./chart (vcluster) with helm...
deploy:vcluster-k8s Deployed helm chart (Release revision: 1)
deploy:vcluster-k8s Successfully deployed vcluster-k8s with helm
dev:vcluster Waiting for pod to become ready...
dev:vcluster DevSpace is waiting, because Pod vcluster-devspace-7b467dc9cd-hbtbz has status: Init:1/3
dev:vcluster Selected pod vcluster-devspace-7b467dc9cd-hbtbz
dev:vcluster ports Port forwarding started on: 2346 -> 2345
dev:vcluster sync  Sync started on: . <-> .
dev:vcluster sync  Waiting for initial sync to complete
dev:vcluster sync  Initial sync completed
dev:vcluster term  Opening shell to syncer:vcluster-devspace-7b467dc9cd-hbtbz (pod:container)

   ____              ____
  |  _ \  _____   __/ ___| _ __   __ _  ___ ___
  | | | |/ _ \ \ / /\___ \| '_ \ / _` |/ __/ _ \
  | |_| |  __/\ V /  ___) | |_) | (_| | (_|  __/
  |____/ \___| \_/  |____/| .__/ \__,_|\___\___|
                          |_|

Welcome to your development container!
This is how you can work with it:
- Run `go run -mod vendor cmd/vcluster/main.go start` to start vcluster
- Run `devspace enter -n vcluster --pod vcluster-devspace-7b467dc9cd-hbtbz -c syncer` to create another shell into this container
- Run `kubectl ...` from within the container to access the vcluster if its started
- Files will be synchronized between your local machine and this container

NOTE: you may need to provide additional flags through the command line, because the flags set from the chart are ignored in the dev mode.

If you wish to run vcluster in the debug mode with delve, run:
  `dlv debug ./cmd/vcluster/main.go --listen=0.0.0.0:2345 --api-version=2 --output /tmp/__debug_bin --headless --build-flags="-mod=vendor" -- start`
  Wait until the `API server listening at: [::]:2345` message appears
  Start the "Debug vcluster (localhost:2346)" configuration in VSCode to connect your debugger session.
  Note: vcluster won't start until you connect with the debugger.
  Note: vcluster will be stopped once you detach your debugger session.

TIP: hit an up arrow on your keyboard to find the commands mentioned above :)

vcluster-0:vcluster-dev$
```

Now, your terminal is running in DevSpace, and you can develop and test with DevSpace.

### Developing and Testing vCluster using DevSpace

Start vcluster in DevSpace via `go run`

```
vcluster-0:vcluster-dev$ go run -mod vendor cmd/vcluster/main.go start
```

Now, you can start to work with the virtual cluster based on the source code. This vCluster is running on your local Kubernetes cluster.

If you change a file locally, DevSpace will automatically sync the file into the Devspace container. After any changes, re-run the same command in the DevSpace terminal to apply the changes.

#### Start vcluster in DevSpace in debug mode via `dlv`

You can either debug with Delve within DevSpace or locally. Devspace is more convenient as no port forwarding is required.

Run vCluster in the debug mode with Delve in the `vcluster` directory. Note: Other sessions of DevSpace will need to be terminated before starting another

```
devspace dev -n vcluster
```

Once DevSpace launches and you are in the `vcluster` pod, run the following delve command.

```
vcluster-0:vcluster-dev$ dlv debug ./cmd/vcluster/main.go --listen=0.0.0.0:2345 --api-version=2 --output /tmp/__debug_bin --headless --build-flags="-mod=vendor" -- start
```

Wait until the `API server listening at: [::]:2345` message appears.

Start the `Debug vcluster (localhost:2346)` configuration in Visual Studio Code to connect your debugger session.

If you are not using  Visual Studio Code, configure your IDE to connect to `localhost:2346` for the "remote" delve debugging.

**Note:** vCluster won't start until you connect with the debugger.
**Note:** vCluster will be stopped once you detach your debugger session.

### Access your vCluster and Set your local KubeConfig

Download the [vCluster CLI](https://www.vcluster.com/docs/get-started/) and use it to connect to your virtual cluster.

By connecting to the vCluster using the CLI, you set your local KubeConfig to the virtual cluster

```
vcluster connect vcluster
```

## Build and Test the vcluster CLI tool

Build the CLI tool

```
go generate ./... && go build -o vcluster cmd/vclusterctl/main.go # build vcluster cli
```

Test the built CLI tool

```
./vcluster create v1 # create vcluster
```

## Developing without DevSpace

### Pre-requisites

- [Golang v1.22](https://go.dev/doc/install)
- [Goreleaser](https://goreleaser.com/install/)
- [Just](https://github.com/casey/just)
- [Kind](https://kind.sigs.k8s.io/)

### Uninstall vCluster CLI

If you already have vCluster CLI installed, please make sure to uninstall it first.

### Build vCluster CLI from Source Code

```
$ just build-snapshot

• starting build...
[...]
  • running before hooks
    • running                                        hook=go mod tidy
    • running                                        hook=just embed-charts 0.20.0-next
    • running                                        hook=just clean-release
    • running                                        hook=just copy-assets
    • running                                        hook=just generate-vcluster-images 0.20.0-next
  [...]
  • building binaries
    • partial build                                  match=target=darwin_arm64
    • partial build                                  match=target=darwin_arm64
    • building                                       binary=dist/<ARCH>/vcluster
  [...]
  • build succeeded after 11s
```

### Verify vCluster CLI was compiled correctly

```
$ ./dist/<ARCH>/vcluster version
vcluster version 0.20.0-next
```

### Bring up a local K8s cluster using Kind

```
kind create cluster

Creating cluster "kind" ...
 ✓ Ensuring node image (kindest/node:v1.29.2) 🖼
 ✓ Preparing nodes 📦
 ✓ Writing configuration 📜
 ✓ Starting control-plane 🕹️
 ✓ Installing CNI 🔌
 ✓ Installing StorageClass 💾
Set kubectl context to "kind-kind"
You can now use your cluster with:

kubectl cluster-info --context kind-kind

```

### Build vCluster Container Image

```
docker build . -t my-vcluster:0.0.1
```

Note: Feel free to push this image into your own registry.

#### Importing vCluster Container Image for kind Users

If using kind as your local Kubernetes cluster, you need to import the image into kind.

```
kind load docker-image my-vcluster:0.0.1
```

### Create vCluster with self-compiled vCluster CLI

For vCluster v0.20+:

#### Create a `vcluster.yaml`

Create a `vcluster.yaml` that sets the image to be your locally built Docker image.

```yaml
controlPlane:
  statefulSet:
    imagePullPolicy: Never
    image:
      registry: ""
      repository: my-vcluster
      tag: 0.0.1
```

#### Deploy vCluster

Launch your vCluster using your `vcluster.yaml`

```
./dist/<ARCH>/vcluster create my-vcluster -n my-vcluster -f ./vcluster.yaml --local-chart-dir chart
```

### Access your vCluster and Set your local KubeConfig

By connecting to the vCluster using the CLI, you set your local KubeConfig to the virtual cluster

```
./dist/<ARCH>/vcluster connect my-vcluster
```

# Running vCluster Tests

All of the tests are located in the vcluster directory.

## Running the Unit Test Suite

Run the entire unit test suite.

```
./hack/test.sh
```

## Running the e2e Test Suite

Run the e2e tests, that are located in the e2e folder.

```
just delete-kind
just e2e

```

If [Ginkgo](https://github.com/onsi/ginkgo#global-installation) is already installed, run  `ginkgo -v`.

## Run the Conformance Tests

For running conformance tests, please take a look at [conformance tests](https://github.com/loft-sh/tree/vcluster/main/conformance/v1.21)

# License

This project is licensed under the Apache 2.0 License.

# Copyright notice

It is important to state that you retain copyright for your contributions, but agree to license them for usage by the project and author(s) under the Apache 2.0 license. Git retains history of authorship, but we use a catch-all statement rather than individual names.
