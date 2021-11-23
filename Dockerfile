# Build the manager binary
FROM golang:1.16 as builder

WORKDIR /vcluster
ARG TARGETOS
ARG TARGETARCH

# Install kubectl for development
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/`curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt`/bin/linux/amd64/kubectl && chmod +x ./kubectl && mv ./kubectl /usr/local/bin/kubectl

# Install Delve for debugging
RUN if [ "${TARGETARCH}" = "amd64" ]; then go get github.com/go-delve/delve/cmd/dlv; fi

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
COPY vendor/ vendor/

# Copy the go source
COPY cmd/vcluster cmd/vcluster
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
# Ensure the default group(0) owns all files and folders in /vcluster and /.cache 
# to allow sync to /vcluster with devspace and allow go to write into build cache even when run as non-root
RUN chgrp -R 0 /vcluster /.cache /.config && \
    chmod -R g=u /vcluster /.cache /.config

# Set home to "/" in order to for kubectl to automatically pick up vcluster kube config 
ENV HOME /

# Build cmd
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GO111MODULE=on go build -mod vendor -o vcluster cmd/vcluster/main.go

ENTRYPOINT ["go", "run", "-mod", "vendor", "cmd/vcluster/main.go"]

# we use alpine for easier debugging
FROM alpine

# Set root path as working directory
WORKDIR /

COPY --from=builder /vcluster/vcluster .
COPY manifests/ /manifests/

ENTRYPOINT ["/vcluster"]
