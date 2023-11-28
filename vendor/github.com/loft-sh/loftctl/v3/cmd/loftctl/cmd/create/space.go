package create

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/loft-sh/loftctl/v3/pkg/client/naming"
	"github.com/loft-sh/loftctl/v3/pkg/config"
	"github.com/loft-sh/loftctl/v3/pkg/space"
	"github.com/loft-sh/loftctl/v3/pkg/util"
	"k8s.io/apimachinery/pkg/util/wait"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"

	clusterv1 "github.com/loft-sh/agentapi/v3/pkg/apis/loft/cluster/v1"
	agentstoragev1 "github.com/loft-sh/agentapi/v3/pkg/apis/loft/storage/v1"
	storagev1 "github.com/loft-sh/api/v3/pkg/apis/storage/v1"
	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/use"
	"github.com/loft-sh/loftctl/v3/pkg/clihelper"
	"github.com/loft-sh/loftctl/v3/pkg/constants"
	"github.com/loft-sh/loftctl/v3/pkg/kube"
	"github.com/loft-sh/loftctl/v3/pkg/parameters"
	"github.com/loft-sh/loftctl/v3/pkg/task"
	"github.com/loft-sh/loftctl/v3/pkg/version"
	kerrors "k8s.io/apimachinery/pkg/api/errors"

	managementv1 "github.com/loft-sh/api/v3/pkg/apis/management/v1"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/client"
	"github.com/loft-sh/loftctl/v3/pkg/client/helper"
	pdefaults "github.com/loft-sh/loftctl/v3/pkg/defaults"
	"github.com/loft-sh/loftctl/v3/pkg/kubeconfig"
	"github.com/loft-sh/loftctl/v3/pkg/upgrade"
	"github.com/loft-sh/log"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SpaceCmd holds the cmd flags
type SpaceCmd struct {
	*flags.GlobalFlags

	SleepAfter                   int64
	DeleteAfter                  int64
	Cluster                      string
	Project                      string
	CreateContext                bool
	SwitchContext                bool
	DisableDirectClusterEndpoint bool
	Template                     string
	Version                      string
	Set                          []string
	ParametersFile               string
	SkipWait                     bool

	UseExisting bool
	Recreate    bool
	Update      bool

	DisplayName string
	Description string
	Links       []string
	Annotations []string
	Labels      []string

	User string
	Team string

	Log log.Logger
}

