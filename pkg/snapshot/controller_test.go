package snapshot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	snapshotapi "github.com/loft-sh/api/v4/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testRequestNamespace = "vcluster-test"
	testVClusterName     = "test"
	testSnapshotURL      = "container:///snapshot-data/test.tar.gz"

	testRequestName = "my-vci-snap-snapshot-request"
	testRequestURL  = "s3://my-bucket/backups/my-vci-snap.tar.gz"
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

// standaloneReconciler builds a standalone Reconciler backed by a fake client. Reconciler shadows
// several reconcilerBase fields, so both copies are set.
func standaloneReconciler(t *testing.T, objs ...client.Object) *Reconciler {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to register corev1 scheme: %v", err)
	}
	vConfig := &config.VirtualClusterConfig{}
	vConfig.ControlPlane.Standalone.Enabled = true
	logger := loghelper.New("test")

	return &Reconciler{
		reconcilerBase: reconcilerBase{
			vConfig:            vConfig,
			requestsKubeClient: fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build(),
			logger:             logger,
		},
		vConfig: vConfig,
		logger:  logger,
	}
}

func requestConfigMap(age time.Duration) *corev1.ConfigMap {
	return &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name:      testRequestName,
		Namespace: "kube-system",
		// as the API server would set it
		CreationTimestamp: metav1.NewTime(time.Now().Add(-age)),
	}}
}

func snapshotRequest() *snapshotapi.Request {
	return &snapshotapi.Request{Spec: snapshotapi.RequestSpec{URL: testRequestURL}}
}

// TestResolveSnapshotOptions_StandalonePull pins the standalone behavior: the storage location comes
// from the request's (non-secret) URL, credentials are pulled from the platform (per instance) and
// overlaid, a resolver error requeues rather than fails, and credentials are cached so repeated
// reconciles don't re-hit the platform.
func TestResolveSnapshotOptions_StandalonePull(t *testing.T) {
	orig := pro.ResolveSnapshotCredentials
	defer func() { pro.ResolveSnapshotCredentials = orig }()

	r := standaloneReconciler(t)
	configMap := requestConfigMap(0)

	// resolver error -> requeue (not fail)
	pro.ResolveSnapshotCredentials = func(context.Context) (*snapshotapi.Options, error) {
		return nil, errors.New("platform unreachable")
	}
	options, requeue, err := r.resolveSnapshotOptions(context.Background(), configMap, snapshotRequest())
	if err != nil || !requeue || options != nil {
		t.Fatalf("resolver error should requeue: got options=%v requeue=%v err=%v", options, requeue, err)
	}

	// resolver success -> bucket/key from the request URL; type, connection settings and credentials
	// from the platform; result cached
	calls := 0
	pro.ResolveSnapshotCredentials = func(context.Context) (*snapshotapi.Options, error) {
		calls++
		creds := &snapshotapi.Options{Type: "s3"}
		creds.S3.AccessKeyID = "AKID"
		creds.S3.SecretAccessKey = "SECRET"
		creds.S3.SessionToken = "TOKEN"
		creds.S3.Region = "eu-west-1"
		creds.S3.S3URL = "https://minio.example:9000"
		return creds, nil
	}
	options, requeue, err = r.resolveSnapshotOptions(context.Background(), configMap, snapshotRequest())
	if err != nil || requeue || options == nil {
		t.Fatalf("resolver success expected: got options=%v requeue=%v err=%v", options, requeue, err)
	}
	if options.S3.Bucket != "my-bucket" || options.S3.Key != "backups/my-vci-snap.tar.gz" {
		t.Errorf("bucket/key should come from the request URL: got bucket=%q key=%q", options.S3.Bucket, options.S3.Key)
	}
	if options.Type != "s3" || options.S3.Region != "eu-west-1" || options.S3.S3URL != "https://minio.example:9000" {
		t.Errorf("type and connection settings should come from the platform: got type=%q region=%q url=%q", options.Type, options.S3.Region, options.S3.S3URL)
	}
	if options.S3.AccessKeyID != "AKID" || options.S3.SecretAccessKey != "SECRET" || options.S3.SessionToken != "TOKEN" {
		t.Errorf("credentials should come from the platform: got %+v", options.S3)
	}

	// second call -> served from cache, resolver not invoked again
	if _, _, err = r.resolveSnapshotOptions(context.Background(), configMap, snapshotRequest()); err != nil {
		t.Fatalf("cached call errored: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected credentials resolver called once (then cached), got %d", calls)
	}
}

