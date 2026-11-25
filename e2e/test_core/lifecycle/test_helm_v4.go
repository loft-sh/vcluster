package lifecycle

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/loft-sh/vcluster/e2e/constants"
	"github.com/loft-sh/vcluster/e2e/labels"
	"github.com/loft-sh/vcluster/pkg/util/random"
)

// upgradeAnnotation is set on the control plane StatefulSet by the upgrade, to
// check that the new value was applied.
const (
	upgradeAnnotationKey   = "e2e-helm"
	upgradeAnnotationValue = "v4"
)

// HelmV4ApplyMethodSpec checks that the CLI keeps client-side apply when the
// helm on PATH is helm 4, which otherwise creates new releases with
// server-side apply. It also checks that helm 4 can upgrade and delete a
// release helm 3 created. The spec downloads the helm versions it needs, so it
// does not depend on the helm CI puts on PATH.
func HelmV4ApplyMethodSpec() {
	Describe("Tenant cluster create, upgrade and delete with helm 4", labels.PR, labels.Core, labels.CLI, func() {
		Context("client-side apply", Ordered, func() {
			// Ordered because each spec works on the release the create spec
			// made.
			var (
				clusterName string
				namespace   string
				hostClient  kubernetes.Interface
				pathEnv     string
			)

			BeforeAll(func(ctx context.Context) {
				clusterName = "e2e-h4-" + random.String(6)
				namespace = clusterName
				hostClient = hostKubeClient()
				pathEnv = helmV4PathEnv(ctx)
			})

			AfterAll(func(ctx context.Context) {
				// Cleanup when a spec failed before the delete spec. CI runs
				// with --teardown=false, which skips this.
				_, err := runVClusterCmdEnv(ctx, []string{pathEnv}, "delete", clusterName, "-n", namespace, "--delete-namespace", "--ignore-not-found")
				Expect(err).To(Succeed())
			})

			It("should create the release with client-side apply", func(ctx context.Context) {
				By("Creating a tenant cluster with helm 4 on PATH", func() {
					_, err := runVClusterCmdEnv(ctx, []string{pathEnv}, createArgs(clusterName, namespace)...)
					Expect(err).To(Succeed())
				})

				By("Waiting for tenant cluster to be ready", func() {
					waitForVClusterReady(ctx, hostClient, clusterName, namespace)
				})

				expectClientSideApply(ctx, hostClient, clusterName, namespace, 1)
			})

			It("should upgrade the release with client-side apply", func(ctx context.Context) {
				upgradeWithHelmV4(ctx, hostClient, pathEnv, clusterName, namespace)
				expectClientSideApply(ctx, hostClient, clusterName, namespace, 2)
			})

			It("should delete the tenant cluster with helm 4", func(ctx context.Context) {
				deleteWithHelmV4(ctx, hostClient, pathEnv, clusterName, namespace)
			})
		})

		Context("release created by helm 3", Ordered, func() {
			// Ordered because each spec works on the release the create spec
			// made. This is the switch existing users make when they move to
			// the vcluster-cli image with helm 4.
			var (
				clusterName string
				namespace   string
				hostClient  kubernetes.Interface
				pathEnv     string
				pathEnvV3   string
			)

			BeforeAll(func(ctx context.Context) {
				clusterName = "e2e-h3to4-" + random.String(6)
				namespace = clusterName
				hostClient = hostKubeClient()
				pathEnv = helmV4PathEnv(ctx)
				pathEnvV3 = helmV3PathEnv(ctx)
			})

			AfterAll(func(ctx context.Context) {
				// Cleanup when a spec failed before the delete spec. CI runs
				// with --teardown=false, which skips this.
				_, err := runVClusterCmdEnv(ctx, []string{pathEnv}, "delete", clusterName, "-n", namespace, "--delete-namespace", "--ignore-not-found")
				Expect(err).To(Succeed())
			})

			It("should create the release with helm 3", func(ctx context.Context) {
				By(fmt.Sprintf("Creating a tenant cluster with helm %s on PATH", helmV3Version), func() {
					_, err := runVClusterCmdEnv(ctx, []string{pathEnvV3}, createArgs(clusterName, namespace)...)
					Expect(err).To(Succeed())
				})

				By("Waiting for tenant cluster to be ready", func() {
					waitForVClusterReady(ctx, hostClient, clusterName, namespace)
				})

				By("Checking helm 3 wrote revision 1", func() {
					release := getRelease(ctx, hostClient, clusterName, namespace, 1)
					Expect(release).NotTo(HaveKey("apply_method"), "helm 3 records no apply_method")
				})
			})

			// Helm 4 keeps a release without apply_method on client-side apply
			// even without --server-side=false, so this checks the upgrade
			// works, not the flag. The client-side apply context covers the
			// flag.
			It("should upgrade a release created by helm 3 with helm 4", func(ctx context.Context) {
				upgradeWithHelmV4(ctx, hostClient, pathEnv, clusterName, namespace)
				expectClientSideApply(ctx, hostClient, clusterName, namespace, 2)
			})

			It("should delete a release created by helm 3 with helm 4", func(ctx context.Context) {
				deleteWithHelmV4(ctx, hostClient, pathEnv, clusterName, namespace)
			})
		})
	})
}