// NewSpaceCmd creates a new command
func NewSpaceCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	cmd := &SpaceCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}
	description := product.ReplaceWithHeader("create space", `
Creates a new space for the given project, if
it does not yet exist.

Example:
loft create space myspace
loft create space myspace --project myproject
loft create space myspace --project myproject --team myteam
########################################################
	`)
	if upgrade.IsPlugin == "true" {
		description = `
########################################################
################ devspace create space #################
########################################################
Creates a new space for the given project, if
it does not yet exist.

Example:
devspace create space myspace
devspace create space myspace --project myproject
devspace create space myspace --project myproject --team myteam
########################################################
	`
	}
	c := &cobra.Command{
		Use:   "space" + util.SpaceNameOnlyUseLine,
		Short: "Creates a new space in the given cluster",
		Long:  description,
		Args:  util.SpaceNameOnlyValidator,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Check for newer version
			upgrade.PrintNewerVersionWarning()

			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	p, _ := defaults.Get(pdefaults.KeyProject, "")
	c.Flags().StringVar(&cmd.DisplayName, "display-name", "", "The display name to show in the UI for this space")
	c.Flags().StringVar(&cmd.Description, "description", "", "The description to show in the UI for this space")
	c.Flags().StringSliceVar(&cmd.Links, "link", []string{}, linksHelpText)
	c.Flags().StringVar(&cmd.Cluster, "cluster", "", "The cluster to use")
	c.Flags().StringVarP(&cmd.Project, "project", "p", p, "The project to use")
	c.Flags().StringSliceVarP(&cmd.Labels, "labels", "l", []string{}, "Comma separated labels to apply to the space")
	c.Flags().StringSliceVar(&cmd.Annotations, "annotations", []string{}, "Comma separated annotations to apply to the space")
	c.Flags().StringVar(&cmd.User, "user", "", "The user to create the space for")
	c.Flags().StringVar(&cmd.Team, "team", "", "The team to create the space for")
	c.Flags().Int64Var(&cmd.SleepAfter, "sleep-after", 0, "DEPRECATED: If set to non zero, will tell the space to sleep after specified seconds of inactivity")
	c.Flags().Int64Var(&cmd.DeleteAfter, "delete-after", 0, "DEPRECATED: If set to non zero, will tell loft to delete the space after specified seconds of inactivity")
	c.Flags().BoolVar(&cmd.CreateContext, "create-context", true, product.Replace("If loft should create a kube context for the space"))
	c.Flags().BoolVar(&cmd.SwitchContext, "switch-context", true, product.Replace("If loft should switch the current context to the new context"))
	c.Flags().BoolVar(&cmd.SkipWait, "skip-wait", false, "If true, will not wait until the space is running")
	c.Flags().BoolVar(&cmd.Recreate, "recreate", false, product.Replace("If enabled and there already exists a space with this name, Loft will delete it first"))
	c.Flags().BoolVar(&cmd.Update, "update", false, "If enabled and a space already exists, will update the template, version and parameters")
	c.Flags().BoolVar(&cmd.UseExisting, "use", false, product.Replace("If loft should use the space if its already there"))
	c.Flags().StringVar(&cmd.Template, "template", "", "The space template to use")
	c.Flags().StringVar(&cmd.Version, "version", "", "The template version to use")
	c.Flags().StringSliceVar(&cmd.Set, "set", []string{}, "Allows specific template parameters to be set. E.g. --set myParameter=myValue")
	c.Flags().StringVar(&cmd.ParametersFile, "parameters", "", "The file where the parameter values for the apps are specified")
	c.Flags().BoolVar(&cmd.DisableDirectClusterEndpoint, "disable-direct-cluster-endpoint", false, "When enabled does not use an available direct cluster endpoint to connect to the space")
	return c
}

// Run executes the command
func (cmd *SpaceCmd) Run(ctx context.Context, args []string) error {
	spaceName := args[0]
	baseClient, err := client.NewClientFromPath(cmd.Config)
	if err != nil {
		return err
	}

	err = client.VerifyVersion(baseClient)
	if err != nil {
		return err
	}

	// determine cluster name
	cmd.Cluster, cmd.Project, err = helper.SelectProjectOrCluster(baseClient, cmd.Cluster, cmd.Project, true, cmd.Log)
	if err != nil {
		return err
	}

	// create legacy space?
	if cmd.Project == "" {
		// create legacy space
		return cmd.legacyCreateSpace(ctx, baseClient, spaceName)
	}

	// create space
	return cmd.createSpace(ctx, baseClient, spaceName)
}