// TestResolveSnapshotOptions_PushedSecretWins pins that the pull is only for requests that arrive
// without options. A request the tenant created itself carries its own credentials, and pulling the
// platform's instead would pair them with a location they were not issued for.
func TestResolveSnapshotOptions_PushedSecretWins(t *testing.T) {
	orig := pro.ResolveSnapshotCredentials
	defer func() { pro.ResolveSnapshotCredentials = orig }()

	pulled := false
	pro.ResolveSnapshotCredentials = func(context.Context) (*snapshotapi.Options, error) {
		pulled = true
		return &snapshotapi.Options{Type: "s3"}, nil
	}

	pushed := &snapshotapi.Options{Type: "s3"}
	pushed.S3.Bucket = "own-bucket"
	pushed.S3.Key = "own/key.tar.gz"
	pushed.S3.AccessKeyID = "OWN"
	optionsJSON, err := json.Marshal(pushed)
	if err != nil {
		t.Fatalf("marshal options: %v", err)
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRequestName,
			Namespace: "kube-system",
			Labels:    map[string]string{snapshotapi.SnapshotRequestLabel: ""},
		},
		Data: map[string][]byte{snapshotapi.OptionsKey: optionsJSON},
	}

	r := standaloneReconciler(t, secret)
	options, requeue, err := r.resolveSnapshotOptions(context.Background(), requestConfigMap(0), snapshotRequest())
	if err != nil || requeue || options == nil {
		t.Fatalf("pushed Secret should resolve: got options=%v requeue=%v err=%v", options, requeue, err)
	}
	if pulled {
		t.Error("the platform must not be asked when the request carries its own options")
	}
	if options.S3.Bucket != "own-bucket" || options.S3.AccessKeyID != "OWN" {
		t.Errorf("options should come from the pushed Secret: got %+v", options.S3)
	}
}

// TestResolveSnapshotOptions_TerminalPullErrors pins which pull failures are hopeless. Retrying them
// only delays the real reason reaching the request's Status.Error, and the platform schedules nothing
// for the instance while a request is in progress.
func TestResolveSnapshotOptions_TerminalPullErrors(t *testing.T) {
	orig := pro.ResolveSnapshotCredentials
	defer func() { pro.ResolveSnapshotCredentials = orig }()

	groupResource := schema.GroupResource{Group: "management.loft.sh", Resource: "virtualclusterinstances"}
	tests := []struct {
		name    string
		err     error
		wantMsg string
	}{
		{
			name:    "no storage configured on the platform",
			err:     kerrors.NewNotFound(groupResource, "my-vci"),
			wantMsg: "no snapshot storage configured",
		},
		{
			name:    "license lapsed or token cannot use the instance",
			err:     kerrors.NewForbidden(groupResource, "my-vci", errors.New("ScheduledSnapshots is not allowed")),
			wantMsg: "refused to supply snapshot credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pro.ResolveSnapshotCredentials = func(context.Context) (*snapshotapi.Options, error) {
				return nil, tt.err
			}

			// fresh request: without the terminal check the deadline would requeue this one
			_, requeue, err := standaloneReconciler(t).resolveSnapshotOptions(context.Background(), requestConfigMap(0), snapshotRequest())
			if requeue {
				t.Fatal("expected a failure, got a requeue")
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantMsg) {
				t.Fatalf("error should explain the cause: got %v", err)
			}
			if !errors.Is(err, tt.err) {
				t.Errorf("error should wrap the platform error, got %v", err)
			}
		})
	}
}

