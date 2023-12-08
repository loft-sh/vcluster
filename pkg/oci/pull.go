package oci

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/loft-sh/vcluster/pkg/etcd"
	"github.com/loft-sh/vcluster/pkg/registry"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

const PullOciImageEnv = "PULL_OCI_IMAGE"

var alreadyRestoredPath = path.Join("/data", "restored.vcluster")

func Pull(
	ctx context.Context,
	hostClient kubernetes.Interface,
	vClusterNamespace string,
) error {
	// check if env var is set
	target := os.Getenv(PullOciImageEnv)
	if target == "" {
		return nil
	}
	klog.Infof("Try to pull vCluster from %s", target)

	// check if restored already
	_, err := os.Stat(alreadyRestoredPath)
	if err == nil {
		klog.Info("vCluster was already pulled")
		return nil
	}

	// try to fetch username & password
	username, password, err := fetchUserPassword(ctx, hostClient, vClusterNamespace)
	if err != nil {
		return err
	}

	// pull image
	etcdReader, err := registry.Pull(
		ctx,
		target,
		username,
		password,
	)
	if err != nil {
		return err
	}

	// insert etcd data
	err = etcd.Restore(ctx, etcdReader)
	if err != nil {
		return fmt.Errorf("restore etcd: %w", err)
	}

	// TODO: restore PVCs

	// make sure we don't restore another time
	err = os.WriteFile(alreadyRestoredPath, nil, 0666)
	if err != nil {
		return err
	}

	return nil
}

func PullConfig(
	ctx context.Context,
	target string,
) (*registry.VClusterConfig, error) {
	ref, err := name.ParseReference(target)
	if err != nil {
		return nil, err
	}

	username := ""
	password := ""
	authConfig, err := registry.GetAuthConfig(ref.Context().RegistryStr())
	if err == nil && authConfig != nil {
		username = authConfig.Username
		password = authConfig.Secret
	}

	img, err := remote.Image(ref, remote.WithContext(ctx), remote.WithAuth(&authn.Basic{
		Username: username,
		Password: password,
	}))
	if err != nil {
		return nil, err
	}

	configReader, err := registry.FindLayerWithMediaType(img, registry.ConfigMediaType)
	if err != nil {
		return nil, err
	}
	defer configReader.Close()

	vClusterConfig := &registry.VClusterConfig{}
	err = json.NewDecoder(configReader).Decode(vClusterConfig)
	if err != nil {
		return nil, fmt.Errorf("decode vCluster config: %w", err)
	}

	return vClusterConfig, nil
}

func CreateUserPasswordForRegistry(ctx context.Context, client kubernetes.Interface, name, namespace string, destination string) error {
	registryName, _, _, err := ParseReference(destination)
	if err != nil {
		return err
	}

	secretName := name + "-pull-credentials"
	authConfig, err := registry.GetAuthConfig(registryName)
	if err == nil && authConfig != nil && authConfig.Username != "" {
		pullSecret, err := client.CoreV1().Secrets(namespace).Get(ctx, name+"-pull-credentials", metav1.GetOptions{})
		if err != nil && !kerrors.IsNotFound(err) {
			return fmt.Errorf("get pull credentials: %w", err)
		} else if err == nil {
			// update
			if pullSecret.Data == nil {
				pullSecret.Data = make(map[string][]byte)
			}

			pullSecret.Data["username"] = []byte(authConfig.Username)
			pullSecret.Data["password"] = []byte(authConfig.Secret)
			_, err = client.CoreV1().Secrets(namespace).Update(ctx, pullSecret, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("update pull credentials: %w", err)
			}
		} else if err != nil {
			_, err = client.CoreV1().Secrets(namespace).Create(ctx, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: namespace,
				},
				Data: map[string][]byte{
					"username": []byte(authConfig.Username),
					"password": []byte(authConfig.Secret),
				},
			}, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("create pull credentials: %w", err)
			}
		}
	} else {
		err = client.CoreV1().Secrets(namespace).Delete(ctx, secretName, metav1.DeleteOptions{})
		if err != nil && !kerrors.IsNotFound(err) {
			return fmt.Errorf("delete pull credentials: %w", err)
		}
	}

	return nil
}

func fetchUserPassword(ctx context.Context, client kubernetes.Interface, namespace string) (string, string, error) {
	username := ""
	password := ""
	pullSecret, err := client.CoreV1().Secrets(namespace).Get(ctx, translate.Suffix+"-pull-credentials", metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return "", "", fmt.Errorf("get pull credentials: %w", err)
	} else if err == nil && pullSecret.Data != nil {
		username = string(pullSecret.Data["username"])
		password = string(pullSecret.Data["password"])
	}

	return username, password, nil
}
