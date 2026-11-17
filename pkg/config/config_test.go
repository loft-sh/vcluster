package config

import (
	"testing"

	"github.com/loft-sh/vcluster/config"
)

func TestVirtualClusterConfig_OnlyLeaderServes(t *testing.T) {
	tests := []struct {
		name       string
		configure  func(c *config.Config)
		standalone bool
		want       bool
	}{
		{name: "embedded database by default", configure: func(*config.Config) {}, want: true},
		{name: "embedded database explicitly", configure: func(c *config.Config) { c.ControlPlane.BackingStore.Database.Embedded.Enabled = true }, want: true},
		{name: "embedded database on standalone", configure: func(*config.Config) {}, standalone: true, want: false},
		{name: "external database", configure: func(c *config.Config) { c.ControlPlane.BackingStore.Database.External.Enabled = true }, want: false},
		{name: "embedded etcd", configure: func(c *config.Config) { c.ControlPlane.BackingStore.Etcd.Embedded.Enabled = true }, want: false},
		{name: "external etcd", configure: func(c *config.Config) { c.ControlPlane.BackingStore.Etcd.External.Enabled = true }, want: false},
		{name: "deployed etcd", configure: func(c *config.Config) { c.ControlPlane.BackingStore.Etcd.Deploy.Enabled = true }, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vConfig := VirtualClusterConfig{}
			tt.configure(&vConfig.Config)
			vConfig.ControlPlane.Standalone.Enabled = tt.standalone

			if got := vConfig.OnlyLeaderServes(); got != tt.want {
				t.Fatalf("OnlyLeaderServes() = %t, want %t", got, tt.want)
			}
		})
	}
}