// TestResolveSnapshotOptions_BackendMismatch pins the guard against using credentials against a backend
// they were not issued for, which would otherwise authenticate anonymously rather than fail.
func TestResolveSnapshotOptions_BackendMismatch(t *testing.T) {
	orig := pro.ResolveSnapshotCredentials
	defer func() { pro.ResolveSnapshotCredentials = orig }()

	pro.ResolveSnapshotCredentials = func(context.Context) (*snapshotapi.Options, error) {
		creds := &snapshotapi.Options{Type: "oci"}
		creds.OCI.Username = "robot"
		return creds, nil
	}

	_, requeue, err := standaloneReconciler(t).resolveSnapshotOptions(context.Background(), requestConfigMap(0), snapshotRequest())
	if requeue {
		t.Fatal("expected a failure, got a requeue")
	}
	if err == nil || !strings.Contains(err.Error(), "credentials for") {
		t.Fatalf("expected a backend mismatch error, got %v", err)
	}
}

// TestResolveSnapshotOptions_CredentialDeadline pins the bound on a permanently failing credential
// pull: past the deadline the request must fail with the real reason instead of requeueing forever.
// Without it a request whose credentials never resolve stays in a non-terminal phase, and since the
// platform will not schedule while anything is in progress, that instance silently stops taking
// snapshots for good.
func TestResolveSnapshotOptions_CredentialDeadline(t *testing.T) {
	orig := pro.ResolveSnapshotCredentials
	defer func() { pro.ResolveSnapshotCredentials = orig }()

	resolveErr := errors.New("platform unreachable")
	pro.ResolveSnapshotCredentials = func(context.Context) (*snapshotapi.Options, error) {
		return nil, resolveErr
	}

	// within the deadline the failure is still treated as transient
	// a fresh Reconciler per call: a failed pull must not be cached
	_, requeue, err := standaloneReconciler(t).
		resolveSnapshotOptions(context.Background(), requestConfigMap(credentialResolveDeadline/2), snapshotRequest())
	if err != nil || !requeue {
		t.Fatalf("within the deadline: expected requeue, got requeue=%v err=%v", requeue, err)
	}

	// past it the request fails, and the error names the underlying cause so it lands in Status.Error
	_, requeue, err = standaloneReconciler(t).
		resolveSnapshotOptions(context.Background(), requestConfigMap(credentialResolveDeadline+time.Minute), snapshotRequest())
	if requeue {
		t.Fatal("past the deadline: expected a failure, got a requeue")
	}
	if !errors.Is(err, resolveErr) {
		t.Fatalf("past the deadline: error should wrap the resolver error, got %v", err)
	}
}

// warmCredentialsCache does one successful pull so the reconciler holds a cache entry, then ages it
// past the fresh TTL so the next call has to decide what to do about an unreachable platform.
func warmCredentialsCache(t *testing.T, r *Reconciler, staleFor time.Duration) {
	t.Helper()

	pro.ResolveSnapshotCredentials = func(context.Context) (*snapshotapi.Options, error) {
		creds := &snapshotapi.Options{Type: "s3"}
		creds.S3.AccessKeyID = "CACHED"
		return creds, nil
	}
	if _, _, err := r.resolveSnapshotOptions(context.Background(), requestConfigMap(0), snapshotRequest()); err != nil {
		t.Fatalf("warming the cache: %v", err)
	}

	cached := r.snapshotCredentials.Load()
	if cached == nil {
		t.Fatal("warming the cache: nothing was cached")
	}
	aged := *cached
	aged.expiry = time.Now().Add(-staleFor)
	r.snapshotCredentials.Store(&aged)
}

