package util

import (
	"errors"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

func GetCause(err error) string {
	if err == nil {
		return ""
	}

	var statusErr *kerrors.StatusError

	if errors.As(err, &statusErr) {
		details := statusErr.Status().Details
		if details != nil && len(details.Causes) > 0 {
			return details.Causes[0].Message
		}

		return statusErr.Error()
	}

	return err.Error()
}
