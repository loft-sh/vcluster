package volumes

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
)

type Restorer interface {
	// Reconcile volumes restore request.
	Reconcile(ctx context.Context, requestObj runtime.Object, requestName string, request *RestoreRequestSpec, status *RestoreRequestStatus) error
}