// TestResolveSnapshotOptions_StaleCredentialsOutliveThePlatform pins that an unreachable platform does
// not cost a backup we already hold credentials for. Without this the request requeues until the
// deadline and the snapshot is skipped, even though the last credentials the platform supplied are
// still in hand and almost certainly still valid.
func TestResolveSnapshotOptions_StaleCredentialsOutliveThePlatform(t *testing.T) {
	orig := pro.ResolveSnapshotCredentials
	defer func() { pro.ResolveSnapshotCredentials = orig }()

	r := standaloneReconciler(t)
	warmCredentialsCache(t, r, 2*snapshotCredentialsCacheTTL)

	calls := 0
	pro.ResolveSnapshotCredentials = func(context.Context) (*snapshotapi.Options, error) {
		calls++
		return nil, errors.New("platform unreachable")
	}

	// well past the deadline: without the fallback this would be a hard failure, not just a requeue
	options, requeue, err := r.resolveSnapshotOptions(
		context.Background(), requestConfigMap(credentialResolveDeadline+time.Minute), snapshotRequest())
	if err != nil || requeue {
		t.Fatalf("cached credentials should carry the snapshot through an outage: requeue=%v err=%v", requeue, err)
	}
	if options.S3.AccessKeyID != "CACHED" {
		t.Errorf("AccessKeyID = %q, want CACHED", options.S3.AccessKeyID)
	}
	if options.S3.Bucket != "my-bucket" {
		t.Errorf("the location must still come from the request URL, got bucket %q", options.S3.Bucket)
	}
	if calls != 1 {
		t.Errorf("the platform should still be tried on every request, got %d calls", calls)
	}

	// the fallback must not extend the entry, so the platform keeps being retried and stops being
	// bypassed the moment it answers again
	if _, _, err := r.resolveSnapshotOptions(context.Background(), requestConfigMap(0), snapshotRequest()); err != nil {
		t.Fatalf("second outage call: %v", err)
	}
	if calls != 2 {
		t.Errorf("expected the platform retried on the second request too, got %d calls", calls)
	}
}

// TestResolveSnapshotOptions_TerminalErrorsNeverFallBack pins that the fallback only covers silence.
// A platform that answers Forbidden or NotFound has revoked the pull, and serving cached credentials
// over that answer would keep snapshotting an instance the platform already said no to.
func TestResolveSnapshotOptions_TerminalErrorsNeverFallBack(t *testing.T) {
	orig := pro.ResolveSnapshotCredentials
	defer func() { pro.ResolveSnapshotCredentials = orig }()

	groupResource := schema.GroupResource{Group: "management.loft.sh", Resource: "virtualclusterinstances"}
	tests := []struct {
		name string
		err  error
	}{
		{"forbidden", kerrors.NewForbidden(groupResource, testRequestName, errors.New("license lapsed"))},
		{"not found", kerrors.NewNotFound(groupResource, testRequestName)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := standaloneReconciler(t)
			warmCredentialsCache(t, r, 2*snapshotCredentialsCacheTTL)

			pro.ResolveSnapshotCredentials = func(context.Context) (*snapshotapi.Options, error) {
				return nil, tt.err
			}

			_, requeue, err := r.resolveSnapshotOptions(context.Background(), requestConfigMap(0), snapshotRequest())
			if err == nil || requeue {
				t.Fatalf("a refusal must not be masked by cached credentials: requeue=%v err=%v", requeue, err)
			}
		})
	}
}

