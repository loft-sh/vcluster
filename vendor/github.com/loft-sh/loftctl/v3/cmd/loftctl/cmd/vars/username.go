package vars

import (
	"os"

	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/client"
	"github.com/loft-sh/loftctl/v3/pkg/client/helper"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	ErrUserSetNoLogin = errors.New("not logged in, but predefined var LOFT_USERNAME is used")
)

type usernameCmd struct {
	*flags.GlobalFlags
}

func newUsernameCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &usernameCmd{
		GlobalFlags: globalFlags,
	}

	return &cobra.Command{
		Use:   "username",
		Short: product.Replace("Prints the current loft username"),
		Args:  cobra.NoArgs,
		RunE:  cmd.Run,
	}
}

// Run executes the command logic
func (cmd *usernameCmd) Run(cobraCmd *cobra.Command, args []string) error {
	baseClient, err := client.NewClientFromPath(cmd.Config)
	if err != nil {
		return ErrUserSetNoLogin
	}

	client, err := baseClient.Management()
	if err != nil {
		return err
	}

	ctx := cobraCmd.Context()

	userName, teamName, err := helper.GetCurrentUser(ctx, client)
	if err != nil {
		return err
	} else if teamName != nil {
		return errors.New("logged in with a team and not a user")
	}

	_, err = os.Stdout.Write([]byte(userName.Username))
	return err
}
