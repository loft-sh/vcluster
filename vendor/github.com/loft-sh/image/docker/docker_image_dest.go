package docker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/docker/distribution/registry/api/errcode"
	v2 "github.com/docker/distribution/registry/api/v2"
	"github.com/loft-sh/image/docker/reference"
	"github.com/loft-sh/image/internal/blobinfocache"
	"github.com/loft-sh/image/internal/imagedestination/impl"
	"github.com/loft-sh/image/internal/imagedestination/stubs"
	"github.com/loft-sh/image/internal/private"
	"github.com/loft-sh/image/internal/putblobdigest"
	"github.com/loft-sh/image/internal/streamdigest"
	"github.com/loft-sh/image/internal/uploadreader"
	"github.com/loft-sh/image/manifest"
	compressiontypes "github.com/loft-sh/image/pkg/compression/types"
	"github.com/loft-sh/image/types"
	"github.com/opencontainers/go-digest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
)

type dockerImageDestination struct {
	impl.Compat
	impl.PropertyMethodsInitialize
	stubs.IgnoresOriginalOCIConfig
	stubs.NoPutBlobPartialInitialize

	ref dockerReference
	c   *dockerClient
	// State
	manifestDigest digest.Digest // or "" if not yet known.
}

// newImageDestination creates a new ImageDestination for the specified image reference.
func newImageDestination(sys *types.SystemContext, ref dockerReference) (private.ImageDestination, error) {
	registryConfig, err := loadRegistryConfiguration(sys)
	if err != nil {
		return nil, err
	}
	c, err := newDockerClientFromRef(sys, ref, registryConfig, true, "pull,push")
	if err != nil {
		return nil, err
	}
	mimeTypes := []string{
		imgspecv1.MediaTypeImageManifest,
		manifest.DockerV2Schema2MediaType,
		imgspecv1.MediaTypeImageIndex,
		manifest.DockerV2ListMediaType,
	}
	if c.sys == nil || !c.sys.DockerDisableDestSchema1MIMETypes {
		mimeTypes = append(mimeTypes, manifest.DockerV2Schema1SignedMediaType, manifest.DockerV2Schema1MediaType)
	}

	dest := &dockerImageDestination{
		PropertyMethodsInitialize: impl.PropertyMethods(impl.Properties{
			SupportedManifestMIMETypes:     mimeTypes,
			DesiredLayerCompression:        types.Compress,
			MustMatchRuntimeOS:             false,
			IgnoresEmbeddedDockerReference: false, // We do want the manifest updated; older registry versions refuse manifests if the embedded reference does not match.
			HasThreadSafePutBlob:           true,
		}),
		NoPutBlobPartialInitialize: stubs.NoPutBlobPartial(ref),

		ref: ref,
		c:   c,
	}
	dest.Compat = impl.AddCompat(dest)
	return dest, nil
}

// Reference returns the reference used to set up this destination.  Note that this should directly correspond to user's intent,
// e.g. it should use the public hostname instead of the result of resolving CNAMEs or following redirects.
func (d *dockerImageDestination) Reference() types.ImageReference {
	return d.ref
}

// Close removes resources associated with an initialized ImageDestination, if any.
func (d *dockerImageDestination) Close() error {
	return d.c.Close()
}

// SupportsSignatures returns an error (to be displayed to the user) if the destination certainly can't store signatures.
// Note: It is still possible for PutSignatures to fail if SupportsSignatures returns nil.
func (d *dockerImageDestination) SupportsSignatures(ctx context.Context) error {
	if err := d.c.detectProperties(ctx); err != nil {
		return err
	}
	switch {
	case d.c.supportsSignatures:
		return nil
	case d.c.signatureBase != nil:
		return nil
	default:
		return errors.New("Internal error: X-Registry-Supports-Signatures extension not supported, and lookaside should not be empty configuration")
	}
}

// AcceptsForeignLayerURLs returns false iff foreign layers in manifest should be actually
// uploaded to the image destination, true otherwise.
func (d *dockerImageDestination) AcceptsForeignLayerURLs() bool {
	return true
}

