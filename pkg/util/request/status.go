package request

import (
	"encoding/json"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func SucceedWithObject(w http.ResponseWriter, obj interface{}) {
	bytes, _ := json.Marshal(obj)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(int(http.StatusOK))
	_, _ = w.Write(bytes)
}

func SucceedWithStatus(w http.ResponseWriter) {
	bytes, _ := json.Marshal(NewSuccessStatus())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(int(http.StatusOK))
	_, _ = w.Write(bytes)
}

func FailWithStatus(w http.ResponseWriter, req *http.Request, code int32, err error) {
	klog.V(3).Info(req.URL.Path + ": " + err.Error())

	reason := metav1.StatusReasonUnknown
	switch code {
	case http.StatusBadRequest:
		reason = metav1.StatusReasonBadRequest
	case http.StatusForbidden:
		reason = metav1.StatusReasonForbidden
	case http.StatusUnauthorized:
		reason = metav1.StatusReasonUnauthorized
	case http.StatusInternalServerError:
		reason = metav1.StatusReasonInternalError
	case http.StatusNotFound:
		reason = metav1.StatusReasonNotFound
	}

	bytes, _ := json.Marshal(NewErrorRequestStatus(code, reason, err))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(int(code))
	_, _ = w.Write(bytes)
}

func NewSuccessStatus() *metav1.Status {
	return &metav1.Status{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Status",
			APIVersion: "v1",
		},
		Status: metav1.StatusSuccess,
		Code:   http.StatusOK,
	}
}

func NewErrorRequestStatus(code int32, reason metav1.StatusReason, err error) *metav1.Status {
	return &metav1.Status{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Status",
			APIVersion: "v1",
		},
		Status:  metav1.StatusFailure,
		Message: err.Error(),
		Reason:  reason,
		Code:    code,
	}
}
