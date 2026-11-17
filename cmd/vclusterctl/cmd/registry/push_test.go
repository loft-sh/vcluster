package registry

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/loft-sh/log"
)

func TestPushHelmChartPassesDashPrefixedFileAsPositional(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake helm is a shell script")
	}

	// fake helm records its arguments, one per line
	binDir := t.TempDir()
	argsFile := filepath.Join(binDir, "args")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + argsFile + "\n"
	err := os.WriteFile(filepath.Join(binDir, "helm"), []byte(script), 0o755)
	if err != nil {
		t.Fatalf("write fake helm: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	o := &PushOptions{
		Log:                 log.Discard,
		HelmChartRepository: "charts",
	}
	err = o.pushHelmChart(context.Background(), "-evil-0.1.0.tgz", 5000)
	if err != nil {
		t.Fatalf("pushHelmChart() error = %v", err)
	}

	raw, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read recorded args: %v", err)
	}
	got := strings.Split(strings.TrimSpace(string(raw)), "\n")
	want := []string{"push", "--plain-http", "--", "-evil-0.1.0.tgz", "oci://127.0.0.1:5000/charts"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("helm args = %q, want %q", got, want)
	}
}
