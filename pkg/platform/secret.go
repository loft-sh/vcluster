package platform

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const DefaultPlatformSecretName = "vcluster-platform-api-key"

func ApplyPlatformSecret(
	ctx context.Context,
	config *config.CLI,
	kubeClient kubernetes.Interface,
	importName,
	namespace,
	project,
	accessKey,
	host string,
	insecure bool,
	certificateAuthorityData []byte,
) error {
	var err error
	accessKey, host, insecure, err = getAccessKeyAndHost(ctx, config, accessKey, host, insecure)
	if err != nil {
		return fmt.Errorf("get access key and host: %w", err)
	}

	// build secret payload
	payload := map[string][]byte{
		"accessKey":                []byte(accessKey),
		"host":                     []byte(strings.TrimPrefix(host, "https://")),
		"insecure":                 []byte(strconv.FormatBool(insecure)),
		"certificateAuthorityData": certificateAuthorityData,
	}
	if project != "" {
		payload["project"] = []byte(project)
	}
	if importName != "" {
		payload["name"] = []byte(importName)
	}

	// check if secret already exists
	keySecret, err := kubeClient.CoreV1().Secrets(namespace).Get(ctx, DefaultPlatformSecretName, metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return fmt.Errorf("error getting platform secret %s/%s: %w", namespace, DefaultPlatformSecretName, err)
	} else if kerrors.IsNotFound(err) {
		_, err = kubeClient.CoreV1().Secrets(namespace).Create(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      DefaultPlatformSecretName,
				Namespace: namespace,
			},
			Data: payload,
		}, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("error creating platform secret %s/%s: %w", namespace, DefaultPlatformSecretName, err)
		}

		return nil
	} else if reflect.DeepEqual(keySecret.Data, payload) {
		// no update needed, just return
		return nil
	}

	// create the patch
	patch := ctrlclient.MergeFrom(keySecret.DeepCopy())
	keySecret.Data = payload
	patchBytes, err := patch.Data(keySecret)
	if err != nil {
		return fmt.Errorf("error creating patch for platform secret %s/%s: %w", namespace, DefaultPlatformSecretName, err)
	}

	// patch the secret
	_, err = kubeClient.CoreV1().Secrets(namespace).Patch(ctx, keySecret.Name, patch.Type(), patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("error patching platform secret %s/%s: %w", namespace, DefaultPlatformSecretName, err)
	}

	return nil
}

func getAccessKeyAndHost(ctx context.Context, config *config.CLI, accessKey, host string, insecure bool) (string, string, bool, error) {
	if host != "" && accessKey != "" {
		return accessKey, host, insecure, nil
	}

	platformClient, err := InitClientFromConfig(ctx, config)
	if err != nil {
		return "", "", false, err
	}
	if host == "" {
		host = strings.TrimPrefix(platformClient.Config().Platform.Host, "https://")
	}
	if !insecure {
		insecure = platformClient.Config().Platform.Insecure
	}

	managementClient, err := platformClient.Management()
	if err != nil {
		return "", "", false, fmt.Errorf("create management client: %w", err)
	}

	// check if we need to find access key
	if accessKey != "" {
		return accessKey, host, insecure, nil
	}

	// is the access key still valid?
	platformConfig := platformClient.Config().Platform
	if platformConfig.VirtualClusterAccessKey != "" {
		selfCtx, cancel := context.WithTimeout(ctx, 1*time.Minute)
		self, err := managementClient.Loft().ManagementV1().Selves().Create(selfCtx, &managementv1.Self{
			Spec: managementv1.SelfSpec{
				AccessKey: platformConfig.VirtualClusterAccessKey,
			},
		}, metav1.CreateOptions{})
		cancel()
		if err != nil || self.Status.Subject != platformClient.Self().Status.Subject {
			platformConfig.VirtualClusterAccessKey = ""
		}
	}

	// check if we need to create a virtual cluster access key
	if platformConfig.VirtualClusterAccessKey == "" {
		user := ""
		team := ""
		if platformClient.Self().Status.User != nil {
			user = platformClient.Self().Status.User.Name
		}
		if platformClient.Self().Status.Team != nil {
			team = platformClient.Self().Status.Team.Name
		}

		accessKey, err := managementClient.Loft().ManagementV1().OwnedAccessKeys().Create(ctx, &managementv1.OwnedAccessKey{
			Spec: managementv1.OwnedAccessKeySpec{
				AccessKeySpec: storagev1.AccessKeySpec{
					DisplayName: "vCluster CLI Activation Key",
					User:        user,
					Team:        team,
					Scope: &storagev1.AccessKeyScope{
						Roles: []storagev1.AccessKeyScopeRole{
							{
								Role: storagev1.AccessKeyScopeRoleVCluster,
							},
						},
					},
				},
			},
		}, metav1.CreateOptions{})
		if err != nil {
			return "", "", false, fmt.Errorf("create owned access key: %w", err)
		}

		platformConfig.VirtualClusterAccessKey = accessKey.Spec.Key
		platformClient.Config().Platform = platformConfig
		if err := platformClient.Save(); err != nil {
			return "", "", false, fmt.Errorf("save vCluster platform config: %w", err)
		}
	}

	return platformConfig.VirtualClusterAccessKey, host, insecure, nil
}
