package list

import (
	"context"
	"strings"
	"time"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/table"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/platform/kube"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
)

// SharedSecretsCmd holds the cmd flags
type SharedSecretsCmd struct {
	*flags.GlobalFlags
	Namespace   string
	Project     []string
	All         bool
	AllProjects bool

	log log.Logger
}

// newSharedSecretsCmd creates a new command
func newSharedSecretsCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &SharedSecretsCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}
	description := product.ReplaceWithHeader("list secrets", `
List the shared secrets you have access to

Example:
vcluster platform list secrets
########################################################
	`)
	c := &cobra.Command{
		Use:   "secrets",
		Short: "Lists all the shared secrets you have access to",
		Long:  description,
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			return cmd.Run(cobraCmd)
		},
	}

	c.Flags().StringArrayVarP(&cmd.Project, "project", "p", []string{}, "The project(s) to read project secrets from. If omitted will list global secrets")
	c.Flags().StringVarP(&cmd.Namespace, "namespace", "n", "", product.Replace("The namespace in the vCluster cluster to read global secrets from. If omitted will query all accessible global secrets"))
	c.Flags().BoolVarP(&cmd.All, "all", "a", false, "Display global and project secrets. May be used with the --project flag to display global secrets and a subset of project secrets")
	c.Flags().BoolVar(&cmd.AllProjects, "all-projects", false, "Display project secrets for all projects.")
	return c
}

// Run executes the functionality
func (cmd *SharedSecretsCmd) Run(command *cobra.Command) error {
	platformClient, err := platform.InitClientFromConfig(command.Context(), cmd.LoadedConfig(cmd.log))
	if err != nil {
		return err
	}

	managementClient, err := platformClient.Management()
	if err != nil {
		return err
	}

	if cmd.All {
		sharedSecretList, err := managementClient.Loft().ManagementV1().SharedSecrets(cmd.Namespace).List(command.Context(), metav1.ListOptions{})
		if err != nil {
			return err
		}

		var sharedSecrets []*managementv1.SharedSecret
		for idx := range sharedSecretList.Items {
			sharedSecretItem := sharedSecretList.Items[idx]
			sharedSecrets = append(sharedSecrets, &sharedSecretItem)
		}

		projectSecrets, err := platform.GetProjectSecrets(command.Context(), managementClient, cmd.Project...)
		if err != nil {
			return err
		}

		return cmd.printAllSecrets(sharedSecrets, projectSecrets)
	} else if cmd.AllProjects {
		projectSecrets, err := platform.GetProjectSecrets(command.Context(), managementClient)
		if err != nil {
			return err
		}

		return cmd.printProjectSecrets(projectSecrets)
	}
	if len(cmd.Project) == 0 {
		return cmd.printSharedSecrets(command.Context(), managementClient, cmd.Namespace)
	}
	projectSecrets, err := platform.GetProjectSecrets(command.Context(), managementClient, cmd.Project...)
	if err != nil {
		return err
	}

	return cmd.printProjectSecrets(projectSecrets)
}

func (cmd *SharedSecretsCmd) printSharedSecrets(ctx context.Context, managementClient kube.Interface, namespace string) error {
	secrets, err := managementClient.Loft().ManagementV1().SharedSecrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	header := []string{
		"Name",
		"Namespace",
		"Keys",
		"Age",
	}
	var values [][]string
	for _, secret := range secrets.Items {
		var keyNames []string
		for k := range secret.Spec.Data {
			keyNames = append(keyNames, k)
		}

		values = append(values, []string{
			secret.Name,
			secret.Namespace,
			strings.Join(keyNames, ","),
			duration.HumanDuration(time.Since(secret.CreationTimestamp.Time)),
		})
	}

	table.PrintTable(cmd.log, header, values)
	return nil
}

func (cmd *SharedSecretsCmd) printProjectSecrets(projectSecrets []*platform.ProjectProjectSecret) error {
	header := []string{
		"Name",
		"Namespace",
		"Project",
		"Keys",
		"Age",
	}
	var values [][]string
	for _, secret := range projectSecrets {
		projectSecret := secret.ProjectSecret
		var keyNames []string

		for k := range projectSecret.Spec.Data {
			keyNames = append(keyNames, k)
		}

		values = append(values, []string{
			projectSecret.Name,
			projectSecret.Namespace,
			secret.Project,
			strings.Join(keyNames, ","),
			duration.HumanDuration(time.Since(projectSecret.CreationTimestamp.Time)),
		})
	}

	table.PrintTable(cmd.log, header, values)
	return nil
}

func (cmd *SharedSecretsCmd) printAllSecrets(
	sharedSecrets []*managementv1.SharedSecret,
	projectSecrets []*platform.ProjectProjectSecret,
) error {
	header := []string{
		"Name",
		"Namespace",
		"Project",
		"Keys",
		"Age",
	}

	var values [][]string
	for _, secret := range sharedSecrets {
		var keyNames []string
		for k := range secret.Spec.Data {
			keyNames = append(keyNames, k)
		}

		values = append(values, []string{
			secret.Name,
			secret.Namespace,
			"",
			strings.Join(keyNames, ","),
			duration.HumanDuration(time.Since(secret.CreationTimestamp.Time)),
		})
	}

	for _, secret := range projectSecrets {
		projectSecret := secret.ProjectSecret
		var keyNames []string
		for k := range projectSecret.Spec.Data {
			keyNames = append(keyNames, k)
		}

		values = append(values, []string{
			projectSecret.Name,
			projectSecret.Namespace,
			secret.Project,
			strings.Join(keyNames, ","),
			duration.HumanDuration(time.Since(projectSecret.CreationTimestamp.Time)),
		})
	}

	table.PrintTable(cmd.log, header, values)
	return nil
}
