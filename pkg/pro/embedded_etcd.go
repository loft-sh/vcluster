package pro

import (
	"context"

	"k8s.io/client-go/kubernetes"
)

var StartEmbeddedEtcd = func(_ context.Context, _, _ string, _ kubernetes.Interface, _ string, _ int, _ string, _ bool, _ []string, _ bool) (func(), error) {
	return nil, NewFeatureError("embedded etcd")
}
