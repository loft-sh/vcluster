package pro

import (
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/reset"
	loftctlflags "github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

func NewResetCmd(loftctlGlobalFlags *loftctlflags.GlobalFlags) *cobra.Command {
	description := `########################################################
################## vcluster pro reset ##################
########################################################
	`
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset configuration",
		Long:  description,
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(NewPasswordCmd(loftctlGlobalFlags))

	return cmd
}

func NewPasswordCmd(globalFlags *loftctlflags.GlobalFlags) *cobra.Command {
	cmd := &reset.PasswordCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	description := `########################################################
############## vcluster pro reset password #############
########################################################
Resets the password of a user.

Example:
vcluster pro reset password
vcluster pro reset password --user admin
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

	return c
}
