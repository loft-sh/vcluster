package applier

import (
	"errors"
	"strings"
	"testing"

	"github.com/go-logr/logr/funcr"
)

func TestLogApplyOutput(t *testing.T) {
	tests := []struct {
		name     string
		stdout   string
		stderr   string
		applyErr error
		want     []string
		notWant  []string
	}{
		{
			name:   "successful apply treats stderr as routine output",
			stdout: "deployment.apps/vcluster configured\n",
			stderr: "Warning: resource is missing an annotation\n",
			want: []string{
				`"level"=0`,
				`"stdout"="deployment.apps/vcluster configured"`,
				`"stderr"="Warning: resource is missing an annotation"`,
			},
			notWant: []string{`"error"=`},
		},
		{
			name:     "failed apply reports stderr as an error",
			stdout:   "deployment.apps/vcluster configured\n",
			stderr:   "Warning: resource is missing an annotation\n",
			applyErr: errors.New("apply failed"),
			want: []string{
				`"stdout"="deployment.apps/vcluster configured"`,
				`"stderr"="Warning: resource is missing an annotation"`,
				`"error"="apply failed"`,
			},
		},
		{
			name:     "failed apply with empty stderr still logs the error",
			stdout:   "deployment.apps/vcluster configured\n",
			applyErr: errors.New("error from SetNamespace: namespace is required"),
			want: []string{
				`"stdout"="deployment.apps/vcluster configured"`,
				`"error"="error from SetNamespace: namespace is required"`,
				`"msg"="apply failed"`,
			},
			notWant: []string{`"stderr"=`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var entries []string
			log := funcr.New(func(prefix, args string) {
				entries = append(entries, prefix+args)
			}, funcr.Options{Verbosity: 0})

			logApplyOutput(log, tt.stdout, tt.stderr, tt.applyErr)

			output := strings.Join(entries, "\n")
			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Errorf("log output %q does not contain %q", output, want)
				}
			}
			for _, notWant := range tt.notWant {
				if strings.Contains(output, notWant) {
					t.Errorf("log output %q unexpectedly contains %q", output, notWant)
				}
			}
		})
	}
}
