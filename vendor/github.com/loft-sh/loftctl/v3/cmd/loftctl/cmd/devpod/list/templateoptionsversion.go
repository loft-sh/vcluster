package list

import (
	"context"
	"fmt"
	"os"

	managementv1 "github.com/loft-sh/api/v3/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v3/pkg/apis/storage/v1"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/client"
	"github.com/loft-sh/loftctl/v3/pkg/version"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

// TemplateOptionsVersionCmd holds the cmd flags
type TemplateOptionsVersionCmd struct {
	*flags.GlobalFlags

	log log.Logger
}

// NewTemplateOptionsVersionCmd creates a new command
func NewTemplateOptionsVersionCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &TemplateOptionsVersionCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}
	c := &cobra.Command{
		Use:   "templateoptionsversion",
		Short: "Lists template options for a specific version for the DevPod provider",
		Long: `
#######################################################
############ loft devpod list templates ###############
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	return c
}

func (cmd *TemplateOptionsVersionCmd) Run(ctx context.Context) error {
	projectName := os.Getenv("LOFT_PROJECT")
	if projectName == "" {
		return fmt.Errorf("LOFT_PROJECT environment variable is empty")
	}
	templateName := os.Getenv("LOFT_TEMPLATE")
	if templateName == "" {
		return fmt.Errorf("LOFT_TEMPLATE environment variable is empty")
	}
	templateVersion := os.Getenv("LOFT_TEMPLATE_VERSION")
	if templateName == "" {
		return fmt.Errorf("LOFT_TEMPLATE_VERSION environment variable is empty")
	}

	baseClient, err := client.NewClientFromPath(cmd.Config)
	if err != nil {
		return err
	}

	managementClient, err := baseClient.Management()
	if err != nil {
		return err
	}

	// check template
	template, err := FindTemplate(ctx, managementClient, projectName, templateName)
	if err != nil {
		return err
	}

	// get parameters
	parameters, err := GetTemplateParameters(template, templateVersion)
	if err != nil {
		return err
	}

	// print to stdout
	return printOptions(&OptionsFormat{Options: parametersToOptions(parameters)})
}

func GetTemplateParameters(template *managementv1.DevPodWorkspaceTemplate, templateVersion string) ([]storagev1.AppParameter, error) {
	if templateVersion == "latest" {
		templateVersion = ""
	}

	if templateVersion == "" {
		latestVersion := version.GetLatestVersion(template)
		if latestVersion == nil {
			return nil, fmt.Errorf("couldn't find any version in template")
		}

		return latestVersion.(*storagev1.DevPodWorkspaceTemplateVersion).Parameters, nil
	}

	_, latestMatched, err := version.GetLatestMatchedVersion(template, templateVersion)
	if err != nil {
		return nil, err
	} else if latestMatched == nil {
		return nil, fmt.Errorf("couldn't find any matching version to %s", templateVersion)
	}

	return latestMatched.(*storagev1.DevPodWorkspaceTemplateVersion).Parameters, nil
}
