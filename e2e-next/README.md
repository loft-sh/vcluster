## How to add a new test

1. Identify the critical tests that cover the core functionality of your feature. These will be run on every PR and should be marked with `labels.PR`.
2. Other tests, like edge cases or optional flows, can have different labels. Any new labels must be added to `labels/labels.go`.
3. Add your test files in the appropriate folder: `test_features` for feature tests, `test_integrations` for integration tests
4. If you created a new folder, make sure to add the import in `e2e_suite_test.go`.

### Preparations
* Install ginkgo cli via `go install github.com/onsi/ginkgo/v2/ginkgo`
* Install [kind](https://kind.sigs.k8s.io/docs/user/quick-start/)
```

### Setup an environment for manual testing
If you only want an environment that matches a test, you can create the environment without running the tests:
1. Run `just build-snapshot`
2. Run `just build-cli-snapshot`
3. Run `just setup [LABEL]`
4. Perform any manual testing using the deployed virtual cluster environment

### Develop a test
If you want to iterate over a test, you can do:
1. Run `just build-snapshot`
2. Run `just build-cli-snapshot`
3. Run `just iterate-e2e [LABEL]`
4. Rerun step 1 if you modify the vCluster code
5. Rerun step 2 if you modify the vCluster CLI code
6. Rerun step 2 as you modify the tests

### Run a test
If you just want to run a test, you can do:
1. Run `just build-snapshot`
2. Run `just build-cli-snapshot`
3. Run `just iterate-e2e [LABEL]`

### Destroy the kind cluster
If you want to cleanup test environment you can do:
1. Run `just teardown [LABEL]`
