#!/usr/bin/env bash
# This script will build vcluster using root podman socket of CRC instance
set -e
eval $(crc podman-env --root)
echo "Building using the following podman socket - $CONTAINER_HOST"
podman --remote build -f Dockerfile "$@"