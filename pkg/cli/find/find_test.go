package find_test

import (
	"reflect"
	"testing"

	"github.com/loft-sh/vcluster/pkg/cli/find"
)

func TestVClusterFromContext(t *testing.T) {
	type test struct {
		originalContext string
		want            []string
	}

	tests := []test{
		{originalContext: "foo", want: []string{"", "", ""}},
		{originalContext: "vcluster_context", want: []string{"vcluster_context", "", ""}},
		{originalContext: "vcluster_context_foobar", want: []string{"vcluster_context_foobar", "", ""}},
		{originalContext: "vcluster_test_vcluster-test_minikube", want: []string{"test", "vcluster-test", "minikube"}},
		{originalContext: "vcluster_test_vcluster-test_minikube_test", want: []string{"test", "vcluster-test", "minikube_test"}},
	}

	for _, test := range tests {
		a, b, c := find.VClusterFromContext(test.originalContext)
		got := []string{a, b, c}
		if !reflect.DeepEqual(got, test.want) {
			t.Fatalf("got %v, want %v", got, test.want)
		}
	}
}
