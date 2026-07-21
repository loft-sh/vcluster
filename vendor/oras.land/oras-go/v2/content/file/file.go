/*
Copyright The ORAS Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package file provides implementation of a content store based on file system.
package file

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/internal/cas"
	"oras.land/oras-go/v2/internal/graph"
	"oras.land/oras-go/v2/internal/ioutil"
	"oras.land/oras-go/v2/internal/resolver"
)

// bufPool is a pool of byte buffers that can be reused for copying content
// between files.
var bufPool = sync.Pool{
	New: func() interface{} {
		// the buffer size should be larger than or equal to 128 KiB
		// for performance considerations.
		// we choose 1 MiB here so there will be less disk I/O.
		buffer := make([]byte, 1<<20) // buffer size = 1 MiB
		return &buffer
	},
}

const (
	// AnnotationDigest is the annotation key for the digest of the uncompressed content.
	AnnotationDigest = "io.deis.oras.content.digest"
	// AnnotationUnpack is the annotation key for indication of unpacking.
	AnnotationUnpack = "io.deis.oras.content.unpack"
	// defaultBlobMediaType specifies the default blob media type.
	defaultBlobMediaType = ocispec.MediaTypeImageLayer
	// defaultBlobDirMediaType specifies the default blob directory media type.
	defaultBlobDirMediaType = ocispec.MediaTypeImageLayerGzip
	// defaultFallbackPushSizeLimit specifies the default size limit for pushing no-name contents.
	defaultFallbackPushSizeLimit = 1 << 22 // 4 MiB
)

// Store represents a file system based store, which implements `oras.Target`.
//
// In the file store, the contents described by names are location-addressed
// by file paths. Meanwhile, the file paths are mapped to a virtual CAS
// where all metadata are stored in the memory.
//
// The contents that are not described by names are stored in a fallback storage,
// which is a limited memory CAS by default.
// As all the metadata are stored in the memory, the file store
// cannot be restored from the file system.
//
// After use, the file store needs to be closed by calling the [Store.Close] function.
// The file store cannot be used after being closed.
type Store struct {
	// TarReproducible controls if the tarballs generated
	// for the added directories are reproducible.
	// When specified, some metadata such as change time
	// will be removed from the files in the tarballs. Default value: false.
	TarReproducible bool
	// AllowPathTraversalOnWrite controls if path traversal is allowed
	// when writing files. When specified, writing files
	// outside the working directory will be allowed. Default value: false.
	AllowPathTraversalOnWrite bool
	// DisableOverwrite controls if push operations can overwrite existing files.
	// When specified, saving files to existing paths will be disabled.
	// Default value: false.
	DisableOverwrite bool
	// ForceCAS controls if files with same content but different names are
	// deduped after push operations. When a DAG is copied between CAS
	// targets, nodes are deduped by content. By default, file store restores
	// deduped successor files after a node is copied. This may result in two
	// files with identical content. If this is not the desired behavior,
	// ForceCAS can be specified to enforce CAS style dedup.
	// Default value: false.
	ForceCAS bool
	// IgnoreNoName controls if push operations should ignore descriptors
	// without a name. When specified, corresponding content will be discarded.
	// Otherwise, content will be saved to a fallback storage.
	// A typical scenario is pulling an arbitrary artifact masqueraded as OCI
	// image to file store. This option can be specified to discard unnamed
	// manifest and config file, while leaving only named layer files.
	// Default value: false.
	IgnoreNoName bool
	// SkipUnpack controls if push operations should skip unpacking files. This
	// value overrides the [AnnotationUnpack].
	// Default value: false.
	SkipUnpack bool
	// PreservePermissions controls whether to preserve file permissions when unpacking,
	// disregarding the active umask, similar to tar's `--preserve-permissions`
	PreservePermissions bool

	workingDir   string   // the working directory of the file store
	closed       int32    // if the store is closed - 0: false, 1: true.
	digestToPath sync.Map // map[digest.Digest]string
	nameToStatus sync.Map // map[string]*nameStatus
	tmpFiles     sync.Map // map[string]bool

	fallbackStorage content.Storage
	resolver        content.TagResolver
	graph           *graph.Memory
}

// nameStatus contains a flag indicating if a name exists,
// and a RWMutex protecting it.
type nameStatus struct {
	sync.RWMutex
	exists bool
}

// New creates a file store, using a default limited memory CAS
// as the fallback storage for contents without names.
// When pushing content without names, the size of content being pushed
// cannot exceed the default size limit: 4 MiB.
func New(workingDir string) (*Store, error) {
	return NewWithFallbackLimit(workingDir, defaultFallbackPushSizeLimit)
}

// NewWithFallbackLimit creates a file store, using a default
// limited memory CAS as the fallback storage for contents without names.
// When pushing content without names, the size of content being pushed
// cannot exceed the size limit specified by the `limit` parameter.
func NewWithFallbackLimit(workingDir string, limit int64) (*Store, error) {
	m := cas.NewMemory()
	ls := content.LimitStorage(m, limit)
	return NewWithFallbackStorage(workingDir, ls)
}

// NewWithFallbackStorage creates a file store,
// using the provided fallback storage for contents without names.
func NewWithFallbackStorage(workingDir string, fallbackStorage content.Storage) (*Store, error) {
	workingDirAbs, err := filepath.Abs(workingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path for %s: %w", workingDir, err)
	}

	return &Store{
		workingDir:      workingDirAbs,
		fallbackStorage: fallbackStorage,
		resolver:        resolver.NewMemory(),
		graph:           graph.NewMemory(),
	}, nil
}

// Close closes the file store and cleans up all the temporary files used by it.
// The store cannot be used after being closed.
// This function is not go-routine safe.
func (s *Store) Close() error {
	if s.isClosedSet() {
		return nil
	}
	s.setClosed()

	var errs []string
	s.tmpFiles.Range(func(name, _ interface{}) bool {
		if err := os.Remove(name.(string)); err != nil {
			errs = append(errs, err.Error())
		}
		return true
	})

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

// Fetch fetches the content identified by the descriptor.
func (s *Store) Fetch(ctx context.Context, target ocispec.Descriptor) (io.ReadCloser, error) {
	if s.isClosedSet() {
		return nil, ErrStoreClosed
	}

	// if the target has name, check if the name exists.
	name := target.Annotations[ocispec.AnnotationTitle]
	if name != "" && !s.nameExists(name) {
		return nil, fmt.Errorf("%s: %s: %w", name, target.MediaType, errdef.ErrNotFound)
	}

	// check if the content exists in the store
	val, exists := s.digestToPath.Load(target.Digest)
	if exists {
		path := val.(string)

		fp, err := os.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("%s: %s: %w", target.Digest, target.MediaType, errdef.ErrNotFound)
			}
			return nil, err
		}

		return fp, nil
	}

	// if the content does not exist in the store,
	// then fall back to the fallback storage.
	return s.fallbackStorage.Fetch(ctx, target)
}

// Push pushes the content, matching the expected descriptor.
// If name is not specified in the descriptor, the content will be pushed to
// the fallback storage by default, or will be discarded when
// Store.IgnoreNoName is true.
func (s *Store) Push(ctx context.Context, expected ocispec.Descriptor, content io.Reader) error {
	if s.isClosedSet() {
		return ErrStoreClosed
	}

	if err := s.push(ctx, expected, content); err != nil {
		if errors.Is(err, errSkipUnnamed) {
			return nil
		}
		return err
	}

	if !s.ForceCAS {
		if err := s.restoreDuplicates(ctx, expected); err != nil {
			return fmt.Errorf("failed to restore duplicated file: %w", err)
		}
	}

	return s.graph.Index(ctx, s, expected)
}

// push pushes the content, matching the expected descriptor.
// If name is not specified in the descriptor, the content will be pushed to
// the fallback storage by default, or will be discarded when
// Store.IgnoreNoName is true.
func (s *Store) push(ctx context.Context, expected ocispec.Descriptor, content io.Reader) error {
	name := expected.Annotations[ocispec.AnnotationTitle]
	if name == "" {
		if s.IgnoreNoName {
			return errSkipUnnamed
		}
		return s.fallbackStorage.Push(ctx, expected, content)
	}

	// check the status of the name
	status := s.status(name)
	status.Lock()
	defer status.Unlock()

	if status.exists {
		return fmt.Errorf("%s: %w", name, ErrDuplicateName)
	}

	target, err := s.resolveWritePath(name)
	if err != nil {
		return fmt.Errorf("failed to resolve path for writing: %w", err)
	}

	if needUnpack := expected.Annotations[AnnotationUnpack]; needUnpack == "true" && !s.SkipUnpack {
		err = s.pushDir(name, target, expected, content)
	} else {
		err = s.pushFile(target, expected, content)
	}
	if err != nil {
		return err
	}

	// update the name status as existed
	status.exists = true
	return nil
}

// restoreDuplicates restores successor files with same content but different
// names.
// See Store.ForceCAS for more info.
func (s *Store) restoreDuplicates(ctx context.Context, desc ocispec.Descriptor) error {
	successors, err := content.Successors(ctx, s, desc)
	if err != nil {
		return err
	}
	for _, successor := range successors {
		name := successor.Annotations[ocispec.AnnotationTitle]
		if name == "" || s.nameExists(name) {
			continue
		}
		if err := func() error {
			desc := ocispec.Descriptor{
				MediaType: successor.MediaType,
				Digest:    successor.Digest,
				Size:      successor.Size,
			}
			rc, err := s.Fetch(ctx, desc)
			if err != nil {
				return fmt.Errorf("%q: %s: %w", name, desc.MediaType, err)
			}
			defer rc.Close()
			if err := s.push(ctx, successor, rc); err != nil {
				return fmt.Errorf("%q: %s: %w", name, desc.MediaType, err)
			}
			return nil
		}(); err != nil {
			switch {
			case errors.Is(err, errdef.ErrNotFound):
				// allow pushing manifests before blobs
			case errors.Is(err, ErrDuplicateName):
				// in case multiple goroutines are pushing or restoring the same
				// named content, the error is ignored
			default:
				return err
			}
		}
	}
	return nil
}

// Exists returns true if the described content exists.
func (s *Store) Exists(ctx context.Context, target ocispec.Descriptor) (bool, error) {
	if s.isClosedSet() {
		return false, ErrStoreClosed
	}

	// if the target has name, check if the name exists.
	name := target.Annotations[ocispec.AnnotationTitle]
	if name != "" && !s.nameExists(name) {
		return false, nil
	}

	// check if the content exists in the store
	_, exists := s.digestToPath.Load(target.Digest)
	if exists {
		return true, nil
	}

	// if the content does not exist in the store,
	// then fall back to the fallback storage.
	return s.fallbackStorage.Exists(ctx, target)
}

// Resolve resolves a reference to a descriptor.
func (s *Store) Resolve(ctx context.Context, ref string) (ocispec.Descriptor, error) {
	if s.isClosedSet() {
		return ocispec.Descriptor{}, ErrStoreClosed
	}

	if ref == "" {
		return ocispec.Descriptor{}, errdef.ErrMissingReference
	}

	return s.resolver.Resolve(ctx, ref)
}

// Tag tags a descriptor with a reference string.
func (s *Store) Tag(ctx context.Context, desc ocispec.Descriptor, ref string) error {
	if s.isClosedSet() {
		return ErrStoreClosed
	}

	if ref == "" {
		return errdef.ErrMissingReference
	}

	exists, err := s.Exists(ctx, desc)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%s: %s: %w", desc.Digest, desc.MediaType, errdef.ErrNotFound)
	}

	return s.resolver.Tag(ctx, desc, ref)
}

// Predecessors returns the nodes directly pointing to the current node.
// Predecessors returns nil without error if the node does not exists in the
// store.
func (s *Store) Predecessors(ctx context.Context, node ocispec.Descriptor) ([]ocispec.Descriptor, error) {
	if s.isClosedSet() {
		return nil, ErrStoreClosed
	}

	return s.graph.Predecessors(ctx, node)
}

// Add adds a file or a directory into the file store.
// Hard links within the directory are treated as regular files.
func (s *Store) Add(ctx context.Context, name, mediaType, path string) (ocispec.Descriptor, error) {
	if s.isClosedSet() {
		return ocispec.Descriptor{}, ErrStoreClosed
	}

	if name == "" {
		return ocispec.Descriptor{}, ErrMissingName
	}

	// check the status of the name
	status := s.status(name)
	status.Lock()
	defer status.Unlock()

	if status.exists {
		return ocispec.Descriptor{}, fmt.Errorf("%s: %w", name, ErrDuplicateName)
	}

	if path == "" {
		path = name
	}
	path = s.absPath(path)

	fi, err := os.Stat(path)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to stat %s: %w", path, err)
	}

	// generate descriptor
	var desc ocispec.Descriptor
	if fi.IsDir() {
		desc, err = s.descriptorFromDir(ctx, name, mediaType, path)
	} else {
		desc, err = s.descriptorFromFile(fi, mediaType, path)
	}
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to generate descriptor from %s: %w", path, err)
	}

	if desc.Annotations == nil {
		desc.Annotations = make(map[string]string)
	}
	desc.Annotations[ocispec.AnnotationTitle] = name

	// update the name status as existed
	status.exists = true
	return desc, nil
}

// saveFile saves content matching the descriptor to the given file.
func (s *Store) saveFile(fp *os.File, expected ocispec.Descriptor, content io.Reader) (err error) {
	defer func() {
		closeErr := fp.Close()
		if err == nil {
			err = closeErr
		}
	}()
	path := fp.Name()

	buf := bufPool.Get().(*[]byte)
	defer bufPool.Put(buf)
	if err := ioutil.CopyBuffer(fp, content, *buf, expected); err != nil {
		return fmt.Errorf("failed to copy content to %s: %w", path, err)
	}

	s.digestToPath.Store(expected.Digest, path)
	return nil
}

// pushFile saves content matching the descriptor to the target path.
func (s *Store) pushFile(target string, expected ocispec.Descriptor, content io.Reader) error {
	if err := ensureDir(filepath.Dir(target)); err != nil {
		return fmt.Errorf("failed to ensure directories of the target path: %w", err)
	}

	fp, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", target, err)
	}

	return s.saveFile(fp, expected, content)
}

// pushDir saves content matching the descriptor to the target directory.
func (s *Store) pushDir(name, target string, expected ocispec.Descriptor, content io.Reader) (err error) {
	if err := ensureDir(target); err != nil {
		return fmt.Errorf("failed to ensure directories of the target path: %w", err)
	}

	gz, err := s.tempFile()
	if err != nil {
		return err
	}

	gzPath := gz.Name()
	// the digest of the gz is verified while saving
	if err := s.saveFile(gz, expected, content); err != nil {
		return fmt.Errorf("failed to save gzip to %s: %w", gzPath, err)
	}

	checksum := expected.Annotations[AnnotationDigest]
	buf := bufPool.Get().(*[]byte)
	defer bufPool.Put(buf)
	if err := extractTarGzip(target, name, gzPath, checksum, *buf, s.PreservePermissions); err != nil {
		return fmt.Errorf("failed to extract tar to %s: %w", target, err)
	}
	return nil
}

// descriptorFromDir generates descriptor from the given directory.
func (s *Store) descriptorFromDir(ctx context.Context, name, mediaType, dir string) (desc ocispec.Descriptor, err error) {
	// make a temp file to store the gzip
	gz, err := s.tempFile()
	if err != nil {
		return ocispec.Descriptor{}, err
	}
	defer func() {
		closeErr := gz.Close()
		if err == nil {
			err = closeErr
		}
	}()

	// compress the directory
	gzDigester := digest.Canonical.Digester()
	gzw := gzip.NewWriter(io.MultiWriter(gz, gzDigester.Hash()))
	defer func() {
		closeErr := gzw.Close()
		if err == nil {
			err = closeErr
		}
	}()

	tarDigester := digest.Canonical.Digester()
	tw := io.MultiWriter(gzw, tarDigester.Hash())
	buf := bufPool.Get().(*[]byte)
	defer bufPool.Put(buf)
	if err := tarDirectory(ctx, dir, name, tw, s.TarReproducible, *buf); err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to tar %s: %w", dir, err)
	}

	// flush all
	if err := gzw.Close(); err != nil {
		return ocispec.Descriptor{}, err
	}
	if err := gz.Sync(); err != nil {
		return ocispec.Descriptor{}, err
	}

	fi, err := gz.Stat()
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	// map gzip digest to gzip path
	gzDigest := gzDigester.Digest()
	s.digestToPath.Store(gzDigest, gz.Name())

	// generate descriptor
	if mediaType == "" {
		mediaType = defaultBlobDirMediaType
	}

	return ocispec.Descriptor{
		MediaType: mediaType,
		Digest:    gzDigest, // digest for the compressed content
		Size:      fi.Size(),
		Annotations: map[string]string{
			AnnotationDigest: tarDigester.Digest().String(), // digest fot the uncompressed content
			AnnotationUnpack: "true",                        // the content needs to be unpacked
		},
	}, nil
}

// descriptorFromFile generates descriptor from the given file.
func (s *Store) descriptorFromFile(fi os.FileInfo, mediaType, path string) (desc ocispec.Descriptor, err error) {
	fp, err := os.Open(path)
	if err != nil {
		return ocispec.Descriptor{}, err
	}
	defer func() {
		closeErr := fp.Close()
		if err == nil {
			err = closeErr
		}
	}()

	dgst, err := digest.FromReader(fp)
	if err != nil {
		return ocispec.Descriptor{}, err
	}
	// map digest to file path
	s.digestToPath.Store(dgst, path)

	// generate descriptor
	if mediaType == "" {
		mediaType = defaultBlobMediaType
	}

	return ocispec.Descriptor{
		MediaType: mediaType,
		Digest:    dgst,
		Size:      fi.Size(),
	}, nil
}

// resolveWritePath resolves the path to write for the given name.
func (s *Store) resolveWritePath(name string) (string, error) {
	path := s.absPath(name)
	if !s.AllowPathTraversalOnWrite {
		base, err := filepath.Abs(s.workingDir)
		if err != nil {
			return "", err
		}
		target, err := filepath.Abs(path)
		if err != nil {
			return "", err
		}
		rel, err := filepath.Rel(base, target)
		if err != nil {
			return "", ErrPathTraversalDisallowed
		}
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "../") || rel == ".." {
			return "", ErrPathTraversalDisallowed
		}
	}
	if s.DisableOverwrite {
		if _, err := os.Stat(path); err == nil {
			return "", ErrOverwriteDisallowed
		} else if !os.IsNotExist(err) {
			return "", err
		}
	}
	return path, nil
}

// status returns the nameStatus for the given name.
func (s *Store) status(name string) *nameStatus {
	v, _ := s.nameToStatus.LoadOrStore(name, &nameStatus{sync.RWMutex{}, false})
	status := v.(*nameStatus)
	return status
}

// nameExists returns if the given name exists in the file store.
func (s *Store) nameExists(name string) bool {
	status := s.status(name)
	status.RLock()
	defer status.RUnlock()

	return status.exists
}

// tempFile creates a temp file with the file name format "oras_file_randomString",
// and returns the pointer to the temp file.
func (s *Store) tempFile() (*os.File, error) {
	tmp, err := os.CreateTemp("", "oras_file_*")
	if err != nil {
		return nil, err
	}

	s.tmpFiles.Store(tmp.Name(), true)
	return tmp, nil
}

// absPath returns the absolute path of the path.
func (s *Store) absPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(s.workingDir, path)
}

// isClosedSet returns true if the `closed` flag is set, otherwise returns false.
func (s *Store) isClosedSet() bool {
	return atomic.LoadInt32(&s.closed) == 1
}

// setClosed sets the `closed` flag.
func (s *Store) setClosed() {
	atomic.StoreInt32(&s.closed, 1)
}

// ensureDir ensures the directories of the path exists.
func ensureDir(path string) error {
	return os.MkdirAll(path, 0777)
}
