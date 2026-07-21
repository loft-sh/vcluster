package clihelper

import (
	"testing"
)

func TestCheckHelmVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		wantErr bool
	}{
		{
			name:    "version with v prefix - valid",
			version: "v3.12.3",
			wantErr: false,
		},
		{
			name:    "version without v prefix - valid",
			version: "3.12.3",
			wantErr: false,
		},
		{
			name:    "version below minimum - with v prefix",
			version: "v2.0.0",
			wantErr: true,
		},
		{
			name:    "version below minimum - without v prefix",
			version: "2.0.0",
			wantErr: true,
		},
		{
			name:    "equal to minimum version - with v prefix",
			version: MinHelmVersion,
			wantErr: false,
		},
		{
			name:    "equal to minimum version - without v prefix",
			version: MinHelmVersion[1:], // remove 'v' prefix
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckHelmVersion(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckHelmVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
