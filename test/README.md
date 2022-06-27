### How to execute the e2e tests locally

1. Start the vcluster via `devspace run dev` and running `go run -mod vendor cmd/vcluster/main.go start --sync 'networkpolicies'` in the terminal
2. Then start the e2e tests via `VCLUSTER_SUFFIX=vcluster go test -v -ginkgo.v -ginkgo.skip='.*NetworkPolicy.*'`
