package cmd

import (
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/mitchellh/go-homedir"
	"gotest.tools/assert"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGetHelmBinaryPath(t *testing.T) {
	_, err := exec.LookPath("helm")
	if err != nil {
		helmBinaryPath, err := GetHelmBinaryPath(log.GetInstance())
		assert.NilError(t, err)
		home, err := homedir.Dir()
		assert.NilError(t, err)
		installPath := filepath.Join(home, DefaultHomeVClusterFolder, "bin", "helm")
		assert.Equal(t, helmBinaryPath, installPath)
	}
}
