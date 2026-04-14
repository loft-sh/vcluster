---
paths:
  - "e2e-next/**/*.go"
---
<!-- Generic core: e2e-tdd-workflow plugin references/e2e-test-structure-core.md -->

# e2e-next Test Structure Rules

| Do | Don't |
|---|---|
| Register `DeferCleanup` immediately after creating a resource, before any assertions. If an assertion fails before cleanup is registered, the resource leaks. | Register cleanup after assertions — a failed assertion skips the rest of the block and the resource leaks forever. |
| Add failure context to assertions inside `Eventually`: `g.Expect(phase).To(Equal("Ready"), "reason: %v, message: %v", obj.Status.Reason, obj.Status.Message)` | Write `g.Expect(phase).To(Equal("Ready"))` inside `Eventually` with no context — when it times out you get no diagnostics. |
| Use `By("description", func() { ... })` with a closure so Ginkgo reports which step failed and how long it took. | Use bare `By("description")` followed by loose statements — Ginkgo can't attribute failures to the step. |
| Default to `BeforeEach`. Only use `Ordered` for true sequential dependencies. | Use `Ordered` for convenience — every `Ordered` context is a parallelization bottleneck. |
| Tolerate `NotFound` in cleanup, assert everything else. See `e2e-error-handling.md` for patterns. | Use `_, _ =` to swallow errors in `DeferCleanup` — silent failures mask regressions and leak resources. |
| Clean up only the specific resources you created, by name or label selector. | `List` all resources of a type in cleanup — parallel tests create the same types and you'll delete their resources. |
| Assert specific error messages: `Expect(err).To(MatchError(ContainSubstring("...")))` or `kerrs.Be(metav1.StatusReason...)` | Write `Expect(err).To(HaveOccurred())` — this passes on unrelated failures like connectivity errors. |
| Use existing cluster definitions from `clusters/` and `setup/template` for YAML rendering. Browse available definitions before writing any direct client call in test setup or cleanup. | Write one-off raw client calls when a cluster definition or template helper exists — this duplicates error handling, skips context tracking, and diverges from the standard patterns over time. |
