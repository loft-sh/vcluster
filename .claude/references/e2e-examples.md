# E2E Test Examples — Annotated Patterns

Concrete excerpts from production tests in `e2e-next/`. These are ground truth — read them before writing new tests.

---

## 1. DeferCleanup Pattern

Rendered YAML cleanup and vCluster teardown are handled by `setup/lazyvcluster.LazyVCluster` - you do not register them by hand. `SynchronizedBeforeSuite` only provisions the host kind cluster now.

```go
// suite_myfeature_test.go — lazy helper owns YAML cleanup + vCluster teardown
func suiteMyFeature() {
    Describe("myfeature-vcluster", labels.MyFeature, Ordered,
        cluster.Use(clusters.HostCluster),
        func() {
            BeforeAll(func(ctx context.Context) context.Context {
                return lazyvcluster.LazyVCluster(ctx, myFeatureName, myFeatureYAML)
            })
            // specs...
        },
    )
}
```

Inside `It` blocks - register cleanup immediately after resource creation, before any further assertions:

```go
By("creating source namespace on host", func() {
    _, err := hostClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
        ObjectMeta: metav1.ObjectMeta{Name: fromNS},
    }, metav1.CreateOptions{})
    Expect(err).NotTo(HaveOccurred())
    DeferCleanup(func(ctx context.Context) {
        Expect(hostClient.CoreV1().Namespaces().Delete(ctx, fromNS, metav1.DeleteOptions{})).To(Succeed())
        Eventually(func(g Gomega) {
            _, err := hostClient.CoreV1().Namespaces().Get(ctx, fromNS, metav1.GetOptions{})
            g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
        }).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
    })
})
```

---

## 2. Eventually + Polling

Source: `e2e-next/test_core/sync/test_servicesync.go`, `e2e-next/test_deploy/test_helm_charts.go`

Use the `g Gomega` parameter inside `Eventually`. Bare `Expect()` inside `Eventually` panics instead of retrying. Include failure context for debuggability.

**Waiting for a resource to appear:**

```go
var toService *corev1.Service
By("waiting for the replicated service to appear in vcluster", func() {
    Eventually(func(g Gomega) {
        toService, err = vClusterClient.CoreV1().Services(toNS).Get(ctx, toName, metav1.GetOptions{})
        g.Expect(err).NotTo(HaveOccurred())
    }).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
})
```

**Waiting for a resource to be deleted:**

```go
By("waiting for the replicated service to be removed from vcluster", func() {
    Eventually(func(g Gomega) {
        _, err := vClusterClient.CoreV1().Services(toNS).Get(ctx, toName, metav1.GetOptions{})
        g.Expect(kerrors.IsNotFound(err)).To(BeTrue(), "replicated service should be deleted after source is gone")
    }).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
})
```

**Checking deploy status with descriptive failure messages:**

```go
Eventually(func(g Gomega) {
    cm, err := vClusterClient.CoreV1().ConfigMaps(deploy.VClusterDeployConfigMapNamespace).
        Get(ctx, deploy.VClusterDeployConfigMap, metav1.GetOptions{})
    g.Expect(err).NotTo(HaveOccurred(), "Deploy configmap should exist")
    status := deploy.ParseStatus(cm)
    g.Expect(status.Charts).To(HaveLen(2), "Should have 2 charts configured")
    for _, chart := range status.Charts {
        g.Expect(chart.Phase).To(
            Equal(string(deploy.StatusSuccess)),
            fmt.Sprintf("Chart %s is not in Success phase, got phase=%s, reason=%s, message=%s",
                chart.Name, chart.Phase, chart.Reason, chart.Message))
    }
}).
    WithPolling(constants.PollingInterval).
    WithTimeout(constants.PollingTimeout).
    Should(Succeed(), "Both charts should be successfully deployed")
```

**Returning a value from Eventually for direct assertion:**

```go
Eventually(func(g Gomega) []appsv1.Deployment {
    deployList, err := vClusterClient.AppsV1().Deployments(ChartOCINamespace).List(ctx, metav1.ListOptions{
        LabelSelector: k8slabels.SelectorFromSet(HelmOCIDeploymentLabels).String(),
    })
    g.Expect(err).NotTo(HaveOccurred(), "Should be able to list deployments")
    return deployList.Items
}).
    WithPolling(constants.PollingInterval).
    WithTimeout(constants.PollingTimeout).
    Should(HaveLen(1), "Should have exactly one fluent-bit deployment")
```

