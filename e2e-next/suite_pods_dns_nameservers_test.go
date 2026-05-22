// Suite: pods-dns-nameservers-vcluster
// vCluster: sync.toHost.pods.dns.nameservers with two host-scoped Service
// references. PreSetup creates the shared host namespace, two CoreDNS
// instances (Deployment + ConfigMap), and the two label-selected
// Services that front them so the syncer can resolve them on first
// reconcile and pods can actually verify DNS resolution against unique
// records served only by these instances.
// Run:      just run-e2e 'pr && sync'
package e2e_next

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/e2e-next/setup/lazyvcluster"
	test_core "github.com/loft-sh/vcluster/e2e-next/test_core/sync"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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

// podsDNSNameserversCoreDNSImage is the official CoreDNS image used by
// both DNS instances. Pinned so behavior does not drift with upstream.
const podsDNSNameserversCoreDNSImage = "coredns/coredns:1.11.3"

//go:embed vcluster-pods-dns-nameservers.yaml
var podsDNSNameserversVClusterYAML string

const podsDNSNameserversVClusterName = "pods-dns-nameservers-vcluster"

func init() { suitePodsDNSNameserversVCluster() }

func suitePodsDNSNameserversVCluster() {
	// Ordered: BeforeAll must complete before any spec runs so the vCluster
	// and host namespace exist. Specs themselves are independent — each
	// recreates its own pods via BeforeEach and shares no sequential state.
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

// podsDNSNameserversPreSetup provisions the host namespace, two CoreDNS
// instances (one per DNS service), and the two label-selected ClusterIP
// Services that front them. Runs before the vCluster starts so the
// syncer sees both Services on first reconcile and pods can verify DNS
// resolution against the per-instance custom records.
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

	for _, inst := range podsDNSNameserversInstances() {
		if err := podsDNSNameserversCreateInstance(ctx, hostClient, inst); err != nil {
			return err
		}
	}

	// Wait for both CoreDNS Deployments to be Available before the
	// vCluster starts: the syncer resolves the Services at sync time and
	// pods later exec nslookup against the resolved IPs.
	if err := podsDNSNameserversWaitForDeployments(ctx, hostClient); err != nil {
		return err
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
// the pre-setup and waits for termination. The next spec's BeforeEach
// re-creates the namespace immediately, so leaving it in Terminating state
// would race the next pre-setup with "forbidden: namespace is being
// terminated".
func podsDNSNameserversCleanup(ctx context.Context) {
	GinkgoHelper()
	hostCluster := cluster.From(ctx, constants.GetHostClusterName())
	if hostCluster == nil {
		return
	}
	hostClient, err := kubernetes.NewForConfig(hostCluster.KubernetesRestConfig())
	Expect(err).To(Succeed())

	err = hostClient.CoreV1().Namespaces().Delete(ctx, test_core.PodDNSNameserversNamespace, metav1.DeleteOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		Expect(err).To(Succeed())
	}

	pollCtx, cancel := context.WithTimeout(ctx, constants.PollingTimeoutLong)
	defer cancel()
	for {
		_, err := hostClient.CoreV1().Namespaces().Get(pollCtx, test_core.PodDNSNameserversNamespace, metav1.GetOptions{})
		if kerrors.IsNotFound(err) {
			return
		}
		select {
		case <-pollCtx.Done():
			Expect(pollCtx.Err()).To(Succeed(), "namespace %s did not finish terminating", test_core.PodDNSNameserversNamespace)
			return
		case <-time.After(constants.PollingInterval):
		}
	}
}

// podsDNSNameserversInstance describes one CoreDNS instance: a
// ConfigMap with a Corefile answering a unique A-record, a Deployment
// running CoreDNS with that Corefile, and a Service that selects the
// Deployment's pods. The Service carries the label the syncer
// configuration selects on.
type podsDNSNameserversInstance struct {
	serviceName string
	selectorVal string
	corefile    string
}

func podsDNSNameserversInstances() []podsDNSNameserversInstance {
	return []podsDNSNameserversInstance{
		{
			serviceName: test_core.PodDNSNameserversPrimaryService,
			selectorVal: "primary",
			corefile: `. {
    hosts {
        ` + test_core.PodDNSNameserversPrimaryExpectedIP + ` ` + test_core.PodDNSNameserversPrimaryExpectedName + `
    }
    log
    errors
    forward . /etc/resolv.conf
}
`,
		},
		{
			serviceName: test_core.PodDNSNameserversSecondaryService,
			selectorVal: "secondary",
			corefile: `. {
    hosts {
        ` + test_core.PodDNSNameserversSecondaryExpectedIP + ` ` + test_core.PodDNSNameserversSecondaryExpectedName + `
    }
    log
    errors
    forward . /etc/resolv.conf
}
`,
		},
	}
}

func podsDNSNameserversCreateInstance(ctx context.Context, hostClient kubernetes.Interface, inst podsDNSNameserversInstance) error {
	ns := test_core.PodDNSNameserversNamespace
	podLabels := map[string]string{"app": inst.serviceName}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      inst.serviceName,
			Namespace: ns,
		},
		Data: map[string]string{"Corefile": inst.corefile},
	}
	if _, err := hostClient.CoreV1().ConfigMaps(ns).Create(ctx, cm, metav1.CreateOptions{}); err != nil && !kerrors.IsAlreadyExists(err) {
		return fmt.Errorf("create ConfigMap %s: %w", cm.Name, err)
	}

	replicas := int32(1)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      inst.serviceName,
			Namespace: ns,
			Labels:    podLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: podLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: podLabels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "coredns",
						Image: podsDNSNameserversCoreDNSImage,
						Args:  []string{"-conf", "/etc/coredns/Corefile"},
						Ports: []corev1.ContainerPort{
							{Name: "dns", ContainerPort: 53, Protocol: corev1.ProtocolUDP},
							{Name: "dns-tcp", ContainerPort: 53, Protocol: corev1.ProtocolTCP},
						},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "config",
							MountPath: "/etc/coredns",
						}},
					}},
					Volumes: []corev1.Volume{{
						Name: "config",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: inst.serviceName},
							},
						},
					}},
				},
			},
		},
	}
	if _, err := hostClient.AppsV1().Deployments(ns).Create(ctx, dep, metav1.CreateOptions{}); err != nil && !kerrors.IsAlreadyExists(err) {
		return fmt.Errorf("create Deployment %s: %w", dep.Name, err)
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      inst.serviceName,
			Namespace: ns,
			Labels:    map[string]string{test_core.PodDNSNameserversLabelKey: inst.selectorVal},
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: podLabels,
			Ports: []corev1.ServicePort{
				{Name: "dns", Port: 53, Protocol: corev1.ProtocolUDP, TargetPort: intstr.FromInt(53)},
				{Name: "dns-tcp", Port: 53, Protocol: corev1.ProtocolTCP, TargetPort: intstr.FromInt(53)},
			},
		},
	}
	if _, err := hostClient.CoreV1().Services(ns).Create(ctx, svc, metav1.CreateOptions{}); err != nil && !kerrors.IsAlreadyExists(err) {
		return fmt.Errorf("create service %s: %w", svc.Name, err)
	}

	return nil
}

// podsDNSNameserversWaitForDeployments polls until both CoreDNS
// Deployments report Available. Returning before that risks the
// resolver picking up Services with no backing pods, so the e2e
// nslookup verification would race the rollout.
func podsDNSNameserversWaitForDeployments(ctx context.Context, hostClient kubernetes.Interface) error {
	deadline := constants.PollingTimeoutLong
	pollCtx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()

	for _, inst := range podsDNSNameserversInstances() {
		name := inst.serviceName
		for {
			dep, err := hostClient.AppsV1().Deployments(test_core.PodDNSNameserversNamespace).Get(pollCtx, name, metav1.GetOptions{})
			if err == nil {
				available := false
				for _, c := range dep.Status.Conditions {
					if c.Type == appsv1.DeploymentAvailable && c.Status == corev1.ConditionTrue {
						available = true
						break
					}
				}
				if available && dep.Status.ReadyReplicas >= 1 {
					break
				}
			}
			select {
			case <-pollCtx.Done():
				return fmt.Errorf("timed out waiting for Deployment %s to become Available: %w", name, pollCtx.Err())
			case <-time.After(constants.PollingInterval):
			}
		}
	}
	return nil
}
