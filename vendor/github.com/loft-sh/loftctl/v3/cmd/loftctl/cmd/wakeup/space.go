package wakeup

import (
	"context"
	"fmt"
	"time"

	managementv1 "github.com/loft-sh/api/v3/pkg/apis/management/v1"
	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/pkg/config"
	"github.com/loft-sh/loftctl/v3/pkg/space"
	"github.com/loft-sh/loftctl/v3/pkg/util"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/client"
	"github.com/loft-sh/loftctl/v3/pkg/client/helper"
	"github.com/loft-sh/loftctl/v3/pkg/client/naming"
	pdefaults "github.com/loft-sh/loftctl/v3/pkg/defaults"
	"github.com/loft-sh/loftctl/v3/pkg/upgrade"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SpaceCmd holds the cmd flags
type SpaceCmd struct {
	*flags.GlobalFlags

	Project string
	Cluster string
	Log     log.Logger
}

// NewSpaceCmd creates a new command
func NewSpaceCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	cmd := &SpaceCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	description := product.ReplaceWithHeader("wakeup space", `
wakeup resumes a sleeping space

Example:
loft wakeup space myspace
loft wakeup space myspace --project myproject
#######################################################
	`)
	if upgrade.IsPlugin == "true" {
		description = `
#######################################################
################ devspace wakeup space ################
#######################################################
wakeup resumes a sleeping space

Example:
devspace wakeup space myspace
devspace wakeup space myspace --project myproject
#######################################################
	`
	}

	c := &cobra.Command{
		Use:   "space" + util.SpaceNameOnlyUseLine,
		Short: "Wakes up a space",
		Long:  description,
		Args:  util.SpaceNameOnlyValidator,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	p, _ := defaults.Get(pdefaults.KeyProject, "")
	c.Flags().StringVarP(&cmd.Project, "project", "p", p, "The project to use")
	c.Flags().StringVar(&cmd.Cluster, "cluster", "", "The cluster to use")
	return c
}

// Run executes the functionality
func (cmd *SpaceCmd) Run(ctx context.Context, args []string) error {
	baseClient, err := client.NewClientFromPath(cmd.Config)
	if err != nil {
		return err
	}

	spaceName := ""
	if len(args) > 0 {
		spaceName = args[0]
	}

	cmd.Cluster, cmd.Project, spaceName, err = helper.SelectSpaceInstanceOrSpace(baseClient, spaceName, cmd.Project, cmd.Cluster, cmd.Log)
	if err != nil {
		return err
	}

	if cmd.Project == "" {
		return cmd.legacySpaceWakeUp(ctx, baseClient, spaceName)
	}

	return cmd.spaceWakeUp(ctx, baseClient, spaceName)
}

func (cmd *SpaceCmd) spaceWakeUp(ctx context.Context, baseClient client.Client, spaceName string) error {
	managementClient, err := baseClient.Management()
	if err != nil {
		return err
	}

	_, err = space.WaitForSpaceInstance(ctx, managementClient, naming.ProjectNamespace(cmd.Project), spaceName, true, cmd.Log)
	if err != nil {
		return err
	}

	return nil
}

func (cmd *SpaceCmd) legacySpaceWakeUp(ctx context.Context, baseClient client.Client, spaceName string) error {
	clusterClient, err := baseClient.Cluster(cmd.Cluster)
	if err != nil {
		return err
	}

	managementClient, err := baseClient.Management()
	if err != nil {
		return err
	}

	// get current user / team
	self, err := managementClient.Loft().ManagementV1().Selves().Create(ctx, &managementv1.Self{}, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "get self")
	} else if self.Status.User == nil && self.Status.Team == nil {
		return fmt.Errorf("no user or team name returned")
	}

	configs, err := clusterClient.Agent().ClusterV1().SleepModeConfigs(spaceName).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	sleepModeConfig := &configs.Items[0]
	sleepModeConfig.Spec.ForceSleep = false
	sleepModeConfig.Spec.ForceSleepDuration = nil
	sleepModeConfig.Status.LastActivity = time.Now().Unix()

	_, err = clusterClient.Agent().ClusterV1().SleepModeConfigs(spaceName).Create(ctx, sleepModeConfig, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// wait for sleeping
	cmd.Log.Info("Wait until space wakes up...")
	err = wait.PollUntilContextTimeout(ctx, time.Second, config.Timeout(), false, func(ctx context.Context) (done bool, err error) {
		configs, err := clusterClient.Agent().ClusterV1().SleepModeConfigs(spaceName).List(ctx, metav1.ListOptions{})
		if err != nil {
			return false, err
		}

		return configs.Items[0].Status.SleepingSince == 0, nil
	})
	if err != nil {
		return fmt.Errorf("error waiting for space to wake up: %w", err)
	}

	cmd.Log.Donef("Successfully woken up space %s", spaceName)
	return nil
}
