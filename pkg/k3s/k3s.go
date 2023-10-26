package k3s

import (
	"context"
	"fmt"
	"os"

	"github.com/loft-sh/vcluster/pkg/util/random"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var tokenPath = "/data/server/token"

func EnsureK3SToken(ctx context.Context, currentNamespaceClient kubernetes.Interface, currentNamespace, vClusterName string) error {
	// check if secret exists
	secretName := fmt.Sprintf("vc-k3s-%s", vClusterName)
	_, err := currentNamespaceClient.CoreV1().Secrets(currentNamespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	} else if err == nil {
		return nil
	}

	// try to read token file (migration case)
	token, err := os.ReadFile(tokenPath)
	if err != nil {
		token = []byte(random.String(64))
	}

	// create k3s secret
	_, err = currentNamespaceClient.CoreV1().Secrets(currentNamespace).Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: currentNamespace,
		},
		Data: map[string][]byte{
			"token": token,
		},
		Type: corev1.SecretTypeOpaque,
	}, metav1.CreateOptions{})
	if err != nil && !kerrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}
