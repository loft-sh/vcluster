package create

import (
	"context"
	"fmt"
	"strconv"
	"time"

	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	cliconfig "github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/kube"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/platform/clihelper"
	pdefaults "github.com/loft-sh/vcluster/pkg/platform/defaults"
	"github.com/loft-sh/vcluster/pkg/platform/kubeconfig"
	"github.com/loft-sh/vcluster/pkg/projectutil"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const linksHelpText = `Labeled Links to annotate the object with.
These links will be visible from the UI. When used with update, existing links will be replaced.
E.g. --link 'Prod=http://exampleprod.com,Dev=http://exampledev.com'`

// NamespaceCmd holds the cmd flags
type NamespaceCmd struct {
	*flags.GlobalFlags

	SleepAfter                   int64
	DeleteAfter                  int64
	Cluster                      string
	Project                      string
	CreateContext                bool
	SwitchContext                bool
	DisableDirectClusterEndpoint bool
	Template                     string
	TemplateVersion              string
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

// newNamespaceCmd creates a new command
func newNamespaceCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	cmd := &NamespaceCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}
	description := product.ReplaceWithHeader("create namespace", `
Creates a new vCluster platform namespace for the given project, if
it does not yet exist.
Example:
vcluster platform create namespace myspace
vcluster platform create namespace myspace --project myproject
vcluster platform create namespace myspace --project myproject --team myteam
########################################################
	`)
	c := &cobra.Command{
		Use:     "namespace" + util.NamespaceNameOnlyUseLine,
		Short:   "Creates a new vCluster platform namespace in the given cluster",
		Long:    description,
		Args:    util.NamespaceNameOnlyValidator,
		Aliases: []string{"space"},
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Check for newer version
			upgrade.PrintNewerVersionWarning()

			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	p, _ := defaults.Get(pdefaults.KeyProject, "")
	c.Flags().StringVar(&cmd.DisplayName, "display-name", "", "The display name to show in the UI for this namespace")
	c.Flags().StringVar(&cmd.Description, "description", "", "The description to show in the UI for this namespace")
	c.Flags().StringSliceVar(&cmd.Links, "link", []string{}, linksHelpText)
	c.Flags().StringVar(&cmd.Cluster, "cluster", "", "The cluster to use")
	c.Flags().StringVarP(&cmd.Project, "project", "p", p, "The project to use")
	c.Flags().StringSliceVarP(&cmd.Labels, "labels", "l", []string{}, "Comma separated labels to apply to the namespace")
	c.Flags().StringSliceVar(&cmd.Annotations, "annotations", []string{}, "Comma separated annotations to apply to the namespace")
	c.Flags().StringVar(&cmd.User, "user", "", "The user to create the namespace for")
	c.Flags().StringVar(&cmd.Team, "team", "", "The team to create the namespace for")
	c.Flags().BoolVar(&cmd.CreateContext, "create-context", true, product.Replace("If loft should create a kube context for the namespace"))
	c.Flags().BoolVar(&cmd.SwitchContext, "switch-context", true, product.Replace("If loft should switch the current context to the new context"))
	c.Flags().BoolVar(&cmd.SkipWait, "skip-wait", false, "If true, will not wait until the namespace is running")
	c.Flags().BoolVar(&cmd.Recreate, "recreate", false, product.Replace("If enabled and there already exists a namespace with this name, Loft will delete it first"))
	c.Flags().BoolVar(&cmd.Update, "update", false, "If enabled and a namespace already exists, will update the template, version and parameters")
	c.Flags().BoolVar(&cmd.UseExisting, "use", false, product.Replace("If loft should use the namespace if its already there"))
	c.Flags().StringVar(&cmd.Template, "template", "", "The namespace template to use")
	c.Flags().StringVar(&cmd.TemplateVersion, "template-version", "", "The template version to use")
	c.Flags().StringSliceVar(&cmd.Set, "set", []string{}, "Allows specific template parameters to be set. E.g. --set myParameter=myValue")
	c.Flags().StringVar(&cmd.ParametersFile, "parameters", "", "The file where the parameter values for the apps are specified")
	c.Flags().BoolVar(&cmd.DisableDirectClusterEndpoint, "disable-direct-cluster-endpoint", false, "When enabled does not use an available direct cluster endpoint to connect to the namespace")
	return c
}

// Run executes the command
func (cmd *NamespaceCmd) Run(ctx context.Context, args []string) error {
	spaceName := args[0]
	cfg := cmd.LoadedConfig(cmd.Log)
	platformClient, err := platform.InitClientFromConfig(ctx, cfg)
	if err != nil {
		return err
	}

	// determine cluster name
	cmd.Cluster, cmd.Project, err = platform.SelectProjectOrCluster(ctx, platformClient, cmd.Cluster, cmd.Project, false, cmd.Log)
	if err != nil {
		return err
	}

	// create space
	return cmd.createSpace(ctx, platformClient, spaceName, cfg)
}

func (cmd *NamespaceCmd) createSpace(ctx context.Context, platformClient platform.Client, spaceName string, cfg *cliconfig.CLI) error {
	spaceNamespace := projectutil.ProjectNamespace(cmd.Project)
	managementClient, err := platformClient.Management()
	if err != nil {
		return err
	}

	// get current user / team
	if cmd.User == "" && cmd.Team == "" {
		userName, teamName, err := platform.GetCurrentUser(ctx, managementClient)
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
			return fmt.Errorf("couldn't retrieve namespace instance: %w", err)
		} else if err == nil {
			// delete the space
			err = managementClient.Loft().ManagementV1().SpaceInstances(spaceNamespace).Delete(ctx, spaceName, metav1.DeleteOptions{})
			if err != nil && !kerrors.IsNotFound(err) {
				return fmt.Errorf("couldn't delete namespace instance: %w", err)
			}
		}
	}

	var spaceInstance *managementv1.SpaceInstance
	// make sure we wait until space is deleted
	spaceInstance, err = managementClient.Loft().ManagementV1().SpaceInstances(spaceNamespace).Get(ctx, spaceName, metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return fmt.Errorf("couldn't retrieve namespace instance: %w", err)
	} else if err == nil && spaceInstance.DeletionTimestamp != nil {
		cmd.Log.Infof("Waiting until namespace is deleted...")

		// wait until the space instance is deleted
		waitErr := wait.PollUntilContextTimeout(ctx, time.Second, clihelper.Timeout(), false, func(ctx context.Context) (done bool, err error) {
			spaceInstance, err = managementClient.Loft().ManagementV1().SpaceInstances(spaceNamespace).Get(ctx, spaceName, metav1.GetOptions{})
			if err != nil && !kerrors.IsNotFound(err) {
				return false, err
			} else if err == nil && spaceInstance.DeletionTimestamp != nil {
				return false, nil
			}

			return true, nil
		})
		if waitErr != nil {
			return errors.Wrap(err, "get namespace instance")
		}

		spaceInstance = nil
	} else if kerrors.IsNotFound(err) {
		spaceInstance = nil
	}

	// if the space already exists and flag is not set, we terminate
	if !cmd.Update && !cmd.UseExisting && spaceInstance != nil {
		return fmt.Errorf("namespace %s already exists in project %s", spaceName, cmd.Project)
	}

	// create space if necessary
	if spaceInstance == nil {
		// resolve template
		spaceTemplate, resolvedParameters, err := cmd.resolveTemplate(ctx, platformClient)
		if err != nil {
			return err
		}

		// create space instance
		zone, offset := time.Now().Zone()
		spaceInstance = &managementv1.SpaceInstance{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: projectutil.ProjectNamespace(cmd.Project),
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
						Version: cmd.TemplateVersion,
					},
					ClusterRef: storagev1.ClusterRef{
						Cluster: cmd.Cluster,
					},
					Parameters: resolvedParameters,
				},
			},
		}
		kube.SetCustomLinksAnnotation(spaceInstance, cmd.Links)
		_, err = kube.UpdateLabels(spaceInstance, cmd.Labels)
		if err != nil {
			return err
		}
		_, err = kube.UpdateAnnotations(spaceInstance, cmd.Annotations)
		if err != nil {
			return err
		}
		// create space
		cmd.Log.Infof("Creating namespace %s in project %s with template %s...", ansi.Color(spaceName, "white+b"), ansi.Color(cmd.Project, "white+b"), ansi.Color(spaceTemplate.Name, "white+b"))
		spaceInstance, err = managementClient.Loft().ManagementV1().SpaceInstances(spaceInstance.Namespace).Create(ctx, spaceInstance, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "create namespace")
		}
	} else if cmd.Update {
		// resolve template
		spaceTemplate, resolvedParameters, err := cmd.resolveTemplate(ctx, platformClient)
		if err != nil {
			return err
		}

		// update space instance
		if spaceInstance.Spec.TemplateRef == nil {
			return fmt.Errorf("namespace instance doesn't use a template, cannot update space")
		}

		oldSpace := spaceInstance.DeepCopy()

		templateRefChanged := spaceInstance.Spec.TemplateRef.Name != spaceTemplate.Name
		paramsChanged := spaceInstance.Spec.Parameters != resolvedParameters
		versionChanged := (cmd.TemplateVersion != "" && spaceInstance.Spec.TemplateRef.Version != cmd.TemplateVersion)
		linksChanged := kube.SetCustomLinksAnnotation(spaceInstance, cmd.Links)
		labelsChanged, err := kube.UpdateLabels(spaceInstance, cmd.Labels)
		if err != nil {
			return err
		}
		annotationsChanged, err := kube.UpdateAnnotations(spaceInstance, cmd.Annotations)
		if err != nil {
			return err
		}

		// check if update is needed
		if templateRefChanged || paramsChanged || versionChanged || linksChanged || labelsChanged || annotationsChanged {
			spaceInstance.Spec.TemplateRef.Name = spaceTemplate.Name
			spaceInstance.Spec.TemplateRef.Version = cmd.TemplateVersion
			spaceInstance.Spec.Parameters = resolvedParameters

			patch := crclient.MergeFrom(oldSpace)
			patchData, err := patch.Data(spaceInstance)
			if err != nil {
				return errors.Wrap(err, "calculate update patch")
			}
			cmd.Log.Infof("Updating namespace in cluster %s in project %s...", ansi.Color(spaceName, "white+b"), ansi.Color(cmd.Project, "white+b"))
			cmd.Log.Debugf("Patch data:\n%s\n...", string(patchData))
			spaceInstance, err = managementClient.Loft().ManagementV1().SpaceInstances(spaceInstance.Namespace).Patch(ctx, spaceInstance.Name, patch.Type(), patchData, metav1.PatchOptions{})
			if err != nil {
				return errors.Wrap(err, "patch namespace")
			}
		} else {
			cmd.Log.Infof("Skip updating namespace...")
		}
	}

	// wait until space is ready
	spaceInstance, err = platform.WaitForSpaceInstance(ctx, managementClient, spaceInstance.Namespace, spaceInstance.Name, !cmd.SkipWait, cmd.Log)
	if err != nil {
		return err
	}
	cmd.Log.Donef("Successfully created the namespace %s in project %s", ansi.Color(spaceName, "white+b"), ansi.Color(cmd.Project, "white+b"))

	// should we create a kube context for the space
	if cmd.CreateContext {
		// create kube context options
		contextOptions, err := platform.CreateSpaceInstanceOptions(ctx, platformClient, cmd.Config, cmd.Project, spaceInstance, cmd.SwitchContext)
		if err != nil {
			return err
		}

		// update kube config
		err = kubeconfig.UpdateKubeConfig(contextOptions, cfg)
		if err != nil {
			return err
		}

		cmd.Log.Donef("Successfully updated kube context to use namespace %s in project %s", ansi.Color(spaceName, "white+b"), ansi.Color(cmd.Project, "white+b"))
	}

	return nil
}

