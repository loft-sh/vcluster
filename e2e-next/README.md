## How to add a new test

1. Identify the critical tests that cover the core functionality of your feature. These will be run on every PR and should be marked with `labels.PR`.
2. Other tests, like edge cases or optional flows, can have different labels. Any new labels must be added to `labels/labels.go`.
3. Add your test files in the appropriate folder: `test_features` for feature tests, `test_integrations` for integration tests
4. If you created a new folder, make sure to add the import in `e2e_suite_test.go`.

### Preparations
* Install [vcluster cli](https://www.vcluster.com/docs/vcluster)
* Install ginkgo cli via `go install github.com/onsi/ginkgo/v2/ginkgo@v2.23.4`
* Install [kind](https://kind.sigs.k8s.io/docs/user/quick-start/)
```

### Develop a test
If you want to iterate over a test, you can do:
1. Run `just build-snapshot`
2. Run `just setup [LABEL]` and it will do the following:
  * Create a kind cluster
  * Load vcluster image into kind cluster

Then iterate via:
1. Run `just iterate-e2e [LABEL]`
2. Change things, then rerun above command

### Run a test
If you just want to run a test, you can do:
1. Run `just build-snapshot`
2. Run `just iterate-e2e [LABEL]`

### Destroy the kind cluster
If you want to cleanup the state, you can do:
1. Run `just teardown my-feature`