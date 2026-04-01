package test_core

import (
	"context"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/util/random"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// hostSAAnnotationName is the annotation set by the SA syncer to record the virtual SA name.
const hostSAAnnotationName = "vcluster.loft.sh/object-name"

// hostSANamespaceLabel is the label set by the SA syncer to record the virtual namespace.
const hostSANamespaceLabel = "vcluster.loft.sh/namespace"

// expectedPullSecret is the imagePullSecret name configured in vcluster-sa-imagepullsecrets.yaml.
const expectedPullSecret = "e2e-test-registry-secret"

// selectorLabel is the label key/value that the imagePullSecretSelector matches on.
const selectorLabelKey = "inject-pull-secrets"
const selectorLabelValue = "true"

// DescribeSAImagePullSecrets registers tests that verify workloadServiceAccount imagePullSecrets
// are propagated to synced host ServiceAccounts when they match the configured selector.
func DescribeSAImagePullSecrets() {
	Describe("ServiceAccount imagePullSecrets propagation",
		labels.Core,
		labels.PR,
		labels.Sync,
		labels.ServiceAccounts,
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

			// findHostSA lists host ServiceAccounts for the given virtual namespace and returns
			// the one whose vcluster ownership annotation matches saName.
			findHostSA := func(ctx context.Context, g Gomega, virtualNS, saName string) *corev1.ServiceAccount {
				GinkgoHelper()
				sas, err := hostClient.CoreV1().ServiceAccounts("").List(ctx, metav1.ListOptions{
					LabelSelector: hostSANamespaceLabel + "=" + virtualNS,
				})
				g.Expect(err).To(Succeed(), "listing host SAs for virtual namespace %q", virtualNS)

				for i := range sas.Items {
					if sas.Items[i].Annotations[hostSAAnnotationName] == saName {
						return &sas.Items[i]
					}
				}
				g.Expect(false).To(BeTrue(), "host SA for virtual SA %q/%q not found yet", virtualNS, saName)
				return nil
			}

			It("propagates imagePullSecrets to a SA whose labels match the selector", func(ctx context.Context) {
				suffix := random.String(6)
				nsName := "sa-pull-secret-test-" + suffix
				saName := "sa-matching-" + suffix

				By("creating a virtual namespace", func() {
					_, err := vClusterClient.CoreV1().Namespaces().Create(ctx,
						&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}},
						metav1.CreateOptions{})
					Expect(err).To(Succeed())
					DeferCleanup(func(ctx context.Context) {
						err := vClusterClient.CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{})
						if !kerrors.IsNotFound(err) {
							Expect(err).To(Succeed())
						}
					})
				})

				By("creating a virtual SA with the matching label", func() {
					_, err := vClusterClient.CoreV1().ServiceAccounts(nsName).Create(ctx,
						&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
							Name:      saName,
							Namespace: nsName,
							Labels:    map[string]string{selectorLabelKey: selectorLabelValue},
						}},
						metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				By("waiting for the host SA to have the configured imagePullSecrets", func() {
					Eventually(func(g Gomega) {
						hostSA := findHostSA(ctx, g, nsName, saName)
						g.Expect(hostSA.ImagePullSecrets).To(ContainElement(
							corev1.LocalObjectReference{Name: expectedPullSecret},
						), "host SA imagePullSecrets should contain %q", expectedPullSecret)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})

			It("does not propagate imagePullSecrets to a SA whose labels do not match the selector", func(ctx context.Context) {
				suffix := random.String(6)
				nsName := "sa-pull-secret-test-" + suffix
				saName := "sa-nonmatching-" + suffix

				By("creating a virtual namespace", func() {
					_, err := vClusterClient.CoreV1().Namespaces().Create(ctx,
						&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}},
						metav1.CreateOptions{})
					Expect(err).To(Succeed())
					DeferCleanup(func(ctx context.Context) {
						err := vClusterClient.CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{})
						if !kerrors.IsNotFound(err) {
							Expect(err).To(Succeed())
						}
					})
				})

				By("creating a virtual SA without the matching label", func() {
					_, err := vClusterClient.CoreV1().ServiceAccounts(nsName).Create(ctx,
						&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
							Name:      saName,
							Namespace: nsName,
							// no inject-pull-secrets label
						}},
						metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				By("waiting for the host SA to exist and verifying it has no imagePullSecrets", func() {
					Consistently(func(g Gomega) {
						hostSA := findHostSA(ctx, g, nsName, saName)
						g.Expect(hostSA.ImagePullSecrets).To(BeEmpty(),
							"host SA imagePullSecrets should be empty for a non-matching SA")
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})

			// Ordered: the second spec removes the SA label; it must run after the first spec
			// has verified that imagePullSecrets were initially set.
			Context("when the SA label is removed after initial sync", Ordered, func() {
				var (
					nsName string
					saName string
				)

				BeforeAll(func(ctx context.Context) {
					suffix := random.String(6)
					nsName = "sa-pull-secret-test-" + suffix
					saName = "sa-label-removed-" + suffix

					_, err := vClusterClient.CoreV1().Namespaces().Create(ctx,
						&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}},
						metav1.CreateOptions{})
					Expect(err).To(Succeed())
					DeferCleanup(func(ctx context.Context) {
						err := vClusterClient.CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{})
						if !kerrors.IsNotFound(err) {
							Expect(err).To(Succeed())
						}
					})

					_, err = vClusterClient.CoreV1().ServiceAccounts(nsName).Create(ctx,
						&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
							Name:      saName,
							Namespace: nsName,
							Labels:    map[string]string{selectorLabelKey: selectorLabelValue},
						}},
						metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				It("host SA initially has imagePullSecrets while the SA matches the selector", func(ctx context.Context) {
					By("waiting for imagePullSecrets to appear on the host SA", func() {
						Eventually(func(g Gomega) {
							hostSA := findHostSA(ctx, g, nsName, saName)
							g.Expect(hostSA.ImagePullSecrets).To(ContainElement(
								corev1.LocalObjectReference{Name: expectedPullSecret},
							), "host SA should have imagePullSecrets while selector label is set")
						}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
					})
				})

				It("host SA loses imagePullSecrets after the selector label is removed", func(ctx context.Context) {
					By("removing the selector label from the virtual SA", func() {
						sa, err := vClusterClient.CoreV1().ServiceAccounts(nsName).Get(ctx, saName, metav1.GetOptions{})
						Expect(err).To(Succeed())
						delete(sa.Labels, selectorLabelKey)
						_, err = vClusterClient.CoreV1().ServiceAccounts(nsName).Update(ctx, sa, metav1.UpdateOptions{})
						Expect(err).To(Succeed())
					})

					By("waiting for imagePullSecrets to be cleared on the host SA", func() {
						Eventually(func(g Gomega) {
							hostSA := findHostSA(ctx, g, nsName, saName)
							g.Expect(hostSA.ImagePullSecrets).To(BeEmpty(),
								"host SA imagePullSecrets should be cleared after selector label is removed")
						}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
					})
				})
			})
		},
	)
}
