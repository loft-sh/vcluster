package cmd

import (
	"fmt"
	"os"

	storagev1 "github.com/loft-sh/api/v3/pkg/apis/storage/v1"
	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/backup"
	"github.com/loft-sh/loftctl/v3/pkg/clihelper"
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
			Question:     product.Replace("Seems like Loft was not installed into namespace %s, do you want to continue?"),
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
