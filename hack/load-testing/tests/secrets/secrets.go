package secrets

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/hack/load-testing/tests/framework"
	"github.com/loft-sh/vcluster/pkg/util/random"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestSecrets(ctx context.Context, kubeClient client.Client, amount int64, namespace string) error {
	err := framework.CreateNamespace(ctx, kubeClient, namespace)
	if err != nil {
		return err
	}

	for i := int64(0); i < amount; i++ {
		if i%int64(100) == 0 {
			klog.FromContext(ctx).Info("Creating secret", "n", i)
		}

		err = kubeClient.Create(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "secret-",
				Namespace:    namespace,
			},
			Data: map[string][]byte{
				random.String(32): []byte(random.String(1024)),
			},
		})
		if err != nil {
			if kerrors.IsAlreadyExists(err) {
				continue
			}

			return fmt.Errorf("error creating secret: %w", err)
		}
	}

	return nil
}
