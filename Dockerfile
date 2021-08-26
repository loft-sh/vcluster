# Build the manager binary
FROM golang:1.16 as builder

WORKDIR /vcluster
ARG TARGETOS
ARG TARGETARCH

# Install kubectl for development
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/`curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt`/bin/linux/amd64/kubectl && chmod +x ./kubectl && mv ./kubectl /usr/local/bin/kubectl

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
COPY vendor/ vendor/

# Copy the go source
COPY cmd/vcluster cmd/vcluster
COPY pkg/ pkg/

ENV GO111MODULE on
ENV DEBUG true

# Build cmd
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GO111MODULE=on go build -mod vendor -o vcluster cmd/vcluster/main.go

ENTRYPOINT ["go", "run", "-mod", "vendor", "cmd/vcluster/main.go"]

# we use alpine for easier debugging
FROM alpine

# Set root path as working directory
WORKDIR /

COPY --from=builder /vcluster/vcluster .

ENTRYPOINT ["/vcluster"]
