package osimage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/api/v4/pkg/clientset/versioned/fake"
	"github.com/loft-sh/log"
	"gotest.tools/v3/assert"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stesting "k8s.io/client-go/testing"
)

const (
	imageName = "ubuntu-24.04"
	checksum  = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
)

// store stands in for the object store the pre-signed targets point at. A target's URL
// carries the generation it was signed in, so a refreshed target is distinguishable from the
// expired one it replaces.
type store struct {
	server *httptest.Server

	mutex sync.Mutex
	parts map[int32][]byte
	// partGeneration records which signing a part finally landed on, so a test can tell a
	// retry of the same target from a retry of a refreshed one.
	partGeneration map[int32]int
	generation     int
	// failures[partNumber] is how many more attempts on that part are answered with failStatus.
	failures   map[int32]int
	failStatus int
}

func newStore(t *testing.T) *store {
	t.Helper()

	s := &store{
		parts:          map[int32][]byte{},
		partGeneration: map[int32]int{},
		failures:       map[int32]int{},
		generation:     1,
		failStatus:     http.StatusInternalServerError,
	}
	s.server = httptest.NewServer(http.HandlerFunc(s.serve))
	t.Cleanup(s.server.Close)
	return s
}

func (s *store) serve(w http.ResponseWriter, r *http.Request) {
	generation, partNumber, err := parseTargetPath(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.failures[partNumber] > 0 {
		s.failures[partNumber]--
		http.Error(w, "try again", s.failStatus)
		return
	}

	body := &bytes.Buffer{}
	if _, err := body.ReadFrom(r.Body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.parts[partNumber] = body.Bytes()
	s.partGeneration[partNumber] = generation
}

func parseTargetPath(path string) (generation int, partNumber int32, err error) {
	fields := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(fields) != 2 {
		return 0, 0, fmt.Errorf("unexpected target path %q", path)
	}

	generation, err = strconv.Atoi(fields[0])
	if err != nil {
		return 0, 0, err
	}
	part, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0, 0, err
	}
	return generation, int32(part), nil
}

// url signs a target for the store's current generation. Called while upload holds the
// store lock, so it takes none of its own.
func (s *store) url(partNumber int32) string {
	return fmt.Sprintf("%s/%d/%d", s.server.URL, s.generation, partNumber)
}

func (s *store) assembled() []byte {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	assembled := &bytes.Buffer{}
	for partNumber := int32(1); ; partNumber++ {
		part, ok := s.parts[partNumber]
		if !ok {
			return assembled.Bytes()
		}
		assembled.Write(part)
	}
}

// platform serves the upload subresource the way the aggregated API does: batches of
// batchSize targets, paging with fromPart until the last part is covered.
type platform struct {
	store         *store
	partSizeBytes int64
	batchSize     int32

	mutex     sync.Mutex
	uploads   []managementv1.OSImageUploadSpec
	finalized []string
}

func newPlatform(store *store, partSizeBytes int64, batchSize int32) *platform {
	return &platform{store: store, partSizeBytes: partSizeBytes, batchSize: batchSize}
}

func (p *platform) client(objects ...runtime.Object) *fake.Clientset {
	clientset := fake.NewSimpleClientset(objects...)
	clientset.PrependReactor("create", "osimages", func(action k8stesting.Action) (bool, runtime.Object, error) {
		create, ok := action.(k8stesting.CreateAction)
		if !ok {
			return false, nil, nil
		}

		switch create.GetSubresource() {
		case "upload":
			return true, p.upload(create.GetObject().(*managementv1.OSImageUpload)), nil
		case "finalize":
			finalize := create.GetObject().(*managementv1.OSImageFinalize)
			p.mutex.Lock()
			defer p.mutex.Unlock()
			p.finalized = append(p.finalized, finalize.Spec.Checksum)
			return true, finalize, nil
		}
		return false, nil, nil
	})
	return clientset
}

func (p *platform) upload(request *managementv1.OSImageUpload) *managementv1.OSImageUpload {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.uploads = append(p.uploads, request.Spec)

	// Every call re-signs, so a refreshed target is a new generation to the store.
	p.store.mutex.Lock()
	p.store.generation = len(p.uploads)
	p.store.mutex.Unlock()

	totalParts := int32((request.Spec.SizeBytes + p.partSizeBytes - 1) / p.partSizeBytes)
	fromPart := max(request.Spec.FromPart, 1)

	response := &managementv1.OSImageUpload{Status: managementv1.OSImageUploadStatus{
		PartSizeBytes: p.partSizeBytes,
		TotalParts:    totalParts,
	}}
	for partNumber := fromPart; partNumber < fromPart+p.batchSize && partNumber <= totalParts; partNumber++ {
		response.Status.Targets = append(response.Status.Targets, managementv1.OSImageUploadTarget{
			PartNumber: partNumber,
			URL:        p.store.url(partNumber),
		})
	}
	if next := fromPart + p.batchSize; next <= totalParts {
		response.Status.NextPart = &next
	}
	return response
}

func (p *platform) uploadCalls() []managementv1.OSImageUploadSpec {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return slices.Clone(p.uploads)
}

func (p *platform) finalizeCalls() []string {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return slices.Clone(p.finalized)
}

func newTestUploader(images *fake.Clientset) *Uploader {
	uploader := NewUploader(images.ManagementV1().OSImages(), log.Discard)
	uploader.backoff = 0
	uploader.pollInterval = time.Millisecond
	return uploader
}

func testImage(size int) []byte {
	image := make([]byte, size)
	for i := range image {
		image[i] = byte(i)
	}
	return image
}

func TestUploadPagesEveryPart(t *testing.T) {
	image := testImage(25)
	store := newStore(t)
	// 25 bytes over 4 byte parts is 7 parts, which three-target batches page three times.
	platform := newPlatform(store, 4, 3)

	uploader := newTestUploader(platform.client())
	assert.NilError(t, uploader.Upload(t.Context(), imageName, bytes.NewReader(image), int64(len(image))))
	assert.NilError(t, uploader.Finalize(t.Context(), imageName, checksum))

	assert.DeepEqual(t, image, store.assembled())
	assert.DeepEqual(t, []managementv1.OSImageUploadSpec{
		{SizeBytes: 25, FromPart: 1},
		{SizeBytes: 25, FromPart: 4},
		{SizeBytes: 25, FromPart: 7},
	}, platform.uploadCalls())
	assert.DeepEqual(t, []string{checksum}, platform.finalizeCalls())
}

func TestUploadRetriesAFailedPart(t *testing.T) {
	image := testImage(12)
	store := newStore(t)
	store.failures[2] = 2
	platform := newPlatform(store, 4, 3)

	err := newTestUploader(platform.client()).Upload(t.Context(), imageName, bytes.NewReader(image), int64(len(image)))
	assert.NilError(t, err)

	assert.DeepEqual(t, image, store.assembled())
	// Retried in place, so no second batch was requested for the same parts.
	assert.Equal(t, 1, len(platform.uploadCalls()))
}

func TestUploadRefreshesAnExpiredTarget(t *testing.T) {
	image := testImage(12)
	store := newStore(t)
	store.failStatus = http.StatusForbidden
	// An expired signature is not retried in place, so one rejection sends the part through
	// the refresh path.
	store.failures[2] = 1
	platform := newPlatform(store, 4, 3)

	err := newTestUploader(platform.client()).Upload(t.Context(), imageName, bytes.NewReader(image), int64(len(image)))
	assert.NilError(t, err)

	assert.DeepEqual(t, image, store.assembled())
	assert.DeepEqual(t, []managementv1.OSImageUploadSpec{
		{SizeBytes: 12, FromPart: 1},
		{SizeBytes: 12, FromPart: 2},
	}, platform.uploadCalls())
	// Part 2 landed on the refreshed target, the others on the ones first handed out.
	assert.DeepEqual(t, map[int32]int{1: 1, 2: 2, 3: 1}, store.partGeneration)
}

func TestUploadFailsWhenAPartNeverLands(t *testing.T) {
	image := testImage(12)
	store := newStore(t)
	store.failures[2] = 1000
	platform := newPlatform(store, 4, 3)

	err := newTestUploader(platform.client()).Upload(t.Context(), imageName, bytes.NewReader(image), int64(len(image)))
	assert.ErrorContains(t, err, "upload part 2")
}

func TestUploadRejectsACursorThatDoesNotAdvance(t *testing.T) {
	image := testImage(12)
	store := newStore(t)
	platform := newPlatform(store, 4, 3)
	client := platform.client()
	client.PrependReactor("create", "osimages", func(action k8stesting.Action) (bool, runtime.Object, error) {
		create, ok := action.(k8stesting.CreateAction)
		if !ok || create.GetSubresource() != "upload" {
			return false, nil, nil
		}

		response := platform.upload(create.GetObject().(*managementv1.OSImageUpload))
		stuck := int32(1)
		response.Status.NextPart = &stuck
		return true, response, nil
	})

	err := newTestUploader(client).Upload(t.Context(), imageName, bytes.NewReader(image), int64(len(image)))
	assert.ErrorContains(t, err, "does not advance")
}

func TestWaitForReady(t *testing.T) {
	tests := []struct {
		name        string
		phases      []storagev1.OSImagePhase
		conditions  agentstoragev1.Conditions
		expectedErr string
	}{
		{
			name:   "ready after uploading",
			phases: []storagev1.OSImagePhase{storagev1.OSImagePhaseUploading, storagev1.OSImagePhaseReady},
		},
		{
			name:   "ready after an unobserved image",
			phases: []storagev1.OSImagePhase{"", storagev1.OSImagePhasePending, storagev1.OSImagePhaseReady},
		},
		{
			name:        "failed reports the condition",
			phases:      []storagev1.OSImagePhase{storagev1.OSImagePhaseFailed},
			conditions:  agentstoragev1.Conditions{{Message: "stored 10 bytes, declared 12"}},
			expectedErr: "stored 10 bytes, declared 12",
		},
		{
			name:        "failed without a condition",
			phases:      []storagev1.OSImagePhase{storagev1.OSImagePhaseFailed},
			expectedErr: "no reason reported",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := fake.NewSimpleClientset()
			var reads int
			client.PrependReactor("get", "osimages", func(k8stesting.Action) (bool, runtime.Object, error) {
				phase := test.phases[min(reads, len(test.phases)-1)]
				reads++
				return true, &managementv1.OSImage{
					ObjectMeta: metav1.ObjectMeta{Name: imageName},
					Status: managementv1.OSImageStatus{OSImageStatus: storagev1.OSImageStatus{
						Phase:      phase,
						Conditions: test.conditions,
					}},
				}, nil
			})

			err := newTestUploader(client).WaitForReady(t.Context(), imageName, time.Second)
			if test.expectedErr == "" {
				assert.NilError(t, err)
				assert.Equal(t, len(test.phases), reads)
				return
			}
			assert.ErrorContains(t, err, test.expectedErr)
		})
	}
}