// sizeCounter is an io.Writer which only counts the total size of its input.
type sizeCounter struct{ size int64 }

func (c *sizeCounter) Write(p []byte) (n int, err error) {
	c.size += int64(len(p))
	return len(p), nil
}

// PutBlobWithOptions writes contents of stream and returns data representing the result.
// inputInfo.Digest can be optionally provided if known; if provided, and stream is read to the end without error, the digest MUST match the stream contents.
// inputInfo.Size is the expected length of stream, if known.
// inputInfo.MediaType describes the blob format, if known.
// WARNING: The contents of stream are being verified on the fly.  Until stream.Read() returns io.EOF, the contents of the data SHOULD NOT be available
// to any other readers for download using the supplied digest.
// If stream.Read() at any time, ESPECIALLY at end of input, returns an error, PutBlobWithOptions MUST 1) fail, and 2) delete any data stored so far.
func (d *dockerImageDestination) PutBlobWithOptions(ctx context.Context, stream io.Reader, inputInfo types.BlobInfo, options private.PutBlobOptions) (private.UploadedBlob, error) {
	// If requested, precompute the blob digest to prevent uploading layers that already exist on the registry.
	// This functionality is particularly useful when BlobInfoCache has not been populated with compressed digests,
	// the source blob is uncompressed, and the destination blob is being compressed "on the fly".
	if inputInfo.Digest == "" && d.c.sys != nil && d.c.sys.DockerRegistryPushPrecomputeDigests {
		logrus.Debugf("Precomputing digest layer for %s", reference.Path(d.ref.ref))
		streamCopy, cleanup, err := streamdigest.ComputeBlobInfo(d.c.sys, stream, &inputInfo)
		if err != nil {
			return private.UploadedBlob{}, err
		}
		defer cleanup()
		stream = streamCopy
	}

	if inputInfo.Digest != "" {
		// This should not really be necessary, at least the copy code calls TryReusingBlob automatically.
		// Still, we need to check, if only because the "initiate upload" endpoint does not have a documented "blob already exists" return value.
		haveBlob, reusedInfo, err := d.tryReusingExactBlob(ctx, inputInfo, options.Cache)
		if err != nil {
			return private.UploadedBlob{}, err
		}
		if haveBlob {
			return private.UploadedBlob{Digest: reusedInfo.Digest, Size: reusedInfo.Size}, nil
		}
	}

	// FIXME? Chunked upload, progress reporting, etc.
	uploadPath := fmt.Sprintf(blobUploadPath, reference.Path(d.ref.ref))
	logrus.Debugf("Uploading %s", uploadPath)
	res, err := d.c.makeRequest(ctx, http.MethodPost, uploadPath, nil, nil, v2Auth, nil)
	if err != nil {
		return private.UploadedBlob{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusAccepted {
		logrus.Debugf("Error initiating layer upload, response %#v", *res)
		return private.UploadedBlob{}, fmt.Errorf("initiating layer upload to %s in %s: %w", uploadPath, d.c.registry, registryHTTPResponseToError(res))
	}
	uploadLocation, err := res.Location()
	if err != nil {
		return private.UploadedBlob{}, fmt.Errorf("determining upload URL: %w", err)
	}

	digester, stream := putblobdigest.DigestIfCanonicalUnknown(stream, inputInfo)
	sizeCounter := &sizeCounter{}
	stream = io.TeeReader(stream, sizeCounter)

	uploadLocation, err = func() (*url.URL, error) { // A scope for defer
		uploadReader := uploadreader.NewUploadReader(stream)
		// This error text should never be user-visible, we terminate only after makeRequestToResolvedURL
		// returns, so there isn’t a way for the error text to be provided to any of our callers.
		defer uploadReader.Terminate(errors.New("Reading data from an already terminated upload"))
		res, err = d.c.makeRequestToResolvedURL(ctx, http.MethodPatch, uploadLocation, map[string][]string{"Content-Type": {"application/octet-stream"}}, uploadReader, inputInfo.Size, v2Auth, nil)
		if err != nil {
			logrus.Debugf("Error uploading layer chunked %v", err)
			return nil, err
		}
		defer res.Body.Close()
		if !successStatus(res.StatusCode) {
			return nil, fmt.Errorf("uploading layer chunked: %w", registryHTTPResponseToError(res))
		}
		uploadLocation, err := res.Location()
		if err != nil {
			return nil, fmt.Errorf("determining upload URL: %w", err)
		}
		return uploadLocation, nil
	}()
	if err != nil {
		return private.UploadedBlob{}, err
	}
	blobDigest := digester.Digest()

	// FIXME: DELETE uploadLocation on failure (does not really work in docker/distribution servers, which incorrectly require the "delete" action in the token's scope)

	locationQuery := uploadLocation.Query()
	locationQuery.Set("digest", blobDigest.String())
	uploadLocation.RawQuery = locationQuery.Encode()
	res, err = d.c.makeRequestToResolvedURL(ctx, http.MethodPut, uploadLocation, map[string][]string{"Content-Type": {"application/octet-stream"}}, nil, -1, v2Auth, nil)
	if err != nil {
		return private.UploadedBlob{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		logrus.Debugf("Error uploading layer, response %#v", *res)
		return private.UploadedBlob{}, fmt.Errorf("uploading layer to %s: %w", uploadLocation, registryHTTPResponseToError(res))
	}

	logrus.Debugf("Upload of layer %s complete", blobDigest)
	options.Cache.RecordKnownLocation(d.ref.Transport(), bicTransportScope(d.ref), blobDigest, newBICLocationReference(d.ref))
	return private.UploadedBlob{Digest: blobDigest, Size: sizeCounter.size}, nil
}

// blobExists returns true iff repo contains a blob with digest, and if so, also its size.
// If the destination does not contain the blob, or it is unknown, blobExists ordinarily returns (false, -1, nil);
// it returns a non-nil error only on an unexpected failure.
func (d *dockerImageDestination) blobExists(ctx context.Context, repo reference.Named, digest digest.Digest, extraScope *authScope) (bool, int64, error) {
	if err := digest.Validate(); err != nil { // Make sure digest.String() does not contain any unexpected characters
		return false, -1, err
	}
	checkPath := fmt.Sprintf(blobsPath, reference.Path(repo), digest.String())
	logrus.Debugf("Checking %s", checkPath)
	res, err := d.c.makeRequest(ctx, http.MethodHead, checkPath, nil, nil, v2Auth, extraScope)
	if err != nil {
		return false, -1, err
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		size, err := getBlobSize(res)
		if err != nil {
			return false, -1, fmt.Errorf("determining size of blob %s in %s: %w", digest, repo.Name(), err)
		}
		logrus.Debugf("... already exists")
		return true, size, nil
	case http.StatusUnauthorized:
		logrus.Debugf("... not authorized")
		return false, -1, fmt.Errorf("checking whether a blob %s exists in %s: %w", digest, repo.Name(), registryHTTPResponseToError(res))
	case http.StatusNotFound:
		logrus.Debugf("... not present")
		return false, -1, nil
	default:
		return false, -1, fmt.Errorf("checking whether a blob %s exists in %s: %w", digest, repo.Name(), registryHTTPResponseToError(res))
	}
}

// mountBlob tries to mount blob srcDigest from srcRepo to the current destination.
func (d *dockerImageDestination) mountBlob(ctx context.Context, srcRepo reference.Named, srcDigest digest.Digest, extraScope *authScope) error {
	u := url.URL{
		Path: fmt.Sprintf(blobUploadPath, reference.Path(d.ref.ref)),
		RawQuery: url.Values{
			"mount": {srcDigest.String()},
			"from":  {reference.Path(srcRepo)},
		}.Encode(),
	}
	logrus.Debugf("Trying to mount %s", u.Redacted())
	res, err := d.c.makeRequest(ctx, http.MethodPost, u.String(), nil, nil, v2Auth, extraScope)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusCreated:
		logrus.Debugf("... mount OK")
		return nil
	case http.StatusAccepted:
		// Oops, the mount was ignored - either the registry does not support that yet, or the blob does not exist; the registry has started an ordinary upload process.
		// Abort, and let the ultimate caller do an upload when its ready, instead.
		// NOTE: This does not really work in docker/distribution servers, which incorrectly require the "delete" action in the token's scope, and is thus entirely untested.
		uploadLocation, err := res.Location()
		if err != nil {
			return fmt.Errorf("determining upload URL after a mount attempt: %w", err)
		}
		logrus.Debugf("... started an upload instead of mounting, trying to cancel at %s", uploadLocation.Redacted())
		res2, err := d.c.makeRequestToResolvedURL(ctx, http.MethodDelete, uploadLocation, nil, nil, -1, v2Auth, extraScope)
		if err != nil {
			logrus.Debugf("Error trying to cancel an inadvertent upload: %s", err)
		} else {
			defer res2.Body.Close()
			if res2.StatusCode != http.StatusNoContent {
				logrus.Debugf("Error trying to cancel an inadvertent upload, status %s", http.StatusText(res.StatusCode))
			}
		}
		// Anyway, if canceling the upload fails, ignore it and return the more important error:
		return fmt.Errorf("Mounting %s from %s to %s started an upload instead", srcDigest, srcRepo.Name(), d.ref.ref.Name())
	default:
		logrus.Debugf("Error mounting, response %#v", *res)
		return fmt.Errorf("mounting %s from %s to %s: %w", srcDigest, srcRepo.Name(), d.ref.ref.Name(), registryHTTPResponseToError(res))
	}
}

// tryReusingExactBlob is a subset of TryReusingBlob which _only_ looks for exactly the specified
// blob in the current repository, with no cross-repo reuse or mounting; cache may be updated, it is not read.
// The caller must ensure info.Digest is set.
func (d *dockerImageDestination) tryReusingExactBlob(ctx context.Context, info types.BlobInfo, cache blobinfocache.BlobInfoCache2) (bool, private.ReusedBlob, error) {
	exists, size, err := d.blobExists(ctx, d.ref.ref, info.Digest, nil)
	if err != nil {
		return false, private.ReusedBlob{}, err
	}
	if exists {
		cache.RecordKnownLocation(d.ref.Transport(), bicTransportScope(d.ref), info.Digest, newBICLocationReference(d.ref))
		return true, private.ReusedBlob{Digest: info.Digest, Size: size}, nil
	}
	return false, private.ReusedBlob{}, nil
}

func optionalCompressionName(algo *compressiontypes.Algorithm) string {
	if algo != nil {
		return algo.Name()
	}
	return "nil"
}

// TryReusingBlobWithOptions checks whether the transport already contains, or can efficiently reuse, a blob, and if so, applies it to the current destination
// (e.g. if the blob is a filesystem layer, this signifies that the changes it describes need to be applied again when composing a filesystem tree).
// info.Digest must not be empty.
// If the blob has been successfully reused, returns (true, info, nil).
// If the transport can not reuse the requested blob, TryReusingBlob returns (false, {}, nil); it returns a non-nil error only on an unexpected failure.
func (d *dockerImageDestination) TryReusingBlobWithOptions(ctx context.Context, info types.BlobInfo, options private.TryReusingBlobOptions) (bool, private.ReusedBlob, error) {
	if info.Digest == "" {
		return false, private.ReusedBlob{}, errors.New("Can not check for a blob with unknown digest")
	}

	originalCandidateKnownToBeMissing := false
	if impl.OriginalCandidateMatchesTryReusingBlobOptions(options) {
		// First, check whether the blob happens to already exist at the destination.
		haveBlob, reusedInfo, err := d.tryReusingExactBlob(ctx, info, options.Cache)
		if err != nil {
			return false, private.ReusedBlob{}, err
		}
		if haveBlob {
			return true, reusedInfo, nil
		}
		originalCandidateKnownToBeMissing = true
	} else {
		logrus.Debugf("Ignoring exact blob match, compression %s does not match required %s or MIME types %#v",
			optionalCompressionName(options.OriginalCompression), optionalCompressionName(options.RequiredCompression), options.PossibleManifestFormats)
		// We can get here with a blob detected to be zstd when the user wants a zstd:chunked.
		// In that case we keep originalCandiateKnownToBeMissing = false, so that if we find
		// a BIC entry for this blob, we do use that entry and return a zstd:chunked entry
		// with the BIC’s annotations.
		// This is not quite correct, it only works if the BIC also contains an acceptable _location_.
		// Ideally, we could look up just the compression algorithm/annotations for info.digest,
		// and use it even if no location candidate exists and the original dandidate is present.
	}

	// Then try reusing blobs from other locations.
	candidates := options.Cache.CandidateLocations2(d.ref.Transport(), bicTransportScope(d.ref), info.Digest, blobinfocache.CandidateLocations2Options{
		CanSubstitute:           options.CanSubstitute,
		PossibleManifestFormats: options.PossibleManifestFormats,
		RequiredCompression:     options.RequiredCompression,
	})
	for _, candidate := range candidates {
		var candidateRepo reference.Named
		if !candidate.UnknownLocation {
			var err error
			candidateRepo, err = parseBICLocationReference(candidate.Location)
			if err != nil {
				logrus.Debugf("Error parsing BlobInfoCache location reference: %s", err)
				continue
			}
			if candidate.CompressionAlgorithm != nil {
				logrus.Debugf("Trying to reuse blob with cached digest %s compressed with %s in destination repo %s", candidate.Digest.String(), candidate.CompressionAlgorithm.Name(), candidateRepo.Name())
			} else {
				logrus.Debugf("Trying to reuse blob with cached digest %s in destination repo %s", candidate.Digest.String(), candidateRepo.Name())
			}
			// Sanity checks:
			if reference.Domain(candidateRepo) != reference.Domain(d.ref.ref) {
				// OCI distribution spec 1.1 allows mounting blobs without specifying the source repo
				// (the "from" parameter); in that case we might try to use these candidates as well.
				//
				// OTOH that would mean we can’t do the “blobExists” check, and if there is no match
				// we could get an upload request that we would have to cancel.
				logrus.Debugf("... Internal error: domain %s does not match destination %s", reference.Domain(candidateRepo), reference.Domain(d.ref.ref))
				continue
			}
		} else {
			if candidate.CompressionAlgorithm != nil {
				logrus.Debugf("Trying to reuse blob with cached digest %s compressed with %s with no location match, checking current repo", candidate.Digest.String(), candidate.CompressionAlgorithm.Name())
			} else {
				logrus.Debugf("Trying to reuse blob with cached digest %s in destination repo with no location match, checking current repo", candidate.Digest.String())
			}
			// This digest is a known variant of this blob but we don’t
			// have a recorded location in this registry, let’s try looking
			// for it in the current repo.
			candidateRepo = reference.TrimNamed(d.ref.ref)
		}
		if originalCandidateKnownToBeMissing &&
			candidateRepo.Name() == d.ref.ref.Name() && candidate.Digest == info.Digest {
			logrus.Debug("... Already tried the primary destination")
			continue
		}

		// Whatever happens here, don't abort the entire operation.  It's likely we just don't have permissions, and if it is a critical network error, we will find out soon enough anyway.

		// Checking candidateRepo, and mounting from it, requires an
		// expanded token scope.
		extraScope := &authScope{
			resourceType: "repository",
			remoteName:   reference.Path(candidateRepo),
			actions:      "pull",
		}
		// This existence check is not, strictly speaking, necessary: We only _really_ need it to get the blob size, and we could record that in the cache instead.
		// But a "failed" d.mountBlob currently leaves around an unterminated server-side upload, which we would try to cancel.
		// So, without this existence check, it would be 1 request on success, 2 requests on failure; with it, it is 2 requests on success, 1 request on failure.
		// On success we avoid the actual costly upload; so, in a sense, the success case is "free", but failures are always costly.
		// Even worse, docker/distribution does not actually reasonably implement canceling uploads
		// (it would require a "delete" action in the token, and Quay does not give that to anyone, so we can't ask);
		// so, be a nice client and don't create unnecessary upload sessions on the server.
		exists, size, err := d.blobExists(ctx, candidateRepo, candidate.Digest, extraScope)
		if err != nil {
			logrus.Debugf("... Failed: %v", err)
			continue
		}
		if !exists {
			// FIXME? Should we drop the blob from cache here (and elsewhere?)?
			continue // logrus.Debug() already happened in blobExists
		}
		if candidateRepo.Name() != d.ref.ref.Name() {
			if err := d.mountBlob(ctx, candidateRepo, candidate.Digest, extraScope); err != nil {
				logrus.Debugf("... Mount failed: %v", err)
				continue
			}
		}

		options.Cache.RecordKnownLocation(d.ref.Transport(), bicTransportScope(d.ref), candidate.Digest, newBICLocationReference(d.ref))

		return true, private.ReusedBlob{
			Digest:                 candidate.Digest,
			Size:                   size,
			CompressionOperation:   candidate.CompressionOperation,
			CompressionAlgorithm:   candidate.CompressionAlgorithm,
			CompressionAnnotations: candidate.CompressionAnnotations,
		}, nil
	}

	return false, private.ReusedBlob{}, nil
}

// PutManifest writes manifest to the destination.
// When the primary manifest is a manifest list, if instanceDigest is nil, we're saving the list
// itself, else instanceDigest contains a digest of the specific manifest instance to overwrite the
// manifest for; when the primary manifest is not a manifest list, instanceDigest should always be nil.
// FIXME? This should also receive a MIME type if known, to differentiate between schema versions.
// If the destination is in principle available, refuses this manifest type (e.g. it does not recognize the schema),
// but may accept a different manifest type, the returned error must be an ManifestTypeRejectedError.
func (d *dockerImageDestination) PutManifest(ctx context.Context, m []byte, instanceDigest *digest.Digest) error {
	var refTail string
	// If d.ref.isUnknownDigest=true, then we push without a tag, so get the
	// digest that will be used
	if d.ref.isUnknownDigest {
		digest, err := manifest.Digest(m)
		if err != nil {
			return err
		}
		refTail = digest.String()
	} else if instanceDigest != nil {
		// If the instanceDigest is provided, then use it as the refTail, because the reference,
		// whether it includes a tag or a digest, refers to the list as a whole, and not this
		// particular instance.
		refTail = instanceDigest.String()
		// Double-check that the manifest we've been given matches the digest we've been given.
		// This also validates the format of instanceDigest.
		matches, err := manifest.MatchesDigest(m, *instanceDigest)
		if err != nil {
			return fmt.Errorf("digesting manifest in PutManifest: %w", err)
		}
		if !matches {
			manifestDigest, merr := manifest.Digest(m)
			if merr != nil {
				return fmt.Errorf("Attempted to PutManifest using an explicitly specified digest (%q) that didn't match the manifest's digest: %w", instanceDigest.String(), merr)
			}
			return fmt.Errorf("Attempted to PutManifest using an explicitly specified digest (%q) that didn't match the manifest's digest (%q)", instanceDigest.String(), manifestDigest.String())
		}
	} else {
		// Compute the digest of the main manifest, or the list if it's a list, so that we
		// have a digest value to use if we're asked to save a signature for the manifest.
		digest, err := manifest.Digest(m)
		if err != nil {
			return err
		}
		d.manifestDigest = digest
		// The refTail should be either a digest (which we expect to match the value we just
		// computed) or a tag name.
		refTail, err = d.ref.tagOrDigest()
		if err != nil {
			return err
		}
	}

	return d.uploadManifest(ctx, m, refTail)
}

// uploadManifest writes manifest to tagOrDigest.
func (d *dockerImageDestination) uploadManifest(ctx context.Context, m []byte, tagOrDigest string) error {
	path := fmt.Sprintf(manifestPath, reference.Path(d.ref.ref), tagOrDigest)

	headers := map[string][]string{}
	mimeType := manifest.GuessMIMEType(m)
	if mimeType != "" {
		headers["Content-Type"] = []string{mimeType}
	}
	res, err := d.c.makeRequest(ctx, http.MethodPut, path, headers, bytes.NewReader(m), v2Auth, nil)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if !successStatus(res.StatusCode) {
		rawErr := registryHTTPResponseToError(res)
		err := fmt.Errorf("uploading manifest %s to %s: %w", tagOrDigest, d.ref.ref.Name(), rawErr)
		if isManifestInvalidError(rawErr) {
			err = types.ManifestTypeRejectedError{Err: err}
		}
		return err
	}
	// A HTTP server may not be a registry at all, and just return 200 OK to everything
	// (in particular that can fairly easily happen after tearing down a website and
	// replacing it with a global 302 redirect to a new website, completely ignoring the
	// path in the request); in that case we could “succeed” uploading a whole image.
	// With docker/distribution we could rely on a Docker-Content-Digest header being present
	// (because docker/distribution/registry/client has been failing uploads if it was missing),
	// but that has been defined as explicitly optional by
	// https://github.com/opencontainers/distribution-spec/blob/ec90a2af85fe4d612cf801e1815b95bfa40ae72b/spec.md#legacy-docker-support-http-headers
	// So, just note the missing header in a debug log.
	if v := res.Header.Values("Docker-Content-Digest"); len(v) == 0 {
		logrus.Debugf("Manifest upload response didn’t contain a Docker-Content-Digest header, it might not be a container registry")
	}
	return nil
}

// successStatus returns true if the argument is a successful HTTP response
// code (in the range 200 - 399 inclusive).
func successStatus(status int) bool {
	return status >= 200 && status <= 399
}

// isManifestInvalidError returns true iff err from registryHTTPResponseToError is a “manifest invalid” error.
func isManifestInvalidError(err error) bool {
	var ec errcode.ErrorCoder
	if ok := errors.As(err, &ec); !ok {
		return false
	}

	switch ec.ErrorCode() {
	// ErrorCodeManifestInvalid is returned by OpenShift with acceptschema2=false.
	case v2.ErrorCodeManifestInvalid:
		return true
	// ErrorCodeTagInvalid is returned by docker/distribution (at least as of commit ec87e9b6971d831f0eff752ddb54fb64693e51cd)
	// when uploading to a tag (because it can’t find a matching tag inside the manifest)
	case v2.ErrorCodeTagInvalid:
		return true
	// ErrorCodeUnsupported with 'Invalid JSON syntax' is returned by AWS ECR when
	// uploading an OCI manifest that is (correctly, according to the spec) missing
	// a top-level media type. See libpod issue #1719
	// FIXME: remove this case when ECR behavior is fixed
	case errcode.ErrorCodeUnsupported:
		return strings.Contains(err.Error(), "Invalid JSON syntax")
	default:
		return false
	}
}

// putBlobBytesAsOCI uploads a blob with the specified contents, and returns an appropriate
// OCI descriptor.
func (d *dockerImageDestination) putBlobBytesAsOCI(ctx context.Context, contents []byte, mimeType string, options private.PutBlobOptions) (imgspecv1.Descriptor, error) {
	blobDigest := digest.FromBytes(contents)
	info, err := d.PutBlobWithOptions(ctx, bytes.NewReader(contents),
		types.BlobInfo{
			Digest:    blobDigest,
			Size:      int64(len(contents)),
			MediaType: mimeType,
		}, options)
	if err != nil {
		return imgspecv1.Descriptor{}, fmt.Errorf("writing blob %s: %w", blobDigest.String(), err)
	}
	return imgspecv1.Descriptor{
		MediaType: mimeType,
		Digest:    info.Digest,
		Size:      info.Size,
	}, nil
}

// CommitWithOptions marks the process of storing the image as successful and asks for the image to be persisted.
// WARNING: This does not have any transactional semantics:
// - Uploaded data MAY be visible to others before CommitWithOptions() is called
// - Uploaded data MAY be removed or MAY remain around if Close() is called without CommitWithOptions() (i.e. rollback is allowed but not guaranteed)
func (d *dockerImageDestination) CommitWithOptions(ctx context.Context, options private.CommitOptions) error {
	return nil
}
