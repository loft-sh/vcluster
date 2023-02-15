FROM mcr.microsoft.com/devcontainers/go:0-1.19-bullseye

ARG TARGETOS
ARG TARGETARCH

RUN curl -L -o devspace "https://github.com/loft-sh/devspace/releases/latest/download/devspace-linux-${TARGETARCH}" && install -c -m 0755 devspace /usr/local/bin
RUN curl -L -o vcluster "https://github.com/loft-sh/vcluster/releases/latest/download/vcluster-linux-${TARGETARCH}" && install -c -m 0755 vcluster /usr/local/bin
