package types

import (
	"context"
	"io"
	"time"
)

type Snapshot struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Timestamp time.Time `json:"timestamp"`
}

type Storage interface {
	Target() string
	PutObject(ctx context.Context, body io.Reader) error
	GetObject(ctx context.Context) (io.ReadCloser, error)
	List(ctx context.Context) ([]Snapshot, error)
	Delete(ctx context.Context) error
}
