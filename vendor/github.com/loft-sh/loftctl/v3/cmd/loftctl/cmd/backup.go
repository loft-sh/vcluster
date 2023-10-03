package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ghodss/yaml"
	storagev1 "github.com/loft-sh/api/v3/pkg/apis/storage/v1"
	loftclient "github.com/loft-sh/api/v3/pkg/client/clientset_generated/clientset"
	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/client/naming"
	"github.com/loft-sh/loftctl/v3/pkg/clihelper"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/survey"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	scheme = runtime.NewScheme()

	_ = clientgoscheme.AddToScheme(scheme)
	_ = storagev1.AddToScheme(scheme)
)

// BackupCmd holds the cmd flags
type BackupCmd struct {
	*flags.GlobalFlags

	Namespace string
	Skip      []string
	Filename  string

	Log log.Logger
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

	objects := []runtime.Object{}
	if !contains(cmd.Skip, "clusterroletemplates") {
		cmd.Log.Info("Backing up clusterroletemplates...")
		objs, err := backupClusterRoles(ctx, kubeConfig)
		if err != nil {
			cmd.Log.Warn(errors.Wrap(err, "backup clusterroletemplates"))
		} else {
			objects = append(objects, objs...)
		}
	}
	if !contains(cmd.Skip, "clusteraccesses") {
		cmd.Log.Info("Backing up clusteraccesses...")
		users, err := backupClusterAccess(ctx, kubeConfig)
		if err != nil {
			cmd.Log.Warn(errors.Wrap(err, "backup clusteraccesses"))
		} else {
			objects = append(objects, users...)
		}
	}
	if !contains(cmd.Skip, "spaceconstraints") {
		cmd.Log.Info("Backing up spaceconstraints...")
		objs, err := backupSpaceConstraints(ctx, kubeConfig)
		if err != nil {
			cmd.Log.Warn(errors.Wrap(err, "backup spaceconstraints"))
		} else {
			objects = append(objects, objs...)
		}
	}
	if !contains(cmd.Skip, "users") {
		cmd.Log.Info("Backing up users...")
		objs, err := backupUsers(ctx, kubeClient, kubeConfig)
		if err != nil {
			cmd.Log.Warn(errors.Wrap(err, "backup users"))
		} else {
			objects = append(objects, objs...)
		}
	}
	if !contains(cmd.Skip, "teams") {
		cmd.Log.Info("Backing up teams...")
		objs, err := backupTeams(ctx, kubeConfig)
		if err != nil {
			cmd.Log.Warn(errors.Wrap(err, "backup teams"))
		} else {
			objects = append(objects, objs...)
		}
	}
	if !contains(cmd.Skip, "sharedsecrets") {
		cmd.Log.Info("Backing up shared secrets...")
		objs, err := backupSharedSecrets(ctx, kubeConfig)
		if err != nil {
			cmd.Log.Warn(errors.Wrap(err, "backup shared secrets"))
		} else {
			objects = append(objects, objs...)
		}
	}
	if !contains(cmd.Skip, "accesskeys") {
		cmd.Log.Info("Backing up access keys...")
		objs, err := backupAccessKeys(ctx, kubeConfig)
		if err != nil {
			cmd.Log.Warn(errors.Wrap(err, "backup access keys"))
		} else {
			objects = append(objects, objs...)
		}
	}
	if !contains(cmd.Skip, "apps") {
		cmd.Log.Info("Backing up apps...")
		objs, err := backupApps(ctx, kubeConfig)
		if err != nil {
			cmd.Log.Warn(errors.Wrap(err, "backup apps"))
		} else {
			objects = append(objects, objs...)
		}
	}
	if !contains(cmd.Skip, "spacetemplates") {
		cmd.Log.Info("Backing up space templates...")
		objs, err := backupSpaceTemplates(ctx, kubeConfig)
		if err != nil {
			cmd.Log.Warn(errors.Wrap(err, "backup space templates"))
		} else {
			objects = append(objects, objs...)
		}
	}
	if !contains(cmd.Skip, "virtualclustertemplates") {
		cmd.Log.Info("Backing up virtual cluster templates...")
		objs, err := backupVirtualClusterTemplate(ctx, kubeConfig)
		if err != nil {
			cmd.Log.Warn(errors.Wrap(err, "backup virtual cluster templates"))
		} else {
			objects = append(objects, objs...)
		}
	}
	if !contains(cmd.Skip, "clusters") {
		cmd.Log.Info("Backing up clusters...")
		objs, err := backupClusters(ctx, kubeClient, kubeConfig)
		if err != nil {
			cmd.Log.Warn(errors.Wrap(err, "backup clusters"))
		} else {
			objects = append(objects, objs...)
		}
	}
	projects := []string{}
	if !contains(cmd.Skip, "projects") {
		cmd.Log.Info("Backing up projects...")
		objs, projectNames, err := backupProjects(ctx, kubeConfig)
		if err != nil {
			cmd.Log.Warn(errors.Wrap(err, "backup projects"))
		} else {
			objects = append(objects, objs...)
		}

		projects = projectNames
	}
	if len(projects) > 0 {
		if !contains(cmd.Skip, "virtualclusterinstances") {
			cmd.Log.Info("Backing up virtualcluster instances...")
			objs, err := backupVirtualClusterInstances(ctx, kubeConfig, projects)
			if err != nil {
				cmd.Log.Warn(errors.Wrap(err, "backup virtualcluster instances"))
			} else {
				objects = append(objects, objs...)
			}
		}
		if !contains(cmd.Skip, "spaceinstances") {
			cmd.Log.Info("Backing up space instances...")
			objs, err := backupSpaceInstances(ctx, kubeConfig, projects)
			if err != nil {
				cmd.Log.Warn(errors.Wrap(err, "backup space instances"))
			} else {
				objects = append(objects, objs...)
			}
		}
		if !contains(cmd.Skip, "projectsecrets") {
			cmd.Log.Info("Backing up project secrets...")
			objs, err := backupProjectSecrets(ctx, kubeClient, projects)
			if err != nil {
				cmd.Log.Warn(errors.Wrap(err, "backup project secrets"))
			} else {
				objects = append(objects, objs...)
			}
		}
	}

	// create a file
	retString := []string{}
	for _, o := range objects {
		out, err := yaml.Marshal(o)
		if err != nil {
			return errors.Wrap(err, "marshal object")
		}

		retString = append(retString, string(out))
	}

	cmd.Log.Infof("Writing backup to %s...", cmd.Filename)
	err = os.WriteFile(cmd.Filename, []byte(strings.Join(retString, "\n---\n")), 0644)
	if err != nil {
		return err
	}

	cmd.Log.Donef("Wrote backup to %s", cmd.Filename)
	return nil
}

