package generate

import (
	"context"
	"time"

	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/config"
	"github.com/loft-sh/loftctl/v3/pkg/kubeconfig"
	"github.com/loft-sh/loftctl/v3/pkg/upgrade"
	"github.com/loft-sh/log"
	perrors "github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// AdminKubeConfigCmd holds the cmd flags
type AdminKubeConfigCmd struct {
	*flags.GlobalFlags
	log            log.Logger
	Namespace      string
	ServiceAccount string
}

// NewAdminKubeConfigCmd creates a new command
func NewAdminKubeConfigCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &AdminKubeConfigCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}
	description := product.ReplaceWithHeader("generate admin-kube-config", `
Creates a new kube config that can be used to connect
a cluster to loft.

Example:
loft generate admin-kube-config
loft generate admin-kube-config --namespace mynamespace
########################################################
	`)
	if upgrade.IsPlugin == "true" {
		description = `
#######################################################
######## devspace generate admin-kube-config ##########
#######################################################
Creates a new kube config that can be used to connect
a cluster to loft.

Example:
devspace generate admin-kube-config
devspace generate admin-kube-config --namespace mynamespace
#######################################################
	`
	}
	c := &cobra.Command{
		Use:   "admin-kube-config",
		Short: "Generates a new kube config for connecting a cluster",
		Long:  description,
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})
			c, err := loader.ClientConfig()
			if err != nil {
				return err
			}

			return cmd.Run(cobraCmd.Context(), c, cobraCmd, args)
		},
	}

	c.Flags().StringVar(&cmd.Namespace, "namespace", "loft", "The namespace to generate the service account in. The namespace will be created if it does not exist")
	c.Flags().StringVar(&cmd.ServiceAccount, "service-account", "loft-admin", "The service account name to create")
	return c
}

// Run executes the command
func (cmd *AdminKubeConfigCmd) Run(ctx context.Context, c *rest.Config, cobraCmd *cobra.Command, args []string) error {
	token, err := GetAuthToken(ctx, c, cmd.Namespace, cmd.ServiceAccount)
	if err != nil {
		return perrors.Wrap(err, "get auth token")
	}

	// print kube config
	return kubeconfig.PrintTokenKubeConfig(c, string(token))
}

func GetAuthToken(ctx context.Context, c *rest.Config, namespace, serviceAccount string) ([]byte, error) {
	client, err := kubernetes.NewForConfig(c)
	if err != nil {
		return []byte{}, perrors.Wrap(err, "create kube client")
	}

	// make sure namespace exists
	_, err = client.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		if !kerrors.IsAlreadyExists(err) {
			return []byte{}, err
		}
	}

	// create service account
	_, err = client.CoreV1().ServiceAccounts(namespace).Create(ctx, &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: serviceAccount,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		if !kerrors.IsAlreadyExists(err) {
			return []byte{}, err
		}
	}

	// create clusterrolebinding
	_, err = client.RbacV1().ClusterRoleBindings().Create(ctx, &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: serviceAccount + "-binding",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccount,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
	}, metav1.CreateOptions{})
	if err != nil {
		if !kerrors.IsAlreadyExists(err) {
			return []byte{}, err
		}
	}

	// manually create token secret. This approach works for all kubernetes versions
	tokenSecretName := serviceAccount + "-token"
	_, err = client.CoreV1().Secrets(namespace).Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tokenSecretName,
			Namespace: namespace,
			Annotations: map[string]string{
				corev1.ServiceAccountNameKey: serviceAccount,
			},
		},
		Type: corev1.SecretTypeServiceAccountToken,
	}, metav1.CreateOptions{})
	if err != nil {
		if !kerrors.IsAlreadyExists(err) {
			return []byte{}, err
		}
	}

	// wait for secret token to be populated
	token := []byte{}
	err = wait.PollUntilContextTimeout(ctx, 250*time.Millisecond, config.Timeout(), false, func(ctx context.Context) (done bool, err error) {
		secret, err := client.CoreV1().Secrets(namespace).Get(ctx, tokenSecretName, metav1.GetOptions{})
		if err != nil {
			return false, perrors.Wrap(err, "get service account secret")
		}

		ok := false
		token, ok = secret.Data["token"]
		if !ok {
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		return []byte{}, err
	}

	return token, nil
}
