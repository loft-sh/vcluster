package store

import "context"

type Backend interface {
	// List retrieves all saved mappings
	List(ctx context.Context) ([]*Mapping, error)

	// Watch lists and watches for new mappings
	Watch(ctx context.Context) <-chan BackendWatchResponse

	// Save saves the given mapping
	Save(ctx context.Context, mapping *Mapping) error

	// Delete removes the given mapping
	Delete(ctx context.Context, mapping *Mapping) error
}

type BackendWatchResponse struct {
	Events []*BackendWatchEvent
}

type BackendWatchEvent struct {
	Type    BackendWatchEventType
	Mapping *Mapping
}

type BackendWatchEventType string

const (
	BackendWatchEventTypeUpdate BackendWatchEventType = "Update"
	BackendWatchEventTypeDelete BackendWatchEventType = "Delete"
)
