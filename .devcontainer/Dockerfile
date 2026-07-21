FROM mcr.microsoft.com/devcontainers/go:1.25-bookworm

ARG TARGETOS
ARG TARGETARCH

RUN curl -L -o devspace "https://github.com/loft-sh/devspace/releases/latest/download/devspace-linux-${TARGETARCH}" && install -c -m 0755 devspace /usr/local/bin
RUN curl -L -o vcluster "https://github.com/loft-sh/vcluster/releases/latest/download/vcluster-linux-${TARGETARCH}" && install -c -m 0755 vcluster /usr/local/bin
RUN curl -L -o kind "https://kind.sigs.k8s.io/dl/v0.29.0/kind-linux-${TARGETARCH}" && install -c -m 0755 kind /usr/local/bin
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/${TARGETARCH}/kubectl && chmod +x ./kubectl && mv ./kubectl /usr/local/bin
RUN bash -c "curl -s https://get.helm.sh/helm-v3.18.6-linux-${TARGETARCH}.tar.gz > helm3.tar.gz" && tar -zxvf helm3.tar.gz linux-${TARGETARCH}/helm && chmod +x linux-${TARGETARCH}/helm && mv linux-${TARGETARCH}/helm /usr/local/bin && rm helm3.tar.gz && rm -R linux-${TARGETARCH}
