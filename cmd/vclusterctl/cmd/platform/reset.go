package platform

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"

	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/survey"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform/clihelper"
	"github.com/loft-sh/vcluster/pkg/platform/kube"
	"github.com/loft-sh/vcluster/pkg/platform/random"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type PasswordCmd struct {
	*flags.GlobalFlags
	Log log.Logger

	Namespace string
	User      string
	Password  string
	Create    bool
	Force     bool
}

func NewResetCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	description := `########################################################
############# vcluster platform reset ##################
########################################################
	`
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset configuration",
		Long:  description,
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(NewPasswordCmd(globalFlags))

	return cmd
}

func NewPasswordCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &PasswordCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	description := `########################################################
######### vcluster platform reset password #############
########################################################
Resets the password of a user.

Example:
vcluster platform reset password
vcluster platform reset password --user admin
#######################################################
	`

	c := &cobra.Command{
		Use:   "password",
		Short: "Resets the password of a user",
		Long:  description,
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run()
		},
	}

	c.Flags().StringVar(&cmd.User, "user", "admin", "The name of the user to reset the password")
	c.Flags().StringVar(&cmd.Password, "password", "", "The new password to use")
	c.Flags().BoolVar(&cmd.Create, "create", false, "Creates the user if it does not exist")
	c.Flags().BoolVar(&cmd.Force, "force", false, "If user had no password will create one")
	c.Flags().StringVar(&cmd.Namespace, "namespace", clihelper.DefaultPlatformNamespace, "The namespace to use")

	return c
}

// Run executes the functionality
func (cmd *PasswordCmd) Run() error {
	restConfig, err := ctrl.GetConfig()
	if err != nil {
		return errors.Wrap(err, "get kube config")
	}

	managementClient, err := kube.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	// check if cluster has platform installed
	apiResourceList, err := managementClient.Discovery().ServerResourcesForGroupVersion(storagev1.GroupVersion.String())
	if err != nil {
		return fmt.Errorf("looks like the api group storage.loft.sh couldn't be found (%w). Please make sure you select the kube context the platform was installed in for this command to work", err)
	}
	found := false
	for _, apiResource := range apiResourceList.APIResources {
		if apiResource.Kind == "User" {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("resource User in api group storage.loft.sh couldn't be found. Please make sure you select the kube context the platform was installed in for this command to work")
	}

	// get user
	cmd.Log.Infof("Resetting password of user %s", cmd.User)
	user, err := managementClient.Loft().StorageV1().Users().Get(context.Background(), cmd.User, metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return errors.Wrap(err, "get user")
	} else if kerrors.IsNotFound(err) {
		// create user
		if !cmd.Create {
			return fmt.Errorf("user %s was not found, run with '--create' to create this user automatically", cmd.User)
		}

		if cmd.Namespace == "" {
			namespace, err := clihelper.VClusterPlatformInstallationNamespace(context.Background())
			if err != nil {
				return fmt.Errorf("failed to find platform namespace: %w", err)
			}

			cmd.Namespace = namespace
		}

		user, err = managementClient.Loft().StorageV1().Users().Create(context.Background(), &storagev1.User{
			ObjectMeta: metav1.ObjectMeta{
				Name: cmd.User,
			},
			Spec: storagev1.UserSpec{
				Username: cmd.User,
				Subject:  cmd.User,
				Groups: []string{
					"system:masters",
				},
				PasswordRef: &storagev1.SecretRef{
					SecretName:      "loft-password-" + random.String(5),
					SecretNamespace: cmd.Namespace,
					Key:             "password",
				},
			},
		}, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	}
	if user == nil {
		return errors.New("could not obtain user")
	}

	// check if user had a password before
	if user.Spec.PasswordRef == nil || user.Spec.PasswordRef.SecretName == "" || user.Spec.PasswordRef.SecretNamespace == "" || user.Spec.PasswordRef.Key == "" {
		if !cmd.Force {
			return fmt.Errorf("user %s had no password. If you want to force password creation, please run with the '--force' flag", cmd.User)
		}

		user.Spec.PasswordRef = &storagev1.SecretRef{
			SecretName:      "loft-password-" + random.String(5),
			SecretNamespace: cmd.Namespace,
			Key:             "password",
		}
		user, err = managementClient.Loft().StorageV1().Users().Update(context.Background(), user, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrap(err, "update user")
		}
	}

	// now ask user for new password
	password := cmd.Password
	if password == "" {
		for {
			password, err = cmd.Log.Question(&survey.QuestionOptions{
				Question:   "Please enter a new password",
				IsPassword: true,
			})
			password = strings.TrimSpace(password)
			if err != nil {
				return err
			} else if password == "" {
				cmd.Log.Error("Please enter a password")
				continue
			}

			break
		}
	}
	passwordHash := []byte(fmt.Sprintf("%x", sha256.Sum256([]byte(password))))

	// check if secret exists
	passwordSecret, err := managementClient.CoreV1().Secrets(user.Spec.PasswordRef.SecretNamespace).Get(context.Background(), user.Spec.PasswordRef.SecretName, metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	} else if kerrors.IsNotFound(err) {
		_, err = managementClient.CoreV1().Secrets(user.Spec.PasswordRef.SecretNamespace).Create(context.Background(), &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      user.Spec.PasswordRef.SecretName,
				Namespace: user.Spec.PasswordRef.SecretNamespace,
			},
			Data: map[string][]byte{
				user.Spec.PasswordRef.Key: passwordHash,
			},
		}, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "create password secret")
		}
	} else {
		if passwordSecret == nil {
			passwordSecret = &corev1.Secret{}
		}
		if passwordSecret.Data == nil {
			passwordSecret.Data = map[string][]byte{}
		}
		passwordSecret.Data[user.Spec.PasswordRef.Key] = passwordHash
		_, err = managementClient.CoreV1().Secrets(user.Spec.PasswordRef.SecretNamespace).Update(context.Background(), passwordSecret, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrap(err, "update password secret")
		}
	}

	cmd.Log.Donef("Successfully reset password of user %s", cmd.User)
	return nil
}
