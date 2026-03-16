# Migration: e2e_scheduler — test/e2e_scheduler/scheduler/ -> e2e-next/test_scheduler/

## Problem Summary

The old `test/e2e_scheduler/scheduler/` suite contains 2 `It` blocks across 2 files testing virtual scheduler behavior: (1) `scheduler.go` tests taint/toleration-based pod scheduling — it adds a custom taint to all virtual nodes, verifies a pod WITH the matching toleration runs while a pod WITHOUT it stays pending, then cleans up taints and verifies they match host again; (2) `waitforfirstconsumer.go` tests that a StatefulSet with a `WaitForFirstConsumer` PVC template becomes ready — it discovers or creates a suitable StorageClass on the host, creates the StatefulSet in the vcluster, and polls until it has 1 ready replica. Both tests require a vcluster with the virtual scheduler enabled (`controlPlane.advanced.virtualScheduler.enabled: true`, `controlPlane.distro.k8s.scheduler.enabled: true`) and full node sync (`sync.fromHost.nodes.selector.all: true`). The target is `e2e-next/test_scheduler/`.

## Bootstrap Requirements

- **Standard bootstrap works**: no — requires a new vcluster cluster definition with virtual scheduler enabled and full node sync. No external services needed (the old `values.yaml` also included snapshot PVC, manifests, and Helm charts, but those are for sibling tests imported by the suite, not the scheduler tests themselves).

**Provisioning plan**: `[infra]` sub-problem SP-0 creates:
1. A new YAML template `e2e-next/clusters/vcluster-scheduler.yaml` with virtual scheduler + full node sync
2. A new cluster definition `SchedulerVCluster` in `e2e-next/clusters/clusters.go`
3. Registration in `e2e_suite_test.go` (`SynchronizedBeforeSuite` and `DeferCleanup`)

## Old -> New Translation

| Old Pattern | New Pattern |
|---|---|
| `f := framework.DefaultFramework` | `BeforeEach` retrieving clients from context |
| `f.VClusterClient.CoreV1()...` | `cluster.CurrentKubeClientFrom(ctx).CoreV1()...` |
| `f.HostClient.CoreV1()...` | `cluster.KubeClientFrom(ctx, constants.GetHostClusterName()).CoreV1()...` |
| `f.HostCRClient` (controller-runtime) | `cluster.CurrentClusterClientFrom(ctx)` on host — **Note**: the old test uses `f.HostCRClient` to list StorageClasses on the host. In e2e-next, use `cluster.KubeClientFrom(ctx, constants.GetHostClusterName()).StorageV1().StorageClasses().List(...)` (typed client) instead. |
| `f.VClusterCRClient` (controller-runtime) | `cluster.CurrentClusterClientFrom(ctx)` for vcluster CR client |
| `f.Context` | `ctx context.Context` from Ginkgo `It`/`BeforeEach` signature |
| `framework.ExpectNoError(err)` | `Expect(err).NotTo(HaveOccurred())` |
| `framework.ExpectError(err)` | `Expect(err).To(HaveOccurred())` — but **rephrase**: the old test asserts `PollUntilContextTimeout` returns error (timeout). In e2e-next, use `Consistently` to assert the pod does NOT reach Running within a period, which is more semantically precise. |
| `framework.ExpectEqual(false, reflect.DeepEqual(...))` | `Expect(virtualNodesTaints).NotTo(Equal(hostNodesTaints))` |
| `framework.ExpectEqual(true, reflect.DeepEqual(...))` | `Expect(virtualNodesTaints).To(Equal(hostNodesTaints))` |
| `framework.ExpectNotEmpty(list)` | `Expect(list).NotTo(BeEmpty())` |
| `framework.IsDefaultAnnotation(sc.ObjectMeta)` | Inline check: `sc.Annotations["storageclass.kubernetes.io/is-default-class"] == "true"` |
| `wait.PollUntilContextTimeout(ctx, interval, timeout, ...)` | `Eventually(func(g Gomega) { ... }).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())` |
| `translate.ResetObjectMetadata(&obj.ObjectMeta)` | Same — `translate.ResetObjectMetadata` is a utility, not framework |
| `client.MergeFrom(origNode)` + `Patch()` | Same — controller-runtime merge-patch is still used directly |
| Hardcoded names: `"nginx"`, `"nginx1"`, `"test-statefulset"` | Random suffix: `"scheduler-pod-" + suffix`, `"scheduler-sts-" + suffix` |
| `nsName := "default"` | Use `"default"` namespace (scheduler tests need to schedule on nodes, default namespace is fine) |