func backupProjects(ctx context.Context, rest *rest.Config) ([]runtime.Object, []string, error) {
	loftClient, err := loftclient.NewForConfig(rest)
	if err != nil {
		return nil, nil, err
	}

	projectList, err := loftClient.StorageV1().Projects().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, nil, err
	}

	retList := []runtime.Object{}

	projectNames := []string{}
	for _, project := range projectList.Items {
		u := project

		err := resetMetadata(&u)
		if err != nil {
			return nil, nil, err
		}

		retList = append(retList, &u)
		projectNames = append(projectNames, u.Name)
	}

	return retList, projectNames, nil
}

func backupVirtualClusterInstances(ctx context.Context, rest *rest.Config, projects []string) ([]runtime.Object, error) {
	loftClient, err := loftclient.NewForConfig(rest)
	if err != nil {
		return nil, err
	}

	retList := []runtime.Object{}
	for _, projectName := range projects {
		virtualClusterInstanceList, err := loftClient.StorageV1().VirtualClusterInstances(naming.ProjectNamespace(projectName)).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, err
		}

		for _, o := range virtualClusterInstanceList.Items {
			u := o
			u.Status = storagev1.VirtualClusterInstanceStatus{}
			err := resetMetadata(&u)
			if err != nil {
				return nil, err
			}

			retList = append(retList, &u)
		}
	}

	return retList, nil
}

func backupSpaceInstances(ctx context.Context, rest *rest.Config, projects []string) ([]runtime.Object, error) {
	loftClient, err := loftclient.NewForConfig(rest)
	if err != nil {
		return nil, err
	}

	retList := []runtime.Object{}
	for _, projectName := range projects {
		spaceInstanceList, err := loftClient.StorageV1().SpaceInstances(naming.ProjectNamespace(projectName)).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, err
		}

		for _, o := range spaceInstanceList.Items {
			u := o
			u.Status = storagev1.SpaceInstanceStatus{}
			err := resetMetadata(&u)
			if err != nil {
				return nil, err
			}

			retList = append(retList, &u)
		}
	}

	return retList, nil
}

func backupProjectSecrets(ctx context.Context, kubeClient kubernetes.Interface, projects []string) ([]runtime.Object, error) {
	retList := []runtime.Object{}
	for _, projectName := range projects {
		secretList, err := kubeClient.CoreV1().Secrets(naming.ProjectNamespace(projectName)).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, err
		}

		for _, secret := range secretList.Items {
			u := secret

			if !isProjectSecret(u) {
				continue
			}

			err := resetMetadata(&u)
			if err != nil {
				return nil, err
			}

			retList = append(retList, &u)
		}
	}

	return retList, nil
}

