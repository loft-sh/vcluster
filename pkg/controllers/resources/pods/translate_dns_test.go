package pods

import (
	"net"
	"testing"

	"gotest.tools/v3/assert"
)

func TestResolveDNSOverride(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantErr   bool
		checkAddr func(t *testing.T, got string)
	}{
		{
			name:    "empty value returns empty without error",
			input:   "",
			wantErr: false,
			checkAddr: func(t *testing.T, got string) {
				assert.Equal(t, "", got)
			},
		},
		{
			name:    "valid IPv4 is returned as-is",
			input:   "10.96.0.10",
			wantErr: false,
			checkAddr: func(t *testing.T, got string) {
				assert.Equal(t, "10.96.0.10", got)
			},
		},
		{
			name:    "valid IPv6 is returned as-is",
			input:   "fd00::1",
			wantErr: false,
			checkAddr: func(t *testing.T, got string) {
				assert.Equal(t, "fd00::1", got)
			},
		},
		{
			name:    "resolvable hostname returns a valid IP",
			input:   "localhost",
			wantErr: false,
			checkAddr: func(t *testing.T, got string) {
				if net.ParseIP(got) == nil {
					t.Fatalf("expected a valid IP for hostname 'localhost', got %q", got)
				}
			},
		},
		{
			name:      "unresolvable hostname returns error",
			input:     "this.hostname.does.not.exist.invalid",
			wantErr:   true,
			checkAddr: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveDNSOverride(tc.input)
			if tc.wantErr {
				assert.Assert(t, err != nil, "expected an error for input %q", tc.input)
				return
			}
			assert.NilError(t, err)
			tc.checkAddr(t, got)
		})
	}
}
