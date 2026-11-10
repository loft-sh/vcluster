package leaderelection

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/loft-sh/vcluster/config"
	appsv1 "k8s.io/api/apps/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
)

const (
	testNamespace = "vcluster-test"
	testName      = "test"
	testLeader    = "test-0-external-vcluster-controller"
)

func TestShouldReportNotServing(t *testing.T) {
	tests := []struct {
		name             string
		leader           string
		onlyLeaderServes bool
		want             bool
	}{
		{name: "this pod leads", leader: "self", onlyLeaderServes: true, want: false},
		{name: "another pod leads with a shared store", leader: "other", onlyLeaderServes: false, want: false},
		{name: "another pod leads on embedded SQLite", leader: "other", onlyLeaderServes: true, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldReportNotServing(tt.leader, "self", tt.onlyLeaderServes); got != tt.want {
				t.Fatalf("shouldReportNotServing(%q) = %t, want %t", tt.leader, got, tt.want)
			}
		})
	}
}

func TestStatefulSetOrdinal(t *testing.T) {
	tests := []struct {
		pod  string
		want int
	}{
		{pod: testName + "-0", want: 0},
		{pod: testName + "-1", want: 1},
		{pod: testName + "-12", want: 12},
		{pod: testName + "-5d8f7c9b4-x2kqp", want: -1},
		{pod: testName + "-123-x2kqp", want: -1},
		{pod: "other-0", want: -1},
		{pod: testName, want: -1},
	}

	for _, tt := range tests {
		t.Run(tt.pod, func(t *testing.T) {
			if got := statefulSetOrdinal(tt.pod, testName); got != tt.want {
				t.Fatalf("statefulSetOrdinal(%q) = %d, want %d", tt.pod, got, tt.want)
			}
		})
	}
}

func TestControlPlaneWorkload(t *testing.T) {
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
	}
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
	}

	tests := []struct {
		name     string
		objects  []runtime.Object
		stsError error
		want     runtime.Object
		wantErr  bool
	}{
		{name: "statefulset", objects: []runtime.Object{sts}, want: sts},
		{name: "deployment only", objects: []runtime.Object{deploy}, want: deploy},
		{name: "neither", wantErr: true},
		{name: "statefulset lookup fails", objects: []runtime.Object{deploy}, stsError: errors.New("boom"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientset(tt.objects...)
			if tt.stsError != nil {
				client.PrependReactor("get", "statefulsets", func(clienttesting.Action) (bool, runtime.Object, error) {
					return true, nil, tt.stsError
				})
			}

			got, err := controlPlaneWorkload(context.Background(), client, testNamespace, testName)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %T", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if fmt.Sprintf("%T", got) != fmt.Sprintf("%T", tt.want) {
				t.Fatalf("got %T, want %T", got, tt.want)
			}
		})
	}
}

func TestReportNotServing(t *testing.T) {
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
	}

	t.Run("records a warning on the workload", func(t *testing.T) {
		recorder := record.NewFakeRecorder(1)
		reportNotServing(context.Background(), fake.NewClientset(sts), recorder, testNamespace, testName, testLeader, config.StoreTypeEmbeddedDatabase)

		if len(recorder.Events) != 1 {
			t.Fatalf("got %d events, want 1", len(recorder.Events))
		}
		event := <-recorder.Events
		if !strings.HasPrefix(event, "Warning "+EventReasonReplicaNotServing+" ") {
			t.Fatalf("unexpected event %q", event)
		}
		for _, want := range []string{testLeader, string(config.StoreTypeEmbeddedDatabase)} {
			if !strings.Contains(event, want) {
				t.Fatalf("event %q must name %q", event, want)
			}
		}
	})

	t.Run("records nothing without a workload", func(t *testing.T) {
		recorder := record.NewFakeRecorder(1)
		reportNotServing(context.Background(), fake.NewClientset(), recorder, testNamespace, testName, "other", config.StoreTypeEmbeddedDatabase)

		if len(recorder.Events) != 0 {
			t.Fatalf("got %d events, want 0", len(recorder.Events))
		}
	})
}

func TestLeasesForbidden(t *testing.T) {
	leases := schema.GroupResource{Group: "coordination.k8s.io", Resource: "leases"}
	tests := []struct {
		name   string
		getErr error
		want   bool
	}{
		{name: "lease missing", want: false},
		{name: "forbidden", getErr: kerrors.NewForbidden(leases, testName, errors.New("no rule")), want: true},
		{name: "other error", getErr: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientset()
			if tt.getErr != nil {
				client.PrependReactor("get", "leases", func(clienttesting.Action) (bool, runtime.Object, error) {
					return true, nil, tt.getErr
				})
			}

			if got := leasesForbidden(context.Background(), client, testNamespace, testName); got != tt.want {
				t.Fatalf("leasesForbidden() = %t, want %t", got, tt.want)
			}
		})
	}
}