// TestResolveSnapshotOptions_StaleCredentialsExpire pins the ceiling on the fallback: once the entry is
// older than the stale window the reconciler goes back to requeueing and failing on the deadline.
func TestResolveSnapshotOptions_StaleCredentialsExpire(t *testing.T) {
	orig := pro.ResolveSnapshotCredentials
	defer func() { pro.ResolveSnapshotCredentials = orig }()

	r := standaloneReconciler(t)
	warmCredentialsCache(t, r, 2*snapshotCredentialsCacheTTL)

	// age the entry out of the fallback window entirely
	aged := *r.snapshotCredentials.Load()
	aged.staleExpiry = time.Now().Add(-time.Minute)
	r.snapshotCredentials.Store(&aged)

	resolveErr := errors.New("platform unreachable")
	pro.ResolveSnapshotCredentials = func(context.Context) (*snapshotapi.Options, error) {
		return nil, resolveErr
	}

	_, requeue, err := r.resolveSnapshotOptions(context.Background(), requestConfigMap(0), snapshotRequest())
	if err != nil || !requeue {
		t.Fatalf("expired fallback should requeue: requeue=%v err=%v", requeue, err)
	}

	_, requeue, err = r.resolveSnapshotOptions(
		context.Background(), requestConfigMap(credentialResolveDeadline+time.Minute), snapshotRequest())
	if requeue || !errors.Is(err, resolveErr) {
		t.Fatalf("expired fallback past the deadline should fail with the cause: requeue=%v err=%v", requeue, err)
	}
}

// requestWithURL aims a request at an arbitrary URL, which is what whoever can create a Secret-less
// request ConfigMap in the tenant controls.
func requestWithURL(url string) *snapshotapi.Request {
	return &snapshotapi.Request{Spec: snapshotapi.RequestSpec{URL: url}}
}

