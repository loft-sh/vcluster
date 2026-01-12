# NOTE: this file is deprecated and slated for deletion; prefer using the equivalent `just` commands.

all: test bench vet lint check-api-clients check-gofmt ci-test

bench:
	go test -race -bench . -run "Benchmark" ./form

build:
	go build ./...

check-api-clients:
	go run scripts/check_api_clients/main.go

check-gofmt:
	scripts/gofmt.sh check

lint:
	staticcheck

test:
	go run scripts/test_with_stripe_mock/main.go -race ./...

ci-test: test bench check-api-clients
vet:
	go vet ./...

coverage:
	go run scripts/test_with_stripe_mock/main.go -covermode=count -coverprofile=combined.coverprofile ./...

coveralls:
	go install github.com/mattn/goveralls@latest && $(HOME)/go/bin/goveralls -service=github -coverprofile=combined.coverprofile

clean:
	find . -name \*.coverprofile -delete

MAJOR_VERSION := $(shell echo $(VERSION) | sed 's/\..*//')
update-version:
	@echo "$(VERSION)" > VERSION
	@perl -pi -e 's|const clientversion = "[.\d\-\w]+"|const clientversion = "$(VERSION)"|' stripe.go
	@perl -pi -e 's|github.com/stripe/stripe-go/v\d+|github.com/stripe/stripe-go/v$(MAJOR_VERSION)|' README.md
	$(MAKE) normalize-imports

codegen-format: normalize-imports
	scripts/gofmt.sh
	go install golang.org/x/tools/cmd/goimports@v0.24.0 && goimports -w example/generated_examples_test.go

CURRENT_MAJOR_VERSION := $(shell cat VERSION | sed 's/\..*//')
normalize-imports:
	@perl -pi -e 's|github.com/stripe/stripe-go/v\d+|github.com/stripe/stripe-go/v$(CURRENT_MAJOR_VERSION)|' go.mod
	@find . -name '*.go' -exec perl -pi -e 's|github.com/stripe/stripe-go/(v\d+\|\[MAJOR_VERSION\])|github.com/stripe/stripe-go/v$(CURRENT_MAJOR_VERSION)|' {} +

.PHONY: codegen-format update-version normalize-imports
