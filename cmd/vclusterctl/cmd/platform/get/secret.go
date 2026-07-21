package get

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"os"
	"strings"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/survey"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/platform/set"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/platform/clihelper"
	pdefaults "github.com/loft-sh/vcluster/pkg/platform/defaults"
	"github.com/loft-sh/vcluster/pkg/platform/kube"
	"github.com/loft-sh/vcluster/pkg/projectutil"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	OutputYAML  string = "yaml"
	OutputJSON  string = "json"
	OutputValue string = "value"
)

// SecretCmd holds the flags
type SecretCmd struct {
	*flags.GlobalFlags

	log       log.Logger
	Namespace string
	Project   string
	Output    string
	All       bool
}

// newSecretCmd creates a new command
func newSecretCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	cmd := &SecretCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}
	description := product.ReplaceWithHeader("get secret", `
Returns the key value of a project / shared secret.

Example:
vcluster platform get secret test-secret.key
vcluster platform get secret test-secret.key --project myproject

For shared secret, if the --namespace flag is not set, it looks for the secrets in vcluster-platform and loft namespaces.
########################################################
	`)

	useLine, validator := util.NamedPositionalArgsValidator(true, true, "SECRET_NAME")
	c := &cobra.Command{
		Use:   "secret" + useLine,
		Short: "Returns the key value of a project / shared secret",
		Long:  description,
		Args: func(cmd *cobra.Command, args []string) error {
			// bypass validation if the "all" flag is set
			all, err := cmd.Flags().GetBool("all")
			if err != nil {
				return err
			}
			if !all {
				return validator(cmd, args)
			}
			return nil
		},
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	p, _ := defaults.Get(pdefaults.KeyProject, "")
	c.Flags().StringVarP(&cmd.Project, "project", "p", p, "The project to read the project secret from.")
	c.Flags().StringVarP(&cmd.Namespace, "namespace", "n", "", product.Replace("The namespace in the vCluster platform cluster to read the secret from. If omitted will use the namespace where vCluster platform is installed in"))
	c.Flags().BoolVarP(&cmd.All, "all", "a", false, "Display all secret keys")
	c.Flags().StringVarP(&cmd.Output, "output", "o", "", "Output format. One of: (json, yaml, value). If the --all flag is passed 'yaml' will be the default format")
	return c
}

