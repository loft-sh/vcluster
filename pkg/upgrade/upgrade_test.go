package upgrade

import (
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"os"
	"strings"
	"testing"

	"github.com/rhysd/go-github-selfupdate/selfupdate"

	"gotest.tools/assert"
)

func TestSetVersion(t *testing.T) {
	SetVersion("sasd0.0.1hello")
	assert.Equal(t, "0.0.1hello", GetVersion(), "Wrong version set")
}

func TestEraseVersionPrefix(t *testing.T) {
	prefixless, err := eraseVersionPrefix("sasd0.0.1hello")
	if err != nil {
		t.Fatalf("Error erasing Version: %v", err)
	}
	assert.Equal(t, "0.0.1hello", prefixless, "Wrong version set")

	_, err = eraseVersionPrefix(".0.1hello")
	assert.Equal(t, true, err != nil, "No error returned with invalid string")
}

func TestUpgrade(t *testing.T) {
	t.Skip("Skip because of some API-limit")
	//Create TmpFolder
	dir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatalf("Error creating temporary directory: %v", err)
	}

	wdBackup, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}
	err = os.Chdir(dir)
	if err != nil {
		t.Fatalf("Error changing working directory: %v", err)
	}

	// Cleanup temp folder
	defer func() {
		err = os.Chdir(wdBackup)
		if err != nil {
			t.Fatalf("Error changing dir back: %v", err)
		}
		err = os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("Error removing dir: %v", err)
		}
	}()

	logFile, err := os.CreateTemp(dir, "log")
	if err != nil {
		t.Fatalf("Error creating temporary log file: %v", err)
	}
	*(os.Stderr) = *logFile

	latest, found, err := selfupdate.DetectLatest(githubSlug)
	if err != nil {
		t.Fatalf("Error searching for version: %v", err)
	} else if !found {
		t.Fatalf("No version found.")
	}

	versionBackup := version
	version = latest.Version.String()
	defer func() { version = versionBackup }()

	//Newest version already reached
	err = Upgrade("", log.GetInstance())
	assert.Equal(t, false, err != nil, "Upgrade returned error if newest version already reached")
	err = logFile.Close()
	if err != nil {
		t.Fatalf("Error closing temporary log file: %v", err)
	}
	logs, err := os.ReadFile(logFile.Name())
	if err != nil {
		t.Fatalf("Error reading temporary log file: %v", err)
	}
	assert.Equal(t, true, strings.Contains(string(logs), "Current binary is the latest version:  "+version))

	//Invalid githubSlug causes search to return an error
	githubSlugBackup := githubSlug
	githubSlug = ""
	defer func() { githubSlug = githubSlugBackup }()
	err = Upgrade("", log.GetInstance())
	assert.Equal(t, true, err != nil, "No error returned if DetectLatest returns one.")
}
