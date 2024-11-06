package platform

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/blang/semver"
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/platform/kube"
	"github.com/loft-sh/vcluster/pkg/projectutil"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/loft-sh/vcluster/pkg/util/random"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const DefaultPlatformSecretName = "vcluster-platform-api-key"

const CreatedByCLILabel = "vcluster.loft.sh/created-by-cli"

func ApplyPlatformSecret(
	ctx context.Context,
	config *config.CLI,
	kubeClient kubernetes.Interface,
	importName,
	name,
	namespace,
	project,
	accessKey,
	host string,
	insecure bool,
	certificateAuthorityData []byte,
	log log.Logger,
) error {
	// init platform client
	platformClient, err := InitClientFromConfig(ctx, config)
	if err != nil {
		return err
	}

	// set host
	if host == "" {
		host = strings.TrimPrefix(platformClient.Config().Platform.Host, "https://")
	}
	if !insecure {
		insecure = platformClient.Config().Platform.Insecure
	}
	if project == "" {
		project = "default"
	}

	// get access key
	if accessKey == "" {
		// check platform version
		platformVersion, err := platformClient.Version()
		if err != nil {
			return fmt.Errorf("get platform version: %w", err)
		}
		platformVersionSemVer, err := semver.Parse(strings.TrimPrefix(platformVersion.Version, "v"))
		if err != nil {
			return fmt.Errorf("parse platform version: %w", err)
		}

		// for platforms below version v4.0.0-beta.15 we need to do the old way
		if platformVersionSemVer.LTE(semver.MustParse("4.0.0-beta.15")) && platformVersionSemVer.GT(semver.MustParse("4.0.0-alpha.0")) {
			log.Debugf("Get legacy access key for vCluster, because platform is at version %s", platformVersion.Version)
			accessKey, err = getLegacyAccessKeyHost(ctx, platformClient)
			if err != nil {
				return fmt.Errorf("get legacy access key: %w", err)
			}
		} else {
			log.Debug("Get access key for vCluster")
			accessKey, importName, err = getAccessKey(ctx, kubeClient, platformClient, importName, name, namespace, project)
			if err != nil {
				return fmt.Errorf("get access key: %w", err)
			}
		}
	}

	// build secret payload
	secretPayload := map[string][]byte{
		"accessKey":                []byte(accessKey),
		"host":                     []byte(strings.TrimPrefix(host, "https://")),
		"project":                  []byte(project),
		"insecure":                 []byte(strconv.FormatBool(insecure)),
		"certificateAuthorityData": certificateAuthorityData,
	}
	if importName != "" {
		secretPayload["name"] = []byte(importName)
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
				Labels: map[string]string{
					CreatedByCLILabel: "true",
				},
			},
			Data: secretPayload,
		}, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("error creating platform secret %s/%s: %w", namespace, DefaultPlatformSecretName, err)
		}

		return nil
	} else if keySecret != nil && reflect.DeepEqual(keySecret.Data, secretPayload) {
		// no update needed, just return
		return nil
	}

	if keySecret == nil {
		return errors.New("nil keySecret")
	}

	// create the patch
	patch := ctrlclient.MergeFrom(keySecret.DeepCopy())
	keySecret.Data = secretPayload
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

func getAccessKey(ctx context.Context, kubeClient kubernetes.Interface, platformClient Client, importName, name, namespace, project string) (string, string, error) {
	// get management client
	managementClient, err := platformClient.Management()
	if err != nil {
		return "", "", fmt.Errorf("error getting management client: %w", err)
	}

	// get service and then search virtual cluster instance with service uid
	service, err := kubeClient.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return "", "", fmt.Errorf("could not get service %s/%s: %w", namespace, name, err)
	} else if err == nil {
		serviceUID := string(service.UID)

		// find existing vCluster
		virtualClusterList, err := managementClient.Loft().ManagementV1().VirtualClusterInstances(projectutil.ProjectNamespace(project)).List(ctx, metav1.ListOptions{})
		if err != nil {
			return "", "", fmt.Errorf("could not list virtual cluster instances in project %s: %w", project, err)
		}

		// try to find vCluster
		var virtualClusterInstance *managementv1.VirtualClusterInstance
		for _, vci := range virtualClusterList.Items {
			if vci.Status.ServiceUID == serviceUID {
				v := &vci
				virtualClusterInstance = v
				break
			}
		}

		// get access key for existing instance
		if virtualClusterInstance != nil {
			return returnAccessKeyFromInstance(ctx, managementClient, virtualClusterInstance)
		}
	}

	// we need to create a new instance here
	vName := importName
	if vName == "" {
		vName = name
	}

	// try with the regular name first
	created, accessKey, createdName, err := createWithName(ctx, managementClient, project, vName)
	if err != nil {
		return "", "", fmt.Errorf("error creating platform secret %s/%s: %w", namespace, DefaultPlatformSecretName, err)
	} else if created {
		return accessKey, createdName, nil
	} else if importName != "" {
		return "", "", fmt.Errorf("virtual cluster instance with name %s already exists", importName)
	}

	// try with random name
	vName += "-" + random.String(5)
	created, accessKey, createdName, err = createWithName(ctx, managementClient, project, vName)
	if err != nil {
		return "", "", fmt.Errorf("error creating platform secret %s/%s: %w", namespace, DefaultPlatformSecretName, err)
	} else if !created {
		return "", "", fmt.Errorf("couldn't create virtual cluster instance, name %s already exists", vName)
	}

	return accessKey, createdName, nil
}

