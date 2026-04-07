package snapshot

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	snapshotsv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/clientset/versioned"
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	loftlog "github.com/loft-sh/log"
	connectcmd "github.com/loft-sh/vcluster/cmd/vclusterctl/cmd"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	vclusterconfig "github.com/loft-sh/vcluster/pkg/config"
	pkgconstants "github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/helm"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/snapshot/volumes"
	"github.com/loft-sh/vcluster/pkg/util/random"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/ptr"
)

// snapshotCtx holds shared state for a snapshot test group.
type snapshotCtx struct {
	hostClient     kubernetes.Interface
	vClusterClient kubernetes.Interface
	vClusterName   string
	vClusterNS     string
}

func newSnapshotCtx(ctx context.Context) *snapshotCtx {
	GinkgoHelper()
	s := &snapshotCtx{}
	s.hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
	Expect(s.hostClient).NotTo(BeNil())
	s.vClusterClient = cluster.CurrentKubeClientFrom(ctx)
	Expect(s.vClusterClient).NotTo(BeNil())
	s.vClusterName = cluster.CurrentClusterNameFrom(ctx)
	s.vClusterNS = "vcluster-" + s.vClusterName
	return s
}

func (s *snapshotCtx) refreshClient(ctx context.Context) {
	GinkgoHelper()
	By("Reconnecting to the vCluster after restore", func() {
		tmpFile, err := os.CreateTemp("", "vcluster-restore-kubeconfig-*")
		Expect(err).To(Succeed())
		tmpFile.Close()
		DeferCleanup(func(_ context.Context) { os.Remove(tmpFile.Name()) })

		// Use ConnectCmd programmatically (same as old framework's RefreshVirtualClient).
		// The CLI subprocess approach hangs in CI because the background proxy is dead.
		connectCmd := connectcmd.ConnectCmd{
			CobraCmd: &cobra.Command{},
			Log:      loftlog.Discard,
			GlobalFlags: &flags.GlobalFlags{
				Namespace: s.vClusterNS,
			},
			ConnectOptions: cli.ConnectOptions{
				KubeConfig:           tmpFile.Name(),
				BackgroundProxy:      true,
				BackgroundProxyImage: constants.GetVClusterImage(),
			},
		}
		err = connectCmd.Run(ctx, []string{s.vClusterName})
		Expect(err).To(Succeed(), "vcluster connect failed after restore")

		Eventually(func(g Gomega) {
			data, err := os.ReadFile(tmpFile.Name())
			g.Expect(err).To(Succeed())
			g.Expect(data).NotTo(BeEmpty(), "kubeconfig file is empty")

			restConfig, err := clientcmd.RESTConfigFromKubeConfig(data)
			g.Expect(err).To(Succeed())

			newClient, err := kubernetes.NewForConfig(restConfig)
			g.Expect(err).To(Succeed())

			_, err = newClient.CoreV1().ServiceAccounts("default").Get(ctx, "default", metav1.GetOptions{})
			g.Expect(err).To(Succeed(), "vCluster not yet available after restore")

			s.vClusterClient = newClient
		}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
	})
}

func (s *snapshotCtx) deployTestResources(ctx context.Context, testNS string) (
	configMapToRestore *corev1.ConfigMap,
	configMapToDelete *corev1.ConfigMap,
	secretToRestore *corev1.Secret,
	secretToDelete *corev1.Secret,
	deploymentToRestore *appsv1.Deployment,
	serviceToRestore *corev1.Service,
) {
	GinkgoHelper()
	_, err := s.vClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: testNS},
	}, metav1.CreateOptions{})
	Expect(err).To(Succeed())

	configMapToRestore = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "configmap-restore", Namespace: testNS, Labels: map[string]string{"snapshot": "restore"}},
		Data:       map[string]string{"somekey": "somevalue"},
	}
	configMapToDelete = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "configmap-delete", Namespace: testNS, Labels: map[string]string{"snapshot": "delete"}},
		Data:       map[string]string{"somesome": "somevalue"},
	}
	secretToRestore = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "secret-restore", Namespace: testNS, Labels: map[string]string{"snapshot": "restore"}},
		Data:       map[string][]byte{"BOO_BAR": []byte("hello-world")},
	}
	secretToDelete = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "secret-delete", Namespace: testNS, Labels: map[string]string{"snapshot": "delete"}},
		Data:       map[string][]byte{"ANOTHER_ENV": []byte("another-hello-world")},
	}
	deploymentToRestore = &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "deployment-restore", Namespace: testNS, Labels: map[string]string{"snapshot": "restore"}},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To(int32(1)),
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"snapshot": "restore"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"snapshot": "restore"}},
				Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "example-container", Image: "nginx:1.25.0", Ports: []corev1.ContainerPort{{ContainerPort: 80}}}}},
			},
		},
	}
	serviceToRestore = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "snapshot-restore", Namespace: testNS, Labels: map[string]string{"snapshot": "restore"}},
		Spec:       corev1.ServiceSpec{Ports: []corev1.ServicePort{{Name: "https", Port: 443}}, Type: corev1.ServiceTypeClusterIP},
	}

	_, err = s.vClusterClient.CoreV1().Services(testNS).Create(ctx, serviceToRestore, metav1.CreateOptions{})
	Expect(err).To(Succeed())
	_, err = s.vClusterClient.CoreV1().ConfigMaps(testNS).Create(ctx, configMapToRestore, metav1.CreateOptions{})
	Expect(err).To(Succeed())
	_, err = s.vClusterClient.CoreV1().Secrets(testNS).Create(ctx, secretToRestore, metav1.CreateOptions{})
	Expect(err).To(Succeed())
	_, err = s.vClusterClient.AppsV1().Deployments(testNS).Create(ctx, deploymentToRestore, metav1.CreateOptions{})
	Expect(err).To(Succeed())

	return configMapToRestore, configMapToDelete, secretToRestore, secretToDelete, deploymentToRestore, serviceToRestore
}

