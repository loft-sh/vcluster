package backup

import (
	"context"
	"fmt"
	"os"

	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/survey"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/platform/backup"
	"github.com/loft-sh/vcluster/pkg/platform/clihelper"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	clientpkg "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	scheme = runtime.NewScheme()

	_ = clientgoscheme.AddToScheme(scheme)
	_ = storagev1.AddToScheme(scheme)
)

// ManagementCmd holds the cmd flags
type ManagementCmd struct {
	*flags.GlobalFlags
	Log       log.Logger
	Namespace string
	Filename  string
	Skip      []string
}

// newManagementCmd creates a new command for backing up the management plane
func newManagementCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ManagementCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	description := product.ReplaceWithHeader("backup management", `
Backup creates a backup for the vCluster platform management plane

Example:
vcluster platform backup management
########################################################
	`)

	c := &cobra.Command{
		Use:   "management",
		Short: product.Replace("Create a vCluster platform management plane backup"),
		Long:  description,
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			// we need to set the project namespace prefix correctly here
			_, err := platform.InitClientFromConfig(cobraCmd.Context(), cmd.LoadedConfig(cmd.Log))
			if err != nil {
				return fmt.Errorf("create vCluster platform client: %w", err)
			}

			return cmd.run(cobraCmd)
		},
	}

	c.Flags().StringSliceVar(&cmd.Skip, "skip", []string{}, "What resources the backup should skip. Valid options are: users, teams, accesskeys, sharedsecrets, clusters and clusteraccounttemplates")
	c.Flags().StringVar(&cmd.Namespace, "namespace", "", product.Replace("The namespace vCluster platform was installed into"))
	c.Flags().StringVar(&cmd.Filename, "filename", "backup.yaml", "The filename to write the backup to")
	return c
}

// run executes the functionality
func (cmd *ManagementCmd) run(cobraCmd *cobra.Command) error {
	// first load the kube config
	kubeClientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})

	// load the raw config
	kubeConfig, err := kubeClientConfig.ClientConfig()
	if err != nil {
		return fmt.Errorf("there is an error loading your current kube config (%w), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
	}

	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return fmt.Errorf("there is an error loading your current kube config (%w), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
	}

	ctx := cobraCmd.Context()
	ns, err := cmd.getNS(ctx)
	if err != nil {
		return err
	}

	isInstalled, err := clihelper.IsLoftAlreadyInstalled(cobraCmd.Context(), kubeClient, ns)
	if err != nil {
		return err
	} else if !isInstalled {
		answer, err := cmd.Log.Question(&survey.QuestionOptions{
			Question:     fmt.Sprintf(product.Replace("Seems like vCluster platform was not installed into namespace %q, do you want to continue?"), ns),
			DefaultValue: "Yes",
			Options:      []string{"Yes", "No"},
		})
		if err != nil || answer != "Yes" {
			return err
		}
	}

	client, err := clientpkg.New(kubeConfig, clientpkg.Options{Scheme: scheme})
	if err != nil {
		return err
	}

	objects, errors := backup.All(ctx, client, cmd.Skip, func(msg string) {
		cmd.Log.Info(msg)
	})
	for _, err := range errors {
		cmd.Log.Warn(err)
	}
	backupBytes, err := backup.ToYAML(objects)
	if err != nil {
		return err
	}

	// create a file
	cmd.Log.Infof("Writing backup to %s...", cmd.Filename)
	err = os.WriteFile(cmd.Filename, backupBytes, 0644)
	if err != nil {
		return err
	}

	cmd.Log.Donef("Wrote backup to %s", cmd.Filename)
	return nil
}

func (cmd *ManagementCmd) getNS(ctx context.Context) (string, error) {
	if cmd.Namespace != "" {
		return cmd.Namespace, nil
	}
	platformNamespace, err := clihelper.VClusterPlatformInstallationNamespace(ctx)
	if err != nil {
		return "", err
	}
	return platformNamespace, nil
}
