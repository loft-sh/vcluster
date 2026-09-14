package blockingcacheclient

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	admissionregistrationv1ac "k8s.io/client-go/applyconfigurations/admissionregistration/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const testPolicyName = "vcluster-protected-apiservices"

// applyServer stands in for the manager client underneath CacheClient. Apply
// answers with a fixed server response the way controller-runtime does, by
// decoding it into the ApplyConfiguration, and Get serves whatever the test
// says the informer cache currently holds.
type applyServer struct {
	client.Client

	mu       sync.Mutex
	cached   *unstructured.Unstructured
	response *unstructured.Unstructured
	// getErr, when set, is what every Get returns instead of consulting cached.
	getErr error
	// responseNotWrittenBack makes Apply leave the ApplyConfiguration untouched,
	// the way controller-runtime before v0.24 handled typed ApplyConfigurations.
	responseNotWrittenBack bool
}

func (s *applyServer) Get(_ context.Context, key client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.getErr != nil {
		return s.getErr
	}
	if s.cached == nil {
		return kerrors.NewNotFound(schema.GroupResource{Group: "admissionregistration.k8s.io", Resource: "validatingadmissionpolicies"}, key.Name)
	}
	obj.(*unstructured.Unstructured).Object = s.cached.DeepCopy().Object
	return nil
}

func (s *applyServer) Apply(_ context.Context, obj runtime.ApplyConfiguration, _ ...client.ApplyOption) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.responseNotWrittenBack {
		return nil
	}

	body, err := json.Marshal(s.response.Object)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, obj)
}

func (s *applyServer) Status() client.SubResourceWriter {
	return &applyStatusWriter{server: s}
}

func (s *applyServer) setCached(obj *unstructured.Unstructured) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cached = obj
}

type applyStatusWriter struct {
	client.SubResourceWriter

	server *applyServer
}

func (w *applyStatusWriter) Apply(ctx context.Context, obj runtime.ApplyConfiguration, _ ...client.SubResourceApplyOption) error {
	return w.server.Apply(ctx, obj)
}

func policyObject(uid, resourceVersion string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "admissionregistration.k8s.io/v1",
		"kind":       "ValidatingAdmissionPolicy",
		"metadata": map[string]interface{}{
			"name":            testPolicyName,
			"uid":             uid,
			"resourceVersion": resourceVersion,
		},
	}}
}

func policyApplyConfiguration() *admissionregistrationv1ac.ValidatingAdmissionPolicyApplyConfiguration {
	return admissionregistrationv1ac.ValidatingAdmissionPolicy(testPolicyName).
		WithSpec(admissionregistrationv1ac.ValidatingAdmissionPolicySpec().
			WithFailurePolicy(admissionregistrationv1.Fail))
}

// catchUpAfter makes the fake cache serve obj once delay has passed, standing
// in for the informer receiving the watch event for the write.
func catchUpAfter(server *applyServer, delay time.Duration, obj *unstructured.Unstructured) {
	go func() {
		time.Sleep(delay)
		server.setCached(obj)
	}()
}

