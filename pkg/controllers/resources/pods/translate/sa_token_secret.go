package translate

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	PodKind string = "Pod"
)

var PodServiceAccountTokenSecretName string

func SecretNameFromPodName(podName, namespace string) string {
	return translate.Default.PhysicalName(fmt.Sprintf("%s-sa-token", podName), namespace)
}

func GetSecretIfExists(ctx context.Context, pClient client.Client, vPodName, vNamespace string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := pClient.Get(ctx, types.NamespacedName{
		Name:      SecretNameFromPodName(vPodName, vNamespace),
		Namespace: translate.Default.PhysicalNamespace(vNamespace),
	}, secret)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil, nil
		}

		return nil, err
	}

	return secret, nil
}

func SATokenSecret(ctx context.Context, pClient client.Client, vPod *corev1.Pod, tokens map[string]string) error {
	existingSecret, err := GetSecretIfExists(ctx, pClient, vPod.Name, vPod.Namespace)
	if err != nil {
		return err
	}

	// check if we need to delete the secret
	if existingSecret != nil {
		err = pClient.Delete(ctx, existingSecret)
		if err != nil && !kerrors.IsNotFound(err) {
			return err
		}

		existingSecret = nil
	}

	// secret does not exist we need to create it
	if existingSecret == nil {
		// create to secret with the given token
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      SecretNameFromPodName(vPod.Name, vPod.Namespace),
				Namespace: translate.Default.PhysicalNamespace(vPod.Namespace),

				Annotations: map[string]string{
					translate.SkipBackSyncInMultiNamespaceMode: "true",
				},
			},
			Type:       corev1.SecretTypeOpaque,
			StringData: tokens,
		}
		if translate.Owner != nil {
			secret.SetOwnerReferences(translate.GetOwnerReference(nil))
		}

		// create the service account secret
		err = pClient.Create(ctx, secret)
		if err != nil {
			return err
		}

		return nil
	}

	return nil
}

func SetPodAsOwner(ctx context.Context, pPod *corev1.Pod, pClient client.Client, secret *corev1.Secret) error {
	podOwnerReference := metav1.OwnerReference{
		APIVersion: corev1.SchemeGroupVersion.Version,
		Kind:       PodKind,
		Name:       pPod.GetName(),
		UID:        pPod.GetUID(),
	}

	owners := secret.GetOwnerReferences()
	if translate.Owner != nil {
		// check if the current owner is the vcluster service
		for i, owner := range owners {
			if owner.UID == translate.Owner.GetUID() {
				// path this with current pod as owner instead
				secret.ObjectMeta.OwnerReferences[i] = podOwnerReference
				break
			}
		}
	} else {
		// check if pod is already correctly set as one of the owners
		for _, owner := range owners {
			if equality.Semantic.DeepEqual(owner, podOwnerReference) {
				// no update needed
				return nil
			}
		}

		// pod not set as owner update accordingly
		secret.ObjectMeta.OwnerReferences = append(secret.ObjectMeta.OwnerReferences, podOwnerReference)
	}

	return pClient.Update(ctx, secret)
}
