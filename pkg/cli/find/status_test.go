package find

import (
	"testing"
)

func TestSystemdActiveStatus(t *testing.T) {
	tests := []struct {
		name   string
		active bool
		want   Status
	}{
		{
			name:   "active",
			active: true,
			want:   StatusRunning,
		},
		{
			name:   "not active",
			active: false,
			want:   StatusUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := systemdActiveStatus(tt.active)
			if got != tt.want {
				t.Fatalf("systemdActiveStatus(%v) = %v, want %v", tt.active, got, tt.want)
			}
		})
	}
}
