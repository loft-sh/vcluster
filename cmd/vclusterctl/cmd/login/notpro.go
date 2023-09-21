//go:build !pro
// +build !pro

package login

import (
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/spf13/cobra"
)

func NewLoginCmd(*flags.GlobalFlags) (*cobra.Command, error) {
	return nil, nil
}
