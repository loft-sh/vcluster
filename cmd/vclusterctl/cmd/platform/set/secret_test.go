package set

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// Shared secrets were removed in vCluster Platform 5.0, so without a project there is nowhere
// to write. Run must fail before it tries to reach the platform.
func TestRunRequiresProject(t *testing.T) {
	cobraCmd := &cobra.Command{}
	cobraCmd.SetContext(t.Context())

	err := (&SecretCmd{}).Run(cobraCmd, []string{"my-secret.key", "value"})
	if !errors.Is(err, ErrProjectRequired) {
		t.Fatalf("expected ErrProjectRequired, got %v", err)
	}
}

// --namespace is a root persistent flag, so it still parses after the local flag was removed.
// It only ever selected shared secrets, so Run must warn that it is ignored instead of
// silently accepting it.
func TestRunWarnsOnIgnoredNamespace(t *testing.T) {
	var out bytes.Buffer
	cmd := &SecretCmd{log: log.NewStreamLogger(&out, &out, logrus.InfoLevel)}

	root := &cobra.Command{Use: "root", SilenceErrors: true, SilenceUsage: true}
	root.PersistentFlags().StringP("namespace", "n", "", "")
	root.AddCommand(&cobra.Command{
		Use: "secret",
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd, args)
		},
	})
	root.SetArgs([]string{"secret", "-n", "vcluster-platform", "my-secret.key", "value"})

	err := root.ExecuteContext(t.Context())
	if !errors.Is(err, ErrProjectRequired) {
		t.Fatalf("expected ErrProjectRequired, got %v", err)
	}
	if !strings.Contains(out.String(), "--namespace is ignored") {
		t.Fatalf("expected a warning that --namespace is ignored, got %q", out.String())
	}
}
