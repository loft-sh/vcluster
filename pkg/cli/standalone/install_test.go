package standalone

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRenderSystemdServiceFile(t *testing.T) {
	ic := &installContext{
		name:    "vcluster",
		env:     map[string]string{"TEST_ENV": "test_value"},
		confDir: "/etc/vcluster",
		dataDir: "/var/lib/vcluster",
	}

	want := `
[Unit]
Description=vcluster
Documentation=https://vcluster.com
Wants=network-online.target
After=network-online.target dbus.service

[Install]
WantedBy=multi-user.target

[Service]
Type=notify
Environment=TEST_ENV="test_value"
Environment=VCLUSTER_KUBERNETES_BUNDLE=""
Environment=VCLUSTER_NAME="vcluster"
EnvironmentFile=-/etc/default/%N
EnvironmentFile=-/etc/sysconfig/%N
KillMode=process
Delegate=yes
User=root
# Having non-zero Limit*s causes performance problems due to accounting overhead
# in the kernel. We recommend using cgroups to do container-local accounting.
LimitNOFILE=1048576
LimitNPROC=infinity
LimitCORE=infinity
TasksMax=infinity
TimeoutStartSec=0
Restart=always
RestartSec=5s
ExecStart=/var/lib/vcluster/bin/vcluster start --config /etc/vcluster/vcluster.yaml`

	got, err := renderSystemdServiceFile(ic)
	if err != nil {
		t.Errorf("renderSystemdServiceFile() error = %v", err)
		return
	}

	gotString := string(got)
	if gotString != want {
		t.Errorf("renderSystemdServiceFile() diff(want, got) = %s", cmp.Diff(want, gotString))
	}
}
