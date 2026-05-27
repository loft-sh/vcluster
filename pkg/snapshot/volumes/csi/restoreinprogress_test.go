package csi

import (
	"strings"
	"testing"

	snapshotapi "github.com/loft-sh/api/v4/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/snapshot/volumes"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/events"
)

func TestInProgressPVCReconcileFinishedSkippedEmitsWarning(t *testing.T) {
	fakeRecorder := events.NewFakeRecorder(10)
	restorer := &Restorer{
		snapshotHandler: snapshotHandler{
			eventRecorder: fakeRecorder,
			logger:        loghelper.New("restore-skip-event-test"),
		},
	}

	requestObj := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "req", Namespace: "host-ns"}}
	volumeRestoreRequest := volumes.RestoreRequest{
		PersistentVolumeClaim: corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: "data", Namespace: "app-ns"},
		},
	}
	volumeRestoreStatus := volumes.RestoreStatus{Phase: snapshotapi.VolumeSnapshotPhaseSkipped}

	restorer.inProgressPVCReconcileFinished(requestObj, volumeRestoreRequest, volumeRestoreStatus, nil)

	select {
	case event := <-fakeRecorder.Events:
		if !strings.HasPrefix(event, "Warning ") {
			t.Errorf("expected Warning event for skipped restore, got: %s", event)
		}
		if !strings.Contains(event, "VolumeRestoreSkipped") {
			t.Errorf("expected reason VolumeRestoreSkipped, got: %s", event)
		}
		if !strings.Contains(event, "app-ns/data") {
			t.Errorf("expected event to reference the PVC namespace/name, got: %s", event)
		}
		for _, fragment := range []string{
			"already exists",
			"NOT applied",
			"Delete the PersistentVolumeClaim",
			"re-run the restore",
		} {
			if !strings.Contains(event, fragment) {
				t.Errorf("expected event message to contain %q, got: %s", fragment, event)
			}
		}
	default:
		t.Fatal("expected an event to be recorded for skipped restore, got none")
	}
}

func TestInProgressPVCReconcileFinishedCompletedAndFailedUnchanged(t *testing.T) {
	cases := []struct {
		name           string
		status         volumes.RestoreStatus
		expectedPrefix string
		expectedReason string
	}{
		{
			name:           "completed",
			status:         volumes.RestoreStatus{Phase: snapshotapi.VolumeSnapshotPhaseCompleted},
			expectedPrefix: "Normal ",
			expectedReason: "VolumeRestored",
		},
		{
			name:           "failed",
			status:         volumes.RestoreStatus{Phase: snapshotapi.VolumeSnapshotPhaseFailed},
			expectedPrefix: "Warning ",
			expectedReason: "VolumeRestoreFailed",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeRecorder := events.NewFakeRecorder(10)
			restorer := &Restorer{
				snapshotHandler: snapshotHandler{
					eventRecorder: fakeRecorder,
					logger:        loghelper.New("restore-event-regression-test"),
				},
			}

			requestObj := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "req", Namespace: "host-ns"}}
			volumeRestoreRequest := volumes.RestoreRequest{
				PersistentVolumeClaim: corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{Name: "data", Namespace: "app-ns"},
				},
			}

			restorer.inProgressPVCReconcileFinished(requestObj, volumeRestoreRequest, tc.status, nil)

			select {
			case event := <-fakeRecorder.Events:
				if !strings.HasPrefix(event, tc.expectedPrefix) {
					t.Errorf("expected prefix %q, got: %s", tc.expectedPrefix, event)
				}
				if !strings.Contains(event, tc.expectedReason) {
					t.Errorf("expected reason %q, got: %s", tc.expectedReason, event)
				}
			default:
				t.Fatalf("expected an event to be recorded for %s phase, got none", tc.name)
			}
		})
	}
}
