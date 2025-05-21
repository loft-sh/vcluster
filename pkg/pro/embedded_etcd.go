package pro

import (
	"context"

	"k8s.io/client-go/kubernetes"
)

var StartEmbeddedEtcd = func(_ context.Context, _, _, _ string, _, _ int, _ string, _, _ bool, _ kubernetes.Interface) error {
	return NewFeatureError("embedded etcd")
}
