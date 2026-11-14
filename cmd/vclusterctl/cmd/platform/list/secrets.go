package list

import (
	"errors"
	"strings"
	"time"

	"github.com/loft-sh/api/v5/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/table"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/platform/set"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/duration"
)

// SecretsCmd holds the cmd flags
type SecretsCmd struct {
	*flags.GlobalFlags
	Project     []string
	All         bool
	AllProjects bool

	log log.Logger
}

// newSecretsCmd creates a new command
func newSecretsCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &SecretsCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}
	description := product.ReplaceWithHeader("list secrets", `
List the project secrets you have access to

Example:
vcluster platform list secrets --project myproject
vcluster platform list secrets --all-projects
########################################################
	`)
	c := &cobra.Command{
		Use:   "secrets",
		Short: "Lists the project secrets you have access to",
		Long:  description,
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			return cmd.Run(cobraCmd)
		},
	}

	c.Flags().StringArrayVarP(&cmd.Project, "project", "p", []string{}, "The project(s) to read project secrets from")
	c.Flags().BoolVarP(&cmd.All, "all", "a", false, "Display project secrets. May be used with the --project flag to display a subset of project secrets")
	c.Flags().BoolVar(&cmd.AllProjects, "all-projects", false, "Display project secrets for all projects.")
	return c
}

// ErrSharedSecretsRemoved is returned when no project flag is given: that used to list shared
// secrets, which were removed in vCluster CLI v0.38 (vCluster Platform 5.0).
var ErrSharedSecretsRemoved = errors.New("support for shared secrets was removed in vCluster CLI v0.38. Use --project or --all-projects to list project secrets")

// allWithoutSharedSecretsMsg is logged for --all, which used to include shared secrets.
const allWithoutSharedSecretsMsg = "--all no longer includes shared secrets: support for shared secrets was removed in vCluster CLI v0.38"

// Run executes the functionality
func (cmd *SecretsCmd) Run(command *cobra.Command) error {
	// Without a project flag this command used to list shared secrets. Fail instead of
	// succeeding with an empty list, which a script would take as a real answer.
	if !cmd.All && !cmd.AllProjects && len(cmd.Project) == 0 {
		return ErrSharedSecretsRemoved
	}
	if cmd.All {
		cmd.log.Warn(allWithoutSharedSecretsMsg)
	}
	set.WarnIgnoredNamespace(command, cmd.log)

	platformClient, err := platform.InitClientFromConfig(command.Context(), cmd.LoadedConfig(cmd.log))
	if err != nil {
		return err
	}

	managementClient, err := platformClient.Management()
	if err != nil {
		return err
	}

	// --all-projects lists every project and, as before, ignores --project
	projects := cmd.Project
	if cmd.AllProjects {
		projects = nil
	}
	projectSecrets, err := platform.GetProjectSecrets(command.Context(), managementClient, projects...)
	if err != nil {
		return err
	}

	return cmd.printProjectSecrets(projectSecrets)
}

func (cmd *SecretsCmd) printProjectSecrets(projectSecrets []*platform.ProjectProjectSecret) error {
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
