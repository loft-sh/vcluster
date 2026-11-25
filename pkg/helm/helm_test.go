package helm

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/loft-sh/log"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// fakeHelm writes a stand-in for helm that reports version and records the
// arguments of every other call, one call per line.
func fakeHelm(t *testing.T, version string) (helmPath, callsPath string) {
	t.Helper()
	dir := t.TempDir()
	helmPath = filepath.Join(dir, "helm")
	callsPath = filepath.Join(dir, "calls")
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = version ]; then printf '%s' '" + version + "'; echo version >> '" + callsPath + "'; exit 0; fi\n" +
		"printf '%s\\n' \"$*\" >> '" + callsPath + "'\n"
	if err := os.WriteFile(helmPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return helmPath, callsPath
}

func readCalls(t *testing.T, path string) [][]string {
	t.Helper()
	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var calls [][]string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		calls = append(calls, strings.Fields(line))
	}
	return calls
}

func fakeConfig() *clientcmdapi.Config {
	config := clientcmdapi.NewConfig()
	config.Clusters["target"] = &clientcmdapi.Cluster{Server: "https://127.0.0.1:1"}
	config.Contexts["target"] = &clientcmdapi.Context{Cluster: "target"}
	config.CurrentContext = "target"
	return config
}

// TestHelmVersionFlags checks that helm 4 keeps client-side apply and gets the
// renamed --force and --atomic flags, while helm 3 gets none of the flags it
// does not know.
func TestHelmVersionFlags(t *testing.T) {
	tests := []struct {
		version     string
		wantV4Flags bool
	}{
		{version: "v3.22.1"},
		{version: "v3.14.4"},
		{version: "v4.3.2", wantV4Flags: true},
	}

	for _, test := range tests {
		t.Run(test.version, func(t *testing.T) {
			helmPath, callsPath := fakeHelm(t, test.version)
			c := NewClient(fakeConfig(), log.Discard, helmPath)
			ctx := context.Background()
			options := UpgradeOptions{Chart: "vcluster", Repo: "https://charts.loft.sh", Force: true, Atomic: true}

			if err := c.Install(ctx, "release", "ns", options); err != nil {
				t.Fatal(err)
			}
			if err := c.Upgrade(ctx, "release", "ns", options); err != nil {
				t.Fatal(err)
			}
			if err := c.Rollback(ctx, "release", "ns"); err != nil {
				t.Fatal(err)
			}

			calls := readCalls(t, callsPath)
			var versionCalls int
			var helmCalls [][]string
			for _, call := range calls {
				if len(call) == 1 && call[0] == "version" {
					versionCalls++
					continue
				}
				helmCalls = append(helmCalls, call)
			}
			if versionCalls != 1 {
				t.Errorf("helm version was read %d times, want once", versionCalls)
			}
			if len(helmCalls) != 3 {
				t.Fatalf("got %d helm calls, want install, upgrade and rollback: %q", len(helmCalls), helmCalls)
			}

			for _, call := range helmCalls {
				if got := slices.Contains(call, clientSideApply); got != test.wantV4Flags {
					t.Errorf("%s call has %s = %t, want %t: %q", call[0], clientSideApply, got, test.wantV4Flags, call)
				}
			}
			for _, call := range helmCalls[:2] {
				wantForce, wantAtomic := "--force", "--atomic"
				if test.wantV4Flags {
					wantForce, wantAtomic = "--force-replace", "--rollback-on-failure"
				}
				if !slices.Contains(call, wantForce) || !slices.Contains(call, wantAtomic) {
					t.Errorf("%s call = %q, want %s and %s", call[0], call, wantForce, wantAtomic)
				}
				if !test.wantV4Flags && (slices.Contains(call, "--force-replace") || slices.Contains(call, "--rollback-on-failure")) {
					t.Errorf("%s call = %q, want no helm 4 flag names for helm 3", call[0], call)
				}
			}
		})
	}
}