// upgradeWithHelmV4 upgrades the tenant cluster with helm 4 on PATH and checks
// the new value was applied.
func upgradeWithHelmV4(ctx context.Context, hostClient kubernetes.Interface, pathEnv, name, namespace string) {
	GinkgoHelper()

	By("Upgrading the tenant cluster with helm 4 on PATH and a new annotation", func() {
		args := append(createArgs(name, namespace),
			"--upgrade",
			"--set", fmt.Sprintf("controlPlane.statefulSet.annotations.%s=%s", upgradeAnnotationKey, upgradeAnnotationValue),
		)
		_, err := runVClusterCmdEnv(ctx, []string{pathEnv}, args...)
		Expect(err).To(Succeed())
	})

	By("Checking the new value was applied", func() {
		sts, err := hostClient.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		Expect(err).To(Succeed())
		Expect(sts.Annotations).To(HaveKeyWithValue(upgradeAnnotationKey, upgradeAnnotationValue))
	})
}

// deleteWithHelmV4 deletes the tenant cluster with helm 4 on PATH and waits for
// its namespace to be gone.
func deleteWithHelmV4(ctx context.Context, hostClient kubernetes.Interface, pathEnv, name, namespace string) {
	GinkgoHelper()

	By("Deleting the tenant cluster with helm 4 on PATH", func() {
		_, err := runVClusterCmdEnv(ctx, []string{pathEnv}, "delete", name, "-n", namespace, "--delete-namespace")
		Expect(err).To(Succeed())
	})

	By("Checking the namespace is gone", func() {
		Eventually(func(g Gomega, ctx context.Context) {
			_, err := hostClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
			g.Expect(kerrors.IsNotFound(err)).To(BeTrue(), "namespace %s should be deleted", namespace)
		}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
	})
}

// expectClientSideApply checks that helm wrote the control plane StatefulSet
// with client-side apply (operation Update, not Apply) and that the release
// revision records apply_method csa.
func expectClientSideApply(ctx context.Context, hostClient kubernetes.Interface, name, namespace string, revision int) {
	GinkgoHelper()

	By("Checking helm owns the StatefulSet fields through Update", func() {
		sts, err := hostClient.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		Expect(err).To(Succeed())
		var helmOps []string
		for _, entry := range sts.ManagedFields {
			if entry.Manager == "helm" {
				helmOps = append(helmOps, string(entry.Operation))
			}
		}
		Expect(helmOps).To(ConsistOf("Update"), "helm managedFields operations on StatefulSet %s/%s", namespace, name)
	})

	By(fmt.Sprintf("Checking release revision %d records apply_method csa", revision), func() {
		release := getRelease(ctx, hostClient, name, namespace, revision)
		Expect(release).To(HaveKeyWithValue("apply_method", "csa"))
	})
}

// getRelease reads one revision of a helm release from its Secret.
func getRelease(ctx context.Context, hostClient kubernetes.Interface, name, namespace string, revision int) map[string]any {
	GinkgoHelper()
	secretName := fmt.Sprintf("sh.helm.release.v1.%s.v%d", name, revision)
	secret, err := hostClient.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	Expect(err).To(Succeed(), "get release secret %s/%s", namespace, secretName)

	release, err := decodeRelease(secret.Data["release"])
	Expect(err).To(Succeed(), "decode release secret %s/%s", namespace, secretName)
	return release
}

// decodeRelease decodes a helm release stored in a Secret: base64 of the
// gzipped release JSON. This is helm's internal storage format, not a public
// API, so a helm change can break it.
func decodeRelease(data []byte) (map[string]any, error) {
	gz, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, fmt.Errorf("base64: %w", err)
	}
	reader, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		return nil, fmt.Errorf("gzip: %w", err)
	}
	defer reader.Close()
	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("gunzip: %w", err)
	}
	release := map[string]any{}
	if err := json.Unmarshal(raw, &release); err != nil {
		return nil, fmt.Errorf("json: %w", err)
	}
	return release, nil
}

// helmV4PathEnv returns a PATH entry that puts a helm 4 binary first.
func helmV4PathEnv(ctx context.Context) string {
	GinkgoHelper()
	return pathEnvWith(helmV4Dir(ctx))
}

// helmV3PathEnv returns a PATH entry that puts helm helmV3Version first.
func helmV3PathEnv(ctx context.Context) string {
	GinkgoHelper()
	dir, err := cachedHelmDir(ctx, helmV3Version)
	Expect(err).To(Succeed(), "download helm %s", helmV3Version)
	expectHelmVersion(ctx, filepath.Join(dir, "helm"), "v3.")
	return pathEnvWith(dir)
}

func pathEnvWith(dir string) string {
	return "PATH=" + dir + string(os.PathListSeparator) + os.Getenv("PATH")
}

// helmV4Dir returns a directory holding a helm 4 binary named helm. It uses
// E2E_HELM4_DIR when set and otherwise downloads the helm the vcluster-cli
// image ships (HELM_VERSION in Dockerfile.cli.release).
func helmV4Dir(ctx context.Context) string {
	GinkgoHelper()
	if dir := os.Getenv("E2E_HELM4_DIR"); dir != "" {
		expectHelmVersion(ctx, filepath.Join(dir, "helm"), "v4.")
		return dir
	}

	version, err := cliImageHelmVersion()
	Expect(err).To(Succeed(), "read the helm version of the vcluster-cli image")
	Expect(version).To(HavePrefix("v4."), "Dockerfile.cli.release ships helm %s, not helm 4", version)

	dir, err := cachedHelmDir(ctx, version)
	Expect(err).To(Succeed(), "download helm %s", version)
	expectHelmVersion(ctx, filepath.Join(dir, "helm"), "v4.")
	return dir
}

func expectHelmVersion(ctx context.Context, helmPath, prefix string) {
	GinkgoHelper()
	out, err := exec.CommandContext(ctx, helmPath, "version", "--template", "{{.Version}}").Output()
	Expect(err).To(Succeed(), "run %s version", helmPath)
	Expect(strings.TrimSpace(string(out))).To(HavePrefix(prefix), "%s is not helm %s", helmPath, prefix)
}
