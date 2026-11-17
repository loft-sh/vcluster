package find

import (
	"errors"
	"testing"
)

func TestParseSystemdActiveStatus(t *testing.T) {
	tests := []struct {
		name string
		out  []byte
		err  error
		want Status
	}{
		{
			name: "active",
			out:  []byte("active\n"),
			err:  nil,
			want: StatusRunning,
		},
		{
			name: "active without trailing newline",
			out:  []byte("active"),
			err:  nil,
			want: StatusRunning,
		},
		{
			name: "inactive",
			out:  []byte("inactive\n"),
			err:  nil,
			want: StatusUnknown,
		},
		{
			name: "command error",
			out:  []byte("inactive\n"),
			err:  errors.New("exit status 3"),
			want: StatusUnknown,
		},
		{
			name: "empty output",
			out:  nil,
			err:  nil,
			want: StatusUnknown,
		},
		{
			name: "command error with active output",
			out:  []byte("active\n"),
			err:  errors.New("exit status 1"),
			want: StatusUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSystemdActiveStatus(tt.out, tt.err)
			if got != tt.want {
				t.Fatalf("parseSystemdActiveStatus(%q, %v) = %v, want %v", tt.out, tt.err, got, tt.want)
			}
		})
	}
}
