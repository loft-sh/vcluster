set quiet

import? '../sdk-codegen/utils.just'

# ensure tools installed with `go install` are available to call
export PATH := home_directory() + "/go/bin:" + env('PATH')

_default:
    just --list --unsorted

# ⭐ run all unit tests, or pass a package name (./invoice) to only run those tests
test *args="./...":
    go run scripts/test_with_stripe_mock/main.go -race {{ args }}

# check for potential mistakes (slow)
lint: install
    go vet ./...
    staticcheck

# don't depend on `install` in this step! Before formatting, our `go` code isn't syntactically valid
# ⭐ format all files
format: _normalize-imports install
    scripts/gofmt.sh
    goimports -w example/generated_examples_test.go

# verify, but don't modify, the formatting of the files
format-check:
    scripts/gofmt.sh check

# ensures all client structs are properly registered
check-api-clients:
    go run scripts/check_api_clients/main.go

ci-test: test bench check-api-clients

# compile the project
build:
    go build ./...

# install dependencies (including those needed for development). Mostly called by other recipes
install:
    go get -t
    go install honnef.co/go/tools/cmd/staticcheck@v0.4.7
    go install golang.org/x/tools/cmd/goimports@v0.24.0

# run benchmarking to check for performance regressions
bench:
    go test -race -bench . -run "Benchmark" ./form

# called by tooling. It updates the package version in the `VERSION` file and `stripe.go`
[private]
update-version version: && _normalize-imports
    echo "{{ version }}" > VERSION
    perl -pi -e 's|const clientversion = "[.\d\-\w]+"|const clientversion = "{{ version }}"|' stripe.go

# go imports use the package's major version in the path, so we need to update them
# we also generate files with a placeholder `[MAJOR_VERSION]` that we need to replace
# we can pull the major version out of the `VERSION` file
# NOTE: because we run this _after_ other recipes that modify `VERSION`, it's important that we only read the file in the argument evaluation
# (if it's a top-level variable, it's read when the file is parsed, which is too early)
# arguments are only evaluated when the recipe starts
# so, setting it as the default means we get both the variable and the lazy evaluation we need
_normalize-imports major_version=replace_regex(`cat VERSION`, '\..*', ""):
    perl -pi -e 's|github.com/stripe/stripe-go/v\d+|github.com/stripe/stripe-go/v{{ major_version }}|' README.md
    perl -pi -e 's|github.com/stripe/stripe-go/v\d+|github.com/stripe/stripe-go/v{{ major_version }}|' go.mod
    find . -name '*.go' -exec perl -pi -e 's|github.com/stripe/stripe-go/(v\d+\|\[MAJOR_VERSION\])|github.com/stripe/stripe-go/v{{ major_version }}|' {} +
