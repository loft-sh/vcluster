package helm

import (
	"strings"
	"testing"
)

func TestRedactHelmArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "password as separate argument",
			args: []string{"registry", "login", "--username", "alice", "--password", "s3cret", "ghcr.io"},
			want: "registry login --username alice --password ***** ghcr.io",
		},
		{
			name: "password equals form",
			args: []string{"registry", "login", "--password=s3cret", "ghcr.io"},
			want: "registry login --password=***** ghcr.io",
		},
		{
			name: "password last argument has no value",
			args: []string{"registry", "login", "--password"},
			want: "registry login --password",
		},
		{
			name: "password-stdin is not a password value",
			args: []string{"registry", "login", "--password-stdin", "ghcr.io"},
			want: "registry login --password-stdin ghcr.io",
		},
		{
			name: "no password flag",
			args: []string{"pull", "oci://ghcr.io/org/chart"},
			want: "pull oci://ghcr.io/org/chart",
		},
		{
			name: "empty args",
			args: nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redactHelmArgs(tt.args)
			if got != tt.want {
				t.Fatalf("redactHelmArgs() = %q, want %q", got, tt.want)
			}
			if strings.Contains(got, "s3cret") {
				t.Fatalf("redactHelmArgs() leaked password: %q", got)
			}
		})
	}
}
