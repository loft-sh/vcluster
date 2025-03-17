package pro

import "context"

var StartEmbeddedEtcd = func(_ context.Context, _, _, _ string, _, _ int, _ string, _, _ bool) error {
	return NewFeatureError("embedded etcd")
}
