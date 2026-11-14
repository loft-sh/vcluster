package cli

import (
	"context"
	"time"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
)

func waitUntilNotFound(ctx context.Context, interval, timeout time.Duration, get func(context.Context) error) error {
	return wait.PollUntilContextTimeout(ctx, interval, timeout, true, func(ctx context.Context) (bool, error) {
		err := get(ctx)
		if err == nil {
			return false, nil
		}
		if kerrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
}
