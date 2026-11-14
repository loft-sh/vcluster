package list

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/sirupsen/logrus"
)

// Without a project flag, list secrets used to list shared secrets, which were removed in
// vCluster CLI v0.38. It must fail before reaching the platform: an empty list with exit 0
// would pass in CI as a real answer.
func TestRunWithoutProjectFlagFails(t *testing.T) {
	c := newSecretsCmd(&flags.GlobalFlags{})
	cmd := &SecretsCmd{log: log.NewStreamLogger(&bytes.Buffer{}, &bytes.Buffer{}, logrus.InfoLevel)}

	if err := cmd.Run(c); !errors.Is(err, ErrSharedSecretsRemoved) {
		t.Fatalf("expected ErrSharedSecretsRemoved, got %v", err)
	}
}

// --all used to include shared secrets. It still lists project secrets, but must say that
// shared secrets are no longer part of its output. The warning comes before the platform
// client is created, so this test reaches it and then fails on the missing login.
func TestRunWithAllLogsSharedSecretsRemoval(t *testing.T) {
	var out bytes.Buffer
	c := newSecretsCmd(&flags.GlobalFlags{})
	cmd := &SecretsCmd{
		GlobalFlags: &flags.GlobalFlags{Config: filepath.Join(t.TempDir(), "config.json")},
		All:         true,
		log:         log.NewStreamLogger(&out, &out, logrus.InfoLevel),
	}

	_ = cmd.Run(c)
	if !strings.Contains(out.String(), "--all no longer includes shared secrets") {
		t.Fatalf("expected the --all shared secrets notice, got %q", out.String())
	}
}

// The project flags keep their meaning so existing scripts keep working.
func TestProjectFlagsStillParse(t *testing.T) {
	c := newSecretsCmd(&flags.GlobalFlags{})
	if err := c.ParseFlags([]string{"--all", "-a", "--all-projects", "-p", "a", "--project", "b"}); err != nil {
		t.Fatalf("project flags must still parse: %v", err)
	}
}
