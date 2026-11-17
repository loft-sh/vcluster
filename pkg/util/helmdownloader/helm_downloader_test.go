package helmdownloader

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/constants"
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
		installPath := filepath.Join(home, constants.VClusterFolder, "bin", "helm")
		assert.Equal(t, helmBinaryPath, installPath)
	}
}
