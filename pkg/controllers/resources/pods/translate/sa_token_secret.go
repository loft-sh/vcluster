package translate

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	SkipSATokenSecretBacksyncAnnotation = "vcluster.loft.sh/skip-sa-secret-backsync"
)

var PodServiceAccountTokenSecretName string

func SecretNameFromPodName(podName string) string {
	return fmt.Sprintf("%s-sa-token", podName)
}

func checkIfSecretExists(ctx context.Context, vClient client.Client, podName, namespace string) (bool, error) {
	secret := &corev1.Secret{}

	err := vClient.Get(ctx, types.NamespacedName{
		Name:      SecretNameFromPodName(podName),
		Namespace: namespace,
	}, secret)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func SATokenSecret(ctx context.Context, vClient client.Client, vPod *corev1.Pod, tokens map[string]string) error {
	exists, err := checkIfSecretExists(ctx, vClient, vPod.Name, vPod.Namespace)
	if err != nil {
		return err
	}

	if !exists {
		// create to secret with the given token
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      SecretNameFromPodName(vPod.Name),
				Namespace: vPod.Namespace,

				Annotations: map[string]string{
					translate.SkipBacksyncInMultiNamespaceMode: "true",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: corev1.SchemeGroupVersion.Version,
						Kind:       vPod.Kind,
						Name:       vPod.Name,
						UID:        vPod.UID,
					},
				},
			},
			Type:       corev1.SecretTypeOpaque,
			StringData: tokens,
		}

		err := vClient.Create(ctx, secret)
		if err != nil {
			return err
		}

		return nil
	}

	return nil
}
