package snapshot

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	snapshotapi "github.com/loft-sh/api/v4/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testRequestNamespace = "vcluster-test"
	testVClusterName     = "test"
	testSnapshotURL      = "container:///snapshot-data/test.tar.gz"
)

// newRequestConfigMap builds a snapshot request ConfigMap the way the CLI does,
// then stamps it with an explicit name and creation timestamp so tests can
// control request ordering.
func newRequestConfigMap(t *testing.T, name, url string, phase snapshotapi.RequestPhase, created metav1.Time) *corev1.ConfigMap {
	t.Helper()
	req := &snapshotapi.Request{
		RequestMetadata: snapshotapi.RequestMetadata{
			Name:              name,
			CreationTimestamp: created,
		},
		Spec:   snapshotapi.RequestSpec{URL: url},
		Status: snapshotapi.RequestStatus{Phase: phase},
	}
	cm, err := snapshotapi.NewSnapshotRequestConfigMap(testRequestNamespace, testVClusterName, req)
	if err != nil {
		t.Fatalf("failed to build snapshot request ConfigMap: %v", err)
	}
	cm.Name = name
	return cm
}

// phaseOf reads the current request phase back out of the ConfigMap stored in the client.
func phaseOf(t *testing.T, ctx context.Context, c client.Client, name string) snapshotapi.RequestPhase {
	t.Helper()
	var cm corev1.ConfigMap
	if err := c.Get(ctx, types.NamespacedName{Namespace: testRequestNamespace, Name: name}, &cm); err != nil {
		t.Fatalf("failed to get ConfigMap %s: %v", name, err)
	}
	req, err := snapshotapi.UnmarshalRequest(&cm)
	if err != nil {
		t.Fatalf("failed to unmarshal request from ConfigMap %s: %v", name, err)
	}
	return req.Status.Phase
}

// TestCancelPreviousRequests covers the cancellation decision that a new snapshot
// request supersedes an earlier one. This used to be exercised by an e2e spec, but
// after volume snapshots were removed a request completes too fast to reliably catch
// mid-flight, so the behavior is verified here deterministically instead.
func TestCancelPreviousRequests(t *testing.T) {
	newer := metav1.NewTime(time.Unix(2000, 0))
	older := metav1.NewTime(time.Unix(1000, 0))

	tests := []struct {
		name string
		// current is the incoming request passed to cancelPreviousRequests.
		currentName  string
		currentPhase snapshotapi.RequestPhase
		currentURL   string
		currentTime  metav1.Time
		// other is the pre-existing request stored as a ConfigMap.
		otherPhase snapshotapi.RequestPhase
		otherURL   string
		otherTime  metav1.Time

		wantOtherPhase  snapshotapi.RequestPhase
		wantCanContinue bool
	}{
		{
			name:            "cancels an older request that is still creating the etcd backup",
			currentName:     "req-new",
			currentPhase:    snapshotapi.RequestPhaseNotStarted,
			currentURL:      testSnapshotURL,
			currentTime:     newer,
			otherPhase:      snapshotapi.RequestPhaseCreatingEtcdBackup,
			otherURL:        testSnapshotURL,
			otherTime:       older,
			wantOtherPhase:  snapshotapi.RequestPhaseCanceling,
			wantCanContinue: false,
		},
		{
			name:            "cancels an older request that has not started yet",
			currentName:     "req-new",
			currentPhase:    snapshotapi.RequestPhaseNotStarted,
			currentURL:      testSnapshotURL,
			currentTime:     newer,
			otherPhase:      snapshotapi.RequestPhaseNotStarted,
			otherURL:        testSnapshotURL,
			otherTime:       older,
			wantOtherPhase:  snapshotapi.RequestPhaseCanceling,
			wantCanContinue: false,
		},
		{
			name:            "does not cancel an older request that already completed",
			currentName:     "req-new",
			currentPhase:    snapshotapi.RequestPhaseNotStarted,
			currentURL:      testSnapshotURL,
			currentTime:     newer,
			otherPhase:      snapshotapi.RequestPhaseCompleted,
			otherURL:        testSnapshotURL,
			otherTime:       older,
			wantOtherPhase:  snapshotapi.RequestPhaseCompleted,
			wantCanContinue: true,
		},
		{
			name:            "does not cancel a request for a different snapshot URL",
			currentName:     "req-new",
			currentPhase:    snapshotapi.RequestPhaseNotStarted,
			currentURL:      testSnapshotURL,
			currentTime:     newer,
			otherPhase:      snapshotapi.RequestPhaseCreatingEtcdBackup,
			otherURL:        "container:///snapshot-data/other.tar.gz",
			otherTime:       older,
			wantOtherPhase:  snapshotapi.RequestPhaseCreatingEtcdBackup,
			wantCanContinue: true,
		},
		{
			name:            "does nothing once the current request has already started",
			currentName:     "req-new",
			currentPhase:    snapshotapi.RequestPhaseCreatingEtcdBackup,
			currentURL:      testSnapshotURL,
			currentTime:     newer,
			otherPhase:      snapshotapi.RequestPhaseCreatingEtcdBackup,
			otherURL:        testSnapshotURL,
			otherTime:       older,
			wantOtherPhase:  snapshotapi.RequestPhaseCreatingEtcdBackup,
			wantCanContinue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			otherCM := newRequestConfigMap(t, "req-old", tt.otherURL, tt.otherPhase, tt.otherTime)

			scheme := runtime.NewScheme()
			if err := corev1.AddToScheme(scheme); err != nil {
				t.Fatalf("failed to register corev1 scheme: %v", err)
			}
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(otherCM).Build()

			// Reconciler shadows several reconcilerBase fields (logger, vConfig,
			// isHostMode); its own methods read the outer ones, so both must be set.
			logger := loghelper.NewFromExisting(logr.Discard(), "test")
			vConfig := &config.VirtualClusterConfig{HostNamespace: testRequestNamespace}
			r := &Reconciler{
				reconcilerBase: reconcilerBase{
					vConfig:            vConfig,
					requestsKubeClient: fakeClient,
					logger:             logger,
					isHostMode:         true,
				},
				vConfig:    vConfig,
				logger:     logger,
				isHostMode: true,
			}

			current := &snapshotapi.Request{
				RequestMetadata: snapshotapi.RequestMetadata{
					Name:              tt.currentName,
					CreationTimestamp: tt.currentTime,
				},
				Spec:   snapshotapi.RequestSpec{URL: tt.currentURL},
				Status: snapshotapi.RequestStatus{Phase: tt.currentPhase},
			}

			canContinue, err := r.cancelPreviousRequests(ctx, current)
			if err != nil {
				t.Fatalf("cancelPreviousRequests returned error: %v", err)
			}
			if canContinue != tt.wantCanContinue {
				t.Errorf("canContinue = %v, want %v", canContinue, tt.wantCanContinue)
			}
			if got := phaseOf(t, ctx, fakeClient, "req-old"); got != tt.wantOtherPhase {
				t.Errorf("other request phase = %q, want %q", got, tt.wantOtherPhase)
			}
		})
	}
}
