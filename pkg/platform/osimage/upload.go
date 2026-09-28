// Package osimage pushes an OS image's bytes to the platform's object store over the
// pre-signed targets the upload subresource hands out, so they never pass through the
// platform itself.
package osimage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	managementv1client "github.com/loft-sh/api/v4/pkg/clientset/versioned/typed/management/v1"
	"github.com/loft-sh/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
)

const (
	// partAttempts is how often one part is PUT before the batch asks for fresh targets.
	partAttempts = 3

	// retryBackoff is the wait before the second attempt, doubled for every one after it.
	retryBackoff = time.Second

	// pollInterval is how often the image is read while waiting for the controller.
	pollInterval = 2 * time.Second

	// responseHeaderTimeout bounds a store that takes a part and then goes silent. It only
	// starts once the body is written, so a slow upload is not cut off.
	responseHeaderTimeout = 30 * time.Second

	// errorSnippet bounds how much of an object store's error body we quote.
	errorSnippet = 512
)

// Uploader runs the upload flow against one platform.
type Uploader struct {
	images managementv1client.OSImageInterface
	log    log.Logger

	client *http.Client

	// Overridden in tests, where waiting is the whole runtime.
	backoff      time.Duration
	pollInterval time.Duration
}

func NewUploader(images managementv1client.OSImageInterface, logger log.Logger) *Uploader {
	// Cloned rather than built from scratch, so proxy and TLS settings from the environment
	// still apply.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = responseHeaderTimeout

	return &Uploader{
		images:       images,
		log:          logger,
		client:       &http.Client{Transport: transport},
		backoff:      retryBackoff,
		pollInterval: pollInterval,
	}
}

// partFailure is a part whose every attempt failed, kept so the batch can ask for a fresh
// target for exactly the parts that need one.
type partFailure struct {
	partNumber int32
	err        error
}

// statusError is a response the object store rejected, separated from a transport error so
// only the codes worth retrying in place are retried.
type statusError struct {
	code int
	body string
}

func (e *statusError) Error() string {
	if e.body == "" {
		return fmt.Sprintf("object store returned %d", e.code)
	}
	return fmt.Sprintf("object store returned %d: %s", e.code, e.body)
}

// Upload pushes sizeBytes from src to the image's store. It pages through the upload
// subresource, uploading each batch before asking for the next, so a target is used close to
// when it was signed. Finalize assembles the object afterwards.
func (u *Uploader) Upload(ctx context.Context, name string, src io.ReaderAt, sizeBytes int64) error {
	fromPart := int32(1)
	var uploaded int32

	for {
		batch, err := u.targets(ctx, name, sizeBytes, fromPart)
		if err != nil {
			return err
		}
		if len(batch.Status.Targets) == 0 {
			return fmt.Errorf("upload returned no targets for part %d of %d", fromPart, batch.Status.TotalParts)
		}
		if batch.Status.PartSizeBytes <= 0 {
			return fmt.Errorf("upload returned a part size of %d bytes", batch.Status.PartSizeBytes)
		}

		if err := u.putBatch(ctx, name, src, sizeBytes, batch); err != nil {
			return err
		}

		uploaded += int32(len(batch.Status.Targets))
		u.log.Infof("Uploaded %d/%d parts (%s/%s)", uploaded, batch.Status.TotalParts,
			humanize.IBytes(uint64(uploadedBytes(uploaded, batch.Status.PartSizeBytes, sizeBytes))),
			humanize.IBytes(uint64(sizeBytes)))

		if batch.Status.NextPart == nil {
			break
		}
		// A cursor that does not advance would page forever.
		if *batch.Status.NextPart <= fromPart {
			return fmt.Errorf("upload returned next part %d, which does not advance past %d", *batch.Status.NextPart, fromPart)
		}
		fromPart = *batch.Status.NextPart
	}

	return nil
}

// Finalize assembles the uploaded parts into the object and records checksum. It is split
// from Upload so a caller can hash the file after the first upload call has been accepted,
// rather than reading a multi gigabyte image before the platform has said yes.
func (u *Uploader) Finalize(ctx context.Context, name, checksum string) error {
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		_, err := u.images.Finalize(ctx, name, &managementv1.OSImageFinalize{
			Spec: managementv1.OSImageFinalizeSpec{Checksum: checksum},
		}, metav1.CreateOptions{})
		return err
	})
	if err != nil {
		return fmt.Errorf("finalize upload: %w", err)
	}
	return nil
}

// WaitForReady polls until the controller has verified the object. Finalize only assembles
// it; the phase is what says the stored size matched what we declared.
func (u *Uploader) WaitForReady(ctx context.Context, name string, timeout time.Duration) error {
	err := wait.PollUntilContextTimeout(ctx, u.pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
		image, err := u.images.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		switch image.Status.Phase {
		case storagev1.OSImagePhaseReady:
			return true, nil
		case storagev1.OSImagePhaseFailed:
			return false, fmt.Errorf("image failed: %s", failureMessage(image))
		case storagev1.OSImagePhasePending, storagev1.OSImagePhaseUploading:
			return false, nil
		}

		// An empty phase is an image the controller has not observed yet.
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("wait for os image %s to become ready: %w", name, err)
	}
	return nil
}

func (u *Uploader) targets(ctx context.Context, name string, sizeBytes int64, fromPart int32) (*managementv1.OSImageUpload, error) {
	var batch *managementv1.OSImageUpload

	// Both endpoints write session annotations onto the image, so they lose the
	// resourceVersion race against a controller stamping the finalizer or the phase. Both are
	// safe to call again, so a conflict is retried rather than reported.
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		var err error
		batch, err = u.images.Upload(ctx, name, &managementv1.OSImageUpload{
			Spec: managementv1.OSImageUploadSpec{SizeBytes: sizeBytes, FromPart: fromPart},
		}, metav1.CreateOptions{})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("request upload targets from part %d: %w", fromPart, err)
	}
	return batch, nil
}