---

## 3. Cluster Client Usage

Source: `e2e-next/test_core/sync/test_node.go`

Obtain host and vcluster clients from context. Use `cluster.KubeClientFrom` with the host cluster name for the host client, and `cluster.CurrentKubeClientFrom` for the current vcluster client.

```go
var _ = Describe("Node sync",
    labels.Core,
    labels.Sync,
    cluster.Use(clusters.NodesVCluster),
    cluster.Use(clusters.HostCluster),
    func() {
        var (
            hostClient     kubernetes.Interface
            vClusterClient kubernetes.Interface
        )

        BeforeEach(func(ctx context.Context) {
            hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
            Expect(hostClient).NotTo(BeNil())
            vClusterClient = cluster.CurrentKubeClientFrom(ctx)
            Expect(vClusterClient).NotTo(BeNil())
        })

        It("Sync nodes using label selector", func(ctx context.Context) {
            hostname := constants.GetHostClusterName() + "-control-plane"
            Eventually(func(g Gomega) {
                hostNodes, err := hostClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
                g.Expect(err).NotTo(HaveOccurred(), "Failed to list host nodes")

                virtualNodes, err := vClusterClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
                g.Expect(err).NotTo(HaveOccurred(), "Failed to list virtual nodes")
                g.Expect(virtualNodes.Items).ToNot(BeEmpty(), "Virtual nodes list should not be empty")

                // ... assertions comparing host and virtual nodes
            }).
                WithPolling(constants.PollingInterval).
                WithTimeout(constants.PollingTimeout).
                Should(Succeed(), "Node sync should work correctly")
        })
    })
```

---

## 4. Ordered Usage

Source: `test/e2e/snapshot/snapshot.go` (old suite — pattern, not import paths)

Use `Ordered` when specs form a **lifecycle sequence** where later specs depend on state produced by earlier specs, **and** shared setup is genuinely expensive (resource creation, deployment readiness, data provisioning). The snapshot-with-volumes test is the canonical example: `BeforeAll` provisions a namespace, a PVC with written data, and several Kubernetes resources, then each `It` block advances through create → verify → delete → restore → verify-restored.

```go
Describe("controller-based snapshot with volumes", Ordered, func() {
    const (
        controllerTestNamespaceName = "volume-snapshots-test"
        snapshotPath                = "container:///snapshot-data/" + controllerTestNamespaceName + ".tar.gz"
        pvcToRestoreName            = "test-pvc-restore"
        testFileName                = controllerTestNamespaceName + ".txt"
        pvcData                     = "Hello " + controllerTestNamespaceName
    )

    // BeforeAll — expensive: creates namespace, PVC with data written via a pod,
    // plus Deployments, Services, ConfigMaps, and Secrets.
    BeforeAll(func(ctx context.Context) {
        f = framework.DefaultFramework
        deployTestNamespace(controllerTestNamespaceName)
        // Creates PVC, spins up a writer pod, waits for pod readiness, writes data
        createPVCWithData(ctx, f.VClusterClient, controllerTestNamespaceName,
            pvcToRestoreName, testFileName, pvcData)
        // Creates Deployment (nginx), Service, ConfigMap, Secret — waits for deploy ready
        deployTestResources(controllerTestNamespaceName, true)
    })

    // Spec 1: triggers snapshot creation — later specs verify the snapshot exists
    It("Creates the snapshot request", func() {
        createSnapshot(f, true, snapshotPath, true)
    })

    // Spec 2: depends on spec 1 — polls until snapshot completes
    It("Creates the snapshot", func(ctx context.Context) {
        waitForSnapshotToBeCreated(ctx, f)
    })

    // Spec 3: depends on spec 2 — verifies VolumeSnapshot resources were cleaned up
    It("Doesn't contain VolumeSnapshot and VolumeSnapshotContent", func(ctx context.Context) {
        // ... verifies cleanup of temporary snapshot objects
    })

    // Spec 4: depends on spec 2 — deletes PVC so restore can recreate it
    It("Deletes the PVC with test data", func(ctx context.Context) {
        deletePVC(ctx, f, controllerTestNamespaceName, pvcToRestoreName)
    })

    // checkTestResources embeds two It blocks:
    //   "Verify if only the resources in snapshot are available after restore"
    //   "Verify if deleted resources are recreated after restore"
    // Both depend on prior specs having created and snapshotted resources.
    checkTestResources(controllerTestNamespaceName, true, snapshotPath)

    // Spec 7: depends on PVC being restored (without data) by checkTestResources
    It("restores vCluster with volumes", func(ctx context.Context) {
        deletePVC(ctx, f, controllerTestNamespaceName, pvcToRestoreName)
        restoreVCluster(ctx, f, snapshotPath, true, true)
    })

    // Spec 8: depends on spec 7 — verifies PVC is bound after volume restore
    It("has the restored PVC which is bound", func(ctx context.Context) {
        Eventually(func(g Gomega, ctx context.Context) corev1.PersistentVolumeClaimPhase {
            restoredPVC, err := f.VClusterClient.CoreV1().PersistentVolumeClaims(
                controllerTestNamespaceName).Get(ctx, pvcToRestoreName, metav1.GetOptions{})
            g.Expect(err).NotTo(HaveOccurred())
            return restoredPVC.Status.Phase
        }).WithContext(ctx).
            WithPolling(framework.PollInterval).
            WithTimeout(framework.PollTimeoutLong).
            Should(Equal(corev1.ClaimBound))
    })

    // Spec 9: depends on spec 8 — verifies data survived the snapshot/restore cycle
    It("has the restored PVC with data from the volume snapshot", func(ctx context.Context) {
        checkPVCData(ctx, f.VClusterClient, controllerTestNamespaceName,
            pvcToRestoreName, testFileName, pvcData)
    })

    AfterAll(func(ctx context.Context) {
        deletePVC(ctx, f, controllerTestNamespaceName, pvcToRestoreName)
        cleanUpTestResources(ctx, true, controllerTestNamespaceName)
    })
})
```