## Sub-Problems

### SP-0: [infra] Define SchedulerVCluster cluster with virtual scheduler enabled

**Old**: N/A
**Acceptance**: A new vcluster definition `SchedulerVCluster` exists in `e2e-next/clusters/` with virtual scheduler and full node sync enabled, registered in the suite, and bootstrappable via `just -f Justfile.agent bootstrap "scheduler"`.
**Steps**:
1. Create `e2e-next/clusters/vcluster-scheduler.yaml` with virtual scheduler config:
   ```yaml
   sync:
     fromHost:
       nodes:
         enabled: true
         selector:
           all: true
   controlPlane:
     advanced:
       virtualScheduler:
         enabled: true
     distro:
       k8s:
         scheduler:
           enabled: true
     statefulSet:
       image:
         registry: ""
         repository: {{.Repository}}
         tag: {{.Tag}}
   ```
2. Add `SchedulerVCluster` definition in `e2e-next/clusters/clusters.go` with embedded YAML and `template.MustRender`
3. Register cleanup and setup in `e2e-next/e2e_suite_test.go` (`DeferCleanup` + `setup.AllConcurrent`)
4. Add `labels.Scheduler` to `e2e-next/labels/labels.go`
5. Add blank import `_ "github.com/loft-sh/vcluster/e2e-next/test_scheduler"` in `e2e_suite_test.go`

### SP-1: [migrate] Taint/toleration pod scheduling

**Old**: `It("Use taints and toleration to assign virtual node to pod", ...)` in `scheduler.go`
**Acceptance**: Test adds taints to virtual nodes, verifies a toleration-bearing pod runs and a non-tolerating pod stays unschedulable, then cleans up taints and verifies they match host.
**Steps**:
1. Create `e2e-next/test_scheduler/test_scheduler.go` with `Describe("Virtual scheduler taint/toleration scheduling", labels.Scheduler, cluster.Use(clusters.SchedulerVCluster), cluster.Use(clusters.HostCluster), ...)`
2. Implement the `It` block using `Eventually`/`Consistently` patterns, random-suffixed pod names, and `DeferCleanup` for both pods and taint removal
3. Replace `framework.ExpectError(err)` on the non-tolerated pod's poll with `Consistently` asserting the pod phase is NOT Running over `PollingTimeoutShort`

### SP-2: [migrate] WaitForFirstConsumer StatefulSet with PVC

**Old**: `It("Wait for Statefulset to become ready", ...)` in `waitforfirstconsumer.go`
**Acceptance**: Test creates a StatefulSet with a WaitForFirstConsumer PVC template and verifies it reaches 1 ready replica.
**Steps**:
1. Add a second `Describe` (or `Context` within the same file) in `e2e-next/test_scheduler/test_waitforfirstconsumer.go`
2. Discover a WaitForFirstConsumer StorageClass on the host (using typed client), creating one if needed
3. Create the StatefulSet in vcluster with random-suffixed name, register `DeferCleanup` immediately
4. Poll with `Eventually` for `ReadyReplicas == 1` using `constants.PollingTimeoutLong`

### SP-3: [cleanup] Remove migrated old test files

**Old**: All `It` blocks in `test/e2e_scheduler/scheduler/scheduler.go` and `test/e2e_scheduler/scheduler/waitforfirstconsumer.go`
**Acceptance**: The two old test files and their directory are deleted. The blank import `_ "github.com/loft-sh/vcluster/test/e2e_scheduler/scheduler"` is removed from `test/e2e_scheduler/e2e_scheduler_suite_test.go`. If no scheduler-specific `It` blocks remain in the suite (other imports are for sibling e2e tests, not scheduler), the e2e_scheduler suite continues to run but without the scheduler package.
**Steps**:
1. Delete `test/e2e_scheduler/scheduler/scheduler.go` and `test/e2e_scheduler/scheduler/waitforfirstconsumer.go`
2. Delete the `test/e2e_scheduler/scheduler/` directory
3. Remove the blank import `_ "github.com/loft-sh/vcluster/test/e2e_scheduler/scheduler"` from `test/e2e_scheduler/e2e_scheduler_suite_test.go`

## Helper Consolidation

**Inline helper scan results:**

| Function | File | Purpose |
|---|---|---|
| `endpointIPs(addrs []corev1.EndpointAddress) []string` | `e2e-next/test_core/sync/test_servicesync.go` | Extracts/sorts IPs from endpoints |
| `int32Ref(i int32) *int32` | `test/e2e_scheduler/scheduler/waitforfirstconsumer.go` (old) | Pointer helper for int32 |

