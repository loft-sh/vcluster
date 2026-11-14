// Package metrics contains tests for the metrics endpoints the vCluster API
// server exposes.
package metrics

import (
	"bytes"
	"context"
	"crypto/x509"
	"maps"
	"net/http"
	"slices"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e/constants"
	"github.com/loft-sh/vcluster/e2e/labels"
	"github.com/loft-sh/vcluster/pkg/util/kubeconfig"
	"github.com/loft-sh/vcluster/pkg/util/random"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// syncerMetricsPath is the path the chart's ServiceMonitor scrapes. Unlike the
// other /metrics/* paths, which proxy to a local endpoint of another component,
// it is served in-process from the syncer's controller-runtime registry.
const syncerMetricsPath = "/metrics/syncer"

// syncerRegistryMetric identifies the registry being served: the syncer's
// controller-runtime one, rather than a proxied component's. controller-runtime
// registers it when it starts a controller, which happens after the proxy has
// begun serving, so a healthy endpoint can briefly answer without this family.
//
// Note before reusing this spec on an HA control plane: the proxy that serves
// this endpoint starts on every replica, but controllers start behind leader
// election, so a follower exposes the registry without this family.
const syncerRegistryMetric = "controller_runtime_reconcile_total"

// scrapeResult is the outcome of a single GET of syncerMetricsPath.
type scrapeResult struct {
	statusCode  int
	contentType string
	body        []byte
	err         error
}

// scrape performs one GET of syncerMetricsPath with the credentials the
// clientset carries.
func scrape(ctx context.Context, clientset kubernetes.Interface) scrapeResult {
	var res scrapeResult
	result := clientset.CoreV1().RESTClient().Get().
		AbsPath(syncerMetricsPath).
		Do(ctx).
		StatusCode(&res.statusCode).
		ContentType(&res.contentType)

	// Raw() hands back the transport-level error, which for a JSON error
	// response is the bare string "unknown". Error() decodes the metav1.Status
	// in the body instead, so a denial keeps the message the authorizer wrote.
	res.body, _ = result.Raw()
	res.err = result.Error()
	return res
}

// SyncerMetricsSpec registers the spec.
func SyncerMetricsSpec() {
	Describe("Syncer metrics endpoint",
		labels.Core, labels.Metrics,
		func() {
			var (
				vClusterConfig    *rest.Config
				vClusterClientset *kubernetes.Clientset
				vClusterName      string
				vClusterHostNS    string
				hostClient        kubernetes.Interface
			)

			BeforeEach(func(ctx context.Context) context.Context {
				vClusterConfig = cluster.CurrentClusterFrom(ctx).KubernetesRestConfig()
				var err error
				vClusterClientset, err = kubernetes.NewForConfig(vClusterConfig)
				Expect(err).To(Succeed())

				vClusterName = cluster.CurrentClusterNameFrom(ctx)
				vClusterHostNS = "vcluster-" + vClusterName
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
				return ctx
			})

			It("exposes the syncer's Prometheus registry on "+syncerMetricsPath, func(ctx context.Context) {
				var contentType string

				// The registry check polls with the request rather than after it.
				// The proxy serves this path before the controllers start, so an
				// early scrape returns 200 and a body that parses fine off the go
				// and process collectors but carries no reconcile family yet.
				By("scraping the endpoint until it reports the syncer's registry", func() {
					Eventually(func(g Gomega) {
						res := scrape(ctx, vClusterClientset)
						g.Expect(res.err).To(Succeed(), "GET %s failed with status %d", syncerMetricsPath, res.statusCode)
						g.Expect(res.statusCode).To(Equal(http.StatusOK), "GET %s returned body: %s", syncerMetricsPath, res.body)

						parser := expfmt.NewTextParser(model.UTF8Validation)
						families, err := parser.TextToMetricFamilies(bytes.NewReader(res.body))
						g.Expect(err).To(Succeed(), "GET %s did not return parsable Prometheus text format, body: %s", syncerMetricsPath, res.body)
						g.Expect(families).NotTo(BeEmpty(), "%s exposed no metric families", syncerMetricsPath)
						g.Expect(families).To(HaveKey(syncerRegistryMetric),
							"%s should expose %q from the syncer's controller-runtime registry, exposed families: %v",
							syncerMetricsPath, syncerRegistryMetric, slices.Sorted(maps.Keys(families)))

						contentType = res.contentType
					}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})

				By("checking the response is scrapable Prometheus text exposition format", func() {
					Expect(contentType).To(HavePrefix("text/plain"),
						"%s must be scrapable by the ServiceMonitor, got Content-Type %q", syncerMetricsPath, contentType)
				})
			})

			It("accepts the client certificate the chart's ServiceMonitor scrapes with", func(ctx context.Context) {
				// chart/templates/service-monitor.yaml points the /metrics/syncer
				// endpoint's tlsConfig at these three keys of this secret. The key
				// names themselves are pinned against the template in
				// chart/tests/service-monitor_test.yaml; what this spec covers is
				// the other half of that contract, that the secret carries usable
				// credentials and the API server accepts them.
				secretName := kubeconfig.DefaultSecretPrefix + vClusterName
				var scrapeClientset *kubernetes.Clientset

				By("building a client from the "+secretName+" secret the ServiceMonitor mounts", func() {
					secret, err := hostClient.CoreV1().Secrets(vClusterHostNS).Get(ctx, secretName, metav1.GetOptions{})
					Expect(err).To(Succeed(), "the ServiceMonitor mounts secret %s/%s", vClusterHostNS, secretName)

					for _, key := range []string{
						kubeconfig.CADataSecretKey,
						kubeconfig.CertificateSecretKey,
						kubeconfig.CertificateKeySecretKey,
					} {
						Expect(secret.Data).To(HaveKey(key),
							"secret %s/%s must carry %q for the ServiceMonitor tlsConfig, has keys: %v",
							vClusterHostNS, secretName, key, slices.Sorted(maps.Keys(secret.Data)))
						Expect(secret.Data[key]).NotTo(BeEmpty(),
							"secret %s/%s key %q is empty, the ServiceMonitor would scrape with nothing",
							vClusterHostNS, secretName, key)
					}

					// The suite reaches the vCluster through the connect proxy, whose
					// kubeconfig may skip server verification, so leave the base
					// config's server trust alone and only assert the CA the
					// ServiceMonitor mounts is a usable bundle.
					Expect(x509.NewCertPool().AppendCertsFromPEM(secret.Data[kubeconfig.CADataSecretKey])).To(BeTrue(),
						"secret %s/%s key %q is not a PEM certificate bundle Prometheus could trust",
						vClusterHostNS, secretName, kubeconfig.CADataSecretKey)

					// The client credential is what decides whether Prometheus gets
					// past the authorizer, so that is what gets swapped in.
					cfg := rest.AnonymousClientConfig(vClusterConfig)
					cfg.TLSClientConfig.CertData = secret.Data[kubeconfig.CertificateSecretKey]
					cfg.TLSClientConfig.KeyData = secret.Data[kubeconfig.CertificateKeySecretKey]

					scrapeClientset, err = kubernetes.NewForConfig(cfg)
					Expect(err).To(Succeed())
				})

				By("scraping the endpoint with that certificate", func() {
					Eventually(func(g Gomega) {
						res := scrape(ctx, scrapeClientset)
						g.Expect(res.err).To(Succeed(), "GET %s with the ServiceMonitor certificate failed with status %d", syncerMetricsPath, res.statusCode)
						g.Expect(res.statusCode).To(Equal(http.StatusOK), "GET %s returned body: %s", syncerMetricsPath, res.body)
						g.Expect(res.body).NotTo(BeEmpty(), "%s returned an empty body", syncerMetricsPath)
					}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})

			// What keeps this endpoint off limits is its entry in
			// metricsAuthNonResources() (pkg/server/server.go). Without it the path
			// falls through the authorizer union to allowall.New(), and every
			// authenticated tenant identity can read it. The specs above scrape as a
			// cluster-admin identity and stay green either way, so the two denial
			// cases below are what notice if that entry disappears; the third pins
			// that the grant RBAC documents for it is the one that actually works.
			Context("authorization", func() {
				It("denies a scrape with no credentials", func(ctx context.Context) {
					// Anonymous auth is on, so an unauthenticated request is not
					// rejected at authn. It arrives as system:anonymous and the
					// authorizer denies it, which is a 403 rather than a 401.
					anonClientset, err := kubernetes.NewForConfig(rest.AnonymousClientConfig(vClusterConfig))
					Expect(err).To(Succeed())

					Eventually(func(g Gomega) {
						res := scrape(ctx, anonClientset)
						g.Expect(res.err).To(HaveOccurred(), "GET %s without credentials must not succeed", syncerMetricsPath)
						g.Expect(kerrors.IsForbidden(res.err)).To(BeTrue(),
							"expected 403 Forbidden, got status %d and error: %v", res.statusCode, res.err)
						g.Expect(res.err).To(MatchError(ContainSubstring(`User "system:anonymous" cannot get path "`+syncerMetricsPath+`"`)),
							"the denial should name the anonymous user and the path")
					}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutShort).Should(Succeed())
				})

				Context("as a tenant ServiceAccount", func() {
					var (
						suffix      string
						nsName      string
						saName      string
						saUser      string
						saClientset *kubernetes.Clientset
					)

					BeforeEach(func(ctx context.Context) {
						suffix = random.String(6)
						nsName = "syncer-metrics-auth-" + suffix
						saName = "metrics-scraper-" + suffix
						saUser = "system:serviceaccount:" + nsName + ":" + saName

						_, err := vClusterClientset.CoreV1().Namespaces().Create(ctx,
							&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}},
							metav1.CreateOptions{})
						Expect(err).To(Succeed())
						DeferCleanup(func(ctx context.Context) {
							err := vClusterClientset.CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{})
							Expect(ctrlclient.IgnoreNotFound(err)).To(Succeed())
						})

						_, err = vClusterClientset.CoreV1().ServiceAccounts(nsName).Create(ctx,
							&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: saName, Namespace: nsName}},
							metav1.CreateOptions{})
						Expect(err).To(Succeed())

						By("minting a token for "+saUser, func() {
							var token *authenticationv1.TokenRequest
							// The ServiceAccount is created moments earlier, so the
							// TokenRequest can race its propagation to the issuer.
							Eventually(func(g Gomega) {
								var err error
								token, err = vClusterClientset.CoreV1().ServiceAccounts(nsName).CreateToken(ctx, saName,
									&authenticationv1.TokenRequest{}, metav1.CreateOptions{})
								g.Expect(err).To(Succeed(), "minting a token for %s", saUser)
								g.Expect(token.Status.Token).NotTo(BeEmpty(), "token for %s came back empty", saUser)
							}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutShort).Should(Succeed())

							cfg := rest.AnonymousClientConfig(vClusterConfig)
							cfg.BearerToken = token.Status.Token

							var err error
							saClientset, err = kubernetes.NewForConfig(cfg)
							Expect(err).To(Succeed())
						})
					})

					It("denies a scrape without an explicit grant on the path", func(ctx context.Context) {
						Eventually(func(g Gomega) {
							res := scrape(ctx, saClientset)
							g.Expect(res.err).To(HaveOccurred(),
								"GET %s from an ungranted tenant ServiceAccount must not succeed", syncerMetricsPath)
							g.Expect(kerrors.IsForbidden(res.err)).To(BeTrue(),
								"expected 403 Forbidden, got status %d and error: %v", res.statusCode, res.err)
							g.Expect(res.err).To(MatchError(ContainSubstring(`User "`+saUser+`" cannot get path "`+syncerMetricsPath+`"`)),
								"the denial should name the ServiceAccount and the path")
						}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutShort).Should(Succeed())
					})

					It("admits a scrape once granted get on the path", func(ctx context.Context) {
						By("granting the ServiceAccount get on "+syncerMetricsPath, func() {
							// Cluster-scoped, so the name carries the spec's suffix.
							roleName := "syncer-metrics-reader-" + suffix

							_, err := vClusterClientset.RbacV1().ClusterRoles().Create(ctx, &rbacv1.ClusterRole{
								ObjectMeta: metav1.ObjectMeta{Name: roleName},
								Rules: []rbacv1.PolicyRule{{
									NonResourceURLs: []string{syncerMetricsPath},
									Verbs:           []string{"get"},
								}},
							}, metav1.CreateOptions{})
							Expect(err).To(Succeed())
							DeferCleanup(func(ctx context.Context) {
								err := vClusterClientset.RbacV1().ClusterRoles().Delete(ctx, roleName, metav1.DeleteOptions{})
								Expect(ctrlclient.IgnoreNotFound(err)).To(Succeed())
							})

							_, err = vClusterClientset.RbacV1().ClusterRoleBindings().Create(ctx, &rbacv1.ClusterRoleBinding{
								ObjectMeta: metav1.ObjectMeta{Name: roleName},
								RoleRef: rbacv1.RoleRef{
									APIGroup: rbacv1.GroupName,
									Kind:     "ClusterRole",
									Name:     roleName,
								},
								Subjects: []rbacv1.Subject{{
									Kind:      rbacv1.ServiceAccountKind,
									Name:      saName,
									Namespace: nsName,
								}},
							}, metav1.CreateOptions{})
							Expect(err).To(Succeed())
							DeferCleanup(func(ctx context.Context) {
								err := vClusterClientset.RbacV1().ClusterRoleBindings().Delete(ctx, roleName, metav1.DeleteOptions{})
								Expect(ctrlclient.IgnoreNotFound(err)).To(Succeed())
							})
						})

						By("scraping the endpoint as the granted ServiceAccount", func() {
							// Polls for RBAC to reach the authorizer. Only allow
							// decisions are cached (delegatingauthorizer/cache.go), so
							// the earlier denials cannot keep this at 403.
							Eventually(func(g Gomega) {
								res := scrape(ctx, saClientset)
								g.Expect(res.err).To(Succeed(),
									"GET %s as %s failed with status %d", syncerMetricsPath, saUser, res.statusCode)
								g.Expect(res.statusCode).To(Equal(http.StatusOK), "GET %s returned body: %s", syncerMetricsPath, res.body)
								g.Expect(res.body).NotTo(BeEmpty(), "%s returned an empty body", syncerMetricsPath)
							}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutShort).Should(Succeed())
						})
					})
				})
			})
		},
	)
}