func TestWaitForReadyTimesOut(t *testing.T) {
	client := fake.NewSimpleClientset(&managementv1.OSImage{ObjectMeta: metav1.ObjectMeta{Name: imageName}})

	err := newTestUploader(client).WaitForReady(t.Context(), imageName, 10*time.Millisecond)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	// A bare "context deadline exceeded" would not say which image gave up.
	assert.ErrorContains(t, err, imageName)
}

func TestUploadFailsWhenARefreshComesBackShort(t *testing.T) {
	image := testImage(12)
	store := newStore(t)
	store.failStatus = http.StatusForbidden
	store.failures[2] = 1
	platform := newPlatform(store, 4, 3)

	client := platform.client()
	client.PrependReactor("create", "osimages", func(action k8stesting.Action) (bool, runtime.Object, error) {
		create, ok := action.(k8stesting.CreateAction)
		if !ok || create.GetSubresource() != "upload" {
			return false, nil, nil
		}

		response := platform.upload(create.GetObject().(*managementv1.OSImageUpload))
		// The refresh answers without the part that asked for it.
		if create.GetObject().(*managementv1.OSImageUpload).Spec.FromPart > 1 {
			response.Status.Targets = nil
		}
		return true, response, nil
	})

	err := newTestUploader(client).Upload(t.Context(), imageName, bytes.NewReader(image), int64(len(image)))
	assert.ErrorContains(t, err, "refreshed 0 of the 1 targets")
}