**Why `Ordered` is justified here:** Each spec advances a lifecycle (provision → snapshot → cleanup → restore → verify) where later specs depend on side effects from earlier ones. The `BeforeAll` is genuinely expensive — it creates a namespace, provisions a PVC, spins up a writer pod to populate data, and deploys multiple resources with readiness checks. Contrast with the `BeforeEach` examples below, where specs are independent and only need client references.

---

## 5. BeforeEach Pattern

Source: `e2e-next/test_deploy/test_helm_charts.go`, `e2e-next/test_deploy/test_init_manifests.go`

Use `BeforeEach` when specs are independent and only need shared client setup. Each spec operates on pre-existing state (deployed charts, init manifests) and doesn't modify shared resources.

```go
var _ = Describe("Helm charts (regular and OCI) are synced and applied as expected",
    labels.Deploy,
    cluster.Use(clusters.HelmChartsVCluster),
    func() {
        var vClusterClient kubernetes.Interface

        BeforeEach(func(ctx context.Context) {
            vClusterClient = cluster.CurrentKubeClientFrom(ctx)
            Expect(vClusterClient).NotTo(BeNil())
        })

        It("Test if configmap for both charts gets applied", func(ctx context.Context) {
            Eventually(func(g Gomega) {
                // ... check deploy configmap status
            }).
                WithPolling(constants.PollingInterval).
                WithTimeout(constants.PollingTimeout).
                Should(Succeed(), "Both charts should be successfully deployed")
        })

        It("Test nginx release secret existence in vcluster (regular chart)", func(ctx context.Context) {
            // ... independent verification
        })
    })
```

**BeforeEach returning context** (from `test_init_manifests.go`):

```go
var _ = Describe("Init manifests are synced and applied as expected",
    labels.Deploy,
    cluster.Use(clusters.InitManifestsVCluster),
    func() {
        var (
            vClusterName   = clusters.InitManifestsVClusterName
            vClusterClient kubernetes.Interface
        )

        BeforeEach(func(ctx context.Context) {
            vClusterClient = cluster.CurrentKubeClientFrom(ctx)
            Expect(vClusterClient).NotTo(BeNil())
        })

        It("Test if manifest template is synced with the vcluster", func(ctx context.Context) {
            Eventually(func(g Gomega) {
                manifest, err := vClusterClient.CoreV1().ConfigMaps(TestManifestNamespace).
                    Get(ctx, TestManifestName2, metav1.GetOptions{})
                g.Expect(err).NotTo(HaveOccurred(), "ConfigMap should exist")
                g.Expect(manifest.Data["foo"]).To(Equal(vClusterName),
                    "ConfigMap foo value should equal vcluster name")
            }).
                WithPolling(constants.PollingInterval).
                WithTimeout(constants.PollingTimeout).
                Should(Succeed(), "Manifest template should be synced")
        })
    },
)
```