// SnapshotAllSpec registers snapshot and restore tests.
// Snapshot operations on one vCluster interfere with each other (shared configmaps/secrets),
// so they must run sequentially on the same vCluster.
func SnapshotAllSpec() {
	var s snapshotCtx
	Describe("Snapshot and restore",
		Ordered,
		labels.Snapshots,
		func() {
			BeforeAll(func(ctx context.Context) {
				s = *newSnapshotCtx(ctx)
			})

			describeSnapshotRestore(&s)
			describeSnapshotCanceling(&s)
			describeSnapshotDeletion(&s)
		},
	)
}

func describeSnapshotRestore(s *snapshotCtx) {
	// Ordered: create snapshot -> restore & verify resources exist -> restore & verify deleted resources recreated.
	// Each spec depends on the snapshot created in spec 1 and the restore state from the prior spec.
	Describe("controller-based snapshot without volumes", Ordered, func() {
		var (
			testNS       string
			snapshotPath = "container:///snapshot-data/snapshot.tar.gz"
		)
		var (
			configMapToRestore *corev1.ConfigMap
			configMapToDelete  *corev1.ConfigMap
			secretToRestore    *corev1.Secret
			secretToDelete     *corev1.Secret
		)

		BeforeAll(func(ctx context.Context) {
			testNS = "ctrl-snapshot-" + random.String(6)
			// Clean slate - remove any leftover snapshot artifacts from prior groups
			cleanupAllSnapshotArtifacts(ctx, s.hostClient, s.vClusterNS)
			var cmr *corev1.ConfigMap
			var cmd *corev1.ConfigMap
			var sr *corev1.Secret
			var sd *corev1.Secret
			cmr, cmd, sr, sd, _, _ = s.deployTestResources(ctx, testNS)
			configMapToRestore = cmr
			configMapToDelete = cmd
			secretToRestore = sr
			secretToDelete = sd
		})

		It("Creates the snapshot", func(ctx context.Context) {
			createSnapshot(s.vClusterName, s.vClusterNS, true, snapshotPath, false)
			waitForSnapshotToBeCreated(ctx, s.hostClient, s.vClusterNS)
		})

		It("Verifies only snapshot resources exist after restore", func(ctx context.Context) {
			_, err := s.vClusterClient.CoreV1().ConfigMaps(testNS).Create(ctx, configMapToDelete, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			_, err = s.vClusterClient.CoreV1().Secrets(testNS).Create(ctx, secretToDelete, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			svcCreated, err := s.vClusterClient.CoreV1().Services(testNS).Create(ctx, &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Name: "snapshot-delete", Namespace: testNS, Labels: map[string]string{"snapshot": "delete"}},
				Spec:       corev1.ServiceSpec{Ports: []corev1.ServicePort{{Name: "http", Port: 80}}, Type: corev1.ServiceTypeClusterIP},
			}, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			oldRV := svcCreated.ResourceVersion

			restoreVCluster(ctx, s.hostClient, s.vClusterName, s.vClusterNS, snapshotPath, true, false)
			s.refreshClient(ctx)

			// Verify pre-snapshot resources exist
			configmaps, err := s.vClusterClient.CoreV1().ConfigMaps(testNS).List(ctx, metav1.ListOptions{LabelSelector: "snapshot=restore"})
			Expect(err).To(Succeed())
			Expect(configmaps.Items).To(HaveLen(1))
			Expect(configmaps.Items[0].Data).To(Equal(configMapToRestore.Data))
			newRV, _ := strconv.ParseInt(configmaps.Items[0].ResourceVersion, 10, 64)
			oldRVi, _ := strconv.ParseInt(oldRV, 10, 64)
			Expect(newRV).To(BeNumerically(">", oldRVi))

			secrets, err := s.vClusterClient.CoreV1().Secrets(testNS).List(ctx, metav1.ListOptions{LabelSelector: "snapshot=restore"})
			Expect(err).To(Succeed())
			Expect(secrets.Items).To(HaveLen(1))
			Expect(secrets.Items[0].Data).To(Equal(secretToRestore.Data))

			deps, err := s.vClusterClient.AppsV1().Deployments(testNS).List(ctx, metav1.ListOptions{LabelSelector: "snapshot=restore"})
			Expect(err).To(Succeed())
			Expect(deps.Items).To(HaveLen(1))

			// Verify post-snapshot resources are gone
			Eventually(func(g Gomega) {
				cms, err := s.vClusterClient.CoreV1().ConfigMaps(testNS).List(ctx, metav1.ListOptions{LabelSelector: "snapshot=delete"})
				g.Expect(err).To(Succeed())
				g.Expect(cms.Items).To(BeEmpty())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			Eventually(func(g Gomega) {
				secs, err := s.vClusterClient.CoreV1().Secrets(testNS).List(ctx, metav1.ListOptions{LabelSelector: "snapshot=delete"})
				g.Expect(err).To(Succeed())
				g.Expect(secs.Items).To(BeEmpty())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			Eventually(func(g Gomega) {
				svcs, err := s.vClusterClient.CoreV1().Services(testNS).List(ctx, metav1.ListOptions{LabelSelector: "snapshot=delete"})
				g.Expect(err).To(Succeed())
				g.Expect(svcs.Items).To(BeEmpty())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
		})

		It("Verifies deleted resources are recreated after restore", func(ctx context.Context) {
			err := s.vClusterClient.CoreV1().ConfigMaps(testNS).Delete(ctx, configMapToRestore.Name, metav1.DeleteOptions{})
			Expect(err).To(Succeed())
			err = s.vClusterClient.CoreV1().Secrets(testNS).Delete(ctx, secretToRestore.Name, metav1.DeleteOptions{})
			Expect(err).To(Succeed())

			restoreVCluster(ctx, s.hostClient, s.vClusterName, s.vClusterNS, snapshotPath, true, false)
			s.refreshClient(ctx)

			Eventually(func(g Gomega) {
				cms, err := s.vClusterClient.CoreV1().ConfigMaps(testNS).List(ctx, metav1.ListOptions{LabelSelector: "snapshot=restore"})
				g.Expect(err).To(Succeed())
				g.Expect(cms.Items).To(HaveLen(1))
				g.Expect(cms.Items[0].Data).To(Equal(configMapToRestore.Data))
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			Eventually(func(g Gomega) {
				secs, err := s.vClusterClient.CoreV1().Secrets(testNS).List(ctx, metav1.ListOptions{LabelSelector: "snapshot=restore"})
				g.Expect(err).To(Succeed())
				g.Expect(secs.Items).To(HaveLen(1))
				g.Expect(secs.Items[0].Data).To(Equal(secretToRestore.Data))
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
		})

		AfterAll(func(ctx context.Context) {
			deleteSnapshotRequestConfigMaps(ctx, s.hostClient, s.vClusterNS)
		})
	})

	// Ordered: create snapshot with volumes -> verify VolumeSnapshot cleanup -> delete PVC -> restore with volumes -> verify PVC bound + data.
	// Each spec depends on snapshot/restore state from prior specs.
	Describe("controller-based snapshot with volumes", Ordered, func() {
		var (
			testNS, snapshotPath, testFileName, pvcData string
			pvcToRestoreName                            = "test-pvc-restore"
		)

		BeforeAll(func(ctx context.Context) {
			testNS = "vol-snapshot-" + random.String(6)
			snapshotPath = "container:///snapshot-data/" + testNS + ".tar.gz"
			testFileName = testNS + ".txt"
			pvcData = "Hello " + testNS
			cleanupAllSnapshotArtifacts(ctx, s.hostClient, s.vClusterNS)

			// Clean up any VolumeSnapshots left by prior Ordered groups (canceling, deletion)
			// so this group starts with a clean slate.
			hostRestConfig := cluster.From(ctx, constants.GetHostClusterName()).KubernetesRestConfig()
			vsClient, err := snapshotsv1.NewForConfig(hostRestConfig)
			Expect(err).To(Succeed())
			err = vsClient.SnapshotV1().VolumeSnapshots(s.vClusterNS).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{})
			Expect(err).To(Succeed())
			err = vsClient.SnapshotV1().VolumeSnapshotContents().DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{})
			Expect(err).To(Succeed())
			Eventually(func(g Gomega) {
				vs, err := vsClient.SnapshotV1().VolumeSnapshots(s.vClusterNS).List(ctx, metav1.ListOptions{})
				g.Expect(err).To(Succeed())
				g.Expect(vs.Items).To(BeEmpty(), "waiting for VolumeSnapshots cleanup")
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())

			s.deployTestResources(ctx, testNS)
			createPVCWithData(ctx, s.vClusterClient, testNS, pvcToRestoreName, testFileName, pvcData)
		})

		It("Creates the snapshot", func(ctx context.Context) {
			createSnapshot(s.vClusterName, s.vClusterNS, true, snapshotPath, true)
			waitForSnapshotToBeCreated(ctx, s.hostClient, s.vClusterNS)
		})

		It("Verifies VolumeSnapshots are cleaned up", func(ctx context.Context) {
			vClusterRelease, err := helm.NewSecrets(s.hostClient).Get(ctx, s.vClusterName, s.vClusterNS)
			Expect(err).To(Succeed())
			vConfigValues, err := yaml.Marshal(vClusterRelease.Config)
			Expect(err).To(Succeed())
			vClusterConfig, err := vclusterconfig.ParseConfigBytes(vConfigValues, s.vClusterName, nil)
			Expect(err).To(Succeed())

			var restConfig *rest.Config
			var vsNS string
			if vClusterConfig.PrivateNodes.Enabled {
				currentClusterName := cluster.CurrentClusterNameFrom(ctx)
				restConfig = cluster.From(ctx, currentClusterName).KubernetesRestConfig()
				vsNS = testNS
			} else {
				restConfig = cluster.From(ctx, constants.GetHostClusterName()).KubernetesRestConfig()
				vsNS = s.vClusterNS
			}
			snapshotClient, err := snapshotsv1.NewForConfig(restConfig)
			Expect(err).To(Succeed())

			// After a snapshot-restore cycle, the controller should clean up all
			// VolumeSnapshots it created. Use Eventually because cleanup is async.
			Eventually(func(g Gomega) {
				vs, err := snapshotClient.SnapshotV1().VolumeSnapshots(vsNS).List(ctx, metav1.ListOptions{
					LabelSelector: "vcluster.loft.sh/persistentvolumeclaim",
				})
				g.Expect(err).To(Succeed())
				g.Expect(vs.Items).To(BeEmpty(), "VolumeSnapshots still exist: %d remaining", len(vs.Items))
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())

			Eventually(func(g Gomega) {
				vsc, err := snapshotClient.SnapshotV1().VolumeSnapshotContents().List(ctx, metav1.ListOptions{})
				g.Expect(err).To(Succeed())
				g.Expect(vsc.Items).To(BeEmpty(), "VolumeSnapshotContents still exist: %d remaining", len(vsc.Items))
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
		})

		It("Restores vCluster with volumes and verifies PVC data", func(ctx context.Context) {
			deletePVC(ctx, s.vClusterClient, s.hostClient, s.vClusterName, s.vClusterNS, testNS, pvcToRestoreName)
			// PVC restored without data in previous specs; delete again for proper restore
			deletePVC(ctx, s.vClusterClient, s.hostClient, s.vClusterName, s.vClusterNS, testNS, pvcToRestoreName)
			restoreVCluster(ctx, s.hostClient, s.vClusterName, s.vClusterNS, snapshotPath, true, true)
			s.refreshClient(ctx)

			Eventually(func(g Gomega) {
				pvc, err := s.vClusterClient.CoreV1().PersistentVolumeClaims(testNS).Get(ctx, pvcToRestoreName, metav1.GetOptions{})
				g.Expect(err).To(Succeed())
				g.Expect(pvc.Status.Phase).To(Equal(corev1.ClaimBound))
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())

			checkPVCData(ctx, s.vClusterClient, testNS, pvcToRestoreName, testFileName, pvcData)
		})

		AfterAll(func(ctx context.Context) {
			deletePVC(ctx, s.vClusterClient, s.hostClient, s.vClusterName, s.vClusterNS, testNS, pvcToRestoreName)
			deleteSnapshotRequestConfigMaps(ctx, s.hostClient, s.vClusterNS)
		})
	})
}

func describeSnapshotCanceling(s *snapshotCtx) {
	var (
		testNS       string
		snapshotPath string
	)
	const (
		appCount  = 3
		appPrefix = "test-app-"
	)

	// Ordered: BeforeAll creates 2 snapshots back-to-back -> spec 1 verifies 2 requests exist ->
	// spec 2 verifies first was canceled -> spec 3 verifies second completed.
	Describe("Snapshot canceling", Ordered, func() {
		BeforeAll(func(ctx context.Context) {
			testNS = "snap-cancel-" + random.String(6)
			snapshotPath = "container:///snapshot-data/" + testNS + ".tar.gz"
			cleanupAllSnapshotArtifacts(ctx, s.hostClient, s.vClusterNS)
			_, err := s.vClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: testNS},
			}, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			for i := range appCount {
				createAppWithPVC(ctx, s.vClusterClient, testNS, fmt.Sprintf("%s%d", appPrefix, i))
			}
			Eventually(func(g Gomega) {
				for i := range appCount {
					dep, err := s.vClusterClient.AppsV1().Deployments(testNS).Get(ctx, fmt.Sprintf("%s%d", appPrefix, i), metav1.GetOptions{})
					g.Expect(err).To(Succeed())
					g.Expect(dep.Status.AvailableReplicas).To(Equal(int32(1)))
				}
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())

			createSnapshot(s.vClusterName, s.vClusterNS, true, snapshotPath, true)
			// Brief pause to ensure the first snapshot request is registered before
			// the second one arrives - tests the cancellation path where a new
			// snapshot supersedes an in-progress one.
			time.Sleep(time.Second)
			createSnapshot(s.vClusterName, s.vClusterNS, true, snapshotPath, true)
		})

		It("Has 2 snapshot requests", func(ctx context.Context) {
			Eventually(func(g Gomega) {
				cms, err := s.hostClient.CoreV1().ConfigMaps(s.vClusterNS).List(ctx, metav1.ListOptions{
					LabelSelector: pkgconstants.SnapshotRequestLabel,
				})
				g.Expect(err).To(Succeed())
				g.Expect(cms.Items).To(HaveLen(2))
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
		})

		It("Canceled the previous snapshot request", func(ctx context.Context) {
			vClusterRelease, err := helm.NewSecrets(s.hostClient).Get(ctx, s.vClusterName, s.vClusterNS)
			Expect(err).To(Succeed())
			vConfigValues, err := yaml.Marshal(vClusterRelease.Config)
			Expect(err).To(Succeed())
			vClusterConfig, err := vclusterconfig.ParseConfigBytes(vConfigValues, s.vClusterName, nil)
			Expect(err).To(Succeed())

			var restConfig *rest.Config
			var vsNS string
			if vClusterConfig.PrivateNodes.Enabled {
				currentClusterName := cluster.CurrentClusterNameFrom(ctx)
				restConfig = cluster.From(ctx, currentClusterName).KubernetesRestConfig()
				vsNS = testNS
			} else {
				restConfig = cluster.From(ctx, constants.GetHostClusterName()).KubernetesRestConfig()
				vsNS = s.vClusterNS
			}
			snapshotClient, err := snapshotsv1.NewForConfig(restConfig)
			Expect(err).To(Succeed())

			Eventually(func(g Gomega) {
				previousReq, _ := getTwoSnapshotRequests(g, ctx, s.hostClient, s.vClusterNS)
				for pvcName, vsStatus := range previousReq.Status.VolumeSnapshots.Snapshots {
					pvcParts := strings.Split(pvcName, "/")
					g.Expect(pvcParts).To(HaveLen(2))
					vsName := fmt.Sprintf("%s-%s", pvcParts[1], previousReq.Name)
					_, err := snapshotClient.SnapshotV1().VolumeSnapshots(vsNS).Get(ctx, vsName, metav1.GetOptions{})
					g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
					g.Expect(vsStatus.Phase).To(Equal(volumes.RequestPhaseCanceled))
				}
				g.Expect(previousReq.Status.VolumeSnapshots.Phase).To(Equal(volumes.RequestPhaseCanceled))
				g.Expect(previousReq.Status.Phase).To(Equal(snapshot.RequestPhaseCanceled))
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())
		})

		It("Completed the new snapshot request", func(ctx context.Context) {
			Eventually(func(g Gomega) {
				_, newerReq := getTwoSnapshotRequests(g, ctx, s.hostClient, s.vClusterNS)
				for pvcName, vs := range newerReq.Status.VolumeSnapshots.Snapshots {
					g.Expect(vs.Phase).To(Equal(volumes.RequestPhaseCompleted),
						"volume snapshot for PVC %s not completed: %s", pvcName, toJSON(vs))
				}
				g.Expect(newerReq.Status.VolumeSnapshots.Phase).To(Equal(volumes.RequestPhaseCompleted))
				g.Expect(newerReq.Status.Phase).To(Equal(snapshot.RequestPhaseCompleted))
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())
		})

		AfterAll(func(ctx context.Context) {
			err := s.vClusterClient.CoreV1().Namespaces().Delete(ctx, testNS, metav1.DeleteOptions{})
			if !kerrors.IsNotFound(err) {
				Expect(err).To(Succeed())
			}
			deleteSnapshotRequestConfigMaps(ctx, s.hostClient, s.vClusterNS)
		})
	},
	)
}

func describeSnapshotDeletion(s *snapshotCtx) {
	var (
		testNS                    string
		snapshotPath              string
		deleteSnapshotRequestName string
	)
	const (
		appCount  = 3
		appPrefix = "test-app-"
	)

	// Ordered: BeforeAll creates a snapshot -> spec 1 creates a deletion request ->
	// spec 2 verifies the snapshot was deleted.
	Describe("Snapshot deletion", Ordered, func() {
		BeforeAll(func(ctx context.Context) {
			testNS = "snap-delete-" + random.String(6)
			snapshotPath = "container:///snapshot-data/" + testNS + ".tar.gz"
			deleteSnapshotRequestName = "delete-request-" + testNS
			cleanupAllSnapshotArtifacts(ctx, s.hostClient, s.vClusterNS)
			_, err := s.vClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: testNS},
			}, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			for i := range appCount {
				createAppWithPVC(ctx, s.vClusterClient, testNS, fmt.Sprintf("%s%d", appPrefix, i))
			}
			Eventually(func(g Gomega) {
				for i := range appCount {
					dep, err := s.vClusterClient.AppsV1().Deployments(testNS).Get(ctx, fmt.Sprintf("%s%d", appPrefix, i), metav1.GetOptions{})
					g.Expect(err).To(Succeed())
					g.Expect(dep.Status.AvailableReplicas).To(Equal(int32(1)))
				}
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())

			createSnapshot(s.vClusterName, s.vClusterNS, true, snapshotPath, true)
		})

		It("Creates snapshot deletion request", func(ctx context.Context) {
			listOptions := metav1.ListOptions{LabelSelector: pkgconstants.SnapshotRequestLabel}

			var snapshotOptions *snapshot.Options
			Eventually(func(g Gomega) {
				secrets, err := s.hostClient.CoreV1().Secrets(s.vClusterNS).List(ctx, listOptions)
				g.Expect(err).To(Succeed())
				g.Expect(secrets.Items).To(HaveLen(1))
				snapshotOptions, err = snapshot.UnmarshalSnapshotOptions(&secrets.Items[0])
				g.Expect(err).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			waitForSnapshotToBeCreated(ctx, s.hostClient, s.vClusterNS)

			snapshotRequestCMs, err := s.hostClient.CoreV1().ConfigMaps(s.vClusterNS).List(ctx, listOptions)
			Expect(err).To(Succeed())
			Expect(snapshotRequestCMs.Items).To(HaveLen(1))
			snapshotRequest, err := snapshot.UnmarshalSnapshotRequest(&snapshotRequestCMs.Items[0])
			Expect(err).To(Succeed())

			snapshotRequest.Name = deleteSnapshotRequestName
			snapshotRequest.CreationTimestamp = metav1.Now()
			snapshotRequest.Status.Phase = snapshot.RequestPhaseDeleting

			deleteCM, err := snapshot.CreateSnapshotRequestConfigMap(s.vClusterNS, s.vClusterName, snapshotRequest)
			Expect(err).To(Succeed())
			deleteCM.Name = deleteSnapshotRequestName

			deleteSecret, err := snapshot.CreateSnapshotOptionsSecret(
				pkgconstants.SnapshotRequestLabel, s.vClusterNS, s.vClusterName, snapshotOptions)
			Expect(err).To(Succeed())
			deleteSecret.Name = deleteSnapshotRequestName

			_, err = s.hostClient.CoreV1().Secrets(s.vClusterNS).Create(ctx, deleteSecret, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			_, err = s.hostClient.CoreV1().ConfigMaps(s.vClusterNS).Create(ctx, deleteCM, metav1.CreateOptions{})
			Expect(err).To(Succeed())
		})

		It("Has deleted the snapshot", func(ctx context.Context) {
			Eventually(func(g Gomega) {
				cm, err := s.hostClient.CoreV1().ConfigMaps(s.vClusterNS).Get(ctx, deleteSnapshotRequestName, metav1.GetOptions{})
				g.Expect(err).To(Succeed())
				req, err := snapshot.UnmarshalSnapshotRequest(cm)
				g.Expect(err).To(Succeed())
				g.Expect(req.Status.Phase).To(Equal(snapshot.RequestPhaseDeleted))
				g.Expect(req.Status.VolumeSnapshots.Phase).To(Equal(volumes.RequestPhaseDeleted))
				for pvcName, vs := range req.Status.VolumeSnapshots.Snapshots {
					g.Expect(vs.Phase).To(Equal(volumes.RequestPhaseDeleted),
						"volume snapshot for PVC %s not deleted: %s", pvcName, toJSON(vs))
				}
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())
		})

		AfterAll(func(ctx context.Context) {
			err := s.vClusterClient.CoreV1().Namespaces().Delete(ctx, testNS, metav1.DeleteOptions{})
			if !kerrors.IsNotFound(err) {
				Expect(err).To(Succeed())
			}
		})
	},
	)
}

// --- Volume helpers ---

func createAppWithPVC(ctx context.Context, client kubernetes.Interface, namespace, name string) {
	GinkgoHelper()
	createPVC(ctx, client, namespace, name)
	createDeploymentWithVolume(ctx, client, namespace, name, name)
}

func createPVC(ctx context.Context, client kubernetes.Interface, namespace, name string) {
	GinkgoHelper()
	_, err := client.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("1Gi")},
			},
			StorageClassName: ptr.To("csi-hostpath-sc"),
		},
	}, metav1.CreateOptions{})
	Expect(err).To(Succeed())
}

func createDeploymentWithVolume(ctx context.Context, client kubernetes.Interface, namespace, deploymentName, pvcName string) {
	GinkgoHelper()
	_, err := client.AppsV1().Deployments(namespace).Create(ctx, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: deploymentName, Namespace: namespace},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To(int32(1)),
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"snapshot-test-app": deploymentName}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"snapshot-test-app": deploymentName}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name: "snapshot-test-app", Image: "busybox",
						Command:      []string{"sleep", "1000000"},
						VolumeMounts: []corev1.VolumeMount{{Name: "data", MountPath: "/data"}},
					}},
					Volumes: []corev1.Volume{{
						Name:         "data",
						VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: pvcName}},
					}},
				},
			},
		},
	}, metav1.CreateOptions{})
	Expect(err).To(Succeed())
}

func createPVCWithData(ctx context.Context, client kubernetes.Interface, pvcNamespace, pvcName, testFileName, testData string) {
	GinkgoHelper()
	createPVC(ctx, client, pvcNamespace, pvcName)
	deployJob(ctx, client, pvcNamespace, "write-test-data", pvcName,
		fmt.Sprintf("echo '%s' > /data/%s", testData, testFileName), testFileName)
}

func checkPVCData(ctx context.Context, client kubernetes.Interface, pvcNamespace, pvcName, testFileName, testData string) {
	GinkgoHelper()
	script := fmt.Sprintf(`actual=$(cat "/data/%s"); expected=%q;
if [ "$actual" = "$expected" ]; then echo "OK"; else echo "FAIL" >&2; exit 1; fi`, testFileName, testData)
	deployJob(ctx, client, pvcNamespace, "check-test-data", pvcName, script, testFileName)
}

func deployJob(ctx context.Context, client kubernetes.Interface, jobNamespace, jobName, pvcName, command, testFile string) {
	GinkgoHelper()
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{GenerateName: fmt.Sprintf("%s-", jobName), Namespace: jobNamespace},
		Spec: batchv1.JobSpec{
			BackoffLimit: ptr.To(int32(0)),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{{
						Name: "test-job", Image: "busybox:1.36",
						Command:      []string{"sh", "-c", command},
						VolumeMounts: []corev1.VolumeMount{{Name: "data", MountPath: "/data"}},
						WorkingDir:   "/data",
						ReadinessProbe: &corev1.Probe{
							ProbeHandler:        corev1.ProbeHandler{Exec: &corev1.ExecAction{Command: []string{"sh", "-c", "test -f /data/" + testFile}}},
							InitialDelaySeconds: 1, PeriodSeconds: 1, FailureThreshold: 10, TimeoutSeconds: 2,
						},
					}},
					Volumes: []corev1.Volume{{
						Name:         "data",
						VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: pvcName}},
					}},
				},
			},
		},
	}
	job, err := client.BatchV1().Jobs(jobNamespace).Create(ctx, job, metav1.CreateOptions{})
	Expect(err).To(Succeed())

	Eventually(func(g Gomega) {
		j, err := client.BatchV1().Jobs(jobNamespace).Get(ctx, job.Name, metav1.GetOptions{})
		g.Expect(err).To(Succeed())
		g.Expect(j.Status.Succeeded).To(Equal(int32(1)))
	}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())

	err = client.BatchV1().Jobs(jobNamespace).Delete(ctx, job.Name, metav1.DeleteOptions{PropagationPolicy: ptr.To(metav1.DeletePropagationBackground)})
	Expect(err).To(Succeed())
	Eventually(func(g Gomega) {
		_, err := client.BatchV1().Jobs(jobNamespace).Get(ctx, job.Name, metav1.GetOptions{})
		g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
	}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
}

