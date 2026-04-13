---
paths:
  - "e2e-next/**/*.go"
---
<!-- Generic core: e2e-tdd-workflow plugin references/e2e-error-handling.md (identical) -->

# e2e-next Error Handling

## Cleanup: only tolerate IsNotFound, assert everything else

Single-operation cleanup:

```go
// GOOD
DeferCleanup(func(ctx context.Context) {
    err := client.Delete(ctx, name, metav1.DeleteOptions{})
    Expect(clientpkg.IgnoreNotFound(err)).To(Succeed())
})
```

Multi-step cleanup (get → mutate → delete):

```go
// GOOD — check IsNotFound on Get, assert all other errors at every step
DeferCleanup(func(ctx context.Context) {
    obj, err := client.Get(ctx, name, metav1.GetOptions{})
    if kerrors.IsNotFound(err) {
        return // already gone
    }
    Expect(err).NotTo(HaveOccurred())

    delete(obj.Annotations, someAnnotation)
    _, err = client.Update(ctx, obj, metav1.UpdateOptions{})
    Expect(err).NotTo(HaveOccurred())

    err = client.Delete(ctx, name, metav1.DeleteOptions{})
    Expect(clientpkg.IgnoreNotFound(err)).To(Succeed())
})
```

**When NOT to use `IgnoreNotFound`:** `IgnoreNotFound` is appropriate for resources that may have been deleted by cascade, by a prior spec, or by a controller. For resources the test **just created in the same `It` block** and expects to still exist at cleanup time, prefer a strict assertion — `Expect(err).NotTo(HaveOccurred())`. A `NotFound` in this case signals a test bug or unexpected controller behavior, not a benign race.

Never use these patterns:
- `_, _ =` to discard errors — silent failures mask regressions and leak resources
- `if err \!= nil { return }` to bail on any error — this swallows connectivity/RBAC failures, not just NotFound

## Error assertions: assert the specific error, not just occurrence

```go
// BAD — passes on connectivity errors, RBAC failures, or any unrelated error
Expect(err).To(HaveOccurred())

// GOOD
Expect(err).To(MatchError(ContainSubstring("already exists")))
```