func (cmd *SpaceCmd) createSpace(ctx context.Context, baseClient client.Client, spaceName string) error {
	spaceNamespace := naming.ProjectNamespace(cmd.Project)
	managementClient, err := baseClient.Management()
	if err != nil {
		return err
	}

	// get current user / team
	if cmd.User == "" && cmd.Team == "" {
		userName, teamName, err := helper.GetCurrentUser(ctx, managementClient)
		if err != nil {
			return err
		}
		if userName != nil {
			cmd.User = userName.Name
		} else {
			cmd.Team = teamName.Name
		}
	}

	// delete the existing cluster if needed
	if cmd.Recreate {
		_, err := managementClient.Loft().ManagementV1().SpaceInstances(spaceNamespace).Get(ctx, spaceName, metav1.GetOptions{})
		if err != nil && !kerrors.IsNotFound(err) {
			return fmt.Errorf("couldn't retrieve space instance: %w", err)
		} else if err == nil {
			// delete the space
			err = managementClient.Loft().ManagementV1().SpaceInstances(spaceNamespace).Delete(ctx, spaceName, metav1.DeleteOptions{})
			if err != nil && !kerrors.IsNotFound(err) {
				return fmt.Errorf("couldn't delete space instance: %w", err)
			}
		}
	}

	var spaceInstance *managementv1.SpaceInstance
	// make sure we wait until space is deleted
	spaceInstance, err = managementClient.Loft().ManagementV1().SpaceInstances(spaceNamespace).Get(ctx, spaceName, metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return fmt.Errorf("couldn't retrieve space instance: %w", err)
	} else if err == nil && spaceInstance.DeletionTimestamp != nil {
		cmd.Log.Infof("Waiting until space is deleted...")

		// wait until the space instance is deleted
		waitErr := wait.PollUntilContextTimeout(ctx, time.Second, config.Timeout(), false, func(ctx context.Context) (done bool, err error) {
			spaceInstance, err = managementClient.Loft().ManagementV1().SpaceInstances(spaceNamespace).Get(ctx, spaceName, metav1.GetOptions{})
			if err != nil && !kerrors.IsNotFound(err) {
				return false, err
			} else if err == nil && spaceInstance.DeletionTimestamp != nil {
				return false, nil
			}

			return true, nil
		})
		if waitErr != nil {
			return errors.Wrap(err, "get space instance")
		}

		spaceInstance = nil
	} else if kerrors.IsNotFound(err) {
		spaceInstance = nil
	}

	// if the space already exists and flag is not set, we terminate
	if !cmd.Update && !cmd.UseExisting && spaceInstance != nil {
		return fmt.Errorf("space %s already exists in project %s", spaceName, cmd.Project)
	}

	// create space if necessary
	if spaceInstance == nil {
		// resolve template
		spaceTemplate, resolvedParameters, err := cmd.resolveTemplate(baseClient)
		if err != nil {
			return err
		}

		// create space instance
		zone, offset := time.Now().Zone()
		spaceInstance = &managementv1.SpaceInstance{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: naming.ProjectNamespace(cmd.Project),
				Name:      spaceName,
				Annotations: map[string]string{
					clusterv1.SleepModeTimezoneAnnotation: zone + "#" + strconv.Itoa(offset),
				},
			},
			Spec: managementv1.SpaceInstanceSpec{
				SpaceInstanceSpec: storagev1.SpaceInstanceSpec{
					DisplayName: cmd.DisplayName,
					Description: cmd.Description,
					Owner: &storagev1.UserOrTeam{
						User: cmd.User,
						Team: cmd.Team,
					},
					TemplateRef: &storagev1.TemplateRef{
						Name:    spaceTemplate.Name,
						Version: cmd.Version,
					},
					ClusterRef: storagev1.ClusterRef{
						Cluster: cmd.Cluster,
					},
					Parameters: resolvedParameters,
				},
			},
		}
		SetCustomLinksAnnotation(spaceInstance, cmd.Links)
		_, err = UpdateLabels(spaceInstance, cmd.Labels)
		if err != nil {
			return err
		}
		_, err = UpdateAnnotations(spaceInstance, cmd.Annotations)
		if err != nil {
			return err
		}
		// create space
		cmd.Log.Infof("Creating space %s in project %s with template %s...", ansi.Color(spaceName, "white+b"), ansi.Color(cmd.Project, "white+b"), ansi.Color(spaceTemplate.Name, "white+b"))
		spaceInstance, err = managementClient.Loft().ManagementV1().SpaceInstances(spaceInstance.Namespace).Create(ctx, spaceInstance, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "create space")
		}
	} else if cmd.Update {
		// resolve template
		spaceTemplate, resolvedParameters, err := cmd.resolveTemplate(baseClient)
		if err != nil {
			return err
		}

		// update space instance
		if spaceInstance.Spec.TemplateRef == nil {
			return fmt.Errorf("space instance doesn't use a template, cannot update space")
		}

		oldSpace := spaceInstance.DeepCopy()

		templateRefChanged := spaceInstance.Spec.TemplateRef.Name != spaceTemplate.Name
		paramsChanged := spaceInstance.Spec.Parameters != resolvedParameters
		versionChanged := (cmd.Version != "" && spaceInstance.Spec.TemplateRef.Version != cmd.Version)
		linksChanged := SetCustomLinksAnnotation(spaceInstance, cmd.Links)
		labelsChanged, err := UpdateLabels(spaceInstance, cmd.Labels)
		if err != nil {
			return err
		}
		annotationsChanged, err := UpdateAnnotations(spaceInstance, cmd.Annotations)
		if err != nil {
			return err
		}

		// check if update is needed
		if templateRefChanged || paramsChanged || versionChanged || linksChanged || labelsChanged || annotationsChanged {
			spaceInstance.Spec.TemplateRef.Name = spaceTemplate.Name
			spaceInstance.Spec.TemplateRef.Version = cmd.Version
			spaceInstance.Spec.Parameters = resolvedParameters

			patch := client2.MergeFrom(oldSpace)
			patchData, err := patch.Data(spaceInstance)
			if err != nil {
				return errors.Wrap(err, "calculate update patch")
			}
			cmd.Log.Infof("Updating space cluster %s in project %s...", ansi.Color(spaceName, "white+b"), ansi.Color(cmd.Project, "white+b"))
			cmd.Log.Debugf("Patch data:\n%s\n...", string(patchData))
			spaceInstance, err = managementClient.Loft().ManagementV1().SpaceInstances(spaceInstance.Namespace).Patch(ctx, spaceInstance.Name, patch.Type(), patchData, metav1.PatchOptions{})
			if err != nil {
				return errors.Wrap(err, "patch space")
			}
		} else {
			cmd.Log.Infof("Skip updating space...")
		}
	}

	// wait until space is ready
	spaceInstance, err = space.WaitForSpaceInstance(ctx, managementClient, spaceInstance.Namespace, spaceInstance.Name, !cmd.SkipWait, cmd.Log)
	if err != nil {
		return err
	}
	cmd.Log.Donef("Successfully created the space %s in project %s", ansi.Color(spaceName, "white+b"), ansi.Color(cmd.Project, "white+b"))

	// should we create a kube context for the space
	if cmd.CreateContext {
		// create kube context options
		contextOptions, err := use.CreateSpaceInstanceOptions(ctx, baseClient, cmd.Config, cmd.Project, spaceInstance, cmd.DisableDirectClusterEndpoint, cmd.SwitchContext, cmd.Log)
		if err != nil {
			return err
		}

		// update kube config
		err = kubeconfig.UpdateKubeConfig(contextOptions)
		if err != nil {
			return err
		}

		cmd.Log.Donef("Successfully updated kube context to use space %s in project %s", ansi.Color(spaceName, "white+b"), ansi.Color(cmd.Project, "white+b"))
	}

	return nil
}