// TestHelmVersionUnreadable checks that a helm whose version cannot be read is
// treated as helm 3, so no helm-4-only flag is passed.
func TestHelmVersionUnreadable(t *testing.T) {
	dir := t.TempDir()
	helmPath := filepath.Join(dir, "helm")
	callsPath := filepath.Join(dir, "calls")
	script := "#!/bin/sh\nif [ \"$1\" = version ]; then exit 1; fi\nprintf '%s\\n' \"$*\" >> '" + callsPath + "'\n"
	if err := os.WriteFile(helmPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	c := NewClient(fakeConfig(), log.Discard, helmPath)
	if err := c.Upgrade(context.Background(), "release", "ns", UpgradeOptions{Chart: "vcluster", Repo: "https://charts.loft.sh"}); err != nil {
		t.Fatal(err)
	}
	for _, call := range readCalls(t, callsPath) {
		if slices.Contains(call, clientSideApply) {
			t.Errorf("call = %q, want no %s when the helm version is unknown", call, clientSideApply)
		}
	}
}

// TestClientSideApplyArgs checks the arguments given to helm calls that do not
// go through Client: client-side apply for helm 4 only.
func TestClientSideApplyArgs(t *testing.T) {
	tests := []struct {
		version string
		want    []string
	}{
		{version: "v3.22.1"},
		{version: "v3.14.4"},
		{version: "v4.3.2", want: []string{clientSideApply}},
	}

	for _, test := range tests {
		t.Run(test.version, func(t *testing.T) {
			helmPath, _ := fakeHelm(t, test.version)
			if got := ClientSideApplyArgs(context.Background(), helmPath, log.Discard); !slices.Equal(got, test.want) {
				t.Errorf("ClientSideApplyArgs() = %q, want %q", got, test.want)
			}
		})
	}

	t.Run("unreadable", func(t *testing.T) {
		if got := ClientSideApplyArgs(context.Background(), filepath.Join(t.TempDir(), "helm"), log.Discard); got != nil {
			t.Errorf("ClientSideApplyArgs() = %q, want none", got)
		}
	})
}

// TestHelmVersionRetriedAfterFailure checks that a failed version read is not
// cached: the next call reads again and, once it succeeds, the result is kept.
func TestHelmVersionRetriedAfterFailure(t *testing.T) {
	dir := t.TempDir()
	helmPath := filepath.Join(dir, "helm")
	callsPath := filepath.Join(dir, "calls")
	failedPath := filepath.Join(dir, "failed")
	// The first version call fails, every later one reports helm 4.
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = version ]; then echo version >> '" + callsPath + "'\n" +
		"  if [ ! -e '" + failedPath + "' ]; then touch '" + failedPath + "'; exit 1; fi\n" +
		"  printf v4.3.2; exit 0; fi\n" +
		"printf '%s\\n' \"$*\" >> '" + callsPath + "'\n"
	if err := os.WriteFile(helmPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	c := NewClient(fakeConfig(), log.Discard, helmPath)
	options := UpgradeOptions{Chart: "vcluster", Repo: "https://charts.loft.sh"}
	for range 3 {
		if err := c.Upgrade(context.Background(), "release", "ns", options); err != nil {
			t.Fatal(err)
		}
	}

	var versionCalls int
	var upgrades [][]string
	for _, call := range readCalls(t, callsPath) {
		if len(call) == 1 && call[0] == "version" {
			versionCalls++
			continue
		}
		upgrades = append(upgrades, call)
	}
	if versionCalls != 2 {
		t.Errorf("helm version was read %d times, want 2 (the failed read, then one cached success)", versionCalls)
	}
	if len(upgrades) != 3 {
		t.Fatalf("got %d upgrade calls, want 3: %q", len(upgrades), upgrades)
	}
	for i, want := range []bool{false, true, true} {
		if got := slices.Contains(upgrades[i], clientSideApply); got != want {
			t.Errorf("upgrade %d has %s = %t, want %t: %q", i+1, clientSideApply, got, want, upgrades[i])
		}
	}
}
