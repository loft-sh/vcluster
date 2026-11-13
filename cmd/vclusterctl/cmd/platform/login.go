package platform

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/loft-sh/api/v5/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/use"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/platform/clihelper"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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

func NewLoginCmd(gf *flags.GlobalFlags) LoginCmd {
	return LoginCmd{
		GlobalFlags: gf,
		Log:         log.GetInstance(),
	}
}

func NewCobraLoginCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := NewLoginCmd(globalFlags)

	description := `########################################################
############### vcluster platform login ################
########################################################
Login into vCluster platform

Example:
vcluster platform login https://my-vcluster-platform.com
vcluster platform login https://my-vcluster-platform.com --access-key myaccesskey
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

	loginCmd.Flags().StringVar(&cmd.Driver, "use-driver", "", "Switch vCluster driver between platform, helm, and docker")
	loginCmd.Flags().StringVar(&cmd.AccessKey, "access-key", "", "The access key to use")
	loginCmd.Flags().BoolVar(&cmd.Insecure, "insecure", false, product.Replace("Allow login into an insecure Loft instance"))
	loginCmd.Flags().BoolVar(&cmd.DockerLogin, "docker-login", true, "If true, will log into the docker image registries the user has image pull secrets for")
	_ = loginCmd.Flags().MarkDeprecated("docker-login", "it only works with shared secrets, which were removed in vCluster CLI v0.38")

	return loginCmd
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

			domain := cfg.Platform.Host
			if domain == "" {
				domain = "my-vcluster-platform.com"
			}

			err := cmd.printLoginDetails(ctx)
			if err != nil {
				cmd.Log.Fatalf("%s\n\nYou need to log in again via: %s platform login %s %s\n", err.Error(), os.Args[0], domain, insecureFlag)
			}

			cmd.Log.WriteString(logrus.InfoLevel, fmt.Sprintf("\nTo log in as a different user, run: %s platform login %s %s\n\n", os.Args[0], domain, insecureFlag))

			return nil
		}
	} else {
		url = args[0]
	}

	if !strings.HasPrefix(url, "http") {
		url = "https://" + url
	}

	// log into platform
	var err error
	loginClient := platform.NewLoginClientFromConfig(cfg)
	url = strings.TrimSuffix(url, "/")
	if cmd.AccessKey != "" {
		err = loginClient.LoginWithAccessKey(url, cmd.AccessKey, cmd.Insecure)
	} else if cfg.Platform.AccessKey != "" && cfg.Platform.Host == url {
		// check if user was already logged in i.e. config contains the access key for the same host
		if loginErr := loginClient.LoginWithAccessKey(url, cfg.Platform.AccessKey, cmd.Insecure); loginErr != nil {
			if errors.Is(loginErr, platform.ErrInvalidAccessKey) {
				cmd.Log.Warnf("Invalid access key, attempting to login again...")
				err = loginClient.Login(url, cmd.Insecure, cmd.Log)
			} else {
				err = loginErr
			}
		}
	} else {
		err = loginClient.Login(url, cmd.Insecure, cmd.Log)
	}
	if err != nil {
		return err
	}
	cmd.Log.Donef(product.Replace("Successfully logged into Loft instance %s"), ansi.Color(url, "white+b"))

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
	// TODO: Refactor login to rely less on side effects from manipulating the cfg.Platform object in memory
	if cmd.AccessKey != "" {
		cfg.Platform.AccessKey = cmd.AccessKey
	}

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
