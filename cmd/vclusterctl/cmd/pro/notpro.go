//go:build !pro
// +build !pro

package pro

import (
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/spf13/cobra"
)

func NewProCmd(*flags.GlobalFlags) (*cobra.Command, error) {
	return nil, constants.ErrOnlyInPro
}
