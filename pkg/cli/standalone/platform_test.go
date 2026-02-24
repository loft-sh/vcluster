package standalone

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRenderSystemdPlatformConfFile(t *testing.T) {
	options := &AddToPlatformOptions{
		AccessKey:    "abcd",
		Host:         "test.vcluster.platform",
		InstanceName: "test-instance",
		ProjectName:  "test-project",
	}

	want := `
[Service]
Environment=LOFT_PLATFORM_ACCESS_KEY="abcd"
Environment=LOFT_PLATFORM_HOST="test.vcluster.platform"
Environment=LOFT_PLATFORM_INSECURE="false"
Environment=LOFT_PLATFORM_INSTANCE_NAME="test-instance"
Environment=LOFT_PLATFORM_PROJECT_NAME="test-project"
`

	got, err := renderSystemdPlatformConfFile(options)
	if err != nil {
		t.Errorf("renderSystemdServiceFile() error = %v", err)
		return
	}

	gotString := string(got)
	if gotString != want {
		t.Errorf("renderSystemdServiceFile() diff(want, got) = %s", cmp.Diff(want, gotString))
	}
}
