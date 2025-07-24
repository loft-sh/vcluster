package ghcr

import (
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
)

func Test_parsePackageInfo(t *testing.T) {
	type args struct {
		ref name.Reference
	}
	tests := []struct {
		name    string
		args    args
		org     string
		pkg     string
		tag     string
		wantErr bool
	}{
		{
			name: "no user or org",
			args: args{
				ref: name.MustParseReference("ghcr.io/russ-snapshots:1"),
			},
			wantErr: true,
		},
		{
			name: "root package",
			args: args{
				ref: name.MustParseReference("ghcr.io/lizardruss/russ-snapshots:1"),
			},
			org: "lizardruss",
			pkg: "russ-snapshots",
			tag: "1",
		},
		{
			name: "root package no tag",
			args: args{
				ref: name.MustParseReference("ghcr.io/lizardruss/russ-snapshots"),
			},
			org: "lizardruss",
			pkg: "russ-snapshots",
			tag: "latest",
		},
		{
			name: "nested package",
			args: args{
				ref: name.MustParseReference("ghcr.io/lizardruss/my-vcluster/snapshots:1"),
			},
			org: "lizardruss",
			pkg: "my-vcluster/snapshots",
			tag: "1",
		},
		{
			name: "nested package no tag",
			args: args{
				ref: name.MustParseReference("ghcr.io/lizardruss/my-vcluster/snapshots"),
			},
			org: "lizardruss",
			pkg: "my-vcluster/snapshots",
			tag: "latest",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			org, pkg, tag, err := parsePackageInfo(tt.args.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePackageInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if org != tt.org {
				t.Errorf("parsePackageInfo() org = %v, want %v", org, tt.org)
			}
			if pkg != tt.pkg {
				t.Errorf("parsePackageInfo() pkg = %v, want %v", pkg, tt.pkg)
			}
			if tag != tt.tag {
				t.Errorf("parsePackageInfo() tag = %v, want %v", tag, tt.tag)
			}
		})
	}
}

func TestIsGHCRContainerRegistry(t *testing.T) {
	type args struct {
		ref name.Reference
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "ghcr.io registry",
			args: args{
				ref: name.MustParseReference("ghcr.io/lizardruss/russ-snapshots:1"),
			},
			want: true,
		},
		{
			name: "non ghcr.io registry",
			args: args{
				ref: name.MustParseReference("registry.local/lizardruss/russ-snapshots:1"),
			},
			want: false,
		},
		{
			name: "non ghcr.io registry containing ghcr.io",
			args: args{
				ref: name.MustParseReference("ghcr.io.local/lizardruss/russ-snapshots:1"),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsGHCRContainerRegistry(tt.args.ref); got != tt.want {
				t.Errorf("IsGHCRContainerRegistry() = %v, want %v", got, tt.want)
			}
		})
	}
}