func backupClusters(ctx context.Context, kubeClient kubernetes.Interface, rest *rest.Config) ([]runtime.Object, error) {
	loftClient, err := loftclient.NewForConfig(rest)
	if err != nil {
		return nil, err
	}

	clusterList, err := loftClient.StorageV1().Clusters().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	retList := []runtime.Object{}
	for _, o := range clusterList.Items {
		u := o
		u.Status = storagev1.ClusterStatus{}
		err := resetMetadata(&u)
		if err != nil {
			return nil, err
		}

		retList = append(retList, &u)

		// find user secrets
		if u.Spec.Config.SecretName != "" {
			secret, err := getSecret(ctx, kubeClient, u.Spec.Config.SecretNamespace, u.Spec.Config.SecretName)
			if err != nil {
				return nil, errors.Wrap(err, "get cluster secret")
			} else if secret != nil {
				retList = append(retList, secret)
			}
		}
	}

	return retList, nil
}

func backupClusterRoles(ctx context.Context, rest *rest.Config) ([]runtime.Object, error) {
	loftClient, err := loftclient.NewForConfig(rest)
	if err != nil {
		return nil, err
	}

	objs, err := loftClient.StorageV1().ClusterRoleTemplates().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	retList := []runtime.Object{}
	for _, o := range objs.Items {
		u := o
		u.Status = storagev1.ClusterRoleTemplateStatus{}
		err := resetMetadata(&u)
		if err != nil {
			return nil, err
		}

		retList = append(retList, &u)
	}

	return retList, nil
}

func backupSpaceConstraints(ctx context.Context, rest *rest.Config) ([]runtime.Object, error) {
	loftClient, err := loftclient.NewForConfig(rest)
	if err != nil {
		return nil, err
	}

	objs, err := loftClient.StorageV1().SpaceConstraints().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	retList := []runtime.Object{}
	for _, o := range objs.Items {
		u := o
		u.Status = storagev1.SpaceConstraintStatus{}
		err := resetMetadata(&u)
		if err != nil {
			return nil, err
		}

		retList = append(retList, &u)
	}

	return retList, nil
}

func backupClusterAccess(ctx context.Context, rest *rest.Config) ([]runtime.Object, error) {
	loftClient, err := loftclient.NewForConfig(rest)
	if err != nil {
		return nil, err
	}

	objs, err := loftClient.StorageV1().ClusterAccesses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	retList := []runtime.Object{}
	for _, o := range objs.Items {
		u := o
		u.Status = storagev1.ClusterAccessStatus{}
		err := resetMetadata(&u)
		if err != nil {
			return nil, err
		}

		retList = append(retList, &u)
	}

	return retList, nil
}

func backupVirtualClusterTemplate(ctx context.Context, rest *rest.Config) ([]runtime.Object, error) {
	loftClient, err := loftclient.NewForConfig(rest)
	if err != nil {
		return nil, err
	}

	apps, err := loftClient.StorageV1().VirtualClusterTemplates().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	retList := []runtime.Object{}
	for _, o := range apps.Items {
		u := o
		u.Status = storagev1.VirtualClusterTemplateStatus{}
		err := resetMetadata(&u)
		if err != nil {
			return nil, err
		}

		retList = append(retList, &u)
	}

	return retList, nil
}

func backupSpaceTemplates(ctx context.Context, rest *rest.Config) ([]runtime.Object, error) {
	loftClient, err := loftclient.NewForConfig(rest)
	if err != nil {
		return nil, err
	}

	apps, err := loftClient.StorageV1().SpaceTemplates().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	retList := []runtime.Object{}
	for _, o := range apps.Items {
		u := o
		u.Status = storagev1.SpaceTemplateStatus{}
		err := resetMetadata(&u)
		if err != nil {
			return nil, err
		}

		retList = append(retList, &u)
	}

	return retList, nil
}

func backupApps(ctx context.Context, rest *rest.Config) ([]runtime.Object, error) {
	loftClient, err := loftclient.NewForConfig(rest)
	if err != nil {
		return nil, err
	}

	apps, err := loftClient.StorageV1().Apps().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	retList := []runtime.Object{}
	for _, o := range apps.Items {
		u := o
		u.Status = storagev1.AppStatus{}
		err := resetMetadata(&u)
		if err != nil {
			return nil, err
		}

		retList = append(retList, &u)
	}

	return retList, nil
}

func backupAccessKeys(ctx context.Context, rest *rest.Config) ([]runtime.Object, error) {
	loftClient, err := loftclient.NewForConfig(rest)
	if err != nil {
		return nil, err
	}

	accessKeyList, err := loftClient.StorageV1().AccessKeys().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	retList := []runtime.Object{}
	for _, o := range accessKeyList.Items {
		u := o
		u.Status = storagev1.AccessKeyStatus{}
		err := resetMetadata(&u)
		if err != nil {
			return nil, err
		}

		retList = append(retList, &u)
	}

	return retList, nil
}