func getLegacyAccessKeyHost(ctx context.Context, platformClient Client) (string, error) {
	// get management client
	managementClient, err := platformClient.Management()
	if err != nil {
		return "", fmt.Errorf("error getting management client: %w", err)
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
			return "", fmt.Errorf("create owned access key: %w", err)
		}

		platformConfig.VirtualClusterAccessKey = accessKey.Spec.Key
		platformClient.Config().Platform = platformConfig
		if err := platformClient.Save(); err != nil {
			return "", fmt.Errorf("save vCluster platform config: %w", err)
		}
	}

	return platformConfig.VirtualClusterAccessKey, nil
}

func createWithName(ctx context.Context, managementClient kube.Interface, project string, name string) (bool, string, string, error) {
	namespace := projectutil.ProjectNamespace(project)
	virtualClusterInstance, err := managementClient.Loft().ManagementV1().VirtualClusterInstances(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return false, "", "", fmt.Errorf("could not get virtual cluster instance %s/%s: %w", project, name, err)
	} else if err == nil {
		// instance has no service uid yet
		if virtualClusterInstance.Spec.External && virtualClusterInstance.Status.ServiceUID == "" {
			accessKey, createdName, err := returnAccessKeyFromInstance(ctx, managementClient, virtualClusterInstance)
			return true, accessKey, createdName, err
		}

		return false, "", "", nil
	}

	// create virtual cluster instance
	virtualClusterInstance, err = managementClient.Loft().ManagementV1().VirtualClusterInstances(namespace).Create(ctx, &managementv1.VirtualClusterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				CreatedByCLILabel: "true",
			},
		},
		Spec: managementv1.VirtualClusterInstanceSpec{
			VirtualClusterInstanceSpec: storagev1.VirtualClusterInstanceSpec{
				Template: &storagev1.VirtualClusterTemplateDefinition{
					VirtualClusterCommonSpec: storagev1.VirtualClusterCommonSpec{
						HelmRelease: storagev1.VirtualClusterHelmRelease{
							Chart: storagev1.VirtualClusterHelmChart{
								Version: upgrade.GetVersion(),
							},
						},
					},
				},
				External:    true,
				NetworkPeer: true,
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return false, "", "", fmt.Errorf("create virtual cluster instance: %w", err)
	}

	// try to retrieve access key
	accessKey, createdName, err := returnAccessKeyFromInstance(ctx, managementClient, virtualClusterInstance)
	if err != nil {
		_ = managementClient.Loft().ManagementV1().VirtualClusterInstances(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		return false, "", "", err
	}

	return true, accessKey, createdName, err
}

func returnAccessKeyFromInstance(ctx context.Context, managementClient kube.Interface, virtualClusterInstance *managementv1.VirtualClusterInstance) (string, string, error) {
	accessKey, err := managementClient.Loft().ManagementV1().VirtualClusterInstances(virtualClusterInstance.Namespace).GetAccessKey(ctx, virtualClusterInstance.Name, metav1.GetOptions{})
	if err != nil {
		return "", "", fmt.Errorf("get access key for virtual cluster instance %s/%s: %w", virtualClusterInstance.Namespace, virtualClusterInstance.Name, err)
	}

	return accessKey.AccessKey, virtualClusterInstance.Name, nil
}
