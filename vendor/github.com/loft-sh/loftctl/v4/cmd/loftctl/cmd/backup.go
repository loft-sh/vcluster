package cmd

import (
	"fmt"
	"os"

	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/loftctl/v4/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v4/pkg/backup"
	loftclient "github.com/loft-sh/loftctl/v4/pkg/client"
	"github.com/loft-sh/loftctl/v4/pkg/clihelper"
	"github.com/loft-sh/loftctl/v4/pkg/projectutil"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/survey"
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

// BackupCmd holds the cmd flags
type BackupCmd struct {
	*flags.GlobalFlags
	Log       log.Logger
	Namespace string
	Filename  string
	Skip      []string
}

// NewBackupCmd creates a new command
func NewBackupCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &BackupCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	description := product.ReplaceWithHeader("backup", `
Backup creates a backup for the Loft management plane

Example:
loft backup
########################################################
	`)

	c := &cobra.Command{
		Use:   "backup",
		Short: product.Replace("Create a loft management plane backup"),
		Long:  description,
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// create new loft client with that config
			loftClient, err := loftclient.NewClientFromPath(cmd.Config)
			if err != nil {
				return fmt.Errorf("create loft client: %w", err)
			}
			self, err := loftClient.GetSelf(cobraCmd.Context())
			if err != nil {
				return fmt.Errorf("failed to get self: %w", err)
			}
			projectutil.SetProjectNamespacePrefix(self.Status.ProjectNamespacePrefix)
			return cmd.Run(cobraCmd, args)
		},
	}

	c.Flags().StringSliceVar(&cmd.Skip, "skip", []string{}, "What resources the backup should skip. Valid options are: users, teams, accesskeys, sharedsecrets, clusters and clusteraccounttemplates")
	c.Flags().StringVar(&cmd.Namespace, "namespace", "loft", product.Replace("The namespace to loft was installed into"))
	c.Flags().StringVar(&cmd.Filename, "filename", "backup.yaml", "The filename to write the backup to")
	return c
}

// Run executes the functionality
func (cmd *BackupCmd) Run(cobraCmd *cobra.Command, args []string) error {
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

	isInstalled, err := clihelper.IsLoftAlreadyInstalled(cobraCmd.Context(), kubeClient, cmd.Namespace)
	if err != nil {
		return err
	} else if !isInstalled {
		answer, err := cmd.Log.Question(&survey.QuestionOptions{
			Question:     fmt.Sprintf(product.Replace("Seems like Loft was not installed into namespace %q, do you want to continue?"), cmd.Namespace),
			DefaultValue: "Yes",
			Options:      []string{"Yes", "No"},
		})
		if err != nil || answer != "Yes" {
			return err
		}
	}

	ctx := cobraCmd.Context()
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
