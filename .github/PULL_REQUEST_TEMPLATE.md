**What issue type does this pull request address?** (keep at least one, remove the others) 
/kind bugfix
/kind enhancement
/kind feature
/kind documentation
/kind test

**What does this pull request do? Which issues does it resolve?** (use `resolves #<issue_number>` if possible) 
resolves #


**Please provide a short message that should be published in the vcluster release notes**
Fixed an issue where vcluster ...


**What else do we need to know?** 

## E2E Tests

### Default Test Execution
The mandatory PR suite runs automatically. Only specify additional test suites below if needed.

### Adding New Test Suites
When adding a new ginkgo test suite:
- [ ] **Add labels** to the test suite
- [ ] **Update label-filter section** below to execute the new test suite
- [ ] **Verify test suite runs** in CI/CD pipeline

### Additional test suites
<!--
You can specify custom Ginkgo label filters for e2e tests by adding a label-filter code block.

Available labels: core, sync, pr

For a complete list of existing labels, see: e2e-next/labels/labels.go

You can combine labels using Ginkgo syntax: && (AND), || (OR), ! (NOT)

Examples:
- Run only pr tests: "none" (default) - test labeled "pr" are always run
- Additionally run virtual cluster tests: "core"
- Run all tests: "!pr" - litte hack, this results in "!pr || pr" which actually means all tests
- Run tests that have the label 'team' within the 'managementv1' label category: "managementv1: containsAny team" or "managementv1: containsAll team"
- Run tests that have labels 'virtual-cluster-instance' or 'user' within the 'managementv1' label category: "managementv1: consistsOf { virtual-cluster-instance, user }"
-->
Additional test suite(s) that will be executed before the mandatory PR suite:

```label-filter
none
```
