package oras

import (
	"path/filepath"
	"strings"

	"github.com/containerd/containerd/images"
	orascontent "github.com/deislabs/oras/pkg/content"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

type pushOpts struct {
	config              *ocispec.Descriptor
	configMediaType     string
	configAnnotations   map[string]string
	manifestAnnotations map[string]string
	validateName        func(desc ocispec.Descriptor) error
	baseHandlers        []images.Handler
}

func pushOptsDefaults() *pushOpts {
	return &pushOpts{
		validateName: ValidateNameAsPath,
	}
}

// PushOpt allows callers to set options on the oras push
type PushOpt func(o *pushOpts) error

// WithConfig overrides the config
func WithConfig(config ocispec.Descriptor) PushOpt {
	return func(o *pushOpts) error {
		o.config = &config
		return nil
	}
}

// WithConfigMediaType overrides the config media type
func WithConfigMediaType(mediaType string) PushOpt {
	return func(o *pushOpts) error {
		o.configMediaType = mediaType
		return nil
	}
}

// WithConfigAnnotations overrides the config annotations
func WithConfigAnnotations(annotations map[string]string) PushOpt {
	return func(o *pushOpts) error {
		o.configAnnotations = annotations
		return nil
	}
}

// WithManifestAnnotations overrides the manifest annotations
func WithManifestAnnotations(annotations map[string]string) PushOpt {
	return func(o *pushOpts) error {
		o.manifestAnnotations = annotations
		return nil
	}
}

// WithNameValidation validates the image title in the descriptor.
// Pass nil to disable name validation.
func WithNameValidation(validate func(desc ocispec.Descriptor) error) PushOpt {
	return func(o *pushOpts) error {
		o.validateName = validate
		return nil
	}
}

// ValidateNameAsPath validates name in the descriptor as file path in order
// to generate good packages intended to be pulled using the FileStore or
// the oras cli.
// For cross-platform considerations, only unix paths are accepted.
func ValidateNameAsPath(desc ocispec.Descriptor) error {
	// no empty name
	path, ok := orascontent.ResolveName(desc)
	if !ok || path == "" {
		return orascontent.ErrNoName
	}

	// path should be clean
	if target := filepath.ToSlash(filepath.Clean(path)); target != path {
		return errors.Wrap(ErrDirtyPath, path)
	}

	// path should be slash-separated
	if strings.Contains(path, "\\") {
		return errors.Wrap(ErrPathNotSlashSeparated, path)
	}

	// disallow absolute path: covers unix and windows format
	if strings.HasPrefix(path, "/") {
		return errors.Wrap(ErrAbsolutePathDisallowed, path)
	}
	if len(path) > 2 {
		c := path[0]
		if path[1] == ':' && path[2] == '/' && ('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z') {
			return errors.Wrap(ErrAbsolutePathDisallowed, path)
		}
	}

	// disallow path traversal
	if strings.HasPrefix(path, "../") || path == ".." {
		return errors.Wrap(ErrPathTraversalDisallowed, path)
	}

	return nil
}

// WithPushBaseHandler provides base handlers, which will be called before
// any push specific handlers.
func WithPushBaseHandler(handlers ...images.Handler) PushOpt {
	return func(o *pushOpts) error {
		o.baseHandlers = append(o.baseHandlers, handlers...)
		return nil
	}
}
