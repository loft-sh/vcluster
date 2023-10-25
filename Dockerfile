# Build the manager binary
FROM golang:1.21 as builder

WORKDIR /vcluster-dev
ARG TARGETOS
ARG TARGETARCH
ARG BUILD_VERSION=dev
ARG TELEMETRY_PRIVATE_KEY=""
ARG HELM_VERSION="v3.13.1"
ARG ETCD_VER="v3.4.27"

# Install kubectl for development
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/`curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt`/bin/linux/amd64/kubectl && chmod +x ./kubectl && mv ./kubectl /usr/local/bin/kubectl

# Install helm binary
RUN curl -s https://get.helm.sh/helm-${HELM_VERSION}-linux-${TARGETARCH}.tar.gz > helm3.tar.gz && tar -zxvf helm3.tar.gz linux-${TARGETARCH}/helm && chmod +x linux-${TARGETARCH}/helm && mv linux-${TARGETARCH}/helm /usr/local/bin/helm && rm helm3.tar.gz && rm -R linux-${TARGETARCH}

# Install etcd binary
RUN mkdir -p /tmp/etcd-download-test && curl -L https://github.com/etcd-io/etcd/releases/download/${ETCD_VER}/etcd-${ETCD_VER}-linux-${TARGETARCH}.tar.gz -o ./etcd-${ETCD_VER}-linux-${TARGETARCH}.tar.gz && tar xzvf ./etcd-${ETCD_VER}-linux-${TARGETARCH}.tar.gz -C /tmp/etcd-download-test --strip-components=1 && rm -f ./etcd-${ETCD_VER}-linux-${TARGETARCH}.tar.gz && chmod +x /tmp/etcd-download-test/etcd && mv /tmp/etcd-download-test/etcd /usr/local/bin/etcd && rm -R /tmp/etcd-download-test

# Install Delve for debugging
RUN if [ "${TARGETARCH}" = "amd64" ] || [ "${TARGETARCH}" = "arm64" ]; then go install github.com/go-delve/delve/cmd/dlv@latest; fi

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
COPY vendor/ vendor/

# Copy the go source
COPY cmd/vcluster cmd/vcluster
COPY cmd/vclusterctl cmd/vclusterctl
COPY pkg/ pkg/

# Symlink /manifests folder to the synced location for development purposes
RUN ln -s "$(pwd)/manifests" /manifests

ENV GO111MODULE on
ENV DEBUG true

# create and set GOCACHE now, this should slightly speed up the first build inside of the container
# also create /.config folder for GOENV, as dlv needs to write there when starting debugging
RUN mkdir -p /.cache /.config
ENV GOCACHE=/.cache
ENV GOENV=/.config

# Copy and embed the helm charts
COPY charts/ charts/
COPY hack/ hack/
RUN go generate -tags embed_charts ./...

# Set home to "/" in order to for kubectl to automatically pick up vcluster kube config
ENV HOME /

# Build cmd
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GO111MODULE=on go build -mod vendor -tags embed_charts -ldflags "-X github.com/loft-sh/vcluster/pkg/telemetry.SyncerVersion=$BUILD_VERSION -X github.com/loft-sh/vcluster/pkg/telemetry.telemetryPrivateKey=$TELEMETRY_PRIVATE_KEY" -o /vcluster cmd/vcluster/main.go

# RUN useradd -u 12345 nonroot
# USER nonroot

ENTRYPOINT ["go", "run", "-mod", "vendor", "cmd/vcluster/main.go"]

# we use alpine for easier debugging
FROM alpine:3.18

# Set root path as working directory
WORKDIR /

COPY --from=builder /vcluster .
COPY --from=builder /usr/local/bin/helm /usr/local/bin/helm
COPY --from=builder /usr/local/bin/etcd /usr/local/bin/etcd
COPY manifests/ /manifests/

# RUN useradd -u 12345 nonroot
# USER nonroot

ENTRYPOINT ["/vcluster", "start"]
