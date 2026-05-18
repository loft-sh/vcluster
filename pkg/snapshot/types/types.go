package types

import (
	"context"
	"io"

	"github.com/loft-sh/api/v4/pkg/snapshot"
)

type Storage interface {
	Target() string
	PutObject(ctx context.Context, body io.Reader) error
	GetObject(ctx context.Context) (io.ReadCloser, error)
	List(ctx context.Context) ([]snapshot.Snapshot, error)
	Delete(ctx context.Context) error
}