func (u *Uploader) putBatch(ctx context.Context, name string, src io.ReaderAt, sizeBytes int64, batch *managementv1.OSImageUpload) error {
	failed := u.putTargets(ctx, src, sizeBytes, batch.Status.PartSizeBytes, batch.Status.Targets)
	if len(failed) == 0 {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	// Targets expire, so a part that keeps failing gets a freshly signed one before we give
	// up on it. Pointing fromPart back at a part re-signs it without opening a new session.
	refreshed, err := u.targets(ctx, name, sizeBytes, failed[0].partNumber)
	if err != nil {
		return fmt.Errorf("%w (after part %d failed: %w)", err, failed[0].partNumber, failed[0].err)
	}

	retry := make([]managementv1.OSImageUploadTarget, 0, len(failed))
	for _, target := range refreshed.Status.Targets {
		if slices.ContainsFunc(failed, func(f partFailure) bool { return f.partNumber == target.PartNumber }) {
			retry = append(retry, target)
		}
	}

	// Without this a batch whose refresh came back short would report success for a part that
	// was never uploaded.
	if len(retry) != len(failed) {
		return fmt.Errorf("refreshed %d of the %d targets that failed, first was part %d: %w",
			len(retry), len(failed), failed[0].partNumber, failed[0].err)
	}

	if stillFailed := u.putTargets(ctx, src, sizeBytes, refreshed.Status.PartSizeBytes, retry); len(stillFailed) > 0 {
		return fmt.Errorf("upload part %d: %w", stillFailed[0].partNumber, stillFailed[0].err)
	}
	return nil
}

// putTargets uploads one batch concurrently. Every part is attempted even when a sibling
// fails, so one refresh covers all of them.
func (u *Uploader) putTargets(ctx context.Context, src io.ReaderAt, sizeBytes, partSizeBytes int64, targets []managementv1.OSImageUploadTarget) []partFailure {
	var (
		mutex  sync.Mutex
		failed []partFailure
		group  sync.WaitGroup
	)

	for _, target := range targets {
		group.Add(1)
		go func() {
			defer group.Done()

			if err := u.putPart(ctx, src, sizeBytes, partSizeBytes, target); err != nil {
				mutex.Lock()
				defer mutex.Unlock()
				failed = append(failed, partFailure{partNumber: target.PartNumber, err: err})
			}
		}()
	}
	group.Wait()

	// Ordered, so the part we report and the one we refresh from do not depend on scheduling.
	slices.SortFunc(failed, func(a, b partFailure) int { return int(a.partNumber - b.partNumber) })
	return failed
}

func (u *Uploader) putPart(ctx context.Context, src io.ReaderAt, sizeBytes, partSizeBytes int64, target managementv1.OSImageUploadTarget) error {
	// Part numbers are 1-based, so part n starts at (n-1) * partSizeBytes.
	offset := int64(target.PartNumber-1) * partSizeBytes
	length := min(partSizeBytes, sizeBytes-offset)
	if length <= 0 {
		return fmt.Errorf("part %d starts at offset %d, past the end of a %d byte image", target.PartNumber, offset, sizeBytes)
	}

	var err error
	for attempt := range partAttempts {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(u.backoff << (attempt - 1)):
			}
		}

		// A new reader per attempt, since a failed one is left part-way through the part.
		err = u.put(ctx, target.URL, io.NewSectionReader(src, offset, length), length)
		if err == nil {
			return nil
		}
		if !retryable(err) {
			break
		}
	}

	return err
}

func (u *Uploader) put(ctx context.Context, target string, body io.Reader, length int64) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, target, body)
	if err != nil {
		return redactTarget(err)
	}
	// Set explicitly: without it a SectionReader is sent chunked, which pre-signed PUTs reject.
	request.ContentLength = length

	response, err := u.client.Do(request)
	if err != nil {
		return redactTarget(err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode > 299 {
		snippet, _ := io.ReadAll(io.LimitReader(response.Body, errorSnippet))
		return &statusError{code: response.StatusCode, body: string(snippet)}
	}

	// Drained so the connection is reused for the next part.
	_, _ = io.Copy(io.Discard, response.Body)
	return nil
}

// redactTarget drops the query from the URL a transport error carries, since Go keeps it in
// *url.Error and it holds the pre-signed signature.
func redactTarget(err error) error {
	var urlError *url.Error
	if !errors.As(err, &urlError) {
		return err
	}

	target, parseErr := url.Parse(urlError.URL)
	if parseErr != nil {
		return urlError.Err
	}
	return fmt.Errorf("%s %s://%s%s: %w", urlError.Op, target.Scheme, target.Host, target.Path, urlError.Err)
}

// retryable reports whether the same target is worth another attempt. A rejection the store
// will repeat - an expired signature above all - is left to the refresh path instead.
func retryable(err error) bool {
	var status *statusError
	if !errors.As(err, &status) {
		return true
	}
	return status.code >= 500 || status.code == http.StatusRequestTimeout || status.code == http.StatusTooManyRequests
}

func uploadedBytes(parts int32, partSizeBytes, sizeBytes int64) int64 {
	return min(int64(parts)*partSizeBytes, sizeBytes)
}

func failureMessage(image *managementv1.OSImage) string {
	for _, condition := range image.Status.Conditions {
		if condition.Message != "" {
			return condition.Message
		}
	}
	return "no reason reported"
}