func TestFinalizeCarriesTheChecksum(t *testing.T) {
	platform := newPlatform(newStore(t), 4, 3)

	assert.NilError(t, newTestUploader(platform.client()).Finalize(t.Context(), imageName, checksum))
	assert.DeepEqual(t, []string{checksum}, platform.finalizeCalls())
}

func TestRedactTargetDropsTheSignature(t *testing.T) {
	signed := "https://store.example.com/os-images/abc.qcow2?partNumber=2&X-Amz-Signature=deadbeef"
	err := redactTarget(&url.Error{Op: "Put", URL: signed, Err: errors.New("connection reset")})

	assert.ErrorContains(t, err, "connection reset")
	assert.ErrorContains(t, err, "https://store.example.com/os-images/abc.qcow2")
	assert.Assert(t, !strings.Contains(err.Error(), "X-Amz-Signature"), "the pre-signed query reached the error: %v", err)
}

// conflictOnce answers the first call to subresource with a 409, the way the aggregated API
// does when the controller wins the resourceVersion race on the image the session is
// annotated onto.
func conflictOnce(client *fake.Clientset, subresource string) {
	var once sync.Once
	client.PrependReactor("create", "osimages", func(action k8stesting.Action) (bool, runtime.Object, error) {
		create, ok := action.(k8stesting.CreateAction)
		if !ok || create.GetSubresource() != subresource {
			return false, nil, nil
		}

		conflicted := false
		once.Do(func() { conflicted = true })
		if !conflicted {
			return false, nil, nil
		}
		return true, nil, kerrors.NewConflict(
			schema.GroupResource{Group: "management.loft.sh", Resource: "osimages"}, imageName,
			errors.New("the object has been modified"))
	})
}

func TestUploadRetriesAConflictOnTheSession(t *testing.T) {
	image := testImage(12)
	store := newStore(t)
	platform := newPlatform(store, 4, 3)

	client := platform.client()
	conflictOnce(client, "upload")

	assert.NilError(t, newTestUploader(client).Upload(t.Context(), imageName, bytes.NewReader(image), int64(len(image))))
	assert.DeepEqual(t, image, store.assembled())
}

func TestFinalizeRetriesAConflict(t *testing.T) {
	platform := newPlatform(newStore(t), 4, 3)

	client := platform.client()
	conflictOnce(client, "finalize")

	assert.NilError(t, newTestUploader(client).Finalize(t.Context(), imageName, checksum))
	assert.DeepEqual(t, []string{checksum}, platform.finalizeCalls())
}
