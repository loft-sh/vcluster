package clihelper

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/loft-sh/log"
	"github.com/loft-sh/log/survey"
	"k8s.io/client-go/kubernetes/fake"
)

// fakeHelmOnPath puts a stand-in for helm first on PATH. It reports version and
// returns the path of the file that records the arguments of every other call.
func fakeHelmOnPath(t *testing.T, version string) string {
	t.Helper()
	dir := t.TempDir()
	callsPath := filepath.Join(dir, "calls")
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = version ]; then printf '%s' '" + version + "'; exit 0; fi\n" +
		"printf '%s\\n' \"$*\" >> '" + callsPath + "'\n"
	if err := os.WriteFile(filepath.Join(dir, "helm"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return callsPath
}

// lastHelmCall returns the arguments of the only helm call recorded in callsPath.
func lastHelmCall(t *testing.T, callsPath string) []string {
	t.Helper()
	out, err := os.ReadFile(callsPath)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Fields(strings.TrimSpace(string(out)))
}

// TestUpgradeLoftApplyMethod checks that `vcluster platform start` keeps helm 4
// on client-side apply and passes helm 3 no flag it does not know.
func TestUpgradeLoftApplyMethod(t *testing.T) {
	tests := []struct {
		version         string
		wantClientSided bool
	}{
		{version: "v3.22.1"},
		{version: "v4.3.2", wantClientSided: true},
	}

	for _, test := range tests {
		t.Run(test.version, func(t *testing.T) {
			callsPath := fakeHelmOnPath(t, test.version)

			err := UpgradeLoft(context.Background(), fake.NewSimpleClientset(), "vcluster-platform", "", "ctx", "vcluster-platform", []string{"--set", "admin.password=x"}, log.Discard)
			if err != nil {
				t.Fatal(err)
			}

			args := lastHelmCall(t, callsPath)
			if len(args) == 0 || args[0] != "upgrade" {
				t.Fatalf("helm args = %q, want an upgrade", args)
			}
			if got := slices.Contains(args, "--server-side=false"); got != test.wantClientSided {
				t.Errorf("helm args = %q, --server-side=false present = %t, want %t", args, got, test.wantClientSided)
			}
		})
	}
}

// answeringLogger answers every question with answer and discards all output.
type answeringLogger struct {
	log.Logger
	answer string
}

func (l answeringLogger) Question(*survey.QuestionOptions) (string, error) {
	return l.answer, nil
}

// TestEnsureIngressControllerApplyMethod checks that the ingress-nginx install
// `vcluster platform start` offers keeps helm 4 on client-side apply and passes
// helm 3 no flag it does not know.
func TestEnsureIngressControllerApplyMethod(t *testing.T) {
	tests := []struct {
		version         string
		wantClientSided bool
	}{
		{version: "v3.22.1"},
		{version: "v4.3.2", wantClientSided: true},
	}

	for _, test := range tests {
		t.Run(test.version, func(t *testing.T) {
			callsPath := fakeHelmOnPath(t, test.version)

			logger := answeringLogger{Logger: log.Discard, answer: "Yes"}
			if err := EnsureIngressController(context.Background(), fake.NewSimpleClientset(), "ctx", logger); err != nil {
				t.Fatal(err)
			}

			args := lastHelmCall(t, callsPath)
			if len(args) < 2 || args[0] != "install" || args[1] != "ingress-nginx" {
				t.Fatalf("helm args = %q, want an ingress-nginx install", args)
			}
			if got := slices.Contains(args, "--server-side=false"); got != test.wantClientSided {
				t.Errorf("helm args = %q, --server-side=false present = %t, want %t", args, got, test.wantClientSided)
			}
		})
	}
}
