package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	dockerconfig "github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	managementv1 "github.com/loft-sh/api/v3/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v3/pkg/apis/storage/v1"
	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/client"
	"github.com/loft-sh/loftctl/v3/pkg/client/helper"
	"github.com/loft-sh/loftctl/v3/pkg/clihelper"
	"github.com/loft-sh/loftctl/v3/pkg/docker"
	"github.com/loft-sh/loftctl/v3/pkg/kube"
	"github.com/loft-sh/loftctl/v3/pkg/upgrade"
	"github.com/loft-sh/log"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const LoftUrl = "LOFT_URL"

// LoginCmd holds the login cmd flags
type LoginCmd struct {
	*flags.GlobalFlags
	Log         log.Logger
	AccessKey   string
	Insecure    bool
	DockerLogin bool
}

// NewLoginCmd creates a new open command
func NewLoginCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &LoginCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	description := product.ReplaceWithHeader("login", `
Login into loft

Example:
loft login https://my-loft.com
loft login https://my-loft.com --access-key myaccesskey
########################################################
	`)
	if upgrade.IsPlugin == "true" {
		description = `
########################################################
#################### devspace login ####################
########################################################
Login into loft

Example:
devspace login https://my-loft.com
devspace login https://my-loft.com --access-key myaccesskey
########################################################
	`
	}

	loginCmd := &cobra.Command{
		Use:   product.Replace("login [LOFT_HOST]"),
		Short: product.Replace("Login to a loft instance"),
		Long:  description,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Check for newer version
			upgrade.PrintNewerVersionWarning()

			return cmd.RunLogin(cobraCmd.Context(), args)
		},
	}

	loginCmd.Flags().StringVar(&cmd.AccessKey, "access-key", "", "The access key to use")
	loginCmd.Flags().BoolVar(&cmd.Insecure, "insecure", true, product.Replace("Allow login into an insecure Loft instance"))
	loginCmd.Flags().BoolVar(&cmd.DockerLogin, "docker-login", true, "If true, will log into the docker image registries the user has image pull secrets for")
	return loginCmd
}

// RunLogin executes the functionality "loft login"
func (cmd *LoginCmd) RunLogin(ctx context.Context, args []string) error {
	loader, err := client.NewClientFromPath(cmd.Config)
	if err != nil {
		return err
	}

	var url string
	// Print login information
	if len(args) == 0 {
		url = os.Getenv(LoftUrl)
		if url == "" {
			config := loader.Config()
			insecureFlag := ""
			if config.Insecure {
				insecureFlag = "--insecure"
			}

			err := cmd.printLoginDetails(ctx, loader, config)
			if err != nil {
				cmd.Log.Fatalf("%s\n\nYou may need to log in again via: %s login %s %s\n", err.Error(), os.Args[0], config.Host, insecureFlag)
			}

			domain := config.Host
			if domain == "" {
				domain = "my-loft-domain.com"
			}

			cmd.Log.WriteString(logrus.InfoLevel, fmt.Sprintf("\nTo log in as a different user, run: %s login %s %s\n\n", os.Args[0], domain, insecureFlag))

			return nil
		}
	} else {
		url = args[0]
	}

	if !strings.HasPrefix(url, "http") {
		url = "https://" + url
	}

	// log into loft
	url = strings.TrimSuffix(url, "/")
	if cmd.AccessKey != "" {
		err = loader.LoginWithAccessKey(url, cmd.AccessKey, cmd.Insecure)
	} else {
		err = loader.Login(url, cmd.Insecure, cmd.Log)
	}
	if err != nil {
		return err
	}
	cmd.Log.Donef(product.Replace("Successfully logged into Loft instance %s"), ansi.Color(url, "white+b"))

	// skip log into docker registries?
	if !cmd.DockerLogin {
		return nil
	}

	err = dockerLogin(ctx, loader, cmd.Log)
	if err != nil {
		return err
	}

	return nil
}

func (cmd *LoginCmd) printLoginDetails(ctx context.Context, loader client.Client, config *client.Config) error {
	if config.Host == "" {
		cmd.Log.Info("Not logged in")
		return nil
	}

	managementClient, err := loader.Management()
	if err != nil {
		return err
	}

	userName, teamName, err := helper.GetCurrentUser(ctx, managementClient)
	if err != nil {
		return err
	}

	if userName != nil {
		cmd.Log.Infof("Logged into %s as user: %s", config.Host, clihelper.DisplayName(&userName.EntityInfo))
	} else {
		cmd.Log.Infof("Logged into %s as team: %s", config.Host, clihelper.DisplayName(teamName))
	}
	return nil
}