func TestApplyNoOpReturnsWithoutWaitingForACacheChange(t *testing.T) {
	// an apply that changes nothing is skipped by the apiserver, which answers
	// with the live object at its current resource version. The cache already
	// holds that version, so there is nothing to wait for.
	server := &applyServer{cached: policyObject("uid-1", "10"), response: policyObject("uid-1", "10")}
	c := &CacheClient{Client: server}

	start := time.Now()
	if err := c.Apply(context.Background(), policyApplyConfiguration(), client.FieldOwner("test")); err != nil {
		t.Fatalf("no-op apply returned %v, expected nil", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("no-op apply blocked for %s", elapsed)
	}
}

func TestStatusApplyNoOpReturnsWithoutWaitingForACacheChange(t *testing.T) {
	server := &applyServer{cached: policyObject("uid-1", "10"), response: policyObject("uid-1", "10")}
	c := &CacheClient{Client: server}

	start := time.Now()
	if err := c.Status().Apply(context.Background(), policyApplyConfiguration(), client.FieldOwner("test")); err != nil {
		t.Fatalf("no-op status apply returned %v, expected nil", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("no-op status apply blocked for %s", elapsed)
	}
}

func TestApplySkipsTheCacheWaitWhenTheResponseIsNotWrittenBack(t *testing.T) {
	// controller-runtime before v0.24 leaves a typed ApplyConfiguration
	// untouched, so the applied object has no resource version to wait for.
	// The cache is behind here, yet waiting could only run into the timeout.
	server := &applyServer{cached: policyObject("uid-1", "10"), responseNotWrittenBack: true}
	c := &CacheClient{Client: server}

	start := time.Now()
	if err := c.Apply(context.Background(), policyApplyConfiguration(), client.FieldOwner("test")); err != nil {
		t.Fatalf("apply returned %v, expected nil", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("apply blocked for %s although the response was not written back", elapsed)
	}
}

func TestStatusApplySkipsTheCacheWaitWhenTheResponseIsNotWrittenBack(t *testing.T) {
	server := &applyServer{cached: policyObject("uid-1", "10"), responseNotWrittenBack: true}
	c := &CacheClient{Client: server}

	start := time.Now()
	if err := c.Status().Apply(context.Background(), policyApplyConfiguration(), client.FieldOwner("test")); err != nil {
		t.Fatalf("status apply returned %v, expected nil", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("status apply blocked for %s although the response was not written back", elapsed)
	}
}

func TestApplyWaitsUntilCacheReachesReturnedResourceVersion(t *testing.T) {
	server := &applyServer{cached: policyObject("uid-1", "10"), response: policyObject("uid-1", "11")}
	c := &CacheClient{Client: server}
	catchUpAfter(server, 150*time.Millisecond, policyObject("uid-1", "11"))

	start := time.Now()
	if err := c.Apply(context.Background(), policyApplyConfiguration(), client.FieldOwner("test")); err != nil {
		t.Fatalf("apply returned %v, expected nil", err)
	}
	if elapsed := time.Since(start); elapsed < 100*time.Millisecond {
		t.Fatalf("apply returned after %s without waiting for the cache to catch up", elapsed)
	}
}

func TestApplyComparesResourceVersionsNumerically(t *testing.T) {
	// "999" sorts after "1000" as a string, so a string comparison would let
	// the stale cache entry through
	server := &applyServer{cached: policyObject("uid-1", "999"), response: policyObject("uid-1", "1000")}
	c := &CacheClient{Client: server}
	catchUpAfter(server, 150*time.Millisecond, policyObject("uid-1", "1000"))

	start := time.Now()
	if err := c.Apply(context.Background(), policyApplyConfiguration(), client.FieldOwner("test")); err != nil {
		t.Fatalf("apply returned %v, expected nil", err)
	}
	if elapsed := time.Since(start); elapsed < 100*time.Millisecond {
		t.Fatalf("apply returned after %s although the cache was still at an older resource version", elapsed)
	}
}

func TestApplyWaitsForCreatedObjectToAppearInCache(t *testing.T) {
	server := &applyServer{response: policyObject("uid-1", "1")}
	c := &CacheClient{Client: server}
	catchUpAfter(server, 150*time.Millisecond, policyObject("uid-1", "1"))

	start := time.Now()
	if err := c.Apply(context.Background(), policyApplyConfiguration(), client.FieldOwner("test")); err != nil {
		t.Fatalf("apply returned %v, expected nil", err)
	}
	if elapsed := time.Since(start); elapsed < 100*time.Millisecond {
		t.Fatalf("apply returned after %s before the created object reached the cache", elapsed)
	}
}

func TestApplyTimesOutWhenCacheNeverCatchesUp(t *testing.T) {
	server := &applyServer{cached: policyObject("uid-1", "10"), response: policyObject("uid-1", "11")}
	c := &CacheClient{Client: server}

	err := c.Apply(context.Background(), policyApplyConfiguration(), client.FieldOwner("test"))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("apply returned %v, expected context.DeadlineExceeded", err)
	}
}

func TestApplyReturnsWhenCacheHoldsANewerReplacement(t *testing.T) {
	// another actor deleted and recreated the object after the apply, so the
	// cache holds a different UID at a higher resource version. The cache has
	// moved past the write, so there is nothing left to wait for.
	server := &applyServer{cached: policyObject("uid-replacement", "13"), response: policyObject("uid-1", "11")}
	c := &CacheClient{Client: server}

	start := time.Now()
	if err := c.Apply(context.Background(), policyApplyConfiguration(), client.FieldOwner("test")); err != nil {
		t.Fatalf("apply returned %v, expected nil", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("apply blocked for %s although the cache already held a newer replacement", elapsed)
	}
}

func TestApplyTimesOutWhileCacheKeepsServingAnOlderIncarnation(t *testing.T) {
	// the apply recreated a deleted object, so the response carries a new UID
	// at a higher resource version while the cache still serves the old one
	server := &applyServer{cached: policyObject("uid-old", "10"), response: policyObject("uid-new", "11")}
	c := &CacheClient{Client: server}

	err := c.Apply(context.Background(), policyApplyConfiguration(), client.FieldOwner("test"))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("apply returned %v, expected context.DeadlineExceeded", err)
	}
}

func TestApplyReturnsCacheReadErrorsOtherThanNotFound(t *testing.T) {
	// a read error other than NotFound means the cache cannot answer at all,
	// for example a namespace the manager does not watch, so the poll stops
	// and surfaces it instead of waiting out the timeout
	readErr := errors.New("unable to get: namespace is not watched by the cache")
	server := &applyServer{getErr: readErr, response: policyObject("uid-1", "11")}
	c := &CacheClient{Client: server}

	start := time.Now()
	err := c.Apply(context.Background(), policyApplyConfiguration(), client.FieldOwner("test"))
	if !errors.Is(err, readErr) {
		t.Fatalf("apply returned %v, expected the cache read error", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("apply kept polling for %s after the cache read failed", elapsed)
	}
}

func TestApplyTreatsUnregisteredKindAsObserved(t *testing.T) {
	// the cache cannot track a kind the scheme does not know, so there is
	// nothing to wait for and the apply counts as observed right away
	gvk := schema.GroupVersionKind{Group: "admissionregistration.k8s.io", Version: "v1", Kind: "ValidatingAdmissionPolicy"}
	server := &applyServer{getErr: runtime.NewNotRegisteredErrForKind("test", gvk), response: policyObject("uid-1", "11")}
	c := &CacheClient{Client: server}

	start := time.Now()
	if err := c.Apply(context.Background(), policyApplyConfiguration(), client.FieldOwner("test")); err != nil {
		t.Fatalf("apply returned %v, expected nil for an unregistered kind", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("apply blocked for %s on an unregistered kind", elapsed)
	}
}

func TestApplyWaitsForTheExactOpaqueResourceVersion(t *testing.T) {
	// versions that are not decimal counters are opaque, so ordering them says
	// nothing: "z" sorts after "aa" yet may be older. Only the cache serving
	// exactly the returned version proves the write was observed.
	server := &applyServer{cached: policyObject("uid-1", "z"), response: policyObject("uid-1", "aa")}
	c := &CacheClient{Client: server}
	catchUpAfter(server, 150*time.Millisecond, policyObject("uid-1", "aa"))

	start := time.Now()
	if err := c.Apply(context.Background(), policyApplyConfiguration(), client.FieldOwner("test")); err != nil {
		t.Fatalf("apply returned %v, expected nil", err)
	}
	if elapsed := time.Since(start); elapsed < 100*time.Millisecond {
		t.Fatalf("apply returned after %s by ordering opaque resource versions", elapsed)
	}
}

func TestResourceVersionReached(t *testing.T) {
	cases := []struct {
		cached, returned string
		want             bool
	}{
		// decimal counters compare numerically
		{cached: "10", returned: "10", want: true},
		{cached: "11", returned: "10", want: true},
		{cached: "9", returned: "10", want: false},
		{cached: "999", returned: "1000", want: false},
		{cached: "1000", returned: "999", want: true},
		// opaque versions only match exactly
		{cached: "abc", returned: "abc", want: true},
		{cached: "abd", returned: "abc", want: false},
		{cached: "abc", returned: "abd", want: false},
		{cached: "z", returned: "aa", want: false},
		{cached: "aa", returned: "z", want: false},
		// a server that sets no version leaves nothing to wait for
		{cached: "", returned: "", want: true},
		{cached: "10", returned: "", want: false},
	}
	for _, tc := range cases {
		if got := resourceVersionReached(tc.cached, tc.returned); got != tc.want {
			t.Errorf("resourceVersionReached(%q, %q) = %v, want %v", tc.cached, tc.returned, got, tc.want)
		}
	}
}