func deletePVC(ctx context.Context, vClusterClient, _ kubernetes.Interface, _, _, pvcNamespace, pvcName string) {
	GinkgoHelper()
	err := vClusterClient.CoreV1().PersistentVolumeClaims(pvcNamespace).Delete(ctx, pvcName, metav1.DeleteOptions{})
	if kerrors.IsNotFound(err) {
		return
	}
	Expect(err).To(Succeed())
	Eventually(func(g Gomega) {
		_, err := vClusterClient.CoreV1().PersistentVolumeClaims(pvcNamespace).Get(ctx, pvcName, metav1.GetOptions{})
		g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
	}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
}

func getTwoSnapshotRequests(g Gomega, ctx context.Context, hostClient kubernetes.Interface, vClusterNamespace string) (*snapshot.Request, *snapshot.Request) {
	configMaps, err := hostClient.CoreV1().ConfigMaps(vClusterNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: pkgconstants.SnapshotRequestLabel,
	})
	g.Expect(err).To(Succeed())
	g.Expect(configMaps.Items).To(HaveLen(2))

	var previousCM, newerCM corev1.ConfigMap
	if configMaps.Items[0].CreationTimestamp.Time.Before(configMaps.Items[1].CreationTimestamp.Time) {
		previousCM = configMaps.Items[0]
		newerCM = configMaps.Items[1]
	} else {
		previousCM = configMaps.Items[1]
		newerCM = configMaps.Items[0]
	}
	previous, err := snapshot.UnmarshalSnapshotRequest(&previousCM)
	g.Expect(err).To(Succeed())
	newer, err := snapshot.UnmarshalSnapshotRequest(&newerCM)
	g.Expect(err).To(Succeed())
	return previous, newer
}
