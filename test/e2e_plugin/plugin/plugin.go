package plugin

import (
	"time"

	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	pollingInterval     = time.Second * 2
	pollingDurationLong = time.Second * 60
)

var _ = ginkgo.Describe("plugin", func() {
	f := framework.DefaultFramework

	ginkgo.It("test legacy vCluster plugin", func() {
		// check if deployment is there
		gomega.Eventually(func() bool {
			_, err := f.VClusterClient.AppsV1().Deployments("default").Get(f.Context, "mydeployment", metav1.GetOptions{})
			return err == nil
		}).
			WithPolling(pollingInterval).
			WithTimeout(pollingDurationLong).
			Should(gomega.BeTrue())
	})

	ginkgo.It("check hooks work correctly", func() {
		// create a new service
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test123",
				Namespace: "default",
			},
			Spec: corev1.ServiceSpec{
				Type: corev1.ServiceTypeClusterIP,
				Ports: []corev1.ServicePort{
					{
						Name: "test",
						Port: int32(1000),
					},
				},
			},
		}

		// create service
		err := f.VClusterCRClient.Create(f.Context, service)
		framework.ExpectNoError(err)

		// wait for service to become synced
		hostService := &corev1.Service{}
		gomega.Eventually(func() bool {
			err := f.HostCRClient.Get(f.Context, translate.Default.HostName(nil, service.Name, service.Namespace), hostService)
			return err == nil
		}).
			WithPolling(pollingInterval).
			WithTimeout(pollingDurationLong).
			Should(gomega.BeTrue())

		// check if service is synced correctly
		framework.ExpectEqual(len(hostService.Spec.Ports), 2)
		framework.ExpectEqual(hostService.Spec.Ports[1].Name, "plugin")
		framework.ExpectEqual(hostService.Spec.Ports[1].Port, int32(19000))
	})

	ginkgo.It("check secret is imported correctly", func() {
		// create a new secret
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test123",
				Namespace: f.VclusterNamespace,
				Annotations: map[string]string{
					"vcluster.loft.sh/import": "test/test",
				},
			},
			Data: map[string][]byte{
				"test": []byte("test"),
			},
		}

		// create secret
		err := f.HostCRClient.Create(f.Context, secret)
		framework.ExpectNoError(err)

		// wait for secret to become synced
		vSecret := &corev1.Secret{}
		gomega.Eventually(func() bool {
			err := f.VClusterCRClient.Get(f.Context, types.NamespacedName{Name: "test", Namespace: "test"}, vSecret)
			return err == nil
		}).
			WithPolling(pollingInterval).
			WithTimeout(pollingDurationLong).
			Should(gomega.BeTrue())

		// check if secret is synced correctly
		framework.ExpectEqual(len(vSecret.Data), 1)
		framework.ExpectEqual(vSecret.Data["test"], []byte("test"))

		// change secret
		secret.Data["test"] = []byte("newtest")
		err = f.HostCRClient.Update(f.Context, secret)
		framework.ExpectNoError(err)

		// wait for update
		gomega.Eventually(func() bool {
			err := f.VClusterCRClient.Get(f.Context, types.NamespacedName{Name: "test", Namespace: "test"}, vSecret)
			return err == nil && string(vSecret.Data["test"]) == "newtest"
		}).
			WithPolling(pollingInterval).
			WithTimeout(pollingDurationLong).
			Should(gomega.BeTrue())

		// delete secret
		err = f.HostCRClient.Delete(f.Context, secret)
		framework.ExpectNoError(err)

		// wait for delete within vCluster
		gomega.Eventually(func() bool {
			err := f.VClusterCRClient.Get(f.Context, types.NamespacedName{Name: "test", Namespace: "test"}, vSecret)
			return kerrors.IsNotFound(err)
		}).
			WithPolling(pollingInterval).
			WithTimeout(pollingDurationLong).
			Should(gomega.BeTrue())
	})
})
