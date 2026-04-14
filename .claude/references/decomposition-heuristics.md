# Decomposition Heuristics

## The Core Test: "Is This One It()?"

A sub-problem is ready when you can express it as:

```go
It("should {one observable behavior}", func(ctx context.Context) {
    By("{action 1}")
    By("{action 2}")
    By("{verification}")
    // At most 3 By() steps
})
```

If you need more than 3 `By()` steps, the sub-problem is too big. Split it.

## Splitting Strategies

### Strategy 1: Split by CRUD Operation

A feature that creates, reads, updates, and deletes becomes 4 sub-problems:

```
SP-1: Create {resource} with valid input
SP-2: Get {resource} returns expected fields
SP-3: Update {resource} changes specific field
SP-4: Delete {resource} cleans up dependents
```

### Strategy 2: Split by Actor

When multiple roles interact with a feature:

```
SP-1: Admin can create {resource}
SP-2: Project member can view {resource}
SP-3: Non-member gets Forbidden
```

### Strategy 3: Split by State Transition

For lifecycle features (sleep mode, VCI phases):

```
SP-1: Resource starts in {initial state}
SP-2: Trigger moves resource to {intermediate state}
SP-3: Resource reaches {final state}
SP-4: Error condition produces {error state}
```

### Strategy 4: Split by Integration Boundary

When a feature touches multiple systems:

```
SP-1: Management API accepts the request (API layer)
SP-2: Controller reconciles the resource (controller layer)
SP-3: Agent reflects the change (agent layer)
```

### Strategy 5: Split by Input Variation

When behavior depends on configuration:

```
SP-1: Default configuration produces {default behavior}
SP-2: Custom configuration A produces {behavior A}
SP-3: Invalid configuration is rejected with {error}
```

## Merging Rules

Merge two sub-problems if:
- They always modify the same lines of code
- One is meaningless without the other (e.g., "create" and "verify creation" are one sub-problem)
- The combined test would still have <= 3 `By()` steps

## Ordering Rules (TDD)

1. **Infrastructure first**: Types, stubs, code generation, new `setup/` builders, new labels, new constants — everything needed for tests to *compile* (not pass)
2. **Test before implementation**: For each behavior, the test sub-problem is ordered before the implementation sub-problem
3. **Happy-path test before error-path test**: Prove it works, then prove it fails correctly
4. **Independent before dependent**: SP-3 that depends on SP-1 comes after SP-1
5. **Mark parallel groups**: Sub-problems with no dependencies get the same group letter

### The TDD Cycle Pattern

Each behavior produces a test→impl pair:
```
SP-N: [test] Write test for behavior X           → RED  (test fails, behavior missing)
SP-M: [impl] Implement behavior X                → GREEN (test passes)
```

Infrastructure sub-problems (SP-0) don't follow this cycle — they exist to make subsequent tests compilable. A stub handler that returns a zero value or error is infrastructure, not implementation.

Example ordering:
```
SP-0: [infra] Types + stub handler + code generation      (group A)
SP-1: [test]  Happy-path test: Create with defaults        (group B, depends: SP-0) → RED
SP-2: [impl]  Implement Create handler                     (group B, depends: SP-1) → GREEN
SP-3: [test]  Error-path test: Non-admin gets Forbidden    (group C, depends: SP-2) → RED
SP-4: [impl]  Implement RBAC check                         (group C, depends: SP-3) → GREEN
SP-5: [test]  Delete cleans up namespace                   (group D, depends: SP-2) → RED
SP-6: [impl]  Implement Delete with cleanup                (group D, depends: SP-5) → GREEN
```

## Size Calibration

Target: each sub-problem takes 15-60 minutes to implement.

| Too small (merge up) | Right size | Too big (split) |
|----------------------|------------|-----------------|
| "Add an import" | "Create resource with validation" | "Implement full CRUD + RBAC" |
| "Check one field" | "Verify state transition" | "Add feature end-to-end" |
| "Set a label" | "Non-admin gets Forbidden" | "Multi-cluster replication" |

## The SP-0 Pattern

When multiple sub-problems need shared infrastructure (a new builder, a new label, a helper function), extract it as SP-0:

```
SP-0: [infra] Create setup builder for {resource}
  - New file: e2e-next/setup/{resource}/{resource}.go
  - Options: WithGenerateName, With{Feature}
  - CRUD: Create, Delete
  - Context accessors: LastFrom, From
```

This unblocks all downstream sub-problems and is itself testable (the builder compiles and the resource can be created/deleted).

### SP-0 in TDD: Stubs and Scaffolding

In a TDD workflow, SP-0 often includes **stub implementations** — handlers that compile and register but return zero values or errors. This is intentional:

```
SP-0: [infra] Define API types + stub handler
  - New type: VirtualClusterInstanceFoo with +subresource-request marker
  - Run code generation (deepcopy, clientset, register)
  - Stub REST handler: returns empty object or "not implemented" error
  - Register stub in register.go
  - Result: tests can compile and call the endpoint, but get wrong/empty results
```

The stub exists solely so the test in SP-1 can compile and fail for the *right reason* (wrong response, not compile error). The real implementation comes in SP-2.

## Red Flags in Decomposition

- **"And" in the title**: "Create resource and verify permissions" → split
- **Multiple actors**: "Admin creates, user views" → split by actor
- **Multiple resources**: "Create project with VCI and space" → split by resource
- **Conditional behavior**: "If config X then A, else B" → split by input variation
- **"End-to-end"**: Almost always too big. What specific behavior are you testing?