// RunUsers executes the functionality
func (cmd *SecretCmd) Run(ctx context.Context, args []string) error {
	platformClient, err := platform.InitClientFromConfig(ctx, cmd.LoadedConfig(cmd.log))
	if err != nil {
		return err
	}

	managementClient, err := platformClient.Management()
	if err != nil {
		return err
	}

	var secretType set.SecretType

	if cmd.Project != "" {
		secretType = set.ProjectSecret
	} else {
		secretType = set.SharedSecret
	}

	output := cmd.Output

	if cmd.All && output == "" {
		output = OutputYAML
	} else if output == "" {
		output = OutputValue
	}

	if cmd.All && output == OutputValue {
		return errors.Errorf("output format %s is not allowed with the --all flag", OutputValue)
	}

	// get target namespace
	var namespace string

	switch secretType {
	case set.ProjectSecret:
		namespace = projectutil.ProjectNamespace(cmd.Project)
	case set.SharedSecret:
		namespace = cmd.Namespace
	}

	// get secret
	secretName := ""
	keyName := ""
	if len(args) == 1 {
		secret := args[0]
		idx := strings.Index(secret, ".")
		if idx == -1 {
			secretName = secret
		} else {
			secretName = secret[:idx]
			keyName = secret[idx+1:]
		}
	} else {
		secretNameList := []string{}

		switch secretType {
		case set.ProjectSecret:
			secrets, err := managementClient.Loft().ManagementV1().ProjectSecrets(namespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				return errors.Wrap(err, "list project secrets")
			}

			for _, s := range secrets.Items {
				secretNameList = append(secretNameList, s.Name)
			}
		case set.SharedSecret:
			secrets, err := cmd.listSharedSecretsWithFallback(ctx, namespace, managementClient)
			if err != nil {
				return errors.Wrap(err, "list shared secrets")
			}

			for _, s := range secrets.Items {
				secretNameList = append(secretNameList, s.Name)
			}
		}

		if len(secretNameList) == 0 {
			return fmt.Errorf("couldn't find any secrets that could be read. Please make sure to create a shared secret before you try to read it")
		}

		secretName, err = cmd.log.Question(&survey.QuestionOptions{
			Question:     "Please select a secret to read from",
			DefaultValue: secretNameList[0],
			Options:      secretNameList,
		})
		if err != nil {
			return errors.Wrap(err, "ask question")
		}
	}

	if cmd.All && keyName != "" {
		cmd.log.Warnf("secret key %s ignored because --all was passed", keyName)
	}

	var secretData map[string][]byte

	switch secretType {
	case set.ProjectSecret:
		pSecret, err := managementClient.Loft().ManagementV1().ProjectSecrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
		if err != nil {
			return errors.Wrap(err, "get secrets")
		} else if len(pSecret.Spec.Data) == 0 {
			return errors.Errorf("secret %s has no keys to read. Please set a key before trying to read it", secretName)
		}

		secretData = pSecret.Spec.Data
	case set.SharedSecret:
		sSecret, err := cmd.findSharedSecretWithFallback(ctx, secretName, namespace, managementClient)
		if err != nil {
			return errors.Wrap(err, "get secrets")
		} else if len(sSecret.Spec.Data) == 0 {
			return errors.Errorf("secret %s has no keys to read. Please set a key before trying to read it", secretName)
		}

		secretData = sSecret.Spec.Data
	}

	kvs := secretData
	if !cmd.All {
		if keyName == "" {
			var keyNames []string

			for k := range secretData {
				keyNames = append(keyNames, k)
			}

			defaultValue := ""
			if len(keyNames) > 0 {
				defaultValue = keyNames[0]
			}

			keyName, err = cmd.log.Question(&survey.QuestionOptions{
				Question:     "Please select a secret key to read",
				DefaultValue: defaultValue,
				Options:      keyNames,
			})
			if err != nil {
				return errors.Wrap(err, "ask question")
			}
		}

		keyValue, ok := secretData[keyName]
		if ok {
			kvs = map[string][]byte{
				keyName: keyValue,
			}
		}
	}

	var outputData []byte
	if output == OutputValue {
		var ok bool
		outputData, ok = kvs[keyName]
		if !ok {
			return errors.Errorf("key %s does not exist in secret %s", keyName, secretName)
		}
	} else {
		stringValues := map[string]string{}
		for k, v := range kvs {
			stringValues[k] = string(v)
		}

		if output == OutputYAML {
			outputData, err = yaml.Marshal(stringValues)
			if err != nil {
				return err
			}
		} else if output == OutputJSON {
			outputData, err = json.Marshal(stringValues)
			if err != nil {
				return err
			}
		} else {
			return errors.Errorf("unknown output format %s, allowed formats are: value, json, yaml", output)
		}
	}

	_, err = os.Stdout.Write(outputData)
	return err
}

func (cmd *SecretCmd) findSharedSecretWithFallback(ctx context.Context, secretName, namespace string, managementClient kube.Interface) (*managementv1.SharedSecret, error) {
	// look in:
	// 1. namespace defined by the flag
	// 2. vcluster-platform namespace
	// 3. loft namespace
	// if secret cannot be found in these, fail
	var sSecret *managementv1.SharedSecret
	var err error
	var errs []error
	for _, ns := range []string{namespace, clihelper.DefaultPlatformNamespace, clihelper.LegacyPlatformNamespace} {
		if ns == "" {
			// namespace from the command flag may be empty
			continue
		}
		cmd.log.Debugf("looking for secret %s in namespace %s", secretName, ns)
		sSecret, err = managementClient.Loft().ManagementV1().SharedSecrets(ns).Get(ctx, secretName, metav1.GetOptions{})
		if err == nil {
			cmd.log.Debugf("found secret %s in namespace %s", secretName, ns)
			return sSecret, nil
		}
		errs = append(errs, fmt.Errorf("looking for secret %s in namespace %s failed: %w", secretName, ns, err))
	}
	return nil, stderrors.Join(errs...)
}

func (cmd *SecretCmd) listSharedSecretsWithFallback(ctx context.Context, namespace string, managementClient kube.Interface) (*managementv1.SharedSecretList, error) {
	var secrets *managementv1.SharedSecretList
	var err error
	var errs []error
	for _, ns := range []string{namespace, clihelper.DefaultPlatformNamespace, clihelper.LegacyPlatformNamespace} {
		if ns == "" {
			// namespace from the command flag may be empty
			continue
		}
		cmd.log.Debugf("looking for shared secrets in namespace %s", ns)
		secrets, err = managementClient.Loft().ManagementV1().SharedSecrets(ns).List(ctx, metav1.ListOptions{})
		if err == nil && len(secrets.Items) > 0 {
			return secrets, nil
		} else if err == nil && len(secrets.Items) == 0 {
			// if no secrets are found, keep looking in the next namespace
			cmd.log.Debugf("no shared secrets found in namespace %s", ns)
			continue
		}
		errs = append(errs, fmt.Errorf("listing shared secrets in namespace %s failed: %w", ns, err))
	}
	return nil, stderrors.Join(errs...)
}
