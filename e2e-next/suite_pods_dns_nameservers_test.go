// Suite: pods-dns-nameservers-vcluster
// vCluster: sync.toHost.pods.dns.nameservers with two host-scoped Service
// references. PreSetup creates the shared host namespace + DNS Services
// before the vCluster starts so the syncer can resolve them on first
// reconcile. Lifecycle owned by this Describe's BeforeAll + DeferCleanup.
// Run:      just run-e2e 'pr && sync'
package e2e_next

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/e2e-next/setup/lazyvcluster"
	test_core "github.com/loft-sh/vcluster/e2e-next/test_core/sync"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// podsDNSNameserversSyncerNamespace is the host namespace the vCluster
// CLI installs the StatefulSet into (vcluster-<name>). The syncer
// ServiceAccount living there is what needs Service read access in the
// target namespace.
const podsDNSNameserversSyncerNamespace = "vcluster-" + podsDNSNameserversVClusterName

// podsDNSNameserversSyncerServiceAccount is the syncer Pod's
// ServiceAccount, named vc-<vcluster-name>.
const podsDNSNameserversSyncerServiceAccount = "vc-" + podsDNSNameserversVClusterName

//go:embed vcluster-pods-dns-nameservers.yaml
var podsDNSNameserversVClusterYAML string

const podsDNSNameserversVClusterName = "pods-dns-nameservers-vcluster"

func init() { suitePodsDNSNameserversVCluster() }

// Ordered because all It specs depend on the vCluster started by BeforeAll;
// BeforeAll must complete before any spec runs.
func suitePodsDNSNameserversVCluster() {
	Describe("pods-dns-nameservers-vcluster", labels.PR, labels.Sync, Ordered,
		cluster.Use(clusters.HostCluster),
		func() {
			BeforeAll(func(ctx context.Context) context.Context {
				DeferCleanup(podsDNSNameserversCleanup)
				return lazyvcluster.LazyVCluster(ctx,
					podsDNSNameserversVClusterName,
					podsDNSNameserversVClusterYAML,
					lazyvcluster.WithPreSetup(podsDNSNameserversPreSetup),
				)
			})

			test_core.PodDNSNameserversSpec()
		},
	)
}

// podsDNSNameserversPreSetup provisions the host namespace and the two
// label-selected ClusterIP Services that the vCluster's
// sync.toHost.pods.dns.nameservers entries resolve to. Runs before the
// vCluster starts so the syncer sees both Services on first reconcile.
func podsDNSNameserversPreSetup(ctx context.Context) error {
	hostCluster := cluster.From(ctx, constants.GetHostClusterName())
	if hostCluster == nil {
		return fmt.Errorf("host cluster %s not found in context", constants.GetHostClusterName())
	}
	hostClient, err := kubernetes.NewForConfig(hostCluster.KubernetesRestConfig())
	if err != nil {
		return fmt.Errorf("create host client: %w", err)
	}

	_, err = hostClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: test_core.PodDNSNameserversNamespace},
	}, metav1.CreateOptions{})
	if err != nil && !kerrors.IsAlreadyExists(err) {
		return fmt.Errorf("create namespace %s: %w", test_core.PodDNSNameserversNamespace, err)
	}

	for _, svc := range podsDNSNameserversServices() {
		_, err = hostClient.CoreV1().Services(test_core.PodDNSNameserversNamespace).Create(ctx, svc, metav1.CreateOptions{})
		if err != nil && !kerrors.IsAlreadyExists(err) {
			return fmt.Errorf("create service %s: %w", svc.Name, err)
		}
	}

	// The host-scoped DNS resolver runs an informer cache against the
	// target namespace from inside the syncer Pod, so its ServiceAccount
	// needs Service read access there. The Role + RoleBinding may
	// reference an SA that does not yet exist; once the StatefulSet
	// creates it, the binding starts authorizing immediately.
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vcluster-dns-test-svc-reader",
			Namespace: test_core.PodDNSNameserversNamespace,
		},
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{""},
			Resources: []string{"services"},
			Verbs:     []string{"get", "list", "watch"},
		}},
	}
	if _, err := hostClient.RbacV1().Roles(test_core.PodDNSNameserversNamespace).Create(ctx, role, metav1.CreateOptions{}); err != nil && !kerrors.IsAlreadyExists(err) {
		return fmt.Errorf("create Role %s: %w", role.Name, err)
	}

	binding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vcluster-dns-test-svc-reader",
			Namespace: test_core.PodDNSNameserversNamespace,
		},
		Subjects: []rbacv1.Subject{{
			Kind:      rbacv1.ServiceAccountKind,
			Name:      podsDNSNameserversSyncerServiceAccount,
			Namespace: podsDNSNameserversSyncerNamespace,
		}},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     role.Name,
		},
	}
	if _, err := hostClient.RbacV1().RoleBindings(test_core.PodDNSNameserversNamespace).Create(ctx, binding, metav1.CreateOptions{}); err != nil && !kerrors.IsAlreadyExists(err) {
		return fmt.Errorf("create RoleBinding %s: %w", binding.Name, err)
	}

	return nil
}

// podsDNSNameserversCleanup removes the shared host namespace created by
// the pre-setup. Registered in BeforeAll so it runs after all specs and
// after the vCluster itself is torn down.
func podsDNSNameserversCleanup(ctx context.Context) {
	hostCluster := cluster.From(ctx, constants.GetHostClusterName())
	if hostCluster == nil {
		return
	}
	hostClient, err := kubernetes.NewForConfig(hostCluster.KubernetesRestConfig())
	Expect(err).To(Succeed())

	err = hostClient.CoreV1().Namespaces().Delete(ctx, test_core.PodDNSNameserversNamespace, metav1.DeleteOptions{})
	if !kerrors.IsNotFound(err) {
		Expect(err).To(Succeed())
	}
}

// podsDNSNameserversServices returns the two ClusterIP Services that
// sync.toHost.pods.dns.nameservers resolves via label selector.
func podsDNSNameserversServices() []*corev1.Service {
	return []*corev1.Service{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   test_core.PodDNSNameserversPrimaryService,
				Labels: map[string]string{test_core.PodDNSNameserversLabelKey: "primary"},
			},
			Spec: corev1.ServiceSpec{
				Type: corev1.ServiceTypeClusterIP,
				Ports: []corev1.ServicePort{
					{Name: "dns", Port: 53, Protocol: corev1.ProtocolUDP},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   test_core.PodDNSNameserversSecondaryService,
				Labels: map[string]string{test_core.PodDNSNameserversLabelKey: "secondary"},
			},
			Spec: corev1.ServiceSpec{
				Type: corev1.ServiceTypeClusterIP,
				Ports: []corev1.ServicePort{
					{Name: "dns", Port: 53, Protocol: corev1.ProtocolUDP},
				},
			},
		},
	}
}
