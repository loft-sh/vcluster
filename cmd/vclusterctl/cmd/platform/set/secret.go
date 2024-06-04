package set

import (
	"context"
	"fmt"
	"strings"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/survey"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/platform"
	pdefaults "github.com/loft-sh/vcluster/pkg/platform/defaults"
	"github.com/loft-sh/vcluster/pkg/platform/kube"
	"github.com/loft-sh/vcluster/pkg/projectutil"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	ErrNoSecret = errors.New("please specify a secret key to set")
)

type SecretType string

const (
	ProjectSecret SecretType = "Project Secret"
	SharedSecret  SecretType = "Shared Secret"
)

// SecretCmd holds the flags
type SecretCmd struct {
	*flags.GlobalFlags
	Namespace string
	Project   string
	log       log.Logger
}

// NewSecretCmd creates a new command
func NewSecretCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	cmd := &SecretCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}
	description := product.ReplaceWithHeader("set secret", `
Sets the key value of a project / shared secret.


Example:
vcluster platform set secret test-secret.key value
vcluster platform set secret test-secret.key value --project myproject
#######################################################
	`)
	useLine, validator := util.NamedPositionalArgsValidator(true, true, "SECRET_NAME", "SECRET_VALUE")
	c := &cobra.Command{
		Use:   "secret" + useLine,
		Short: "Sets the key value of a project / shared secret",
		Long:  description,
		Args:  validator,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd, args)
		},
	}

	p, _ := defaults.Get(pdefaults.KeyProject, "")
	c.Flags().StringVarP(&cmd.Namespace, "namespace", "n", "", product.Replace("The namespace in the vCluster platform cluster to create the secret in. If omitted will use the namespace were vCluster platform is installed in"))
	c.Flags().StringVarP(&cmd.Project, "project", "p", p, "The project to create the project secret in.")

	return c
}
func (cmd *SecretCmd) Run(cobraCmd *cobra.Command, args []string) error {
	platformClient, err := platform.InitClientFromConfig(cobraCmd.Context(), cmd.LoadedConfig(cmd.log))
	if err != nil {
		return err
	}

	managementClient, err := platformClient.Management()
	if err != nil {
		return err
	}

	// get secret
	secretName := ""
	keyName := ""
	secretArg := args[0]
	idx := strings.Index(secretArg, ".")
	if idx == -1 {
		secretName = secretArg
	} else {
		secretName = secretArg[:idx]
		keyName = secretArg[idx+1:]
	}

	var secretType SecretType

	if cmd.Project != "" {
		secretType = ProjectSecret
	} else {
		secretType = SharedSecret
	}

	ctx := cobraCmd.Context()

	switch secretType {
	case ProjectSecret:
		// get target namespace
		namespace := projectutil.ProjectNamespace(cmd.Project)
		return cmd.setProjectSecret(ctx, managementClient, args, namespace, secretName, keyName)
	case SharedSecret:
		namespace, err := GetSharedSecretNamespace(cmd.Namespace)
		if err != nil {
			return errors.Wrap(err, "get shared secrets namespace")
		}

		return cmd.setSharedSecret(ctx, managementClient, args, namespace, secretName, keyName)
	}

	return nil
}

func (cmd *SecretCmd) setProjectSecret(ctx context.Context, managementClient kube.Interface, args []string, namespace, secretName, keyName string) error {
	secret, err := managementClient.Loft().ManagementV1().ProjectSecrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return errors.Wrap(err, "get secret")
		}

		secret = nil
	}

	if keyName == "" {
		if secret == nil {
			return fmt.Errorf(product.Replace("%w: for example 'loft set secret my-secret.key value'"), ErrNoSecret)
		}
		if len(secret.Spec.Data) == 0 {
			return errors.Errorf(product.Replace("secret %s has no keys. Please specify a key like `loft set secret name.key value`"), secretName)
		}

		keyNames := []string{}
		for k := range secret.Spec.Data {
			keyNames = append(keyNames, k)
		}

		keyName, err = cmd.log.Question(&survey.QuestionOptions{
			Question:     "Please select a secret key to set",
			DefaultValue: keyNames[0],
			Options:      keyNames,
		})
		if err != nil {
			return errors.Wrap(err, "ask question")
		}
	}

	// create the secret
	if secret == nil {
		_, err = managementClient.Loft().ManagementV1().ProjectSecrets(namespace).Create(ctx, &managementv1.ProjectSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name: secretName,
			},
			Spec: managementv1.ProjectSecretSpec{
				Data: map[string][]byte{
					keyName: []byte(args[1]),
				},
			},
		}, metav1.CreateOptions{})
		if err != nil {
			return err
		}

		cmd.log.Donef("Successfully created secret %s with key %s", secretName, keyName)
		return nil
	}

	// Update the secret
	if secret.Spec.Data == nil {
		secret.Spec.Data = map[string][]byte{}
	}
	secret.Spec.Data[keyName] = []byte(args[1])
	_, err = managementClient.Loft().ManagementV1().ProjectSecrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "update secret")
	}

	cmd.log.Donef("Successfully set secret key %s.%s", secretName, keyName)
	return nil
}

func (cmd *SecretCmd) setSharedSecret(ctx context.Context, managementClient kube.Interface, args []string, namespace, secretName, keyName string) error {
	secret, err := managementClient.Loft().ManagementV1().SharedSecrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return errors.Wrap(err, "get secret")
		}

		secret = nil
	}

	if keyName == "" {
		if secret == nil {
			return fmt.Errorf(product.Replace("%w: for example 'loft set secret my-secret.key value'"), ErrNoSecret)
		}
		if len(secret.Spec.Data) == 0 {
			return errors.Errorf(product.Replace("secret %s has no keys. Please specify a key like `loft set secret name.key value`"), secretName)
		}

		keyNames := []string{}
		for k := range secret.Spec.Data {
			keyNames = append(keyNames, k)
		}

		keyName, err = cmd.log.Question(&survey.QuestionOptions{
			Question:     "Please select a secret key to set",
			DefaultValue: keyNames[0],
			Options:      keyNames,
		})
		if err != nil {
			return errors.Wrap(err, "ask question")
		}
	}

	// create the secret
	if secret == nil {
		_, err = managementClient.Loft().ManagementV1().SharedSecrets(namespace).Create(ctx, &managementv1.SharedSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name: secretName,
			},
			Spec: managementv1.SharedSecretSpec{
				SharedSecretSpec: storagev1.SharedSecretSpec{
					Data: map[string][]byte{
						keyName: []byte(args[1]),
					},
				},
			},
		}, metav1.CreateOptions{})
		if err != nil {
			return err
		}

		cmd.log.Donef("Successfully created secret %s with key %s", secretName, keyName)
		return nil
	}

	// Update the secret
	if secret.Spec.Data == nil {
		secret.Spec.Data = map[string][]byte{}
	}
	secret.Spec.Data[keyName] = []byte(args[1])
	_, err = managementClient.Loft().ManagementV1().SharedSecrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "update secret")
	}

	cmd.log.Donef("Successfully set secret key %s.%s", secretName, keyName)
	return nil
}

func GetSharedSecretNamespace(namespace string) (string, error) {
	if namespace == "" {
		namespace = "loft"
	}

	return namespace, nil
}
