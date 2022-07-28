package e2etargetnamespace

import (
	"time"

	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var _ = ginkgo.Describe("Target Namespace", func() {
	f := framework.DefaultFramework
	ginkgo.It("Create vcluster with target namespace", func() {
		ginkgo.By("Create target namespace")
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "vcluster-workload",
			},
		}
		_, err := f.HostClient.CoreV1().Namespaces().Create(f.Context, ns, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		err = wait.Poll(time.Second, time.Minute*1, func() (done bool, err error) {
			namespace, _ := f.HostClient.CoreV1().Namespaces().Get(f.Context, ns.Name, metav1.GetOptions{})
			if namespace.Status.Phase == corev1.NamespaceActive {
				return true, nil
			}
			return false, nil
		})
		framework.ExpectNoError(err)

		ginkgo.By("Create service account, role and role binding in target namespace")
		workloadSaName := "vc-workload-" + f.VclusterName
		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      workloadSaName,
				Namespace: ns.Name,
			},
		}
		_, err = f.HostClient.CoreV1().ServiceAccounts(ns.Name).Create(f.Context, sa, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		role := &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vcluster-workload",
				Namespace: ns.Name,
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"", "networking.k8s.io"},
					Resources: []string{"*"},
					Verbs:     []string{"*"},
				},
			},
		}
		_, err = f.HostClient.RbacV1().Roles(ns.Name).Create(f.Context, role, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		vcSaName := "vc-" + f.VclusterName
		rb := &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vcluster-workload",
				Namespace: ns.Name,
				Labels: map[string]string{
					"app": "vcluster-nginxa-app",
				},
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      vcSaName,
					Namespace: f.VclusterNamespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     role.Name,
			},
		}
		_, err = f.HostClient.RbacV1().RoleBindings(ns.Name).Create(f.Context, rb, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		ginkgo.By("Create workload in vcluster and verify if it's running in targeted namespace")
		pod := &corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "nginx",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "nginx",
						Image: "nginx",
					},
				},
			},
		}

		_, err = f.VclusterClient.CoreV1().Pods("default").Create(f.Context, pod, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		err = wait.Poll(time.Second, time.Minute*2, func() (bool, error) {
			p, _ := f.VclusterClient.CoreV1().Pods("default").Get(f.Context, "nginx", metav1.GetOptions{})
			if p.Status.Phase == corev1.PodRunning {
				return true, nil
			}
			return false, nil
		})
		framework.ExpectNoError(err)

		p, err := f.HostClient.CoreV1().Pods(ns.Name).List(f.Context, metav1.ListOptions{
			LabelSelector: "vcluster.loft.sh/managed-by=" + f.VclusterName,
		})
		framework.ExpectNoError(err)
		framework.ExpectEqual(true, len(p.Items) > 0)
	})
})
