package pro

import (
	"context"
	"errors"

	snapshotapi "github.com/loft-sh/api/v4/pkg/snapshot"
)

// ErrSnapshotCredentialsMalformed marks a response that arrived but could not be read. Terminal: the
// platform answered, so waiting only repeats the same answer.
var ErrSnapshotCredentialsMalformed = errors.New("the platform returned snapshot credentials that could not be read")

// ResolveSnapshotCredentials, when set by the pro runtime, pulls the snapshot storage credentials
// for a standalone, platform-connected tenant from the platform on demand, authenticated with the
// instance access key. Credentials are per instance (per auto-snapshot storage configuration), so
// no snapshot identifier is needed; the storage location comes from the snapshot request itself.
// The returned options carry only credentials and are used in memory only, never persisted in the
// tenant. It is nil in OSS-only builds (standalone auto-snapshots are a platform/pro feature), in
// which case the snapshot controller uses the options pushed alongside the request.
//
// The caller classifies the error to decide retry-vs-fail, so wrap API status errors with %w and a
// response that cannot be read with ErrSnapshotCredentialsMalformed; both are terminal, anything else
// is retried. Options with no Type mean "not provisioned yet" and are retried without caching.
var ResolveSnapshotCredentials func(ctx context.Context) (*snapshotapi.Options, error)
