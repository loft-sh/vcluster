package export_kubeconfig

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e/constants"
	"github.com/loft-sh/vcluster/e2e/labels"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const ServiceAccountSecretName = "export-kubeconfig-serviceaccount"

func ServiceAccountExportSpec() {
	Describe("service account kubeconfig export", labels.Core, func() {
		It("repopulates a GitOps-recreated Secret and repairs its managed credential data", func(ctx context.Context) {
			hostClient := cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
			Expect(hostClient).NotTo(BeNil())
			secrets := hostClient.CoreV1().Secrets(TargetNS)
			var original *corev1.Secret

			By("waiting for the initial exported credential")
			Eventually(func(g Gomega) {
				var err error
				original, err = secrets.Get(ctx, ServiceAccountSecretName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred(), "read exported Secret %s/%s", TargetNS, ServiceAccountSecretName)
				cfg, err := clientcmd.RESTConfigFromKubeConfig(original.Data["config"])
				g.Expect(err).NotTo(HaveOccurred(), "parse kubeconfig from Secret %s/%s", TargetNS, ServiceAccountSecretName)
				expectExportIdentity(ctx, g, cfg)
			}).WithContext(ctx).WithPolling(constants.PollingInterval).
				WithTimeout(constants.PollingTimeoutLong).
				Should(Succeed())

			By("deleting the destination Secret")
			Expect(secrets.Delete(ctx, ServiceAccountSecretName, metav1.DeleteOptions{
				Preconditions: &metav1.Preconditions{UID: &original.UID},
			})).To(Succeed(), "delete exported Secret %s/%s", TargetNS, ServiceAccountSecretName)
			Eventually(func(g Gomega) {
				_, err := secrets.Get(ctx, ServiceAccountSecretName, metav1.GetOptions{})
				g.Expect(kerrors.IsNotFound(err)).To(BeTrue(), "Secret %s/%s must be absent after deletion, got %v", TargetNS, ServiceAccountSecretName, err)
			}).WithContext(ctx).WithPolling(constants.PollingInterval).
				WithTimeout(constants.PollingTimeoutShort).
				Should(Succeed())

			By("recreating an empty destination as a GitOps controller would")
			recreated, err := secrets.Create(ctx, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: ServiceAccountSecretName, Labels: original.Labels, Annotations: original.Annotations,
				},
				Type: corev1.SecretTypeOpaque,
			}, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred(), "recreate exported Secret %s/%s", TargetNS, ServiceAccountSecretName)
			Expect(recreated.UID).NotTo(Equal(original.UID))
			expectRepaired := func(g Gomega) {
				GinkgoHelper()
				secret, err := secrets.Get(ctx, ServiceAccountSecretName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred(), "read recreated Secret %s/%s", TargetNS, ServiceAccountSecretName)
				g.Expect(secret.UID).To(Equal(recreated.UID))
				g.Expect(secret.Labels).To(HaveKeyWithValue("example.com/export", "runtime"))
				cfg, err := clientcmd.RESTConfigFromKubeConfig(secret.Data["config"])
				g.Expect(err).NotTo(HaveOccurred(), "parse repaired kubeconfig from Secret %s/%s", TargetNS, ServiceAccountSecretName)
				g.Expect(len(cfg.CertData)).To(BeZero()) //nolint:ginkgolinter // Do not print credential data on failure.
				g.Expect(len(cfg.KeyData)).To(BeZero())  //nolint:ginkgolinter // Do not print credential data on failure.
				g.Expect(string(secret.Data["token"]) == cfg.BearerToken && cfg.BearerToken != "").To(BeTrue())
				expectExportIdentity(ctx, g, cfg)
			}
			Eventually(func(g Gomega) { expectRepaired(g) }).WithContext(ctx).WithPolling(constants.PollingInterval).
				WithTimeout(constants.PollingTimeoutShort).
				Should(Succeed())

			By("repairing removed credential data while preserving an unrelated key")
			_, err = secrets.Patch(ctx, ServiceAccountSecretName, types.MergePatchType,
				[]byte(`{"data":{"config":null,"token":null,"external-note":"a2VlcA=="}}`), metav1.PatchOptions{})
			Expect(err).NotTo(HaveOccurred(), "remove managed credential data from Secret %s/%s", TargetNS, ServiceAccountSecretName)
			Eventually(func(g Gomega) {
				expectRepaired(g)
				secret, err := secrets.Get(ctx, ServiceAccountSecretName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred(), "read repaired Secret %s/%s", TargetNS, ServiceAccountSecretName)
				g.Expect(string(secret.Data["external-note"])).To(Equal("keep"))
			}).WithContext(ctx).WithPolling(constants.PollingInterval).
				WithTimeout(constants.PollingTimeoutShort).
				Should(Succeed())
		})

		It("renews credentials before expiry without replacing the Secret or its labels", func(ctx context.Context) {
			hostClient := cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
			Expect(hostClient).NotTo(BeNil())
			var originalToken, originalUID string
			var expires time.Time

			By("waiting for a restricted ServiceAccount credential to be exported")
			Eventually(func(g Gomega) {
				secret, err := hostClient.CoreV1().Secrets(TargetNS).Get(ctx, ServiceAccountSecretName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred(), "read exported Secret %s/%s before renewal", TargetNS, ServiceAccountSecretName)
				cfg, err := clientcmd.RESTConfigFromKubeConfig(secret.Data["config"])
				g.Expect(err).NotTo(HaveOccurred(), "parse kubeconfig from Secret %s/%s before renewal", TargetNS, ServiceAccountSecretName)
				g.Expect(cfg.BearerToken).NotTo(BeEmpty(), "export should contain a token")
				g.Expect(len(cfg.CertData)).To(BeZero(), "export must not retain an administrative certificate") //nolint:ginkgolinter // Do not print credential data on failure.
				g.Expect(len(cfg.KeyData)).To(BeZero(), "export must not retain an administrative key")          //nolint:ginkgolinter // Do not print credential data on failure.
				g.Expect(secret.Labels).To(HaveKeyWithValue("example.com/export", "runtime"))
				expectExportIdentity(ctx, g, cfg)

				// Decode only the expiry for the test; authentication is verified above.
				parts := strings.Split(cfg.BearerToken, ".")
				g.Expect(len(parts)).To(Equal(3)) //nolint:ginkgolinter // Do not print the token on failure.
				payload, err := base64.RawURLEncoding.DecodeString(parts[1])
				g.Expect(err).NotTo(HaveOccurred(), "decode token claims from Secret %s/%s", TargetNS, ServiceAccountSecretName)
				claims := struct {
					Expires int64 `json:"exp"`
				}{}
				g.Expect(json.Unmarshal(payload, &claims)).To(Succeed(), "decode token expiration from Secret %s/%s", TargetNS, ServiceAccountSecretName)
				expires = time.Unix(claims.Expires, 0)
				g.Expect(expires.After(time.Now())).To(BeTrue())
				originalToken, originalUID = cfg.BearerToken, string(secret.UID)
			}).WithContext(ctx).WithPolling(constants.PollingInterval).
				WithTimeout(constants.PollingTimeoutLong).
				Should(Succeed())

			By("observing a working replacement credential before the original expires")
			Eventually(func(g Gomega) {
				secret, err := hostClient.CoreV1().Secrets(TargetNS).Get(ctx, ServiceAccountSecretName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred(), "read exported Secret %s/%s after renewal", TargetNS, ServiceAccountSecretName)
				cfg, err := clientcmd.RESTConfigFromKubeConfig(secret.Data["config"])
				g.Expect(err).NotTo(HaveOccurred(), "parse kubeconfig from Secret %s/%s after renewal", TargetNS, ServiceAccountSecretName)
				g.Expect(cfg.BearerToken != "" && cfg.BearerToken != originalToken).To(BeTrue(), "credential has not renewed")
				g.Expect(string(secret.UID)).To(Equal(originalUID))
				g.Expect(secret.Labels).To(HaveKeyWithValue("example.com/export", "runtime"))
				g.Expect(time.Now().Before(expires)).To(BeTrue(), "renewal must precede expiry")
				expectExportIdentity(ctx, g, cfg)
			}).WithContext(ctx).WithPolling(constants.PollingInterval).
				WithTimeout(constants.PollingTimeoutVeryLong).
				Should(Succeed())
		})
	})
}

func expectExportIdentity(ctx context.Context, g Gomega, cfg *rest.Config) {
	GinkgoHelper()
	// Route through the framework's proxy while retaining the exported credentials.
	cfg.Host = cluster.CurrentClusterFrom(ctx).KubernetesRestConfig().Host
	cfg.Timeout = constants.PollingTimeoutShort
	api, err := kubernetes.NewForConfig(cfg)
	g.Expect(err).NotTo(HaveOccurred(), "create client for credentials exported in Secret %s/%s", TargetNS, ServiceAccountSecretName)
	identity, err := api.AuthenticationV1().SelfSubjectReviews().Create(ctx, &authenticationv1.SelfSubjectReview{}, metav1.CreateOptions{})
	g.Expect(err).NotTo(HaveOccurred(), "authenticate credentials exported in Secret %s/%s", TargetNS, ServiceAccountSecretName)
	g.Expect(identity.Status.UserInfo.Username).To(Equal("system:serviceaccount:kube-system:default"))
}