func (cmd *SpaceCmd) resolveTemplate(baseClient client.Client) (*managementv1.SpaceTemplate, string, error) {
	// determine space template to use
	spaceTemplate, err := helper.SelectSpaceTemplate(baseClient, cmd.Project, cmd.Template, cmd.Log)
	if err != nil {
		return nil, "", err
	}

	// get parameters
	var templateParameters []storagev1.AppParameter
	if len(spaceTemplate.Spec.Versions) > 0 {
		if cmd.Version == "" {
			latestVersion := version.GetLatestVersion(spaceTemplate)
			if latestVersion == nil {
				return nil, "", fmt.Errorf("couldn't find any version in template")
			}

			templateParameters = latestVersion.(*storagev1.SpaceTemplateVersion).Parameters
		} else {
			_, latestMatched, err := version.GetLatestMatchedVersion(spaceTemplate, cmd.Version)
			if err != nil {
				return nil, "", err
			} else if latestMatched == nil {
				return nil, "", fmt.Errorf("couldn't find any matching version to %s", cmd.Version)
			}

			templateParameters = latestMatched.(*storagev1.SpaceTemplateVersion).Parameters
		}
	} else {
		templateParameters = spaceTemplate.Spec.Parameters
	}

	// resolve space template parameters
	resolvedParameters, err := parameters.ResolveTemplateParameters(cmd.Set, templateParameters, cmd.ParametersFile)
	if err != nil {
		return nil, "", err
	}

	return spaceTemplate, resolvedParameters, nil
}

