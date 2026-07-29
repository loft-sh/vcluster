#!/usr/bin/env bash

# Set required go flags
export GO111MODULE=on
export GOFLAGS=-mod=vendor

# Test if we can build the program
echo "Building virtual cluster..."
go generate ./... && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build cmd/vcluster/main.go || exit 1

# Merged coverage profile consumed by CI and uploaded to Codecov. We collect a
# per-package profile and append it here so the final file spans every package.
# Previously each package overwrote a single profile, so only the last package
# survived and the coverage was effectively discarded.
COVERAGE_OUT="$(pwd)/coverage.out"
echo "mode: atomic" > "$COVERAGE_OUT"

# List packages
PKGS=$(go list ./... | grep -v -e /vendor/ -e /test -e /e2e)
echo "Start testing..."
fail=false
for pkg in $PKGS; do
 pkgCover=$(mktemp)
 go test -race -coverprofile="$pkgCover" -covermode=atomic "$pkg"
 if [ $? -ne 0 ]; then
   fail=true
 fi
 # Append this package's coverage, dropping its own "mode:" header line so the
 # merged profile keeps a single header.
 if [ -s "$pkgCover" ]; then
   tail -n +2 "$pkgCover" >> "$COVERAGE_OUT"
 fi
 rm -f "$pkgCover"
done

if [ "$fail" = true ]; then
 echo "Failure"
 exit 1
fi