func (cmd *NamespaceCmd) resolveTemplate(ctx context.Context, platformClient platform.Client) (*managementv1.SpaceTemplate, string, error) {
	// determine space template to use
	spaceTemplate, err := platform.SelectSpaceTemplate(ctx, platformClient, cmd.Project, cmd.Template, cmd.Log)
	if err != nil {
		return nil, "", err
	}

	// get parameters
	var templateParameters []storagev1.AppParameter
	if len(spaceTemplate.Spec.Versions) > 0 {
		if cmd.TemplateVersion == "" {
			latestVersion := platform.GetLatestVersion(spaceTemplate)
			if latestVersion == nil {
				return nil, "", fmt.Errorf("couldn't find any version in template")
			}

			templateParameters = latestVersion.(*storagev1.SpaceTemplateVersion).Parameters
		} else {
			_, latestMatched, err := platform.GetLatestMatchedVersion(spaceTemplate, cmd.TemplateVersion)
			if err != nil {
				return nil, "", err
			} else if latestMatched == nil {
				return nil, "", fmt.Errorf("couldn't find any matching version to %s", cmd.TemplateVersion)
			}

			templateParameters = latestMatched.(*storagev1.SpaceTemplateVersion).Parameters
		}
	} else {
		templateParameters = spaceTemplate.Spec.Parameters
	}

	// resolve space template parameters
	resolvedParameters, err := platform.ResolveTemplateParameters(cmd.Set, templateParameters, cmd.ParametersFile)
	if err != nil {
		return nil, "", err
	}

	return spaceTemplate, resolvedParameters, nil
}