**Decisions:**
- `endpointIPs`: Not needed by scheduler tests. No consolidation needed.
- `int32Ref`: Trivial pointer helper. Use `ptr.To[int32](1)` from `k8s.io/utils/ptr` instead — no helper needed.

No consolidation needed.

## Structure

```
Describe("Virtual scheduler taint/toleration scheduling",
    labels.Scheduler,
    cluster.Use(clusters.SchedulerVCluster),
    cluster.Use(clusters.HostCluster))
  BeforeEach: retrieve hostClient, vClusterClient
  It("schedules a pod with matching toleration and blocks a pod without")

Describe("WaitForFirstConsumer StatefulSet scheduling",
    labels.Scheduler, labels.Storage,
    cluster.Use(clusters.SchedulerVCluster),
    cluster.Use(clusters.HostCluster))
  BeforeEach: retrieve hostClient, vClusterClient
  It("creates a StatefulSet with WaitForFirstConsumer PVC and becomes ready")
```

## Design Decisions

1. **Separate files, same package**: The two old test files cover distinct behaviors (taint scheduling vs. PVC binding). Each gets its own file in `test_scheduler/` for clarity: `test_scheduler.go` and `test_waitforfirstconsumer.go`.

2. **New cluster definition required**: No existing vcluster in `e2e-next/clusters/` has virtual scheduler enabled. A new `SchedulerVCluster` with the scheduler YAML is needed. The `NodesVCluster` syncs nodes but doesn't enable the virtual scheduler.

3. **New label `Scheduler`**: The scheduler tests are a distinct feature area. Adding `labels.Scheduler` allows targeted test runs during development (`just -f Justfile.agent test "scheduler"`).

4. **No `labels.PR`**: The old `test/e2e_scheduler` is included in the CI matrix via the `ls -d ./test/e2e*` glob, so it runs on every PR. However, in the e2e-next framework the PR label should only be added once we're confident the test is stable and the scheduler vcluster bootstraps reliably. The implementing agent should add `labels.PR` after confirming green runs.

5. **No `Ordered` needed**: The taint test is a single `It` block that manages its own lifecycle (add taints → create pods → verify → cleanup). The WaitForFirstConsumer test is also a single `It` block. No sequential dependencies between specs exist.

6. **`Consistently` for negative scheduling assertion**: The old test uses `PollUntilContextTimeout` and asserts it errors (timeout). This conflates "pod never ran" with "poll had an error." In the new test, use `Consistently` over `PollingTimeoutShort` (20s) to verify the pod stays in Pending phase, which is more precise and descriptive.

7. **WaitForFirstConsumer StorageClass discovery uses typed client**: The old test uses `f.HostCRClient` (controller-runtime `client.Client`). The e2e-next pattern prefers typed clients from `cluster.KubeClientFrom`. Use `hostClient.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})` instead.

8. **Taint cleanup via DeferCleanup**: The old test cleans up taints inline at the end. The new test should register a `DeferCleanup` that removes added taints from all virtual nodes, ensuring cleanup even on test failure.

9. **Setup matrix**:

| Setup Step | SP-1 (taints) | SP-2 (WaitForFirstConsumer) |
|---|---|---|
| Get hostClient | Y | Y |
| Get vClusterClient | Y | Y |
| List virtual nodes | Y | N |
| Discover StorageClass | N | Y |

Result: `BeforeEach` retrieves both clients (shared). Everything else is spec-specific (inline).

10. **Existing setup pattern for scheduler label**: No existing tests use `labels.Scheduler` — this is the first. The setup pattern follows the same `BeforeEach` + client retrieval as `test_core/sync/test_node.go` (which also uses `cluster.Use(clusters.NodesVCluster)` and `cluster.Use(clusters.HostCluster)`).

## Allowed Directories

- e2e-next/clusters
- e2e-next/labels
- e2e-next/test_scheduler
- e2e-next
- test/e2e_scheduler/scheduler
- test/e2e_scheduler

## Validation

```bash
just -f Justfile.agent bootstrap "scheduler"
just -f Justfile.agent test "scheduler"
```

Then for focused runs during development:
```bash
just -f Justfile.agent test-focus "scheduler" "taint/toleration"
just -f Justfile.agent test-focus "scheduler" "WaitForFirstConsumer"
```

Then verify against `.claude/rules/e2e-quality-checklist.md` (auto-loaded).