// Pins what the request may choose, per backend. Only S3 was covered before, and every field here is a
// string, so a wrong-field assignment compiles.
func TestResolveSnapshotOptions_LocationOverlayPerBackend(t *testing.T) {
	orig := pro.ResolveSnapshotCredentials
	defer func() { pro.ResolveSnapshotCredentials = orig }()

	tests := []struct {
		name        string
		url         string
		credentials func() *snapshotapi.Options
		assert      func(*testing.T, *snapshotapi.Options)
	}{
		{
			name: "s3 takes bucket and key, keeps the platform endpoint",
			url:  "s3://my-bucket/backups/snap.tar.gz",
			credentials: func() *snapshotapi.Options {
				creds := &snapshotapi.Options{Type: "s3"}
				creds.S3.S3URL = "https://minio.example:9000"
				creds.S3.AccessKeyID = "AKID"
				return creds
			},
			assert: func(t *testing.T, got *snapshotapi.Options) {
				t.Helper()
				if got.S3.Bucket != "my-bucket" || got.S3.Key != "backups/snap.tar.gz" {
					t.Errorf("location should come from the request: got bucket=%q key=%q", got.S3.Bucket, got.S3.Key)
				}
				if got.S3.S3URL != "https://minio.example:9000" || got.S3.AccessKeyID != "AKID" {
					t.Errorf("endpoint and credentials should come from the platform: got %+v", got.S3)
				}
			},
		},
		{
			name: "oci takes the repository under the platform's registry",
			url:  "oci://registry.example/team/snapshots",
			credentials: func() *snapshotapi.Options {
				creds := &snapshotapi.Options{Type: "oci"}
				creds.OCI.Repository = "registry.example/base"
				creds.OCI.Username = "robot"
				creds.OCI.Password = "hunter2"
				return creds
			},
			assert: func(t *testing.T, got *snapshotapi.Options) {
				t.Helper()
				if got.OCI.Repository != "registry.example/team/snapshots" {
					t.Errorf("repository should come from the request: got %q", got.OCI.Repository)
				}
				if got.OCI.Username != "robot" || got.OCI.Password != "hunter2" {
					t.Errorf("credentials should come from the platform: got %+v", got.OCI)
				}
			},
		},
		{
			name: "container takes the path, which is the tenant's own filesystem",
			url:  "container:///data/snapshots/snap.tar.gz",
			credentials: func() *snapshotapi.Options {
				return &snapshotapi.Options{Type: "container"}
			},
			assert: func(t *testing.T, got *snapshotapi.Options) {
				t.Helper()
				if got.Container.Path != "/data/snapshots/snap.tar.gz" {
					t.Errorf("path should come from the request: got %q", got.Container.Path)
				}
			},
		},
		{
			name: "azure takes the blob URL under the platform's account",
			url:  "https://acct.blob.core.windows.net/container/snap.tar.gz",
			credentials: func() *snapshotapi.Options {
				creds := &snapshotapi.Options{Type: "azure"}
				creds.Azure.BlobURL = "https://acct.blob.core.windows.net/container"
				creds.Azure.StorageKey = "STORAGE-KEY"
				return creds
			},
			assert: func(t *testing.T, got *snapshotapi.Options) {
				t.Helper()
				if got.Azure.BlobURL != "https://acct.blob.core.windows.net/container/snap.tar.gz" {
					t.Errorf("blob URL should come from the request: got %q", got.Azure.BlobURL)
				}
				if got.Azure.StorageKey != "STORAGE-KEY" {
					t.Errorf("credentials should come from the platform: got %+v", got.Azure)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pro.ResolveSnapshotCredentials = func(context.Context) (*snapshotapi.Options, error) {
				return tt.credentials(), nil
			}

			options, requeue, err := standaloneReconciler(t).resolveSnapshotOptions(context.Background(), requestConfigMap(0), requestWithURL(tt.url))
			if err != nil || requeue || options == nil {
				t.Fatalf("expected resolved options: got options=%v requeue=%v err=%v", options, requeue, err)
			}
			tt.assert(t, options)
		})
	}
}

// Pins that a request cannot redirect the platform's live credentials to a host of its choosing. For
// OCI and Azure the location field names the host, so a blind overlay sends the registry password or
// storage key wherever the request points.
func TestResolveSnapshotOptions_RejectsForeignHost(t *testing.T) {
	orig := pro.ResolveSnapshotCredentials
	defer func() { pro.ResolveSnapshotCredentials = orig }()

	tests := []struct {
		name        string
		url         string
		credentials func() *snapshotapi.Options
	}{
		{
			name: "oci registry the platform did not issue credentials for",
			url:  "oci://evil.example/exfiltrated",
			credentials: func() *snapshotapi.Options {
				creds := &snapshotapi.Options{Type: "oci"}
				creds.OCI.Repository = "registry.example/base"
				creds.OCI.Password = "hunter2"
				return creds
			},
		},
		{
			name: "azure host the platform did not issue credentials for",
			url:  "https://evil.example/container/snap.tar.gz",
			credentials: func() *snapshotapi.Options {
				creds := &snapshotapi.Options{Type: "azure"}
				creds.Azure.BlobURL = "https://acct.blob.core.windows.net/container"
				creds.Azure.StorageKey = "STORAGE-KEY"
				return creds
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pro.ResolveSnapshotCredentials = func(context.Context) (*snapshotapi.Options, error) {
				return tt.credentials(), nil
			}

			options, requeue, err := standaloneReconciler(t).resolveSnapshotOptions(context.Background(), requestConfigMap(0), requestWithURL(tt.url))
			if err == nil {
				t.Fatalf("expected the foreign host to be refused, got options=%+v", options)
			}
			if requeue {
				t.Error("a request naming another host will not start matching on retry, so it must not requeue")
			}
			if !strings.Contains(err.Error(), "evil.example") {
				t.Errorf("the error should name the rejected host: %v", err)
			}
		})
	}
}

// json.Unmarshal accepts `{}`, so an unprovisioned instance parses into a zero-value Options. Returned
// as a success it would cache for a minute and then read as a backend mismatch, failing every request
// in that window on a condition that clears itself.
func TestResolveSnapshotOptions_UnprovisionedCredentialsRequeue(t *testing.T) {
	orig := pro.ResolveSnapshotCredentials
	defer func() { pro.ResolveSnapshotCredentials = orig }()

	calls := 0
	pro.ResolveSnapshotCredentials = func(context.Context) (*snapshotapi.Options, error) {
		calls++
		return &snapshotapi.Options{}, nil
	}

	r := standaloneReconciler(t)
	options, requeue, err := r.resolveSnapshotOptions(context.Background(), requestConfigMap(0), snapshotRequest())
	if err != nil || !requeue || options != nil {
		t.Fatalf("an unprovisioned instance should requeue: got options=%v requeue=%v err=%v", options, requeue, err)
	}

	// nothing usable was pulled, so nothing may be cached: the next reconcile has to ask again
	if _, requeue, err = r.resolveSnapshotOptions(context.Background(), requestConfigMap(0), snapshotRequest()); err != nil || !requeue {
		t.Fatalf("second attempt should requeue too: requeue=%v err=%v", requeue, err)
	}
	if calls != 2 {
		t.Errorf("an unusable pull must not be cached as a success, got %d calls", calls)
	}
}

// The other half: retrying re-asks the same platform for the same bytes, so holding the request open
// until the deadline only blocks the instance for ten minutes before failing identically.
func TestResolveSnapshotOptions_MalformedCredentialsAreTerminal(t *testing.T) {
	orig := pro.ResolveSnapshotCredentials
	defer func() { pro.ResolveSnapshotCredentials = orig }()

	pro.ResolveSnapshotCredentials = func(context.Context) (*snapshotapi.Options, error) {
		return nil, fmt.Errorf("%w: unexpected end of JSON input", pro.ErrSnapshotCredentialsMalformed)
	}

	options, requeue, err := standaloneReconciler(t).resolveSnapshotOptions(context.Background(), requestConfigMap(0), snapshotRequest())
	if err == nil {
		t.Fatalf("expected a terminal failure, got options=%+v", options)
	}
	if requeue {
		t.Error("a malformed response will not parse on retry, so it must not requeue")
	}
}

// The false side of the pull guard, which most traffic takes: OSS builds with a nil seam and every
// non-standalone tenant. Dropping the nil check panics there; dropping the standalone check sends
// pod-based tenants to the platform.
func TestResolveSnapshotOptions_NoPullPath(t *testing.T) {
	orig := pro.ResolveSnapshotCredentials
	defer func() { pro.ResolveSnapshotCredentials = orig }()

	pulled := false
	resolver := func(context.Context) (*snapshotapi.Options, error) {
		pulled = true
		return &snapshotapi.Options{Type: "s3"}, nil
	}

	tests := []struct {
		name       string
		standalone bool
		seam       func(context.Context) (*snapshotapi.Options, error)
	}{
		{name: "OSS build: no pro seam", standalone: true, seam: nil},
		{name: "pod-based tenant: not standalone", standalone: false, seam: resolver},
		{name: "neither", standalone: false, seam: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pulled = false
			pro.ResolveSnapshotCredentials = tt.seam

			r := standaloneReconciler(t)
			r.vConfig.ControlPlane.Standalone.Enabled = tt.standalone

			// a fresh request waits for the Secret to show up in the cache
			_, requeue, err := r.resolveSnapshotOptions(context.Background(), requestConfigMap(0), snapshotRequest())
			if err != nil || !requeue {
				t.Fatalf("a request without its Secret should requeue while it may still arrive: requeue=%v err=%v", requeue, err)
			}

			// once it is clear the Secret is not coming, the request fails rather than requeueing forever
			_, requeue, err = r.resolveSnapshotOptions(context.Background(), requestConfigMap(time.Minute), snapshotRequest())
			if err == nil || requeue {
				t.Fatalf("a Secret that never arrived should fail: requeue=%v err=%v", requeue, err)
			}
			if !strings.Contains(err.Error(), "can't find snapshot request Secret") {
				t.Errorf("expected the missing-Secret error, got %v", err)
			}
			if pulled {
				t.Error("the platform must not be asked for credentials on this path")
			}
		})
	}
}