func dockerLogin(ctx context.Context, loader client.Client, log log.Logger) error {
	managementClient, err := loader.Management()
	if err != nil {
		return err
	}

	// get user name
	userName, teamName, err := helper.GetCurrentUser(ctx, managementClient)
	if err != nil {
		return err
	}

	// collect image pull secrets from team or user
	dockerConfigs := []*configfile.ConfigFile{}
	if userName != nil {
		// get image pull secrets from user
		user, err := managementClient.Loft().ManagementV1().Users().Get(ctx, userName.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		dockerConfigs = append(dockerConfigs, collectImagePullSecrets(ctx, managementClient, user.Spec.ImagePullSecrets, log)...)

		// get image pull secrets from teams
		for _, teamName := range user.Status.Teams {
			team, err := managementClient.Loft().ManagementV1().Teams().Get(ctx, teamName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			dockerConfigs = append(dockerConfigs, collectImagePullSecrets(ctx, managementClient, team.Spec.ImagePullSecrets, log)...)
		}
	} else if teamName != nil {
		// get image pull secrets from team
		team, err := managementClient.Loft().ManagementV1().Teams().Get(ctx, teamName.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		dockerConfigs = append(dockerConfigs, collectImagePullSecrets(ctx, managementClient, team.Spec.ImagePullSecrets, log)...)
	}

	// store docker configs
	if len(dockerConfigs) > 0 {
		dockerConfig, err := docker.NewDockerConfig()
		if err != nil {
			return err
		}

		// log into registries locally
		for _, config := range dockerConfigs {
			for registry, authConfig := range config.AuthConfigs {
				err = dockerConfig.Store(registry, authConfig)
				if err != nil {
					return err
				}

				if registry == "https://index.docker.io/v1/" {
					registry = "docker hub"
				}

				log.Donef("Successfully logged into docker registry '%s'", registry)
			}
		}

		err = dockerConfig.Save()
		if err != nil {
			return errors.Wrap(err, "save docker config")
		}
	}

	return nil
}

func collectImagePullSecrets(ctx context.Context, managementClient kube.Interface, imagePullSecrets []*storagev1.KindSecretRef, log log.Logger) []*configfile.ConfigFile {
	retConfigFiles := []*configfile.ConfigFile{}
	for _, imagePullSecret := range imagePullSecrets {
		// unknown image pull secret type?
		if imagePullSecret.Kind != "SharedSecret" || (imagePullSecret.APIGroup != storagev1.SchemeGroupVersion.Group && imagePullSecret.APIGroup != managementv1.SchemeGroupVersion.Group) {
			continue
		} else if imagePullSecret.SecretName == "" || imagePullSecret.SecretNamespace == "" {
			continue
		}

		sharedSecret, err := managementClient.Loft().ManagementV1().SharedSecrets(imagePullSecret.SecretNamespace).Get(ctx, imagePullSecret.SecretName, metav1.GetOptions{})
		if err != nil {
			log.Warnf("Unable to retrieve image pull secret %s/%s: %v", imagePullSecret.SecretNamespace, imagePullSecret.SecretName, err)
			continue
		} else if len(sharedSecret.Spec.Data) == 0 {
			log.Warnf("Unable to retrieve image pull secret %s/%s: secret is empty", imagePullSecret.SecretNamespace, imagePullSecret.SecretName)
			continue
		} else if imagePullSecret.Key == "" && len(sharedSecret.Spec.Data) > 1 {
			log.Warnf("Unable to retrieve image pull secret %s/%s: secret has multiple keys, but none is specified for image pull secret", imagePullSecret.SecretNamespace, imagePullSecret.SecretName)
			continue
		}

		// determine shared secret key
		key := imagePullSecret.Key
		if key == "" {
			for k := range sharedSecret.Spec.Data {
				key = k
			}
		}

		configFile, err := dockerconfig.LoadFromReader(bytes.NewReader(sharedSecret.Spec.Data[key]))
		if err != nil {
			log.Warnf("Parsing image pull secret %s/%s.%s: %v", imagePullSecret.SecretNamespace, imagePullSecret.SecretName, key, err)
			continue
		}

		retConfigFiles = append(retConfigFiles, configFile)
	}

	return retConfigFiles
}
