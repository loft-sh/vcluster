package throughput

import (
	"context"
	"fmt"
	"time"

	"github.com/loft-sh/vcluster/hack/load-testing/stopwatch"
	"github.com/loft-sh/vcluster/hack/load-testing/tests/framework"
	"github.com/loft-sh/vcluster/pkg/util/random"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestThroughput(ctx context.Context, kubeClient client.Client, namespace string) error {
	err := framework.CreateNamespace(ctx, kubeClient, namespace)
	if err != nil {
		return err
	}

	// create secrets
	amount := 10000
	stopWatch := stopwatch.New(klog.FromContext(ctx))
	for i := 0; i < amount; i++ {
		if i%2000 == 0 {
			klog.FromContext(ctx).Info("Creating secret", "n", i)
		}

		err = kubeClient.Create(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "secret-test-" + random.String(10) + "-",
				Namespace:    namespace,
			},
			Data: map[string][]byte{
				random.String(32): []byte(random.String(1024)),
			},
		})
		if err != nil {
			return fmt.Errorf("error creating secret: %w", err)
		}
	}
	stopWatch.Stop("Create secrets", "amount", amount)

	// list secrets
	secretList := &corev1.SecretList{}
	err = kubeClient.List(ctx, secretList, client.InNamespace(namespace))
	if err != nil {
		return fmt.Errorf("error listing secrets: %w", err)
	}
	stopWatch.Stop("Listing secrets", "amount", amount)

	// update secrets
	for i, secret := range secretList.Items {
		if i%2000 == 0 {
			klog.FromContext(ctx).Info("Updating secret", "n", i)
		}

		secret.Data[random.String(32)] = []byte(random.String(1024))
		err = kubeClient.Update(ctx, &secret)
		if err != nil {
			return fmt.Errorf("error updating secret: %w", err)
		}
	}
	stopWatch.Stop("Updating secrets", "amount", amount)

	// relist secrets
	secretList = &corev1.SecretList{}
	err = kubeClient.List(ctx, secretList, client.InNamespace(namespace))
	if err != nil {
		return fmt.Errorf("error re-listing secrets: %w", err)
	}
	stopWatch.Stop("Re-Listing secrets", "amount", amount)

	// patch secrets
	for i, secret := range secretList.Items {
		if i%2000 == 0 {
			klog.FromContext(ctx).Info("Patching secret", "n", i)
		}

		oldSecret := secret.DeepCopy()
		secret.Data[random.String(32)] = []byte(random.String(1024))
		err = kubeClient.Patch(ctx, &secret, client.MergeFrom(oldSecret))
		if err != nil {
			return fmt.Errorf("error patching secret: %w", err)
		}
	}
	stopWatch.Stop("Patching secrets", "amount", amount)

	// relist secrets
	secretList = &corev1.SecretList{}
	err = kubeClient.List(ctx, secretList, client.InNamespace(namespace))
	if err != nil {
		return fmt.Errorf("error re-listing secrets: %w", err)
	}
	stopWatch.Stop("Re-re-Listing secrets", "amount", amount)

	// delete secrets
	for i, secret := range secretList.Items {
		if i%2000 == 0 {
			klog.FromContext(ctx).Info("Deleting secret", "n", i)
		}

		err = kubeClient.Delete(ctx, &secret)
		if err != nil {
			return fmt.Errorf("error delete secret: %w", err)
		}
	}
	stopWatch.Stop("Deleting secrets", "amount", amount)

	// delete namespace
	err = kubeClient.Delete(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	})
	if err != nil {
		return fmt.Errorf("error deleting namespace: %w", err)
	}
	for {
		err = kubeClient.Get(ctx, types.NamespacedName{Name: namespace}, &corev1.Namespace{})
		if err != nil {
			break
		}

		time.Sleep(time.Millisecond * 100)
	}
	stopWatch.Stop("Deleting namespace", "amount", amount)

	return nil
}
