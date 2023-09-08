package cmd

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/loft-sh/vcluster/pkg/util/cliconfig"
	"github.com/mitchellh/go-homedir"
	"gotest.tools/assert"
)

func TestGetHelmBinaryPath(t *testing.T) {
	_, err := exec.LookPath("helm")
	if err != nil {
		helmBinaryPath, err := GetHelmBinaryPath(context.Background(), log.GetInstance())
		assert.NilError(t, err)
		home, err := homedir.Dir()
		assert.NilError(t, err)
		installPath := filepath.Join(home, cliconfig.VclusterFolder, "bin", "helm")
		assert.Equal(t, helmBinaryPath, installPath)
	}
}