func backupSharedSecrets(ctx context.Context, rest *rest.Config) ([]runtime.Object, error) {
	loftClient, err := loftclient.NewForConfig(rest)
	if err != nil {
		return nil, err
	}

	sharedSecretList, err := loftClient.StorageV1().SharedSecrets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	retList := []runtime.Object{}
	for _, sharedSecret := range sharedSecretList.Items {
		u := sharedSecret
		u.Status = storagev1.SharedSecretStatus{}
		err := resetMetadata(&u)
		if err != nil {
			return nil, err
		}

		retList = append(retList, &u)
	}

	return retList, nil
}

func backupTeams(ctx context.Context, rest *rest.Config) ([]runtime.Object, error) {
	loftClient, err := loftclient.NewForConfig(rest)
	if err != nil {
		return nil, err
	}

	teamList, err := loftClient.StorageV1().Teams().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	retList := []runtime.Object{}
	for _, team := range teamList.Items {
		u := team
		u.Status = storagev1.TeamStatus{}
		err := resetMetadata(&u)
		if err != nil {
			return nil, err
		}

		retList = append(retList, &u)
	}

	return retList, nil
}

func backupUsers(ctx context.Context, kubeClient kubernetes.Interface, rest *rest.Config) ([]runtime.Object, error) {
	loftClient, err := loftclient.NewForConfig(rest)
	if err != nil {
		return nil, err
	}

	userList, err := loftClient.StorageV1().Users().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	retList := []runtime.Object{}
	for _, user := range userList.Items {
		u := user
		u.Status = storagev1.UserStatus{}
		err := resetMetadata(&u)
		if err != nil {
			return nil, err
		}

		retList = append(retList, &u)

		// find user secrets
		if u.Spec.PasswordRef != nil {
			secret, err := getSecret(ctx, kubeClient, u.Spec.PasswordRef.SecretNamespace, u.Spec.PasswordRef.SecretName)
			if err != nil {
				return nil, errors.Wrap(err, "get user secret")
			} else if secret != nil {
				retList = append(retList, secret)
			}
		}
		if u.Spec.CodesRef != nil {
			secret, err := getSecret(ctx, kubeClient, u.Spec.CodesRef.SecretNamespace, u.Spec.CodesRef.SecretName)
			if err != nil {
				return nil, errors.Wrap(err, "get user secret")
			} else if secret != nil {
				retList = append(retList, secret)
			}
		}
	}

	return retList, nil
}

func getSecret(ctx context.Context, kubeClient kubernetes.Interface, namespace, name string) (*corev1.Secret, error) {
	if namespace == "" || name == "" {
		return nil, nil
	}

	secret, err := kubeClient.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return nil, err
	} else if secret != nil {
		err = resetMetadata(secret)
		if err != nil {
			return nil, errors.Wrap(err, "reset metadata secret")
		}

		return secret, nil
	}

	return nil, nil
}

func resetMetadata(obj runtime.Object) error {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return err
	}

	accessor.SetGenerateName("")
	accessor.SetSelfLink("")
	accessor.SetCreationTimestamp(metav1.Time{})
	accessor.SetFinalizers(nil)
	accessor.SetGeneration(0)
	accessor.SetManagedFields(nil)
	accessor.SetOwnerReferences(nil)
	accessor.SetResourceVersion("")
	accessor.SetUID("")
	accessor.SetDeletionTimestamp(nil)

	gvk, err := GVKFrom(obj)
	if err != nil {
		return err
	}

	typeAccessor, err := meta.TypeAccessor(obj)
	if err != nil {
		return err
	}

	typeAccessor.SetKind(gvk.Kind)
	typeAccessor.SetAPIVersion(gvk.GroupVersion().String())
	return nil
}

func contains(arr []string, s string) bool {
	for _, t := range arr {
		if t == s {
			return true
		}
	}
	return false
}

func isProjectSecret(secret corev1.Secret) bool {
	for k, v := range secret.Labels {
		if k == "loft.sh/project-secret" && v == "true" {
			return true
		}
	}

	return false
}

func GVKFrom(obj runtime.Object) (schema.GroupVersionKind, error) {
	gvks, _, err := scheme.ObjectKinds(obj)
	if err != nil {
		return schema.GroupVersionKind{}, err
	} else if len(gvks) != 1 {
		return schema.GroupVersionKind{}, fmt.Errorf("unexpected number of object kinds: %d", len(gvks))
	}

	return gvks[0], nil
}
