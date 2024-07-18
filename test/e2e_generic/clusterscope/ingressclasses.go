package clusterscope

import (
	"context"
	"time"

	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	IngressClassName           = "vcluster-test-ingress-class"
	IngressClassControllerName = "vcluster.loft.sh/vcluster-test-ingress-controller"
)

var _ = ginkgo.Describe("Generic sync cluster scoped resources", func() {
	f := framework.DefaultFramework

	ginkgo.It("sync cluster scoped resource into vcluster", func() {
		ctx := f.Context
		// create an ingress class in host cluster
		_, err := f.HostClient.NetworkingV1().IngressClasses().Create(ctx, &networkingv1.IngressClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: IngressClassName,
			},
			Spec: networkingv1.IngressClassSpec{
				Controller: IngressClassControllerName,
			},
		}, metav1.CreateOptions{})

		framework.ExpectNoError(err)

		var ingClass *networkingv1.IngressClass

		err = wait.PollUntilContextTimeout(ctx, time.Millisecond*500, framework.PollTimeout, true, func(ctx context.Context) (bool, error) {
			ingClass, err = f.VClusterClient.NetworkingV1().IngressClasses().Get(ctx, IngressClassName, metav1.GetOptions{})
			if err != nil {
				if kerrors.IsNotFound(err) {
					return false, nil
				}

				return false, err
			}

			return true, nil
		})

		// check for ingress class controller name
		framework.ExpectEqual(ingClass.Spec.Controller, IngressClassControllerName)
	})

	ginkgo.It("deleting virtual cluster scoped object doesn't delete the physical", func() {
		ctx := f.Context

		err := f.VClusterClient.NetworkingV1().IngressClasses().Delete(ctx, IngressClassName, metav1.DeleteOptions{})
		framework.ExpectNoError(err)

		// should not delete the physical ingress class
		err = wait.PollUntilContextTimeout(ctx, time.Millisecond*500, framework.PollTimeout, true, func(ctx context.Context) (bool, error) {
			_, err = f.HostClient.NetworkingV1().IngressClasses().Get(ctx, IngressClassName, metav1.GetOptions{})
			if err != nil {
				if kerrors.IsNotFound(err) {
					return false, err
				}

				return false, nil
			}

			return true, nil
		})

		framework.ExpectNoError(err)
	})

	ginkgo.It("deleting physical cluster scoped object deletes virtual object", func() {
		ctx := f.Context

		err := f.HostClient.NetworkingV1().IngressClasses().Delete(ctx, IngressClassName, metav1.DeleteOptions{})
		framework.ExpectNoError(err)

		err = wait.PollUntilContextTimeout(ctx, time.Millisecond*500, framework.PollTimeout, true, func(ctx context.Context) (bool, error) {
			_, err = f.VClusterClient.NetworkingV1().IngressClasses().Get(ctx, IngressClassName, metav1.GetOptions{})
			if kerrors.IsNotFound(err) {
				return true, nil
			}

			return false, nil
		})

		framework.ExpectNoError(err)
	})
})