func (cmd *SpaceCmd) legacyCreateSpace(ctx context.Context, baseClient client.Client, spaceName string) error {
	var err error
	if cmd.SkipWait {
		cmd.Log.Warnf("--skip-wait is not supported for legacy space creation, please specify a project instead")
	}

	// determine cluster name
	if cmd.Cluster == "" {
		cmd.Cluster, err = helper.SelectCluster(baseClient, cmd.Log)
		if err != nil {
			return err
		}
	}

	// determine user or team name
	if cmd.User == "" && cmd.Team == "" {
		user, team, err := helper.SelectUserOrTeam(baseClient, cmd.Cluster, cmd.Log)
		if err != nil {
			return err
		} else if user != nil {
			cmd.User = user.Name
		} else if team != nil {
			cmd.Team = team.Name
		}
	}

	managementClient, err := baseClient.Management()
	if err != nil {
		return err
	}

	// get current user / team
	userName, teamName, err := helper.GetCurrentUser(ctx, managementClient)
	if err != nil {
		return err
	}

	// create a cluster client
	clusterClient, err := baseClient.Cluster(cmd.Cluster)
	if err != nil {
		return err
	}

	// check if the cluster exists
	cluster, err := managementClient.Loft().ManagementV1().Clusters().Get(ctx, cmd.Cluster, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsForbidden(err) {
			return fmt.Errorf("cluster '%s' does not exist, or you don't have permission to use it", cmd.Cluster)
		}

		return err
	}

	spaceNotFound := true
	if cmd.UseExisting {
		_, err := clusterClient.Agent().ClusterV1().Spaces().Get(ctx, spaceName, metav1.GetOptions{})
		if err != nil && !kerrors.IsNotFound(err) {
			return err
		}

		spaceNotFound = kerrors.IsNotFound(err)
	}

	if spaceNotFound {
		// get default space template
		spaceTemplate, err := resolveSpaceTemplate(ctx, managementClient, cluster, cmd.Template)
		if err != nil {
			return errors.Wrap(err, "resolve space template")
		} else if spaceTemplate != nil {
			cmd.Log.Infof("Using space template %s to create space %s", clihelper.GetDisplayName(spaceTemplate.Name, spaceTemplate.Spec.DisplayName), spaceName)
		}

		// create space object
		space := &clusterv1.Space{
			ObjectMeta: metav1.ObjectMeta{
				Name:        spaceName,
				Annotations: map[string]string{},
			},
		}
		if cmd.User != "" {
			space.Spec.User = cmd.User
		} else if cmd.Team != "" {
			space.Spec.Team = cmd.Team
		}
		if spaceTemplate != nil {
			space.Annotations = spaceTemplate.Spec.Template.Annotations
			space.Labels = spaceTemplate.Spec.Template.Labels
			if space.Annotations == nil {
				space.Annotations = map[string]string{}
			}
			space.Annotations["loft.sh/space-template"] = spaceTemplate.Name
			if spaceTemplate.Spec.Template.Objects != "" {
				space.Spec.Objects = spaceTemplate.Spec.Template.Objects
			}
		}
		if cmd.SleepAfter > 0 {
			space.Annotations[clusterv1.SleepModeSleepAfterAnnotation] = strconv.FormatInt(cmd.SleepAfter, 10)
		}
		if cmd.DeleteAfter > 0 {
			space.Annotations[clusterv1.SleepModeDeleteAfterAnnotation] = strconv.FormatInt(cmd.DeleteAfter, 10)
		}
		if cmd.DisplayName != "" {
			space.Annotations["loft.sh/display-name"] = cmd.DisplayName
		}
		if cmd.Description != "" {
			space.Annotations["loft.sh/description"] = cmd.Description
		}
		SetCustomLinksAnnotation(space, cmd.Links)

		zone, offset := time.Now().Zone()
		space.Annotations[clusterv1.SleepModeTimezoneAnnotation] = zone + "#" + strconv.Itoa(offset)

		if spaceTemplate != nil && len(spaceTemplate.Spec.Template.Apps) > 0 {
			createTask := &managementv1.Task{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "create-space-",
				},
				Spec: managementv1.TaskSpec{
					TaskSpec: storagev1.TaskSpec{
						DisplayName: "Create Space " + spaceName,
						Target: storagev1.Target{
							Cluster: &storagev1.TargetCluster{
								Cluster: cmd.Cluster,
							},
						},
						Task: storagev1.TaskDefinition{
							SpaceCreationTask: &storagev1.SpaceCreationTask{
								Metadata: metav1.ObjectMeta{
									Name:        space.Name,
									Labels:      space.Labels,
									Annotations: space.Annotations,
								},
								Owner: &storagev1.UserOrTeam{
									User: space.Spec.User,
									Team: space.Spec.Team,
								},
							},
						},
					},
				},
			}
			if userName != nil {
				createTask.Spec.Access = []storagev1.Access{
					{
						Verbs:        []string{"*"},
						Subresources: []string{"*"},
						Users:        []string{userName.Name},
					},
				}
			} else if teamName != nil {
				createTask.Spec.Access = []storagev1.Access{
					{
						Verbs:        []string{"*"},
						Subresources: []string{"*"},
						Teams:        []string{teamName.Name},
					},
				}
			}

			apps, err := resolveApps(ctx, managementClient, spaceTemplate.Spec.Template.Apps)
			if err != nil {
				return errors.Wrap(err, "resolve space template apps")
			}

			appsWithParameters, err := parameters.ResolveAppParameters(apps, cmd.ParametersFile, cmd.Log)
			if err != nil {
				return err
			}

			for _, appWithParameter := range appsWithParameters {
				createTask.Spec.Task.SpaceCreationTask.Apps = append(createTask.Spec.Task.SpaceCreationTask.Apps, agentstoragev1.AppReference{
					Name:       appWithParameter.App.Name,
					Parameters: appWithParameter.Parameters,
				})
			}

			// create the task and stream
			err = task.StreamTask(ctx, managementClient, createTask, os.Stdout, cmd.Log)
			if err != nil {
				return err
			}
		} else {
			// create the space
			_, err = clusterClient.Agent().ClusterV1().Spaces().Create(ctx, space, metav1.CreateOptions{})
			if err != nil {
				return errors.Wrap(err, "create space")
			}
		}

		cmd.Log.Donef("Successfully created the space %s in cluster %s", ansi.Color(spaceName, "white+b"), ansi.Color(cmd.Cluster, "white+b"))
	}

	// should we create a kube context for the space
	if cmd.CreateContext || cmd.UseExisting {
		// create kube context options
		contextOptions, err := use.CreateClusterContextOptions(baseClient, cmd.Config, cluster, spaceName, cmd.DisableDirectClusterEndpoint, cmd.SwitchContext, cmd.Log)
		if err != nil {
			return err
		}

		// update kube config
		err = kubeconfig.UpdateKubeConfig(contextOptions)
		if err != nil {
			return err
		}

		cmd.Log.Donef("Successfully updated kube context to use space %s in cluster %s", ansi.Color(spaceName, "white+b"), ansi.Color(cmd.Cluster, "white+b"))
	}

	return nil
}

func resolveSpaceTemplate(ctx context.Context, client kube.Interface, cluster *managementv1.Cluster, template string) (*managementv1.SpaceTemplate, error) {
	if template == "" && cluster.Annotations != nil && cluster.Annotations[constants.LoftDefaultSpaceTemplate] != "" {
		template = cluster.Annotations[constants.LoftDefaultSpaceTemplate]
	}

	if template != "" {
		spaceTemplate, err := client.Loft().ManagementV1().SpaceTemplates().Get(ctx, template, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		return spaceTemplate, nil
	}

	return nil, nil
}

func resolveApps(ctx context.Context, client kube.Interface, apps []agentstoragev1.AppReference) ([]parameters.NamespacedApp, error) {
	appsList, err := client.Loft().ManagementV1().Apps().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	retApps := []parameters.NamespacedApp{}
	for _, a := range apps {
		found := false
		for _, ma := range appsList.Items {
			if a.Name == "" {
				continue
			}
			if ma.Name == a.Name {
				app := ma
				retApps = append(retApps, parameters.NamespacedApp{
					App: &app,
				})
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("couldn't find app %s. The app either doesn't exist or you have no access to use it", a)
		}
	}

	return retApps, nil
}
