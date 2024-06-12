package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	dockerconfig "github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/use"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/docker"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/platform/clihelper"
	"github.com/loft-sh/vcluster/pkg/platform/kube"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const PlatformURL = "VCLUSTER_PLATFORM_URL"

type LoginCmd struct {
	*flags.GlobalFlags

	Log log.Logger

	Driver      string
	AccessKey   string
	Insecure    bool
	DockerLogin bool
}

func NewLoginCmd(globalFlags *flags.GlobalFlags) (*cobra.Command, error) {
	cmd := LoginCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	description := `########################################################
#################### vcluster login ####################
########################################################
Login into vCluster platform

Example:
vcluster login https://my-vcluster-platform.com
vcluster login https://my-vcluster-platform.com --access-key myaccesskey
########################################################
	`

	loginCmd := &cobra.Command{
		Use:   "login [VCLUSTER_PLATFORM_HOST]",
		Short: "Login to a vCluster platform instance",
		Long:  description,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Check for newer version
			upgrade.PrintNewerVersionWarning()

			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	loginCmd.Flags().StringVar(&cmd.Driver, "use-driver", "", "Switch vCluster driver between platform and helm")
	loginCmd.Flags().StringVar(&cmd.AccessKey, "access-key", "", "The access key to use")
	loginCmd.Flags().BoolVar(&cmd.Insecure, "insecure", true, product.Replace("Allow login into an insecure Loft instance"))
	loginCmd.Flags().BoolVar(&cmd.DockerLogin, "docker-login", true, "If true, will log into the docker image registries the user has image pull secrets for")

	return loginCmd, nil
}

func (cmd *LoginCmd) Run(ctx context.Context, args []string) error {
	cfg := cmd.LoadedConfig(cmd.Log)

	var url string
	// Print login information
	if len(args) == 0 {
		url = os.Getenv(PlatformURL)
		if url == "" {
			insecureFlag := ""
			if cfg.Platform.Insecure {
				insecureFlag = "--insecure"
			}

			err := cmd.printLoginDetails(ctx)
			if err != nil {
				cmd.Log.Fatalf("%s\n\nYou may need to log in again via: %s login %s %s\n", err.Error(), os.Args[0], cfg.Platform.Host, insecureFlag)
			}

			domain := cfg.Platform.Host
			if domain == "" {
				domain = "my-vcluster-platform.com"
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

	// log into platform
	loginClient := platform.NewLoginClientFromConfig(cfg)
	url = strings.TrimSuffix(url, "/")
	var err error
	if cmd.AccessKey != "" {
		err = loginClient.LoginWithAccessKey(url, cmd.AccessKey, cmd.Insecure)
	} else {
		err = loginClient.Login(url, cmd.Insecure, cmd.Log)
	}
	if err != nil {
		return err
	}
	cmd.Log.Donef(product.Replace("Successfully logged into Loft instance %s"), ansi.Color(url, "white+b"))

	// skip log into docker registries?
	if !cmd.DockerLogin {
		return nil
	}

	err = dockerLogin(ctx, cmd.LoadedConfig(cmd.Log), cmd.Log)
	if err != nil {
		return err
	}

	// should switch driver
	if cmd.Driver != "" {
		err := use.SwitchDriver(ctx, cfg, cmd.Driver, log.GetInstance())
		if err != nil {
			return fmt.Errorf("driver switch failed: %w", err)
		}
	}

	return nil
}

func (cmd *LoginCmd) printLoginDetails(ctx context.Context) error {
	cfg := cmd.LoadedConfig(cmd.Log)
	platformClient := platform.NewClientFromConfig(cfg)

	managementClient, err := platformClient.Management()
	if err != nil {
		return err
	}

	userName, teamName, err := platform.GetCurrentUser(ctx, managementClient)
	if err != nil {
		return err
	}

	if userName != nil {
		cmd.Log.Infof("Logged into %s as user: %s", platformClient.Config().Platform.Host, clihelper.DisplayName(&userName.EntityInfo))
	} else {
		cmd.Log.Infof("Logged into %s as team: %s", platformClient.Config().Platform.Host, clihelper.DisplayName(teamName))
	}
	return nil
}

func dockerLogin(ctx context.Context, config *config.CLI, log log.Logger) error {
	platformClient := platform.NewClientFromConfig(config)

	managementClient, err := platformClient.Management()
	if err != nil {
		return err
	}

	// get user name
	userName, teamName, err := platform.GetCurrentUser(ctx, managementClient)
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
