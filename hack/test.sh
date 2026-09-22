#!/usr/bin/env bash

set -euo pipefail

# Set required go flags
export GO111MODULE=on
export GOFLAGS=-mod=vendor

# Test if we can build the program
echo "Building virtual cluster..."
go generate ./...
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /dev/null cmd/vcluster/main.go

# List packages
PKGS=$(go list ./... | grep -v -e /vendor/ -e /test -e /e2e)

# One invocation tests every package in parallel and writes a single merged
# coverage profile, so no per-package stitching is needed. -count=1 stops a warm
# GOCACHE from serving "(cached)" results, which skips the race detector.
echo "Start testing..."
go test -race -covermode=atomic -coverprofile=coverage.out -count=1 ${PKGS}
