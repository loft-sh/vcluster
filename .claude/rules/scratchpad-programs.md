---
globs: ["**/*.go"]
---
<!-- Generic core: e2e-tdd-workflow plugin references/scratchpad-programs.md (identical) -->

## Verifying assumptions with scratchpad programs

When you need to verify an assumption — about API behavior, data shape, timing,
RBAC, sync logic, or anything else — write a small standalone program in
`.agent-scratchpad/` that directly answers the question.

This is not a failure-recovery technique. Use it proactively any time you're
about to write code based on an assumption you haven't confirmed.

### Examples of when to use this

- "Does this CRD have a status subresource?"
- "What does the syncer actually return for a conflicting resource name?"
- "Does client-go's patch merge or replace this nested field?"
- "What order do these finalizers run in?"
- "Is this label selector format valid?"

### How

1. Create a subfolder in `.agent-scratchpad/` named after the question
   (e.g., `.agent-scratchpad/check-patch-merge/main.go`). Each program gets
   its own folder — never put multiple programs in the same folder.

2. Run it:
   ```bash
   cd .agent-scratchpad/check-patch-merge && go run main.go --kubeconfig="${HOME}/.kube/config" --context=kind-kind-cluster
   ```
   If no cluster is needed (pure logic, parsing, transforms), skip the kubeconfig.

3. Read the output, answer the question, delete the folder.

### Guidelines

| Do | Don't |
|----|-------|
| One folder, one file, one question | Build a mini test framework |
| Name the folder after the question | Use generic names like `test1/` |
| Print the answer and exit | Add flags for "reuse later" |
| Use `go run` directly | Create a separate go.mod |
| Delete the folder after use | Leave folders accumulating |
