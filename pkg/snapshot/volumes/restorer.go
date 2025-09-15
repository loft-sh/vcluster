package volumes

import (
	"context"
)

type Restorer interface {
	// Reconcile volumes restore request.
	Reconcile(ctx context.Context, restoreRequestName string, restoreRequest *SnapshotRequest) error
}
