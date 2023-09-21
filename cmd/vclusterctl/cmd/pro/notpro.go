//go:build !pro
// +build !pro

package pro

import (
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/spf13/cobra"
)

func NewProCmd(*flags.GlobalFlags) (*cobra.Command, error) {
	return nil, nil
}
